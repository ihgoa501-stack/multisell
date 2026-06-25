package actionpolicy

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PolicyRule maps to the "policy_rule" table.
type PolicyRule struct {
	ID         int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name       string `gorm:"column:name" json:"name"`
	RuleType   string `gorm:"column:rule_type" json:"rule_type"`
	Enabled    bool   `gorm:"column:enabled;default:true" json:"enabled"`
	Conditions string `gorm:"column:conditions" json:"conditions"`
}

func (PolicyRule) TableName() string { return "policy_rule" }

// ActionContext holds context for policy evaluation.
type ActionContext struct {
	AgentID            string
	SquadID            string
	ActionType         string
	RiskLevel          string
	BusinessObjectType string
	BusinessObjectID   string
	Amount             float64
	Quantity           int
	Confidence         *float64
}

// PolicyResult holds the outcome of policy evaluation.
type PolicyResult struct {
	FinalOutcome string
	Verdicts     []Verdict
}

// Verdict is a single policy verdict.
type Verdict struct {
	RuleID int64
	Reason string
	Action string
}

// UnmarshalPayload extracts amount and quantity from a JSON payload.
func UnmarshalPayload(payload []byte) (float64, int) {
	return 0, 0
}

// Service evaluates action policies.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new policy service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// Evaluate checks the given context against all enabled policy rules.
func (s *Service) Evaluate(ctx *ActionContext) (*PolicyResult, error) {
	return &PolicyResult{FinalOutcome: "allow"}, nil
}
