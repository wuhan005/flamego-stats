package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"

	"github.com/wuhan005/flamego-stats/internal/appconst"
	"github.com/wuhan005/flamego-stats/internal/conf"
	"github.com/wuhan005/flamego-stats/internal/db"
	"github.com/wuhan005/flamego-stats/internal/github"
	"github.com/wuhan005/flamego-stats/internal/syncer"
)

func main() {
	configFilePath := flag.String("config", "./config/flamego-stats.yaml", "path to the configuration file")
	once := flag.Bool("once", false, "run the synchronization once and exit, e.g. for cron jobs")
	only := flag.String("job", "", "run only the job, dependents or activities, all of them if empty")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := conf.Init(*configFilePath); err != nil {
		logrus.WithError(err).Fatal("Failed to initialize configuration")
	}
	if conf.GitHub.Token == "" {
		logrus.Warn("github.token is not configured, the GitHub API is limited to 60 requests per hour")
	}

	gormDB, err := db.Init()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize database")
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to get database connection")
	}
	defer func() { _ = sqlDB.Close() }()

	client, err := github.NewClient(conf.GitHub.Token)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create GitHub client")
	}
	s := syncer.New(gormDB, client)
	logrus.WithFields(logrus.Fields{
		"commit":       appconst.BuildCommit,
		"repository":   conf.Sync.Repository,
		"module":       conf.Sync.Module,
		"organization": conf.Activity.Organization,
	}).Info("Worker started")

	jobs := []*job{
		{name: "dependents", interval: conf.Sync.Interval, run: s.Run, untilNext: untilNextRun},
		{name: "activities", interval: conf.Activity.Interval, run: s.SyncActivities, untilNext: untilNextActivitySync},
	}
	if *only != "" {
		jobs = slices.DeleteFunc(jobs, func(j *job) bool { return j.name != *only })
		if len(jobs) == 0 {
			logrus.Fatalf("Unknown job %q", *only)
		}
	}
	if *once {
		for _, j := range jobs {
			if err := j.run(ctx); err != nil {
				logrus.WithError(err).WithField("job", j.name).Fatal("Failed to synchronize")
			}
		}
		return
	}

	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Go(func() { j.schedule(ctx) })
	}
	wg.Wait()
	logrus.Info("Shutting down")
}

// job is a synchronization run periodically.
type job struct {
	name     string
	interval time.Duration
	run      func(ctx context.Context) error
	// untilNext returns the wait until the next run based on the last run saved, so restarting the worker does not trigger a run immediately.
	untilNext func(ctx context.Context) (time.Duration, error)
}

// schedule runs the job at the interval until ctx is canceled.
func (j *job) schedule(ctx context.Context) {
	logger := logrus.WithField("job", j.name)
	for {
		wait, err := j.untilNext(ctx)
		if err != nil {
			logger.WithError(err).Error("Failed to schedule the next synchronization")
			wait = time.Minute
		}
		if wait > 0 {
			logger.WithField("at", time.Now().Add(wait).Format(time.RFC3339)).Info("Waiting for the next synchronization")
		}
		if err := sleep(ctx, wait); err != nil {
			return
		}

		if err := j.run(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			if errors.Is(err, syncer.ErrLocked) {
				logger.Info("Another worker is synchronizing, skipped")
			} else {
				logger.WithError(err).Error("Failed to synchronize")
			}
			// Retry in at most 30 minutes, not to drain the GitHub quota.
			if err := sleep(ctx, min(j.interval, 30*time.Minute)); err != nil {
				return
			}
		}
	}
}

// untilNextActivitySync returns the wait until the next run based on the latest cursor of the repositories.
func untilNextActivitySync(ctx context.Context) (time.Duration, error) {
	syncedAt, err := db.ActivityRepositories.LatestSyncedAt(ctx)
	if err != nil || syncedAt == nil {
		return 0, err
	}
	return max(time.Until(syncedAt.Add(conf.Activity.Interval)), 0), nil
}

// untilNextRun returns the wait until the next run based on the start time of the latest succeeded run.
func untilNextRun(ctx context.Context) (time.Duration, error) {
	run, err := db.SyncRuns.GetLatestSucceeded(ctx)
	if err != nil {
		if errors.Is(err, db.ErrSyncRunNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return max(time.Until(run.StartedAt.Add(conf.Sync.Interval)), 0), nil
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
