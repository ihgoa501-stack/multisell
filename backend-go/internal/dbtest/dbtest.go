// Package dbtest provides shared test utilities for backend-go tests.
//
// Usage:
//
//	import "github.com/lingmirror/backend-go/internal/dbtest"
//
//	func TestSomething(t *testing.T) {
//	    db := dbtest.NewDB(t, &MyModel{})
//	    svc := NewService(db)
//	    // ...
//	}
package dbtest

import (
	"fmt"
	"sync/atomic"
	"testing"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var dbCounter atomic.Int64

// NewDB creates an isolated in-memory SQLite database and auto-migrates the
// given models. Each call returns an independent database instance, safe for
// parallel test execution.
func NewDB(t testing.TB, models ...interface{}) *gorm.DB {
	t.Helper()

	n := dbCounter.Add(1)
	dsn := fmt.Sprintf("file:test_%d?mode=memory&cache=shared&_journal_mode=WAL&_busy_timeout=5000", n)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("dbtest: failed to open in-memory SQLite: %v", err)
	}

	// ponytail: single connection serializes writes for SQLite concurrency.
	// PostgreSQL (production) handles concurrent writes natively.
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(1)
	}

	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			t.Fatalf("dbtest: AutoMigrate failed: %v", err)
		}
	}

	return db
}

// NewLogger returns a zap development logger suitable for test output.
func NewLogger(t testing.TB) *zap.Logger {
	t.Helper()
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("dbtest: failed to create test logger: %v", err)
	}
	return logger
}

// IToA converts an int64 to a string, matching the per-package helpers
// previously duplicated across test files.
func IToA(n int64) string {
	return fmt.Sprintf("%d", n)
}

// FloatPtr returns a pointer to the given float64 value.
func FloatPtr(v float64) *float64 {
	return &v
}

// IntPtr returns a pointer to the given int value.
func IntPtr(v int) *int {
	return &v
}

// StringPtr returns a pointer to the given string value.
func StringPtr(v string) *string {
	return &v
}
