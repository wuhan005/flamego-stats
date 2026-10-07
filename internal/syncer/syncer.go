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
	fullNames, reportedTotal, err := s.discover(ctx, logger)
	if err != nil {
		return result, errors.Wrap(err, "discover")
	}
	result.ReportedTotal = reportedTotal
	result.Discovered = len(fullNames)

	var (
		mu      sync.Mutex
		seenIDs = make(map[int64]struct{})
	)
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(conf.Sync.Concurrency)
	for _, fullName := range fullNames {
		g.Go(func() error {
			outcome := s.syncRepository(gctx, logger.WithField("repository", fullName), fullName, seenAt, func(id int64) bool {
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

	// The failed repositories have been touched, so the stale ones are really gone from the dependents.
	removed, err := db.Repositories.DeleteNotSeenSince(ctx, seenAt)
	if err != nil {
		return result, errors.Wrap(err, "delete stale repositories")
	}
	result.Removed = int(removed)
	return result, nil
}

// discover collects the dependents from the configured sources, and returns the distinct owner/name along with the total shown on the dependents page.
func (s *Syncer) discover(ctx context.Context, logger *logrus.Entry) (fullNames []string, reportedTotal int, _ error) {
	ignored := make(map[string]struct{}, len(conf.Sync.IgnoreOwners))
	for _, owner := range conf.Sync.IgnoreOwners {
		ignored[strings.ToLower(owner)] = struct{}{}
	}
	seen := map[string]struct{}{strings.ToLower(conf.Sync.Repository): {}}
	add := func(names []string) {
		for _, name := range names {
			key := strings.ToLower(name)
			owner, _, _ := strings.Cut(key, "/")
			if _, ok := ignored[owner]; ok {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			fullNames = append(fullNames, name)
		}
	}

	for _, source := range conf.Sync.Sources {
		switch source {
		case conf.SyncSourceDependents:
			dependents, err := s.client.ListDependents(ctx, conf.Sync.Repository, github.DependentsOptions{
				PackageID: conf.Sync.Dependents.PackageID,
				PageDelay: conf.Sync.Dependents.PageDelay,
			})
			if err != nil {
				return nil, 0, errors.Wrap(err, "list dependents")
			}
			reportedTotal = dependents.Total
			logger.WithFields(logrus.Fields{"count": len(dependents.Repositories), "reported": dependents.Total}).Info("Listed dependents")
			add(dependents.Repositories)

		case conf.SyncSourceCodeSearch:
			query := fmt.Sprintf("%q filename:go.mod", conf.Sync.Module)
			names, err := s.client.SearchCodeRepositories(ctx, query)
			if err != nil {
				return nil, 0, errors.Wrap(err, "search code")
			}
			logger.WithField("count", len(names)).Info("Searched code")
			add(names)
		}
	}

	// An empty list likely means the page layout changed or the API misbehaved, continuing would remove all the existing data.
	if len(fullNames) == 0 {
		return nil, 0, errors.New("no repositories discovered")
	}
	return fullNames, reportedTotal, nil
}

type outcome int

const (
	outcomeSynced outcome = iota
	outcomeSkipped
	outcomeFailed
)

// syncRepository fetches and saves a repository, claim deduplicates by GitHub ID since a renamed repository may show up with both names.
func (s *Syncer) syncRepository(ctx context.Context, logger *logrus.Entry, fullName string, seenAt time.Time, claim func(id int64) bool) outcome {
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
		goModPath, info, err := s.findGoMod(ctx, repo)
		if err != nil {
			return fail(err, "Failed to find go.mod")
		}
		if info != nil {
			record.GoModPath = goModPath
			record.GoVersion = info.GoVersion
			record.ModuleVersion = info.Version
			record.ModuleMinorVersion = info.MinorVersion
			record.RelatedModules = info.RelatedModules
		} else {
			logger.Debug("No go.mod requires the module")
		}
	}

	if err := db.Repositories.Upsert(ctx, record); err != nil {
		return fail(err, "Failed to save repository")
	}
	return outcomeSynced
}

// findGoMod reads the root go.mod first and then the ones in the subdirectories if it does not require the module, info is nil if none is found.
func (s *Syncer) findGoMod(ctx context.Context, repo *github.Repository) (goModPath string, info *gomod.Info, _ error) {
	parse := func(filePath string) (*gomod.Info, error) {
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

	if info, err := parse("go.mod"); err != nil || info != nil {
		return "go.mod", info, err
	}

	paths, err := s.client.FindFiles(ctx, repo.FullName, repo.DefaultBranch, "go.mod")
	if err != nil {
		if errors.Is(err, github.ErrNotFound) {
			return "", nil, nil
		}
		return "", nil, errors.Wrap(err, "find go.mod")
	}
	for _, p := range goModCandidates(paths, path.Base(conf.Sync.Module)) {
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
