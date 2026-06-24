package actionpolicy

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

func (s *Service) ListRules() ([]PolicyRule, error) {
	var rules []PolicyRule
	if err := s.db.Where("enabled = ?", true).Order("priority DESC, id ASC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

func (s *Service) CreateRule(r *PolicyRule) error {
	return s.db.Create(r).Error
}

func (s *Service) UpdateRule(r *PolicyRule) error {
	return s.db.Model(r).Updates(map[string]interface{}{
		"name": r.Name, "description": r.Description, "risk_level": r.RiskLevel,
		"action_type": r.ActionType, "squad_id": r.SquadID, "agent_id": r.AgentID,
		"business_object_type": r.BusinessObjectType, "max_amount": r.MaxAmount,
		"max_quantity": r.MaxQuantity, "min_confidence": r.MinConfidence,
		"auto_approve": r.AutoApprove, "outcome": r.Outcome, "priority": r.Priority,
		"enabled": r.Enabled,
	}).Error
}

func (s *Service) DeleteRule(id int64) error {
	return s.db.Model(&PolicyRule{}).Where("id = ?", id).Update("enabled", false).Error
}

func (s *Service) Evaluate(ctx *ActionContext) (*PolicyEvaluationResult, error) {
	rules, err := s.ListRules()
	if err != nil { return nil, err }

	result := &PolicyEvaluationResult{FinalOutcome: "escalate", Verdicts: make([]RuleVerdict, 0)}
	for _, rule := range rules {
		if !matches(ctx, rule) { continue }
		verdict := RuleVerdict{RuleID: rule.ID, RuleName: rule.Name, Outcome: rule.Outcome, Reason: buildReason(ctx, rule)}
		result.Verdicts = append(result.Verdicts, verdict)
		result.MatchedRules = append(result.MatchedRules, rule)
		if rule.Outcome == "block" { result.FinalOutcome = "block"
		} else if rule.Outcome == "escalate" && result.FinalOutcome != "block" { result.FinalOutcome = "escalate"
		} else if rule.Outcome == "auto_approve" && result.FinalOutcome != "block" { result.FinalOutcome = "auto_approve" }
	}
	return result, nil
}

func matches(ctx *ActionContext, rule PolicyRule) bool {
	if rule.RiskLevel != "" && rule.RiskLevel != "*" && rule.RiskLevel != ctx.RiskLevel { return false }
	if rule.ActionType != "" && rule.ActionType != "*" && rule.ActionType != ctx.ActionType { return false }
	if rule.SquadID != "" && rule.SquadID != "*" && rule.SquadID != ctx.SquadID { return false }
	if rule.AgentID != "" && rule.AgentID != "*" && rule.AgentID != ctx.AgentID { return false }
	if rule.BusinessObjectType != "" && rule.BusinessObjectType != "*" && rule.BusinessObjectType != ctx.BusinessObjectType { return false }
	if rule.MaxAmount != nil && ctx.Amount != nil && *ctx.Amount > *rule.MaxAmount { return false }
	if rule.MaxQuantity != nil && ctx.Quantity != nil && *ctx.Quantity > *rule.MaxQuantity { return false }
	if rule.MinConfidence != nil && ctx.Confidence != nil && *ctx.Confidence < *rule.MinConfidence { return false }
	return true
}

func buildReason(ctx *ActionContext, rule PolicyRule) string {
	parts := make([]string, 0, 4)
	if rule.MaxAmount != nil { parts = append(parts, fmt.Sprintf("max_amount=%.2f", *rule.MaxAmount)) }
	if rule.MaxQuantity != nil { parts = append(parts, fmt.Sprintf("max_qty=%d", *rule.MaxQuantity)) }
	if rule.MinConfidence != nil { parts = append(parts, fmt.Sprintf("min_conf=%.2f", *rule.MinConfidence)) }
	return fmt.Sprintf("%s: %s [%s]", rule.Name, rule.Description, join(parts, ", "))
}

func join(parts []string, sep string) string {
	if len(parts) == 0 { return "no thresholds" }
	s := parts[0]
	for _, p := range parts[1:] { s += sep + p }
	return s
}
