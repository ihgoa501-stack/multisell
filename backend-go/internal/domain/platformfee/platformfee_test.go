package platformfee

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func int64Ptr(v int64) *int64 { return &v }

func TestService_CreateAndGet(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PlatformFeeRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	r, err := svc.Create(&CreateRuleInput{
		FeeType:     "commission",
		FeeRatePct:  dbtest.FloatPtr(5.0),
		CountryCode: "CN",
		Currency:    "CNY",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if r.ID == 0 {
		t.Fatal("ID should be set")
	}
	if r.FeeRatePct != 5.0 {
		t.Fatalf("FeeRatePct = %v", r.FeeRatePct)
	}

	got, err := svc.Get(r.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.FeeType != "commission" {
		t.Fatalf("FeeType = %s", got.FeeType)
	}
}

func TestService_Update(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PlatformFeeRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	r, _ := svc.Create(&CreateRuleInput{FeeType: "fixed", FixedAmount: dbtest.FloatPtr(10)})
	updated, err := svc.Update(r.ID, &UpdateRuleInput{
		FixedAmount: dbtest.FloatPtr(20),
		Status:      dbtest.StringPtr("inactive"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.FixedAmount != 20 {
		t.Fatalf("FixedAmount = %v", updated.FixedAmount)
	}
	if updated.Status != "inactive" {
		t.Fatalf("Status = %s", updated.Status)
	}
}

func TestService_List_Delete(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PlatformFeeRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateRuleInput{FeeType: "commission", FeeRatePct: dbtest.FloatPtr(3)})
	svc.Create(&CreateRuleInput{FeeType: "fixed", FixedAmount: dbtest.FloatPtr(5)})
	svc.Create(&CreateRuleInput{FeeType: "commission", FeeRatePct: dbtest.FloatPtr(5), Status: "inactive"})

	p := common.Pagination{Page: 1, Size: 10}

	items, total, err := svc.List(&p, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d", total)
	}
	_ = items

	// Filter by fee_type
	items, total, err = svc.List(&p, &RuleListFilter{FeeType: "fixed"})
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 fixed, got %d", total)
	}

	// Delete
	if err := svc.Delete(items[0].ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = svc.Get(items[0].ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestService_Calculate_Commission(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PlatformFeeRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateRuleInput{
		FeeType:     "commission",
		FeeRatePct:  dbtest.FloatPtr(5.0),
		CountryCode: "CN",
		PlatformID:  int64Ptr(1),
	})

	res, err := svc.Calculate(&CalculateRequest{
		PlatformID:  1,
		CountryCode: "CN",
		Amount:      1000,
	})
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}
	if !res.Matched {
		t.Fatal("expected match")
	}
	if res.CalculatedFee != 50.0 {
		t.Fatalf("CalculatedFee = %v (expected 50)", res.CalculatedFee)
	}
}

func TestService_Calculate_FixedWithMinMax(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PlatformFeeRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateRuleInput{
		FeeType:     "fixed",
		FixedAmount: dbtest.FloatPtr(30),
		MinAmount:   dbtest.FloatPtr(10),
		MaxAmount:   dbtest.FloatPtr(100),
		PlatformID:  int64Ptr(1),
	})

	res, err := svc.Calculate(&CalculateRequest{PlatformID: 1, Amount: 500})
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}
	if !res.Matched {
		t.Fatal("expected match")
	}
	if res.CalculatedFee != 30 {
		t.Fatalf("CalculatedFee = %v (expected 30)", res.CalculatedFee)
	}
}

func TestService_Calculate_NoMatch(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PlatformFeeRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	res, err := svc.Calculate(&CalculateRequest{PlatformID: 999, Amount: 100})
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}
	if res.Matched {
		t.Fatal("expected no match")
	}
}

func TestService_Calculate_OtherFeeType(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PlatformFeeRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateRuleInput{
		FeeType:     "other",
		FixedAmount: dbtest.FloatPtr(10),
		FeeRatePct:  dbtest.FloatPtr(2.0),
		PlatformID:  int64Ptr(1),
	})

	res, err := svc.Calculate(&CalculateRequest{PlatformID: 1, Amount: 200})
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}
	// other = fixed + amount*rate/100 = 10 + 200*2/100 = 14
	if res.CalculatedFee != 14.0 {
		t.Fatalf("CalculatedFee = %v (expected 14)", res.CalculatedFee)
	}
}

func TestService_Calculate_StorageFeeType(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &PlatformFeeRule{})
	svc := NewService(db, dbtest.NewLogger(t))

	svc.Create(&CreateRuleInput{
		FeeType:     "storage",
		FixedAmount: dbtest.FloatPtr(50),
		PlatformID:  int64Ptr(1),
	})

	res, err := svc.Calculate(&CalculateRequest{PlatformID: 1, Amount: 1000})
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}
	if res.CalculatedFee != 50 {
		t.Fatalf("CalculatedFee = %v (expected 50)", res.CalculatedFee)
	}
}
