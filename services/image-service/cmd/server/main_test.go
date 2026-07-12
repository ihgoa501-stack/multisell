package main

import (
	"strings"
	"testing"
)

func TestProductionAndAcceptanceRejectFileStore(t *testing.T) {
	for _, environment := range []string{"production", "acceptance"} {
		if _, err := openRepository(environment, "file", t.TempDir(), ""); err == nil || !strings.Contains(err.Error(), "forbidden") {
			t.Fatalf("%s must reject file store, got %v", environment, err)
		}
	}
}

func TestDevelopmentFileStoreDoesNotRequireDatabaseURL(t *testing.T) {
	store, err := openRepository("development", "file", t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
}

func TestPostgresStoreRequiresDatabaseURL(t *testing.T) {
	if _, err := openRepository("production", "postgres", t.TempDir(), ""); err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("postgres without DATABASE_URL must fail, got %v", err)
	}
}
