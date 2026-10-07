package api

import (
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/wuhan005/flamego-stats/internal/context"
	"github.com/wuhan005/flamego-stats/internal/db"
	"github.com/wuhan005/flamego-stats/internal/dbutil"
	"github.com/wuhan005/flamego-stats/internal/dto"
)

var Repository repositoryRoute

type repositoryRoute struct{}

// List
// @Summary List the dependents
// @Description List the repositories depending on the module, sort=stars ranks by stars and sort=pushed lists the recently updated.
// @Produce json
// @Param page query int false "Page number, starting from 1"
// @Param pageSize query int false "Page size, defaults to 20, at most 100"
// @Param sort query string false "Order of the list, defaults to stars" Enums(stars, forks, pushed, created, name)
// @Param keyword query string false "Match the full name and the description"
// @Param topic query string false "Filter by the topic"
// @Param language query string false "Filter by the primary language"
// @Param owner query string false "Filter by the owner, case-insensitively"
// @Param version query string false "Filter by the module version, e.g. v1.9.4"
// @Param includeForks query bool false "Include the forked repositories, defaults to false"
// @Success 200 {object} dto.ListRepositoriesResp
// @Failure 400 {string} string "Invalid sort"
// @Failure 500 {string} string "Internal server error"
// @ID listRepositories
// @Router /repositories [get]
func (repositoryRoute) List(ctx context.Context) error {
	sort := db.RepositorySort(ctx.Query("sort"))
	if !sort.IsValid() {
		return ctx.ApiError(http.StatusBadRequest, "Invalid sort")
	}

	repositories, total, err := db.Repositories.List(ctx.Request().Context(), db.ListRepositoriesOptions{
		Pagination: dbutil.Pagination{
			Page:     ctx.QueryInt("page"),
			PageSize: ctx.QueryInt("pageSize"),
		},
		ScopeOptions:  scopeOptions(ctx),
		Sort:          sort,
		Keyword:       ctx.QueryTrim("keyword"),
		Topic:         ctx.QueryTrim("topic"),
		Language:      ctx.QueryTrim("language"),
		Owner:         ctx.QueryTrim("owner"),
		ModuleVersion: ctx.QueryTrim("version"),
	})
	if err != nil {
		logrus.WithContext(ctx.Request().Context()).WithError(err).Error("Failed to list repositories")
		return ctx.ApiServerError()
	}

	items := make([]*dto.Repository, 0, len(repositories))
	for _, repository := range repositories {
		items = append(items, dto.ToRepository(repository))
	}
	return ctx.ApiSuccess(dto.ListRepositoriesResp{
		Repositories: items,
		Total:        total,
	})
}
