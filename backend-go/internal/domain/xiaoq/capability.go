package xiaoq

import (
	"context"

	"github.com/lingmirror/backend-go/internal/domain/businessdecision"
	"github.com/lingmirror/backend-go/internal/domain/demandcase"
	"github.com/lingmirror/backend-go/internal/domain/experiment"
	"github.com/lingmirror/backend-go/internal/domain/integrations"
	"github.com/lingmirror/backend-go/internal/domain/sourcing1688"
)

const (
	CapabilityDemandCaseRead         = "demand_case.read"
	CapabilityDemandCaseDecisionRead = "demand_case.decision_card.read"
	CapabilityExperimentRead         = "experiment.read"
	CapabilityExperimentGateRead     = "experiment.gate_status.read"
	CapabilitySourcing1688Read       = "sourcing_1688.controlled_draft.read"
	CapabilityOrderFactRead          = "order.fact.read"
	CapabilityInventoryLedgerRead    = "inventory.ledger.read"
	CapabilityFulfillmentFactRead    = "fulfillment.fact.read"
	CapabilityAftersalesFactRead     = "aftersales.fact.read"
	CapabilitySettlementRead         = "settlement.reconciliation.read"
	CapabilityProfitFinalRead        = "profit.final.read"
	CapabilityCashReconciliationRead = "cash.reconciliation.read"
	CapabilityBusinessDecisionRead   = "business_decision.read"
	CapabilityBusinessRecommend      = "business_decision.recommendation.create"
	CapabilityOrderFulfillmentRead   = "order.fulfillment.read" // deprecated experiment-scoped v1; never active
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
	{ID: CapabilityDemandCaseRead, Version: "v1", Domain: "demandcase", Description: "读取当前 Owner 的候选市场案件、证据、裁决与快照", InputSchema: demandCaseIDJSONSchema(), OutputSchema: map[string]interface{}{"type": "demand_case_detail"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "preserve truth_status, source, observed_at, run_id, snapshot_id and snapshot hash when available", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.demand_case.read", OwnerExplanation: "小Q只读取你的候选市场案件，不修改任何经营数据；模型费用预算仍未在此能力中确定", Status: "active"},
	{ID: CapabilityDemandCaseDecisionRead, Version: "v1", Domain: "demandcase", Description: "读取当前 Owner 的候选市场决策卡", InputSchema: demandCaseIDJSONSchema(), OutputSchema: map[string]interface{}{"type": "owner_decision_card"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "domain verdict is authoritative; AI cannot upgrade it", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.demand_case.decision_card.read", OwnerExplanation: "小Q只解释系统已有裁决、反证和未知，不替你批准市场；模型费用预算仍未在此能力中确定", Status: "active"},
	{ID: CapabilityExperimentRead, Version: "v1", Domain: "experiment", Description: "读取当前 Owner 的经营事实核验案卷、证据、历史闸门与对象关联", InputSchema: map[string]interface{}{"experiment_id": "non_empty_string"}, OutputSchema: map[string]interface{}{"type": "experiment_detail"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "preserve truth_status, source_uri, observed_at and verification; links and gates are trace-only and cannot prove causality or feedback completion", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.experiment.read", OwnerExplanation: "小Q只读取你的经营事实核验案卷，不将关联、闸门或终态解释为因果或反馈闭环", Status: "active"},
	{ID: CapabilityExperimentGateRead, Version: "v1", Domain: "experiment", Description: "读取当前 Owner 的历史事实核验闸门与阻断项", InputSchema: map[string]interface{}{"experiment_id": "non_empty_string"}, OutputSchema: map[string]interface{}{"type": "experiment_owner_summary"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "experiment gates are trace records only; they cannot authorize an operating decision or prove causality or a feedback loop", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.experiment.gate_status.read", OwnerExplanation: "小Q只解释历史闸门与事实缺口，不得宣称经营决定或反馈闭环已完成", Status: "active"},
	{ID: CapabilitySourcing1688Read, Version: "v1", Domain: "sourcing1688", Description: "读取当前 Owner 的1688受控来源、不可变快照和内部草稿", InputSchema: map[string]interface{}{"source_id": "positive_int64"}, OutputSchema: map[string]interface{}{"type": "sourcing_1688_owner_view"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "preserve snapshot hash and cost truth_status; raw source and published payloads are forbidden", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.sourcing_1688.controlled_draft.read", OwnerExplanation: "小Q只读取受控货源与内部草稿，不发布、不采购、不批准，也不把估算或模拟信息升级为事实", Status: "active"},
	{ID: CapabilityOrderFactRead, Version: "v2", Domain: "integrations", Description: "按订单读取当前 Owner 的不可变平台订单事实", InputSchema: map[string]interface{}{"order_id": "positive_int64"}, OutputSchema: map[string]interface{}{"type": "owner_operating_view"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "exactly one applied external_observed Owner/account order ingest; raw payload and buyer PII forbidden", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.order.fact.read", OwnerExplanation: "小Q按你的订单读取不可变平台事实，不读取买家隐私或原始载荷", Status: "active"},
	{ID: CapabilityInventoryLedgerRead, Version: "v1", Domain: "integrations", Description: "读取订单对应的不可变库存动作账本", InputSchema: map[string]interface{}{"order_id": "positive_int64"}, OutputSchema: map[string]interface{}{"type": "owner_operating_view.inventory"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "only Owner-scoped order_inventory_ledger rows bound to the exact order", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.inventory.ledger.read", OwnerExplanation: "小Q只解释该订单已发生的库存预占、扣减或释放，不修改库存", Status: "active"},
	{ID: CapabilityFulfillmentFactRead, Version: "v1", Domain: "supplychain", Description: "读取订单的外部承运商事件", InputSchema: map[string]interface{}{"order_id": "positive_int64"}, OutputSchema: map[string]interface{}{"type": "owner_operating_view.fulfillment"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "only immutable external_observed carrier events; operator status is not evidence", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.fulfillment.fact.read", OwnerExplanation: "小Q只把不可变承运商事件作为履约事实；人工状态不会升级为签收事实", Status: "active"},
	{ID: CapabilityAftersalesFactRead, Version: "v1", Domain: "aftersales", Description: "读取售后请求、Owner决定状态及外部终局回执", InputSchema: map[string]interface{}{"order_id": "positive_int64"}, OutputSchema: map[string]interface{}{"type": "owner_operating_view.aftersales"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "request and terminal receipt remain separate; no receipt means no external terminal fact", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.aftersales.fact.read", OwnerExplanation: "小Q只读售后事实，不批准退款、不执行退款，也不把提交成功当成退款成功", Status: "active"},
	{ID: CapabilitySettlementRead, Version: "v2", Domain: "settlement", Description: "按订单读取不可变平台结算及精确金额行", InputSchema: map[string]interface{}{"order_id": "positive_int64"}, OutputSchema: map[string]interface{}{"type": "owner_operating_view.settlements"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "external_observed settlement facts bound to exact Owner/account/order/currency; raw payload forbidden", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.settlement.reconciliation.read", OwnerExplanation: "小Q只解释该订单的平台结算事实和费用行，不读取原始敏感载荷", Status: "active"},
	{ID: CapabilityProfitFinalRead, Version: "v2", Domain: "profit", Description: "按订单读取不可变最终利润版本", InputSchema: map[string]interface{}{"order_id": "positive_int64"}, OutputSchema: map[string]interface{}{"type": "owner_operating_view.profit"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "only immutable order_final_profit_version produced from exact actual costs, terminal refunds and external settlement facts", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.profit.final.read", OwnerExplanation: "小Q只读取权威最终利润版本；没有版本就明确保持未知", Status: "active"},
	{ID: CapabilityCashReconciliationRead, Version: "v1", Domain: "finance", Description: "按订单结算读取现金到账对账", InputSchema: map[string]interface{}{"order_id": "positive_int64"}, OutputSchema: map[string]interface{}{"type": "owner_operating_view.cash"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "only same-settlement cash reconciliation; bank raw payload and account details forbidden", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.cash.reconciliation.read", OwnerExplanation: "小Q只解释该订单结算对应的到账对账，不读取银行原始载荷或账户详情", Status: "active"},
	{ID: CapabilityBusinessDecisionRead, Version: "v1", Domain: "businessdecision", Description: "读取当前 Owner 的经营问题、冻结事实、AI建议和Owner决定", InputSchema: map[string]interface{}{"decision_case_id": "positive_int64"}, OutputSchema: map[string]interface{}{"type": "business_decision_detail"}, Risk: "read", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"read_only"}, ExternalSideEffects: false, IdempotencyRequired: false, EvidencePolicy: "frozen manifest is authoritative; AI recommendations stay inferred; Owner decisions are immutable", TimeoutSeconds: 5, RetryLimit: 0, AuditActionType: "xiao_q.capability.business_decision.read", OwnerExplanation: "小Q读取你的经营决定案卷，但不能替你选择、批准或执行", Status: "active"},
	{ID: CapabilityBusinessRecommend, Version: "v1", Domain: "businessdecision", Description: "基于冻结事实保存一条 inferred AI建议", InputSchema: map[string]interface{}{"decision_case_id": "positive_int64", "idempotency_key": "non_empty_string"}, OutputSchema: map[string]interface{}{"type": "ai_recommendation"}, Risk: "suggest", RequiredPermission: "agent.write", ApprovalRequired: false, ExecutionModes: []string{"suggest_only"}, ExternalSideEffects: false, IdempotencyRequired: true, EvidencePolicy: "recommendation binds the case manifest and must remain inferred; never creates OwnerDecision", TimeoutSeconds: 10, RetryLimit: 0, AuditActionType: "xiao_q.capability.business_decision.recommendation.create", OwnerExplanation: "小Q可保存一条推断建议供你审阅，但只有你在经营决定页面的明确操作才能形成Owner决定", Status: "active"},
}

func demandCaseIDJSONSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"demand_case_id": map[string]interface{}{"type": "integer", "minimum": 1},
		},
		"required":             []string{"demand_case_id"},
		"additionalProperties": false,
	}
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

type OwnerOperatingReader interface {
	ReadOwnerOperatingView(ctx context.Context, ownerID, orderID int64) (*integrations.OwnerOperatingView, error)
}
type BusinessDecisionReader interface {
	Get(ctx context.Context, ownerID, id int64) (*businessdecision.Detail, error)
	Recommend(ctx context.Context, ownerID, caseID int64, in businessdecision.RecommendInput) (*businessdecision.AIRecommendation, error)
}
