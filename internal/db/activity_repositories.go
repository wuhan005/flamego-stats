package db

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/wuhan005/flamego-stats/internal/dbutil"
)

var _ ActivityRepositoriesStore = (*activityRepositories)(nil)

// ActivityRepositories is the default instance of the ActivityRepositoriesStore.
var ActivityRepositories ActivityRepositoriesStore

// ActivityRepositoriesStore is the persistent interface for the repositories of the organization whose activities are tracked.
type ActivityRepositoriesStore interface {
	// Upsert saves the repository by GitHubID and restores it if soft-deleted, the synchronization cursor is kept.
	Upsert(ctx context.Context, repository *ActivityRepository) error
	// DeleteExcept soft-deletes the repositories not in githubIDs, e.g. deleted or transferred out of the organization.
	DeleteExcept(ctx context.Context, githubIDs []int64) error
	// List returns the tracked repositories ordered by name.
	List(ctx context.Context) ([]*ActivityRepository, error)
	// MarkSynced sets the cursor of the repository, the next synchronization continues from it.
	MarkSynced(ctx context.Context, id int64, syncedAt time.Time) error
	// LatestSyncedAt returns the latest cursor of all the repositories, it is nil if none has been synchronized.
	LatestSyncedAt(ctx context.Context) (*time.Time, error)
}

func NewActivityRepositoriesStore(db *gorm.DB) ActivityRepositoriesStore {
	return &activityRepositories{db}
}

// ActivityRepository is a public repository of the organization whose activities are tracked.
type ActivityRepository struct {
	dbutil.Model
	// GitHubID is the GitHub repository ID, which is kept after renaming.
	GitHubID int64 `gorm:"column:github_id;uniqueIndex:idx_activity_repositories_github_id"`

	Name          string
	FullName      string
	Description   string
	HTMLURL       string
	DefaultBranch string
	Archived      bool
	Stars         int
	PushedAt      time.Time

	// SyncedAt is the start of the last succeeded synchronization of the repository, it is nil if never synchronized.
	SyncedAt *time.Time
}

type activityRepositories struct {
	*gorm.DB
}

func (db *activityRepositories) Upsert(ctx context.Context, repository *ActivityRepository) error {
	repository.DeletedAt = gorm.DeletedAt{}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "github_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"updated_at", "deleted_at",
			"name", "full_name", "description", "html_url", "default_branch", "archived", "stars", "pushed_at",
		}),
	}).Omit("synced_at").Create(repository).Error
}

func (db *activityRepositories) DeleteExcept(ctx context.Context, githubIDs []int64) error {
	q := db.WithContext(ctx)
	if len(githubIDs) > 0 {
		q = q.Where("github_id NOT IN ?", githubIDs)
	} else {
		q = q.Where("TRUE")
	}
	return q.Delete(&ActivityRepository{}).Error
}

func (db *activityRepositories) List(ctx context.Context) ([]*ActivityRepository, error) {
	var repositories []*ActivityRepository
	return repositories, db.WithContext(ctx).Order("LOWER(name) ASC").Find(&repositories).Error
}

func (db *activityRepositories) MarkSynced(ctx context.Context, id int64, syncedAt time.Time) error {
	return db.WithContext(ctx).Model(&ActivityRepository{}).Where("id = ?", id).UpdateColumn("synced_at", syncedAt).Error
}

func (db *activityRepositories) LatestSyncedAt(ctx context.Context) (*time.Time, error) {
	var latest *time.Time
	return latest, db.WithContext(ctx).Model(&ActivityRepository{}).Select("MAX(synced_at)").Scan(&latest).Error
}
