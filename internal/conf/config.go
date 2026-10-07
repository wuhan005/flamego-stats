package conf

import (
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/spf13/viper"
)

var App struct {
	Port            int           `mapstructure:"port"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

var Postgres struct {
	DSN string `mapstructure:"dsn"`
}

var GitHub struct {
	// Token is the GitHub personal access token, no scope is needed for the public repositories.
	// The API is limited to 60 requests per hour without it.
	Token string `mapstructure:"token"`
}

// SyncSource is where the dependents are discovered from.
type SyncSource string

const (
	// SyncSourceDependents scrapes the "Used by" (network/dependents) page of the repository.
	SyncSourceDependents SyncSource = "dependents"
	// SyncSourceCodeSearch searches the go.mod files referencing the module by the Code Search API, which requires the token.
	SyncSourceCodeSearch SyncSource = "code_search"
)

type DependentsConfig struct {
	// PackageID is the package filter of the dependents page, the default package of the repository is used if empty.
	PackageID string `mapstructure:"package_id"`
	// PageDelay is the delay between the pages to avoid the rate limit of the GitHub website, it defaults to 1s.
	PageDelay time.Duration `mapstructure:"page_delay"`
}

var Sync struct {
	// Repository is the repository to collect the dependents of, in the format of owner/name.
	Repository string `mapstructure:"repository"`
	// Module is the Go module path to resolve the versions from the go.mod of the dependents.
	Module string `mapstructure:"module"`
	// Interval is the interval between two runs, it defaults to 6h.
	Interval time.Duration `mapstructure:"interval"`
	// Concurrency is the number of repositories fetched concurrently, it defaults to 4.
	Concurrency int `mapstructure:"concurrency"`
	// Sources are where the dependents are discovered from, it defaults to dependents only.
	Sources    []SyncSource     `mapstructure:"sources"`
	Dependents DependentsConfig `mapstructure:"dependents"`
	// IgnoreOwners are the users or organizations whose repositories are excluded, case-insensitively.
	IgnoreOwners []string `mapstructure:"ignore_owners"`
}

func Init(configFilePath string) error {
	v := viper.New()
	v.SetConfigFile(configFilePath)
	if err := v.ReadInConfig(); err != nil {
		return errors.Wrap(err, "read config file")
	}

	if err := v.UnmarshalKey("app", &App); err != nil {
		return errors.Wrap(err, "parse app")
	}
	if App.Port <= 0 {
		App.Port = 8080
	}
	if App.ShutdownTimeout <= 0 {
		App.ShutdownTimeout = 10 * time.Second
	}

	if err := v.UnmarshalKey("postgres", &Postgres); err != nil {
		return errors.Wrap(err, "parse postgres")
	}
	if err := v.UnmarshalKey("github", &GitHub); err != nil {
		return errors.Wrap(err, "parse github")
	}
	if err := v.UnmarshalKey("sync", &Sync); err != nil {
		return errors.Wrap(err, "parse sync")
	}
	return normalizeSync()
}

func normalizeSync() error {
	if owner, name, ok := strings.Cut(Sync.Repository, "/"); !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return errors.Newf("sync.repository %q must be in the format of owner/name", Sync.Repository)
	}
	if Sync.Module == "" {
		Sync.Module = "github.com/" + Sync.Repository
	}
	if Sync.Interval <= 0 {
		Sync.Interval = 6 * time.Hour
	}
	if Sync.Interval < time.Minute {
		return errors.New("sync.interval must be at least 1m")
	}
	if Sync.Concurrency <= 0 {
		Sync.Concurrency = 4
	}
	if len(Sync.Sources) == 0 {
		Sync.Sources = []SyncSource{SyncSourceDependents}
	}
	for _, source := range Sync.Sources {
		switch source {
		case SyncSourceDependents:
		case SyncSourceCodeSearch:
			if GitHub.Token == "" {
				return errors.New("sync.sources code_search requires github.token")
			}
		default:
			return errors.Newf("unknown sync source %q", source)
		}
	}
	if Sync.Dependents.PageDelay <= 0 {
		Sync.Dependents.PageDelay = time.Second
	}
	return nil
}
