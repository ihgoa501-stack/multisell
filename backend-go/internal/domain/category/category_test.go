package category

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &Category{})
}

func newService(t *testing.T) *Service {
	t.Helper()
	return NewService(newTestDB(t), dbtest.NewLogger(t))
}

func TestCategory_Create(t *testing.T) {
	svc := newService(t)

	c := &Category{Name: "Electronics"}
	if err := svc.Create(context.Background(), c); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if c.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestCategory_Create_EmptyName(t *testing.T) {
	svc := newService(t)
	if err := svc.Create(context.Background(), &Category{Name: ""}); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestCategory_Create_WithParent(t *testing.T) {
	svc := newService(t)

	parent := &Category{Name: "Parent"}
	_ = svc.Create(context.Background(), parent)

	child := &Category{Name: "Child", ParentID: parent.ID}
	if err := svc.Create(context.Background(), child); err != nil {
		t.Fatalf("Create child failed: %v", err)
	}
	if child.ParentID != parent.ID {
		t.Fatalf("ParentID=%d, want %d", child.ParentID, parent.ID)
	}
}

func TestCategory_GetByID(t *testing.T) {
	svc := newService(t)

	created := &Category{Name: "Findable"}
	_ = svc.Create(context.Background(), created)

	got, err := svc.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Name != "Findable" {
		t.Fatalf("Name=%q, want Findable", got.Name)
	}
}

func TestCategory_GetByID_NotFound(t *testing.T) {
	svc := newService(t)
	if _, err := svc.GetByID(context.Background(), 999); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestCategory_Update(t *testing.T) {
	svc := newService(t)

	c := &Category{Name: "Old"}
	_ = svc.Create(context.Background(), c)

	c.Name = "Updated"
	if err := svc.Update(context.Background(), c); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := svc.GetByID(context.Background(), c.ID)
	if got.Name != "Updated" {
		t.Fatalf("Name=%q, want Updated", got.Name)
	}
}

func TestCategory_Delete(t *testing.T) {
	svc := newService(t)

	c := &Category{Name: "To Delete"}
	_ = svc.Create(context.Background(), c)

	if err := svc.Delete(context.Background(), c.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := svc.GetByID(context.Background(), c.ID); err == nil {
		t.Fatal("expected error after Delete")
	}
}

func TestCategory_GetTree(t *testing.T) {
	svc := newService(t)

	parent := &Category{Name: "Root"}
	_ = svc.Create(context.Background(), parent)

	_ = svc.Create(context.Background(), &Category{Name: "Child1", ParentID: parent.ID})
	_ = svc.Create(context.Background(), &Category{Name: "Child2", ParentID: parent.ID})

	tree, err := svc.GetTree(context.Background())
	if err != nil {
		t.Fatalf("GetTree failed: %v", err)
	}
	if len(tree) != 1 {
		t.Fatalf("roots=%d, want 1", len(tree))
	}
	if len(tree[0].Children) != 2 {
		t.Fatalf("children=%d, want 2", len(tree[0].Children))
	}
}
