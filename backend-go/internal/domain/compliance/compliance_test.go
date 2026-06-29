package compliance

import (
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestComplianceSaveResult(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CheckResult{})
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger)

	t.Run("create_new", func(t *testing.T) {
		now := time.Now()
		risk := RiskHigh
		result := &CheckResult{
			ProductID:  100,
			CheckType:  "compliance",
			Status:     StatusFail,
			RiskLevel:  &risk,
			ScannedAt:  now,
		}

		err := svc.SaveResult(result)
		if err != nil {
			t.Fatalf("SaveResult failed: %v", err)
		}
		if result.ID == 0 {
			t.Error("expected non-zero ID after create")
		}

		// Verify by fetching
		var saved CheckResult
		if err := db.First(&saved, result.ID).Error; err != nil {
			t.Fatalf("fetch saved result: %v", err)
		}
		if saved.ProductID != 100 {
			t.Errorf("got ProductID=%d, want 100", saved.ProductID)
		}
		if saved.Status != StatusFail {
			t.Errorf("got Status=%q, want %q", saved.Status, StatusFail)
		}
	})

	t.Run("idempotent_update", func(t *testing.T) {
		now := time.Now()
		risk := RiskLow
		result := &CheckResult{
			ProductID: 200,
			CheckType: "compliance",
			Status:    StatusPass,
			RiskLevel: &risk,
			ScannedAt: now,
		}

		// First save
		err := svc.SaveResult(result)
		if err != nil {
			t.Fatalf("first SaveResult failed: %v", err)
		}
		if result.ID == 0 {
			t.Fatal("expected non-zero ID")
		}

		// Update the status
		result.Status = StatusWarning
		err = svc.SaveResult(result)
		if err != nil {
			t.Fatalf("second SaveResult failed: %v", err)
		}

		// Verify update
		var saved CheckResult
		if err := db.First(&saved, result.ID).Error; err != nil {
			t.Fatalf("fetch updated result: %v", err)
		}
		if saved.Status != StatusWarning {
			t.Errorf("got Status=%q after update, want %q", saved.Status, StatusWarning)
		}
	})
}

func TestComplianceListResults(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CheckResult{})
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger)

	// Create results with various status and risk levels
	statuses := []string{StatusPass, StatusWarning, StatusFail, StatusPass}
	risks := []string{RiskLow, RiskMedium, RiskHigh, RiskLow}
	for i := 0; i < 4; i++ {
		risk := risks[i]
		result := &CheckResult{
			ProductID:  int64(300 + i),
			CheckType:  "compliance",
			Status:     statuses[i],
			RiskLevel:  &risk,
			ScannedAt:  time.Now(),
		}
		if err := svc.SaveResult(result); err != nil {
			t.Fatalf("setup SaveResult %d failed: %v", i, err)
		}
	}

	t.Run("list_all", func(t *testing.T) {
		p := &common.Pagination{Page: 1, Size: 100}
		items, total, err := svc.ListResults(p, "", "")
		if err != nil {
			t.Fatalf("ListResults failed: %v", err)
		}
		if total != 4 {
			t.Errorf("got total=%d, want 4", total)
		}
		if len(items) != 4 {
			t.Errorf("got %d items, want 4", len(items))
		}
	})

	t.Run("filter_by_status", func(t *testing.T) {
		p := &common.Pagination{Page: 1, Size: 100}
		items, total, err := svc.ListResults(p, StatusPass, "")
		if err != nil {
			t.Fatalf("ListResults with status filter failed: %v", err)
		}
		if total != 2 {
			t.Errorf("got total=%d for pass status, want 2", total)
		}
		if len(items) != 2 {
			t.Errorf("got %d items for pass status, want 2", len(items))
		}
		for _, item := range items {
			if item.Status != StatusPass {
				t.Errorf("expected Status=%q, got %q", StatusPass, item.Status)
			}
		}
	})

	t.Run("filter_by_risk", func(t *testing.T) {
		p := &common.Pagination{Page: 1, Size: 100}
		items, total, err := svc.ListResults(p, "", RiskLow)
		if err != nil {
			t.Fatalf("ListResults with risk filter failed: %v", err)
		}
		if total != 2 {
			t.Errorf("got total=%d for low risk, want 2", total)
		}
		if len(items) != 2 {
			t.Errorf("got %d items for low risk, want 2", len(items))
		}
		for _, item := range items {
			if item.RiskLevel == nil || *item.RiskLevel != RiskLow {
				t.Errorf("expected RiskLevel=%q, got %v", RiskLow, item.RiskLevel)
			}
		}
	})

	t.Run("paginated", func(t *testing.T) {
		p := &common.Pagination{Page: 1, Size: 2}
		items, total, err := svc.ListResults(p, "", "")
		if err != nil {
			t.Fatalf("ListResults paginated failed: %v", err)
		}
		if total != 4 {
			t.Errorf("got total=%d, want 4", total)
		}
		if len(items) != 2 {
			t.Errorf("got %d items, want 2 (page size)", len(items))
		}
	})
}

func TestComplianceSuppressResult(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CheckResult{})
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger)

	t.Run("suppress_existing", func(t *testing.T) {
		risk := RiskMedium
		result := &CheckResult{
			ProductID: 400,
			CheckType: "compliance",
			Status:    StatusWarning,
			RiskLevel: &risk,
			ScannedAt: time.Now(),
		}
		if err := svc.SaveResult(result); err != nil {
			t.Fatalf("setup SaveResult failed: %v", err)
		}

		reason := "false positive — product has valid certificate"
		if err := svc.SuppressResult(result.ID, reason); err != nil {
			t.Fatalf("SuppressResult failed: %v", err)
		}

		var saved CheckResult
		if err := db.First(&saved, result.ID).Error; err != nil {
			t.Fatalf("fetch suppressed result: %v", err)
		}
		if !saved.IsSuppressed {
			t.Error("expected IsSuppressed=true after suppress")
		}
		if saved.SuppressedReason == nil {
			t.Fatal("expected SuppressedReason to be set")
		}
		if *saved.SuppressedReason != reason {
			t.Errorf("got SuppressedReason=%q, want %q", *saved.SuppressedReason, reason)
		}
		if saved.SuppressedAt == nil {
			t.Error("expected SuppressedAt to be set")
		}
	})

	t.Run("suppress_not_found", func(t *testing.T) {
		err := svc.SuppressResult(99999, "not found")
		if err == nil {
			t.Fatal("expected error for non-existent result")
		}
	})
}

func TestComplianceEvidenceSerialization(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CheckResult{})
	logger := dbtest.NewLogger(t)
	svc := NewService(db, logger)

	risk := RiskHigh
	evidence := []EvidenceItem{
		{Rule: "risk_level", Field: "risk_level", Value: "high", Source: "A7ComplianceGuard"},
		{Rule: "certifications", Field: "required_certifications", Value: "CE, FCC", Source: "A7ComplianceGuard"},
	}
	result := &CheckResult{
		ProductID: 500,
		CheckType: "compliance",
		Status:    StatusFail,
		RiskLevel: &risk,
		Evidence:  evidence,
		ScannedAt: time.Now(),
	}

	if err := svc.SaveResult(result); err != nil {
		t.Fatalf("SaveResult failed: %v", err)
	}

	var saved CheckResult
	if err := db.First(&saved, result.ID).Error; err != nil {
		t.Fatalf("fetch saved result: %v", err)
	}
	if len(saved.Evidence) != 2 {
		t.Fatalf("got %d evidence items, want 2", len(saved.Evidence))
	}
	if saved.Evidence[0].Rule != "risk_level" {
		t.Errorf("got evidence[0].Rule=%q, want %q", saved.Evidence[0].Rule, "risk_level")
	}
}
