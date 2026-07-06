package agentos

import (
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestSLAEscalation(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &ai.UnifiedAction{})
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger, nil)

	// Create 5 actions with varying statuses and ages
	now := time.Now()
	actions := []ai.UnifiedAction{
		{SourceTable: "t1", SourceID: "s1", SourceType: "order", Title: "Overdue With User", Status: ai.ActionStatusSuggested, UserID: int64Ptr(100), CreatedAt: now.Add(-2 * time.Hour)},
		{SourceTable: "t1", SourceID: "s2", SourceType: "order", Title: "Overdue No User", Status: ai.ActionStatusSuggested, UserID: nil, CreatedAt: now.Add(-3 * time.Hour)},
		{SourceTable: "t1", SourceID: "s3", SourceType: "order", Title: "Recent Action", Status: ai.ActionStatusSuggested, UserID: int64Ptr(101), CreatedAt: now.Add(-30 * time.Minute)},
		{SourceTable: "t1", SourceID: "s4", SourceType: "order", Title: "Approved Old", Status: ai.ActionStatusApproved, UserID: int64Ptr(102), CreatedAt: now.Add(-3 * time.Hour)},
		{SourceTable: "t1", SourceID: "s5", SourceType: "order", Title: "Rejected Old", Status: ai.ActionStatusRejected, UserID: nil, CreatedAt: now.Add(-3 * time.Hour)},
	}
	for _, a := range actions {
		db.Create(&a)
	}

	if err := svc.SLAEscalation(); err != nil {
		t.Fatalf("SLAEscalation: %v", err)
	}

	// Verify: s1 and s2 should be escalated, others unchanged
	var results []ai.UnifiedAction
	db.Order("source_id ASC").Find(&results)
	if len(results) != 5 {
		t.Fatalf("expected 5 actions, got %d", len(results))
	}
	expected := []struct {
		sourceID string
		status   string
	}{
		{"s1", ai.ActionStatusEscalated},
		{"s2", ai.ActionStatusEscalated},
		{"s3", ai.ActionStatusSuggested},
		{"s4", ai.ActionStatusApproved},
		{"s5", ai.ActionStatusRejected},
	}
	for i, exp := range expected {
		if results[i].SourceID != exp.sourceID {
			t.Fatalf("result[%d].SourceID = %s, want %s", i, results[i].SourceID, exp.sourceID)
		}
		if results[i].Status != exp.status {
			t.Errorf("result[%d] (%s) status = %s, want %s", i, results[i].SourceID, results[i].Status, exp.status)
		}
	}
}

func int64Ptr(v int64) *int64 { return &v }
