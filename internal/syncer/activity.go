package syncer

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/wuhan005/flamego-stats/internal/conf"
	"github.com/wuhan005/flamego-stats/internal/db"
	"github.com/wuhan005/flamego-stats/internal/dbutil"
	"github.com/wuhan005/flamego-stats/internal/github"
)

// activityOverlap is subtracted from the cursor, so the activities written around the previous run are not missed.
const activityOverlap = 10 * time.Minute

// SyncActivities collects the commits, pull requests, issues and releases of the public repositories of the organization,
// it returns ErrLocked if another worker is synchronizing them.
func (s *Syncer) SyncActivities(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return errors.Wrap(err, "get database connection")
	}
	unlock, ok, err := dbutil.TryAdvisoryLock(ctx, sqlDB, db.AdvisoryLockActivity)
	if err != nil {
		return errors.Wrap(err, "acquire lock")
	}
	if !ok {
		return ErrLocked
	}
	defer unlock()

	logger := logrus.WithContext(ctx).WithField("organization", conf.Activity.Organization)
	startedAt := dbutil.Now()
	cutoff := startedAt.Add(-conf.Activity.Retention)

	repositories, err := s.trackRepositories(ctx)
	if err != nil {
		return errors.Wrap(err, "track repositories")
	}

	var (
		attempted     int64
		saved, failed atomic.Int64
	)
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(conf.Sync.Concurrency)
	for _, repo := range repositories {
		// An archived repository is read-only, nothing happens after the last synchronization.
		if repo.Archived && repo.SyncedAt != nil {
			continue
		}
		attempted++
		g.Go(func() error {
			n, err := s.syncRepositoryActivities(gctx, repo, cutoff, startedAt)
			if err != nil {
				if gctx.Err() != nil {
					return gctx.Err()
				}
				logger.WithError(err).WithField("repository", repo.FullName).Warn("Failed to synchronize activities")
				failed.Add(1)
				return nil
			}
			saved.Add(int64(n))
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	removed, err := db.Activities.DeleteBefore(ctx, cutoff)
	if err != nil {
		return errors.Wrap(err, "delete expired activities")
	}

	if attempted > 0 && failed.Load() == attempted {
		return errors.Newf("all %d repositories failed", attempted)
	}
	logger.WithFields(logrus.Fields{
		"repositories": len(repositories),
		"saved":        saved.Load(),
		"failed":       failed.Load(),
		"removed":      removed,
	}).Info("Activities synchronized")
	return nil
}

// trackRepositories saves the public source repositories of the organization, and returns all the tracked ones.
func (s *Syncer) trackRepositories(ctx context.Context) ([]*db.ActivityRepository, error) {
	repos, err := s.client.ListOrgRepositories(ctx, conf.Activity.Organization)
	if err != nil {
		return nil, errors.Wrap(err, "list repositories")
	}

	ids := make([]int64, 0, len(repos))
	for _, repo := range repos {
		// The activities of a fork happen in its upstream.
		if !repo.Public || repo.Fork {
			continue
		}
		if err := db.ActivityRepositories.Upsert(ctx, &db.ActivityRepository{
			GitHubID:      repo.ID,
			Name:          repo.Name,
			FullName:      repo.FullName,
			Description:   repo.Description,
			HTMLURL:       repo.HTMLURL,
			DefaultBranch: repo.DefaultBranch,
			Archived:      repo.Archived,
			Stars:         repo.Stars,
			PushedAt:      repo.PushedAt,
		}); err != nil {
			return nil, errors.Wrapf(err, "save %s", repo.FullName)
		}
		ids = append(ids, repo.ID)
	}
	if err := db.ActivityRepositories.DeleteExcept(ctx, ids); err != nil {
		return nil, errors.Wrap(err, "delete removed repositories")
	}
	return db.ActivityRepositories.List(ctx)
}

// syncRepositoryActivities saves the activities of the repository since its cursor, and returns the number of them.
func (s *Syncer) syncRepositoryActivities(ctx context.Context, repo *db.ActivityRepository, cutoff, startedAt time.Time) (int, error) {
	since := cutoff
	if repo.SyncedAt != nil {
		since = maxTime(cutoff, repo.SyncedAt.Add(-activityOverlap))
	}

	var activities []*db.Activity
	// Pushing the commits and the tags of the releases updates the push time, so they are skipped if it is older.
	if !repo.PushedAt.Before(since) {
		commits, err := s.client.ListCommits(ctx, repo.FullName, repo.DefaultBranch, since)
		if err != nil && !errors.Is(err, github.ErrNotFound) {
			return 0, errors.Wrap(err, "list commits")
		}
		for _, commit := range commits {
			if a := commitActivity(repo, commit); a != nil {
				activities = append(activities, a)
			}
		}

		releases, err := s.client.ListReleases(ctx, repo.FullName, since)
		if err != nil {
			return 0, errors.Wrap(err, "list releases")
		}
		for _, release := range releases {
			activities = append(activities, releaseActivity(repo, release))
		}
	}

	issues, err := s.client.ListIssues(ctx, repo.FullName, since)
	if err != nil {
		return 0, errors.Wrap(err, "list issues")
	}
	for _, issue := range issues {
		activities = append(activities, issueActivities(repo, issue)...)
	}

	// An old issue updated recently is listed too, its opening is older than the retention.
	kept := activities[:0]
	for _, a := range activities {
		if !a.OccurredAt.Before(cutoff) {
			kept = append(kept, a)
		}
	}

	if err := db.Activities.Upsert(ctx, kept); err != nil {
		return 0, errors.Wrap(err, "save activities")
	}
	if err := db.Activities.RenameRepository(ctx, repo.GitHubID, repo.Name); err != nil {
		return 0, errors.Wrap(err, "rename repository")
	}
	if err := db.ActivityRepositories.MarkSynced(ctx, repo.ID, startedAt); err != nil {
		return 0, errors.Wrap(err, "mark synced")
	}
	return len(kept), nil
}

// squashedPullRequest matches the suffix GitHub appends to the title of a squash-merged commit, e.g. "fix: typo (#42)".
var squashedPullRequest = regexp.MustCompile(`\(#(\d+)\)$`)

// commitActivity returns the activity of the commit, it is nil for a merge commit, which the merged pull request stands for.
func commitActivity(repo *db.ActivityRepository, commit *github.Commit) *db.Activity {
	if commit.Merge {
		return nil
	}
	title, _, _ := strings.Cut(strings.TrimSpace(commit.Message), "\n")
	title = strings.TrimSpace(title)
	number := 0
	if m := squashedPullRequest.FindStringSubmatch(title); m != nil {
		number, _ = strconv.Atoi(m[1])
	}
	return newActivity(repo, commit.Author, db.Activity{
		Key:        fmt.Sprintf("commit:%d:%s", repo.GitHubID, commit.SHA),
		Type:       db.ActivityTypeCommit,
		Action:     db.ActivityActionCommitted,
		Number:     number,
		SHA:        commit.SHA,
		Title:      title,
		HTMLURL:    commit.HTMLURL,
		OccurredAt: commit.CommittedAt,
	})
}

// issueActivities returns the opening of the issue or the pull request, and the merging or the closing if it happened.
func issueActivities(repo *db.ActivityRepository, issue *github.Issue) []*db.Activity {
	typ := db.ActivityTypeIssue
	if issue.PullRequest {
		typ = db.ActivityTypePullRequest
	}
	at := func(action db.ActivityAction, occurredAt time.Time) *db.Activity {
		return newActivity(repo, issue.Author, db.Activity{
			Key:        fmt.Sprintf("%s:%d:%s", typ, issue.ID, action),
			Type:       typ,
			Action:     action,
			Number:     issue.Number,
			Title:      issue.Title,
			HTMLURL:    issue.HTMLURL,
			Draft:      issue.Draft && action == db.ActivityActionOpened,
			OccurredAt: occurredAt,
		})
	}

	activities := []*db.Activity{at(db.ActivityActionOpened, issue.CreatedAt)}
	switch {
	case issue.MergedAt != nil:
		activities = append(activities, at(db.ActivityActionMerged, *issue.MergedAt))
	case issue.ClosedAt != nil:
		activities = append(activities, at(db.ActivityActionClosed, *issue.ClosedAt))
	}
	return activities
}

func releaseActivity(repo *db.ActivityRepository, release *github.Release) *db.Activity {
	title := release.Name
	if title == "" {
		title = release.Tag
	}
	return newActivity(repo, release.Author, db.Activity{
		Key:        fmt.Sprintf("release:%d", release.ID),
		Type:       db.ActivityTypeRelease,
		Action:     db.ActivityActionPublished,
		Tag:        release.Tag,
		Title:      title,
		HTMLURL:    release.HTMLURL,
		Prerelease: release.Prerelease,
		OccurredAt: release.PublishedAt,
	})
}

// newActivity fills the repository and the actor of the activity.
func newActivity(repo *db.ActivityRepository, actor github.Actor, a db.Activity) *db.Activity {
	a.RepositoryID = repo.GitHubID
	a.RepositoryName = repo.Name
	a.ActorLogin = actor.Login
	a.ActorAvatarURL = actor.AvatarURL
	a.ActorBot = actor.Bot
	return &a
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
