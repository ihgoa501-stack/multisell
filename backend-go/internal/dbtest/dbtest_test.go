package dbtest

import (
	"testing"
)

type testModel struct {
	ID   uint `gorm:"primarykey"`
	Name string
}

func TestNewDB_CreatesDatabase(t *testing.T) {
	db := NewDB(t, &testModel{})
	if db == nil {
		t.Fatal("NewDB returned nil")
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("sqlDB.Ping() error: %v", err)
	}
}

func TestNewDB_MultipleCallsAreIsolated(t *testing.T) {
	db1 := NewDB(t, &testModel{})
	db2 := NewDB(t, &testModel{})

	sqlDB1, _ := db1.DB()
	sqlDB2, _ := db2.DB()

	if sqlDB1 == sqlDB2 {
		t.Fatal("expected separate database instances")
	}
}

func TestNewDB_AutoMigratesModels(t *testing.T) {
	db := NewDB(t, &testModel{})

	if err := db.AutoMigrate(&testModel{}); err != nil {
		t.Fatalf("AutoMigrate should already have run: %v", err)
	}

	if err := db.Create(&testModel{Name: "hello"}).Error; err != nil {
		t.Fatalf("Create after AutoMigrate failed: %v", err)
	}
}

func TestNewLogger_ReturnsLogger(t *testing.T) {
	logger := NewLogger(t)
	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}
}

func TestIToA(t *testing.T) {
	if got := IToA(42); got != "42" {
		t.Fatalf("IToA(42) = %q, want %q", got, "42")
	}
}

func TestFloatPtr(t *testing.T) {
	p := FloatPtr(3.14)
	if p == nil {
		t.Fatal("FloatPtr returned nil")
	}
	if *p != 3.14 {
		t.Fatalf("FloatPtr = %v, want 3.14", *p)
	}
}

func TestIntPtr(t *testing.T) {
	p := IntPtr(7)
	if p == nil {
		t.Fatal("IntPtr returned nil")
	}
	if *p != 7 {
		t.Fatalf("IntPtr = %v, want 7", *p)
	}
}

func TestStringPtr(t *testing.T) {
	p := StringPtr("hello")
	if p == nil {
		t.Fatal("StringPtr returned nil")
	}
	if *p != "hello" {
		t.Fatalf("StringPtr = %v, want hello", *p)
	}
}
