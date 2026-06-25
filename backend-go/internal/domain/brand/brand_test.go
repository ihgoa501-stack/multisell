package brand

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &Brand{})
}

func TestBrand_Create(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	brand := &Brand{Name: "Test Brand", Description: "A test brand"}
	if err := svc.Create(context.Background(), brand); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if brand.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}
}

func TestBrand_Create_EmptyName(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	brand := &Brand{Name: ""}
	if err := svc.Create(context.Background(), brand); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestBrand_GetByID(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	created := &Brand{Name: "Findable Brand"}
	if err := svc.Create(context.Background(), created); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := svc.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Name != "Findable Brand" {
		t.Fatalf("GetByID.Name = %q, want %q", got.Name, "Findable Brand")
	}
}

func TestBrand_GetByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	if _, err := svc.GetByID(context.Background(), 999); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestBrand_List_Pagination(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	for i := 0; i < 25; i++ {
		if err := svc.Create(context.Background(), &Brand{Name: dbtest.IToA(int64(i))}); err != nil {
			t.Fatalf("Create %d failed: %v", i, err)
		}
	}

	items, total, err := svc.List(context.Background(), 1, 10, "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 25 {
		t.Fatalf("List total = %d, want 25", total)
	}
	if len(items) != 10 {
		t.Fatalf("List returned %d items, want 10", len(items))
	}
}

func TestBrand_ListAll(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	_ = svc.Create(context.Background(), &Brand{Name: "Active1", Status: 1})
	_ = svc.Create(context.Background(), &Brand{Name: "Active2", Status: 1})
	_ = svc.Create(context.Background(), &Brand{Name: "Inactive", Status: 0})
	// SQLite + GORM zero-value handling: explicit UPDATE to set inactive status
	db.Model(&Brand{}).Where("name = ?", "Inactive").Update("status", 0)

	items, err := svc.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("ListAll returned %d items, want 2", len(items))
	}
}

func TestBrand_Update(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	brand := &Brand{Name: "Old Name"}
	if err := svc.Create(context.Background(), brand); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	brand.Name = "Updated Name"
	if err := svc.Update(context.Background(), brand); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := svc.GetByID(context.Background(), brand.ID)
	if got.Name != "Updated Name" {
		t.Fatalf("after Update Name = %q, want %q", got.Name, "Updated Name")
	}
}

func TestBrand_Update_EmptyName(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	brand := &Brand{Name: "Valid"}
	_ = svc.Create(context.Background(), brand)

	brand.Name = ""
	if err := svc.Update(context.Background(), brand); err == nil {
		t.Fatal("expected error for empty name on update")
	}
}

func TestBrand_Delete(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	brand := &Brand{Name: "To Delete"}
	if err := svc.Create(context.Background(), brand); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := svc.Delete(context.Background(), brand.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := svc.GetByID(context.Background(), brand.ID); err == nil {
		t.Fatal("expected error after Delete")
	}
}

func TestBrand_Delete_NotFound(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	if err := svc.Delete(context.Background(), 999); err != nil {
		t.Fatalf("Delete for non-existent ID should succeed: %v", err)
	}
}
