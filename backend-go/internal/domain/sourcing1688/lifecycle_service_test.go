package sourcing1688

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"gorm.io/gorm"
)

const lifecycleTestOwnerID int64 = 7

func newLifecycleTestService(t *testing.T, status string, withDraft bool) (*Service, *gorm.DB, int64) {
	t.Helper()
	db := dbtest.NewDB(t,
		&Sourcing1688Product{},
		&demandCaseRow{}, &productRow{}, &skuRow{}, &mediaRow{}, &costRow{}, &listingRow{}, &draftRow{},
		&approval.ApprovalRequest{}, &operationlog.OperationLog{},
	)
	for _, column := range []struct {
		model     any
		name, sql string
	}{
		{&Sourcing1688Product{}, "lifecycle_status", `ALTER TABLE sourcing_1688_product ADD COLUMN lifecycle_status TEXT NOT NULL DEFAULT 'pending_review'`},
		{&Sourcing1688Product{}, "lifecycle_actor_id", `ALTER TABLE sourcing_1688_product ADD COLUMN lifecycle_actor_id INTEGER`},
		{&Sourcing1688Product{}, "lifecycle_reason", `ALTER TABLE sourcing_1688_product ADD COLUMN lifecycle_reason TEXT NOT NULL DEFAULT ''`},
		{&Sourcing1688Product{}, "lifecycle_updated_at", `ALTER TABLE sourcing_1688_product ADD COLUMN lifecycle_updated_at TIMESTAMP`},
		{&draftRow{}, "approval_id", `ALTER TABLE sourcing_listing_draft ADD COLUMN approval_id INTEGER`},
		{&draftRow{}, "approval_status", `ALTER TABLE sourcing_listing_draft ADD COLUMN approval_status TEXT NOT NULL DEFAULT ''`},
		{&draftRow{}, "approval_rejection_reason", `ALTER TABLE sourcing_listing_draft ADD COLUMN approval_rejection_reason TEXT NOT NULL DEFAULT ''`},
	} {
		if db.Migrator().HasColumn(column.model, column.name) {
			continue
		}
		if err := db.Exec(column.sql).Error; err != nil {
			t.Fatalf("add lifecycle test column: %v", err)
		}
	}
	// SQLite does not execute the PostgreSQL migration. This trigger mirrors the
	// migration's state application so service tests exercise the canonical
	// approval.Service.Review seam instead of directly mutating approval rows.
	if err := db.Exec(`CREATE TRIGGER test_apply_sourcing_approval
		AFTER UPDATE OF status ON approval_request
		WHEN NEW.request_type = 'sourcing_1688_draft' AND OLD.status = 'pending'
		BEGIN
			UPDATE sourcing_listing_draft
			SET approval_status = NEW.status,
				approval_rejection_reason = CASE WHEN NEW.status = 'rejected' THEN NEW.review_note ELSE '' END
			WHERE id = NEW.target_id AND approval_id = NEW.id;
			UPDATE sourcing_1688_product
			SET lifecycle_status = CASE WHEN NEW.status = 'approved' THEN 'approved_draft' ELSE 'editing' END,
				lifecycle_actor_id = NEW.reviewer_user_id,
				lifecycle_reason = CASE WHEN NEW.status = 'rejected' THEN NEW.review_note ELSE '' END
			WHERE id = (SELECT sourcing_product_id FROM sourcing_listing_draft WHERE id = NEW.target_id)
				AND lifecycle_status = 'pending_approval';
		END`).Error; err != nil {
		t.Fatalf("create lifecycle test trigger: %v", err)
	}
	if err := db.Create(&demandCaseRow{ID: 10, OwnerID: lifecycleTestOwnerID, SalesChannel: "test", Status: "experiment_ready"}).Error; err != nil {
		t.Fatalf("create demand case: %v", err)
	}
	demandCaseID := int64(10)
	p := Sourcing1688Product{SourceURL: fmt.Sprintf("https://detail.1688.com/offer/%d.html", len(status)+1), Status: StatusPendingReview, DemandCaseID: &demandCaseID}
	if err := db.Create(&p).Error; err != nil {
		t.Fatalf("create sourcing product: %v", err)
	}
	if err := db.Model(&sourcingLifecycleRow{}).Where("id = ?", p.ID).Updates(map[string]any{"lifecycle_status": status, "lifecycle_reason": ""}).Error; err != nil {
		t.Fatalf("set lifecycle: %v", err)
	}
	snapshotID := int64(99)
	if err := db.Model(&Sourcing1688Product{}).Where("id = ?", p.ID).Update("snapshot_id", snapshotID).Error; err != nil {
		t.Fatalf("set snapshot reference: %v", err)
	}
	if withDraft {
		productID := int64(101)
		if err := db.Create(&productRow{ID: productID, Name: "approval test product", Unit: "piece", CategoryID: 1}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&Sourcing1688Product{}).Where("id = ?", p.ID).Update("product_id", productID).Error; err != nil {
			t.Fatal(err)
		}
		listing := listingRow{ProductID: productID, PlatformID: 1, PlatformSKU: "SKU-1", Status: "draft"}
		if err := db.Create(&listing).Error; err != nil {
			t.Fatal(err)
		}
		draft := draftRow{SourcingProductID: p.ID, SnapshotID: 1, ProductID: productID, ListingID: listing.ID, DemandCaseID: demandCaseID, ExperimentID: "exp-1", CreatedBy: lifecycleTestOwnerID}
		if err := db.Create(&draft).Error; err != nil {
			t.Fatal(err)
		}
	}
	return NewService(db, dbtest.NewLogger(t)), db, p.ID
}

func TestLifecycle_CaptureFailureAndSourceRejectionPersistReasons(t *testing.T) {
	t.Run("capture failure", func(t *testing.T) {
		svc, _, id := newLifecycleTestService(t, LifecyclePendingReview, false)
		got, err := svc.MarkCaptureFailed(id, &CaptureFailureInput{ActorID: lifecycleTestOwnerID, Reason: "login expired"})
		if err != nil {
			t.Fatalf("MarkCaptureFailed: %v", err)
		}
		if got.Status != LifecycleCaptureFailed || got.Reason != "login expired" {
			t.Fatalf("state = %+v", got)
		}
	})
	t.Run("source rejected", func(t *testing.T) {
		svc, db, id := newLifecycleTestService(t, LifecyclePendingReview, false)
		got, err := svc.DecideSourceReview(id, &SourceReviewDecisionInput{OwnerID: lifecycleTestOwnerID, Action: "reject", Notes: "patent evidence missing"})
		if err != nil {
			t.Fatalf("DecideSourceReview: %v", err)
		}
		if got.Status != LifecycleRejected || got.Reason != "patent evidence missing" {
			t.Fatalf("state = %+v", got)
		}
		var p Sourcing1688Product
		if err := db.First(&p, id).Error; err != nil {
			t.Fatal(err)
		}
		if p.Status != LifecycleRejected || p.ReviewNotes != "patent evidence missing" {
			t.Fatalf("legacy review not persisted: %+v", p)
		}
	})
}

func TestLifecycle_FullOwnerApprovalProducesDraftOnly(t *testing.T) {
	svc, db, id := newLifecycleTestService(t, LifecyclePendingReview, false)
	ready, err := svc.DecideSourceReview(id, &SourceReviewDecisionInput{OwnerID: lifecycleTestOwnerID, Action: "approve", Notes: "source verified"})
	if err != nil {
		t.Fatalf("source approval: %v", err)
	}
	if ready.Status != LifecycleReadyForProduct {
		t.Fatalf("status = %s", ready.Status)
	}

	// Conversion is implemented by Convert; this test creates its minimum
	// traceable result and verifies lifecycle behavior after that seam.
	productID := int64(101)
	if err := db.Create(&productRow{ID: productID, Name: "approval test product", Unit: "piece", CategoryID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&Sourcing1688Product{}).Where("id = ?", id).Update("product_id", productID).Error; err != nil {
		t.Fatal(err)
	}
	listing := listingRow{ProductID: productID, PlatformID: 1, PlatformSKU: "SKU-1", Status: "draft"}
	if err := db.Create(&listing).Error; err != nil {
		t.Fatal(err)
	}
	draft := draftRow{SourcingProductID: id, SnapshotID: 1, ProductID: productID, ListingID: listing.ID, DemandCaseID: 10, ExperimentID: "exp-1", CreatedBy: lifecycleTestOwnerID}
	if err := db.Create(&draft).Error; err != nil {
		t.Fatal(err)
	}

	// Convert owns ready_for_product -> editing in the real workflow.
	if err := db.Model(&sourcingLifecycleRow{}).Where("id = ?", id).Update("lifecycle_status", LifecycleEditing).Error; err != nil {
		t.Fatal(err)
	}

	submitted, err := svc.SubmitDraftApproval(id, &DraftApprovalSubmissionInput{RequesterID: lifecycleTestOwnerID, Reason: "ready for Owner preview"})
	if err != nil {
		t.Fatalf("SubmitDraftApproval: %v", err)
	}
	if submitted.Lifecycle.Status != LifecyclePendingApproval || submitted.ApprovalStatus != approval.StatusPending {
		t.Fatalf("submitted = %+v", submitted)
	}
	var req approval.ApprovalRequest
	if err := db.First(&req, submitted.ApprovalID).Error; err != nil {
		t.Fatal(err)
	}
	if req.TargetType != DraftApprovalTargetType || req.TargetID != draft.ID || req.RequestType != DraftApprovalRequestType {
		t.Fatalf("approval linkage = %+v", req)
	}

	approved, err := svc.DecideDraftApproval(id, submitted.ApprovalID, &DraftApprovalDecisionInput{OwnerID: lifecycleTestOwnerID, Action: "approve", Note: "approved as internal draft"})
	if err != nil {
		t.Fatalf("DecideDraftApproval: %v", err)
	}
	if approved.Lifecycle.Status != LifecycleApprovedDraft || approved.ApprovalStatus != approval.StatusApproved {
		t.Fatalf("approved = %+v", approved)
	}
	if err := db.First(&listing, listing.ID).Error; err != nil {
		t.Fatal(err)
	}
	if listing.Status != "draft" {
		t.Fatalf("approval changed listing status to %q", listing.Status)
	}
	var auditCount int64
	if err := db.Model(&operationlog.OperationLog{}).Where("action = ? AND resource_id = ?", "approval.review", fmt.Sprintf("%d", submitted.ApprovalID)).Count(&auditCount).Error; err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("approval audit count = %d", auditCount)
	}
}

func TestLifecycle_DraftRejectionReturnsToEditingAndPersistsReason(t *testing.T) {
	svc, db, id := newLifecycleTestService(t, LifecycleEditing, true)
	submitted, err := svc.SubmitDraftApproval(id, &DraftApprovalSubmissionInput{RequesterID: lifecycleTestOwnerID, Reason: "review"})
	if err != nil {
		t.Fatal(err)
	}
	rejected, err := svc.DecideDraftApproval(id, submitted.ApprovalID, &DraftApprovalDecisionInput{OwnerID: lifecycleTestOwnerID, Action: "reject", Note: "replace main image"})
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Lifecycle.Status != LifecycleEditing || rejected.Lifecycle.Reason != "replace main image" || rejected.ApprovalStatus != approval.StatusRejected {
		t.Fatalf("rejected = %+v", rejected)
	}
	var draft draftRow
	if err := db.Where("sourcing_product_id = ?", id).First(&draft).Error; err != nil {
		t.Fatal(err)
	}
	if draft.ApprovalRejectionReason != "replace main image" || draft.ApprovalStatus != approval.StatusRejected {
		t.Fatalf("draft = %+v", draft)
	}
}

func TestLifecycleApprovalRejectsContentChangedAfterSubmission(t *testing.T) {
	svc, db, id := newLifecycleTestService(t, LifecycleEditing, true)
	submitted, err := svc.SubmitDraftApproval(id, &DraftApprovalSubmissionInput{RequesterID: lifecycleTestOwnerID, Reason: "review exact content"})
	if err != nil {
		t.Fatal(err)
	}
	var draft draftRow
	if err := db.Where("sourcing_product_id = ?", id).First(&draft).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&productRow{}).Where("id = ?", draft.ProductID).Update("name", "changed after submit").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DecideDraftApproval(id, submitted.ApprovalID, &DraftApprovalDecisionInput{OwnerID: lifecycleTestOwnerID, Action: "approve", Note: "must reject stale review"}); !errors.Is(err, ErrWorkflowGate) {
		t.Fatalf("tampered draft approval error = %v", err)
	}
}

func TestLifecycle_RejectsEveryStateEscalation(t *testing.T) {
	tests := []struct {
		name   string
		status string
		draft  bool
		call   func(*Service, int64) error
	}{
		{"failure after review", LifecycleReadyForProduct, false, func(s *Service, id int64) error {
			_, e := s.MarkCaptureFailed(id, &CaptureFailureInput{ActorID: lifecycleTestOwnerID, Reason: "late"})
			return e
		}},
		{"review twice", LifecycleReadyForProduct, false, func(s *Service, id int64) error {
			_, e := s.DecideSourceReview(id, &SourceReviewDecisionInput{OwnerID: lifecycleTestOwnerID, Action: "approve", Notes: "again"})
			return e
		}},
		{"ready directly to approval", LifecycleReadyForProduct, true, func(s *Service, id int64) error {
			_, e := s.SubmitDraftApproval(id, &DraftApprovalSubmissionInput{RequesterID: lifecycleTestOwnerID, Reason: "skip"})
			return e
		}},
		{"editing directly to approved", LifecycleEditing, true, func(s *Service, id int64) error {
			_, e := s.DecideDraftApproval(id, 1, &DraftApprovalDecisionInput{OwnerID: lifecycleTestOwnerID, Action: "approve", Note: "skip"})
			return e
		}},
		{"approved resubmission", LifecycleApprovedDraft, true, func(s *Service, id int64) error {
			_, e := s.SubmitDraftApproval(id, &DraftApprovalSubmissionInput{RequesterID: lifecycleTestOwnerID, Reason: "again"})
			return e
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, id := newLifecycleTestService(t, tc.status, tc.draft)
			if err := tc.call(svc, id); !errors.Is(err, ErrInvalidLifecycleTransition) {
				t.Fatalf("expected transition error, got %v", err)
			}
		})
	}
}

func TestLifecycle_RejectsWrongOwnerAndMismatchedApproval(t *testing.T) {
	t.Run("wrong owner", func(t *testing.T) {
		svc, _, id := newLifecycleTestService(t, LifecyclePendingReview, false)
		_, err := svc.DecideSourceReview(id, &SourceReviewDecisionInput{OwnerID: 999, Action: "approve", Notes: "not mine"})
		if !errors.Is(err, ErrWorkflowGate) {
			t.Fatalf("expected workflow gate, got %v", err)
		}
	})
	t.Run("approval from another draft", func(t *testing.T) {
		svc, _, id := newLifecycleTestService(t, LifecycleEditing, true)
		submitted, err := svc.SubmitDraftApproval(id, &DraftApprovalSubmissionInput{RequesterID: lifecycleTestOwnerID, Reason: "review"})
		if err != nil {
			t.Fatal(err)
		}
		_, err = svc.DecideDraftApproval(id, submitted.ApprovalID+1, &DraftApprovalDecisionInput{OwnerID: lifecycleTestOwnerID, Action: "approve", Note: "wrong"})
		if !errors.Is(err, ErrWorkflowGate) {
			t.Fatalf("expected workflow gate, got %v", err)
		}
	})
	t.Run("listing is no longer draft", func(t *testing.T) {
		svc, db, id := newLifecycleTestService(t, LifecycleEditing, true)
		submitted, err := svc.SubmitDraftApproval(id, &DraftApprovalSubmissionInput{RequesterID: lifecycleTestOwnerID, Reason: "review"})
		if err != nil {
			t.Fatal(err)
		}
		var draft draftRow
		if err := db.Where("sourcing_product_id = ?", id).First(&draft).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&listingRow{}).Where("id = ?", draft.ListingID).Update("status", "published").Error; err != nil {
			t.Fatal(err)
		}
		_, err = svc.DecideDraftApproval(id, submitted.ApprovalID, &DraftApprovalDecisionInput{OwnerID: lifecycleTestOwnerID, Action: "approve", Note: "must stay draft"})
		if !errors.Is(err, ErrWorkflowGate) {
			t.Fatalf("expected workflow gate, got %v", err)
		}
	})
}

func TestLifecycle_TransitionMatrixRejectsAllNonEdges(t *testing.T) {
	states := []string{
		LifecycleCaptureFailed, LifecyclePendingReview, LifecycleRejected,
		LifecycleReadyForProduct, LifecycleEditing, LifecyclePendingApproval,
		LifecycleApprovedDraft,
	}
	edges := []struct{ from, to string }{
		{LifecyclePendingReview, LifecycleCaptureFailed},
		{LifecyclePendingReview, LifecycleRejected},
		{LifecyclePendingReview, LifecycleReadyForProduct},
		{LifecycleReadyForProduct, LifecycleEditing},
		{LifecycleEditing, LifecyclePendingApproval},
		{LifecyclePendingApproval, LifecycleEditing},
		{LifecyclePendingApproval, LifecycleApprovedDraft},
	}
	for _, edge := range edges {
		for _, current := range states {
			err := requireTransition(current, edge.from, edge.to)
			if current == edge.from && err != nil {
				t.Errorf("legal edge %s -> %s rejected: %v", edge.from, edge.to, err)
			}
			if current != edge.from && !errors.Is(err, ErrInvalidLifecycleTransition) {
				t.Errorf("illegal escalation %s -> %s was not rejected", current, edge.to)
			}
		}
	}
}
