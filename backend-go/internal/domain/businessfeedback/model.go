package businessfeedback

import (
	"encoding/json"
	"time"
)

type ControlledAction struct {
	ID                int64           `json:"id"`
	OwnerID           int64           `json:"owner_id"`
	OwnerDecisionID   int64           `json:"owner_decision_id"`
	ApprovalID        int64           `json:"approval_id"`
	CapabilityID      string          `json:"capability_id"`
	CommandType       string          `json:"command_type"`
	TargetType        string          `json:"target_type"`
	TargetID          string          `json:"target_id"`
	IdempotencyKey    string          `json:"idempotency_key"`
	InputPayload      json.RawMessage `gorm:"type:jsonb" json:"input_payload"`
	InputSHA256       string          `json:"input_sha256"`
	Status            string          `json:"status"`
	CommandBusinessID string          `json:"command_business_id,omitempty"`
	FailureMessage    string          `json:"failure_message,omitempty"`
	ExecutedAt        *time.Time      `json:"executed_at,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
}

func (ControlledAction) TableName() string { return "business_controlled_action" }

type ActionObservation struct {
	ID                   int64     `json:"id"`
	OwnerID              int64     `json:"owner_id"`
	ControlledActionID   int64     `json:"controlled_action_id"`
	SourceObjectID       int64     `json:"source_object_id"`
	EvidenceKind         string    `json:"evidence_kind"`
	TruthStatus          string    `json:"truth_status"`
	SourceObjectType     string    `json:"source_object_type"`
	SourceManifestSHA256 string    `json:"source_manifest_sha256"`
	ObservedAt           time.Time `json:"observed_at"`
	TargetMetric         string    `json:"target_metric"`
	TargetValue          string    `json:"target_value"`
	ActualValue          string    `json:"actual_value"`
	ComparisonNote       string    `json:"comparison_note"`
	CreatedAt            time.Time `json:"created_at"`
}

func (ActionObservation) TableName() string { return "business_action_observation" }

type NextActionRecommendation struct {
	ID                 int64     `json:"id"`
	OwnerID            int64     `json:"owner_id"`
	ControlledActionID int64     `json:"controlled_action_id"`
	RecommendationText string    `json:"recommendation_text"`
	Rationale          string    `json:"rationale"`
	TruthStatus        string    `json:"truth_status"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
}

func (NextActionRecommendation) TableName() string { return "business_next_action_recommendation" }

type CreateActionInput struct {
	OwnerDecisionID int64           `json:"owner_decision_id"`
	CapabilityID    string          `json:"capability_id"`
	CommandType     string          `json:"command_type"`
	TargetType      string          `json:"target_type"`
	TargetID        string          `json:"target_id"`
	ApprovalID      int64           `json:"approval_id"`
	IdempotencyKey  string          `json:"idempotency_key"`
	InputPayload    json.RawMessage `json:"input_payload"`
}

type CreateObservationInput struct {
	EvidenceKind     string `json:"evidence_kind"`
	SourceObjectType string `json:"source_object_type"`
	SourceObjectID   int64  `json:"source_object_id"`
	TargetMetric     string `json:"target_metric"`
	TargetValue      string `json:"target_value"`
	ActualValue      string `json:"actual_value"`
	ComparisonNote   string `json:"comparison_note"`
}

type CreateRecommendationInput struct {
	RecommendationText string `json:"recommendation_text"`
	Rationale          string `json:"rationale"`
}
type ActionDetail struct {
	Action              ControlledAction           `json:"action"`
	Observations        []ActionObservation        `json:"observations"`
	NextRecommendations []NextActionRecommendation `json:"next_recommendations"`
}
