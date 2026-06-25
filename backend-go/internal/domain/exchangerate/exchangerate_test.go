package exchangerate

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestService_CreateAndGetLatest(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExchangeRate{})
	svc := NewService(db, dbtest.NewLogger(t))

	er, err := svc.Create(&CreateInput{
		FromCurrency:  "USD",
		ToCurrency:    "CNY",
		Rate:          7.25,
		EffectiveDate: "2026-06-01",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if er.ID == 0 {
		t.Fatal("ID should be set")
	}
	if er.FromCurrency != "USD" {
		t.Fatalf("FromCurrency = %s", er.FromCurrency)
	}
	if er.Rate != 7.25 {
		t.Fatalf("Rate = %v", er.Rate)
	}

	latest, err := svc.GetLatest("USD", "CNY")
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if latest.Rate != 7.25 {
		t.Fatalf("Rate = %v", latest.Rate)
	}
}

func TestService_UpdateByPair(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExchangeRate{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateInput{
		FromCurrency:  "USD",
		ToCurrency:    "CNY",
		Rate:          7.25,
		EffectiveDate: "2026-06-01",
	})

	updated, err := svc.UpdateByPair("USD", "CNY", &UpdateInput{
		Rate:          7.30,
		EffectiveDate: "2026-06-15",
	})
	if err != nil {
		t.Fatalf("UpdateByPair: %v", err)
	}
	if updated.Rate != 7.30 {
		t.Fatalf("Rate = %v", updated.Rate)
	}

	latest, _ := svc.GetLatest("USD", "CNY")
	if latest.Rate != 7.30 {
		t.Fatalf("Rate after GetLatest = %v", latest.Rate)
	}
}

func TestService_List(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExchangeRate{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateInput{FromCurrency: "USD", ToCurrency: "CNY", Rate: 7.25, EffectiveDate: "2026-06-01"})
	svc.Create(&CreateInput{FromCurrency: "EUR", ToCurrency: "CNY", Rate: 7.80, EffectiveDate: "2026-06-01"})
	svc.Create(&CreateInput{FromCurrency: "USD", ToCurrency: "JPY", Rate: 145.0, EffectiveDate: "2026-06-01"})

	p := common.Pagination{Page: 1, Size: 10}

	items, total, err := svc.List(&p, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d", total)
	}
	_ = items

	// Filter
	items, total, err = svc.List(&p, &ListFilter{FromCurrency: "USD"})
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 USD rates, got %d", total)
	}
}

func TestService_Delete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExchangeRate{})
	svc := NewService(db, dbtest.NewLogger(t))

	er, _ := svc.Create(&CreateInput{
		FromCurrency:  "USD",
		ToCurrency:    "CNY",
		Rate:          7.25,
		EffectiveDate: "2026-06-01",
	})
	if err := svc.Delete(er.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := svc.GetLatest("USD", "CNY")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestService_GetLatest_NotFound(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExchangeRate{})
	svc := NewService(db, dbtest.NewLogger(t))

	_, err := svc.GetLatest("XYZ", "ABC")
	if err == nil {
		t.Fatal("expected error for non-existent pair")
	}
}

func TestService_UpdateByPair_NotFound(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExchangeRate{})
	svc := NewService(db, dbtest.NewLogger(t))

	_, err := svc.UpdateByPair("USD", "CNY", &UpdateInput{Rate: 7.0})
	if err == nil {
		t.Fatal("expected error for non-existent pair")
	}
}
