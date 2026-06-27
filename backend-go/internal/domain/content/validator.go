package content

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lingmirror/backend-go/internal/ai"
	"go.uber.org/zap"
)

// ContentReview is the result of LLM-as-Judge validation.
type ContentReview struct {
	Passed             bool     `json:"passed"`
	Issues             []string `json:"issues,omitempty"`
	AdjustedConfidence float64  `json:"adjusted_confidence"`
}

// HasIssues returns true if there are any validation issues.
func (r *ContentReview) HasIssues() bool {
	return len(r.Issues) > 0
}

// ContentValidator validates generated content using LLM-as-Judge.
type ContentValidator struct {
	aiOrch *ai.Orchestrator
	logger *zap.Logger
}

// NewContentValidator creates a new ContentValidator.
func NewContentValidator(aiOrch *ai.Orchestrator, logger *zap.Logger) *ContentValidator {
	return &ContentValidator{aiOrch: aiOrch, logger: logger}
}

// Validate performs multi-layer validation on generated content.
func (v *ContentValidator) Validate(content *GeneratedContent, in *ValidateRequest) (*ContentReview, error) {
	review := &ContentReview{Passed: true, AdjustedConfidence: content.Confidence}

	// 1. Length checks per platform.
	maxTitle := 200
	if in.Platform == "ozon" && in.Language == "ru" {
		maxTitle = 100
	}
	if len([]rune(content.Title)) > maxTitle {
		review.Issues = append(review.Issues, fmt.Sprintf("标题过长: %d字符 (限制%d)", len([]rune(content.Title)), maxTitle))
	} else if len([]rune(content.Title)) < 10 && content.Title != "" {
		review.Issues = append(review.Issues, "标题过短")
	}

	// 2. Banned word check.
	banned := []string{"лучший", "best", "#1", "100%", "guaranteed"}
	lowerTitle := strings.ToLower(content.Title)
	lowerDesc := strings.ToLower(content.Description)
	for _, w := range banned {
		if strings.Contains(lowerTitle, w) || strings.Contains(lowerDesc, w) {
			review.Issues = append(review.Issues, "禁用词: "+w)
		}
	}

	// 3. LLM review for semantic issues.
	reviewPrompt := fmt.Sprintf(`Review this product content for issues:
Title: %s
Description: %s
Language: %s
Platform: %s

Check for:
- Hallucinated facts or claims
- Inappropriate or culturally insensitive content
- Contradictions or nonsense
- Missing key information

Reply with JSON: {"has_issues": true/false, "issues": ["issue1"], "adjusted_confidence": 0.0-1.0}`,
		content.Title, content.Description, in.Language, in.Platform)

	resp, err := v.aiOrch.Run(&ai.RunAgentRequest{
		AgentID:       "content_validator",
		DecisionPoint: "validate_content",
		Context:       map[string]interface{}{"prompt": reviewPrompt, "content": content},
	})
	if err == nil {
		// Parse review result from the output.
		hasIssues, _ := resp.Output["has_issues"].(bool)
		if hasIssues {
			review.Passed = false
			if issues, ok := resp.Output["issues"].([]interface{}); ok {
				for _, i := range issues {
					review.Issues = append(review.Issues, fmt.Sprintf("%v", i))
				}
			}
			if conf, ok := resp.Output["adjusted_confidence"].(float64); ok {
				review.AdjustedConfidence = conf
			} else {
				review.AdjustedConfidence = content.Confidence * 0.8 // reduce if can't parse
			}
		}
	} else {
		// LLM review failed; adjust confidence down as a safety measure.
		v.logger.Warn("LLM review failed for content", zap.Error(err))
		review.AdjustedConfidence = content.Confidence * 0.85
	}

	if !review.Passed {
		review.AdjustedConfidence = content.Confidence * 0.7
	}

	// Marshal the review for consistent output from the validator.
	if raw, err := json.Marshal(resp.Output); err == nil {
		var llmReview ContentReview
		if json.Unmarshal(raw, &llmReview) == nil {
			if llmReview.Passed == false || len(llmReview.Issues) > 0 {
				review.Passed = llmReview.Passed
				review.Issues = append(review.Issues, llmReview.Issues...)
				if llmReview.AdjustedConfidence > 0 {
					review.AdjustedConfidence = llmReview.AdjustedConfidence
				}
			}
		}
	}

	return review, nil
}
