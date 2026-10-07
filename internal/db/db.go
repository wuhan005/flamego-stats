package db

import (
	"time"

	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/wuhan005/flamego-stats/internal/conf"
	"github.com/wuhan005/flamego-stats/internal/dbutil"
)

// Init initializes the database.
func Init() (*gorm.DB, error) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  conf.Postgres.DSN,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		SkipDefaultTransaction: true,
		NowFunc: func() time.Time {
			return dbutil.Now()
		},
		Logger: logger.New(
			logrus.New(),
			logger.Config{
				SlowThreshold:             3 * time.Second,
				LogLevel:                  logger.Silent,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
	})
	if err != nil {
		return nil, errors.Wrap(err, "open connection")
	}

	tables := []interface{}{
		&Repository{}, &SyncRun{},
	}

	// Serialize startup DDL across the server and the worker. A transaction releases the advisory
	// lock even when migration fails or the process loses its connection.
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", AdvisoryLockMigrate).Error; err != nil {
			return err
		}
		return tx.AutoMigrate(tables...)
	}); err != nil {
		return nil, errors.Wrap(err, "auto migrate")
	}

	SetDatabaseStore(db)
	return db, nil
}

const (
	// AdvisoryLockMigrate serializes the startup migration.
	AdvisoryLockMigrate int64 = 0x666c616d65676f
	// AdvisoryLockSync ensures only one worker synchronizes at a time.
	AdvisoryLockSync int64 = 0x666c616d65676f + 1
)

// SetDatabaseStore sets the database table store.
func SetDatabaseStore(db *gorm.DB) {
	Repositories = NewRepositoriesStore(db)
	SyncRuns = NewSyncRunsStore(db)
	Stats = NewStatsStore(db)
}
