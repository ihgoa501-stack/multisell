package listing

import (
	"encoding/json"
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &ProductListing{})
}

func newService(t *testing.T) *Service {
	t.Helper()
	db := newTestDB(t)
	return NewService(db, dbtest.NewLogger(t), NewSKUProvider(db), NewDecisionReader(db))
}

func TestListing_Create(t *testing.T) {
	svc := newService(t)

	l, err := svc.Create(&CreateListingInput{
		ProductID:  1,
		PlatformID: 2,
		Status:     "draft",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if l.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if l.Status != "draft" {
		t.Fatalf("Status=%q, want draft", l.Status)
	}
}

func TestListing_Create_DefaultStatus(t *testing.T) {
	svc := newService(t)

	l, err := svc.Create(&CreateListingInput{
		ProductID:  1,
		PlatformID: 2,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if l.Status != "draft" {
		t.Fatalf("Status=%q, want draft", l.Status)
	}
}

func TestListing_GetByID(t *testing.T) {
	svc := newService(t)

	created, err := svc.Create(&CreateListingInput{
		ProductID:  1,
		PlatformID: 2,
		Status:     "active",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := svc.GetByID(created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.PlatformID != 2 {
		t.Fatalf("PlatformID=%d, want 2", got.PlatformID)
	}
}

func TestListing_GetByID_NotFound(t *testing.T) {
	svc := newService(t)

	if _, err := svc.GetByID(999); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestListing_ListByProduct(t *testing.T) {
	svc := newService(t)

	_, _ = svc.Create(&CreateListingInput{ProductID: 1, PlatformID: 1})
	_, _ = svc.Create(&CreateListingInput{ProductID: 1, PlatformID: 2})
	_, _ = svc.Create(&CreateListingInput{ProductID: 2, PlatformID: 1})

	items, err := svc.ListByProduct(1)
	if err != nil {
		t.Fatalf("ListByProduct failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
}

func TestListing_List(t *testing.T) {
	svc := newService(t)

	_, _ = svc.Create(&CreateListingInput{ProductID: 1, PlatformID: 1, Status: "draft"})
	_, _ = svc.Create(&CreateListingInput{ProductID: 2, PlatformID: 2, Status: "active"})
	_, _ = svc.Create(&CreateListingInput{ProductID: 3, PlatformID: 1, Status: "draft"})

	// No filters
	items, total, err := svc.List(&common.Pagination{Page: 1, Size: 10}, nil, "", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 3 {
		t.Fatalf("total=%d, want 3", total)
	}
	if len(items) != 3 {
		t.Fatalf("items=%d, want 3", len(items))
	}
}

func TestListing_List_FilterByPlatform(t *testing.T) {
	svc := newService(t)

	_, _ = svc.Create(&CreateListingInput{ProductID: 1, PlatformID: 1})
	_, _ = svc.Create(&CreateListingInput{ProductID: 2, PlatformID: 2})

	pid := int64(1)
	items, total, err := svc.List(&common.Pagination{Page: 1, Size: 10}, &pid, "", "")
	if err != nil {
		t.Fatalf("List with platform filter failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("total=%d, want 1", total)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d, want 1", len(items))
	}
}

func TestListing_Update(t *testing.T) {
	svc := newService(t)

	created, _ := svc.Create(&CreateListingInput{ProductID: 1, PlatformID: 1, Status: "draft"})

	status := "submitted"
	updated, err := svc.Update(created.ID, &UpdateListingInput{Status: &status})
	if err != nil {
		t.Fatalf("Update draft -> submitted failed: %v", err)
	}
	if updated.Status != "submitted" {
		t.Fatalf("after Update Status=%q, want submitted", updated.Status)
	}
}

func TestListing_Publish(t *testing.T) {
	svc := newService(t)

	created, _ := svc.Create(&CreateListingInput{ProductID: 1, PlatformID: 1})
	payload := json.RawMessage(`{"external_id":"ext-123"}`)

	published, err := svc.Publish(created.ID, payload)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	if published.Status != "submitted" {
		t.Fatalf("after Publish Status=%q, want submitted", published.Status)
	}
	if published.LastSyncAt == nil {
		t.Fatal("expected LastSyncAt to be set after Publish")
	}
}

func TestListing_SyncStatus(t *testing.T) {
	svc := newService(t)

	created, _ := svc.Create(&CreateListingInput{ProductID: 1, PlatformID: 1, Status: "draft"})
	// Submit listing first, then sync status to approved
	submitted, _ := svc.Publish(created.ID, json.RawMessage(`{}`))

	synced, err := svc.SyncStatus(submitted.ID, "approved", "Platform approved")
	if err != nil {
		t.Fatalf("SyncStatus failed: %v", err)
	}
	if synced.Status != "approved" {
		t.Fatalf("after SyncStatus Status=%q, want approved", synced.Status)
	}
	if synced.SyncMessage != "Platform approved" {
		t.Fatalf("SyncMessage=%q", synced.SyncMessage)
	}
}

func TestListing_Delete(t *testing.T) {
	svc := newService(t)

	created, _ := svc.Create(&CreateListingInput{ProductID: 1, PlatformID: 1})
	if err := svc.Delete(created.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := svc.GetByID(created.ID); err == nil {
		t.Fatal("expected error after Delete")
	}
}

func TestListing_Delete_NotFound(t *testing.T) {
	svc := newService(t)

	if err := svc.Delete(999); err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

// ---------- State machine integration tests ----------

func TestListing_InvalidStatusTransition(t *testing.T) {
	svc := newService(t)

	created, _ := svc.Create(&CreateListingInput{ProductID: 1, PlatformID: 1, Status: "draft"})
	status := "active"
	_, err := svc.Update(created.ID, &UpdateListingInput{Status: &status})
	if err == nil {
		t.Fatal("expected error for invalid transition draft -> active")
	}
}

func TestListing_SyncStatus_InvalidTransition(t *testing.T) {
	svc := newService(t)

	created, _ := svc.Create(&CreateListingInput{ProductID: 1, PlatformID: 1, Status: "draft"})
	// Can't go from draft directly to ended
	_, err := svc.SyncStatus(created.ID, "ended", "")
	if err == nil {
		t.Fatal("expected error for invalid transition draft -> ended")
	}
}

func TestListing_ValidStatusTransition(t *testing.T) {
	svc := newService(t)

	created, _ := svc.Create(&CreateListingInput{ProductID: 1, PlatformID: 1, Status: "draft"})
	status := "submitted"
	updated, err := svc.Update(created.ID, &UpdateListingInput{Status: &status})
	if err != nil {
		t.Fatalf("Update draft -> submitted failed: %v", err)
	}
	if updated.Status != "submitted" {
		t.Fatalf("Status=%q, want submitted", updated.Status)
	}
}

func TestListing_FullLifecycle(t *testing.T) {
	svc := newService(t)

	// Create a new listing in draft
	l, _ := svc.Create(&CreateListingInput{ProductID: 1, PlatformID: 1, Status: "draft"})

	// draft -> submitted
	s1 := "submitted"
	l, err := svc.Update(l.ID, &UpdateListingInput{Status: &s1})
	if err != nil {
		t.Fatalf("draft -> submitted: %v", err)
	}

	// submitted -> approved
	l, err = svc.SyncStatus(l.ID, "approved", "")
	if err != nil {
		t.Fatalf("submitted -> approved: %v", err)
	}

	// approved -> active
	s3 := "active"
	l, err = svc.Update(l.ID, &UpdateListingInput{Status: &s3})
	if err != nil {
		t.Fatalf("approved -> active: %v", err)
	}

	// active -> paused
	s4 := "paused"
	l, err = svc.Update(l.ID, &UpdateListingInput{Status: &s4})
	if err != nil {
		t.Fatalf("active -> paused: %v", err)
	}

	// paused -> ended
	s5 := "ended"
	l, err = svc.Update(l.ID, &UpdateListingInput{Status: &s5})
	if err != nil {
		t.Fatalf("paused -> ended: %v", err)
	}

	if l.Status != "ended" {
		t.Fatalf("final status=%q, want ended", l.Status)
	}
}

func TestListing_RejectedTransition(t *testing.T) {
	svc := newService(t)

	created, _ := svc.Create(&CreateListingInput{ProductID: 1, PlatformID: 1, Status: "draft"})
	status := "submitted"
	created, _ = svc.Update(created.ID, &UpdateListingInput{Status: &status})

	// submitted -> rejected
	synced, err := svc.SyncStatus(created.ID, "rejected", "Does not meet platform requirements")
	if err != nil {
		t.Fatalf("submitted -> rejected should be valid: %v", err)
	}
	if synced.Status != "rejected" {
		t.Fatalf("Status=%q, want rejected", synced.Status)
	}
}
