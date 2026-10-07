package route

import (
	"net/http"

	"github.com/MEDIGO/go-healthz"
	"github.com/flamego/flamego"
	"gorm.io/gorm"

	"github.com/wuhan005/flamego-stats/frontend"
	appcontext "github.com/wuhan005/flamego-stats/internal/context"
	"github.com/wuhan005/flamego-stats/internal/route/api"
)

// Options contains the dependencies used by the application router.
type Options struct {
	// DB is the database connection.
	DB *gorm.DB
}

// New creates the router of the application.
// @Title Flamego Stats API
// @Version 1.0
// @BasePath /_
func New(opts Options) *flamego.Flame {
	f := flamego.Classic()

	f.Use(appcontext.Contexter())

	f.Group("/_", func() {
		f.Get("/overview", api.Stats.Overview)
		f.Get("/repositories", api.Repository.List)
		f.Group("/stats", func() {
			f.Get("/owners", api.Stats.Owners)
			f.Get("/topics", api.Stats.Topics)
			f.Get("/languages", api.Stats.Languages)
			f.Get("/versions", api.Stats.Versions)
			f.Get("/go-versions", api.Stats.GoVersions)
			f.Get("/modules", api.Stats.Modules)
		})
		f.Get("/sync-runs", api.SyncRun.List)
	})

	f.Group("/-", func() {
		f.Get("/healthz", healthz.Handler().ServeHTTP)
		f.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
			sqlDB, err := opts.DB.DB()
			if err == nil {
				err = sqlDB.PingContext(r.Context())
			}
			if err != nil {
				http.Error(w, "Not ready", http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		})
	})

	f.NotFound(frontend.Handler().ServeHTTP)
	return f
}
