package xiaoq

import (
	"context"

	"github.com/lingmirror/backend-go/internal/domain/demandcase"
	"github.com/lingmirror/backend-go/internal/domain/experiment"
	"github.com/lingmirror/backend-go/internal/domain/sourcing1688"
)

const (
	CapabilityDemandCaseRead         = "demand_case.read"
	CapabilityDemandCaseDecisionRead = "demand_case.decision_card.read"
	CapabilityExperimentRead         = "experiment.read"
	CapabilityExperimentGateRead     = "experiment.gate_status.read"
	CapabilitySourcing1688Read       = "sourcing_1688.controlled_draft.read"
	CapabilityOrderFulfillmentRead   = "order.fulfillment.read"
	CapabilitySettlementRead         = "settlement.reconciliation.read"
	CapabilityProfitFinalRead        = "profit.final.read"
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
	{ID: CapabilityExperimentRead, Version: "v1", Domain: "experiment", Description: "读取当前 Owner 的经营实验案件、证据、闸门与对象关联", InputSchema: map[string]interface{}{"experiment_id": "non_empty_string"}, OutputSchema: map[string]interface{}{"type": "experiment_detail"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "preserve truth_status, source_uri, observed_at, verification and gate decision without upgrading facts", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.experiment.read", OwnerExplanation: "小Q只读取你的经营实验事实，不新增证据、不评估闸门、不改变实验状态", Status: "active"},
	{ID: CapabilityExperimentGateRead, Version: "v1", Domain: "experiment", Description: "读取当前 Owner 的经营实验闸门状态、阻断项与终局状态", InputSchema: map[string]interface{}{"experiment_id": "non_empty_string"}, OutputSchema: map[string]interface{}{"type": "experiment_owner_summary"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "experiment domain gates and owner summary are authoritative; AI cannot pass a gate", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.experiment.gate_status.read", OwnerExplanation: "小Q只解释已有闸门与缺口，不能替 Owner 核验证据或通过闸门", Status: "active"},
	{ID: CapabilitySourcing1688Read, Version: "v1", Domain: "sourcing1688", Description: "读取当前 Owner 的1688受控来源、不可变快照和内部草稿", InputSchema: map[string]interface{}{"source_id": "positive_int64"}, OutputSchema: map[string]interface{}{"type": "sourcing_1688_owner_view"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "preserve snapshot hash and cost truth_status; raw source and published payloads are forbidden", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.sourcing_1688.controlled_draft.read", OwnerExplanation: "小Q只读取受控货源与内部草稿，不发布、不采购、不批准，也不把估算或模拟信息升级为事实", Status: "active"},
	{ID: CapabilityOrderFulfillmentRead, Version: "v1", Domain: "experiment", Description: "从当前 Owner 的实验读取唯一关联订单的脱敏履约记录", InputSchema: map[string]interface{}{"experiment_id": "non_empty_string"}, OutputSchema: map[string]interface{}{"type": "owner_order_closure"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "order timestamps remain unknown/internal_record without external provenance; customer PII is forbidden", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.order.fulfillment.read", OwnerExplanation: "小Q只读取实验关联订单的脱敏内部记录，不把付款或签收记录升级为真实外部事实", Status: "active"},
	{ID: CapabilitySettlementRead, Version: "v1", Domain: "experiment", Description: "读取实验关联结算的来源可信度与逐项对账状态", InputSchema: map[string]interface{}{"experiment_id": "non_empty_string"}, OutputSchema: map[string]interface{}{"type": "owner_settlement_closure"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "only platform_import/api_sync with imported_at and fully matched items is trusted; raw payload is forbidden", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.settlement.reconciliation.read", OwnerExplanation: "小Q只解释结算来源和对账状态，不修改结算，也不读取原始敏感载荷", Status: "active"},
	{ID: CapabilityProfitFinalRead, Version: "v1", Domain: "experiment", Description: "读取同一订单的最终利润记录及阻断项", InputSchema: map[string]interface{}{"experiment_id": "non_empty_string"}, OutputSchema: map[string]interface{}{"type": "owner_profit_closure"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "only same-order final order_profit_record with no missing costs and one trusted settlement can be final", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.profit.final.read", OwnerExplanation: "小Q只读取最终利润记录；临时利润、缺成本或多结算混合会明确阻断", Status: "active"},
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

type ExperimentReader interface {
	GetDetail(ctx context.Context, id string, ownerID int64) (*experiment.Detail, error)
	OwnerSummary(ctx context.Context, id string, ownerID int64) (*experiment.OwnerSummary, error)
}

type Sourcing1688Reader interface {
	ReadOwnerView(ctx context.Context, sourceID, ownerID int64) (*sourcing1688.OwnerView, error)
}

type BusinessClosureReader interface {
	ReadOwnerBusinessClosure(ctx context.Context, ownerID int64, experimentID string) (*experiment.OwnerBusinessClosureView, error)
}
