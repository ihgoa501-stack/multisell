package operationlog

import (
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func int64Ptr(v int64) *int64 { return &v }

func TestService_CRUD(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OperationLog{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create via Log convenience helper
	err := svc.Log("order", "create", "123", "admin", "创建订单 #123")
	if err != nil {
		t.Fatalf("Log: %v", err)
	}

	// Create directly
	err = svc.Create(&OperationLog{
		Module:     "product",
		Action:     "update",
		ResourceID: "456",
		Operator:   "admin",
		Content:    "更新商品价格",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// List
	items, total, err := svc.List(ListFilter{}, 1, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d", total)
	}
	_ = items

	// GetByID
	got, err := svc.GetByID(1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Module != "order" {
		t.Fatalf("Module = %s", got.Module)
	}

	// Filter by module
	items, total, err = svc.List(ListFilter{Module: "product"}, 1, 10)
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if total != 1 {
		t.Fatalf("filtered total = %d", total)
	}

	// Filter by time range
	from := time.Now().Add(-time.Hour)
	to := time.Now().Add(time.Hour)
	items, total, err = svc.List(ListFilter{From: from, To: to}, 1, 10)
	if err != nil {
		t.Fatalf("List time filtered: %v", err)
	}
	if total != 2 {
		t.Fatalf("time filtered total = %d (expected 2)", total)
	}
}

func TestService_LogStructured(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OperationLog{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create a structured log with all fields
	err := svc.LogStructured(&StructuredLogInput{
		Module:            "listing_task",
		Action:            "listing_task.execute",
		ResourceID:        "42",
		Operator:          "system",
		Content:           "listing_task_id=42 product_id=1 platform_id=10",
		Result:            "success",
		TriggerType:       "system",
		AgentSuggestionID: int64Ptr(7),
		ApprovalID:        int64Ptr(99),
		EntityType:        "listing_task",
		EntityID:          42,
	})
	if err != nil {
		t.Fatalf("LogStructured: %v", err)
	}

	// Verify it was persisted
	var all []OperationLog
	db.Find(&all)
	if len(all) != 1 {
		t.Fatalf("expected 1 log, got %d", len(all))
	}
	l := all[0]
	if l.TriggerType != "system" {
		t.Errorf("TriggerType = %q", l.TriggerType)
	}
	if l.EntityType != "listing_task" {
		t.Errorf("EntityType = %q", l.EntityType)
	}
	if l.EntityID != 42 {
		t.Errorf("EntityID = %d", l.EntityID)
	}
	if l.ApprovalID == nil || *l.ApprovalID != 99 {
		t.Errorf("ApprovalID = %v", l.ApprovalID)
	}
	if l.AgentSuggestionID == nil || *l.AgentSuggestionID != 7 {
		t.Errorf("AgentSuggestionID = %v", l.AgentSuggestionID)
	}
	if l.Result != "success" {
		t.Errorf("Result = %q", l.Result)
	}
}

func TestService_LogStructured_Minimal(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OperationLog{})
	svc := NewService(db, dbtest.NewLogger(t))

	// LogStructured with only required fields (optional fields omitted)
	err := svc.LogStructured(&StructuredLogInput{
		Module:     "approval",
		Action:     "approval.review",
		ResourceID: "10",
		Operator:   "owner",
		Content:    "approved listing task",
		Result:     "approved",
	})
	if err != nil {
		t.Fatalf("LogStructured minimal: %v", err)
	}

	var all []OperationLog
	db.Find(&all)
	if len(all) != 1 {
		t.Fatalf("expected 1 log, got %d", len(all))
	}
	l := all[0]
	if l.TriggerType != "" {
		t.Errorf("expected empty TriggerType, got %q", l.TriggerType)
	}
	if l.ApprovalID != nil {
		t.Errorf("expected nil ApprovalID, got %v", l.ApprovalID)
	}
}

func TestService_LogStructured_AgentActionAudit(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OperationLog{})
	svc := NewService(db, dbtest.NewLogger(t))

	err := svc.LogStructured(&StructuredLogInput{
		Module:      "agent_action",
		Action:      "price_update",
		ResourceID:  "SKU001",
		Operator:    "A6",
		Content:     "risk_level=high mode=production suggested_price=29.99",
		Result:      "pending_approval",
		TriggerType: "agent",
		ApprovalID:  int64Ptr(101),
		EntityType:  "sku",
		EntityID:    42,
	})
	if err != nil {
		t.Fatalf("LogStructured agent action: %v", err)
	}

	var all []OperationLog
	db.Find(&all)
	if len(all) != 1 {
		t.Fatalf("expected 1 log, got %d", len(all))
	}
	l := all[0]
	if l.Module != "agent_action" {
		t.Errorf("Module = %q", l.Module)
	}
	if l.Action != "price_update" {
		t.Errorf("Action = %q", l.Action)
	}
	if l.Operator != "A6" {
		t.Errorf("Operator = %q", l.Operator)
	}
	if l.TriggerType != "agent" {
		t.Errorf("TriggerType = %q", l.TriggerType)
	}
	if l.Result != "pending_approval" {
		t.Errorf("Result = %q", l.Result)
	}
	if l.ApprovalID == nil || *l.ApprovalID != 101 {
		t.Errorf("ApprovalID = %v", l.ApprovalID)
	}
}
