package actionpolicy

import (
	"encoding/json"
	"fmt"
	"time"

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

// ──────────────────────────────────────────────────────────────
// PolicyRule CRUD (existing)
// ──────────────────────────────────────────────────────────────

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

func (s *Service) ToggleRule(id int64) error {
	return s.db.Model(&PolicyRule{}).Where("id = ?", id).
		Update("enabled", gorm.Expr("NOT enabled")).Error
}

func (s *Service) Evaluate(ctx *ActionContext) (*PolicyEvaluationResult, error) {
	rules, err := s.ListRules()
	if err != nil {
		return nil, err
	}

	result := &PolicyEvaluationResult{FinalOutcome: "escalate", Verdicts: make([]RuleVerdict, 0)}
	for _, rule := range rules {
		if !matches(ctx, rule) {
			continue
		}
		verdict := RuleVerdict{RuleID: rule.ID, RuleName: rule.Name, Outcome: rule.Outcome, Reason: buildReason(ctx, rule)}
		result.Verdicts = append(result.Verdicts, verdict)
		result.MatchedRules = append(result.MatchedRules, rule)
		if rule.Outcome == "block" {
			result.FinalOutcome = "block"
		} else if rule.Outcome == "escalate" && result.FinalOutcome != "block" {
			result.FinalOutcome = "escalate"
		} else if rule.Outcome == "auto_approve" && result.FinalOutcome != "block" {
			result.FinalOutcome = "auto_approve"
		}
	}
	return result, nil
}

func matches(ctx *ActionContext, rule PolicyRule) bool {
	if rule.RiskLevel != "" && rule.RiskLevel != "*" && rule.RiskLevel != ctx.RiskLevel {
		return false
	}
	if rule.ActionType != "" && rule.ActionType != "*" && rule.ActionType != ctx.ActionType {
		return false
	}
	if rule.SquadID != "" && rule.SquadID != "*" && rule.SquadID != ctx.SquadID {
		return false
	}
	if rule.AgentID != "" && rule.AgentID != "*" && rule.AgentID != ctx.AgentID {
		return false
	}
	if rule.BusinessObjectType != "" && rule.BusinessObjectType != "*" && rule.BusinessObjectType != ctx.BusinessObjectType {
		return false
	}
	if rule.MaxAmount != nil && ctx.Amount != nil && *ctx.Amount > *rule.MaxAmount {
		return false
	}
	if rule.MaxQuantity != nil && ctx.Quantity != nil && *ctx.Quantity > *rule.MaxQuantity {
		return false
	}
	if rule.MinConfidence != nil && ctx.Confidence != nil && *ctx.Confidence < *rule.MinConfidence {
		return false
	}
	return true
}

func buildReason(ctx *ActionContext, rule PolicyRule) string {
	parts := make([]string, 0, 4)
	if rule.MaxAmount != nil {
		parts = append(parts, fmt.Sprintf("max_amount=%.2f", *rule.MaxAmount))
	}
	if rule.MaxQuantity != nil {
		parts = append(parts, fmt.Sprintf("max_qty=%d", *rule.MaxQuantity))
	}
	if rule.MinConfidence != nil {
		parts = append(parts, fmt.Sprintf("min_conf=%.2f", *rule.MinConfidence))
	}
	return fmt.Sprintf("%s: %s [%s]", rule.Name, rule.Description, join(parts, ", "))
}

func join(parts []string, sep string) string {
	if len(parts) == 0 {
		return "no thresholds"
	}
	s := parts[0]
	for _, p := range parts[1:] {
		s += sep + p
	}
	return s
}

// ──────────────────────────────────────────────────────────────
// ApprovalPolicy CRUD (new)
// ──────────────────────────────────────────────────────────────

// ListPolicies returns all approval policies.
func (s *Service) ListPolicies() ([]ApprovalPolicy, error) {
	var policies []ApprovalPolicy
	if err := s.db.Order("id ASC").Find(&policies).Error; err != nil {
		return nil, err
	}
	return policies, nil
}

// GetPolicy returns a single approval policy by ID.
func (s *Service) GetPolicy(id int64) (*ApprovalPolicy, error) {
	var p ApprovalPolicy
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// CreatePolicy creates a new approval policy.
func (s *Service) CreatePolicy(p *ApprovalPolicy) error {
	return s.db.Create(p).Error
}

// UpdatePolicy updates an existing approval policy.
func (s *Service) UpdatePolicy(p *ApprovalPolicy) error {
	return s.db.Model(p).Updates(map[string]interface{}{
		"name":              p.Name,
		"agent_id":          p.AgentID,
		"decision_point":    p.DecisionPoint,
		"min_trust_score":   p.MinTrustScore,
		"requires_approval": p.RequiresApproval,
	}).Error
}

// DeletePolicy deletes an approval policy.
func (s *Service) DeletePolicy(id int64) error {
	return s.db.Delete(&ApprovalPolicy{}, id).Error
}

// GetMatchingPolicy finds the first ApprovalPolicy that matches the given
// agent+decision_point with RequiresApproval=true.
func (s *Service) GetMatchingPolicy(agentID, decisionPoint string) (*ApprovalPolicy, error) {
	var p ApprovalPolicy
	if err := s.db.Where("agent_id = ? AND decision_point = ? AND requires_approval = ?",
		agentID, decisionPoint, true).First(&p).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// ──────────────────────────────────────────────────────────────
// ApprovalRequest operations (new)
// ──────────────────────────────────────────────────────────────

// ListRequests returns approval requests, optionally filtered by status.
func (s *Service) ListRequests(status string) ([]ApprovalRequest, error) {
	q := s.db.Order("created_at DESC")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var reqs []ApprovalRequest
	if err := q.Find(&reqs).Error; err != nil {
		return nil, err
	}
	return reqs, nil
}

// SubmitApproval creates a new pending ApprovalRequest for the given agent+decision.
func (s *Service) SubmitApproval(policyID int64, agentID, decisionPoint string, payload json.RawMessage, requestedBy string) (*ApprovalRequest, error) {
	req := &ApprovalRequest{
		PolicyID:      policyID,
		AgentID:       agentID,
		DecisionPoint: decisionPoint,
		Payload:       payload,
		Status:        StatusPending,
		RequestedBy:   requestedBy,
	}
	if err := s.db.Create(req).Error; err != nil {
		return nil, err
	}
	return req, nil
}

// Review approves or rejects an approval request.
func (s *Service) Review(id int64, approve bool, reviewer string) (*ApprovalRequest, error) {
	var req ApprovalRequest
	if err := s.db.First(&req, id).Error; err != nil {
		return nil, err
	}
	if req.Status != StatusPending {
		return nil, fmt.Errorf("request %d is not pending (current: %s)", id, req.Status)
	}
	now := time.Now()
	status := StatusApproved
	if !approve {
		status = StatusRejected
	}
	updates := map[string]interface{}{
		"status":      status,
		"reviewed_by": reviewer,
		"reviewed_at": now,
	}
	if err := s.db.Model(&req).Updates(updates).Error; err != nil {
		return nil, err
	}
	req.Status = status
	req.ReviewedBy = &reviewer
	req.ReviewedAt = &now
	return &req, nil
}

// GetRequest returns a single approval request by ID.
func (s *Service) GetRequest(id int64) (*ApprovalRequest, error) {
	var req ApprovalRequest
	if err := s.db.First(&req, id).Error; err != nil {
		return nil, err
	}
	return &req, nil
}
