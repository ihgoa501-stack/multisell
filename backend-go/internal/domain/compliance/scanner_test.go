package compliance

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

// productTestModel creates a product table for test isolation.
type productTestModel struct {
	ID         int64  `gorm:"column:id;primaryKey;autoIncrement"`
	Name       string `gorm:"column:name;not null"`
	CategoryID int64  `gorm:"column:category_id"`
}

func (productTestModel) TableName() string { return "product" }

func TestScanner_ScanPaginated_EmptyDB(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &productTestModel{})
	s := NewScanner(db, dbtest.NewLogger(t))

	result, err := s.ScanPaginated(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalProducts != 0 {
		t.Errorf("expected 0 total, got %d", result.TotalProducts)
	}
	if result.ScannedProducts != 0 {
		t.Errorf("expected 0 scanned, got %d", result.ScannedProducts)
	}
	if result.IssuesFound != 0 {
		t.Errorf("expected 0 issues, got %d", result.IssuesFound)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}
}

func TestScanner_ScanPaginated_WithProducts(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &productTestModel{})
	s := NewScanner(db, dbtest.NewLogger(t))

	// Insert test products directly via raw SQL.
	for i := 1; i <= 3; i++ {
		if err := db.Exec(
			"INSERT INTO product (id, name, category_id) VALUES (?, ?, ?)",
			i, "Test Product "+dbtest.IToA(int64(i)), int64(100+i),
		).Error; err != nil {
			t.Fatalf("insert product %d: %v", i, err)
		}
	}

	result, err := s.ScanPaginated(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalProducts != 3 {
		t.Errorf("expected 3 total, got %d", result.TotalProducts)
	}
	if result.ScannedProducts != 3 {
		t.Errorf("expected 3 scanned, got %d", result.ScannedProducts)
	}
	if result.IssuesFound != 0 {
		t.Errorf("expected 0 issues, got %d", result.IssuesFound)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}
}
