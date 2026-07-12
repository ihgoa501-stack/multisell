package schemadrift

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestMigrationChecker_AllApplied(t *testing.T) {
	dir := t.TempDir()
	createMigrationFile(t, dir, "000001_init.up.sql")
	createMigrationFile(t, dir, "000002_add_users.up.sql")
	createMigrationFile(t, dir, "000003_add_orders.up.sql")

	db := openTestDB(t)
	createSchemaMigrations(t, db, 1, 2, 3)

	checker := NewMigrationChecker(db, newTestLogger(t), dir)
	health := checker.Check()

	if health.CurrentVersion != 3 {
		t.Errorf("expected CurrentVersion=3, got %d", health.CurrentVersion)
	}
	if health.ExpectedVersion != 3 {
		t.Errorf("expected ExpectedVersion=3, got %d", health.ExpectedVersion)
	}
	if len(health.UnappliedMigrations) != 0 {
		t.Errorf("expected 0 unapplied, got %d: %+v", len(health.UnappliedMigrations), health.UnappliedMigrations)
	}
	if len(health.MissingMigrations) != 0 {
		t.Errorf("expected 0 missing, got %d: %+v", len(health.MissingMigrations), health.MissingMigrations)
	}
}

func TestMigrationChecker_Unapplied(t *testing.T) {
	dir := t.TempDir()
	createMigrationFile(t, dir, "000001_init.up.sql")
	createMigrationFile(t, dir, "000002_add_users.up.sql")
	createMigrationFile(t, dir, "000003_add_orders.up.sql")

	db := openTestDB(t)
	createSchemaMigrations(t, db, 1) // only version 1 applied

	checker := NewMigrationChecker(db, newTestLogger(t), dir)
	health := checker.Check()

	if health.CurrentVersion != 1 {
		t.Errorf("expected CurrentVersion=1, got %d", health.CurrentVersion)
	}
	if health.ExpectedVersion != 3 {
		t.Errorf("expected ExpectedVersion=3, got %d", health.ExpectedVersion)
	}
	if len(health.UnappliedMigrations) != 2 {
		t.Fatalf("expected 2 unapplied migrations, got %d", len(health.UnappliedMigrations))
	}
	if health.UnappliedMigrations[0].Version != 2 {
		t.Errorf("expected unapplied[0].Version=2, got %d", health.UnappliedMigrations[0].Version)
	}
	if health.UnappliedMigrations[1].Version != 3 {
		t.Errorf("expected unapplied[1].Version=3, got %d", health.UnappliedMigrations[1].Version)
	}
}

func TestMigrationChecker_DuplicateVersions(t *testing.T) {
	dir := t.TempDir()
	createMigrationFile(t, dir, "000001_init.up.sql")
	createMigrationFile(t, dir, "000001_duplicate.up.sql") // same version!
	createMigrationFile(t, dir, "000002_add_users.up.sql")

	db := openTestDB(t)
	createSchemaMigrations(t, db, 1, 2)

	checker := NewMigrationChecker(db, newTestLogger(t), dir)
	health := checker.Check()

	if len(health.DuplicateVersions) != 1 {
		t.Fatalf("expected 1 duplicate version, got %d: %v", len(health.DuplicateVersions), health.DuplicateVersions)
	}
	if health.DuplicateVersions[0] != 1 {
		t.Errorf("expected duplicate version 1, got %d", health.DuplicateVersions[0])
	}
}

func TestMigrationChecker_AllowsIntentionalVersionGaps(t *testing.T) {
	dir := t.TempDir()
	createMigrationFile(t, dir, "000001_init.up.sql")
	createMigrationFile(t, dir, "000003_add_orders.up.sql") // version 2 is missing

	db := openTestDB(t)
	createSchemaMigrations(t, db, 1, 3)

	checker := NewMigrationChecker(db, newTestLogger(t), dir)
	health := checker.Check()

	if len(health.MissingMigrations) != 0 {
		t.Fatalf("intentional version gap was treated as missing history: %+v", health.MissingMigrations)
	}
}

func TestMigrationChecker_CurrentVersionMustHaveFile(t *testing.T) {
	dir := t.TempDir()
	createMigrationFile(t, dir, "000001_init.up.sql")
	createMigrationFile(t, dir, "000003_add_orders.up.sql")
	db := openTestDB(t)
	createSchemaMigrations(t, db, 2)
	health := NewMigrationChecker(db, newTestLogger(t), dir).Check()
	if len(health.MissingMigrations) != 1 || health.MissingMigrations[0].Version != 2 {
		t.Fatalf("missing current file = %+v", health.MissingMigrations)
	}
}

func TestMigrationChecker_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	db := openTestDB(t)
	createSchemaMigrations(t, db)

	checker := NewMigrationChecker(db, newTestLogger(t), dir)
	health := checker.Check()

	if health.ExpectedVersion != 0 {
		t.Errorf("expected ExpectedVersion=0, got %d", health.ExpectedVersion)
	}
	if health.FileCount != 0 {
		t.Errorf("expected FileCount=0, got %d", health.FileCount)
	}
}

func TestMigrationChecker_FormatSummary(t *testing.T) {
	h := MigrationHealth{
		CurrentVersion:  1,
		ExpectedVersion: 3,
		FileCount:       3,
		AppliedInDB:     1,
		UnappliedMigrations: []MigrationFile{
			{Version: 2, Name: "add_users"},
			{Version: 3, Name: "add_orders"},
		},
		MissingMigrations: []MigrationFile{
			{Version: 4, Name: "(gap between 3 and 5)"},
		},
		DuplicateVersions: []int{1},
	}
	summary := h.FormatSummary()
	if summary == "" {
		t.Fatal("expected non-empty summary")
	}
	if !strings.Contains(summary, "current=1") {
		t.Error("summary should contain current version")
	}
	if !strings.Contains(summary, "unapplied") {
		t.Error("summary should mention unapplied")
	}
	if !strings.Contains(summary, "duplicate") {
		t.Error("summary should mention duplicates")
	}
}

func TestMigrationChecker_IgnoresDownFiles(t *testing.T) {
	dir := t.TempDir()
	createMigrationFile(t, dir, "000001_init.up.sql")
	createMigrationFile(t, dir, "000001_init.down.sql") // should be ignored
	createMigrationFile(t, dir, "000002_add_users.up.sql")

	db := openTestDB(t)
	createSchemaMigrations(t, db, 1, 2)

	checker := NewMigrationChecker(db, newTestLogger(t), dir)
	health := checker.Check()

	if health.FileCount != 2 { // only .up.sql files
		t.Errorf("expected 2 files (ignoring .down.sql), got %d", health.FileCount)
	}
}

func TestMigrationChecker_IgnoresNonMigrationFiles(t *testing.T) {
	dir := t.TempDir()
	createMigrationFile(t, dir, "000001_init.up.sql")
	createMigrationFile(t, dir, "README.md")
	createMigrationFile(t, dir, "validate.sql")

	db := openTestDB(t)
	createSchemaMigrations(t, db, 1)

	checker := NewMigrationChecker(db, newTestLogger(t), dir)
	health := checker.Check()

	if health.FileCount != 1 { // only 000001_init.up.sql
		t.Errorf("expected 1 migration file, got %d", health.FileCount)
	}
}

// --- Test helpers ---

func createMigrationFile(t *testing.T, dir, name string) {
	t.Helper()
	content := "SELECT 1;\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("failed to create migration file %s: %v", name, err)
	}
}

func createSchemaMigrations(t *testing.T, db *gorm.DB, versions ...int) {
	t.Helper()
	db.Exec("CREATE TABLE IF NOT EXISTS schema_migrations (version BIGINT NOT NULL PRIMARY KEY, dirty BOOLEAN NOT NULL)")
	if len(versions) > 0 {
		version := versions[len(versions)-1]
		db.Exec("INSERT OR REPLACE INTO schema_migrations (version, dirty) VALUES (?, ?)", version, false)
	}
}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t)
}

func newTestLogger(t *testing.T) *zap.Logger {
	t.Helper()
	return dbtest.NewLogger(t)
}
