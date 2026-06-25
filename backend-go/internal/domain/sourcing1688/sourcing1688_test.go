package sourcing1688

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestService_CreateAndGet(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	p, err := svc.Create(&CreateInput{
		SourceURL:    "https://detail.1688.com/offer/123.html",
		SupplierName: "某供应商",
		Price1688:    dbtest.FloatPtr(50.0),
		MinOrderQty:  dbtest.IntPtr(10),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.ID == 0 {
		t.Fatal("ID should be set")
	}
	if p.SourceURL != "https://detail.1688.com/offer/123.html" {
		t.Fatalf("SourceURL = %s", p.SourceURL)
	}
	if p.Status != "pending" {
		t.Fatalf("Status = %s, expected pending", p.Status)
	}

	got, err := svc.Get(p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.SupplierName != "某供应商" {
		t.Fatalf("SupplierName = %s", got.SupplierName)
	}
}

func TestService_Update(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	p, _ := svc.Create(&CreateInput{SourceURL: "https://detail.1688.com/offer/456.html"})
	updated, err := svc.Update(p.ID, &UpdateInput{
		SupplierName:  dbtest.StringPtr("新供应商"),
		Price1688:     dbtest.FloatPtr(60.0),
		Status:        dbtest.StringPtr("reviewed"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.SupplierName != "新供应商" {
		t.Fatalf("SupplierName = %s", updated.SupplierName)
	}
	if updated.Price1688 != 60.0 {
		t.Fatalf("Price1688 = %v", updated.Price1688)
	}
}

func TestService_List(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateInput{SourceURL: "url1", SupplierName: "A供应商"})
	svc.Create(&CreateInput{SourceURL: "url2", SupplierName: "B供应商", Status: "imported"})
	svc.Create(&CreateInput{SourceURL: "url3", SupplierName: "C供应商", Status: "rejected"})

	p := common.Pagination{Page: 1, Size: 10}

	items, total, err := svc.List(&p, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d", total)
	}
	_ = items

	items, total, err = svc.List(&p, &ListFilter{Status: "imported"})
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 imported, got %d", total)
	}
}

func TestService_Delete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	p, _ := svc.Create(&CreateInput{SourceURL: "url_del"})
	if err := svc.Delete(p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := svc.Get(p.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestService_Import(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	p, _ := svc.Create(&CreateInput{SourceURL: "url_import"})
	imported, err := svc.Import(p.ID, &ImportInput{ImportedBy: "admin"})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if imported.Status != "imported" {
		t.Fatalf("Status = %s", imported.Status)
	}
}

func TestService_Reject(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	p, _ := svc.Create(&CreateInput{SourceURL: "url_reject"})
	rejected, err := svc.Reject(p.ID, &RejectInput{RejectedBy: "admin", Reason: "价格过高"})
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if rejected.Status != "rejected" {
		t.Fatalf("Status = %s", rejected.Status)
	}
}

func TestService_Summary(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Sourcing1688Product{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateInput{SourceURL: "u1", Status: "pending"})
	svc.Create(&CreateInput{SourceURL: "u2", Status: "imported"})
	svc.Create(&CreateInput{SourceURL: "u3", Status: "pending"})

	summary, err := svc.Summary()
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if summary.Total != 3 {
		t.Fatalf("Total = %d", summary.Total)
	}
	if summary.ByStatus["pending"] != 2 {
		t.Fatalf("pending = %d", summary.ByStatus["pending"])
	}
}
