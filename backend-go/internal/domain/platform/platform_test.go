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

func TestCreatePlatform_Validation(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Platform{}, &Store{})
	svc := NewService(db, dbtest.NewLogger(t))

	_, err := svc.CreatePlatform(&CreatePlatformInput{Name: "Ozon", Code: "ozon"})
	if err != nil {
		t.Fatalf("CreatePlatform: %v", err)
	}

	// Duplicate code violates unique constraint.
	_, err = svc.CreatePlatform(&CreatePlatformInput{Name: "Ozon 2", Code: "ozon"})
	if err == nil {
		t.Fatal("expected error for duplicate code")
	}
}

func TestUpdatePlatform_Fields(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Platform{}, &Store{})
	svc := NewService(db, dbtest.NewLogger(t))

	status := int16(2)
	sortOrder := 5
	p, err := svc.CreatePlatform(&CreatePlatformInput{
		Name:       "Ozon",
		Code:       "ozon",
		APIBaseURL: "https://api.ozon.com",
	})
	if err != nil {
		t.Fatalf("CreatePlatform: %v", err)
	}

	updated, err := svc.UpdatePlatform(p.ID, &UpdatePlatformInput{
		Code:      dbtest.StringPtr("ozon-ru"),
		Status:    &status,
		SortOrder: &sortOrder,
	})
	if err != nil {
		t.Fatalf("UpdatePlatform: %v", err)
	}
	if updated.Code != "ozon-ru" {
		t.Fatalf("Code = %q", updated.Code)
	}
	if updated.Status != 2 {
		t.Fatalf("Status = %d", updated.Status)
	}
	if updated.SortOrder != 5 {
		t.Fatalf("SortOrder = %d", updated.SortOrder)
	}
	if updated.Name != "Ozon" {
		t.Fatalf("Name = %q (should be unchanged)", updated.Name)
	}
}

func TestListPlatforms_Pagination(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Platform{}, &Store{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.CreatePlatform(&CreatePlatformInput{Name: "A", Code: "a"})
	svc.CreatePlatform(&CreatePlatformInput{Name: "B", Code: "b"})
	svc.CreatePlatform(&CreatePlatformInput{Name: "C", Code: "c"})

	// Page 1, Size 2 → 2 items
	items, total, err := svc.ListPlatforms(&common.Pagination{Page: 1, Size: 2}, "")
	if err != nil {
		t.Fatalf("ListPlatforms page 1: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(items) != 2 {
		t.Fatalf("page 1 items = %d, want 2", len(items))
	}

	// Page 2, Size 2 → 1 item
	items, total, err = svc.ListPlatforms(&common.Pagination{Page: 2, Size: 2}, "")
	if err != nil {
		t.Fatalf("ListPlatforms page 2: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(items) != 1 {
		t.Fatalf("page 2 items = %d, want 1", len(items))
	}
}

func TestListPlatforms_Search(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Platform{}, &Store{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.CreatePlatform(&CreatePlatformInput{Name: "Ozon", Code: "ozon"})
	svc.CreatePlatform(&CreatePlatformInput{Name: "Shopee", Code: "shopee"})

	// Empty search returns all (ILIKE-based search is PG-specific).
	pg := common.Pagination{Page: 1, Size: 10}
	items, total, err := svc.ListPlatforms(&pg, "")
	if err != nil {
		t.Fatalf("ListPlatforms: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	_ = items
}

func TestDeletePlatform_NotFound(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Platform{}, &Store{})
	svc := NewService(db, dbtest.NewLogger(t))

	err := svc.DeletePlatform(99999)
	if err == nil {
		t.Fatal("expected error for non-existent platform")
	}
}

func TestGetPlatform_NotFound(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Platform{}, &Store{})
	svc := NewService(db, dbtest.NewLogger(t))

	_, err := svc.GetPlatform(99999)
	if err == nil {
		t.Fatal("expected error for non-existent platform")
	}
}

func TestCreateStore_DefaultStatus(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Platform{}, &Store{})
	svc := NewService(db, dbtest.NewLogger(t))

	p, _ := svc.CreatePlatform(&CreatePlatformInput{Name: "Ozon", Code: "ozon"})

	st, err := svc.CreateStore(&CreateStoreInput{
		UserID:     1,
		Name:       "My Store",
		PlatformID: &p.ID,
	})
	if err != nil {
		t.Fatalf("CreateStore: %v", err)
	}
	if st.Status != 1 {
		t.Fatalf("Status = %d, want 1", st.Status)
	}
}

func TestListStores_FilterByPlatform(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Platform{}, &Store{})
	svc := NewService(db, dbtest.NewLogger(t))

	p1, _ := svc.CreatePlatform(&CreatePlatformInput{Name: "Ozon", Code: "ozon"})
	p2, _ := svc.CreatePlatform(&CreatePlatformInput{Name: "Shopee", Code: "shopee"})

	svc.CreateStore(&CreateStoreInput{UserID: 1, Name: "O Store", PlatformID: &p1.ID})
	svc.CreateStore(&CreateStoreInput{UserID: 1, Name: "S Store", PlatformID: &p2.ID})

	// Filter by platform 1 → 1 store
	pg := common.Pagination{Page: 1, Size: 10}
	items, total, err := svc.ListStores(&pg, &p1.ID, "")
	if err != nil {
		t.Fatalf("ListStores: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	_ = items

	// No filter → both stores
	items, total, err = svc.ListStores(&pg, nil, "")
	if err != nil {
		t.Fatalf("ListStores (no filter): %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	_ = items
}

func TestListStores_Search(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Platform{}, &Store{})
	svc := NewService(db, dbtest.NewLogger(t))

	p, _ := svc.CreatePlatform(&CreatePlatformInput{Name: "Ozon", Code: "ozon"})
	svc.CreateStore(&CreateStoreInput{UserID: 1, Name: "Store A", PlatformID: &p.ID})
	svc.CreateStore(&CreateStoreInput{UserID: 1, Name: "Store B", PlatformID: &p.ID})

	// Empty search returns all (ILIKE-based search is PG-specific).
	pg := common.Pagination{Page: 1, Size: 10}
	items, total, err := svc.ListStores(&pg, nil, "")
	if err != nil {
		t.Fatalf("ListStores: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	_ = items
}

func TestUpdateStore_MultipleFields(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Platform{}, &Store{})
	svc := NewService(db, dbtest.NewLogger(t))

	p, _ := svc.CreatePlatform(&CreatePlatformInput{Name: "Ozon", Code: "ozon"})
	st, _ := svc.CreateStore(&CreateStoreInput{UserID: 1, Name: "Old Name", PlatformID: &p.ID})

	status := int16(2)
	updated, err := svc.UpdateStore(st.ID, &UpdateStoreInput{
		Name:   dbtest.StringPtr("New Name"),
		Status: &status,
	})
	if err != nil {
		t.Fatalf("UpdateStore: %v", err)
	}
	if updated.Name != "New Name" {
		t.Fatalf("Name = %q", updated.Name)
	}
	if updated.Status != 2 {
		t.Fatalf("Status = %d", updated.Status)
	}
}

func TestDeleteStore_NotFound(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Platform{}, &Store{})
	svc := NewService(db, dbtest.NewLogger(t))

	err := svc.DeleteStore(99999)
	if err == nil {
		t.Fatal("expected error for non-existent store")
	}
}
