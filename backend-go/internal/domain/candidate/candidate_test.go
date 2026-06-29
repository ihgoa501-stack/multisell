package candidate

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestService_CreateAndGet(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	price := 50.0
	weight := 0.5
	salePrice := 120.0

	c, err := svc.Create(&CreateCandidateInput{
		Title:             "Test Product",
		Description:       "A test product description",
		PurchasePrice:     &price,
		PackageWeightKg:   &weight,
		TargetSalePrice:   &salePrice,
		DestinationCountry: "US",
		CreatedBy:         "tester",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.ID == 0 {
		t.Fatal("ID should be set")
	}
	if c.Title != "Test Product" {
		t.Fatalf("Title = %s", c.Title)
	}
	if c.Status != "draft" {
		t.Fatalf("Status = %s", c.Status)
	}

	got, err := svc.GetByID(c.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != c.ID {
		t.Fatal("ID mismatch")
	}
}

func TestService_List(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	price := 50.0
	svc.Create(&CreateCandidateInput{Title: "Prod A", PurchasePrice: &price, CreatedBy: "tester"})
	svc.Create(&CreateCandidateInput{Title: "Prod B", PurchasePrice: &price, CreatedBy: "tester"})
	svc.Create(&CreateCandidateInput{Title: "Prod C", PurchasePrice: &price, CreatedBy: "tester"})

	p := common.Pagination{Page: 1, Size: 10}
	items, total, err := svc.List(&p, "", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d", total)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d", len(items))
	}
}

func TestService_Update(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	price := 50.0
	c, _ := svc.Create(&CreateCandidateInput{Title: "Original", PurchasePrice: &price, CreatedBy: "tester"})

	newTitle := "Updated"
	updated, err := svc.Update(c.ID, &UpdateCandidateInput{Title: &newTitle})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != "Updated" {
		t.Fatalf("Title = %s", updated.Title)
	}
}

func TestService_Delete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	price := 50.0
	c, _ := svc.Create(&CreateCandidateInput{Title: "ToDelete", PurchasePrice: &price, CreatedBy: "tester"})

	if err := svc.Delete(c.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := svc.GetByID(c.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestService_Count(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CandidateProduct{})
	svc := NewService(db, dbtest.NewLogger(t))

	price := 50.0
	svc.Create(&CreateCandidateInput{Title: "A", PurchasePrice: &price, CreatedBy: "tester"})
	svc.Create(&CreateCandidateInput{Title: "B", PurchasePrice: &price, CreatedBy: "tester"})

	total, err := svc.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d", total)
	}
}
