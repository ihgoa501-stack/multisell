package xiaoq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/domain/businessdecision"
	"github.com/lingmirror/backend-go/internal/domain/demandcase"
	"github.com/lingmirror/backend-go/internal/domain/experiment"
	"github.com/lingmirror/backend-go/internal/domain/integrations"
	"github.com/lingmirror/backend-go/internal/domain/sourcing1688"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service struct {
	db         *gorm.DB
	logger     *zap.Logger
	demand     DemandCaseReader
	experiment ExperimentReader
	sourcing   Sourcing1688Reader
	closure    BusinessClosureReader
	operating  OwnerOperatingReader
	decision   BusinessDecisionReader
	provider   ai.LLMProvider
	traces     TraceRecorder
}

func (s *Service) WithOwnerOperatingReader(reader OwnerOperatingReader) *Service {
	s.operating = reader
	return s
}
func (s *Service) WithBusinessDecisionReader(reader BusinessDecisionReader) *Service {
	s.decision = reader
	return s
}

func (s *Service) WithSourcingReader(reader Sourcing1688Reader) *Service {
	s.sourcing = reader
	return s
}

func (s *Service) WithBusinessClosureReader(reader BusinessClosureReader) *Service {
	s.closure = reader
	return s
}

type TraceRecorder interface {
	Start(*ai.CreateTraceInput) (string, error)
	AppendEvent(string, *ai.AppendEventInput) (*ai.AITraceEvent, error)
	AddEvidence(string, *ai.AddEvidenceInput) (*ai.AIEvidenceRef, error)
	Complete(string, *ai.CompleteTraceInput) (*ai.AITrace, error)
	GetDetail(string) (*ai.TraceDetail, error)
}

func NewService(db *gorm.DB, logger *zap.Logger, demand DemandCaseReader, experiment ExperimentReader, provider ai.LLMProvider, traces TraceRecorder) *Service {
	return &Service{db: db, logger: logger, demand: demand, experiment: experiment, provider: provider, traces: traces}
}

func (s *Service) Identity() Identity {
	return Identity{AgentID: AgentID, Name: "小Q", Description: "凌镜 Owner 的受控经营 Agent", Mode: "read_only_v1"}
}

func (s *Service) Capabilities() []Capability { return Capabilities() }

func (s *Service) SendMessage(ctx context.Context, ownerID int64, in MessageInput) (*MessageResponse, error) {
	message := strings.TrimSpace(in.Message)
	if ownerID <= 0 || message == "" || len([]rune(message)) > MaxMessageRunes {
		return nil, ErrInvalidInput
	}
	target := strings.TrimSpace(in.TargetType)
	if target == "" && in.DemandCaseID > 0 {
		target = TargetDemandCase
	}
	if target == TargetExperiment {
		if strings.TrimSpace(in.ExperimentID) == "" || in.DemandCaseID != 0 || in.SourceID != 0 || in.OrderID != 0 || in.DecisionCaseID != 0 || s.experiment == nil {
			return nil, ErrInvalidInput
		}
		return s.sendExperimentMessage(ctx, ownerID, message, in.ExperimentID)
	}
	if target == TargetOperatingFacts {
		if in.OrderID <= 0 || in.DemandCaseID != 0 || in.SourceID != 0 || in.DecisionCaseID != 0 || strings.TrimSpace(in.ExperimentID) != "" || in.CreateRecommendation || strings.TrimSpace(in.IdempotencyKey) != "" || s.operating == nil {
			return nil, ErrInvalidInput
		}
		return s.sendOperatingFactsMessage(ctx, ownerID, message, in.OrderID)
	}
	if target == TargetBusinessDecision {
		if in.DecisionCaseID <= 0 || in.DemandCaseID != 0 || in.SourceID != 0 || in.OrderID != 0 || strings.TrimSpace(in.ExperimentID) != "" || s.decision == nil {
			return nil, ErrInvalidInput
		}
		if in.CreateRecommendation && strings.TrimSpace(in.IdempotencyKey) == "" {
			return nil, ErrInvalidInput
		}
		if !in.CreateRecommendation && strings.TrimSpace(in.IdempotencyKey) != "" {
			return nil, ErrInvalidInput
		}
		return s.sendBusinessDecisionMessage(ctx, ownerID, message, in.DecisionCaseID, in.CreateRecommendation, in.IdempotencyKey)
	}
	if target == TargetBusinessClosure {
		if strings.TrimSpace(in.ExperimentID) == "" || in.DemandCaseID != 0 || in.SourceID != 0 || s.closure == nil {
			return nil, ErrInvalidInput
		}
		return s.sendBusinessClosureMessage(ctx, ownerID, message, in.ExperimentID)
	}
	if target == TargetSourcing1688 {
		if in.SourceID <= 0 || in.DemandCaseID != 0 || strings.TrimSpace(in.ExperimentID) != "" || s.sourcing == nil {
			return nil, ErrInvalidInput
		}
		return s.sendSourcingMessage(ctx, ownerID, message, in.SourceID)
	}
	if target != TargetDemandCase || in.DemandCaseID <= 0 || in.SourceID != 0 || in.OrderID != 0 || in.DecisionCaseID != 0 || strings.TrimSpace(in.ExperimentID) != "" || in.CreateRecommendation || strings.TrimSpace(in.IdempotencyKey) != "" || s.demand == nil {
		return nil, ErrInvalidInput
	}
	if _, ok := activeCapability(CapabilityDemandCaseRead); !ok {
		return nil, ErrCapabilityUnavailable
	}
	if _, ok := activeCapability(CapabilityDemandCaseDecisionRead); !ok {
		return nil, ErrCapabilityUnavailable
	}
	if s.provider.Name() == "stub" {
		return s.sendDemandStubMessage(ctx, ownerID, message, in.DemandCaseID)
	}
	return s.sendDemandAgentMessage(ctx, ownerID, message, in.DemandCaseID)
}

func (s *Service) sendOperatingFactsMessage(ctx context.Context, ownerID int64, message string, orderID int64) (*MessageResponse, error) {
	capabilities := []string{CapabilityOrderFactRead, CapabilityInventoryLedgerRead, CapabilityFulfillmentFactRead, CapabilityAftersalesFactRead, CapabilitySettlementRead, CapabilityProfitFinalRead, CapabilityCashReconciliationRead}
	for _, id := range capabilities {
		if _, ok := activeCapability(id); !ok {
			return nil, ErrCapabilityUnavailable
		}
	}
	timedCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	traceID, err := s.startReadTrace(ownerID, "operating_facts_explain", "xiaoq-operating-facts-v2", map[string]interface{}{"target_type": TargetOperatingFacts, "question": message, "order_id": orderID})
	if err != nil {
		return nil, err
	}
	view, err := s.operating.ReadOwnerOperatingView(timedCtx, ownerID, orderID)
	if err != nil {
		return nil, s.capabilityFailure(traceID, CapabilityOrderFactRead, err, map[string]interface{}{"order_id": orderID})
	}
	for _, id := range capabilities {
		if _, err = s.traces.AppendEvent(traceID, &ai.AppendEventInput{EventType: "capability_call", Content: id, Payload: mustJSON(map[string]interface{}{"order_id": orderID, "risk": "read", "status": "succeeded"})}); err != nil {
			return nil, s.failTrace(traceID, err)
		}
	}
	for _, ref := range view.Evidence {
		if _, err = s.traces.AddEvidence(traceID, &ai.AddEvidenceInput{SourceType: ref.SourceType, SourceID: fmt.Sprint(ref.SourceID), Title: ref.SourceType, Summary: ref.TruthStatus, Payload: mustJSON(ref)}); err != nil {
			return nil, s.failTrace(traceID, err)
		}
	}
	payload, err := json.Marshal(struct {
		Question     string                           `json:"question"`
		View         *integrations.OwnerOperatingView `json:"view"`
		Capabilities []string                         `json:"capabilities"`
	}{message, view, capabilities})
	if err != nil {
		return nil, s.failTrace(traceID, err)
	}
	final := &MessageResponse{TraceID: traceID, AgentID: AgentID, Mode: "read_only_v2", TargetType: TargetOperatingFacts, OrderID: orderID, Trusted: false, Unknowns: append([]string(nil), view.Unknowns...), Blockers: append([]string(nil), view.Blockers...)}
	if s.provider.Name() == "stub" {
		final.Answer, final.TruthStatus, final.Provider, final.Model = "这是模拟回答，不能作为可信经营结论。请直接核对订单事实、库存账本、履约、售后、结算、利润和现金证据。", TruthMock, "stub", "stub-v1"
	} else {
		resp, e := s.provider.Chat(timedCtx, &ai.LLMRequest{System: "你是凌镜的小Q。只能根据同一Owner、同一order_id的权威经营事实视图回答。严格保留每条truth_status；不得读取或推测买家PII、平台/承运商/银行原始载荷；不得把缺失事实、人工状态、已提交请求或批次级现金升级为订单终局；不得作Owner决定或执行任何动作。用简明中文分事实、阻断、未知和下一步。", Messages: []ai.LLMMessage{{Role: "user", Content: string(payload)}}, MaxTokens: 900, Metadata: map[string]interface{}{"agent_id": AgentID, "order_id": orderID}})
		if e != nil {
			return nil, s.capabilityFailure(traceID, CapabilityOrderFactRead, e, map[string]interface{}{"order_id": orderID})
		}
		final.Answer, final.TruthStatus, final.Provider, final.Model, final.TokensIn, final.TokensOut, final.LatencyMs = resp.Answer, TruthInferred, s.provider.Name(), resp.Model, resp.TokensIn, resp.TokensOut, resp.LatencyMs
	}
	for _, ref := range view.Evidence {
		final.Evidence = append(final.Evidence, EvidenceItem{ID: ref.SourceID, Title: ref.SourceType, TruthStatus: ref.TruthStatus, ObservedAt: ref.ObservedAt.UTC().Format(time.RFC3339), Summary: ref.Summary, SnapshotSHA256: ref.SHA256})
	}
	final.Links = []ResponseLink{{Label: "订单事实", Href: "/orders/" + strconv.FormatInt(orderID, 10)}, {Label: "小Q 执行记录", Href: "/xiaoq/traces/" + traceID}}
	final.Provenance = Provenance{Provider: final.Provider, Model: final.Model, TokensIn: final.TokensIn, TokensOut: final.TokensOut, LatencyMs: final.LatencyMs}
	if err = s.complete(traceID, final, "completed"); err != nil {
		return nil, &RunError{TraceID: traceID, Err: err}
	}
	return final, nil
}

func (s *Service) sendBusinessDecisionMessage(ctx context.Context, ownerID int64, message string, caseID int64, create bool, key string) (*MessageResponse, error) {
	if _, ok := activeCapability(CapabilityBusinessDecisionRead); !ok {
		return nil, ErrCapabilityUnavailable
	}
	if create {
		if _, ok := activeCapability(CapabilityBusinessRecommend); !ok {
			return nil, ErrCapabilityUnavailable
		}
	}
	timedCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	traceID, err := s.startReadTrace(ownerID, "business_decision_explain", "xiaoq-business-decision-v1", map[string]interface{}{"target_type": TargetBusinessDecision, "question": message, "decision_case_id": caseID, "create_recommendation": create})
	if err != nil {
		return nil, err
	}
	detail, err := s.decision.Get(timedCtx, ownerID, caseID)
	if err != nil {
		return nil, s.capabilityFailure(traceID, CapabilityBusinessDecisionRead, err, map[string]interface{}{"decision_case_id": caseID})
	}
	if _, err = s.traces.AppendEvent(traceID, &ai.AppendEventInput{EventType: "capability_call", Content: CapabilityBusinessDecisionRead, Payload: mustJSON(map[string]interface{}{"decision_case_id": caseID, "risk": "read", "status": "succeeded", "manifest_sha256": detail.Case.ManifestSHA256})}); err != nil {
		return nil, s.failTrace(traceID, err)
	}
	if _, err = s.traces.AddEvidence(traceID, &ai.AddEvidenceInput{SourceType: "business_decision_fact_snapshot", SourceID: fmt.Sprint(detail.Snapshot.ID), Title: "冻结经营事实", Summary: detail.Snapshot.TruthStatus, Payload: mustJSON(map[string]interface{}{"decision_case_id": caseID, "object_type": detail.Snapshot.ObjectType, "object_id": detail.Snapshot.ObjectID, "truth_status": detail.Snapshot.TruthStatus, "payload_sha256": detail.Snapshot.PayloadSHA256, "source_observed_at": detail.Snapshot.SourceObservedAt})}); err != nil {
		return nil, s.failTrace(traceID, err)
	}
	payload, err := json.Marshal(struct {
		Question string                   `json:"question"`
		Detail   *businessdecision.Detail `json:"detail"`
	}{message, detail})
	if err != nil {
		return nil, s.failTrace(traceID, err)
	}
	final := &MessageResponse{TraceID: traceID, AgentID: AgentID, Mode: "decision_support_v1", TargetType: TargetBusinessDecision, DecisionCaseID: caseID, Trusted: false}
	if s.provider.Name() == "stub" {
		final.Answer, final.TruthStatus, final.Provider, final.Model = "这是模拟回答，不能保存为可信经营建议，也不能替Owner形成决定。", TruthMock, "stub", "stub-v1"
		create = false
	} else {
		resp, e := s.provider.Chat(timedCtx, &ai.LLMRequest{System: "你是凌镜的小Q。只能根据冻结经营事实和现有案卷提出建议。回答必须保持inferred，说明依据和未知；不得替Owner选择、批准或执行，不得声称因果成立。", Messages: []ai.LLMMessage{{Role: "user", Content: string(payload)}}, MaxTokens: 800, Metadata: map[string]interface{}{"agent_id": AgentID, "decision_case_id": caseID}})
		if e != nil {
			return nil, s.capabilityFailure(traceID, CapabilityBusinessDecisionRead, e, map[string]interface{}{"decision_case_id": caseID})
		}
		final.Answer, final.TruthStatus, final.Provider, final.Model, final.TokensIn, final.TokensOut, final.LatencyMs = resp.Answer, TruthInferred, s.provider.Name(), resp.Model, resp.TokensIn, resp.TokensOut, resp.LatencyMs
	}
	if create {
		rec, e := s.decision.Recommend(timedCtx, ownerID, caseID, businessdecision.RecommendInput{Recommendation: final.Answer, Rationale: "小Q基于冻结事实清单生成；请Owner核对证据与未知后决定", TruthStatus: "inferred", Unknowns: detail.Case.Unknowns, IdempotencyKey: strings.TrimSpace(key)})
		if e != nil {
			return nil, s.capabilityFailure(traceID, CapabilityBusinessRecommend, e, map[string]interface{}{"decision_case_id": caseID})
		}
		final.RecommendationID = rec.ID
		if _, e = s.traces.AppendEvent(traceID, &ai.AppendEventInput{EventType: "capability_call", Content: CapabilityBusinessRecommend, Payload: mustJSON(map[string]interface{}{"decision_case_id": caseID, "recommendation_id": rec.ID, "risk": "suggest", "status": "succeeded", "truth_status": "inferred", "manifest_sha256": rec.ManifestSHA256})}); e != nil {
			return nil, s.failTrace(traceID, e)
		}
	}
	final.Evidence = []EvidenceItem{{ID: detail.Snapshot.ID, Title: "冻结经营事实", TruthStatus: detail.Snapshot.TruthStatus, ObservedAt: detail.Snapshot.SourceObservedAt.UTC().Format(time.RFC3339), Summary: detail.Snapshot.ObjectType, SnapshotSHA256: detail.Snapshot.PayloadSHA256}}
	final.Unknowns = append([]string(nil), detail.Case.Unknowns...)
	final.Links = []ResponseLink{{Label: "经营决定案卷", Href: "/business-decisions/" + strconv.FormatInt(caseID, 10)}, {Label: "小Q 执行记录", Href: "/xiaoq/traces/" + traceID}}
	final.Provenance = Provenance{Provider: final.Provider, Model: final.Model, TokensIn: final.TokensIn, TokensOut: final.TokensOut, LatencyMs: final.LatencyMs}
	if err = s.complete(traceID, final, "completed"); err != nil {
		return nil, &RunError{TraceID: traceID, Err: err}
	}
	return final, nil
}

func (s *Service) sendBusinessClosureMessage(ctx context.Context, ownerID int64, message, experimentID string) (*MessageResponse, error) {
	capabilities := []string{CapabilityOrderFulfillmentRead, CapabilitySettlementRead, CapabilityProfitFinalRead}
	timeoutSeconds := 5
	for _, id := range capabilities {
		capability, ok := activeCapability(id)
		if !ok {
			return nil, ErrCapabilityUnavailable
		}
		if capability.TimeoutSeconds > 0 && capability.TimeoutSeconds < timeoutSeconds {
			timeoutSeconds = capability.TimeoutSeconds
		}
	}
	timedCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	traceID, err := s.startReadTrace(ownerID, "business_closure_explain", "xiaoq-business-closure-v1", map[string]interface{}{"target_type": TargetBusinessClosure, "question": message, "experiment_id": experimentID})
	if err != nil {
		return nil, err
	}
	view, err := s.closure.ReadOwnerBusinessClosure(timedCtx, ownerID, experimentID)
	if err != nil {
		return nil, s.capabilityFailure(traceID, CapabilityOrderFulfillmentRead, err, map[string]interface{}{"experiment_id": experimentID})
	}
	for _, id := range capabilities {
		if _, err := s.traces.AppendEvent(traceID, &ai.AppendEventInput{EventType: "capability_call", Content: id, Payload: mustJSON(map[string]interface{}{"experiment_id": experimentID, "risk": "read", "status": "succeeded"})}); err != nil {
			return nil, s.failTrace(traceID, err)
		}
	}
	for _, ref := range view.EvidenceRefs {
		if _, err := s.traces.AddEvidence(traceID, &ai.AddEvidenceInput{SourceType: ref.SourceType, SourceID: ref.SourceID, Title: ref.SourceType, Summary: ref.TruthStatus, Payload: mustJSON(ref)}); err != nil {
			return nil, s.failTrace(traceID, err)
		}
	}
	inputJSON, err := json.Marshal(struct {
		Question     string                               `json:"question"`
		View         *experiment.OwnerBusinessClosureView `json:"view"`
		Capabilities []string                             `json:"capabilities"`
	}{message, view, capabilities})
	if err != nil {
		return nil, s.failTrace(traceID, err)
	}
	final := &MessageResponse{TraceID: traceID, AgentID: AgentID, Mode: "read_only_v1", TargetType: TargetBusinessClosure, ExperimentID: experimentID, Trusted: false}
	if s.provider.Name() == "stub" {
		final.Answer, final.TruthStatus, final.Provider, final.Model = "这是模拟回答，不能作为可信经营结论。订单记录仍需外部核验；请检查结算对账、最终利润和未知项。", TruthMock, "stub", "stub-v1"
	} else {
		resp, providerErr := s.provider.Chat(timedCtx, &ai.LLMRequest{System: "你是凌镜的小Q。只能根据Owner经营闭环只读视图回答。订单付款/签收的内部记录仍是unknown，不得称为actual；只在settlement trusted且fully_reconciled、profit final且无缺失成本且没有多结算混合时说明系统记录为最终利润。售后观察期和现金金额/币种一致性当前保持unknown，不得声称售后已关闭或现金已完全回收。不得输出或推测客户个人信息。", Messages: []ai.LLMMessage{{Role: "user", Content: string(inputJSON)}}, MaxTokens: 800, Metadata: map[string]interface{}{"agent_id": AgentID, "experiment_id": experimentID}})
		if providerErr != nil {
			return nil, s.capabilityFailure(traceID, CapabilityProfitFinalRead, providerErr, map[string]interface{}{"experiment_id": experimentID})
		}
		final.Answer, final.TruthStatus, final.Provider, final.Model, final.TokensIn, final.TokensOut, final.LatencyMs = resp.Answer, TruthInferred, s.provider.Name(), resp.Model, resp.TokensIn, resp.TokensOut, resp.LatencyMs
	}
	for _, ref := range view.EvidenceRefs {
		id, _ := strconv.ParseInt(ref.SourceID, 10, 64)
		final.Evidence = append(final.Evidence, EvidenceItem{ID: id, Title: ref.SourceType, TruthStatus: ref.TruthStatus, Summary: ref.Summary})
	}
	final.Unknowns = append(append([]string{}, view.Unknowns...), view.Blockers...)
	final.Links = []ResponseLink{{Label: "经营实验", Href: "/experiments/" + experimentID}, {Label: "小Q 执行记录", Href: "/xiaoq/traces/" + traceID}}
	final.Provenance = Provenance{Provider: final.Provider, Model: final.Model, TokensIn: final.TokensIn, TokensOut: final.TokensOut, LatencyMs: final.LatencyMs}
	if err := s.complete(traceID, final, "completed"); err != nil {
		return nil, &RunError{TraceID: traceID, Err: err}
	}
	return final, nil
}

func (s *Service) sendSourcingMessage(ctx context.Context, ownerID int64, message string, sourceID int64) (*MessageResponse, error) {
	capability, ok := activeCapability(CapabilitySourcing1688Read)
	if !ok {
		return nil, ErrCapabilityUnavailable
	}
	timedCtx, cancel := context.WithTimeout(ctx, time.Duration(capability.TimeoutSeconds)*time.Second)
	defer cancel()
	traceID, err := s.startReadTrace(ownerID, "sourcing_1688_explain", "xiaoq-sourcing-1688-v1", map[string]interface{}{"target_type": TargetSourcing1688, "question": message, "source_id": sourceID})
	if err != nil {
		return nil, err
	}
	view, err := s.sourcing.ReadOwnerView(timedCtx, sourceID, ownerID)
	if err != nil {
		return nil, s.capabilityFailure(traceID, CapabilitySourcing1688Read, err, map[string]interface{}{"source_id": sourceID})
	}
	if _, err := s.traces.AppendEvent(traceID, &ai.AppendEventInput{EventType: "capability_call", Content: CapabilitySourcing1688Read, Payload: mustJSON(map[string]interface{}{"source_id": sourceID, "risk": "read", "status": "succeeded"})}); err != nil {
		return nil, s.failTrace(traceID, err)
	}
	if _, err := s.traces.AddEvidence(traceID, &ai.AddEvidenceInput{SourceType: "sourcing_1688_snapshot", SourceID: fmt.Sprint(view.Snapshot.ID), Title: "1688不可变来源快照", Summary: "quoted", Payload: mustJSON(map[string]interface{}{"source_id": sourceID, "snapshot_id": view.Snapshot.ID, "snapshot_sha256": view.Snapshot.RawSHA256, "source_reference": view.Snapshot.SourceReference, "collected_at": view.Snapshot.CollectedAt})}); err != nil {
		return nil, s.failTrace(traceID, err)
	}
	for _, cost := range view.Costs {
		if _, err := s.traces.AddEvidence(traceID, &ai.AddEvidenceInput{SourceType: "sourcing_1688_cost", SourceID: fmt.Sprint(cost.ID), Title: cost.CostType, Summary: cost.TruthStatus, Payload: mustJSON(cost)}); err != nil {
			return nil, s.failTrace(traceID, err)
		}
	}
	for _, media := range view.Media {
		if _, err := s.traces.AddEvidence(traceID, &ai.AddEvidenceInput{SourceType: "sourcing_1688_media", SourceID: fmt.Sprint(media.ID), Title: media.MediaRole, Summary: media.RightsStatus, Payload: mustJSON(media)}); err != nil {
			return nil, s.failTrace(traceID, err)
		}
	}
	inputJSON, err := json.Marshal(struct {
		Question   string                  `json:"question"`
		SourceID   int64                   `json:"source_id"`
		View       *sourcing1688.OwnerView `json:"view"`
		Capability string                  `json:"capability"`
	}{message, sourceID, view, CapabilitySourcing1688Read})
	if err != nil {
		return nil, s.failTrace(traceID, err)
	}
	final := &MessageResponse{TraceID: traceID, AgentID: AgentID, Mode: "read_only_v1", TargetType: TargetSourcing1688, SourceID: sourceID, Trusted: false}
	if s.provider.Name() == "stub" {
		final.Answer, final.TruthStatus, final.Provider, final.Model = "这是模拟回答，不能作为可信经营建议。请核对来源快照、草稿绑定、成本真实性和限制项。", TruthMock, "stub", "stub-v1"
	} else {
		resp, providerErr := s.provider.Chat(timedCtx, &ai.LLMRequest{System: "你是凌镜的小Q。只能根据提供的1688受控来源与内部草稿只读视图回答。严格保留成本 truth_status；图片权利没有独立真实性字段时必须保持 unknown；不得声称来源、图片权利、费用、渠道契约已经外部核验；不得发布、采购、批准或改变状态。用简明中文说明当前草稿、证据、限制和下一步。", Messages: []ai.LLMMessage{{Role: "user", Content: string(inputJSON)}}, MaxTokens: 800, Metadata: map[string]interface{}{"agent_id": AgentID, "source_id": sourceID}})
		if providerErr != nil {
			failure := mustJSON(map[string]interface{}{"error": providerErr.Error()})
			if _, appendErr := s.traces.AppendEvent(traceID, &ai.AppendEventInput{EventType: "provider_error", Content: "LLM provider call failed", Payload: failure}); appendErr != nil {
				return nil, s.failTrace(traceID, errors.Join(providerErr, appendErr))
			}
			_, _ = s.traces.Complete(traceID, &ai.CompleteTraceInput{FinalOutput: failure, RiskLevel: "low", Status: "failed"})
			return nil, &RunError{TraceID: traceID, Err: providerErr}
		}
		final.Answer, final.TruthStatus, final.Provider, final.Model, final.TokensIn, final.TokensOut, final.LatencyMs = resp.Answer, TruthInferred, s.provider.Name(), resp.Model, resp.TokensIn, resp.TokensOut, resp.LatencyMs
		if _, err := s.traces.AppendEvent(traceID, &ai.AppendEventInput{EventType: "model_response", Content: "provider response received", Payload: mustJSON(map[string]interface{}{"provider": final.Provider, "model": final.Model, "truth_status": final.TruthStatus})}); err != nil {
			return nil, s.failTrace(traceID, err)
		}
	}
	s.addSourcingGrounding(final, view)
	if err := s.complete(traceID, final, "completed"); err != nil {
		return nil, &RunError{TraceID: traceID, Err: err}
	}
	return final, nil
}

func (s *Service) addSourcingGrounding(response *MessageResponse, view *sourcing1688.OwnerView) {
	response.Evidence = []EvidenceItem{{ID: view.Snapshot.ID, Title: "1688不可变来源快照", TruthStatus: "quoted", SourceURL: view.Snapshot.SourceReference, ObservedAt: view.Snapshot.CollectedAt.UTC().Format(time.RFC3339), Summary: "snapshot", SnapshotID: view.Snapshot.ID, SnapshotSHA256: view.Snapshot.RawSHA256}}
	response.Unknowns = append([]string(nil), view.Limitations...)
	for _, cost := range view.Costs {
		response.Evidence = append(response.Evidence, EvidenceItem{ID: cost.ID, Title: cost.CostType, TruthStatus: cost.TruthStatus, SourceURL: cost.SourceReference, ObservedAt: cost.ObservedAt.UTC().Format(time.RFC3339), Summary: fmt.Sprintf("%.2f %s", cost.Amount, cost.Currency), SnapshotID: view.Snapshot.ID, SnapshotSHA256: view.Snapshot.RawSHA256})
		if cost.TruthStatus == "unknown" || cost.TruthStatus == "mock" || cost.TruthStatus == "inferred" || cost.TruthStatus == "estimated" {
			response.Unknowns = append(response.Unknowns, cost.CostType+"（"+cost.TruthStatus+"）")
		}
	}
	for _, media := range view.Media {
		observedAt := ""
		if media.ObservedAt != nil {
			observedAt = media.ObservedAt.UTC().Format(time.RFC3339)
		}
		response.Evidence = append(response.Evidence, EvidenceItem{ID: media.ID, Title: media.MediaRole, TruthStatus: media.TruthStatus, SourceURL: media.SourceReference, ObservedAt: observedAt, Summary: "rights_status=" + media.RightsStatus, SnapshotID: view.Snapshot.ID, SnapshotSHA256: view.Snapshot.RawSHA256})
		response.Unknowns = append(response.Unknowns, media.MediaRole+"图片权利真实性未知（status="+media.RightsStatus+"）")
	}
	response.Links = []ResponseLink{
		{Label: "1688受控货源", Href: "/sourcing1688?record_id=" + fmt.Sprint(response.SourceID)},
		{Label: "关联候选市场", Href: "/demand-cases/" + fmt.Sprint(view.Source.DemandCaseID)},
		{Label: "关联经营实验", Href: "/experiments/" + view.Source.ExperimentID},
		{Label: "小Q 执行记录", Href: "/xiaoq/traces/" + response.TraceID},
	}
	response.Provenance = Provenance{Provider: response.Provider, Model: response.Model, TokensIn: response.TokensIn, TokensOut: response.TokensOut, LatencyMs: response.LatencyMs}
}

func (s *Service) sendExperimentMessage(ctx context.Context, ownerID int64, message, experimentID string) (*MessageResponse, error) {
	if _, ok := activeCapability(CapabilityExperimentRead); !ok {
		return nil, ErrCapabilityUnavailable
	}
	if _, ok := activeCapability(CapabilityExperimentGateRead); !ok {
		return nil, ErrCapabilityUnavailable
	}
	traceID, err := s.startReadTrace(ownerID, "experiment_explain", "xiaoq-experiment-v1", map[string]interface{}{"target_type": TargetExperiment, "question": message, "experiment_id": experimentID})
	if err != nil {
		return nil, err
	}
	detail, err := s.experiment.GetDetail(ctx, experimentID, ownerID)
	if err != nil {
		return nil, s.capabilityFailure(traceID, CapabilityExperimentRead, err, map[string]interface{}{"experiment_id": experimentID})
	}
	if _, err := s.traces.AppendEvent(traceID, &ai.AppendEventInput{EventType: "capability_call", Content: CapabilityExperimentRead, Payload: mustJSON(map[string]interface{}{"experiment_id": experimentID, "risk": "read", "status": "succeeded"})}); err != nil {
		return nil, s.failTrace(traceID, err)
	}
	summary, err := s.experiment.OwnerSummary(ctx, experimentID, ownerID)
	if err != nil {
		return nil, s.capabilityFailure(traceID, CapabilityExperimentGateRead, err, map[string]interface{}{"experiment_id": experimentID})
	}
	if _, err := s.traces.AppendEvent(traceID, &ai.AppendEventInput{EventType: "capability_call", Content: CapabilityExperimentGateRead, Payload: mustJSON(map[string]interface{}{"experiment_id": experimentID, "risk": "read", "status": "succeeded"})}); err != nil {
		return nil, s.failTrace(traceID, err)
	}
	contextPayload := struct {
		Question     string                   `json:"question"`
		ExperimentID string                   `json:"experiment_id"`
		Detail       *experiment.Detail       `json:"detail"`
		GateStatus   *experiment.OwnerSummary `json:"gate_status"`
		Capabilities []string                 `json:"capabilities"`
	}{message, experimentID, detail, summary, []string{CapabilityExperimentRead, CapabilityExperimentGateRead}}
	inputJSON, err := json.Marshal(contextPayload)
	if err != nil {
		return nil, err
	}
	for _, ev := range detail.Evidence {
		observedAt, verifiedAt := "", ""
		if ev.ObservedAt != nil {
			observedAt = ev.ObservedAt.UTC().Format(time.RFC3339)
		}
		if ev.VerifiedAt != nil {
			verifiedAt = ev.VerifiedAt.UTC().Format(time.RFC3339)
		}
		payload := map[string]interface{}{"experiment_id": experimentID, "stage": ev.Stage, "evidence_kind": ev.EvidenceKind, "truth_status": ev.TruthStatus, "source_uri": ev.SourceURI, "observed_at": observedAt, "verified_by": ev.VerifiedBy, "verified_at": verifiedAt}
		if _, err := s.traces.AddEvidence(traceID, &ai.AddEvidenceInput{SourceType: "experiment_evidence", SourceID: fmt.Sprint(ev.ID), Title: ev.Title, Summary: ev.TruthStatus, Payload: mustJSON(payload)}); err != nil {
			return nil, s.failTrace(traceID, err)
		}
	}

	if s.provider.Name() == "stub" {
		answer := "这是模拟回答，不能作为可信经营建议。实验当前阶段为「" + detail.Case.Stage + "」；请核对证据、闸门阻断项和未知事实。"
		final := &MessageResponse{TraceID: traceID, AgentID: AgentID, Mode: "read_only_v1", TargetType: TargetExperiment, ExperimentID: experimentID, Answer: answer, TruthStatus: TruthMock, Trusted: false, Provider: "stub", Model: "stub-v1"}
		s.addExperimentGrounding(final, detail, summary)
		if err := s.complete(traceID, final, "completed"); err != nil {
			return nil, &RunError{TraceID: traceID, Err: err}
		}
		return final, nil
	}
	request := &ai.LLMRequest{
		System:   "你是凌镜的小Q。只能根据提供的历史经营事实核验案卷与闸门记录回答。严格保留 actual/quoted/estimated/unknown/mock/inferred；不得新增或核验证据、通过闸门或改变状态。对象关联、闸门通过、交易终态、利润或现金到账都不证明因果关系或反馈闭环，不得宣称最终经营决定已被该案卷授权。用简明中文说明当前阶段、已有证据、阻断项和未知。",
		Messages: []ai.LLMMessage{{Role: "user", Content: string(inputJSON)}}, MaxTokens: 800,
		Metadata: map[string]interface{}{"agent_id": AgentID, "experiment_id": experimentID},
	}
	resp, err := s.provider.Chat(ctx, request)
	if err != nil {
		failure := mustJSON(map[string]interface{}{"error": err.Error()})
		if _, appendErr := s.traces.AppendEvent(traceID, &ai.AppendEventInput{EventType: "provider_error", Content: "LLM provider call failed", Payload: failure}); appendErr != nil {
			return nil, s.failTrace(traceID, errors.Join(err, appendErr))
		}
		_, _ = s.traces.Complete(traceID, &ai.CompleteTraceInput{FinalOutput: failure, RiskLevel: "low", Status: "failed"})
		return nil, &RunError{TraceID: traceID, Err: err}
	}
	final := &MessageResponse{TraceID: traceID, AgentID: AgentID, Mode: "read_only_v1", TargetType: TargetExperiment, ExperimentID: experimentID, Answer: resp.Answer, TruthStatus: TruthInferred, Trusted: false, Provider: s.provider.Name(), Model: resp.Model, TokensIn: resp.TokensIn, TokensOut: resp.TokensOut, LatencyMs: resp.LatencyMs}
	s.addExperimentGrounding(final, detail, summary)
	if _, err := s.traces.AppendEvent(traceID, &ai.AppendEventInput{EventType: "model_response", Content: "provider response received", Payload: mustJSON(map[string]interface{}{"provider": final.Provider, "model": final.Model, "tokens_in": final.TokensIn, "tokens_out": final.TokensOut, "truth_status": final.TruthStatus})}); err != nil {
		return nil, s.failTrace(traceID, err)
	}
	if err := s.complete(traceID, final, "completed"); err != nil {
		return nil, &RunError{TraceID: traceID, Err: err}
	}
	return final, nil
}

func (s *Service) addExperimentGrounding(response *MessageResponse, detail *experiment.Detail, summary *experiment.OwnerSummary) {
	response.Evidence = make([]EvidenceItem, 0, len(detail.Evidence))
	response.Unknowns = append([]string(nil), summary.Blockers...)
	for _, item := range detail.Evidence {
		observedAt, verifiedAt := "", ""
		if item.ObservedAt != nil {
			observedAt = item.ObservedAt.UTC().Format(time.RFC3339)
		}
		if item.VerifiedAt != nil {
			verifiedAt = item.VerifiedAt.UTC().Format(time.RFC3339)
		}
		response.Evidence = append(response.Evidence, EvidenceItem{ID: item.ID, Title: item.Title, TruthStatus: item.TruthStatus, SourceURL: item.SourceURI, ObservedAt: observedAt, Summary: item.EvidenceKind + "/" + item.Stage, VerifiedBy: item.VerifiedBy, VerifiedAt: verifiedAt})
		if item.TruthStatus == experiment.TruthUnknown || item.TruthStatus == experiment.TruthMock || item.TruthStatus == experiment.TruthInferred {
			response.Unknowns = append(response.Unknowns, item.Title+"（"+item.TruthStatus+"）")
		}
	}
	if summary.FinalProfitStatus != experiment.ProfitFinal {
		response.Unknowns = append(response.Unknowns, "最终利润尚未确认（"+summary.FinalProfitStatus+"）")
	}
	if summary.CashRecoveryStatus != experiment.CashRecovered {
		response.Unknowns = append(response.Unknowns, "现金回收尚未确认（"+summary.CashRecoveryStatus+"）")
	}
	response.Links = []ResponseLink{
		{Label: "经营实验", Href: "/experiments/" + response.ExperimentID},
		{Label: "实验闸门状态", Href: "/api/v1/experiments/" + response.ExperimentID + "/owner-summary"},
		{Label: "小Q 执行记录", Href: "/xiaoq/traces/" + response.TraceID},
	}
	response.Provenance = Provenance{Provider: response.Provider, Model: response.Model, TokensIn: response.TokensIn, TokensOut: response.TokensOut, LatencyMs: response.LatencyMs}
}

func (s *Service) addGrounding(response *MessageResponse, detail *demandcase.Detail, card *demandcase.OwnerDecisionCard) {
	evidenceCount := 0
	if detail != nil {
		evidenceCount = len(detail.Evidence)
	}
	response.Evidence = make([]EvidenceItem, 0, evidenceCount)
	response.Unknowns = make([]string, 0)
	if detail != nil {
		snapshotHashes := make(map[int64]string, len(detail.Snapshots))
		for _, snapshot := range detail.Snapshots {
			snapshotHashes[snapshot.ID] = snapshot.RawSHA256
		}
		for _, item := range detail.Evidence {
			observedAt := ""
			if item.ObservedAt != nil {
				observedAt = item.ObservedAt.UTC().Format(time.RFC3339)
			}
			response.Evidence = append(response.Evidence, EvidenceItem{ID: item.ID, Title: item.Title, TruthStatus: item.TruthStatus, SourceURL: item.SourceURI, ObservedAt: observedAt, Summary: item.Kind + "/" + item.Dimension, RunID: item.RunID, SnapshotID: item.SnapshotID, SnapshotSHA256: snapshotHashes[item.SnapshotID]})
			if item.TruthStatus == demandcase.TruthUnknown || item.TruthStatus == demandcase.TruthMock || item.TruthStatus == demandcase.TruthInferred {
				response.Unknowns = append(response.Unknowns, item.Title+"（"+item.TruthStatus+"）")
			}
		}
	}
	if card != nil && strings.TrimSpace(card.NotProven) != "" {
		response.Unknowns = append(response.Unknowns, card.NotProven)
	}
	response.Links = []ResponseLink{
		{Label: "候选市场案件", Href: "/demand-cases/" + fmt.Sprint(response.DemandCaseID)},
		{Label: "Owner 决策卡", Href: "/api/v1/demand-cases/" + fmt.Sprint(response.DemandCaseID) + "/decision-card"},
		{Label: "小Q 执行记录", Href: "/xiaoq/traces/" + response.TraceID},
	}
	response.Provenance = Provenance{Provider: response.Provider, Model: response.Model, TokensIn: response.TokensIn, TokensOut: response.TokensOut, LatencyMs: response.LatencyMs}
}

func (s *Service) complete(traceID string, response *MessageResponse, status string) error {
	out, _ := json.Marshal(response)
	_, err := s.traces.Complete(traceID, &ai.CompleteTraceInput{FinalOutput: out, ModelName: response.Model, RiskLevel: "low", TokenCount: response.TokensIn + response.TokensOut, Status: status})
	return err
}

func (s *Service) startReadTrace(ownerID int64, decisionPoint, promptVersion string, input map[string]interface{}) (string, error) {
	return s.traces.Start(&ai.CreateTraceInput{
		AgentID: AgentID, DecisionPoint: decisionPoint, UserID: &ownerID,
		ModelProvider: s.provider.Name(), PromptVersion: promptVersion, InputContext: mustJSON(input),
	})
}

func (s *Service) capabilityFailure(traceID, capabilityID string, cause error, target map[string]interface{}) error {
	payload := map[string]interface{}{"risk": "read", "status": "failed", "error": cause.Error()}
	for key, value := range target {
		payload[key] = value
	}
	_, appendErr := s.traces.AppendEvent(traceID, &ai.AppendEventInput{EventType: "capability_failed", Content: capabilityID, Payload: mustJSON(payload)})
	failure := mustJSON(map[string]interface{}{"capability": capabilityID, "error": cause.Error()})
	_, completeErr := s.traces.Complete(traceID, &ai.CompleteTraceInput{FinalOutput: failure, RiskLevel: "low", Status: "failed"})
	return &RunError{TraceID: traceID, Err: errors.Join(cause, appendErr, completeErr)}
}

func (s *Service) failTrace(traceID string, cause error) error {
	failure := mustJSON(map[string]interface{}{"error": cause.Error()})
	_, _ = s.traces.Complete(traceID, &ai.CompleteTraceInput{FinalOutput: failure, RiskLevel: "low", Status: "failed"})
	return &RunError{TraceID: traceID, Err: cause}
}

func (s *Service) GetTrace(ctx context.Context, ownerID int64, traceID string) (*ai.TraceDetail, error) {
	if ownerID <= 0 || strings.TrimSpace(traceID) == "" {
		return nil, ErrTraceNotFound
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&ai.AITrace{}).Where("trace_id = ? AND user_id = ? AND agent_id = ?", traceID, ownerID, AgentID).Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, ErrTraceNotFound
	}
	return s.traces.GetDetail(traceID)
}

func mustJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
