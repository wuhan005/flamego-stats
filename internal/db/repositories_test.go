package db

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestDB connects to the PostgreSQL of the PG* environment variables in a temporary schema, and skips the test if not configured.
func newTestDB(t *testing.T, models ...interface{}) *gorm.DB {
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
	require.NoError(t, gormDB.AutoMigrate(models...))
	return gormDB
}

func TestRepositories(t *testing.T) {
	ctx := context.Background()
	gormDB := newTestDB(t, &Repository{})
	store := NewRepositoriesStore(gormDB)
	stats := NewStatsStore(gormDB)

	seenAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repos := []*Repository{
		{
			GitHubID: 1, OwnerID: 10, OwnerLogin: "gogs", OwnerType: "Organization", FullName: "gogs/gogs", Name: "gogs",
			Language: "Go", Topics: []string{"git", "self-hosted"}, Stars: 100, PushedAt: seenAt.Add(-time.Hour),
			GoVersion: "1.22", ModuleVersion: "v1.9.4", ModuleMinorVersion: "v1.9", RelatedModules: []string{"github.com/flamego/session"},
		},
		{
			GitHubID: 2, OwnerID: 20, OwnerLogin: "wuhan005", OwnerType: "User", FullName: "wuhan005/Elaina", Name: "Elaina",
			Language: "Go", Topics: []string{"self-hosted"}, Stars: 10, PushedAt: seenAt.Add(-time.Minute),
			GoVersion: "1.9", ModuleVersion: "v1.10.0", ModuleMinorVersion: "v1.10", RelatedModules: []string{"github.com/flamego/session", "github.com/flamego/binding"},
		},
		{
			GitHubID: 3, OwnerID: 20, OwnerLogin: "wuhan005", OwnerType: "User", FullName: "wuhan005/NekoBox", Name: "NekoBox",
			Language: "Vue", Stars: 50, PushedAt: seenAt.Add(-2 * time.Hour),
			GoVersion: "1.22.0", ModuleVersion: "v1.9.4", ModuleMinorVersion: "v1.9",
		},
		{
			GitHubID: 4, OwnerID: 30, OwnerLogin: "someone", FullName: "someone/gogs", Name: "gogs",
			Fork: true, Stars: 0, PushedAt: seenAt,
		},
	}
	for _, repo := range repos {
		repo.LastSeenAt = seenAt
		require.NoError(t, store.Upsert(ctx, repo))
	}

	// Upsert overwrites the existing repository.
	renamed := *repos[0]
	renamed.ID = 0
	renamed.FullName, renamed.Stars = "gogs/gogs-renamed", 120
	require.NoError(t, store.Upsert(ctx, &renamed))
	got, err := store.GetByGitHubID(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "gogs/gogs-renamed", got.FullName)
	require.Equal(t, 120, got.Stars)
	require.Equal(t, repos[0].ID, got.ID)

	t.Run("list", func(t *testing.T) {
		list, total, err := store.List(ctx, ListRepositoriesOptions{})
		require.NoError(t, err)
		require.Equal(t, int64(3), total)
		require.Equal(t, []string{"gogs/gogs-renamed", "wuhan005/NekoBox", "wuhan005/Elaina"}, fullNames(list))

		list, total, err = store.List(ctx, ListRepositoriesOptions{ScopeOptions: ScopeOptions{IncludeForks: true}, Sort: RepositorySortPushed})
		require.NoError(t, err)
		require.Equal(t, int64(4), total)
		require.Equal(t, []string{"someone/gogs", "wuhan005/Elaina", "gogs/gogs-renamed", "wuhan005/NekoBox"}, fullNames(list))

		list, _, err = store.List(ctx, ListRepositoriesOptions{Topic: "self-hosted", Owner: "WUHAN005"})
		require.NoError(t, err)
		require.Equal(t, []string{"wuhan005/Elaina"}, fullNames(list))

		list, _, err = store.List(ctx, ListRepositoriesOptions{Keyword: "neko"})
		require.NoError(t, err)
		require.Equal(t, []string{"wuhan005/NekoBox"}, fullNames(list))
	})

	t.Run("stats", func(t *testing.T) {
		overview, err := stats.Overview(ctx, ScopeOptions{})
		require.NoError(t, err)
		require.Equal(t, &Overview{Repositories: 3, Owners: 2, Stars: 180, Resolved: 3}, overview)

		owners, err := stats.TopOwners(ctx, TopOptions{Limit: 10})
		require.NoError(t, err)
		require.Len(t, owners, 2)
		require.Equal(t, &OwnerStat{OwnerID: 20, Login: "wuhan005", Type: "User", Repositories: 2, Stars: 60}, owners[0])
		require.Equal(t, "gogs", owners[1].Login)

		topics, err := stats.Topics(ctx, TopOptions{Limit: 10})
		require.NoError(t, err)
		require.Equal(t, []*CountStat{{Name: "self-hosted", Count: 2}, {Name: "git", Count: 1}}, topics)

		languages, err := stats.Languages(ctx, TopOptions{ScopeOptions: ScopeOptions{IncludeForks: true}})
		require.NoError(t, err)
		require.Equal(t, []*CountStat{{Name: "Go", Count: 2}, {Name: "Vue", Count: 1}}, languages)

		versions, err := stats.ModuleVersions(ctx, VersionOptions{})
		require.NoError(t, err)
		require.Equal(t, []*CountStat{{Name: "v1.10.0", Count: 1}, {Name: "v1.9.4", Count: 2}}, versions)

		versions, err = stats.ModuleVersions(ctx, VersionOptions{Granularity: VersionGranularityMinor})
		require.NoError(t, err)
		require.Equal(t, []*CountStat{{Name: "v1.10", Count: 1}, {Name: "v1.9", Count: 2}}, versions)

		goVersions, err := stats.GoVersions(ctx, ScopeOptions{})
		require.NoError(t, err)
		require.Equal(t, []*CountStat{{Name: "1.22", Count: 2}, {Name: "1.9", Count: 1}}, goVersions)

		modules, err := stats.RelatedModules(ctx, TopOptions{})
		require.NoError(t, err)
		require.Equal(t, []*CountStat{{Name: "github.com/flamego/session", Count: 2}, {Name: "github.com/flamego/binding", Count: 1}}, modules)
	})

	t.Run("remove stale", func(t *testing.T) {
		nextSeenAt := seenAt.Add(time.Hour)
		require.NoError(t, store.Touch(ctx, "WUHAN005/elaina", nextSeenAt))
		renamed.LastSeenAt = nextSeenAt
		require.NoError(t, store.Upsert(ctx, &renamed))

		removed, err := store.DeleteNotSeenSince(ctx, nextSeenAt)
		require.NoError(t, err)
		require.Equal(t, int64(2), removed)

		_, total, err := store.List(ctx, ListRepositoriesOptions{ScopeOptions: ScopeOptions{IncludeForks: true}})
		require.NoError(t, err)
		require.Equal(t, int64(2), total)

		// The removed repository is restored when it shows up again.
		repos[2].LastSeenAt = nextSeenAt
		require.NoError(t, store.Upsert(ctx, repos[2]))
		_, total, err = store.List(ctx, ListRepositoriesOptions{})
		require.NoError(t, err)
		require.Equal(t, int64(3), total)
	})

	t.Run("delete permanently", func(t *testing.T) {
		require.NoError(t, store.DeleteByGitHubID(ctx, 3))
		_, err := store.GetByGitHubID(ctx, 3)
		require.ErrorIs(t, err, ErrRepositoryNotFound)
	})
}

func TestSyncRuns(t *testing.T) {
	ctx := context.Background()
	store := NewSyncRunsStore(newTestDB(t, &SyncRun{}))

	_, err := store.GetLatestSucceeded(ctx)
	require.ErrorIs(t, err, ErrSyncRunNotFound)

	first, err := store.Start(ctx)
	require.NoError(t, err)
	require.NoError(t, store.Finish(ctx, first.ID, SyncRunResult{ReportedTotal: 133, Discovered: 130, Synced: 128, Failed: 2}, nil))

	second, err := store.Start(ctx)
	require.NoError(t, err)
	n, err := store.AbortRunning(ctx, "interrupted")
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	latest, err := store.GetLatestSucceeded(ctx)
	require.NoError(t, err)
	require.Equal(t, first.ID, latest.ID)
	require.Equal(t, SyncRunResult{ReportedTotal: 133, Discovered: 130, Synced: 128, Failed: 2}, latest.SyncRunResult)
	require.NotNil(t, latest.FinishedAt)

	runs, err := store.List(ctx, 10)
	require.NoError(t, err)
	require.Len(t, runs, 2)
	require.Equal(t, second.ID, runs[0].ID)
	require.Equal(t, SyncRunStatusFailed, runs[0].Status)
	require.Equal(t, "interrupted", runs[0].Error)
}

func fullNames(repos []*Repository) []string {
	names := make([]string, 0, len(repos))
	for _, repo := range repos {
		names = append(names, repo.FullName)
	}
	return names
}
