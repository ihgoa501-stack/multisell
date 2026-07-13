package main

import (
	"strings"
	"testing"
)

func TestPhotoroomSandboxCannotBeEnabledInProduction(t *testing.T) {
	if err := validatePhotoroomEnvironment("production", true); err == nil {
		t.Fatal("production accepted Photoroom sandbox")
	}
	if err := validatePhotoroomEnvironment("acceptance", true); err != nil {
		t.Fatalf("acceptance fixture rejected: %v", err)
	}
}

func TestDeploymentEnvironmentIsClosedEnum(t *testing.T) {
	for _, value := range []string{"development", "acceptance", "production"} {
		if err := validateDeploymentEnvironment(value); err != nil {
			t.Fatalf("%s rejected: %v", value, err)
		}
	}
	for _, value := range []string{"", "test", "staging", "prod", "unknown"} {
		if err := validateDeploymentEnvironment(value); err == nil {
			t.Fatalf("%q accepted", value)
		}
	}
}

func TestAcceptanceAndProductionRequireIndependentLongSecrets(t *testing.T) {
	longA := strings.Repeat("a", 32)
	longB := strings.Repeat("b", 32)
	for _, environment := range []string{"acceptance", "production"} {
		if err := validateServiceSecrets(environment, longA, longB); err != nil {
			t.Fatalf("valid %s secrets rejected: %v", environment, err)
		}
		for _, pair := range [][2]string{{"short", longB}, {longA, longA}} {
			if err := validateServiceSecrets(environment, pair[0], pair[1]); err == nil {
				t.Fatalf("%s accepted unsafe secrets", environment)
			}
		}
	}
	if err := validateServiceSecrets("development", "dev", longB); err != nil {
		t.Fatalf("development rejected: %v", err)
	}
}

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
