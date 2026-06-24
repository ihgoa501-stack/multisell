package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lingmirror/backend-go/internal/config"
	"github.com/lingmirror/backend-go/internal/database"
	"github.com/lingmirror/backend-go/internal/httpx"
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

	// Connect to database
	db, err := database.Connect(cfg, logger)
	if err != nil {
		sugar.Fatalf("failed to connect database: %v", err)
	}

	// Setup router
	router := httpx.NewRouter(db, cfg, logger)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	sugar.Infof("server listening on %s", addr)

	go func() {
		if err := router.Run(addr); err != nil {
			sugar.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	sugar.Info("shutting down server...")
}
