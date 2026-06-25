package platform

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestService_Platform_CRUD(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Platform{}, &Store{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create
	p, err := svc.CreatePlatform(&CreatePlatformInput{
		Name:       "Ozon",
		Code:       "ozon",
		APIBaseURL: "https://api.ozon.com",
	})
	if err != nil {
		t.Fatalf("CreatePlatform: %v", err)
	}
	if p.ID == 0 {
		t.Fatal("ID should be set")
	}
	if p.Code != "ozon" {
		t.Fatalf("Code = %s", p.Code)
	}
	if p.Status != 1 {
		t.Fatalf("Status = %d", p.Status)
	}

	// Get
	got, err := svc.GetPlatform(p.ID)
	if err != nil {
		t.Fatalf("GetPlatform: %v", err)
	}
	if got.Name != "Ozon" {
		t.Fatalf("Name = %s", got.Name)
	}

	// List
	pg := common.Pagination{Page: 1, Size: 10}
	items, total, err := svc.ListPlatforms(&pg, "")
	if err != nil {
		t.Fatalf("ListPlatforms: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d", total)
	}
	_ = items

	// Update
	updated, err := svc.UpdatePlatform(p.ID, &UpdatePlatformInput{
		Name: dbtest.StringPtr("Ozon RU"),
	})
	if err != nil {
		t.Fatalf("UpdatePlatform: %v", err)
	}
	if updated.Name != "Ozon RU" {
		t.Fatalf("Name = %s", updated.Name)
	}

	// Delete
	if err := svc.DeletePlatform(p.ID); err != nil {
		t.Fatalf("DeletePlatform: %v", err)
	}
	_, err = svc.GetPlatform(p.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestService_Store_CRUD(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Platform{}, &Store{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create a platform first
	p, _ := svc.CreatePlatform(&CreatePlatformInput{Name: "Shopee", Code: "shopee"})

	// Create store
	st, err := svc.CreateStore(&CreateStoreInput{
		UserID:            1,
		Name:              "我的店铺",
		PlatformID:        &p.ID,
		PlatformAccountID: "shop123",
	})
	if err != nil {
		t.Fatalf("CreateStore: %v", err)
	}
	if st.ID == 0 {
		t.Fatal("ID should be set")
	}
	if st.Name != "我的店铺" {
		t.Fatalf("Name = %s", st.Name)
	}

	// GetStore
	got, err := svc.GetStore(st.ID)
	if err != nil {
		t.Fatalf("GetStore: %v", err)
	}
	if got.UserID != 1 {
		t.Fatalf("UserID = %d", got.UserID)
	}

	// ListStores
	pg := common.Pagination{Page: 1, Size: 10}
	items, total, err := svc.ListStores(&pg, &p.ID, "")
	if err != nil {
		t.Fatalf("ListStores: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d", total)
	}
	_ = items

	// UpdateStore
	updated, err := svc.UpdateStore(st.ID, &UpdateStoreInput{
		Name: dbtest.StringPtr("新店名"),
	})
	if err != nil {
		t.Fatalf("UpdateStore: %v", err)
	}
	if updated.Name != "新店名" {
		t.Fatalf("Name = %s", updated.Name)
	}

	// DeleteStore
	if err := svc.DeleteStore(st.ID); err != nil {
		t.Fatalf("DeleteStore: %v", err)
	}
	_, err = svc.GetStore(st.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestService_Platform_ListSearch(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Platform{}, &Store{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.CreatePlatform(&CreatePlatformInput{Name: "Ozon", Code: "ozon"})
	svc.CreatePlatform(&CreatePlatformInput{Name: "Shopee", Code: "shopee"})

	// List without search (ILIKE is PG-specific)
	pg := common.Pagination{Page: 1, Size: 10}
	items, total, err := svc.ListPlatforms(&pg, "")
	if err != nil {
		t.Fatalf("ListPlatforms: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d", total)
	}
	_ = items
}
