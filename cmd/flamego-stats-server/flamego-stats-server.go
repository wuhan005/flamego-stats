package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"

	"github.com/wuhan005/flamego-stats/internal/appconst"
	"github.com/wuhan005/flamego-stats/internal/conf"
	"github.com/wuhan005/flamego-stats/internal/db"
	"github.com/wuhan005/flamego-stats/internal/route"
)

func main() {
	configFilePath := flag.String("config", "./config/flamego-stats.yaml", "path to the configuration file")
	flag.Parse()
	signals, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	if err := conf.Init(*configFilePath); err != nil {
		logrus.WithError(err).Fatal("Failed to initialize configuration")
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

	address := fmt.Sprintf("0.0.0.0:%d", conf.App.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logrus.WithError(err).Fatalf("Failed to listen on %s", address)
	}
	logrus.WithField("commit", appconst.BuildCommit).Infof("Listening on %s", address)

	server := &http.Server{
		Handler:           route.New(route.Options{DB: gormDB}),
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := serve(signals, listener, server, conf.App.ShutdownTimeout); err != nil {
		logrus.WithError(err).Error("Server stopped with incomplete graceful shutdown")
	}
	logrus.Info("Shutting down")
}

// serve runs the server until signaled, and then waits up to timeout for the in-flight requests.
func serve(signals context.Context, listener net.Listener, server *http.Server, timeout time.Duration) error {
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()

	var serveErr error
	select {
	case <-signals.Done():
	case serveErr = <-served:
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	shutdownErr := server.Shutdown(ctx)
	if shutdownErr != nil {
		_ = server.Close()
	}

	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		return serveErr
	}
	return shutdownErr
}
