package api

import (
	"github.com/sirupsen/logrus"

	"github.com/wuhan005/flamego-stats/internal/context"
	"github.com/wuhan005/flamego-stats/internal/db"
	"github.com/wuhan005/flamego-stats/internal/dto"
)

var SyncRun syncRunRoute

type syncRunRoute struct{}

// List
// @Summary List the synchronization runs
// @Description The latest synchronization runs, the newest first. The discovered counts chart the trend of the dependents.
// @Produce json
// @Param limit query int false "Number of items, defaults to 20, at most 100"
// @Success 200 {array} dto.SyncRun
// @Failure 500 {string} string "Internal server error"
// @ID listSyncRuns
// @Router /sync-runs [get]
func (syncRunRoute) List(ctx context.Context) error {
	runs, err := db.SyncRuns.List(ctx.Request().Context(), limit(ctx))
	if err != nil {
		logrus.WithContext(ctx.Request().Context()).WithError(err).Error("Failed to list sync runs")
		return ctx.ApiServerError()
	}
	return ctx.ApiSuccess(dto.ToSyncRuns(runs))
}
