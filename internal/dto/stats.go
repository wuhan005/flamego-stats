package dto

import (
	"time"

	"github.com/wuhan005/flamego-stats/internal/db"
)

// Overview is the summary of the repositories depending on the module.
type Overview struct {
	// Repository and Module are the repository and the Go module to collect the dependents of.
	Repository string `json:"repository"`
	Module     string `json:"module"`

	Repositories int64 `json:"repositories"`
	Owners       int64 `json:"owners"`
	Stars        int64 `json:"stars"`
	Forks        int64 `json:"forks"`
	// Resolved is the number of dependents whose module version is resolved.
	Resolved int64 `json:"resolved"`
	// ReportedTotal is the approximate number shown in "Used by" on GitHub, including the forked and private repositories.
	ReportedTotal int `json:"reportedTotal"`
	// LastSyncedAt is when the latest succeeded synchronization finished, it is null if never synchronized.
	LastSyncedAt *time.Time `json:"lastSyncedAt" extensions:"x-nullable"`
} // @name Overview

// CountItem is the number of the repositories grouped by name.
type CountItem struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
} // @name CountItem

func ToCountItems(stats []*db.CountStat) []*CountItem {
	items := make([]*CountItem, 0, len(stats))
	for _, s := range stats {
		items = append(items, &CountItem{Name: s.Name, Count: s.Count})
	}
	return items
}

// OwnerRank is the owner ranked by the number of the repositories depending on the module.
type OwnerRank struct {
	OwnerBrief
	Repositories int64 `json:"repositories"`
	Stars        int64 `json:"stars"`
} // @name OwnerRank

func ToOwnerRanks(stats []*db.OwnerStat) []*OwnerRank {
	items := make([]*OwnerRank, 0, len(stats))
	for _, s := range stats {
		items = append(items, &OwnerRank{
			OwnerBrief: OwnerBrief{
				Login:     s.Login,
				Type:      s.Type,
				AvatarURL: s.AvatarURL,
				HTMLURL:   ownerHTMLURL(s.Login),
			},
			Repositories: s.Repositories,
			Stars:        s.Stars,
		})
	}
	return items
}
