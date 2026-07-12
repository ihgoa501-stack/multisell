package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/lingmirror/image-service/internal/api"
	"github.com/lingmirror/image-service/internal/blobstore"
	"github.com/lingmirror/image-service/internal/core"
	"github.com/lingmirror/image-service/internal/jobs"
	pgstore "github.com/lingmirror/image-service/internal/postgres"
	"github.com/lingmirror/image-service/internal/providers"
)

func main() {
	data := env("IMAGE_SERVICE_DATA_DIR", "./data")
	environment := strings.ToLower(env("IMAGE_SERVICE_ENVIRONMENT", "development"))
	storage := strings.ToLower(env("IMAGE_SERVICE_JOB_STORE", "file"))
	store, err := openRepository(environment, storage, data, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	blobs, err := blobstore.New(filepath.Join(data, "blobs"))
	if err != nil {
		log.Fatal(err)
	}
	secret := os.Getenv("IMAGE_SERVICE_SHARED_SECRET")
	if secret == "" {
		log.Fatal("IMAGE_SERVICE_SHARED_SECRET is required")
	}
	executionKey := os.Getenv("IMAGE_SERVICE_EXECUTION_TOKEN_SECRET")
	if len(executionKey) < 32 || executionKey == secret {
		log.Fatal("IMAGE_SERVICE_EXECUTION_TOKEN_SECRET must be at least 32 bytes and distinct from IMAGE_SERVICE_SHARED_SECRET")
	}
	registry := providers.NewRegistry()
	if err := registry.Register("DETERMINISTIC_RESIZE", jobs.NewDeterministicExecutor(blobs)); err != nil {
		log.Fatal(err)
	}
	worker, err := jobs.NewWorker(store, registry, "image-worker-1", 30*time.Second, 100*time.Millisecond)
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		if err := worker.Run(ctx); err != nil && err != context.Canceled {
			log.Printf("worker stopped: %v", err)
		}
	}()
	addr := env("IMAGE_SERVICE_ADDR", "127.0.0.1:8092")
	log.Printf("image service listening on %s", addr)
	httpServer := &http.Server{Addr: addr, Handler: api.New(secret, store, blobs, executionKey).Handler(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func openRepository(environment, storage, data, databaseURL string) (core.Repository, error) {
	switch storage {
	case "postgres":
		openCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return pgstore.Open(openCtx, databaseURL)
	case "file":
		if environment == "production" || environment == "acceptance" {
			return nil, fmt.Errorf("IMAGE_SERVICE_JOB_STORE=file is forbidden outside development/test; configure postgres and DATABASE_URL")
		}
		return core.OpenStore(filepath.Join(data, "jobs.json"))
	default:
		return nil, fmt.Errorf("unsupported IMAGE_SERVICE_JOB_STORE %q", storage)
	}
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
