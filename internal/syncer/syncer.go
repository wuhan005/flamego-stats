package syncer

import (
	"cmp"
	"context"
	"fmt"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"github.com/wuhan005/flamego-stats/internal/conf"
	"github.com/wuhan005/flamego-stats/internal/db"
	"github.com/wuhan005/flamego-stats/internal/dbutil"
	"github.com/wuhan005/flamego-stats/internal/github"
	"github.com/wuhan005/flamego-stats/internal/gomod"
)

// maxGoModFiles is the maximum number of go.mod files tried in the subdirectories when the root go.mod does not require the module.
const maxGoModFiles = 10

// ErrLocked means another worker is synchronizing.
var ErrLocked = errors.New("another synchronization is running")

// Syncer synchronizes the GitHub repositories depending on the module to the database.
type Syncer struct {
	db     *gorm.DB
	client *github.Client
}

func New(db *gorm.DB, client *github.Client) *Syncer {
	return &Syncer{db: db, client: client}
}

// Run runs a full synchronization and records it in sync_runs, it returns ErrLocked if another worker is synchronizing.
func (s *Syncer) Run(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return errors.Wrap(err, "get database connection")
	}
	unlock, ok, err := dbutil.TryAdvisoryLock(ctx, sqlDB, db.AdvisoryLockSync)
	if err != nil {
		return errors.Wrap(err, "acquire lock")
	}
	if !ok {
		return ErrLocked
	}
	defer unlock()

	// No other run is in progress while holding the lock, the remaining running runs are from the crashed workers.
	if n, err := db.SyncRuns.AbortRunning(ctx, "interrupted"); err != nil {
		return errors.Wrap(err, "abort interrupted runs")
	} else if n > 0 {
		logrus.WithField("count", n).Warn("Marked interrupted synchronization runs as failed")
	}

	run, err := db.SyncRuns.Start(ctx)
	if err != nil {
		return errors.Wrap(err, "start run")
	}
	logger := logrus.WithContext(ctx).WithField("run", run.ID)
	logger.Info("Synchronization started")

	result, runErr := s.run(ctx, logger, run.StartedAt)
	// Save the result even if ctx is canceled.
	if err := db.SyncRuns.Finish(context.WithoutCancel(ctx), run.ID, result, runErr); err != nil {
		logger.WithError(err).Error("Failed to finish synchronization run")
	}
	if runErr != nil {
		return runErr
	}

	logger.WithFields(logrus.Fields{
		"reported":   result.ReportedTotal,
		"discovered": result.Discovered,
		"synced":     result.Synced,
		"skipped":    result.Skipped,
		"failed":     result.Failed,
		"removed":    result.Removed,
	}).Info("Synchronization finished")
	return nil
}

func (s *Syncer) run(ctx context.Context, logger *logrus.Entry, seenAt time.Time) (db.SyncRunResult, error) {
	var result db.SyncRunResult
	candidates, reportedTotal, warnings, err := s.discover(ctx, logger)
	if err != nil {
		return result, errors.Wrap(err, "discover")
	}
	result.ReportedTotal = reportedTotal
	result.Discovered = len(candidates)
	if len(warnings) > 0 {
		result.Partial = true
		result.Warning = strings.Join(warnings, "; ")
	}

	var (
		mu      sync.Mutex
		seenIDs = make(map[int64]struct{})
	)
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(conf.Sync.Concurrency)
	for _, c := range candidates {
		g.Go(func() error {
			outcome := s.syncRepository(gctx, logger.WithField("repository", c.FullName), c, seenAt, func(id int64) bool {
				mu.Lock()
				defer mu.Unlock()
				if _, ok := seenIDs[id]; ok {
					return false
				}
				seenIDs[id] = struct{}{}
				return true
			})

			mu.Lock()
			defer mu.Unlock()
			switch outcome {
			case outcomeSynced:
				result.Synced++
			case outcomeSkipped:
				result.Skipped++
			case outcomeFailed:
				result.Failed++
			}
			return gctx.Err()
		})
	}
	if err := g.Wait(); err != nil {
		return result, err
	}

	// The repositories of the failed sources are not touched, removing the stale ones would remove them by mistake.
	if result.Partial {
		logger.WithField("warning", result.Warning).Warn("Some sources failed, skipped removing the stale repositories")
		return result, nil
	}

	// The failed repositories have been touched, so the stale ones are really gone from the dependents.
	removed, err := db.Repositories.DeleteNotSeenSince(ctx, seenAt)
	if err != nil {
		return result, errors.Wrap(err, "delete stale repositories")
	}
	result.Removed = int(removed)
	return result, nil
}

// candidate is a repository discovered from the sources.
type candidate struct {
	FullName string
	// Dependent means it is listed on the dependents page, otherwise it is collected only if its go.mod requires the module.
	Dependent bool
	// GoModPaths are the go.mod files matched by the code search, which are tried before scanning the tree.
	GoModPaths []string
}

// discover collects the candidates from the configured sources along with the total shown on the dependents page.
// The failed sources are returned as the warnings unless all of them fail.
func (s *Syncer) discover(ctx context.Context, logger *logrus.Entry) (candidates []*candidate, reportedTotal int, warnings []string, _ error) {
	ignored := make(map[string]struct{}, len(conf.Sync.IgnoreOwners))
	for _, owner := range conf.Sync.IgnoreOwners {
		ignored[strings.ToLower(owner)] = struct{}{}
	}
	self := strings.ToLower(conf.Sync.Repository)
	byName := make(map[string]*candidate)
	add := func(fullName string, dependent bool, goModPaths []string) {
		key := strings.ToLower(fullName)
		owner, _, _ := strings.Cut(key, "/")
		if _, ok := ignored[owner]; ok || key == self {
			return
		}
		c, ok := byName[key]
		if !ok {
			c = &candidate{FullName: fullName}
			byName[key] = c
			candidates = append(candidates, c)
		}
		c.Dependent = c.Dependent || dependent
		for _, p := range goModPaths {
			if !slices.Contains(c.GoModPaths, p) {
				c.GoModPaths = append(c.GoModPaths, p)
			}
		}
	}

	for _, source := range conf.Sync.Sources {
		var err error
		switch source {
		case conf.SyncSourceDependents:
			var dependents *github.Dependents
			dependents, err = s.client.ListDependents(ctx, conf.Sync.Repository, github.DependentsOptions{
				PackageID: conf.Sync.Dependents.PackageID,
				PageDelay: conf.Sync.Dependents.PageDelay,
			})
			if err == nil {
				reportedTotal = dependents.Total
				logger.WithFields(logrus.Fields{"count": len(dependents.Repositories), "reported": dependents.Total}).Info("Listed dependents")
				for _, fullName := range dependents.Repositories {
					add(fullName, true, nil)
				}
			}

		case conf.SyncSourceCodeSearch:
			var repos []*github.CodeSearchRepository
			repos, err = s.client.SearchCodeRepositories(ctx, conf.Sync.CodeSearch.Query)
			if err == nil {
				logger.WithFields(logrus.Fields{"count": len(repos), "query": conf.Sync.CodeSearch.Query}).Info("Searched code")
				for _, repo := range repos {
					add(repo.FullName, false, repo.Paths)
				}
			}
		}

		if err != nil {
			if ctx.Err() != nil {
				return nil, 0, nil, ctx.Err()
			}
			logger.WithError(err).WithField("source", source).Warn("Failed to discover repositories from the source")
			warnings = append(warnings, fmt.Sprintf("%s: %v", source, err))
		}
	}

	// An empty list likely means the page layout changed or the API misbehaved, continuing would remove all the existing data.
	if len(candidates) == 0 {
		if len(warnings) > 0 {
			return nil, 0, nil, errors.Newf("no repositories discovered: %s", strings.Join(warnings, "; "))
		}
		return nil, 0, nil, errors.New("no repositories discovered")
	}
	return candidates, reportedTotal, warnings, nil
}

type outcome int

const (
	outcomeSynced outcome = iota
	outcomeSkipped
	outcomeFailed
)

// syncRepository fetches and saves a repository, claim deduplicates by GitHub ID since a renamed repository may show up with both names.
func (s *Syncer) syncRepository(ctx context.Context, logger *logrus.Entry, c *candidate, seenAt time.Time, claim func(id int64) bool) outcome {
	fullName := c.FullName
	fail := func(err error, message string) outcome {
		if ctx.Err() != nil {
			return outcomeFailed
		}
		logger.WithError(err).Warn(message)
		if err := db.Repositories.Touch(ctx, fullName, seenAt); err != nil {
			logger.WithError(err).Error("Failed to touch repository")
		}
		return outcomeFailed
	}

	repo, err := s.client.GetRepository(ctx, fullName)
	if err != nil {
		if errors.Is(err, github.ErrNotFound) {
			logger.Debug("Repository does not exist")
			return outcomeSkipped
		}
		return fail(err, "Failed to get repository")
	}
	// The token may be able to access the private repositories, which must never be collected.
	if !repo.Public {
		logger.Debug("Repository is not public")
		if err := db.Repositories.DeleteByGitHubID(ctx, repo.ID); err != nil {
			logger.WithError(err).Error("Failed to delete repository that is no longer public")
		}
		return outcomeSkipped
	}
	if !claim(repo.ID) {
		return outcomeSkipped
	}

	existing, err := db.Repositories.GetByGitHubID(ctx, repo.ID)
	if err != nil && !errors.Is(err, db.ErrRepositoryNotFound) {
		return fail(err, "Failed to get existing repository")
	}

	record := &db.Repository{
		GitHubID:        repo.ID,
		OwnerID:         repo.OwnerID,
		OwnerLogin:      repo.OwnerLogin,
		OwnerType:       repo.OwnerType,
		OwnerAvatarURL:  repo.OwnerAvatarURL,
		Name:            repo.Name,
		FullName:        repo.FullName,
		Description:     repo.Description,
		Homepage:        repo.Homepage,
		HTMLURL:         repo.HTMLURL,
		Language:        repo.Language,
		License:         repo.License,
		Topics:          repo.Topics,
		DefaultBranch:   repo.DefaultBranch,
		Fork:            repo.Fork,
		Archived:        repo.Archived,
		Stars:           repo.Stars,
		Forks:           repo.Forks,
		Watchers:        repo.Watchers,
		OpenIssues:      repo.OpenIssues,
		GitHubCreatedAt: repo.CreatedAt,
		GitHubUpdatedAt: repo.UpdatedAt,
		PushedAt:        repo.PushedAt,
		LastSeenAt:      seenAt,
		SyncedAt:        dbutil.Now(),
	}

	// The go.mod does not change without new pushes, reuse the previous result to save the API quota unless it was not found.
	if existing != nil && existing.PushedAt.Equal(repo.PushedAt) && existing.GoModPath != "" {
		record.GoModPath = existing.GoModPath
		record.GoVersion = existing.GoVersion
		record.ModuleVersion = existing.ModuleVersion
		record.ModuleMinorVersion = existing.ModuleMinorVersion
		record.RelatedModules = existing.RelatedModules
	} else {
		goModPath, info, err := s.findGoMod(ctx, repo, c.GoModPaths)
		if err != nil {
			return fail(err, "Failed to find go.mod")
		}
		switch {
		case info != nil:
			record.GoModPath = goModPath
			record.GoVersion = info.GoVersion
			record.ModuleVersion = info.Version
			record.ModuleMinorVersion = info.MinorVersion
			record.RelatedModules = info.RelatedModules
		case !c.Dependent:
			// The code search also matches the go.mod files mentioning the module otherwise, e.g. github.com/foo/flamego-bar.
			logger.Debug("No go.mod requires the module, skipped the code search match")
			return outcomeSkipped
		default:
			logger.Debug("No go.mod requires the module")
		}
	}

	if err := db.Repositories.Upsert(ctx, record); err != nil {
		return fail(err, "Failed to save repository")
	}
	return outcomeSynced
}

// findGoMod tries the root go.mod, the hinted ones and then the ones in the subdirectories until one requires the module, info is nil if none is found.
func (s *Syncer) findGoMod(ctx context.Context, repo *github.Repository, hints []string) (goModPath string, info *gomod.Info, _ error) {
	tried := make(map[string]bool)
	parse := func(filePath string) (*gomod.Info, error) {
		tried[filePath] = true
		data, err := s.client.GetFile(ctx, repo.FullName, repo.DefaultBranch, filePath)
		if err != nil {
			if errors.Is(err, github.ErrNotFound) {
				return nil, nil
			}
			return nil, errors.Wrapf(err, "get %s", filePath)
		}
		info, err := gomod.Parse(filePath, data, conf.Sync.Module)
		if err != nil {
			// Skip the go.mod not requiring the module or failed to parse, and try the next one.
			return nil, nil
		}
		return info, nil
	}

	for _, p := range append([]string{"go.mod"}, hints...) {
		if tried[p] || isVendored(p) {
			continue
		}
		if info, err := parse(p); err != nil || info != nil {
			return p, info, err
		}
	}

	paths, err := s.client.FindFiles(ctx, repo.FullName, repo.DefaultBranch, "go.mod")
	if err != nil {
		if errors.Is(err, github.ErrNotFound) {
			return "", nil, nil
		}
		return "", nil, errors.Wrap(err, "find go.mod")
	}
	for _, p := range goModCandidates(paths, path.Base(conf.Sync.Module)) {
		if tried[p] {
			continue
		}
		if info, err := parse(p); err != nil || info != nil {
			return p, info, err
		}
	}
	return "", nil, nil
}

// goModCandidates excludes the root and vendored go.mod files, puts the ones whose directory contains keyword first (e.g. go/flamego/go.mod of web-frameworks), and returns at most maxGoModFiles of them.
func goModCandidates(paths []string, keyword string) []string {
	keyword = strings.ToLower(keyword)
	candidates := make([]string, 0, len(paths))
	for _, p := range paths {
		if p != "go.mod" && !isVendored(p) {
			candidates = append(candidates, p)
		}
	}
	slices.SortStableFunc(candidates, func(a, b string) int {
		return cmp.Compare(goModRank(a, keyword), goModRank(b, keyword))
	})
	return candidates[:min(len(candidates), maxGoModFiles)]
}

func goModRank(p, keyword string) int {
	if strings.Contains(strings.ToLower(path.Dir(p)), keyword) {
		return 0
	}
	return 1
}

// isVendored reports whether the path is in a directory like vendor or testdata, which does not represent the project's own dependencies.
func isVendored(p string) bool {
	for _, dir := range strings.Split(p, "/") {
		switch dir {
		case "vendor", "testdata", "third_party", "node_modules":
			return true
		}
	}
	return false
}
