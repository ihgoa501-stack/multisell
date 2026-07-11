package experiment

import "time"

const (
	StageOpportunity  = "opportunity"
	StageProduct      = "product"
	StageSupply       = "supply"
	StageChannel      = "channel"
	StageOrder        = "order"
	StageFulfillment  = "fulfillment"
	StageAftersales   = "aftersales"
	StageProfit       = "profit"
	StageCash         = "cash"
	StageDecision     = "decision"
	ResultPass        = "pass"
	ResultConditional = "conditional"
	ResultReturn      = "return"
	ResultReject      = "reject"
	ResultExpired     = "expired"
	TruthActual       = "actual"
	TruthQuoted       = "quoted"
	TruthEstimated    = "estimated"
	TruthUnknown      = "unknown"
	TruthMock         = "mock"
	TruthInferred     = "inferred"
	StatusActive      = "active"
	StatusBlocked     = "blocked"
	StatusCompleted   = "completed"
	StatusStopped     = "stopped"
	ProfitPending     = "pending"
	ProfitProvisional = "provisional"
	ProfitFinal       = "final"
	CashPending       = "pending"
	CashReceivable    = "receivable"
	CashSettled       = "settled"
	CashRecovered     = "recovered"
)

type ExperimentCase struct {
	ID                  int64      `gorm:"primaryKey" json:"id"`
	ExperimentID        string     `gorm:"uniqueIndex;size:40;not null" json:"experiment_id"`
	Name                string     `gorm:"not null" json:"name"`
	Stage               string     `gorm:"size:24;not null" json:"stage"`
	Status              string     `gorm:"size:24;default:active" json:"status"`
	FinalProfitStatus   string     `gorm:"size:24;default:pending" json:"final_profit_status"`
	FinalRevenue        float64    `json:"final_revenue"`
	FinalTotalCost      float64    `json:"final_total_cost"`
	FinalProfitAmount   float64    `json:"final_profit_amount"`
	ProfitCurrency      string     `gorm:"size:8" json:"profit_currency"`
	CashRecoveryStatus  string     `gorm:"size:24;default:pending" json:"cash_recovery_status"`
	CashRecoveredAmount float64    `json:"cash_recovered_amount"`
	CashCurrency        string     `gorm:"size:8" json:"cash_currency"`
	CashRecoveredAt     *time.Time `json:"cash_recovered_at"`
	FinalDecision       string     `gorm:"size:16" json:"final_decision"`
	OwnerID             int64      `json:"owner_id"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func (ExperimentCase) TableName() string { return "experiment_case" }

type GateDecision struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	ExperimentID string    `gorm:"index;size:40;not null" json:"experiment_id"`
	Stage        string    `gorm:"size:24;not null" json:"stage"`
	GateCode     string    `gorm:"size:80;not null" json:"gate_code"`
	Result       string    `gorm:"size:16;not null" json:"result"`
	Reason       string    `json:"reason"`
	EvidenceIDs  string    `gorm:"type:text" json:"-"`
	DecidedBy    int64     `json:"decided_by"`
	CreatedAt    time.Time `json:"created_at"`
}

func (GateDecision) TableName() string { return "experiment_gate_decision" }

type EvidenceRecord struct {
	ID           int64      `gorm:"primaryKey" json:"id"`
	ExperimentID string     `gorm:"index;size:40;not null" json:"experiment_id"`
	Stage        string     `gorm:"size:24;not null" json:"stage"`
	EvidenceKind string     `gorm:"size:16;not null;default:support" json:"evidence_kind"`
	TruthStatus  string     `gorm:"size:16;not null" json:"truth_status"`
	Title        string     `gorm:"not null" json:"title"`
	SourceURI    string     `json:"source_uri"`
	ObservedAt   *time.Time `json:"observed_at"`
	ExpiresAt    *time.Time `json:"expires_at"`
	VerifiedBy   int64      `json:"verified_by"`
	VerifiedAt   *time.Time `json:"verified_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (EvidenceRecord) TableName() string { return "experiment_evidence" }

type ObjectLink struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	ExperimentID string    `gorm:"uniqueIndex:ux_exp_object;size:40;not null" json:"experiment_id"`
	ObjectType   string    `gorm:"uniqueIndex:ux_exp_object;size:40;not null" json:"object_type"`
	ObjectID     string    `gorm:"uniqueIndex:ux_exp_object;size:80;not null" json:"object_id"`
	CreatedAt    time.Time `json:"created_at"`
}

func (ObjectLink) TableName() string { return "experiment_object_link" }

type GateInput struct {
	Stage       string  `json:"stage"`
	GateCode    string  `json:"gate_code"`
	Result      string  `json:"result"`
	Reason      string  `json:"reason"`
	EvidenceIDs []int64 `json:"evidence_ids"`
	DecidedBy   int64   `json:"decided_by"`
}
type Detail struct {
	Case        ExperimentCase   `json:"case"`
	Gates       []GateDecision   `json:"gates"`
	Evidence    []EvidenceRecord `json:"evidence"`
	ObjectLinks []ObjectLink     `json:"object_links"`
}
type OwnerSummary struct {
	ExperimentID        string     `json:"experiment_id"`
	Stage               string     `json:"stage"`
	PassedGates         int64      `json:"passed_gates"`
	Blockers            []string   `json:"blockers"`
	FinalProfitStatus   string     `json:"final_profit_status"`
	FinalRevenue        float64    `json:"final_revenue"`
	FinalTotalCost      float64    `json:"final_total_cost"`
	FinalProfitAmount   float64    `json:"final_profit_amount"`
	ProfitCurrency      string     `json:"profit_currency"`
	CashRecoveryStatus  string     `json:"cash_recovery_status"`
	CashRecoveredAmount float64    `json:"cash_recovered_amount"`
	CashCurrency        string     `json:"cash_currency"`
	CashRecoveredAt     *time.Time `json:"cash_recovered_at"`
	FinalDecision       string     `json:"final_decision"`
}
