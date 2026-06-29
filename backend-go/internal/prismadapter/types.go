package prismadapter

import "context"

// PrismService defines the interface for interacting with the Prism image engine.
type PrismService interface {
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
}

// GenerateRequest is the payload sent to Prism for image generation.
type GenerateRequest struct {
	ImageURL  string `json:"image_url"`
	Platform  string `json:"platform"`
	ProductID int64  `json:"product_id"`
}

// GenerateResponse is the result from Prism after image generation and compliance check.
type GenerateResponse struct {
	JobID            string           `json:"job_id"`
	OutputURL        string           `json:"output_url"`
	ComplianceReport ComplianceReport `json:"compliance_report"`
	RiskScore        float64          `json:"risk_score"`
	FailureReasons   []string         `json:"failure_reasons,omitempty"`
}

// ComplianceReport describes the compliance status of a generated image.
type ComplianceReport struct {
	Status  string   `json:"status"` // "pass", "warning", "fail"
	Reasons []string `json:"reasons,omitempty"`
}

// ComplianceStatus constants.
const (
	StatusPass    = "pass"
	StatusWarning = "warning"
	StatusFail    = "fail"
)
