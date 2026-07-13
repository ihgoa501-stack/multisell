package businessdecision

import (
	"encoding/json"
	"time"
)

const (
	DecisionSelected     = "selected"
	DecisionRejected     = "rejected"
	DecisionPaused       = "paused"
	DecisionMoreEvidence = "request_more_evidence"
)

type Case struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	OwnerID        int64     `gorm:"not null;index" json:"owner_id"`
	Question       string    `gorm:"type:text;not null" json:"question"`
	Target         string    `gorm:"type:text;not null" json:"target"`
	ObjectType     string    `gorm:"size:64;not null" json:"object_type"`
	ObjectID       int64     `gorm:"not null" json:"object_id"`
	TruthStatus    string    `gorm:"size:24;not null" json:"truth_status"`
	UnknownsJSON   string    `gorm:"type:text;not null" json:"-"`
	ManifestSHA256 string    `gorm:"size:64;not null" json:"manifest_sha256"`
	IdempotencyKey string    `gorm:"size:128;not null;uniqueIndex:ux_business_case_owner_key" json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	Unknowns       []string  `gorm:"-" json:"unknowns"`
}

func (Case) TableName() string { return "business_decision_case" }

type FactSnapshot struct {
	ID               int64     `gorm:"primaryKey" json:"id"`
	DecisionCaseID   int64     `gorm:"not null;index" json:"decision_case_id"`
	OwnerID          int64     `gorm:"not null;index" json:"owner_id"`
	ObjectType       string    `gorm:"size:64;not null" json:"object_type"`
	ObjectID         int64     `gorm:"not null" json:"object_id"`
	TruthStatus      string    `gorm:"size:24;not null" json:"truth_status"`
	SourceTable      string    `gorm:"size:96;not null" json:"source_table"`
	SourceObservedAt time.Time `gorm:"not null" json:"source_observed_at"`
	PayloadJSON      string    `gorm:"type:text;not null" json:"payload_json"`
	PayloadSHA256    string    `gorm:"size:64;not null" json:"payload_sha256"`
	CreatedAt        time.Time `json:"created_at"`
}

func (FactSnapshot) TableName() string { return "business_decision_fact_snapshot" }

type AIRecommendation struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	DecisionCaseID int64     `gorm:"not null;index" json:"decision_case_id"`
	OwnerID        int64     `gorm:"not null;index" json:"owner_id"`
	Recommendation string    `gorm:"type:text;not null" json:"recommendation"`
	Rationale      string    `gorm:"type:text;not null" json:"rationale"`
	TruthStatus    string    `gorm:"size:24;not null" json:"truth_status"`
	UnknownsJSON   string    `gorm:"type:text;not null" json:"-"`
	ManifestSHA256 string    `gorm:"size:64;not null" json:"manifest_sha256"`
	IdempotencyKey string    `gorm:"size:128;not null;uniqueIndex:ux_business_ai_owner_key" json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	Unknowns       []string  `gorm:"-" json:"unknowns"`
}

func (AIRecommendation) TableName() string { return "business_ai_recommendation" }

type OwnerDecision struct {
	ID               int64           `gorm:"primaryKey" json:"id"`
	DecisionCaseID   int64           `gorm:"not null;index" json:"decision_case_id"`
	OwnerID          int64           `gorm:"not null;index" json:"owner_id"`
	RecommendationID *int64          `json:"recommendation_id,omitempty"`
	Decision         string          `gorm:"size:32;not null" json:"decision"`
	CapabilityID     string          `gorm:"size:120;not null" json:"capability_id,omitempty"`
	CommandType      string          `gorm:"size:120;not null" json:"command_type,omitempty"`
	TargetType       string          `gorm:"size:80;not null" json:"target_type,omitempty"`
	TargetID         string          `gorm:"size:160;not null" json:"target_id,omitempty"`
	InputSHA256      string          `gorm:"size:64;not null" json:"input_sha256,omitempty"`
	InputPayload     json.RawMessage `gorm:"type:jsonb;not null" json:"input_payload,omitempty"`
	Reason           string          `gorm:"type:text;not null" json:"reason"`
	ManifestSHA256   string          `gorm:"size:64;not null" json:"manifest_sha256"`
	IdempotencyKey   string          `gorm:"size:128;not null;uniqueIndex:ux_business_owner_decision_key" json:"-"`
	CreatedAt        time.Time       `json:"created_at"`
}

func (OwnerDecision) TableName() string { return "business_owner_decision" }

type CreateCaseInput struct {
	Question       string   `json:"question"`
	Target         string   `json:"target"`
	ObjectType     string   `json:"object_type"`
	ObjectID       int64    `json:"object_id"`
	Unknowns       []string `json:"unknowns"`
	IdempotencyKey string   `json:"idempotency_key"`
}
type RecommendInput struct {
	Recommendation string   `json:"recommendation"`
	Rationale      string   `json:"rationale"`
	TruthStatus    string   `json:"truth_status"`
	Unknowns       []string `json:"unknowns"`
	IdempotencyKey string   `json:"idempotency_key"`
}
type DecideInput struct {
	RecommendationID *int64          `json:"recommendation_id"`
	Decision         string          `json:"decision"`
	CapabilityID     string          `json:"capability_id"`
	CommandType      string          `json:"command_type"`
	TargetType       string          `json:"target_type"`
	TargetID         string          `json:"target_id"`
	InputSHA256      string          `json:"input_sha256"`
	InputPayload     json.RawMessage `json:"input_payload"`
	Reason           string          `json:"reason"`
	ManifestSHA256   string          `json:"manifest_sha256"`
	IdempotencyKey   string          `json:"idempotency_key"`
}
type Detail struct {
	Case            Case               `json:"case"`
	Snapshot        FactSnapshot       `json:"fact_snapshot"`
	Recommendations []AIRecommendation `json:"ai_recommendations"`
	Decisions       []OwnerDecision    `json:"owner_decisions"`
}

type ListItem struct {
	Case
	LatestDecision *OwnerDecision `json:"latest_owner_decision,omitempty"`
}
type FactOption struct {
	ObjectType  string    `json:"object_type"`
	ObjectID    int64     `json:"object_id"`
	Label       string    `json:"label"`
	TruthStatus string    `json:"truth_status"`
	ObservedAt  time.Time `json:"observed_at"`
}
