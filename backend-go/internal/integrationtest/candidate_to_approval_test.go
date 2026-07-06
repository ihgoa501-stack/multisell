// Package integrationtest verifies cross-domain business flows end to end.
//
// These tests use in-memory SQLite (dbtest) to avoid external DB dependencies.
// They validate that the service layer wiring between domains is correct:
// flows that span Candidate → Completeness → Profit → Approval all work together.
package integrationtest

import (
	"encoding/json"
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/domain/completeness"
	"github.com/lingmirror/backend-go/internal/domain/profit"
	"gorm.io/gorm"
)

// models returns all GORM models needed by the candidate→approval flow.
func models() []interface{} {
	return []interface{}{
		&candidate.CandidateProduct{},
		&candidate.CollectLead{},
		&completeness.CompletenessCheck{},
		&completeness.CompletenessDimension{},
		&profit.ProfitSummary{},
		&approval.ApprovalRequest{},
	}
}

// TestCandidateToApprovalFlow exercises the full business chain:
//
//	candidate product → completeness check → profit calculation → approval request → approve
func TestCandidateToApprovalFlow(t *testing.T) {
	db := dbtest.NewDB(t, models()...)
	log := dbtest.NewLogger(t)

	candSvc := candidate.NewService(db, log)
	compSvc := completeness.NewService(db, log)
	profitSvc := profit.NewService(db, log, nil, 7.2)
	approvalSvc := approval.NewService(db, log, nil)

	// -- Step 1: Create CandidateProduct --
	price := 99.50
	in := &candidate.CreateCandidateInput{
		Title:            "Integration Flow Test",
		PurchasePrice:    &price,
		PurchaseCurrency: "CNY",
	}
	prod, err := candSvc.Create(in)
	if err != nil {
		t.Fatalf("Create candidate: %v", err)
	}
	if prod.ID == 0 {
		t.Fatal("expected non-zero candidate ID")
	}
	t.Logf("candidate_product id=%d", prod.ID)

	// -- Step 2: Completeness check --
	result, err := compSvc.Check(prod.ID, "integration_test")
	if err != nil {
		t.Fatalf("Completeness Check: %v", err)
	}
	if result.Score < 0 || result.Score > 100 {
		t.Fatalf("score out of range: %f", result.Score)
	}
	t.Logf("completeness score=%.1f%% status=%s dims=%d",
		result.Score, result.Status, len(result.Dimensions))

	// -- Step 3: Profit calculation --
	pr, err := profitSvc.Calculate(prod.ID, "integration_test")
	if err != nil {
		t.Fatalf("Profit Calculate: %v", err)
	}
	if pr.ProductID != prod.ID {
		t.Fatalf("profit product_id: got %d, want %d", pr.ProductID, prod.ID)
	}
	t.Logf("profit total_cost=%.2f margin=%.1f%% status=%s",
		pr.TotalCost, pr.ProfitMargin, pr.Status)

	// -- Step 4: Create approval request --
	req, err := approvalSvc.Create(&approval.CreateApprovalInput{
		ProductID:   prod.ID,
		RequestType: "listing_review",
		Requester:   "integration_test",
		Reason:      "Verify cross-domain flow",
		TargetType:  "candidate_product",
		TargetID:    prod.ID,
	})
	if err != nil {
		t.Fatalf("Create approval: %v", err)
	}
	if req.ID == 0 {
		t.Fatal("expected non-zero approval ID")
	}
	if req.Status != "pending" {
		t.Fatalf("approval status: got %q, want %q", req.Status, "pending")
	}
	t.Logf("approval id=%d status=%s", req.ID, req.Status)

	// -- Step 5: Review (approve) --
	reviewed, err := approvalSvc.Review(req.ID, &approval.ReviewApprovalInput{
		Action:     "approve",
		Reviewer:   "owner",
		ReviewNote: "Looks good",
	})
	if err != nil {
		t.Fatalf("Review approval: %v", err)
	}
	if reviewed.Status != "approved" {
		t.Fatalf("after approve: status=%q", reviewed.Status)
	}
	t.Logf("approval id=%d → %s", reviewed.ID, reviewed.Status)

	// -- Step 6: Verify via find approved --
	found, err := approvalSvc.FindApprovedByTarget("candidate_product", prod.ID, "listing_review")
	if err != nil {
		t.Fatalf("FindApprovedByTarget: %v", err)
	}
	if found == nil || found.ID != req.ID {
		t.Fatal("FindApprovedByTarget returned wrong record")
	}
}

// TestCandidateJSONPersistence verifies JSON blob fields survive write-then-read.
func TestCandidateJSONPersistence(t *testing.T) {
	db := dbtest.NewDB(t, models()...)
	log := dbtest.NewLogger(t)

	svc := candidate.NewService(db, log)

	images := json.RawMessage(`["https://img.example.com/1.jpg"]`)
	spec := json.RawMessage(`{"color": "red"}`)
	price := 200.0

	prod, err := svc.Create(&candidate.CreateCandidateInput{
		Title:            "JSON Persist Test",
		Images:           images,
		SpecJSON:         spec,
		PurchasePrice:    &price,
		PurchaseCurrency: "USD",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	saved, err := svc.GetByID(prod.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if string(saved.Images) != string(images) {
		t.Fatalf("images: got %s, want %s", saved.Images, images)
	}
	if string(saved.SpecJSON) != string(spec) {
		t.Fatalf("spec: got %s, want %s", saved.SpecJSON, spec)
	}
	if saved.PurchasePrice != price {
		t.Fatalf("purchase_price: got %.2f, want %.2f", saved.PurchasePrice, price)
	}
}

// TestCandidateSkipFieldAndDelete verifies SkipFieldCheck + Delete.
func TestCandidateSkipFieldAndDelete(t *testing.T) {
	db := dbtest.NewDB(t, models()...)
	log := dbtest.NewLogger(t)

	candSvc := candidate.NewService(db, log)
	compSvc := completeness.NewService(db, log)

	price := 50.0
	prod, err := candSvc.Create(&candidate.CreateCandidateInput{
		Title:            "SkipField Test",
		PurchasePrice:    &price,
		PurchaseCurrency: "CNY",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Skip a field
	if _, err := candSvc.SkipField(prod.ID, "main_image"); err != nil {
		t.Fatalf("SkipField: %v", err)
	}

	result, err := compSvc.Check(prod.ID, "integration_test")
	if err != nil {
		t.Fatalf("Completeness Check: %v", err)
	}
	if result.Score < 0 || result.Score > 100 {
		t.Fatalf("score out of range: %f", result.Score)
	}

	// Delete
	if err := candSvc.Delete(prod.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = candSvc.GetByID(prod.ID)
	if err != gorm.ErrRecordNotFound {
		t.Fatal("expected ErrRecordNotFound after delete")
	}
}

// TestCandidatePagination tests paginated listing with multiple products.
func TestCandidatePagination(t *testing.T) {
	db := dbtest.NewDB(t, models()...)
	log := dbtest.NewLogger(t)

	svc := candidate.NewService(db, log)

	for i := 0; i < 5; i++ {
		price := 100.0 + float64(i)
		_, err := svc.Create(&candidate.CreateCandidateInput{
			Title:            "Page Test",
			PurchasePrice:    &price,
			PurchaseCurrency: "CNY",
		})
		if err != nil {
			t.Fatalf("Create #%d: %v", i, err)
		}
	}

	p := common.Pagination{Page: 1, Size: 2}
	items, total, err := svc.List(&p, &candidate.ListCandidateFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 5 {
		t.Fatalf("total: got %d, want 5", total)
	}
	if len(items) != 2 {
		t.Fatalf("page 1 items: got %d, want 2", len(items))
	}

	p.Page = 3
	items, total, err = svc.List(&p, &candidate.ListCandidateFilter{})
	if err != nil {
		t.Fatalf("List page 3: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("page 3 items: got %d, want 1", len(items))
	}
}
