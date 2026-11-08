package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"

	"github.com/wuhan005/flamego-stats/internal/conf"
	"github.com/wuhan005/flamego-stats/internal/context"
	"github.com/wuhan005/flamego-stats/internal/db"
	"github.com/wuhan005/flamego-stats/internal/dto"
)

const (
	// summaryDays is the length of the period of the activity summary.
	summaryDays = 30
	// dailyDays is the number of the days of the daily counts, 13 weeks.
	dailyDays = 91

	summaryContributors = 8
	summaryReleases     = 5
)

var Activity activityRoute

type activityRoute struct{}

// List
// @Summary List the activities
// @Description The commits, pull requests, issues and releases of the repositories of the organization, the newest first.
// @Description Without the type, the commits squash-merged from the pull requests are folded into the merged pull requests.
// @Produce json
// @Param type query string false "Type of the activities, all of them if empty" Enums(commit, pull_request, issue, release)
// @Param repository query string false "Filter by the repository name, case-insensitively"
// @Param excludeBots query bool false "Exclude the activities of the GitHub Apps, defaults to false"
// @Param cursor query string false "The nextCursor of the previous page"
// @Param limit query int false "Number of items, defaults to 20, at most 100"
// @Success 200 {object} dto.ListActivitiesResp
// @Failure 400 {string} string "Invalid type or cursor"
// @Failure 500 {string} string "Internal server error"
// @ID listActivities
// @Router /activities [get]
func (activityRoute) List(ctx context.Context) error {
	typ := db.ActivityType(ctx.Query("type"))
	if !typ.IsValid() {
		return ctx.ApiError(http.StatusBadRequest, "Invalid type")
	}
	var before *db.ActivityCursor
	if cursor := ctx.Query("cursor"); cursor != "" {
		c, err := parseActivityCursor(cursor)
		if err != nil {
			return ctx.ApiError(http.StatusBadRequest, "Invalid cursor")
		}
		before = c
	}

	n := limit(ctx)
	// One more item tells whether there is a following page.
	activities, err := db.Activities.List(ctx.Request().Context(), db.ListActivitiesOptions{
		ActivityFilter: db.ActivityFilter{
			Repository:  ctx.QueryTrim("repository"),
			ExcludeBots: ctx.QueryBool("excludeBots"),
		},
		Type:   typ,
		Before: before,
		Limit:  n + 1,
	})
	if err != nil {
		logrus.WithContext(ctx.Request().Context()).WithError(err).Error("Failed to list activities")
		return ctx.ApiServerError()
	}

	resp := dto.ListActivitiesResp{}
	if len(activities) > n {
		activities = activities[:n]
		last := activities[n-1]
		resp.NextCursor = formatActivityCursor(&db.ActivityCursor{OccurredAt: last.OccurredAt, ID: last.ID})
	}
	resp.Activities = dto.ToActivities(conf.Activity.Organization, activities)
	return ctx.ApiSuccess(resp)
}

// Summary
// @Summary Summarize the activities
// @Description The counts of the last 30 days and the 30 days before, the daily counts of the last 13 weeks, the repositories, the top contributors and the latest releases.
// @Produce json
// @Param excludeBots query bool false "Exclude the activities of the GitHub Apps, defaults to false"
// @Param tz query string false "IANA time zone of the daily counts, e.g. Asia/Shanghai, defaults to UTC"
// @Success 200 {object} dto.ActivitySummary
// @Failure 400 {string} string "Invalid tz"
// @Failure 500 {string} string "Internal server error"
// @ID getActivitySummary
// @Router /activities/summary [get]
func (activityRoute) Summary(ctx context.Context) error {
	// Local is the time zone of the server, which PostgreSQL does not know.
	tz := ctx.Query("tz")
	loc, err := time.LoadLocation(tz)
	if err != nil || tz == "Local" {
		return ctx.ApiError(http.StatusBadRequest, "Invalid tz")
	}

	summary, err := summarizeActivities(ctx, db.ActivityFilter{ExcludeBots: ctx.QueryBool("excludeBots")}, loc)
	if err != nil {
		logrus.WithContext(ctx.Request().Context()).WithError(err).Error("Failed to summarize activities")
		return ctx.ApiServerError()
	}
	return ctx.ApiSuccess(summary)
}

func summarizeActivities(ctx context.Context, filter db.ActivityFilter, loc *time.Location) (*dto.ActivitySummary, error) {
	c := ctx.Request().Context()
	now := time.Now()
	since := now.AddDate(0, 0, -summaryDays)

	current, err := db.Activities.Counts(c, filter, since, now)
	if err != nil {
		return nil, errors.Wrap(err, "count current")
	}
	previous, err := db.Activities.Counts(c, filter, since.AddDate(0, 0, -summaryDays), since)
	if err != nil {
		return nil, errors.Wrap(err, "count previous")
	}

	today := now.In(loc)
	firstDay := time.Date(today.Year(), today.Month(), today.Day()-(dailyDays-1), 0, 0, 0, 0, loc)
	days, err := db.Activities.Daily(c, filter, firstDay, loc)
	if err != nil {
		return nil, errors.Wrap(err, "count daily")
	}

	repositories, err := db.Activities.RepositoryCounts(c, filter, since)
	if err != nil {
		return nil, errors.Wrap(err, "count repositories")
	}
	actors, err := db.Activities.TopActors(c, filter, since, summaryContributors)
	if err != nil {
		return nil, errors.Wrap(err, "rank actors")
	}
	releases, err := db.Activities.List(c, db.ListActivitiesOptions{ActivityFilter: filter, Type: db.ActivityTypeRelease, Limit: summaryReleases})
	if err != nil {
		return nil, errors.Wrap(err, "list releases")
	}
	lastSyncedAt, err := db.ActivityRepositories.LatestSyncedAt(c)
	if err != nil {
		return nil, errors.Wrap(err, "get last synced time")
	}

	summary := &dto.ActivitySummary{
		Organization: conf.Activity.Organization,
		Days:         summaryDays,
		Current:      dto.ToActivityCounts(current),
		Previous:     dto.ToActivityCounts(previous),
		Daily:        fillDays(days, firstDay, dailyDays),
		Repositories: make([]*dto.RepositoryActivity, 0, len(repositories)),
		Contributors: make([]*dto.Contributor, 0, len(actors)),
		Releases:     dto.ToActivities(conf.Activity.Organization, releases),
		LastSyncedAt: lastSyncedAt,
	}
	for _, r := range repositories {
		summary.Repositories = append(summary.Repositories, &dto.RepositoryActivity{
			Name:        r.Name,
			Description: r.Description,
			HTMLURL:     r.HTMLURL,
			Archived:    r.Archived,
			Count:       r.Count,
		})
	}
	for _, a := range actors {
		summary.Contributors = append(summary.Contributors, &dto.Contributor{
			Actor: dto.ToActor(a.Login, a.AvatarURL, a.Bot),
			Count: a.Count,
		})
	}
	return summary, nil
}

// fillDays returns every day from firstDay with the counts of days, the missing days count 0.
func fillDays(days []*db.DailyCount, firstDay time.Time, n int) []*dto.DailyCount {
	counts := make(map[string]int64, len(days))
	for _, d := range days {
		counts[d.Day] = d.Count
	}
	items := make([]*dto.DailyCount, 0, n)
	for i := range n {
		date := firstDay.AddDate(0, 0, i).Format(time.DateOnly)
		items = append(items, &dto.DailyCount{Date: date, Count: counts[date]})
	}
	return items
}

// formatActivityCursor encodes the cursor as <occurred at in Unix microseconds>-<ID>.
func formatActivityCursor(c *db.ActivityCursor) string {
	return fmt.Sprintf("%d-%d", c.OccurredAt.UnixMicro(), c.ID)
}

func parseActivityCursor(s string) (*db.ActivityCursor, error) {
	at, id, ok := strings.Cut(s, "-")
	if !ok {
		return nil, errors.New("missing separator")
	}
	micro, err := strconv.ParseInt(at, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "parse time")
	}
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "parse ID")
	}
	return &db.ActivityCursor{OccurredAt: time.UnixMicro(micro), ID: n}, nil
}
