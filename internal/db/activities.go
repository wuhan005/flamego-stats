package db

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var _ ActivitiesStore = (*activities)(nil)

// Activities is the default instance of the ActivitiesStore.
var Activities ActivitiesStore

// ActivitiesStore is the persistent interface for the activities of the repositories of the organization.
type ActivitiesStore interface {
	// Upsert saves the activities by Key, the existing ones are overwritten.
	Upsert(ctx context.Context, activities []*Activity) error
	// RenameRepository updates the repository name of the activities of the repository.
	RenameRepository(ctx context.Context, repositoryID int64, name string) error
	// DeleteBefore deletes the activities that occurred before the time, and returns the number of them.
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
	// List returns the activities in the order of the feed, the newest first.
	List(ctx context.Context, opts ListActivitiesOptions) ([]*Activity, error)
	// Counts returns the numbers of the activities of each kind that occurred in [since, until).
	Counts(ctx context.Context, filter ActivityFilter, since, until time.Time) (*ActivityCounts, error)
	// Daily returns the number of the feed activities of each day since the time in loc, the days without any are omitted.
	Daily(ctx context.Context, filter ActivityFilter, since time.Time, loc *time.Location) ([]*DailyCount, error)
	// RepositoryCounts returns every tracked repository with the number of its feed activities since the time, the most active first.
	RepositoryCounts(ctx context.Context, filter ActivityFilter, since time.Time) ([]*RepositoryActivityStat, error)
	// TopActors returns the actors ordered by the number of their feed activities since the time.
	TopActors(ctx context.Context, filter ActivityFilter, since time.Time, limit int) ([]*ActorStat, error)
}

func NewActivitiesStore(db *gorm.DB) ActivitiesStore {
	return &activities{db}
}

// ActivityType is the kind of the subject of an activity.
type ActivityType string

const (
	ActivityTypeCommit      ActivityType = "commit"
	ActivityTypePullRequest ActivityType = "pull_request"
	ActivityTypeIssue       ActivityType = "issue"
	ActivityTypeRelease     ActivityType = "release"
)

// IsValid reports whether the type is supported, the empty type means all of them.
func (t ActivityType) IsValid() bool {
	switch t {
	case "", ActivityTypeCommit, ActivityTypePullRequest, ActivityTypeIssue, ActivityTypeRelease:
		return true
	}
	return false
}

// ActivityAction is what happened to the subject of an activity.
type ActivityAction string

const (
	ActivityActionCommitted ActivityAction = "committed"
	ActivityActionOpened    ActivityAction = "opened"
	ActivityActionMerged    ActivityAction = "merged"
	ActivityActionClosed    ActivityAction = "closed"
	ActivityActionPublished ActivityAction = "published"
)

// Activity is something that happened in a repository of the organization, e.g. a pull request was merged.
type Activity struct {
	ID int64 `gorm:"primarykey"`
	// Key identifies the activity on GitHub, e.g. pull_request:<issue ID>:merged, so the repeated synchronizations update it in place.
	Key    string `gorm:"uniqueIndex"`
	Type   ActivityType
	Action ActivityAction

	// RepositoryID is the GitHub ID of the repository.
	RepositoryID   int64 `gorm:"index"`
	RepositoryName string

	// Number is the number of the pull request or the issue. For a commit, it is the pull request squash-merged as it, or 0.
	Number int
	// SHA is the hash of the commit.
	SHA string
	// Tag is the tag name of the release.
	Tag     string
	Title   string
	HTMLURL string
	// Draft marks the opened pull request as a draft, Prerelease marks the release as a pre-release.
	Draft      bool
	Prerelease bool

	// ActorLogin is the name in the commit if the commit author is not linked to any account, ActorAvatarURL is empty then.
	ActorLogin     string
	ActorAvatarURL string
	ActorBot       bool

	OccurredAt time.Time `gorm:"index"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ActivityFilter are the filters shared by the feed and the statistics.
type ActivityFilter struct {
	// Repository is the repository name matched case-insensitively, all the repositories if empty.
	Repository string
	// ExcludeBots excludes the activities of the GitHub Apps.
	ExcludeBots bool
}

func (f ActivityFilter) apply(q *gorm.DB) *gorm.DB {
	if f.Repository != "" {
		q = q.Where("LOWER(activities.repository_name) = ?", strings.ToLower(f.Repository))
	}
	if f.ExcludeBots {
		q = q.Where("activities.actor_bot = ?", false)
	}
	return q
}

// notFolded excludes the commits squash-merged from the pull requests, the merged pull requests stand for them in the feed.
const notFolded = `NOT (activities.type = 'commit' AND activities.number > 0 AND EXISTS (
	SELECT 1 FROM activities AS pr
	WHERE pr.repository_id = activities.repository_id AND pr.type = 'pull_request' AND pr.action = 'merged' AND pr.number = activities.number
))`

// ActivityCursor is the position of an activity in the feed.
type ActivityCursor struct {
	OccurredAt time.Time
	ID         int64
}

// ListActivitiesOptions are the options of listing the activities.
type ListActivitiesOptions struct {
	ActivityFilter
	// Type lists all the types if empty, with the squash-merged commits folded into their pull requests.
	Type ActivityType
	// Before lists the activities after the cursor in the feed if not nil.
	Before *ActivityCursor
	Limit  int
}

type activities struct {
	*gorm.DB
}

// upsertActivityColumns are the columns overwritten on conflict, e.g. the title of a pull request may be edited.
var upsertActivityColumns = []string{
	"updated_at", "repository_name", "number", "title", "html_url", "draft", "prerelease",
	"actor_login", "actor_avatar_url", "actor_bot", "occurred_at",
}

func (db *activities) Upsert(ctx context.Context, activities []*Activity) error {
	if len(activities) == 0 {
		return nil
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns(upsertActivityColumns),
	}).CreateInBatches(activities, 200).Error
}

func (db *activities) RenameRepository(ctx context.Context, repositoryID int64, name string) error {
	return db.WithContext(ctx).Model(&Activity{}).
		Where("repository_id = ? AND repository_name <> ?", repositoryID, name).
		UpdateColumn("repository_name", name).Error
}

func (db *activities) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	result := db.WithContext(ctx).Where("occurred_at < ?", before).Delete(&Activity{})
	return result.RowsAffected, result.Error
}

func (db *activities) List(ctx context.Context, opts ListActivitiesOptions) ([]*Activity, error) {
	q := opts.apply(db.WithContext(ctx).Model(&Activity{}))
	if opts.Type == "" {
		q = q.Where(notFolded)
	} else {
		q = q.Where("type = ?", opts.Type)
	}
	if opts.Before != nil {
		q = q.Where("(occurred_at, id) < (?, ?)", opts.Before.OccurredAt, opts.Before.ID)
	}
	var activities []*Activity
	return activities, q.Order("occurred_at DESC, id DESC").Limit(opts.Limit).Find(&activities).Error
}

// ActivityCounts is the numbers of the activities of each kind.
type ActivityCounts struct {
	Commits            int64
	OpenedPullRequests int64
	MergedPullRequests int64
	OpenedIssues       int64
	ClosedIssues       int64
	Releases           int64
}

func (db *activities) Counts(ctx context.Context, filter ActivityFilter, since, until time.Time) (*ActivityCounts, error) {
	var counts ActivityCounts
	return &counts, filter.apply(db.WithContext(ctx).Model(&Activity{})).
		Where("occurred_at >= ? AND occurred_at < ?", since, until).
		Select(`COUNT(*) FILTER (WHERE type = 'commit') AS commits,
			COUNT(*) FILTER (WHERE type = 'pull_request' AND action = 'opened') AS opened_pull_requests,
			COUNT(*) FILTER (WHERE type = 'pull_request' AND action = 'merged') AS merged_pull_requests,
			COUNT(*) FILTER (WHERE type = 'issue' AND action = 'opened') AS opened_issues,
			COUNT(*) FILTER (WHERE type = 'issue' AND action = 'closed') AS closed_issues,
			COUNT(*) FILTER (WHERE type = 'release') AS releases`).
		Scan(&counts).Error
}

// DailyCount is the number of the activities of a day.
type DailyCount struct {
	// Day is in the format of 2006-01-02.
	Day   string
	Count int64
}

func (db *activities) Daily(ctx context.Context, filter ActivityFilter, since time.Time, loc *time.Location) ([]*DailyCount, error) {
	var days []*DailyCount
	return days, filter.apply(db.WithContext(ctx).Model(&Activity{})).
		Where("occurred_at >= ?", since).
		Where(notFolded).
		Select("to_char(occurred_at AT TIME ZONE ?, 'YYYY-MM-DD') AS day, COUNT(*) AS count", loc.String()).
		Group("day").
		Order("day ASC").
		Scan(&days).Error
}

// RepositoryActivityStat is a tracked repository with the number of its activities.
type RepositoryActivityStat struct {
	Name        string
	Description string
	HTMLURL     string
	Archived    bool
	Count       int64
}

func (db *activities) RepositoryCounts(ctx context.Context, filter ActivityFilter, since time.Time) ([]*RepositoryActivityStat, error) {
	counts := filter.apply(db.WithContext(ctx).Model(&Activity{})).
		Where("occurred_at >= ?", since).
		Where(notFolded).
		Select("repository_id, COUNT(*) AS count").
		Group("repository_id")

	var stats []*RepositoryActivityStat
	return stats, db.WithContext(ctx).Model(&ActivityRepository{}).
		Joins("LEFT JOIN (?) AS counts ON counts.repository_id = activity_repositories.github_id", counts).
		Select("name, description, html_url, archived, COALESCE(counts.count, 0) AS count").
		Order("count DESC, LOWER(name) ASC").
		Scan(&stats).Error
}

// ActorStat is an actor with the number of their activities.
type ActorStat struct {
	Login     string
	AvatarURL string
	Bot       bool
	Count     int64
}

func (db *activities) TopActors(ctx context.Context, filter ActivityFilter, since time.Time, limit int) ([]*ActorStat, error) {
	var actors []*ActorStat
	// An actor keeps the login and the avatar across the activities, so MAX picks a consistent value.
	return actors, filter.apply(db.WithContext(ctx).Model(&Activity{})).
		Where("occurred_at >= ?", since).
		Where(notFolded).
		Select("actor_login AS login, MAX(actor_avatar_url) AS avatar_url, BOOL_OR(actor_bot) AS bot, COUNT(*) AS count").
		Group("actor_login").
		Order("count DESC, login ASC").
		Limit(limit).
		Scan(&actors).Error
}
