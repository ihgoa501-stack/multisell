package feedback

import (
	"encoding/json"
)

// TriageActionData holds the data needed to create a triage UnifiedAction.
type TriageActionData struct {
	SubmissionID int64
	Title        string
	FeedbackType string
	Priority     int
	Status       string
	Severity     string
	Confidence   float64
}

// TriageDecision is the result of auto-triage analysis.
type TriageDecision struct {
	Action          string  // "accept" | "reject" | "needs_review"
	RequiresApproval bool
	Reason          string
	AssignedTo      *int64
}

// TriageHandler is a callback that creates an AgentOS UnifiedAction for a new feedback submission.
// If nil, the integration is disabled (feedback works standalone).
type TriageHandler func(data *TriageActionData) error

// AutoTriage analyzes a submission and decides what should happen to it.
// This is the "要不要做" decision logic.
func AutoTriage(feedbackType, severity string, priority int, confidence float64) *TriageDecision {
	// Critical bugs → auto-accept, high urgency
	if severity == "critical" || (feedbackType == TypeBug && severity == "major") {
		return &TriageDecision{
			Action:           "accept",
			RequiresApproval: false,
			Reason:           "自动采纳：严重Bug需要立即处理",
		}
	}

	// High confidence + high priority → auto-accept with approval
	if confidence >= 0.8 && priority >= 50 {
		return &TriageDecision{
			Action:           "accept",
			RequiresApproval: false,
			Reason:           "自动采纳：高置信度高优先级反馈",
		}
	}

	// Medium confidence + medium priority → needs human review
	if confidence >= 0.5 {
		return &TriageDecision{
			Action:           "needs_review",
			RequiresApproval: true,
			Reason:           "需人工审阅：AI置信度一般",
		}
	}

	// Low confidence → definitely needs human review
	return &TriageDecision{
		Action:           "needs_review",
		RequiresApproval: true,
		Reason:           "需人工审阅：AI置信度较低，建议人工判断",
	}
}

// BuildTriagePayload creates the JSON payload for a UnifiedAction from a submission.
func BuildTriagePayload(sub *Submission) []byte {
	payload := map[string]interface{}{
		"submission_id": sub.ID,
		"title":         sub.Title,
		"description":   sub.Description,
		"feedback_type": sub.FeedbackType,
		"severity":      sub.Severity,
		"priority":      sub.Priority,
		"status":        sub.Status,
	}
	data, _ := json.Marshal(payload)
	return data
}
