package db

import (
	"cmp"
	"context"
	"slices"

	"github.com/cockroachdb/errors"
	"golang.org/x/mod/semver"
	"gorm.io/gorm"
)

var _ StatsStore = (*stats)(nil)

// Stats is the default instance of the StatsStore.
var Stats StatsStore

// StatsStore is the aggregation interface over the repositories.
type StatsStore interface {
	// Overview returns the totals of the repositories.
	Overview(ctx context.Context, opts ScopeOptions) (*Overview, error)
	// TopOwners returns the owners ordered by the number of repositories, then by the total stars.
	TopOwners(ctx context.Context, opts TopOptions) ([]*OwnerStat, error)
	// Topics returns the repository topics ordered by the number of repositories.
	Topics(ctx context.Context, opts TopOptions) ([]*CountStat, error)
	// Languages returns the primary languages ordered by the number of repositories, the repositories without a language are ignored.
	Languages(ctx context.Context, opts TopOptions) ([]*CountStat, error)
	// ModuleVersions returns the module version distribution in descending version order, the unresolved repositories are ignored.
	ModuleVersions(ctx context.Context, opts VersionOptions) ([]*CountStat, error)
	// GoVersions returns the MAJOR.MINOR distribution of the go directive in descending version order.
	GoVersions(ctx context.Context, opts ScopeOptions) ([]*CountStat, error)
	// RelatedModules returns the other modules of the same organization ordered by the number of repositories.
	RelatedModules(ctx context.Context, opts TopOptions) ([]*CountStat, error)
}

func NewStatsStore(db *gorm.DB) StatsStore {
	return &stats{db}
}

type stats struct {
	*gorm.DB
}

// Overview is the totals of the repositories.
type Overview struct {
	Repositories int64
	Owners       int64
	Stars        int64
	Forks        int64
	// Resolved is the number of repositories whose module version is resolved.
	Resolved int64
}

// CountStat is the number of the repositories grouped by Name.
type CountStat struct {
	Name  string
	Count int64
}

// OwnerStat is the repositories of an owner.
type OwnerStat struct {
	OwnerID   int64
	Login     string
	Type      string
	AvatarURL string
	// Repositories is the number of repositories of the owner depending on the module.
	Repositories int64
	Stars        int64
}

// TopOptions are the options of the ranked statistics.
type TopOptions struct {
	ScopeOptions
	// Limit is the maximum number of items, it is unlimited if not positive.
	Limit int
}

func (opts TopOptions) apply(q *gorm.DB) *gorm.DB {
	q = opts.ScopeOptions.apply(q)
	if opts.Limit > 0 {
		q = q.Limit(opts.Limit)
	}
	return q
}

// VersionGranularity is how the versions are grouped.
type VersionGranularity string

const (
	VersionGranularityExact VersionGranularity = "exact"
	VersionGranularityMinor VersionGranularity = "minor"
)

// VersionOptions are the options of the module version distribution.
type VersionOptions struct {
	ScopeOptions
	// Granularity is the same as exact if empty.
	Granularity VersionGranularity
}

func (db *stats) Overview(ctx context.Context, opts ScopeOptions) (*Overview, error) {
	var overview Overview
	err := opts.apply(db.WithContext(ctx).Model(&Repository{})).
		Select(`COUNT(*) AS repositories,
			COUNT(DISTINCT owner_id) AS owners,
			COALESCE(SUM(stars), 0) AS stars,
			COALESCE(SUM(forks), 0) AS forks,
			COUNT(*) FILTER (WHERE module_version <> '') AS resolved`).
		Scan(&overview).Error
	if err != nil {
		return nil, errors.Wrap(err, "scan")
	}
	return &overview, nil
}

func (db *stats) TopOwners(ctx context.Context, opts TopOptions) ([]*OwnerStat, error) {
	var owners []*OwnerStat
	// The rows of an owner are refreshed together in every run, so MAX picks the consistent latest value.
	return owners, opts.apply(db.WithContext(ctx).Model(&Repository{})).
		Select(`owner_id,
			MAX(owner_login) AS login,
			MAX(owner_type) AS type,
			MAX(owner_avatar_url) AS avatar_url,
			COUNT(*) AS repositories,
			COALESCE(SUM(stars), 0) AS stars`).
		Group("owner_id").
		Order("repositories DESC, stars DESC, login ASC").
		Scan(&owners).Error
}

func (db *stats) Topics(ctx context.Context, opts TopOptions) ([]*CountStat, error) {
	var topics []*CountStat
	return topics, opts.apply(db.WithContext(ctx).Model(&Repository{})).
		Joins("CROSS JOIN LATERAL unnest(repositories.topics) AS topic").
		Select("topic AS name, COUNT(*) AS count").
		Group("topic").
		Order("count DESC, name ASC").
		Scan(&topics).Error
}

func (db *stats) Languages(ctx context.Context, opts TopOptions) ([]*CountStat, error) {
	var languages []*CountStat
	return languages, opts.apply(db.WithContext(ctx).Model(&Repository{})).
		Where("language <> ''").
		Select("language AS name, COUNT(*) AS count").
		Group("language").
		Order("count DESC, name ASC").
		Scan(&languages).Error
}

func (db *stats) ModuleVersions(ctx context.Context, opts VersionOptions) ([]*CountStat, error) {
	column := "module_version"
	if opts.Granularity == VersionGranularityMinor {
		column = "module_minor_version"
	}
	var versions []*CountStat
	if err := opts.apply(db.WithContext(ctx).Model(&Repository{})).
		Where(column + " <> ''").
		Select(column + " AS name, COUNT(*) AS count").
		Group(column).
		Scan(&versions).Error; err != nil {
		return nil, err
	}
	sortVersionsDesc(versions)
	return versions, nil
}

func (db *stats) GoVersions(ctx context.Context, opts ScopeOptions) ([]*CountStat, error) {
	// The go directive has the patch version (1.22.0) since Go 1.21, group it with the legacy format (1.22) by MAJOR.MINOR.
	const minor = `SUBSTRING(go_version FROM '^[0-9]+\.[0-9]+')`
	var versions []*CountStat
	if err := opts.apply(db.WithContext(ctx).Model(&Repository{})).
		Where(minor + " IS NOT NULL").
		Select(minor + " AS name, COUNT(*) AS count").
		Group(minor).
		Scan(&versions).Error; err != nil {
		return nil, err
	}
	// The go directive has no "v" prefix, which semver requires.
	slices.SortFunc(versions, func(a, b *CountStat) int {
		return semver.Compare("v"+b.Name, "v"+a.Name)
	})
	return versions, nil
}

func (db *stats) RelatedModules(ctx context.Context, opts TopOptions) ([]*CountStat, error) {
	var modules []*CountStat
	return modules, opts.apply(db.WithContext(ctx).Model(&Repository{})).
		Joins("CROSS JOIN LATERAL unnest(repositories.related_modules) AS module").
		Select("module AS name, COUNT(*) AS count").
		Group("module").
		Order("count DESC, name ASC").
		Scan(&modules).Error
}

// sortVersionsDesc sorts the versions in descending semver order, the invalid versions are the last.
func sortVersionsDesc(versions []*CountStat) {
	slices.SortFunc(versions, func(a, b *CountStat) int {
		if c := semver.Compare(b.Name, a.Name); c != 0 {
			return c
		}
		return cmp.Compare(b.Count, a.Count)
	})
}
