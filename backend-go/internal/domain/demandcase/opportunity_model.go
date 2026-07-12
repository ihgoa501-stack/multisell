package demandcase

import "time"

const (
	MarketDecisionSelected     = "selected"
	MarketDecisionRejected     = "rejected"
	MarketDecisionPaused       = "paused"
	MarketDecisionMoreEvidence = "request_more_evidence"
	OpportunityDraft           = "draft"
	OpportunityEvidenceMissing = "evidence_missing"
	OpportunityReady           = "ready_for_owner"
	OpportunityApproved        = "approved"
	OpportunityRejected        = "rejected"
	OpportunityPaused          = "paused"
)

// MarketOwnerDecision is an immutable Owner decision based on one frozen verdict.
type MarketOwnerDecision struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	DemandCaseID   int64     `gorm:"index;not null" json:"demand_case_id"`
	OwnerID        int64     `gorm:"index;not null" json:"owner_id"`
	VerdictID      int64     `gorm:"not null" json:"verdict_id"`
	Decision       string    `gorm:"size:32;not null" json:"decision"`
	Reason         string    `gorm:"type:text;not null" json:"reason"`
	EvidenceHash   string    `gorm:"size:64;not null" json:"evidence_hash"`
	IdempotencyKey string    `gorm:"size:128;not null;uniqueIndex:ux_market_decision_owner_key" json:"-"`
	CreatedAt      time.Time `json:"created_at"`
}

func (MarketOwnerDecision) TableName() string { return "market_owner_decision" }

type ProductOpportunity struct {
	ID                       int64     `gorm:"primaryKey" json:"id"`
	OwnerID                  int64     `gorm:"index;not null" json:"owner_id"`
	DemandCaseID             int64     `gorm:"index;not null" json:"demand_case_id"`
	MarketDecisionID         int64     `gorm:"not null" json:"market_decision_id"`
	Title                    string    `gorm:"size:240;not null" json:"title"`
	ConsumerProblem          string    `gorm:"type:text;not null" json:"consumer_problem"`
	ProductThesis            string    `gorm:"type:text;not null" json:"product_thesis"`
	TargetChannel            string    `gorm:"size:160;not null" json:"target_channel"`
	ValueHypothesis          string    `gorm:"type:text;not null" json:"value_hypothesis"`
	PriceHypothesis          string    `gorm:"type:text;not null" json:"price_hypothesis"`
	SourceURI                string    `gorm:"type:text;not null" json:"source_uri"`
	TruthStatus              string    `gorm:"size:16;not null" json:"truth_status"`
	StrongestCounterevidence string    `gorm:"type:text;not null" json:"strongest_counterevidence"`
	UnknownsJSON             string    `gorm:"type:text;not null" json:"-"`
	StopCondition            string    `gorm:"type:text;not null" json:"stop_condition"`
	Status                   string    `gorm:"size:32;not null" json:"status"`
	Version                  int64     `gorm:"not null;default:1" json:"version"`
	ContentHash              string    `gorm:"size:64;not null" json:"content_hash"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
	Unknowns                 []string  `gorm:"-" json:"unknowns"`
}

func (ProductOpportunity) TableName() string { return "product_opportunity" }

type ProductOpportunityDecision struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	OpportunityID  int64     `gorm:"index;not null" json:"opportunity_id"`
	OwnerID        int64     `gorm:"index;not null" json:"owner_id"`
	Version        int64     `gorm:"not null" json:"version"`
	ContentHash    string    `gorm:"size:64;not null" json:"content_hash"`
	Decision       string    `gorm:"size:16;not null" json:"decision"`
	Reason         string    `gorm:"type:text;not null" json:"reason"`
	IdempotencyKey string    `gorm:"size:128;not null;uniqueIndex:ux_opportunity_decision_owner_key" json:"-"`
	CreatedAt      time.Time `json:"created_at"`
}

func (ProductOpportunityDecision) TableName() string { return "product_opportunity_decision" }

type MarketDecisionInput struct {
	Decision       string `json:"decision"`
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
}
type ProductOpportunityInput struct {
	DemandCaseID             int64    `json:"demand_case_id"`
	MarketDecisionID         int64    `json:"market_decision_id"`
	Title                    string   `json:"title"`
	ConsumerProblem          string   `json:"consumer_problem"`
	ProductThesis            string   `json:"product_thesis"`
	TargetChannel            string   `json:"target_channel"`
	ValueHypothesis          string   `json:"value_hypothesis"`
	PriceHypothesis          string   `json:"price_hypothesis"`
	SourceURI                string   `json:"source_uri"`
	TruthStatus              string   `json:"truth_status"`
	StrongestCounterevidence string   `json:"strongest_counterevidence"`
	StopCondition            string   `json:"stop_condition"`
	Unknowns                 []string `json:"unknowns"`
}
type OpportunityDecisionInput struct {
	Decision       string `json:"decision"`
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
	Version        int64  `json:"version"`
}
