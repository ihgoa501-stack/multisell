package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/domain/approval"
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
	if err := db.AutoMigrate(&AITrace{}, &AITraceEvent{}, &AIEvidenceRef{}, &UnifiedAction{}, &trustscore.TrustScore{}, &actionpolicy.PolicyRule{}, &approval.ApprovalRequest{}); err != nil {
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
