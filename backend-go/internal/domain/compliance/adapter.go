package compliance

import (
	"context"
	"fmt"

	"github.com/lingmirror/backend-go/internal/agent/impl"
)

// CheckInput holds the parameters for a compliance check.
type CheckInput struct {
	ProductName string `json:"product_name"`
	Category    string `json:"category"`
	Country     string `json:"country"`
	Platform    string `json:"platform"`
}

// CheckOutput holds the result of a compliance check.
type CheckOutput struct {
	Certifications   []string `json:"certifications"`
	Restrictions     []string `json:"restrictions"`
	RiskLevel        string   `json:"risk_level"`
	BlockedPlatforms []string `json:"blocked_platforms"`
	Confidence       float64  `json:"confidence"`
}

// A7Adapter wraps the A7 ComplianceGuardAgent for domain-level use.
type A7Adapter struct {
	agent *impl.ComplianceGuardAgent
}

// NewA7Adapter creates a new A7Adapter.
func NewA7Adapter() *A7Adapter {
	return &A7Adapter{
		agent: impl.NewComplianceGuardAgent(),
	}
}

// RunCheck performs a compliance check via the A7 agent and returns the
// structured output. The second return value is the agent's confidence
// score (0.0–1.0).
func (a *A7Adapter) RunCheck(input *CheckInput) (*CheckOutput, float64, error) {
	ctx := map[string]interface{}{
		"product_name":    input.ProductName,
		"category":        input.Category,
		"target_country":  input.Country,
		"target_platform": input.Platform,
	}

	outputMap, confidence, _, err := a.agent.Decide(context.Background(), "compliance_check", ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("compliance check failed: %w", err)
	}

	out := &CheckOutput{}

	if certs, ok := outputMap["required_certifications"].([]string); ok {
		out.Certifications = certs
	}
	if restrictions, ok := outputMap["restrictions"].([]string); ok {
		out.Restrictions = restrictions
	}
	if risk, ok := outputMap["risk_level"].(string); ok {
		out.RiskLevel = risk
	}
	if blocked, ok := outputMap["blocked_platforms"].([]string); ok {
		out.BlockedPlatforms = blocked
	}

	out.Confidence = confidence
	return out, confidence, nil
}
