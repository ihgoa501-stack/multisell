package reliability

import (
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/costcontrol"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

// setupDB creates a test DB with the llm_budgets table auto-migrated.
func setupDB(t testing.TB) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &LLMBudget{}, &costcontrol.CostLog{})
}

func TestService_GetBudget_Default(t *testing.T) {
	t.Parallel()
	db := setupDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	// Auto-creates table via Create
	b, err := svc.GetBudget()
	if err != nil {
		t.Fatalf("GetBudget: %v", err)
	}
	if b.MonthlyLimitUSD != 0 {
		t.Fatalf("MonthlyLimitUSD = %v (expected 0)", b.MonthlyLimitUSD)
	}
	if b.CurrentMonthUSD != 0 {
		t.Fatalf("CurrentMonthUSD = %v (expected 0)", b.CurrentMonthUSD)
	}
	if b.IsPaused {
		t.Fatalf("IsPaused = true (expected false)")
	}
}

func TestService_SetBudget(t *testing.T) {
	t.Parallel()
	db := setupDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	b, err := svc.SetBudget(100)
	if err != nil {
		t.Fatalf("SetBudget: %v", err)
	}
	r := InBudgetResponse(b)
	if r.MonthlyLimitUSD != 100 {
		t.Fatalf("MonthlyLimitUSD = %v (expected 100)", b.MonthlyLimitUSD)
	}
	if r.IsPaused {
		t.Fatalf("IsPaused = true (expected false)")
	}
	if r.RemainingUSD != 100 {
		t.Fatalf("RemainingUSD = %v (expected 100)", r.RemainingUSD)
	}
}

func TestService_CheckBudget_OK(t *testing.T) {
	t.Parallel()
	db := setupDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	_, err := svc.SetBudget(100)
	if err != nil {
		t.Fatalf("SetBudget: %v", err)
	}
	ok, err := svc.CheckBudget()
	if err != nil {
		t.Fatalf("CheckBudget: %v", err)
	}
	if !ok {
		t.Fatal("CheckBudget = false (expected true)")
	}
}

func TestService_CheckBudget_Exceeded(t *testing.T) {
	t.Parallel()
	db := setupDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	_, err := svc.SetBudget(50)
	if err != nil {
		t.Fatalf("SetBudget: %v", err)
	}
	// Record spend over the limit
	if err := svc.RecordSpend(60); err != nil {
		t.Fatalf("RecordSpend: %v", err)
	}
	ok, err := svc.CheckBudget()
	if err != ErrBudgetExceeded {
		t.Fatalf("CheckBudget err = %v (expected ErrBudgetExceeded)", err)
	}
	if ok {
		t.Fatal("CheckBudget = true (expected false)")
	}
}

func TestService_CheckBudget_ExceededByCostLogs(t *testing.T) {
	t.Parallel()
	db := setupDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	_, err := svc.SetBudget(50)
	if err != nil {
		t.Fatalf("SetBudget: %v", err)
	}
	if err := db.Create(&costcontrol.CostLog{
		AgentID:    "A1",
		Model:      "claude-sonnet",
		CostUSD:    60,
		WindowDate: time.Now(),
	}).Error; err != nil {
		t.Fatalf("create cost log: %v", err)
	}
	ok, err := svc.CheckBudget()
	if err != ErrBudgetExceeded {
		t.Fatalf("CheckBudget err = %v (expected ErrBudgetExceeded)", err)
	}
	if ok {
		t.Fatal("CheckBudget = true (expected false)")
	}
	b, err := svc.GetBudget()
	if err != nil {
		t.Fatalf("GetBudget: %v", err)
	}
	if b.CurrentMonthUSD != 60 {
		t.Fatalf("CurrentMonthUSD = %v (expected 60)", b.CurrentMonthUSD)
	}
}

func TestService_RecordSpend_AutoPause(t *testing.T) {
	t.Parallel()
	db := setupDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	_, err := svc.SetBudget(100)
	if err != nil {
		t.Fatalf("SetBudget: %v", err)
	}
	// Record spend that reaches the limit
	if err := svc.RecordSpend(100); err != nil {
		t.Fatalf("RecordSpend: %v", err)
	}
	b, err := svc.GetBudget()
	if err != nil {
		t.Fatalf("GetBudget: %v", err)
	}
	if !b.IsPaused {
		t.Fatal("IsPaused = false (expected true after hitting limit)")
	}
	if b.CurrentMonthUSD != 100 {
		t.Fatalf("CurrentMonthUSD = %v (expected 100)", b.CurrentMonthUSD)
	}
}

func TestInBudgetResponse(t *testing.T) {
	t.Parallel()
	b := &LLMBudget{
		MonthlyLimitUSD: 200,
		CurrentMonthUSD: 50,
		BudgetMonth:     "2026-07",
	}
	r := InBudgetResponse(b)
	if r.MonthlyLimitUSD != 200 {
		t.Fatalf("MonthlyLimitUSD = %v", r.MonthlyLimitUSD)
	}
	if r.RemainingUSD != 150 {
		t.Fatalf("RemainingUSD = %v (expected 150)", r.RemainingUSD)
	}
	if r.CurrentMonthUSD != 50 {
		t.Fatalf("CurrentMonthUSD = %v", r.CurrentMonthUSD)
	}
}
