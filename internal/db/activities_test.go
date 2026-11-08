package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestActivities(t *testing.T) {
	ctx := context.Background()
	gormDB := newTestDB(t, &ActivityRepository{}, &Activity{})
	repos := NewActivityRepositoriesStore(gormDB)
	store := NewActivitiesStore(gormDB)

	for _, repo := range []*ActivityRepository{
		{GitHubID: 1, Name: "web", FullName: "acme/web"},
		{GitHubID: 2, Name: "cli", FullName: "acme/cli"},
		{GitHubID: 3, Name: "docs", FullName: "acme/docs"},
	} {
		require.NoError(t, repos.Upsert(ctx, repo))
	}

	day := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	activities := []*Activity{
		{Key: "pr:1:opened", Type: ActivityTypePullRequest, Action: ActivityActionOpened, RepositoryID: 1, RepositoryName: "web", Number: 7, ActorLogin: "alice", OccurredAt: day.Add(1 * time.Hour)},
		{Key: "pr:1:merged", Type: ActivityTypePullRequest, Action: ActivityActionMerged, RepositoryID: 1, RepositoryName: "web", Number: 7, ActorLogin: "alice", OccurredAt: day.Add(20 * time.Hour)},
		// Squash-merged from #7, folded into its merging.
		{Key: "commit:1:a1", Type: ActivityTypeCommit, Action: ActivityActionCommitted, RepositoryID: 1, RepositoryName: "web", Number: 7, ActorLogin: "alice", OccurredAt: day.Add(20 * time.Hour)},
		// #9 of cli is not a merged pull request, so the commit is kept.
		{Key: "commit:2:b2", Type: ActivityTypeCommit, Action: ActivityActionCommitted, RepositoryID: 2, RepositoryName: "cli", Number: 9, ActorLogin: "bob", OccurredAt: day.Add(2 * time.Hour)},
		{Key: "issue:3:opened", Type: ActivityTypeIssue, Action: ActivityActionOpened, RepositoryID: 2, RepositoryName: "cli", Number: 3, ActorLogin: "dependabot[bot]", ActorBot: true, OccurredAt: day.Add(3 * time.Hour)},
		{Key: "release:1", Type: ActivityTypeRelease, Action: ActivityActionPublished, RepositoryID: 1, RepositoryName: "web", Tag: "v1.0.0", ActorLogin: "alice", OccurredAt: day.Add(-48 * time.Hour)},
	}
	require.NoError(t, store.Upsert(ctx, activities))

	// Upsert overwrites the existing activity by the key.
	edited := *activities[0]
	edited.ID, edited.Title = 0, "Edited"
	require.NoError(t, store.Upsert(ctx, []*Activity{&edited}))

	keys := func(activities []*Activity) []string {
		var keys []string
		for _, a := range activities {
			keys = append(keys, a.Key)
		}
		return keys
	}

	t.Run("feed", func(t *testing.T) {
		got, err := store.List(ctx, ListActivitiesOptions{Limit: 10})
		require.NoError(t, err)
		require.Equal(t, []string{"pr:1:merged", "issue:3:opened", "commit:2:b2", "pr:1:opened", "release:1"}, keys(got))
		require.Equal(t, "Edited", got[3].Title)

		page, err := store.List(ctx, ListActivitiesOptions{Limit: 2, Before: &ActivityCursor{OccurredAt: got[1].OccurredAt, ID: got[1].ID}})
		require.NoError(t, err)
		require.Equal(t, []string{"commit:2:b2", "pr:1:opened"}, keys(page))
	})

	t.Run("filters", func(t *testing.T) {
		got, err := store.List(ctx, ListActivitiesOptions{Type: ActivityTypeCommit, Limit: 10})
		require.NoError(t, err)
		require.Equal(t, []string{"commit:1:a1", "commit:2:b2"}, keys(got), "the type lists the folded commits too")

		got, err = store.List(ctx, ListActivitiesOptions{ActivityFilter: ActivityFilter{Repository: "CLI", ExcludeBots: true}, Limit: 10})
		require.NoError(t, err)
		require.Equal(t, []string{"commit:2:b2"}, keys(got))
	})

	t.Run("counts", func(t *testing.T) {
		got, err := store.Counts(ctx, ActivityFilter{}, day, day.Add(24*time.Hour))
		require.NoError(t, err)
		require.Equal(t, &ActivityCounts{Commits: 2, OpenedPullRequests: 1, MergedPullRequests: 1, OpenedIssues: 1}, got)
	})

	t.Run("daily in the time zone", func(t *testing.T) {
		shanghai, err := time.LoadLocation("Asia/Shanghai")
		require.NoError(t, err)
		// 20:00 UTC is the next day in Shanghai.
		got, err := store.Daily(ctx, ActivityFilter{}, day.Add(-72*time.Hour), shanghai)
		require.NoError(t, err)
		require.Equal(t, []*DailyCount{{Day: "2026-01-08", Count: 1}, {Day: "2026-01-10", Count: 3}, {Day: "2026-01-11", Count: 1}}, got)
	})

	t.Run("repositories and actors", func(t *testing.T) {
		got, err := store.RepositoryCounts(ctx, ActivityFilter{}, day)
		require.NoError(t, err)
		var counts []any
		for _, r := range got {
			counts = append(counts, r.Name, r.Count)
		}
		require.Equal(t, []any{"cli", int64(2), "web", int64(2), "docs", int64(0)}, counts)

		actors, err := store.TopActors(ctx, ActivityFilter{ExcludeBots: true}, day, 10)
		require.NoError(t, err)
		require.Len(t, actors, 2)
		require.Equal(t, "alice", actors[0].Login)
		require.Equal(t, int64(2), actors[0].Count)
	})

	t.Run("retention and removal", func(t *testing.T) {
		removed, err := store.DeleteBefore(ctx, day)
		require.NoError(t, err)
		require.Equal(t, int64(1), removed)

		require.NoError(t, repos.DeleteExcept(ctx, []int64{1, 2}))
		tracked, err := repos.List(ctx)
		require.NoError(t, err)
		require.Len(t, tracked, 2)

		syncedAt := day.Add(time.Hour)
		require.NoError(t, repos.MarkSynced(ctx, tracked[0].ID, syncedAt))
		// Upsert keeps the cursor.
		require.NoError(t, repos.Upsert(ctx, &ActivityRepository{GitHubID: tracked[0].GitHubID, Name: tracked[0].Name}))
		latest, err := repos.LatestSyncedAt(ctx)
		require.NoError(t, err)
		require.NotNil(t, latest)
		require.True(t, syncedAt.Equal(*latest))
	})
}
