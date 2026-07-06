package reliability

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return dbtest.NewDB(t, &AgentStatus{}, &LLMCostRecord{}, &FailureRecord{})
}

func TestGetAgentStatus_Empty(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	statuses, err := svc.GetAgentStatus(context.Background())
	if err != nil {
		t.Fatalf("GetAgentStatus failed: %v", err)
	}
	if len(statuses) != 0 {
		t.Fatalf("expected 0 statuses, got %d", len(statuses))
	}
}

func TestUpsertAgentHeartbeat_Create(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	err := svc.UpsertAgentHeartbeat(context.Background(), "A5", "Stock Alert", "commerce", "running", "")
	if err != nil {
		t.Fatalf("UpsertAgentHeartbeat failed: %v", err)
	}

	statuses, _ := svc.GetAgentStatus(context.Background())
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].AgentID != "A5" {
		t.Fatalf("AgentID = %q, want %q", statuses[0].AgentID, "A5")
	}
	if statuses[0].Status != "running" {
		t.Fatalf("Status = %q, want %q", statuses[0].Status, "running")
	}
}

func TestUpsertAgentHeartbeat_Update(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	_ = svc.UpsertAgentHeartbeat(context.Background(), "A5", "Stock Alert", "commerce", "running", "")
	err := svc.UpsertAgentHeartbeat(context.Background(), "A5", "Stock Alert", "commerce", "error", "oops")
	if err != nil {
		t.Fatalf("Upsert update failed: %v", err)
	}

	statuses, _ := svc.GetAgentStatus(context.Background())
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status after update, got %d", len(statuses))
	}
	if statuses[0].ErrorReason != "oops" {
		t.Fatalf("ErrorReason = %q, want %q", statuses[0].ErrorReason, "oops")
	}
}

func TestRecordGetResolveFailure(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	err := svc.RecordFailure(context.Background(), "A5", "stock_alert", "timeout", 3)
	if err != nil {
		t.Fatalf("RecordFailure failed: %v", err)
	}

	records, err := svc.GetFailures(context.Background())
	if err != nil {
		t.Fatalf("GetFailures failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(records))
	}
	if records[0].Status != "pending" {
		t.Fatalf("Status = %q, want %q", records[0].Status, "pending")
	}

	err = svc.ResolveFailure(context.Background(), records[0].ID)
	if err != nil {
		t.Fatalf("ResolveFailure failed: %v", err)
	}

	records, _ = svc.GetFailures(context.Background())
	if records[0].Status != "resolved" {
		t.Fatalf("after resolve Status = %q, want %q", records[0].Status, "resolved")
	}
	if records[0].ResolvedAt == nil {
		t.Fatal("ResolvedAt should be set after resolve")
	}
}

func TestRecordLLMCost(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	err := svc.RecordLLMCost(context.Background(), LLMCostRecord{
		AgentID:      "A2",
		ModelName:    "gpt-4",
		InputTokens:  100,
		OutputTokens: 50,
		CostUSD:      0.015,
	})
	if err != nil {
		t.Fatalf("RecordLLMCost failed: %v", err)
	}

	// GetLLMCost with period="today" should find the record
	resp, err := svc.GetLLMCost(context.Background(), "today")
	if err != nil {
		t.Fatalf("GetLLMCost failed: %v", err)
	}
	if resp.TotalTokens != 150 {
		t.Fatalf("TotalTokens = %d, want %d", resp.TotalTokens, 150)
	}
	if resp.TotalCostUSD != 0.015 {
		t.Fatalf("TotalCostUSD = %f, want %f", resp.TotalCostUSD, 0.015)
	}
	if len(resp.ByAgent) != 1 {
		t.Fatalf("expected 1 agent cost, got %d", len(resp.ByAgent))
	}
}

func TestGetLLMCost_Empty(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	resp, err := svc.GetLLMCost(context.Background(), "today")
	if err != nil {
		t.Fatalf("GetLLMCost failed: %v", err)
	}
	if resp.TotalTokens != 0 {
		t.Fatalf("expected 0 total tokens, got %d", resp.TotalTokens)
	}
}
