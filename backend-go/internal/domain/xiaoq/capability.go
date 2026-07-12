package xiaoq

import (
	"context"

	"github.com/lingmirror/backend-go/internal/domain/demandcase"
)

const (
	CapabilityDemandCaseRead         = "demand_case.read"
	CapabilityDemandCaseDecisionRead = "demand_case.decision_card.read"
)

type Capability struct {
	ID                  string                 `json:"id"`
	Version             string                 `json:"version"`
	Domain              string                 `json:"domain"`
	Description         string                 `json:"description"`
	InputSchema         map[string]interface{} `json:"input_schema"`
	OutputSchema        map[string]interface{} `json:"output_schema"`
	Risk                string                 `json:"risk"`
	RequiredPermission  string                 `json:"required_permission"`
	ApprovalRequired    bool                   `json:"approval_required"`
	ExecutionModes      []string               `json:"execution_modes"`
	ExternalSideEffects bool                   `json:"external_side_effects"`
	IdempotencyRequired bool                   `json:"idempotency_required"`
	EvidencePolicy      string                 `json:"evidence_policy"`
	TimeoutSeconds      int                    `json:"timeout_seconds"`
	RetryLimit          int                    `json:"retry_limit"`
	AuditActionType     string                 `json:"audit_action_type"`
	OwnerExplanation    string                 `json:"owner_explanation"`
	Status              string                 `json:"status"`
}

var readCapabilities = []Capability{
	{ID: CapabilityDemandCaseRead, Version: "v1", Domain: "demandcase", Description: "读取当前 Owner 的候选市场案件、证据、裁决与快照", InputSchema: map[string]interface{}{"demand_case_id": "positive_int64"}, OutputSchema: map[string]interface{}{"type": "demand_case_detail"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "preserve truth_status, source, observed_at, run_id, snapshot_id and snapshot hash when available", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.demand_case.read", OwnerExplanation: "小Q只读取你的候选市场案件，不修改任何经营数据；模型费用预算仍未在此能力中确定", Status: "active"},
	{ID: CapabilityDemandCaseDecisionRead, Version: "v1", Domain: "demandcase", Description: "读取当前 Owner 的候选市场决策卡", InputSchema: map[string]interface{}{"demand_case_id": "positive_int64"}, OutputSchema: map[string]interface{}{"type": "owner_decision_card"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "domain verdict is authoritative; AI cannot upgrade it", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.demand_case.decision_card.read", OwnerExplanation: "小Q只解释系统已有裁决、反证和未知，不替你批准市场；模型费用预算仍未在此能力中确定", Status: "active"},
}

func Capabilities() []Capability {
	out := make([]Capability, 0, len(readCapabilities))
	for _, capability := range readCapabilities {
		if capability.Status == "active" {
			out = append(out, capability)
		}
	}
	return out
}

func activeCapability(id string) (Capability, bool) {
	for _, capability := range readCapabilities {
		if capability.ID == id && capability.Status == "active" {
			return capability, true
		}
	}
	return Capability{}, false
}

type DemandCaseReader interface {
	Get(ctx context.Context, id, ownerID int64) (*demandcase.Detail, error)
	DecisionCard(ctx context.Context, id, ownerID int64) (*demandcase.OwnerDecisionCard, error)
}
