package dto

import (
	"time"

	"github.com/wuhan005/flamego-stats/internal/db"
)

// SyncRun is a synchronization run of the worker.
type SyncRun struct {
	ID            int64      `json:"id"`
	Status        string     `json:"status" enums:"running,succeeded,failed"`
	StartedAt     time.Time  `json:"startedAt"`
	FinishedAt    *time.Time `json:"finishedAt" extensions:"x-nullable"`
	ReportedTotal int        `json:"reportedTotal"`
	Discovered    int        `json:"discovered"`
	Synced        int        `json:"synced"`
	Skipped       int        `json:"skipped"`
	Failed        int        `json:"failed"`
	Removed       int        `json:"removed"`
	Error         string     `json:"error"`
} // @name SyncRun

func ToSyncRuns(runs []*db.SyncRun) []*SyncRun {
	items := make([]*SyncRun, 0, len(runs))
	for _, run := range runs {
		items = append(items, &SyncRun{
			ID:            run.ID,
			Status:        string(run.Status),
			StartedAt:     run.StartedAt,
			FinishedAt:    run.FinishedAt,
			ReportedTotal: run.ReportedTotal,
			Discovered:    run.Discovered,
			Synced:        run.Synced,
			Skipped:       run.Skipped,
			Failed:        run.Failed,
			Removed:       run.Removed,
			Error:         run.Error,
		})
	}
	return items
}
