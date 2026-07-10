package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/guardrails"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"github.com/lingmirror/backend-go/internal/domain/trustscore"
	"github.com/lingmirror/backend-go/internal/domain/actionpolicy"
)

// testDBCounter ensures each test gets a unique in-memory SQLite DB so
// tests don't share state via the shared cache.
var testDBCounter atomic.Int64

// newTestDB returns an isolated in-memory SQLite DB with AI tables
// auto-migrated. We use GORM AutoMigrate on the model structs rather
// than raw DDL so the test doesn't depend on PostgreSQL.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	n := testDBCounter.Add(1)
	dsn := fmt.Sprintf("file:lingmirror_test_%d?mode=memory&cache=shared&_busy_timeout=5000", n)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&AITrace{}, &AITraceEvent{}, &AIEvidenceRef{}, &UnifiedAction{}, &trustscore.TrustScore{}, &actionpolicy.PolicyRule{}, &approval.ApprovalRequest{}, &operationlog.OperationLog{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return db
}

func testLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}

func TestTraceWriter_Start_Append_Complete(t *testing.T) {
	db := newTestDB(t)
	w := NewTraceWriter(db, testLogger())

	traceID, err := w.Start(&CreateTraceInput{
		AgentID:       "A5",
		DecisionPoint: "stock_alert",
		ModelName:     "gpt-4o-mini",
		InputContext:  json.RawMessage(`{"sku_id":1}`),
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !strings.HasPrefix(traceID, "trc_") {
		t.Fatalf("traceID prefix: %q", traceID)
	}

	// Append two events; verify seq increments.
	ev1, err := w.AppendEvent(traceID, &AppendEventInput{EventType: "prompt_start", Content: "hello"})
	if err != nil {
		t.Fatalf("AppendEvent 1: %v", err)
	}
	if ev1.Seq != 1 {
		t.Fatalf("seq 1 = %d", ev1.Seq)
	}
	ev2, err := w.AppendEvent(traceID, &AppendEventInput{EventType: "tool_call", Content: "fetch_inventory"})
	if err != nil {
		t.Fatalf("AppendEvent 2: %v", err)
	}
	if ev2.Seq != 2 {
		t.Fatalf("seq 2 = %d", ev2.Seq)
	}

	// Add evidence.
	ev, err := w.AddEvidence(traceID, &AddEvidenceInput{
		SourceType: "inventory",
		SourceID:   "sku-1",
		Title:      "库存快照",
		Summary:    "12 件",
	})
	if err != nil || ev.ID == 0 {
		t.Fatalf("AddEvidence: %v %+v", err, ev)
	}

	// Complete the trace.
	conf := 0.82
	completed, err := w.Complete(traceID, &CompleteTraceInput{
		FinalOutput: json.RawMessage(`{"recommendation":"补货"}`),
		Confidence:  &conf,
		RiskLevel:   "medium",
		TokenCount:  200,
		Status:      "completed",
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if completed.Status != "completed" {
		t.Fatalf("status = %q", completed.Status)
	}
	if completed.Confidence == nil || *completed.Confidence != 0.82 {
		t.Fatalf("confidence = %+v", completed.Confidence)
	}

	// Full detail should have 2 events + 1 evidence.
	detail, err := w.GetDetail(traceID)
	if err != nil {
		t.Fatalf("GetDetail: %v", err)
	}
	if len(detail.Events) != 2 {
		t.Fatalf("events = %d", len(detail.Events))
	}
	if len(detail.Evidence) != 1 {
		t.Fatalf("evidence = %d", len(detail.Evidence))
	}
}

func TestService_ActionLifecycle(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	// Create action.
	a, err := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace",
		SourceID:    "trc_test_1",
		SourceType:  "agent_run",
		AgentID:     "A6",
		SquadID:     "autonomous",
		ActionType:  "profit_check",
		Title:       "调价止损",
		RiskLevel:   "medium",
		ProposedBy:  "agent:A6",
	})
	if err != nil {
		t.Fatalf("CreateAction: %v", err)
	}
	if a.Status != "suggested" {
		t.Fatalf("status = %q", a.Status)
	}
	if !a.RequiresApproval {
		t.Fatal("should require approval by default")
	}

	// Approve.
	approved, err := svc.ApproveAction(a.ID, "alice", nil, "")
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if approved.Status != "approved" {
		t.Fatalf("status = %q", approved.Status)
	}

	// Reject should fail (already approved).
	if _, err := svc.RejectAction(a.ID, "bob", nil, "no"); err == nil {
		t.Fatal("expected reject-after-approve to fail")
	}

	// Execute.
	executed, err := svc.ExecuteAction(a.ID, nil, "alice", "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if executed.Status != "executed" {
		t.Fatalf("status = %q", executed.Status)
	}

	// Review.
	reviewed, err := svc.ReviewAction(a.ID)
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if reviewed.Status != "reviewed" {
		t.Fatalf("status = %q", reviewed.Status)
	}
}

func TestService_ExecuteWithoutApproval(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	// Action that requires approval.
	a, _ := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_test_2", SourceType: "agent_run",
		AgentID: "A7", ActionType: "compliance_check", Title: "移除敏感词",
		RiskLevel: "high", ProposedBy: "agent:A7",
	})
	// Execute without approving first → should fail.
	_, err := svc.ExecuteAction(a.ID, nil, "alice", "")
	if err == nil {
		t.Fatal("expected Execute to fail without approval")
	}
}

func TestService_ExecuteAutoApproved(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	// Action that does NOT require approval.
	noApproval := false
	a, _ := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_test_3", SourceType: "agent_run",
		AgentID: "A2", ActionType: "listing_optimize", Title: "重写标题",
		RiskLevel: "low", ProposedBy: "agent:A2", RequiresApproval: &noApproval,
	})
	// Execute directly → should succeed.
	executed, err := svc.ExecuteAction(a.ID, nil, "system", "")
	if err != nil {
		t.Fatalf("Execute auto-approved: %v", err)
	}
	if executed.Status != "executed" {
		t.Fatalf("status = %q", executed.Status)
	}
}

func TestService_ListTraces_Filtering(t *testing.T) {
	db := newTestDB(t)
	w := NewTraceWriter(db, testLogger())

	// Start 3 traces with different agents.
	_, _ = w.Start(&CreateTraceInput{AgentID: "A1", DecisionPoint: "product_scout"})
	_, _ = w.Start(&CreateTraceInput{AgentID: "A5", DecisionPoint: "stock_alert"})
	_, _ = w.Start(&CreateTraceInput{AgentID: "A5", DecisionPoint: "replenishment_plan"})

	svc := NewService(db, testLogger())
	p := common.Pagination{Page: 1, Size: 10}

	// All
	all, total, err := svc.ListTraces(&p, &TraceListFilter{})
	if err != nil || total != 3 {
		t.Fatalf("ListTraces all: total=%d err=%v", total, err)
	}
	if len(all) != 3 {
		t.Fatalf("len = %d", len(all))
	}

	// Filter by agent
	a5, total, _ := svc.ListTraces(&p, &TraceListFilter{AgentID: "A5"})
	if total != 2 {
		t.Fatalf("A5 total = %d", total)
	}
	for _, tr := range a5 {
		if tr.AgentID != "A5" {
			t.Fatalf("expected A5, got %s", tr.AgentID)
		}
	}
}

func TestRegistry_DefaultAgents(t *testing.T) {
	r := DefaultRegistry()
	if len(r.Agents) != 17 {
		t.Fatalf("expected 17 agents, got %d", len(r.Agents))
	}
	ids := r.IDs()
	want := []string{"A1", "A2", "A3", "A4", "A5", "A6", "A7", "G1", "G2", "G3", "G0", "A8", "A9", "A10", "A11", "content_ai", "scheduler"}
	for i, w := range want {
		if ids[i] != w {
			t.Fatalf("ids[%d] = %s, want %s", i, ids[i], w)
		}
	}

	// Case-insensitive lookup.
	a, ok := r.Get("a5")
	if !ok || a.ID != "A5" {
		t.Fatalf("Get(a5) = %+v ok=%v", a, ok)
	}

	// Unknown agent.
	_, ok = r.Get("Z9")
	if ok {
		t.Fatal("expected Z9 to be unknown")
	}

	}
func TestOrchestrator_Run_StubProvider(t *testing.T) {
	db := newTestDB(t)
	orch := NewOrchestrator(db, testLogger())
	// Default provider is stub — no API key needed.

	result, err := orch.Run(&RunAgentRequest{
		AgentID:       "A5",
		DecisionPoint: "stock_alert",
		Context: map[string]interface{}{
				"sku_code":          "TEST-SKU-001",
				"sellable_stock":    float64(100),
				"sales_7d":          float64(35),
				"lead_time_days":    float64(30),
				"safety_stock_days": float64(16),
				"message":           "缺货",
			},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.HasPrefix(result.TraceID, "trc_") {
		t.Fatalf("traceID = %q", result.TraceID)
	}
	if result.AgentID != "A5" {
		t.Fatalf("agentID = %s", result.AgentID)
	}
	if result.Output["risk_reason"] == nil {
		t.Fatal("missing risk_reason")
	}
	if result.Action == nil {
		t.Fatal("A5 is supervised — should create an action")
	}
	if result.Action.Status != "suggested" {
		t.Fatalf("action status = %s", result.Action.Status)
	}

	// Verify trace was persisted with events.
	detail, err := orch.traces.GetDetail(result.TraceID)
	if err != nil {
		t.Fatalf("GetDetail: %v", err)
	}
	if len(detail.Events) == 0 {
		t.Fatal("no events persisted")
	}
	if len(detail.Evidence) == 0 {
		t.Fatal("no evidence persisted")
	}
}

func TestOrchestrator_Run_UnknownAgent(t *testing.T) {
	db := newTestDB(t)
	orch := NewOrchestrator(db, testLogger())
	_, err := orch.Run(&RunAgentRequest{AgentID: "Z9", DecisionPoint: "noop"})
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
}

func TestOrchestrator_Run_InvalidDecisionPoint(t *testing.T) {
	db := newTestDB(t)
	orch := NewOrchestrator(db, testLogger())
	_, err := orch.Run(&RunAgentRequest{AgentID: "A5", DecisionPoint: "unknown_point"})
	if err == nil {
		t.Fatal("expected error for invalid decision point")
	}
}

func TestOrchestrator_Chat_Routing(t *testing.T) {
	db := newTestDB(t)
	orch := NewOrchestrator(db, testLogger())

	cases := []struct {
		msg       string
		wantAgent string
	}{
		{"库存严重不足", "A5"},
		{"利润率太低", "A6"},
		{"帮我优化 listing", "A2"},
		{"广告 ACOS 太高", "A3"},
		{"查一下合规", "A7"},
		{"折扣太大有风险", "G3"},
		{"dashboard 概览", "G1"},
		{"hello world", "G1"}, // fallback
	}
	for _, c := range cases {
		result, err := orch.Chat(c.msg, nil)
		if err != nil {
			t.Fatalf("Chat(%q): %v", c.msg, err)
		}
		if result.AgentID != c.wantAgent {
			t.Fatalf("Chat(%q): agent = %s, want %s", c.msg, result.AgentID, c.wantAgent)
		}
	}
}

// StubProvider direct tests.

func TestStubProvider_Chat(t *testing.T) {
	p := &StubProvider{logger: testLogger()}
	resp, err := p.Chat(context.Background(), &LLMRequest{
		Messages: []LLMMessage{{Role: "user", Content: "检查库存"}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if !strings.Contains(resp.Answer, "补货") {
		t.Fatalf("answer = %q", resp.Answer)
	}
	if resp.Model != "stub-v1" {
		t.Fatalf("model = %s", resp.Model)
	}
}

func TestStubProvider_Stream(t *testing.T) {
	p := &StubProvider{logger: testLogger()}
	ch, err := p.ChatStream(context.Background(), &LLMRequest{
		Messages: []LLMMessage{{Role: "user", Content: "利润"}},
	})
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	var sb strings.Builder
	gotDone := false
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("chunk err: %v", chunk.Err)
		}
		if chunk.Done {
			gotDone = true
		}
		sb.WriteString(chunk.Text)
	}
	if !gotDone {
		t.Fatal("stream did not close with Done")
	}
	if sb.Len() == 0 {
		t.Fatal("empty stream output")
	}
}

// Test 3: Verify the orchestrator approval creation pattern — creating a
// UnifiedAction with requires_approval=true, then manually creating a linked
// approval_request (as the orchestrator would), and verifying through the
// approval service.
func TestService_ApprovalCreationPattern(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&approval.ApprovalRequest{}); err != nil {
		t.Fatalf("migrate approval_request: %v", err)
	}

	aiSvc := NewService(db, testLogger())
	approvalSvc := approval.NewService(db, testLogger(), nil)

	// Create a UnifiedAction with requires_approval=true
	a, err := aiSvc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace",
		SourceID:    "trc_test_3",
		SourceType:  "agent_run",
		AgentID:     "A5",
		ActionType:  "stock_alert",
		Title:       "补货建议",
		RiskLevel:   "medium",
		ProposedBy:  "agent:A5",
	})
	if err != nil {
		t.Fatalf("CreateAction: %v", err)
	}
	if a.Status != "suggested" {
		t.Fatalf("expected status 'suggested', got %q", a.Status)
	}
	if !a.RequiresApproval {
		t.Fatal("expected RequiresApproval=true for supervised agent action")
	}

	// Manually create an approval_request linked to the unified_action
	// (this is the pattern the orchestrator uses).
	req, err := approvalSvc.Create(&approval.CreateApprovalInput{
		ProductID:   1,
		RequestType: "unified_action",
		Requester:   "agent:A5",
		EntityType:  "unified_action",
		EntityID:    a.ID,
	})
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}

	// Verify entity_type and entity_id are correctly set
	if req.EntityType != "unified_action" {
		t.Errorf("entity_type = %q, want %q", req.EntityType, "unified_action")
	}
	if req.EntityID != a.ID {
		t.Errorf("entity_id = %d, want %d", req.EntityID, a.ID)
	}

	// Verify it can be queried from the approval service
	has, err := approvalSvc.HasPendingForEntity("unified_action", a.ID)
	if err != nil {
		t.Fatalf("HasPendingForEntity: %v", err)
	}
	if !has {
		t.Error("expected HasPendingForEntity to return true for pending approval")
	}
}

// ---------- route table tests ----------

func TestRouteChat_FirstKMScout(t *testing.T) {
	tests := []struct {
		name   string
		msg    string
		wantDP string
	}{
		{"家居+俄罗斯+Ozon", "我想调研家居类目，目标俄罗斯 Ozon", "first_km_scout"},
		{"research+Russia", "research home category for Russia Ozon", "first_km_scout"},
		{"宠物+Amazon", "调研宠物用品，美国 Amazon", "first_km_scout"},
		{"plain research no market", "调研家居", "product_research"},
		{"stock alert", "库存不足", "stock_alert"},
		{"profit check", "利润分析", "profit_check"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, dp := routeChat(tc.msg)
			if dp != tc.wantDP {
				t.Errorf("routeChat(%q) = %q, want %q", tc.msg, dp, tc.wantDP)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Execution Gate focused tests
// ---------------------------------------------------------------------------

func TestService_CreateAction_PersistsExecutionModeAndIdempotencyKey(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	// Default execution_mode.
	a, err := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_exec_1", SourceType: "agent_run",
		AgentID: "A6", ActionType: "profit_check", Title: "default mode",
		ProposedBy: "agent:A6",
	})
	if err != nil {
		t.Fatalf("CreateAction: %v", err)
	}
	if a.ExecutionMode != "production" {
		t.Errorf("default execution_mode = %q, want %q", a.ExecutionMode, "production")
	}

	// Explicit production.
	a, err = svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_exec_2", SourceType: "agent_run",
		AgentID: "A6", ActionType: "profit_check", Title: "prod mode",
		ExecutionMode: "production", ProposedBy: "agent:A6",
	})
	if err != nil {
		t.Fatalf("CreateAction: %v", err)
	}
	if a.ExecutionMode != "production" {
		t.Errorf("production execution_mode = %q", a.ExecutionMode)
	}

	// Explicit dry_run.
	a, err = svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_exec_3", SourceType: "agent_run",
		AgentID: "A6", ActionType: "profit_check", Title: "dry run",
		ExecutionMode: "dry_run", ProposedBy: "agent:A6",
	})
	if err != nil {
		t.Fatalf("CreateAction: %v", err)
	}
	if a.ExecutionMode != "dry_run" {
		t.Errorf("dry_run execution_mode = %q", a.ExecutionMode)
	}

	// IdempotencyKey persisted.
	a, err = svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_exec_4", SourceType: "agent_run",
		AgentID: "A6", ActionType: "profit_check", Title: "idempotent",
		IdempotencyKey: "idem-001", ProposedBy: "agent:A6",
	})
	if err != nil {
		t.Fatalf("CreateAction: %v", err)
	}
	if a.IdempotencyKey != "idem-001" {
		t.Errorf("idempotency_key = %q, want %q", a.IdempotencyKey, "idem-001")
	}
}

func TestService_DryRunExecute_NoSideEffects(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	noApproval := false
	a, err := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_dry_1", SourceType: "agent_run",
		AgentID: "A2", ActionType: "listing_optimize", Title: "dry run test",
		RiskLevel: "low", ProposedBy: "agent:A2",
		RequiresApproval: &noApproval, ExecutionMode: "dry_run",
	})
	if err != nil {
		t.Fatalf("CreateAction: %v", err)
	}

	// Execute dry-run — should return without changing status or writing executed_by.
	result, err := svc.ExecuteAction(a.ID, nil, "alice", "")
	if err != nil {
		t.Fatalf("ExecuteAction dry_run: %v", err)
	}
	if result.Status != "suggested" {
		t.Errorf("dry_run should not change status; got %q", result.Status)
	}
	if result.ExecutedBy != "" {
		t.Errorf("dry_run should not write executed_by; got %q", result.ExecutedBy)
	}
	if result.ExecutedByUserID != nil {
		t.Errorf("dry_run should not write executed_by_user_id; got %v", result.ExecutedByUserID)
	}
	if result.ExecutingAt != nil {
		t.Errorf("dry_run should not write executing_at")
	}

	// Verify DB state unchanged.
	var fromDB UnifiedAction
	if err := db.First(&fromDB, a.ID).Error; err != nil {
		t.Fatalf("read from DB: %v", err)
	}
	if fromDB.Status != "suggested" {
		t.Errorf("DB status = %q after dry_run, want %q", fromDB.Status, "suggested")
	}
}

func TestService_Execute_UnknownMode_ReturnsError(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	noApproval := false
	a, err := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_unk_1", SourceType: "agent_run",
		AgentID: "A2", ActionType: "listing_optimize", Title: "unknown mode",
		RiskLevel: "low", ProposedBy: "agent:A2",
		RequiresApproval: &noApproval, ExecutionMode: "bogus_mode",
	})
	if err != nil {
		t.Fatalf("CreateAction: %v", err)
	}

	_, err = svc.ExecuteAction(a.ID, nil, "alice", "")
	if err == nil {
		t.Fatal("expected error for unknown execution mode")
	}
	if !strings.Contains(err.Error(), "unknown execution mode") {
		t.Errorf("error = %q, want 'unknown execution mode'", err.Error())
	}

	// Verify no state change in DB.
	var fromDB UnifiedAction
	db.First(&fromDB, a.ID)
	if fromDB.Status != "suggested" {
		t.Errorf("status changed to %q after unknown mode error", fromDB.Status)
	}
}

func TestService_ExecuteAction_Sandbox(t *testing.T) {
	// Mock execCommand to avoid running actual Docker/Git commands in unit tests.
	oldExecCommand := execCommand
	defer func() { execCommand = oldExecCommand }()

	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "mock sandbox output")
	}

	db := newTestDB(t)
	svc := NewService(db, testLogger())
	noApproval := false
	a, err := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_sb_1", SourceType: "agent_run",
		AgentID: "A2", ActionType: "listing_optimize", Title: "sandbox run",
		RiskLevel: "low", ProposedBy: "agent:A2", RequiresApproval: &noApproval,
		ExecutionMode: "sandbox",
	})
	if err != nil {
		t.Fatalf("CreateAction: %v", err)
	}

	result, err := svc.ExecuteAction(a.ID, nil, "alice", "")
	if err != nil {
		t.Fatalf("expected sandbox trigger attempt, got: %v", err)
	}
	if result.Status != "executed" && result.Status != "failed" {
		t.Errorf("expected executed/failed status from sandbox run, got: %s", result.Status)
	}
}

func TestService_ExecuteAction_Sandbox_Failure(t *testing.T) {
	// Mock execCommand to return an error (using "false" command).
	oldExecCommand := execCommand
	defer func() { execCommand = oldExecCommand }()

	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}

	db := newTestDB(t)
	svc := NewService(db, testLogger())
	noApproval := false
	a, err := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_sbx_1", SourceType: "agent_run",
		AgentID: "A2", ActionType: "listing_optimize", Title: "sandbox failure test",
		RiskLevel: "low", ProposedBy: "agent:A2",
		RequiresApproval: &noApproval, ExecutionMode: "sandbox",
	})
	if err != nil {
		t.Fatalf("CreateAction: %v", err)
	}

	_, err = svc.ExecuteAction(a.ID, nil, "alice", "")
	if err == nil {
		t.Fatal("expected error for sandbox failure")
	}
	if !strings.Contains(err.Error(), "sandbox run failed") {
		t.Errorf("error = %q, want 'sandbox run failed'", err.Error())
	}

	// Verify DB state updated to "failed".
	var fromDB UnifiedAction
	db.First(&fromDB, a.ID)
	if fromDB.Status != "failed" {
		t.Errorf("status = %q after sandbox failure, want 'failed'", fromDB.Status)
	}
}

func TestService_ApproveAction_SavesUserID(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	a, err := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_uid_1", SourceType: "agent_run",
		AgentID: "A6", ActionType: "profit_check", Title: "user id test",
		RiskLevel: "medium", ProposedBy: "agent:A6",
	})
	if err != nil {
		t.Fatalf("CreateAction: %v", err)
	}

	uid := int64(42)
	approved, err := svc.ApproveAction(a.ID, "alice", &uid, "")
	if err != nil {
		t.Fatalf("ApproveAction: %v", err)
	}
	if approved.ApprovedByUserID == nil {
		t.Fatal("expected approved_by_user_id to be set")
	}
	if *approved.ApprovedByUserID != 42 {
		t.Errorf("approved_by_user_id = %d, want 42", *approved.ApprovedByUserID)
	}
	if approved.ApprovedBy != "alice" {
		t.Errorf("approved_by = %q, want %q", approved.ApprovedBy, "alice")
	}

	// Verify DB.
	var fromDB UnifiedAction
	db.First(&fromDB, a.ID)
	if fromDB.ApprovedByUserID == nil || *fromDB.ApprovedByUserID != 42 {
		t.Errorf("DB approved_by_user_id = %v, want 42", fromDB.ApprovedByUserID)
	}

	// Also test RejectAction saves rejected_by_user_id.
	// Create another action and reject it.
	a2, err := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_uid_2", SourceType: "agent_run",
		AgentID: "A6", ActionType: "profit_check", Title: "reject uid test",
		RiskLevel: "medium", ProposedBy: "agent:A6",
	})
	if err != nil {
		t.Fatalf("CreateAction: %v", err)
	}
	uid2 := int64(99)
	rejected, err := svc.RejectAction(a2.ID, "bob", &uid2, "not needed")
	if err != nil {
		t.Fatalf("RejectAction: %v", err)
	}
	if rejected.RejectedByUserID == nil {
		t.Fatal("expected rejected_by_user_id to be set")
	}
	if *rejected.RejectedByUserID != 99 {
		t.Errorf("rejected_by_user_id = %d, want 99", *rejected.RejectedByUserID)
	}
}

func TestService_ExecuteAction_SavesUserID(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())

	noApproval := false
	a, err := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_exuid_1", SourceType: "agent_run",
		AgentID: "A2", ActionType: "listing_optimize", Title: "exec user id",
		RiskLevel: "low", ProposedBy: "agent:A2",
		RequiresApproval: &noApproval,
	})
	if err != nil {
		t.Fatalf("CreateAction: %v", err)
	}

	uid := int64(77)
	executed, err := svc.ExecuteAction(a.ID, &uid, "system", "")
	if err != nil {
		t.Fatalf("ExecuteAction: %v", err)
	}
	if executed.ExecutedByUserID == nil {
		t.Fatal("expected executed_by_user_id to be set")
	}
	if *executed.ExecutedByUserID != 77 {
		t.Errorf("executed_by_user_id = %d, want 77", *executed.ExecutedByUserID)
	}
	if executed.ExecutedBy != "system" {
		t.Errorf("executed_by = %q, want %q", executed.ExecutedBy, "system")
	}

	// Verify DB.
	var fromDB UnifiedAction
	db.First(&fromDB, a.ID)
	if fromDB.ExecutedByUserID == nil || *fromDB.ExecutedByUserID != 77 {
		t.Errorf("DB executed_by_user_id = %v, want 77", fromDB.ExecutedByUserID)
	}
}


func TestService_ExecuteAction_Idempotency_Reexecution(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())
	noApproval := false
	a, err := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_idem_1", SourceType: "agent_run",
		AgentID: "A2", ActionType: "listing_optimize", Title: "idempotent exec",
		RiskLevel: "low", ProposedBy: "agent:A2",
		RequiresApproval: &noApproval, IdempotencyKey: "idem-rex-001",
	})
	if err != nil { t.Fatalf("CreateAction: %v", err) }
	first, err := svc.ExecuteAction(a.ID, nil, "alice", "")
	if err != nil { t.Fatalf("first ExecuteAction: %v", err) }
	if first.Status != "executed" { t.Fatalf("first status = %q", first.Status) }
	second, err := svc.ExecuteAction(a.ID, nil, "alice", "")
	if err != nil { t.Fatalf("second (idempotent) ExecuteAction: %v", err) }
	if second.Status != "executed" { t.Errorf("second status = %q, want %q", second.Status, "executed") }
}

func TestService_ExecuteAction_GuardrailsCheck(t *testing.T) {
	db := newTestDB(t)
	t.Run("blocked", func(t *testing.T) {
		// Use an ExecutionGuard that blocks purchases above /bin/zsh.
		eg := guardrails.NewExecutionGuardWithRules([]guardrails.ExecutionRule{
			{Name: "block_purchases_above_1", MaxAmount: 1, ActionTypes: []string{"purchase"}},
		})
		chain := guardrails.NewChain()
		chain.Add(eg)
		svc := NewService(db, testLogger()).WithGuard(chain)
		noApproval := false
		a, _ := svc.CreateAction(&CreateActionInput{
			SourceTable: "ai_trace", SourceID: "trc_grd_1", SourceType: "agent_run",
			AgentID: "A2", ActionType: "purchase", Title: "blocked purchase",
			RiskLevel: "low", ProposedBy: "agent:A2", RequiresApproval: &noApproval,
			Payload: json.RawMessage(`{"amount":100}`),
		})
		_, err := svc.ExecuteAction(a.ID, nil, "alice", "")
		if err == nil || !errors.Is(err, ErrBlockedByGuardrails) {
			t.Fatalf("expected guard blocked error, got %v", err)
		}
		var fromDB UnifiedAction
		db.First(&fromDB, a.ID)
		if fromDB.Status != "suggested" {
			t.Errorf("status changed to %q after guard block", fromDB.Status)
		}
	})
	t.Run("passes", func(t *testing.T) {
		svc := NewService(db, testLogger())
		noApproval := false
		a, _ := svc.CreateAction(&CreateActionInput{
			SourceTable: "ai_trace", SourceID: "trc_grd_3", SourceType: "agent_run",
			AgentID: "A2", ActionType: "listing_optimize", Title: "guard passes",
			RiskLevel: "low", ProposedBy: "agent:A2", RequiresApproval: &noApproval,
		})
		result, err := svc.ExecuteAction(a.ID, nil, "alice", "")
		if err != nil { t.Fatalf("expected success, got %v", err) }
		if result.Status != "executed" { t.Errorf("status = %q, want %q", result.Status, "executed") }
	})
}

func TestService_ExecuteAction_ConcurrentClaim(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())
	noApproval := false
	a, err := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_conc_1", SourceType: "agent_run",
		AgentID: "A2", ActionType: "listing_optimize", Title: "concurrent",
		RiskLevel: "low", ProposedBy: "agent:A2",
		RequiresApproval: &noApproval,
	})
	if err != nil { t.Fatalf("CreateAction: %v", err) }
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.ExecuteAction(a.ID, nil, "alice", "")
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	var errs []error
	for e := range results {
		errs = append(errs, e)
	}
	if len(errs) != 2 { t.Fatalf("expected 2 results, got %d", len(errs)) }
	success := 0
	for _, e := range errs {
		if e == nil { success++ }
	}
	if success != 1 { t.Errorf("expected exactly 1 success, got %d", success) }
}

func TestService_ExecuteAction_Expired(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, testLogger())
	noApproval := false
	a, _ := svc.CreateAction(&CreateActionInput{
		SourceTable: "ai_trace", SourceID: "trc_exp_1", SourceType: "agent_run",
		AgentID: "A2", ActionType: "listing_optimize", Title: "expired action",
		RiskLevel: "low", ProposedBy: "agent:A2", RequiresApproval: &noApproval,
	})
	// Force Creation Date to 3 hours ago
	db.Model(&UnifiedAction{}).Where("id = ?", a.ID).Update("created_at", time.Now().Add(-3 * time.Hour))

	_, err := svc.ExecuteAction(a.ID, nil, "alice", "")
	if err == nil || !errors.Is(err, ErrActionExpired) {
		t.Fatalf("expected ErrActionExpired, got: %v", err)
	}
}
