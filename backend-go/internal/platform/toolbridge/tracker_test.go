package toolbridge

import (
	"fmt"
	"testing"
)

func TestExternalCallTracker_RecordsSuccess(t *testing.T) {
	tr := NewExternalCallTracker(3)
	tr.RecordCall("shopee", nil)
	tr.RecordCall("shopee", nil)
	stats := tr.Stats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 platform, got %d", len(stats))
	}
	if stats[0].TotalCalls != 2 || stats[0].FailedCalls != 0 {
		t.Errorf("expected 2 total, 0 failed; got %d/%d", stats[0].TotalCalls, stats[0].FailedCalls)
	}
	if stats[0].Degraded {
		t.Error("should not be degraded after successes")
	}
}

func TestExternalCallTracker_DegradedAfterThreshold(t *testing.T) {
	tr := NewExternalCallTracker(3)
	for i := 0; i < 3; i++ {
		tr.RecordCall("ozon", fmt.Errorf("timeout #%d", i+1))
	}
	if !tr.IsDegraded("ozon") {
		t.Error("expected ozon to be degraded after 3 consecutive failures")
	}
	// Recovery
	tr.RecordCall("ozon", nil)
	if tr.IsDegraded("ozon") {
		t.Error("expected ozon to recover after success")
	}
}

func TestExternalCallTracker_StatsFormat(t *testing.T) {
	tr := NewExternalCallTracker(3)
	tr.RecordCall("shopee", fmt.Errorf("auth failed"))
	stats := tr.Stats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 platform")
	}
	if stats[0].LastError != "auth failed" {
		t.Errorf("expected auth failed, got %q", stats[0].LastError)
	}
	if stats[0].LastFailureAt == "" {
		t.Error("expected last_failure_at to be set")
	}
}
