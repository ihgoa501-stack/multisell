package demandcase

import "time"

const (
	VerdictLead            = "lead"
	VerdictEvidenceMissing = "evidence_missing"
	VerdictRejected        = "rejected"
	VerdictExperimentReady = "experiment_ready"
	EvidenceSupport        = "support"
	EvidenceCounter        = "counter"
	EvidenceConflict       = "conflict"
	TruthActual            = "actual"
	TruthQuoted            = "quoted"
	TruthEstimated         = "estimated"
	TruthUnknown           = "unknown"
	TruthMock              = "mock"
	TruthInferred          = "inferred"
	DimensionDemand        = "demand"
	DimensionCompetition   = "competition"
	DimensionAcquisition   = "acquisition"
	DimensionFulfillment   = "fulfillment"
	DimensionCompliance    = "compliance"
	DimensionPayment       = "payment"
	DimensionAftersales    = "aftersales"
	DimensionProfit        = "profit_verifiability"
)

var RequiredDimensions = []string{DimensionDemand, DimensionCompetition, DimensionAcquisition, DimensionFulfillment, DimensionCompliance, DimensionPayment, DimensionAftersales, DimensionProfit}

type DemandCase struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	OwnerID       int64     `gorm:"index;not null" json:"owner_id"`
	Region        string    `gorm:"size:80;not null" json:"region"`
	Consumer      string    `gorm:"size:240;not null" json:"consumer"`
	NeedScenario  string    `gorm:"size:400;not null" json:"need_scenario"`
	SalesChannel  string    `gorm:"size:160;not null" json:"sales_channel"`
	StopCondition string    `gorm:"type:text" json:"stop_condition"`
	Status        string    `gorm:"size:32;not null;default:lead" json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (DemandCase) TableName() string { return "demand_case" }

type DemandEvidence struct {
	ID           int64      `gorm:"primaryKey" json:"id"`
	DemandCaseID int64      `gorm:"index;not null" json:"demand_case_id"`
	Dimension    string     `gorm:"size:40;not null" json:"dimension"`
	Kind         string     `gorm:"size:16;not null" json:"kind"`
	TruthStatus  string     `gorm:"size:16;not null" json:"truth_status"`
	Title        string     `gorm:"type:text;not null" json:"title"`
	SourceURI    string     `gorm:"type:text" json:"source_uri"`
	ObservedAt   *time.Time `json:"observed_at"`
	RunID        string     `gorm:"size:80;not null" json:"run_id"`
	SnapshotID   int64      `gorm:"index;not null;default:0" json:"snapshot_id"`
	Fatal        bool       `json:"fatal"`
	CreatedAt    time.Time  `json:"created_at"`
}

type ResearchBatch struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	BatchKey  string    `gorm:"uniqueIndex:ux_owner_batch;size:100;not null" json:"batch_key"`
	OwnerID   int64     `gorm:"uniqueIndex:ux_owner_batch;not null" json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (ResearchBatch) TableName() string { return "demand_research_batch" }

type ResearchSnapshot struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	BatchID      int64     `gorm:"index;not null" json:"batch_id"`
	OwnerID      int64     `gorm:"uniqueIndex:ux_owner_research_run;not null" json:"-"`
	DemandCaseID int64     `gorm:"index;not null" json:"demand_case_id"`
	RunID        string    `gorm:"uniqueIndex:ux_owner_research_run;size:80;not null" json:"run_id"`
	RunType      string    `gorm:"uniqueIndex:ux_owner_research_run;size:24;not null" json:"run_type"`
	Collector    string    `gorm:"size:120;not null" json:"collector"`
	SourceURI    string    `gorm:"type:text;not null" json:"source_uri"`
	CollectedAt  time.Time `json:"collected_at"`
	RawPayload   string    `gorm:"type:text;not null" json:"-"`
	RawSHA256    string    `gorm:"size:64;not null" json:"raw_sha256"`
	CreatedAt    time.Time `json:"created_at"`
}

func (ResearchSnapshot) TableName() string { return "demand_research_snapshot" }

func (DemandEvidence) TableName() string { return "demand_evidence" }

type DemandVerdict struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	DemandCaseID int64     `gorm:"index;not null" json:"demand_case_id"`
	Status       string    `gorm:"size:32;not null" json:"status"`
	BlockersJSON string    `gorm:"type:text" json:"-"`
	Reason       string    `gorm:"type:text" json:"reason"`
	EvaluatedBy  int64     `gorm:"not null" json:"evaluated_by"`
	CreatedAt    time.Time `json:"created_at"`
	Blockers     []string  `gorm:"-" json:"blockers"`
}

func (DemandVerdict) TableName() string { return "demand_verdict" }

type Detail struct {
	Case       DemandCase         `json:"case"`
	Evidence   []DemandEvidence   `json:"evidence"`
	Verdict    *DemandVerdict     `json:"verdict,omitempty"`
	Snapshots  []ResearchSnapshot `json:"snapshots"`
	DataAccess []DataAccessRecord `json:"data_access"`
}

type OwnerDecisionCard struct {
	DemandCaseID             int64  `json:"demand_case_id"`
	Verdict                  string `json:"verdict"`
	Hypothesis               string `json:"hypothesis"`
	Proven                   string `json:"proven"`
	NotProven                string `json:"not_proven"`
	StrongestCounterevidence string `json:"strongest_counterevidence"`
	NextAuthorityOrCost      string `json:"next_authority_or_cost"`
	StopCondition            string `json:"stop_condition"`
}
