package operationlog

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestService_CreateWithContextHonorsCancellation(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OperationLog{})
	svc := NewService(db, dbtest.NewLogger(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.CreateWithContext(ctx, &OperationLog{Module: "audit", Action: "cancelled"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("CreateWithContext error = %v, want context.Canceled", err)
	}

	var count int64
	if err := db.Model(&OperationLog{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("cancelled audit write persisted %d rows", count)
	}
}

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

// ---------------------------------------------------------------------------
// RedactSensitive tests
// ---------------------------------------------------------------------------

func TestRedactSensitive_EmptyString(t *testing.T) {
	if got := RedactSensitive(""); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestRedactSensitive_NonJSON(t *testing.T) {
	input := "plain text, no JSON"
	if got := RedactSensitive(input); got != input {
		t.Errorf("expected unchanged, got %q", got)
	}
}

func TestRedactSensitive_PasswordField(t *testing.T) {
	input := `{"password": "s3cret!", "name": "test"}`
	got := RedactSensitive(input)
	if strings.Contains(got, "s3cret!") {
		t.Errorf("password was not redacted: %s", got)
	}
	if !strings.Contains(got, "***REDACTED***") {
		t.Errorf("expected [REDACTED] marker: %s", got)
	}
	// Verify JSON is still valid and name is preserved
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("invalid JSON after redaction: %v", err)
	}
	if parsed["name"] != "test" {
		t.Errorf("expected name=test, got %v", parsed["name"])
	}
}

func TestRedactSensitive_APIKeyField(t *testing.T) {
	input := `{"api_key": "sk-1234567890abcdef", "data": "hello"}`
	got := RedactSensitive(input)
	if strings.Contains(got, "sk-1234567890abcdef") {
		t.Errorf("api_key was not redacted: %s", got)
	}
	if !strings.Contains(got, "***REDACTED***") {
		t.Errorf("expected [REDACTED] marker: %s", got)
	}
}

func TestRedactSensitive_TokenAndSecret(t *testing.T) {
	input := `{"token": "abc123", "api_secret": "xyz789", "user": "admin"}`
	got := RedactSensitive(input)
	if strings.Contains(got, "abc123") || strings.Contains(got, "xyz789") {
		t.Errorf("sensitive values not redacted: %s", got)
	}
}

func TestRedactSensitive_AccessKey(t *testing.T) {
	input := `{"access_key": "AKIA123456", "access_secret": "verysecret"}`
	got := RedactSensitive(input)
	if strings.Contains(got, "AKIA123456") || strings.Contains(got, "verysecret") {
		t.Errorf("access keys not redacted: %s", got)
	}
}

func TestRedactSensitive_JWT(t *testing.T) {
	input := `{"jwt": "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0", "sub": "123"}`
	got := RedactSensitive(input)
	if strings.Contains(got, "eyJhbGciOiJIUzI1NiJ9") {
		t.Errorf("jwt was not redacted: %s", got)
	}
}

func TestRedactSensitive_Authorization(t *testing.T) {
	input := `{"authorization": "Bearer abc123"}`
	got := RedactSensitive(input)
	if strings.Contains(got, "Bearer") {
		t.Errorf("authorization was not redacted: %s", got)
	}
}

func TestRedactSensitive_NestedObject(t *testing.T) {
	input := `{"outer": {"inner": {"password": "hunter2"}, "keep": "me"}}`
	got := RedactSensitive(input)
	if strings.Contains(got, "hunter2") {
		t.Errorf("nested password was not redacted: %s", got)
	}
	if !strings.Contains(got, "me") {
		t.Errorf("non-sensitive nested field should be preserved: %s", got)
	}
}

func TestRedactSensitive_ArrayOfObjects(t *testing.T) {
	input := `[{"token": "abc"}, {"name": "safe"}]`
	got := RedactSensitive(input)
	if strings.Contains(got, "abc") {
		t.Errorf("token in array was not redacted: %s", got)
	}
	if !strings.Contains(got, "safe") {
		t.Errorf("non-sensitive value should be preserved: %s", got)
	}
}

func TestRedactSensitive_LogService(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OperationLog{})
	svc := NewService(db, dbtest.NewLogger(t))

	err := svc.Log("auth", "login", "1", "admin", `{"password": "hunter2", "user": "admin"}`)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}

	var all []OperationLog
	db.Find(&all)
	if len(all) != 1 {
		t.Fatalf("expected 1 log, got %d", len(all))
	}
	if strings.Contains(all[0].Content, "hunter2") {
		t.Errorf("password was not redacted via Log: %s", all[0].Content)
	}
}

func TestRedactSensitive_LogStructuredService(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &OperationLog{})
	svc := NewService(db, dbtest.NewLogger(t))

	err := svc.LogStructured(&StructuredLogInput{
		Module:     "auth",
		Action:     "update_credential",
		ResourceID: "1",
		Operator:   "admin",
		Content:    `{"api_key": "sk-abc123", "name": "test"}`,
		Result:     "success",
	})
	if err != nil {
		t.Fatalf("LogStructured: %v", err)
	}

	var all []OperationLog
	db.Find(&all)
	if len(all) != 1 {
		t.Fatalf("expected 1 log, got %d", len(all))
	}
	if strings.Contains(all[0].Content, "sk-abc123") {
		t.Errorf("api_key was not redacted via LogStructured: %s", all[0].Content)
	}
}
