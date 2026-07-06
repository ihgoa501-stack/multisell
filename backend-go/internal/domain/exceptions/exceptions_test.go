package exceptions

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
)

func TestService_CRUD(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExceptionItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create
	e := &ExceptionItem{
		SourceModule: "order",
		Severity:     "high",
		Title:        "订单支付异常",
		Description:  "订单 #12345 支付超时",
		Status:       "open",
	}
	err := svc.Create(e)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if e.ID == 0 {
		t.Fatal("ID should be set")
	}

	// GetByID
	got, err := svc.GetByID(e.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title != "订单支付异常" {
		t.Fatalf("Title = %s", got.Title)
	}

	// List
	items, total, err := svc.List(ListFilter{}, 1, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d", total)
	}
	_ = items

	// Filter by severity
	items, total, err = svc.List(ListFilter{Severity: "high"}, 1, 10)
	if err != nil {
		t.Fatalf("List filtered: %v", err)
	}
	if total != 1 {
		t.Fatalf("filtered total = %d", total)
	}

	// Assign
	assigned, err := svc.Assign(e.ID, "user_zhang")
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if assigned.AssignedTo != "user_zhang" {
		t.Fatalf("AssignedTo = %s", assigned.AssignedTo)
	}
	if assigned.Status != "assigned" {
		t.Fatalf("Status = %s", assigned.Status)
	}

	// Resolve
	resolved, err := svc.Resolve(e.ID, "user_li", "已处理")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Status != "resolved" {
		t.Fatalf("Status = %s", resolved.Status)
	}
	if resolved.ResolvedBy != "user_li" {
		t.Fatalf("ResolvedBy = %s", resolved.ResolvedBy)
	}

	// Delete
	if err := svc.Delete(e.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = svc.GetByID(e.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestService_AutoDetect(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExceptionItem{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Create minimal related tables for auto-detection queries.
	db.Exec(`CREATE TABLE order_profit_record (id INTEGER PRIMARY KEY, order_id INTEGER UNIQUE, profit REAL)`)
	db.Exec(`CREATE TABLE inventory (id INTEGER PRIMARY KEY, sku_id INTEGER, quantity INTEGER, locked_quantity INTEGER DEFAULT 0, safety_stock INTEGER DEFAULT 0)`)
	db.Exec(`CREATE TABLE fulfillment_tracking (id INTEGER PRIMARY KEY, order_id INTEGER, is_lost INTEGER DEFAULT 0, is_returned INTEGER DEFAULT 0, is_damaged INTEGER DEFAULT 0)`)
	db.Exec(`CREATE TABLE sales_order (id INTEGER PRIMARY KEY, platform_id INTEGER, shipping_fee REAL DEFAULT 0, platform_fee REAL DEFAULT 0, pay_amount REAL DEFAULT 0, shipped_at TIMESTAMP, delivered_at TIMESTAMP)`)
	db.Exec(`CREATE TABLE platform_fee_rule (id INTEGER PRIMARY KEY, platform_id INTEGER, fee_type TEXT, fee_rate_pct REAL, status TEXT DEFAULT 'active', priority INTEGER DEFAULT 0)`)

	ctx := context.Background()

	// Empty data -- should not error and return no items.
	items, err := svc.AutoDetect(ctx)
	if err != nil {
		t.Fatalf("AutoDetect on empty data: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items on empty data, got %d", len(items))
	}

	// Seed a loss order.
	db.Exec(`INSERT INTO order_profit_record (id, order_id, profit) VALUES (1, 101, -50.0)`)

	items, err = svc.AutoDetect(ctx)
	if err != nil {
		t.Fatalf("AutoDetect: %v", err)
	}

	found := false
	for _, item := range items {
		if item.SourceType == TypeLossOrder && item.SourceID != nil && *item.SourceID == 101 {
			found = true
			if item.Severity != "high" {
				t.Errorf("loss order severity = %s, want high", item.Severity)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected loss order exception for order 101")
	}

	// Duplicate avoidance: second detection should not create another exception for order 101.
	items, err = svc.AutoDetect(ctx)
	if err != nil {
		t.Fatalf("AutoDetect (2nd call): %v", err)
	}
	dupCount := 0
	for _, item := range items {
		if item.SourceType == TypeLossOrder && item.SourceID != nil && *item.SourceID == 101 {
			dupCount++
		}
	}
	if dupCount > 0 {
		t.Fatal("AutoDetect created duplicate exception for order 101")
	}
}

func TestService_Suggest(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExceptionItem{})
	logger := dbtest.NewLogger(t)

	// Without LLM provider — falls back to rule-based defaults
	svc := NewService(db, logger)
	e := &ExceptionItem{
		SourceModule: "order",
		SourceType:   TypeLossOrder,
		Severity:     "high",
		Title:        "亏损订单",
		Description:  "订单 #12345 亏损 -50.00",
		Status:       "open",
	}
	if err := svc.Create(e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	sug, err := svc.Suggest(context.Background(), e.ID)
	if err != nil {
		t.Fatalf("Suggest without LLM: %v", err)
	}
	if sug.ExceptionID != e.ID {
		t.Fatalf("ExceptionID = %d, want %d", sug.ExceptionID, e.ID)
	}
	if sug.SuggestionText == "" {
		t.Fatal("SuggestionText should not be empty (fallback)")
	}
	if sug.SuggestedAction != "adjust_price" {
		t.Fatalf("SuggestedAction = %q, want adjust_price", sug.SuggestedAction)
	}
	if sug.RiskLevel != "high" {
		t.Fatalf("RiskLevel = %q, want high", sug.RiskLevel)
	}
	if sug.AutoExecutable {
		t.Fatal("AutoExecutable should be false for high risk")
	}

	// With stub LLM provider — stub returns keyword-matched text
	llm := ai.NewLLMProvider(logger)
	svc = svc.WithLLM(llm)

	sug2, err := svc.Suggest(context.Background(), e.ID)
	if err != nil {
		t.Fatalf("Suggest with LLM: %v", err)
	}
	if sug2.ExceptionID != e.ID {
		t.Fatalf("ExceptionID = %d, want %d", sug2.ExceptionID, e.ID)
	}
	if sug2.SuggestionText == "" {
		t.Fatal("SuggestionText should not be empty")
	}
	if sug2.SuggestedAction == "" {
		t.Fatal("SuggestedAction should not be empty")
	}
	if sug2.RiskLevel == "" {
		t.Fatal("RiskLevel should not be empty")
	}

	// Unknown exception ID
	_, err = svc.Suggest(context.Background(), 99999)
	if err == nil {
		t.Fatal("expected error for unknown exception")
	}
}

func TestService_ResolveWithApproval(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExceptionItem{}, &operationlog.OperationLog{}, &approval.ApprovalRequest{})
	logger := dbtest.NewLogger(t)

	oplogSvc := operationlog.NewService(db, logger)
	approvalSvc := approval.NewService(db, logger, oplogSvc)

	svc := NewService(db, logger).
		WithOperationLog(oplogSvc).
		WithApproval(approvalSvc)

	// Low severity exception — resolves directly, no approval request created
	e := &ExceptionItem{
		SourceModule: "inventory",
		SourceType:   TypeOutOfStock,
		Severity:     "low",
		Title:        "库存偏低",
		Description:  "SKU #100 库存较低",
		Status:       "open",
	}
	if err := svc.Create(e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	resolved, err := svc.ResolveWithApproval(context.Background(), e.ID, "owner_zhang", "建议补货", "restock")
	if err != nil {
		t.Fatalf("ResolveWithApproval low risk: %v", err)
	}
	if resolved.Status != "resolved" {
		t.Fatalf("Status = %s, want resolved", resolved.Status)
	}

	var lowApprovalCount int64
	db.Model(&approval.ApprovalRequest{}).Where("target_type = ? AND target_id = ?", "exception", e.ID).Count(&lowApprovalCount)
	if lowApprovalCount != 0 {
		t.Fatalf("expected 0 approval requests for low risk, got %d", lowApprovalCount)
	}

	// High severity exception — resolve creates an approval request
	e2 := &ExceptionItem{
		SourceModule: "order",
		SourceType:   TypeLossOrder,
		Severity:     "high",
		Title:        "高亏损订单",
		Description:  "订单 #99999 亏损 -5000.00",
		Status:       "open",
	}
	if err := svc.Create(e2); err != nil {
		t.Fatalf("Create e2: %v", err)
	}

	resolved2, err := svc.ResolveWithApproval(context.Background(), e2.ID, "owner_li", "建议取消订单", "cancel_order")
	if err != nil {
		t.Fatalf("ResolveWithApproval high risk: %v", err)
	}
	if resolved2.Status != "resolved" {
		t.Fatalf("Status = %s, want resolved", resolved2.Status)
	}
	if resolved2.ResolvedBy != "owner_li" {
		t.Fatalf("ResolvedBy = %s, want owner_li", resolved2.ResolvedBy)
	}

	var highApprovalCount int64
	db.Model(&approval.ApprovalRequest{}).Where("target_type = ? AND target_id = ?", "exception", e2.ID).Count(&highApprovalCount)
	if highApprovalCount != 1 {
		t.Fatalf("expected 1 approval request for high risk, got %d", highApprovalCount)
	}
}

func TestService_ResolveWithApproval_NoServices(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ExceptionItem{})
	logger := dbtest.NewLogger(t)

	// No operation log or approval service configured — should still work
	svc := NewService(db, logger)

	e := &ExceptionItem{
		SourceModule: "order",
		SourceType:   TypeLossOrder,
		Severity:     "high",
		Title:        "亏损订单",
		Description:  "订单 #12345 亏损 -50.00",
		Status:       "open",
	}
	if err := svc.Create(e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	resolved, err := svc.ResolveWithApproval(context.Background(), e.ID, "owner_wang", "建议调价", "adjust_price")
	if err != nil {
		t.Fatalf("ResolveWithApproval no services: %v", err)
	}
	if resolved.Status != "resolved" {
		t.Fatalf("Status = %s, want resolved", resolved.Status)
	}
}
