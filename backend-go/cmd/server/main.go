package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/lingmirror/backend-go/internal/config"
	"github.com/lingmirror/backend-go/internal/database"
	"github.com/lingmirror/backend-go/internal/httpx"
	"github.com/lingmirror/backend-go/internal/schemadrift"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := config.NewLogger(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	sugar := logger.Sugar()
	sugar.Info("starting LingMirror backend server")

	// Initialize Sentry (only if DSN is configured)
	if cfg.Sentry.DSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.Sentry.DSN,
			Environment:      cfg.Server.Mode,
			AttachStacktrace: true,
		}); err != nil {
			sugar.Errorf("failed to init Sentry: %v", err)
		} else {
			sugar.Info("Sentry initialized")
		}
	}
	defer sentry.Flush(2 * time.Second)

	// Connect to database
	db, err := database.Connect(cfg, logger)
	if err != nil {
		sugar.Fatalf("failed to connect database: %v", err)
	}

	// Schema drift detection at startup
	driftDetector := schemadrift.New(db, logger, schemadrift.Config{
		Enabled: cfg.SchemaDrift.Enabled,
		OnDrift: cfg.SchemaDrift.OnDrift,
	})
	driftDetector.Check()

	// Setup router
	router := httpx.NewRouter(db, cfg, logger)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	sugar.Infof("server listening on %s", addr)

	srv := &http.Server{
		Addr:    addr,
		Handler: router.Handler(),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			sugar.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	sugar.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		sugar.Fatalf("server forced to shutdown: %v", err)
	}

	sugar.Info("server exited gracefully")
}
