package api

import (
	"net/http"

	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"

	"github.com/wuhan005/flamego-stats/internal/conf"
	"github.com/wuhan005/flamego-stats/internal/context"
	"github.com/wuhan005/flamego-stats/internal/db"
	"github.com/wuhan005/flamego-stats/internal/dto"
)

var Stats statsRoute

type statsRoute struct{}

// Overview
// @Summary Get the overview
// @Description The totals of the dependents, owners, stars and forks, along with the latest succeeded synchronization.
// @Produce json
// @Param includeForks query bool false "Include the forked repositories, defaults to false"
// @Success 200 {object} dto.Overview
// @Failure 500 {string} string "Internal server error"
// @ID getOverview
// @Router /overview [get]
func (statsRoute) Overview(ctx context.Context) error {
	overview, err := db.Stats.Overview(ctx.Request().Context(), scopeOptions(ctx))
	if err != nil {
		logrus.WithContext(ctx.Request().Context()).WithError(err).Error("Failed to get overview")
		return ctx.ApiServerError()
	}

	resp := dto.Overview{
		Repository:   conf.Sync.Repository,
		Module:       conf.Sync.Module,
		Repositories: overview.Repositories,
		Owners:       overview.Owners,
		Stars:        overview.Stars,
		Forks:        overview.Forks,
		Resolved:     overview.Resolved,
	}
	run, err := db.SyncRuns.GetLatestSucceeded(ctx.Request().Context())
	switch {
	case err == nil:
		resp.ReportedTotal = run.ReportedTotal
		resp.LastSyncedAt = run.FinishedAt
	case !errors.Is(err, db.ErrSyncRunNotFound):
		logrus.WithContext(ctx.Request().Context()).WithError(err).Error("Failed to get latest sync run")
		return ctx.ApiServerError()
	}
	return ctx.ApiSuccess(resp)
}

// Owners
// @Summary Rank the owners
// @Description The owners ordered by the number of dependents, then by the total stars.
// @Produce json
// @Param limit query int false "Number of items, defaults to 20, at most 100"
// @Param includeForks query bool false "Include the forked repositories, defaults to false"
// @Success 200 {array} dto.OwnerRank
// @Failure 500 {string} string "Internal server error"
// @ID listTopOwners
// @Router /stats/owners [get]
func (statsRoute) Owners(ctx context.Context) error {
	owners, err := db.Stats.TopOwners(ctx.Request().Context(), db.TopOptions{ScopeOptions: scopeOptions(ctx), Limit: limit(ctx)})
	if err != nil {
		logrus.WithContext(ctx.Request().Context()).WithError(err).Error("Failed to list top owners")
		return ctx.ApiServerError()
	}
	return ctx.ApiSuccess(dto.ToOwnerRanks(owners))
}

// Topics
// @Summary List the topics
// @Description The repository topics ordered by the number of dependents.
// @Produce json
// @Param limit query int false "Number of items, defaults to 20, at most 100"
// @Param includeForks query bool false "Include the forked repositories, defaults to false"
// @Success 200 {array} dto.CountItem
// @Failure 500 {string} string "Internal server error"
// @ID listTopics
// @Router /stats/topics [get]
func (statsRoute) Topics(ctx context.Context) error {
	topics, err := db.Stats.Topics(ctx.Request().Context(), db.TopOptions{ScopeOptions: scopeOptions(ctx), Limit: limit(ctx)})
	if err != nil {
		logrus.WithContext(ctx.Request().Context()).WithError(err).Error("Failed to list topics")
		return ctx.ApiServerError()
	}
	return ctx.ApiSuccess(dto.ToCountItems(topics))
}

// Languages
// @Summary List the languages
// @Description The primary languages ordered by the number of dependents.
// @Produce json
// @Param limit query int false "Number of items, defaults to 20, at most 100"
// @Param includeForks query bool false "Include the forked repositories, defaults to false"
// @Success 200 {array} dto.CountItem
// @Failure 500 {string} string "Internal server error"
// @ID listLanguages
// @Router /stats/languages [get]
func (statsRoute) Languages(ctx context.Context) error {
	languages, err := db.Stats.Languages(ctx.Request().Context(), db.TopOptions{ScopeOptions: scopeOptions(ctx), Limit: limit(ctx)})
	if err != nil {
		logrus.WithContext(ctx.Request().Context()).WithError(err).Error("Failed to list languages")
		return ctx.ApiServerError()
	}
	return ctx.ApiSuccess(dto.ToCountItems(languages))
}

// Versions
// @Summary List the module versions
// @Description The distribution of the module versions in the go.mod of the dependents, in descending version order.
// @Produce json
// @Param granularity query string false "Group by the exact or the minor versions, defaults to exact" Enums(exact, minor)
// @Param includeForks query bool false "Include the forked repositories, defaults to false"
// @Success 200 {array} dto.CountItem
// @Failure 400 {string} string "Invalid granularity"
// @Failure 500 {string} string "Internal server error"
// @ID listVersions
// @Router /stats/versions [get]
func (statsRoute) Versions(ctx context.Context) error {
	granularity := db.VersionGranularity(ctx.Query("granularity", string(db.VersionGranularityExact)))
	if granularity != db.VersionGranularityExact && granularity != db.VersionGranularityMinor {
		return ctx.ApiError(http.StatusBadRequest, "Invalid granularity")
	}

	versions, err := db.Stats.ModuleVersions(ctx.Request().Context(), db.VersionOptions{ScopeOptions: scopeOptions(ctx), Granularity: granularity})
	if err != nil {
		logrus.WithContext(ctx.Request().Context()).WithError(err).Error("Failed to list module versions")
		return ctx.ApiServerError()
	}
	return ctx.ApiSuccess(dto.ToCountItems(versions))
}

// GoVersions
// @Summary List the Go versions
// @Description The MAJOR.MINOR distribution of the go directive in the go.mod of the dependents, in descending version order.
// @Produce json
// @Param includeForks query bool false "Include the forked repositories, defaults to false"
// @Success 200 {array} dto.CountItem
// @Failure 500 {string} string "Internal server error"
// @ID listGoVersions
// @Router /stats/go-versions [get]
func (statsRoute) GoVersions(ctx context.Context) error {
	versions, err := db.Stats.GoVersions(ctx.Request().Context(), scopeOptions(ctx))
	if err != nil {
		logrus.WithContext(ctx.Request().Context()).WithError(err).Error("Failed to list Go versions")
		return ctx.ApiServerError()
	}
	return ctx.ApiSuccess(dto.ToCountItems(versions))
}

// Modules
// @Summary Rank the related modules
// @Description The other modules of the same organization (e.g. github.com/flamego/session) used by the dependents, ordered by the number of dependents.
// @Produce json
// @Param limit query int false "Number of items, defaults to 20, at most 100"
// @Param includeForks query bool false "Include the forked repositories, defaults to false"
// @Success 200 {array} dto.CountItem
// @Failure 500 {string} string "Internal server error"
// @ID listModules
// @Router /stats/modules [get]
func (statsRoute) Modules(ctx context.Context) error {
	modules, err := db.Stats.RelatedModules(ctx.Request().Context(), db.TopOptions{ScopeOptions: scopeOptions(ctx), Limit: limit(ctx)})
	if err != nil {
		logrus.WithContext(ctx.Request().Context()).WithError(err).Error("Failed to list related modules")
		return ctx.ApiServerError()
	}
	return ctx.ApiSuccess(dto.ToCountItems(modules))
}
