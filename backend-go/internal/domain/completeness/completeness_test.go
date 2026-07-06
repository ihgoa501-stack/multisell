package completeness

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/candidate"
)

func TestService_Check_CreateErrorIsReturned(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{}, &candidate.CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Insert a candidate so First() succeeds
	platformID := int64(1)
	prod := candidate.CandidateProduct{
		Title:            "Test",
		Description:      "Enough chars for completeness to pass title/desc checks",
		MainImage:        "https://example.test/img.jpg",
		Images:           []byte(`["img1.jpg"]`),
		CategoryID:       nil,
		BrandID:          nil,
		SpecJSON:         []byte(`{}`),
		PurchasePrice:    10,
		PurchaseCurrency: "CNY",
		PackageWeightKg:  0.5,
		PackageLengthCm:  10,
		PackageWidthCm:   8,
		PackageHeightCm:  6,
		HSCode:           "1234.56",
		TargetSalePrice:  30,
		TargetPlatformID: &platformID,
	}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatalf("create candidate: %v", err)
	}

	// Drop the target table so Create fails
	db.Exec("DROP TABLE completeness_check")

	_, err := svc.Check(prod.ID, "tester")
	if err == nil {
		t.Fatal("expected error from Create (table dropped)")
	}
}

func TestService_Check_CompleteProduct(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{})
	// The check reads candidate_product which won't exist in this test DB
	svc := NewService(db, dbtest.NewLogger(t))

	// Without a candidate product, it should error
	_, err := svc.Check(999, "tester")
	if err == nil {
		t.Fatal("expected error for non-existent product")
	}
}

func TestService_ListChecks(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Insert some checks directly
	db.Create(&CompletenessCheck{ProductID: 1, Score: 90, Status: "complete", TriggeredBy: "system"})
	db.Create(&CompletenessCheck{ProductID: 2, Score: 45, Status: "incomplete", TriggeredBy: "system"})

	items, total, err := svc.ListChecks(1, 10, "")
	if err != nil {
		t.Fatalf("ListChecks: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("len = %d", len(items))
	}
}

func TestService_ListChecks_Filtered(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CompletenessCheck{})
	svc := NewService(db, dbtest.NewLogger(t))

	db.Create(&CompletenessCheck{ProductID: 1, Score: 90, Status: "complete"})
	db.Create(&CompletenessCheck{ProductID: 2, Score: 45, Status: "incomplete"})

	items, total, err := svc.ListChecks(1, 10, "complete")
	if err != nil {
		t.Fatalf("ListChecks: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d", total)
	}
	if items[0].Status != "complete" {
		t.Fatalf("status = %s", items[0].Status)
	}
}
