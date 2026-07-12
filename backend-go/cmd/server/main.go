package main

// @title           LingMirror API
// @version         0.3.0
// @description     Cross-border e-commerce AI AgentOS
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization

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

	// Setup router (returns App with Engine, Bus, Scheduler)
	app := httpx.NewRouter(db, cfg, logger)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	sugar.Infof("server listening on %s", addr)

	srv := &http.Server{
		Addr:              addr,
		Handler:           app.Engine.Handler(),
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		MaxHeaderBytes:    cfg.Server.MaxHeaderBytes,
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

	// Fail readiness first, then stop accepting requests and let in-flight HTTP
	// work finish while its scheduler/event-bus dependencies are still alive.
	app.BeginDrain()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		sugar.Errorf("HTTP drain deadline exceeded: %v", err)
		if closeErr := srv.Close(); closeErr != nil {
			sugar.Errorf("forced HTTP close failed: %v", closeErr)
		}
	}

	app.Scheduler.Shutdown()
	busCtx, busCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	if err := app.Bus.StopWithContext(busCtx); err != nil {
		sugar.Errorf("event bus drain deadline exceeded: %v", err)
	}
	busCancel()
	app.Cancel()

	sugar.Info("server exited gracefully")
}
