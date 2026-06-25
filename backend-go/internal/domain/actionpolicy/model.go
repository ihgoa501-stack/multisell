package actionpolicy

import (
	"encoding/json"
	"time"
)

type PolicyRule struct {
	ID          int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name        string `gorm:"column:name;not null" json:"name"`
	Description string `gorm:"column:description" json:"description"`
	RiskLevel         string `gorm:"column:risk_level" json:"risk_level,omitempty"`
	ActionType        string `gorm:"column:action_type" json:"action_type,omitempty"`
	SquadID           string `gorm:"column:squad_id" json:"squad_id,omitempty"`
	AgentID           string `gorm:"column:agent_id" json:"agent_id,omitempty"`
	BusinessObjectType string `gorm:"column:business_object_type" json:"business_object_type,omitempty"`
	MaxAmount     *float64 `gorm:"column:max_amount;type:numeric(12,2)" json:"max_amount,omitempty"`
	MaxQuantity   *int     `gorm:"column:max_quantity" json:"max_quantity,omitempty"`
	MinConfidence *float64 `gorm:"column:min_confidence;type:numeric(5,4)" json:"min_confidence,omitempty"`
	AutoApprove bool   `gorm:"column:auto_approve;not null;default:false" json:"auto_approve"`
	Outcome     string `gorm:"column:outcome;not null;default:escalate" json:"outcome"`
	Priority    int    `gorm:"column:priority;not null;default:0" json:"priority"`
	Enabled   bool      `gorm:"column:enabled;not null;default:true" json:"enabled"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (PolicyRule) TableName() string { return "approval_policy_rule" }

type PolicyEvaluationResult struct {
	MatchedRules []PolicyRule  `json:"matched_rules"`
	FinalOutcome string        `json:"final_outcome"`
	Verdicts     []RuleVerdict `json:"verdicts"`
}

type RuleVerdict struct {
	RuleID   int64  `json:"rule_id"`
	RuleName string `json:"rule_name"`
	Outcome  string `json:"outcome"`
	Reason   string `json:"reason"`
}

type ActionContext struct {
	AgentID            string
	SquadID            string
	ActionType         string
	RiskLevel          string
	BusinessObjectType string
	BusinessObjectID   string
	Amount             *float64
	Quantity           *int
	Confidence         *float64
}

func UnmarshalPayload(raw json.RawMessage) (amount *float64, quantity *int) {
	if len(raw) == 0 {
		return nil, nil
	}
	var p struct {
		Amount   *float64 `json:"amount"`
		Price    *float64 `json:"price"`
		Quantity *int     `json:"quantity"`
		Count    *int     `json:"count"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, nil
	}
	if p.Amount != nil { amount = p.Amount
	} else if p.Price != nil { amount = p.Price }
	if p.Quantity != nil { quantity = p.Quantity
	} else if p.Count != nil { quantity = p.Count }
	return
}
