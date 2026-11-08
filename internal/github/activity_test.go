package github

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestListOrgRepositoriesPaginates(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/orgs/acme/repos", r.URL.Path)
		require.Equal(t, "public", r.URL.Query().Get("type"))
		switch r.URL.Query().Get("page") {
		case "", "1":
			w.Header().Set("Link", fmt.Sprintf(`<http://%s/orgs/acme/repos?type=public&page=2>; rel="next"`, r.Host))
			_, _ = fmt.Fprint(w, `[{"id":1,"name":"web","full_name":"acme/web"}]`)
		case "2":
			_, _ = fmt.Fprint(w, `[{"id":2,"name":"cli","full_name":"acme/cli","fork":true}]`)
		default:
			t.Fatalf("unexpected page %q", r.URL.Query().Get("page"))
		}
	}))

	repos, err := client.ListOrgRepositories(context.Background(), "acme")
	require.NoError(t, err)
	require.Len(t, repos, 2)
	require.Equal(t, "acme/web", repos[0].FullName)
	require.True(t, repos[1].Fork)
}

func TestListCommits(t *testing.T) {
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/acme/web/commits", r.URL.Path)
		require.Equal(t, "main", r.URL.Query().Get("sha"))
		require.Equal(t, since.Format(time.RFC3339), r.URL.Query().Get("since"))
		_, _ = fmt.Fprint(w, `[
			{"sha":"a1","html_url":"https://github.com/acme/web/commit/a1","author":{"login":"alice","avatar_url":"https://avatars/alice","type":"User"},
			 "commit":{"message":"feat: add login (#7)\n\nDetails","committer":{"date":"2026-01-03T00:00:00Z"}},"parents":[{"sha":"p1"}]},
			{"sha":"b2","author":null,"commit":{"message":"Merge branch 'dev'","author":{"name":"Bob"},"committer":{"date":"2026-01-02T00:00:00Z"}},"parents":[{"sha":"p1"},{"sha":"p2"}]}
		]`)
	}))

	commits, err := client.ListCommits(context.Background(), "acme/web", "main", since)
	require.NoError(t, err)
	require.Equal(t, []*Commit{
		{
			SHA:         "a1",
			Message:     "feat: add login (#7)\n\nDetails",
			HTMLURL:     "https://github.com/acme/web/commit/a1",
			Author:      Actor{Login: "alice", AvatarURL: "https://avatars/alice"},
			CommittedAt: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC),
		},
		{
			SHA:         "b2",
			Message:     "Merge branch 'dev'",
			Author:      Actor{Login: "Bob"},
			CommittedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
			Merge:       true,
		},
	}, commits)
}

func TestListCommitsOfEmptyRepository(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = fmt.Fprint(w, `{"message":"Git Repository is empty."}`)
	}))

	_, err := client.ListCommits(context.Background(), "acme/empty", "main", time.Time{})
	require.ErrorIs(t, err, ErrNotFound)
}

func TestListIssues(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/acme/web/issues", r.URL.Path)
		require.Equal(t, "all", r.URL.Query().Get("state"))
		require.Equal(t, "updated", r.URL.Query().Get("sort"))
		_, _ = fmt.Fprint(w, `[
			{"id":10,"number":7,"title":"Add login","user":{"login":"dependabot[bot]","type":"Bot"},"created_at":"2026-01-01T00:00:00Z",
			 "closed_at":"2026-01-03T00:00:00Z","pull_request":{"merged_at":"2026-01-03T00:00:00Z"}},
			{"id":11,"number":8,"title":"Crash","user":{"login":"carol","type":"User"},"created_at":"2026-01-02T00:00:00Z"}
		]`)
	}))

	issues, err := client.ListIssues(context.Background(), "acme/web", time.Time{})
	require.NoError(t, err)
	require.Len(t, issues, 2)

	merged := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	require.True(t, issues[0].PullRequest)
	require.True(t, issues[0].Author.Bot)
	require.Equal(t, &merged, issues[0].MergedAt)

	require.False(t, issues[1].PullRequest)
	require.Nil(t, issues[1].ClosedAt)
	require.Nil(t, issues[1].MergedAt)
}

func TestListReleasesStopsAtOlderPage(t *testing.T) {
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NotEqual(t, "2", r.URL.Query().Get("page"), "the page after an older release is not needed")
		w.Header().Set("Link", fmt.Sprintf(`<http://%s%s?page=2>; rel="next"`, r.Host, r.URL.Path))
		_, _ = fmt.Fprint(w, `[
			{"id":3,"tag_name":"v1.2.0","name":"","draft":true,"created_at":"2026-01-05T00:00:00Z"},
			{"id":2,"tag_name":"v1.1.0","name":"v1.1.0 Beta","prerelease":true,"author":{"login":"alice"},"created_at":"2026-01-04T00:00:00Z","published_at":"2026-01-04T00:00:00Z"},
			{"id":1,"tag_name":"v1.0.0","created_at":"2025-12-01T00:00:00Z","published_at":"2025-12-01T00:00:00Z"}
		]`)
	}))

	releases, err := client.ListReleases(context.Background(), "acme/web", since)
	require.NoError(t, err)
	require.Equal(t, []*Release{{
		ID:          2,
		Tag:         "v1.1.0",
		Name:        "v1.1.0 Beta",
		Author:      Actor{Login: "alice"},
		Prerelease:  true,
		PublishedAt: time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC),
	}}, releases)
}
