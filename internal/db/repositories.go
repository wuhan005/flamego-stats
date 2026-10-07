package db

import (
	"context"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/wuhan005/flamego-stats/internal/dbutil"
)

var _ RepositoriesStore = (*repositories)(nil)

// Repositories is the default instance of the RepositoriesStore.
var Repositories RepositoriesStore

// RepositoriesStore is the persistent interface for the repositories depending on the module.
type RepositoriesStore interface {
	// Upsert saves the repository by GitHubID, the existing one is overwritten and restored even if soft-deleted.
	Upsert(ctx context.Context, repository *Repository) error
	// GetByGitHubID returns the repository with the given GitHub ID, including the soft-deleted ones.
	// It returns ErrRepositoryNotFound if the repository does not exist.
	GetByGitHubID(ctx context.Context, githubID int64) (*Repository, error)
	// Touch refreshes the last seen time of the repository failed to sync, so it is not treated as removed.
	Touch(ctx context.Context, fullName string, seenAt time.Time) error
	// DeleteNotSeenSince soft-deletes the repositories not seen since before, and returns the number of them.
	DeleteNotSeenSince(ctx context.Context, before time.Time) (int64, error)
	// DeleteByGitHubID permanently deletes the repository with the given GitHub ID, e.g. it is no longer public.
	DeleteByGitHubID(ctx context.Context, githubID int64) error
	// List returns the paginated repositories along with the total count.
	List(ctx context.Context, opts ListRepositoriesOptions) ([]*Repository, int64, error)
}

func NewRepositoriesStore(db *gorm.DB) RepositoriesStore {
	return &repositories{db}
}

// Repository is a GitHub repository depending on the module.
type Repository struct {
	dbutil.Model
	// GitHubID is the GitHub repository ID, which is kept after renaming or transferring.
	GitHubID int64 `gorm:"column:github_id;uniqueIndex:idx_repositories_github_id"`

	OwnerID        int64 `gorm:"index"`
	OwnerLogin     string
	OwnerType      string // User or Organization.
	OwnerAvatarURL string

	Name          string
	FullName      string `gorm:"index"`
	Description   string
	Homepage      string
	HTMLURL       string
	Language      string         `gorm:"index"`
	License       string         // SPDX ID.
	Topics        pq.StringArray `gorm:"type:text[];not null;default:'{}'"`
	DefaultBranch string
	Fork          bool
	Archived      bool

	Stars      int `gorm:"index"`
	Forks      int
	Watchers   int
	OpenIssues int

	GitHubCreatedAt time.Time `gorm:"column:github_created_at"`
	GitHubUpdatedAt time.Time `gorm:"column:github_updated_at"`
	PushedAt        time.Time `gorm:"index"`

	// GoModPath is the path of the go.mod requiring the module, it is empty if not found.
	GoModPath string
	// GoVersion is the version of the go directive in the go.mod.
	GoVersion string
	// ModuleVersion is the required version of the module in the go.mod, with the replace directives applied.
	ModuleVersion string `gorm:"index"`
	// ModuleMinorVersion is the vMAJOR.MINOR of ModuleVersion for grouping by the minor versions.
	ModuleMinorVersion string
	// RelatedModules are the other required modules of the same organization, e.g. github.com/flamego/session.
	RelatedModules pq.StringArray `gorm:"type:text[];not null;default:'{}'"`

	// LastSeenAt is the last time the repository was discovered as a dependent.
	LastSeenAt time.Time `gorm:"index"`
	// SyncedAt is the last time the repository was fetched from GitHub.
	SyncedAt time.Time
}

type repositories struct {
	*gorm.DB
}

// upsertColumns are the columns overwritten on conflict, id and created_at keep the values of the first insert.
var upsertColumns = []string{
	"updated_at", "deleted_at",
	"owner_id", "owner_login", "owner_type", "owner_avatar_url",
	"name", "full_name", "description", "homepage", "html_url", "language", "license", "topics", "default_branch", "fork", "archived",
	"stars", "forks", "watchers", "open_issues",
	"github_created_at", "github_updated_at", "pushed_at",
	"go_mod_path", "go_version", "module_version", "module_minor_version", "related_modules",
	"last_seen_at", "synced_at",
}

func (db *repositories) Upsert(ctx context.Context, repository *Repository) error {
	// A nil pq.StringArray is written as NULL, which violates the not null constraint.
	if repository.Topics == nil {
		repository.Topics = pq.StringArray{}
	}
	if repository.RelatedModules == nil {
		repository.RelatedModules = pq.StringArray{}
	}
	repository.DeletedAt = gorm.DeletedAt{}

	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "github_id"}},
		DoUpdates: clause.AssignmentColumns(upsertColumns),
	}).Create(repository).Error
}

var ErrRepositoryNotFound = errors.New("repository does not exist")

func (db *repositories) GetByGitHubID(ctx context.Context, githubID int64) (*Repository, error) {
	var repository Repository
	if err := db.WithContext(ctx).Unscoped().Where("github_id = ?", githubID).First(&repository).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRepositoryNotFound
		}
		return nil, err
	}
	return &repository, nil
}

func (db *repositories) Touch(ctx context.Context, fullName string, seenAt time.Time) error {
	return db.WithContext(ctx).Model(&Repository{}).
		Where("LOWER(full_name) = ?", strings.ToLower(fullName)).
		UpdateColumn("last_seen_at", seenAt).Error
}

func (db *repositories) DeleteNotSeenSince(ctx context.Context, before time.Time) (int64, error) {
	result := db.WithContext(ctx).Where("last_seen_at < ?", before).Delete(&Repository{})
	return result.RowsAffected, result.Error
}

func (db *repositories) DeleteByGitHubID(ctx context.Context, githubID int64) error {
	return db.WithContext(ctx).Unscoped().Where("github_id = ?", githubID).Delete(&Repository{}).Error
}

// RepositorySort is the order of the repository list.
type RepositorySort string

const (
	RepositorySortStars   RepositorySort = "stars"
	RepositorySortForks   RepositorySort = "forks"
	RepositorySortPushed  RepositorySort = "pushed"
	RepositorySortCreated RepositorySort = "created"
	RepositorySortName    RepositorySort = "name"
)

// repositoryOrders end with id as the tiebreaker to keep the pagination stable.
var repositoryOrders = map[RepositorySort]string{
	RepositorySortStars:   "stars DESC, id ASC",
	RepositorySortForks:   "forks DESC, id ASC",
	RepositorySortPushed:  "pushed_at DESC, id ASC",
	RepositorySortCreated: "github_created_at DESC, id ASC",
	RepositorySortName:    "LOWER(full_name) ASC, id ASC",
}

// IsValid reports whether the sort is supported, the empty sort is the same as stars.
func (s RepositorySort) IsValid() bool {
	if s == "" {
		return true
	}
	_, ok := repositoryOrders[s]
	return ok
}

// ScopeOptions are the filters shared by the statistics and the repository list.
type ScopeOptions struct {
	// IncludeForks includes the forked repositories, they are excluded if false.
	IncludeForks bool
}

func (opts ScopeOptions) apply(q *gorm.DB) *gorm.DB {
	if !opts.IncludeForks {
		q = q.Where("fork = ?", false)
	}
	return q
}

// ListRepositoriesOptions are the options of listing the repositories.
type ListRepositoriesOptions struct {
	dbutil.Pagination
	ScopeOptions
	// Sort is the same as stars if empty.
	Sort RepositorySort
	// Keyword matches the full name and the description case-insensitively.
	Keyword string
	// Topic, Language, Owner and ModuleVersion filter the repositories exactly when not empty, Owner is case-insensitive.
	Topic         string
	Language      string
	Owner         string
	ModuleVersion string
}

func (db *repositories) List(ctx context.Context, opts ListRepositoriesOptions) ([]*Repository, int64, error) {
	q := opts.apply(db.WithContext(ctx).Model(&Repository{}))
	if k := strings.TrimSpace(opts.Keyword); k != "" {
		pattern := "%" + dbutil.EscapeLike(strings.ToLower(k)) + "%"
		q = q.Where("LOWER(full_name) LIKE ? OR LOWER(description) LIKE ?", pattern, pattern)
	}
	if opts.Topic != "" {
		q = q.Where("? = ANY(topics)", opts.Topic)
	}
	if opts.Language != "" {
		q = q.Where("language = ?", opts.Language)
	}
	if opts.Owner != "" {
		q = q.Where("LOWER(owner_login) = ?", strings.ToLower(opts.Owner))
	}
	if opts.ModuleVersion != "" {
		q = q.Where("module_version = ?", opts.ModuleVersion)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, errors.Wrap(err, "count")
	}

	order, ok := repositoryOrders[opts.Sort]
	if !ok {
		order = repositoryOrders[RepositorySortStars]
	}
	limit, offset := opts.LimitOffset()
	var repositories []*Repository
	return repositories, total, q.Order(order).Limit(limit).Offset(offset).Find(&repositories).Error
}
