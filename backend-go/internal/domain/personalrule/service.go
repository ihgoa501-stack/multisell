package personalrule

import (
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides CRUD and ApplyRules operations for personal rules.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new personal rule service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// ListRules returns active rules for a user, optionally filtered by agent and
// decision point, sorted by priority (highest first).
func (s *Service) ListRules(userID int64, agentID, decisionPoint string) ([]PersonalRule, error) {
	q := s.db.Where("user_id = ? AND status = 'active'", userID)
	if agentID != "" {
		q = q.Where("agent_id = ?", agentID)
	}
	if decisionPoint != "" {
		q = q.Where("decision_point = ?", decisionPoint)
	}
	var rules []PersonalRule
	if err := q.Order("priority DESC, created_at DESC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// CreateRule creates a new personal rule.
func (s *Service) CreateRule(rule *PersonalRule) error {
	if rule.Source == "" {
		rule.Source = "manual"
	}
	if rule.Status == "" {
		rule.Status = "active"
	}
	return s.db.Create(rule).Error
}

// UpdateRule updates a personal rule. If the status changes, it writes a
// RuleMarkChange audit record.
func (s *Service) UpdateRule(rule *PersonalRule) error {
	var old PersonalRule
	if err := s.db.First(&old, rule.ID).Error; err != nil {
		return err
	}

	// Collect the updates.
	updates := map[string]interface{}{
		"rule_name":       rule.RuleName,
		"rule_type":       rule.RuleType,
		"rule_condition":  rule.RuleCondition,
		"rule_action":     rule.RuleAction,
		"priority":        rule.Priority,
		"source":          rule.Source,
		"source_decisions": rule.SourceDecisions,
		"status":          rule.Status,
		"confidence":      rule.Confidence,
		"times_applied":   rule.TimesApplied,
		"times_overridden": rule.TimesOverridden,
		"updated_at":      time.Now(),
	}

	if err := s.db.Model(&PersonalRule{}).Where("id = ?", rule.ID).Updates(updates).Error; err != nil {
		return err
	}

	// Record status change if it changed.
	if old.Status != rule.Status {
		if err := s.recordMarkChange(old, rule.Status); err != nil {
			s.logger.Warn("failed to record rule mark change", zap.Int64("rule_id", rule.ID), zap.Error(err))
		}
	}

	return nil
}

// DeleteRule deletes (hard-deletes) a personal rule.
func (s *Service) DeleteRule(id int64) error {
	return s.db.Delete(&PersonalRule{}, id).Error
}

// recordMarkChange writes a RuleMarkChange entry for a status transition.
func (s *Service) recordMarkChange(old PersonalRule, newStatus string) error {
	oldVal, _ := json.Marshal(old.Status)
	newVal, _ := json.Marshal(newStatus)

	ctx, _ := json.Marshal(map[string]interface{}{
		"rule_name": old.RuleName,
		"rule_type": old.RuleType,
	})

	change := &RuleMarkChange{
		TargetType:    "personal_rule",
		TargetID:      old.ID,
		FieldPath:     "$.status",
		OldValue:      oldVal,
		NewValue:      newVal,
		SourceType:    "manual",
		SourceID:      fmt.Sprintf("user_%d", old.UserID),
		ChangeSummary: fmt.Sprintf("Status changed: %s → %s", old.Status, newStatus),
		ContextJSON:   ctx,
	}

	return s.db.Create(change).Error
}

// ApplyRulesInput describes the context for rule application.
type ApplyRulesInput struct {
	UserID        int64
	AgentID       string
	DecisionPoint string
	Output        map[string]interface{}
}

// ApplyRulesOutput is the result of applying rules.
type ApplyRulesOutput struct {
	Output        map[string]interface{} `json:"output"`
	Results       []RuleResult           `json:"results"`
	Blocked       bool                   `json:"blocked"`
	AppliedRuleID int64                  `json:"applied_rule_id"`
}

// ApplyRules applies all matching personal rules to the given output map.
// Rules are evaluated in priority order. Block-type rules stop evaluation.
// Matching rules have their action applied to the output.
func (s *Service) ApplyRules(in *ApplyRulesInput) (*ApplyRulesOutput, error) {
	rules, err := s.ListRules(in.UserID, in.AgentID, in.DecisionPoint)
	if err != nil {
		return nil, err
	}

	out := &ApplyRulesOutput{
		Output:  in.Output,
		Results: make([]RuleResult, 0),
	}

	// Deep-copy the output so rules don't leak between evaluations.
	output := deepCopyMap(in.Output)

	for _, rule := range rules {
		// Parse condition.
		var cond Condition
		if len(rule.RuleCondition) > 0 {
			if err := json.Unmarshal(rule.RuleCondition, &cond); err != nil {
				s.logger.Warn("invalid rule_condition JSON, skipping rule",
					zap.Int64("rule_id", rule.ID), zap.Error(err))
				continue
			}
		}

		// Check condition match.
		matched := matchesCondition(output, &cond)
		if !matched {
			continue
		}

		// Parse action config.
		var actConfig map[string]interface{}
		if len(rule.RuleAction) > 0 {
			if err := json.Unmarshal(rule.RuleAction, &actConfig); err != nil {
				s.logger.Warn("invalid rule_action JSON, skipping rule",
					zap.Int64("rule_id", rule.ID), zap.Error(err))
				continue
			}
		}

		// Apply the action based on rule_type.
		result := applyAction(output, rule.RuleType, actConfig)
		result.RuleID = rule.ID
		result.RuleName = rule.RuleName
		result.RuleType = rule.RuleType

		out.Results = append(out.Results, *result)

		// Update rule statistics.
		s.db.Model(&PersonalRule{}).Where("id = ?", rule.ID).Updates(map[string]interface{}{
			"times_applied":  gorm.Expr("times_applied + 1"),
			"last_applied_at": time.Now(),
		})

		// Block action stops further rule evaluation.
		if result.Blocked {
			out.Blocked = true
			out.AppliedRuleID = rule.ID
			break
		}
	}

	out.Output = output
	return out, nil
}

// deepCopyMap creates a deep copy of a map[string]interface{}.
func deepCopyMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	data, err := json.Marshal(src)
	if err != nil {
		return src
	}
	var dst map[string]interface{}
	if err := json.Unmarshal(data, &dst); err != nil {
		return src
	}
	return dst
}
