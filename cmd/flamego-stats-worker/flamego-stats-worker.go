package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
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
		"commit":     appconst.BuildCommit,
		"repository": conf.Sync.Repository,
		"module":     conf.Sync.Module,
	}).Info("Worker started")

	if *once {
		if err := s.Run(ctx); err != nil {
			logrus.WithError(err).Fatal("Failed to synchronize")
		}
		return
	}

	for {
		wait, err := untilNextRun(ctx)
		if err != nil {
			logrus.WithError(err).Error("Failed to schedule the next synchronization")
			wait = time.Minute
		}
		if wait > 0 {
			logrus.WithField("at", time.Now().Add(wait).Format(time.RFC3339)).Info("Waiting for the next synchronization")
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			logrus.Info("Shutting down")
			return
		case <-timer.C:
		}

		if err := s.Run(ctx); err != nil {
			if ctx.Err() != nil {
				logrus.Info("Shutting down")
				return
			}
			if errors.Is(err, syncer.ErrLocked) {
				logrus.Info("Another worker is synchronizing, skipped")
			} else {
				logrus.WithError(err).Error("Failed to synchronize")
			}
			// Retry in at most 30 minutes, not to drain the GitHub quota.
			if err := sleep(ctx, min(conf.Sync.Interval, 30*time.Minute)); err != nil {
				return
			}
		}
	}
}

// untilNextRun returns the wait until the next run based on the start time of the latest succeeded run, so restarting the worker does not trigger a run immediately.
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
