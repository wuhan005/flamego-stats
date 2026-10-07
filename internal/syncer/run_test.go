package syncer

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/wuhan005/flamego-stats/internal/conf"
	"github.com/wuhan005/flamego-stats/internal/db"
	"github.com/wuhan005/flamego-stats/internal/github"
)

// newTestDB connects to the PostgreSQL of the PG* environment variables in a temporary schema, and skips the test if not configured.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if os.Getenv("PGHOST") == "" {
		t.Skip("PGHOST is not set")
	}
	admin, err := gorm.Open(postgres.New(postgres.Config{DSN: "", PreferSimpleProtocol: true}), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	require.NoError(t, admin.Exec("CREATE SCHEMA "+schema).Error)
	t.Cleanup(func() {
		_ = admin.Exec("DROP SCHEMA " + schema + " CASCADE").Error
		if sqlDB, err := admin.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	gormDB, err := gorm.Open(postgres.New(postgres.Config{DSN: "search_path=" + schema, PreferSimpleProtocol: true}), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := gormDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, gormDB.AutoMigrate(&db.Repository{}, &db.SyncRun{}))
	db.SetDatabaseStore(gormDB)
	return gormDB
}

// fakeGitHub serves the dependents page and the REST API of the repositories:
//   - acme/dep is listed on the dependents page and requires the module in the root go.mod.
//   - acme/sub is only found by the code search, and requires the module in api/go.mod.
//   - acme/fp is only found by the code search, and its go.mod does not require the module.
type fakeGitHub struct {
	searchFails atomic.Bool
	treeCalls   atomic.Int32
}

func (f *fakeGitHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	goMod := func(require string) string {
		return fmt.Sprintf(`{"type":"file","encoding":"base64","content":%q}`,
			base64.StdEncoding.EncodeToString([]byte("module example.com/app\n\ngo 1.26.0\n\nrequire "+require+"\n")))
	}
	repo := func(id int, fullName string) string {
		owner, name, _ := strings.Cut(fullName, "/")
		return fmt.Sprintf(`{"id":%d,"name":%q,"full_name":%q,"owner":{"id":10,"login":%q,"type":"Organization"},
			"default_branch":"main","pushed_at":"2026-01-01T00:00:00Z","visibility":"public"}`, id, name, fullName, owner)
	}

	switch r.URL.Path {
	case "/flamego/flamego/network/dependents":
		_, _ = fmt.Fprint(w, `<div class="table-list-header-toggle"><a class="selected">2 Repositories</a></div>
			<div data-test-id="dg-repo-pkg-dependent"><a data-hovercard-type="repository" href="/acme/dep">dep</a></div>`)
	case "/api/search/code":
		if f.searchFails.Load() {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = fmt.Fprint(w, `{"message":"Validation Failed"}`)
			return
		}
		_, _ = fmt.Fprint(w, `{"total_count":3,"items":[
			{"path":"go.mod","repository":{"full_name":"acme/dep"}},
			{"path":"api/go.mod","repository":{"full_name":"acme/sub"}},
			{"path":"go.mod","repository":{"full_name":"acme/fp"}}
		]}`)
	case "/api/repos/acme/dep":
		_, _ = fmt.Fprint(w, repo(1, "acme/dep"))
	case "/api/repos/acme/sub":
		_, _ = fmt.Fprint(w, repo(2, "acme/sub"))
	case "/api/repos/acme/fp":
		_, _ = fmt.Fprint(w, repo(3, "acme/fp"))
	case "/api/repos/acme/dep/contents/go.mod":
		_, _ = fmt.Fprint(w, goMod("github.com/flamego/flamego v1.13.0"))
	case "/api/repos/acme/sub/contents/api/go.mod":
		_, _ = fmt.Fprint(w, goMod("github.com/flamego/flamego v1.12.0"))
	case "/api/repos/acme/fp/contents/go.mod":
		_, _ = fmt.Fprint(w, goMod("github.com/foo/flamego-bar v1.0.0"))
	case "/api/repos/acme/fp/git/trees/main":
		f.treeCalls.Add(1)
		_, _ = fmt.Fprint(w, `{"sha":"main","tree":[{"path":"go.mod","type":"blob"}]}`)
	default:
		if strings.Contains(r.URL.Path, "/git/trees/") {
			f.treeCalls.Add(1)
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"message":"Not Found"}`)
	}
}

func TestRun(t *testing.T) {
	ctx := context.Background()
	gormDB := newTestDB(t)

	fake := &fakeGitHub{}
	server := httptest.NewServer(fake)
	defer server.Close()
	client, err := github.NewClient("test-token", github.WithURLs(server.URL+"/api/", server.URL))
	require.NoError(t, err)

	conf.Sync.Repository = "flamego/flamego"
	conf.Sync.Module = "github.com/flamego/flamego"
	conf.Sync.Concurrency = 2
	conf.Sync.Sources = []conf.SyncSource{conf.SyncSourceDependents, conf.SyncSourceCodeSearch}
	conf.Sync.Dependents.PageDelay = time.Millisecond
	conf.Sync.CodeSearch.Query = "flamego filename:go.mod"

	stale := func(githubID int64, fullName string) {
		require.NoError(t, db.Repositories.Upsert(ctx, &db.Repository{GitHubID: githubID, FullName: fullName, LastSeenAt: time.Unix(0, 0)}))
	}
	fullNames := func() []string {
		list, _, err := db.Repositories.List(ctx, db.ListRepositoriesOptions{Sort: db.RepositorySortName})
		require.NoError(t, err)
		names := make([]string, 0, len(list))
		for _, repo := range list {
			names = append(names, fmt.Sprintf("%s@%s:%s", repo.FullName, repo.GoModPath, repo.ModuleVersion))
		}
		return names
	}

	t.Run("complete", func(t *testing.T) {
		stale(100, "acme/old")
		require.NoError(t, New(gormDB, client).Run(ctx))

		require.Equal(t, []string{"acme/dep@go.mod:v1.13.0", "acme/sub@api/go.mod:v1.12.0"}, fullNames())
		// The go.mod of acme/sub is found by the path of the code search without scanning the tree.
		require.Equal(t, int32(1), fake.treeCalls.Load())

		runs, err := db.SyncRuns.List(ctx, 1)
		require.NoError(t, err)
		require.Equal(t, db.SyncRunStatusSucceeded, runs[0].Status)
		require.Equal(t, db.SyncRunResult{ReportedTotal: 2, Discovered: 3, Synced: 2, Skipped: 1, Removed: 1}, runs[0].SyncRunResult)
	})

	t.Run("partial", func(t *testing.T) {
		fake.searchFails.Store(true)
		stale(101, "acme/old2")
		require.NoError(t, New(gormDB, client).Run(ctx))

		// acme/sub is only found by the failed code search, it is kept along with the stale repository.
		require.Equal(t, []string{"acme/dep@go.mod:v1.13.0", "acme/old2@:", "acme/sub@api/go.mod:v1.12.0"}, fullNames())

		runs, err := db.SyncRuns.List(ctx, 1)
		require.NoError(t, err)
		require.Equal(t, db.SyncRunStatusSucceeded, runs[0].Status)
		require.True(t, runs[0].Partial)
		require.Zero(t, runs[0].Removed)
		require.Contains(t, runs[0].Error, "code_search")
	})

	t.Run("all sources fail", func(t *testing.T) {
		conf.Sync.Sources = []conf.SyncSource{conf.SyncSourceCodeSearch}
		err := New(gormDB, client).Run(ctx)
		require.ErrorContains(t, err, "no repositories discovered: code_search")
	})
}
