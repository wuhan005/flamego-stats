package db

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"gorm.io/gorm"

	"github.com/wuhan005/flamego-stats/internal/dbutil"
)

var _ SyncRunsStore = (*syncRuns)(nil)

// SyncRuns is the default instance of the SyncRunsStore.
var SyncRuns SyncRunsStore

// SyncRunsStore is the persistent interface for the synchronization runs of the worker.
type SyncRunsStore interface {
	// Start creates a running synchronization run.
	Start(ctx context.Context) (*SyncRun, error)
	// Finish saves the result of the run, the status is failed if err is not nil.
	Finish(ctx context.Context, runID int64, result SyncRunResult, err error) error
	// AbortRunning marks all the running runs as failed to recover from the crashed workers, and returns the number of them.
	AbortRunning(ctx context.Context, reason string) (int64, error)
	// List returns the latest synchronization runs, the newest first.
	List(ctx context.Context, limit int) ([]*SyncRun, error)
	// GetLatestSucceeded returns the latest succeeded synchronization run.
	// It returns ErrSyncRunNotFound if there is no succeeded run.
	GetLatestSucceeded(ctx context.Context) (*SyncRun, error)
}

func NewSyncRunsStore(db *gorm.DB) SyncRunsStore {
	return &syncRuns{db}
}

type SyncRunStatus string

const (
	SyncRunStatusRunning   SyncRunStatus = "running"
	SyncRunStatusSucceeded SyncRunStatus = "succeeded"
	SyncRunStatusFailed    SyncRunStatus = "failed"
)

// SyncRunResult is the counts of a synchronization run.
type SyncRunResult struct {
	// ReportedTotal is the approximate number of dependents shown on GitHub, it is 0 without the dependents source.
	ReportedTotal int
	// Discovered is the number of distinct repositories discovered in the run.
	Discovered int
	// Synced is the number of repositories saved.
	Synced int
	// Skipped is the number of repositories that no longer exist, are duplicated or ignored.
	Skipped int
	// Failed is the number of repositories failed to fetch, they keep the data of the previous run.
	Failed int
	// Removed is the number of repositories removed because they are no longer discovered.
	Removed int
}

// SyncRun is a synchronization run of the worker.
type SyncRun struct {
	ID            int64         `gorm:"primarykey"`
	Status        SyncRunStatus `gorm:"index"`
	StartedAt     time.Time     `gorm:"index"`
	FinishedAt    *time.Time
	SyncRunResult `gorm:"embedded"`
	Error         string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type syncRuns struct {
	*gorm.DB
}

func (db *syncRuns) Start(ctx context.Context) (*SyncRun, error) {
	run := &SyncRun{
		Status:    SyncRunStatusRunning,
		StartedAt: dbutil.Now(),
	}
	if err := db.WithContext(ctx).Create(run).Error; err != nil {
		return nil, err
	}
	return run, nil
}

func (db *syncRuns) Finish(ctx context.Context, runID int64, result SyncRunResult, err error) error {
	status, message := SyncRunStatusSucceeded, ""
	if err != nil {
		status, message = SyncRunStatusFailed, err.Error()
	}
	return db.WithContext(ctx).Model(&SyncRun{}).Where("id = ?", runID).Updates(map[string]interface{}{
		"status":         status,
		"finished_at":    dbutil.Now(),
		"reported_total": result.ReportedTotal,
		"discovered":     result.Discovered,
		"synced":         result.Synced,
		"skipped":        result.Skipped,
		"failed":         result.Failed,
		"removed":        result.Removed,
		"error":          message,
	}).Error
}

func (db *syncRuns) AbortRunning(ctx context.Context, reason string) (int64, error) {
	result := db.WithContext(ctx).Model(&SyncRun{}).Where("status = ?", SyncRunStatusRunning).Updates(map[string]interface{}{
		"status":      SyncRunStatusFailed,
		"finished_at": dbutil.Now(),
		"error":       reason,
	})
	return result.RowsAffected, result.Error
}

func (db *syncRuns) List(ctx context.Context, limit int) ([]*SyncRun, error) {
	var runs []*SyncRun
	return runs, db.WithContext(ctx).Order("id DESC").Limit(limit).Find(&runs).Error
}

var ErrSyncRunNotFound = errors.New("sync run does not exist")

func (db *syncRuns) GetLatestSucceeded(ctx context.Context) (*SyncRun, error) {
	var run SyncRun
	if err := db.WithContext(ctx).Where("status = ?", SyncRunStatusSucceeded).Order("id DESC").First(&run).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSyncRunNotFound
		}
		return nil, err
	}
	return &run, nil
}
