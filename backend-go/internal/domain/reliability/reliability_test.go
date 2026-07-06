package reliability

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func newTestSvc(t *testing.T) *Service {
	t.Helper()
	db := dbtest.NewDB(t, &LLMBudget{})
	return NewService(db, zap.NewNop())
}

func TestBudgetDefault(t *testing.T) {
	svc := newTestSvc(t)
	b, err := svc.GetBudget()
	if err != nil {
		t.Fatalf("GetBudget: %v", err)
	}
	if b.MonthlyLimitUSD != 200 {
		t.Fatalf("expected default limit 200, got %.2f", b.MonthlyLimitUSD)
	}
	if b.IsPaused {
		t.Fatal("expected default is_paused=false")
	}
}

func TestBudgetSet(t *testing.T) {
	svc := newTestSvc(t)
	if err := svc.SetBudget(500); err != nil {
		t.Fatalf("SetBudget: %v", err)
	}
	b, err := svc.GetBudget()
	if err != nil {
		t.Fatalf("GetBudget: %v", err)
	}
	if b.MonthlyLimitUSD != 500 {
		t.Fatalf("expected limit 500, got %.2f", b.MonthlyLimitUSD)
	}
}

func TestBudgetCheck(t *testing.T) {
	svc := newTestSvc(t)
	ok, err := svc.CheckBudget()
	if err != nil {
		t.Fatalf("CheckBudget: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true for fresh budget")
	}
}

func TestBudgetExceeded(t *testing.T) {
	svc := newTestSvc(t)
	if err := svc.RecordSpend(9999); err != nil {
		t.Fatalf("RecordSpend: %v", err)
	}
	ok, err := svc.CheckBudget()
	if err != nil {
		t.Fatalf("CheckBudget: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false after exceeding budget")
	}
	b, err := svc.GetBudget()
	if err != nil {
		t.Fatalf("GetBudget: %v", err)
	}
	if !b.IsPaused {
		t.Fatal("expected is_paused=true after exceeding budget")
	}
}
