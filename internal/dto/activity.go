package dto

import (
	"strings"
	"time"

	"github.com/wuhan005/flamego-stats/internal/db"
)

// Activity is something that happened in a repository of the organization.
type Activity struct {
	ID         int64                   `json:"id"`
	Type       string                  `json:"type" enums:"commit,pull_request,issue,release"`
	Action     string                  `json:"action" enums:"committed,opened,merged,closed,published"`
	Repository ActivityRepositoryBrief `json:"repository"`
	// Number is the number of the pull request or the issue. For a commit, it is the pull request squash-merged as it, or 0.
	Number int `json:"number"`
	// SHA is the hash of the commit, Tag is the tag name of the release.
	SHA        string    `json:"sha"`
	Tag        string    `json:"tag"`
	Title      string    `json:"title"`
	HTMLURL    string    `json:"htmlURL"`
	Draft      bool      `json:"draft"`
	Prerelease bool      `json:"prerelease"`
	Actor      Actor     `json:"actor"`
	OccurredAt time.Time `json:"occurredAt"`
} // @name Activity

// ActivityRepositoryBrief is the repository of an activity.
type ActivityRepositoryBrief struct {
	Name    string `json:"name"`
	HTMLURL string `json:"htmlURL"`
} // @name ActivityRepositoryBrief

// Actor is the GitHub account behind an activity.
type Actor struct {
	// Login is the name in the commit if the commit author is not linked to any account, AvatarURL and HTMLURL are empty then.
	Login     string `json:"login"`
	AvatarURL string `json:"avatarURL"`
	HTMLURL   string `json:"htmlURL"`
	Bot       bool   `json:"bot"`
} // @name Actor

func ToActivity(organization string, a *db.Activity) *Activity {
	return &Activity{
		ID:     a.ID,
		Type:   string(a.Type),
		Action: string(a.Action),
		Repository: ActivityRepositoryBrief{
			Name:    a.RepositoryName,
			HTMLURL: "https://github.com/" + organization + "/" + a.RepositoryName,
		},
		Number:     a.Number,
		SHA:        a.SHA,
		Tag:        a.Tag,
		Title:      a.Title,
		HTMLURL:    a.HTMLURL,
		Draft:      a.Draft,
		Prerelease: a.Prerelease,
		Actor:      ToActor(a.ActorLogin, a.ActorAvatarURL, a.ActorBot),
		OccurredAt: a.OccurredAt,
	}
}

func ToActivities(organization string, activities []*db.Activity) []*Activity {
	items := make([]*Activity, 0, len(activities))
	for _, a := range activities {
		items = append(items, ToActivity(organization, a))
	}
	return items
}

func ToActor(login, avatarURL string, bot bool) Actor {
	actor := Actor{Login: login, AvatarURL: avatarURL, Bot: bot}
	switch {
	case avatarURL == "":
	case bot && strings.HasSuffix(login, "[bot]"):
		actor.HTMLURL = "https://github.com/apps/" + strings.TrimSuffix(login, "[bot]")
	default:
		actor.HTMLURL = ownerHTMLURL(login)
	}
	return actor
}

// ListActivitiesResp is the response of the activity feed.
type ListActivitiesResp struct {
	Activities []*Activity `json:"activities"`
	// NextCursor lists the following activities as the cursor parameter, it is empty at the end of the feed.
	NextCursor string `json:"nextCursor"`
} // @name ListActivitiesResp

// ActivityCounts is the numbers of the activities of each kind in a period.
type ActivityCounts struct {
	Commits            int64 `json:"commits"`
	OpenedPullRequests int64 `json:"openedPullRequests"`
	MergedPullRequests int64 `json:"mergedPullRequests"`
	OpenedIssues       int64 `json:"openedIssues"`
	ClosedIssues       int64 `json:"closedIssues"`
	Releases           int64 `json:"releases"`
} // @name ActivityCounts

func ToActivityCounts(c *db.ActivityCounts) ActivityCounts {
	return ActivityCounts{
		Commits:            c.Commits,
		OpenedPullRequests: c.OpenedPullRequests,
		MergedPullRequests: c.MergedPullRequests,
		OpenedIssues:       c.OpenedIssues,
		ClosedIssues:       c.ClosedIssues,
		Releases:           c.Releases,
	}
}

// DailyCount is the number of the activities of a day.
type DailyCount struct {
	// Date is in the format of 2006-01-02 in the requested time zone.
	Date  string `json:"date"`
	Count int64  `json:"count"`
} // @name DailyCount

// RepositoryActivity is a tracked repository with the number of its activities in the period.
type RepositoryActivity struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	HTMLURL     string `json:"htmlURL"`
	Archived    bool   `json:"archived"`
	Count       int64  `json:"count"`
} // @name RepositoryActivity

// Contributor is an actor with the number of their activities in the period.
type Contributor struct {
	Actor
	Count int64 `json:"count"`
} // @name Contributor

// ActivitySummary is the statistics of the recent activities of the organization.
type ActivitySummary struct {
	Organization string `json:"organization"`
	// Days is the length of the period of Current, Repositories and Contributors, which ends now.
	Days    int            `json:"days"`
	Current ActivityCounts `json:"current"`
	// Previous is the counts of the period of the same length before Current.
	Previous ActivityCounts `json:"previous"`
	// Daily is every day of the recent weeks, the oldest first and today the last.
	Daily []*DailyCount `json:"daily"`
	// Repositories are all the tracked repositories, the most active first.
	Repositories []*RepositoryActivity `json:"repositories"`
	Contributors []*Contributor        `json:"contributors"`
	// Releases are the latest releases.
	Releases     []*Activity `json:"releases"`
	LastSyncedAt *time.Time  `json:"lastSyncedAt" extensions:"x-nullable"`
} // @name ActivitySummary
