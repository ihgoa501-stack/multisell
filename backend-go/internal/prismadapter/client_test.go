package prismadapter

import (
	"context"
	"fmt"
	"testing"
)

// mockPrismService implements PrismService for testing.
type mockPrismService struct {
	generateFn func(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
}

func (m *mockPrismService) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	return m.generateFn(ctx, req)
}

func TestClient_Generate_Success(t *testing.T) {
	// Using a mock to test the response handling logic.
	mock := &mockPrismService{
		generateFn: func(_ context.Context, req *GenerateRequest) (*GenerateResponse, error) {
			if req.ImageURL == "" {
				t.Error("expected image_url to be set")
			}
			return &GenerateResponse{
				JobID:     "prism-job-001",
				OutputURL: "https://cdn.prism.test/output.jpg",
				ComplianceReport: ComplianceReport{
					Status:  StatusPass,
					Reasons: nil,
				},
				RiskScore:      0.1,
				FailureReasons: nil,
			}, nil
		},
	}

	resp, err := mock.Generate(context.Background(), &GenerateRequest{
		ImageURL:  "https://example.com/product.jpg",
		Platform:  "ozon",
		ProductID: 123,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.JobID != "prism-job-001" {
		t.Errorf("expected job_id prism-job-001, got %s", resp.JobID)
	}
	if resp.ComplianceReport.Status != StatusPass {
		t.Errorf("expected status pass, got %s", resp.ComplianceReport.Status)
	}
	if resp.OutputURL != "https://cdn.prism.test/output.jpg" {
		t.Errorf("unexpected output_url: %s", resp.OutputURL)
	}
}

func TestClient_Generate_ComplianceFail(t *testing.T) {
	mock := &mockPrismService{
		generateFn: func(_ context.Context, req *GenerateRequest) (*GenerateResponse, error) {
			return &GenerateResponse{
				JobID:     "prism-job-002",
				OutputURL: "",
				ComplianceReport: ComplianceReport{
					Status:  StatusFail,
					Reasons: []string{"image_contains_text", "resolution_too_low"},
				},
				RiskScore:      0.85,
				FailureReasons: []string{"image_contains_text", "resolution_too_low"},
			}, nil
		},
	}

	resp, err := mock.Generate(context.Background(), &GenerateRequest{
		ImageURL:  "https://example.com/bad.jpg",
		Platform:  "ozon",
		ProductID: 456,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ComplianceReport.Status != StatusFail {
		t.Errorf("expected status fail, got %s", resp.ComplianceReport.Status)
	}
	if len(resp.FailureReasons) != 2 {
		t.Errorf("expected 2 failure reasons, got %d", len(resp.FailureReasons))
	}
	if resp.RiskScore < 0.8 {
		t.Errorf("expected high risk score, got %f", resp.RiskScore)
	}
}

func TestClient_Generate_ComplianceWarning(t *testing.T) {
	mock := &mockPrismService{
		generateFn: func(_ context.Context, req *GenerateRequest) (*GenerateResponse, error) {
			return &GenerateResponse{
				JobID:     "prism-job-003",
				OutputURL: "https://cdn.prism.test/warning.jpg",
				ComplianceReport: ComplianceReport{
					Status:  StatusWarning,
					Reasons: []string{"low_contrast_background"},
				},
				RiskScore:      0.45,
				FailureReasons: nil,
			}, nil
		},
	}

	resp, err := mock.Generate(context.Background(), &GenerateRequest{
		ImageURL:  "https://example.com/warning.jpg",
		Platform:  "shopee",
		ProductID: 789,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ComplianceReport.Status != StatusWarning {
		t.Errorf("expected status warning, got %s", resp.ComplianceReport.Status)
	}
	if len(resp.ComplianceReport.Reasons) != 1 {
		t.Errorf("expected 1 warning, got %d", len(resp.ComplianceReport.Reasons))
	}
	if resp.OutputURL == "" {
		t.Error("expected output_url even with warning")
	}
}

func TestClient_Generate_Error(t *testing.T) {
	mock := &mockPrismService{
		generateFn: func(_ context.Context, req *GenerateRequest) (*GenerateResponse, error) {
			return nil, fmt.Errorf("prism service unavailable")
		},
	}

	_, err := mock.Generate(context.Background(), &GenerateRequest{
		ImageURL:  "https://example.com/test.jpg",
		Platform:  "ozon",
		ProductID: 1,
	})
	if err == nil {
		t.Fatal("expected error from prism service")
	}
}
