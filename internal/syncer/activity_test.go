package syncer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/wuhan005/flamego-stats/internal/db"
	"github.com/wuhan005/flamego-stats/internal/github"
)

var testActivityRepo = &db.ActivityRepository{GitHubID: 100, Name: "web"}

func TestCommitActivity(t *testing.T) {
	at := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	got := commitActivity(testActivityRepo, &github.Commit{
		SHA:         "a1",
		Message:     "  feat: add login (#7)\n\nDetails",
		HTMLURL:     "https://github.com/acme/web/commit/a1",
		Author:      github.Actor{Login: "alice", AvatarURL: "https://avatars/alice"},
		CommittedAt: at,
	})
	require.Equal(t, &db.Activity{
		Key:            "commit:100:a1",
		Type:           db.ActivityTypeCommit,
		Action:         db.ActivityActionCommitted,
		RepositoryID:   100,
		RepositoryName: "web",
		Number:         7,
		SHA:            "a1",
		Title:          "feat: add login (#7)",
		HTMLURL:        "https://github.com/acme/web/commit/a1",
		ActorLogin:     "alice",
		ActorAvatarURL: "https://avatars/alice",
		OccurredAt:     at,
	}, got)

	require.Zero(t, commitActivity(testActivityRepo, &github.Commit{SHA: "b2", Message: "fix #7 in the middle"}).Number)
	require.Nil(t, commitActivity(testActivityRepo, &github.Commit{SHA: "c3", Merge: true}), "merge commits are skipped")
}

func TestIssueActivities(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	closed := created.Add(48 * time.Hour)

	type event struct {
		Key    string
		Type   db.ActivityType
		Action db.ActivityAction
		At     time.Time
		Draft  bool
	}
	events := func(activities []*db.Activity) []event {
		var got []event
		for _, a := range activities {
			got = append(got, event{a.Key, a.Type, a.Action, a.OccurredAt, a.Draft})
		}
		return got
	}

	t.Run("merged pull request", func(t *testing.T) {
		got := issueActivities(testActivityRepo, &github.Issue{
			ID: 10, Number: 7, PullRequest: true, Draft: true, CreatedAt: created, ClosedAt: &closed, MergedAt: &closed,
		})
		require.Equal(t, []event{
			{"pull_request:10:opened", db.ActivityTypePullRequest, db.ActivityActionOpened, created, true},
			{"pull_request:10:merged", db.ActivityTypePullRequest, db.ActivityActionMerged, closed, false},
		}, events(got))
	})

	t.Run("closed issue", func(t *testing.T) {
		got := issueActivities(testActivityRepo, &github.Issue{ID: 11, Number: 8, CreatedAt: created, ClosedAt: &closed})
		require.Equal(t, []event{
			{"issue:11:opened", db.ActivityTypeIssue, db.ActivityActionOpened, created, false},
			{"issue:11:closed", db.ActivityTypeIssue, db.ActivityActionClosed, closed, false},
		}, events(got))
	})

	t.Run("open issue", func(t *testing.T) {
		got := issueActivities(testActivityRepo, &github.Issue{ID: 12, Number: 9, CreatedAt: created})
		require.Equal(t, []event{
			{"issue:12:opened", db.ActivityTypeIssue, db.ActivityActionOpened, created, false},
		}, events(got))
	})
}

func TestReleaseActivityFallsBackToTag(t *testing.T) {
	got := releaseActivity(testActivityRepo, &github.Release{ID: 5, Tag: "v1.0.0", Author: github.Actor{Login: "bot[bot]", Bot: true}})
	require.Equal(t, "release:5", got.Key)
	require.Equal(t, "v1.0.0", got.Title)
	require.Equal(t, "v1.0.0", got.Tag)
	require.True(t, got.ActorBot)
}
