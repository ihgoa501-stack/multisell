package exchangerate

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &ExchangeRate{})
}

func newService(t *testing.T) *Service {
	t.Helper()
	return NewService(newTestDB(t), dbtest.NewLogger(t))
}

func seedRate(t *testing.T, svc *Service, from, to string, rate float64, date string) *ExchangeRate {
	t.Helper()
	r, err := svc.Create(&CreateInput{
		FromCurrency:  from,
		ToCurrency:    to,
		Rate:          rate,
		EffectiveDate: date,
	})
	if err != nil {
		t.Fatalf("seedRate failed: %v", err)
	}
	return r
}

func TestExchangeRate_Create(t *testing.T) {
	svc := newService(t)

	r, err := svc.Create(&CreateInput{
		FromCurrency:  "USD",
		ToCurrency:    "CNY",
		Rate:          7.25,
		EffectiveDate: "2025-06-01",
		Source:        "ecb",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if r.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if r.FromCurrency != "USD" {
		t.Fatalf("FromCurrency = %q, want USD", r.FromCurrency)
	}
	if r.ToCurrency != "CNY" {
		t.Fatalf("ToCurrency = %q, want CNY", r.ToCurrency)
	}
}

func TestExchangeRate_Create_DefaultSource(t *testing.T) {
	svc := newService(t)

	r, err := svc.Create(&CreateInput{
		FromCurrency:  "EUR",
		ToCurrency:    "GBP",
		Rate:          0.85,
		EffectiveDate: "2025-06-01",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if r.Source != "manual" {
		t.Fatalf("Source = %q, want manual", r.Source)
	}
}

func TestExchangeRate_Create_InvalidDate(t *testing.T) {
	svc := newService(t)

	_, err := svc.Create(&CreateInput{
		FromCurrency:  "USD",
		ToCurrency:    "CNY",
		Rate:          7.25,
		EffectiveDate: "not-a-date",
	})
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}

func TestExchangeRate_GetLatest(t *testing.T) {
	svc := newService(t)

	seedRate(t, svc, "USD", "CNY", 7.20, "2025-05-01")
	seedRate(t, svc, "USD", "CNY", 7.25, "2025-06-01")
	seedRate(t, svc, "USD", "CNY", 7.10, "2025-04-01")

	got, err := svc.GetLatest("usd", "cny")
	if err != nil {
		t.Fatalf("GetLatest failed: %v", err)
	}
	if got.Rate != 7.25 {
		t.Fatalf("GetLatest.Rate = %v, want 7.25", got.Rate)
	}
}

func TestExchangeRate_GetLatest_NotFound(t *testing.T) {
	svc := newService(t)

	_, err := svc.GetLatest("XYZ", "ABC")
	if err == nil {
		t.Fatal("expected error for non-existent pair")
	}
}

func TestExchangeRate_UpdateByPair(t *testing.T) {
	svc := newService(t)

	seedRate(t, svc, "USD", "JPY", 150.0, "2025-06-01")

	updated, err := svc.UpdateByPair("USD", "JPY", &UpdateInput{Rate: 152.5})
	if err != nil {
		t.Fatalf("UpdateByPair failed: %v", err)
	}
	if updated.Rate != 152.5 {
		t.Fatalf("Rate after update = %v, want 152.5", updated.Rate)
	}
}

func TestExchangeRate_UpdateByPair_WithDate(t *testing.T) {
	svc := newService(t)

	seedRate(t, svc, "GBP", "USD", 1.27, "2025-06-01")

	updated, err := svc.UpdateByPair("GBP", "USD", &UpdateInput{
		Rate:          1.30,
		EffectiveDate: "2025-07-01",
	})
	if err != nil {
		t.Fatalf("UpdateByPair failed: %v", err)
	}
	if updated.Rate != 1.30 {
		t.Fatalf("Rate = %v, want 1.30", updated.Rate)
	}
	if updated.EffectiveDate.Year() != 2025 || updated.EffectiveDate.Month() != 7 {
		t.Fatalf("EffectiveDate = %v, want 2025-07-01", updated.EffectiveDate)
	}
}

func TestExchangeRate_UpdateByPair_NotFound(t *testing.T) {
	svc := newService(t)

	_, err := svc.UpdateByPair("ZZZ", "YYY", &UpdateInput{Rate: 1.0})
	if err == nil {
		t.Fatal("expected error for non-existent pair")
	}
}

func TestExchangeRate_Delete(t *testing.T) {
	svc := newService(t)

	r := seedRate(t, svc, "AUD", "USD", 0.65, "2025-06-01")

	if err := svc.Delete(r.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := svc.GetLatest("AUD", "USD")
	if err == nil {
		t.Fatal("expected error after Delete")
	}
}

func TestExchangeRate_Delete_NotFound(t *testing.T) {
	svc := newService(t)

	if err := svc.Delete(999); err != nil {
		t.Fatalf("Delete for non-existent ID should succeed: %v", err)
	}
}

func TestExchangeRate_List_Pagination(t *testing.T) {
	svc := newService(t)

	for i := 0; i < 25; i++ {
		seedRate(t, svc, "USD", "CNY", float64(i), "2025-01-01")
	}

	p := &common.Pagination{Page: 1, Size: 10}
	items, total, err := svc.List(p, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 25 {
		t.Fatalf("List total = %d, want 25", total)
	}
	if len(items) != 10 {
		t.Fatalf("List returned %d items, want 10", len(items))
	}
}

func TestExchangeRate_List_FilterByCurrency(t *testing.T) {
	svc := newService(t)

	seedRate(t, svc, "USD", "CNY", 7.25, "2025-06-01")
	seedRate(t, svc, "EUR", "GBP", 0.85, "2025-06-01")
	seedRate(t, svc, "USD", "JPY", 150.0, "2025-06-01")

	p := &common.Pagination{Page: 1, Size: 20}
	f := &ListFilter{FromCurrency: "USD"}
	items, total, err := svc.List(p, f)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("List total = %d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("List returned %d items, want 2", len(items))
	}
}
