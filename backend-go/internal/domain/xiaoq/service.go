package xiaoq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/domain/demandcase"
	"github.com/lingmirror/backend-go/internal/domain/experiment"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service struct {
	db         *gorm.DB
	logger     *zap.Logger
	demand     DemandCaseReader
	experiment ExperimentReader
	provider   ai.LLMProvider
	traces     TraceRecorder
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
		if strings.TrimSpace(in.ExperimentID) == "" || in.DemandCaseID != 0 || s.experiment == nil {
			return nil, ErrInvalidInput
		}
		return s.sendExperimentMessage(ctx, ownerID, message, in.ExperimentID)
	}
	if target != TargetDemandCase || in.DemandCaseID <= 0 || strings.TrimSpace(in.ExperimentID) != "" || s.demand == nil {
		return nil, ErrInvalidInput
	}
	if _, ok := activeCapability(CapabilityDemandCaseRead); !ok {
		return nil, ErrCapabilityUnavailable
	}
	if _, ok := activeCapability(CapabilityDemandCaseDecisionRead); !ok {
		return nil, ErrCapabilityUnavailable
	}
	traceID, err := s.startReadTrace(ownerID, "demand_case_explain", "xiaoq-v1", map[string]interface{}{"target_type": TargetDemandCase, "question": message, "demand_case_id": in.DemandCaseID})
	if err != nil {
		return nil, err
	}
	detail, err := s.demand.Get(ctx, in.DemandCaseID, ownerID)
	if err != nil {
		return nil, s.capabilityFailure(traceID, CapabilityDemandCaseRead, err, map[string]interface{}{"demand_case_id": in.DemandCaseID})
	}
	if _, err := s.traces.AppendEvent(traceID, &ai.AppendEventInput{EventType: "capability_call", Content: CapabilityDemandCaseRead, Payload: mustJSON(map[string]interface{}{"demand_case_id": in.DemandCaseID, "risk": "read", "status": "succeeded"})}); err != nil {
		return nil, s.failTrace(traceID, err)
	}
	card, err := s.demand.DecisionCard(ctx, in.DemandCaseID, ownerID)
	if err != nil {
		return nil, s.capabilityFailure(traceID, CapabilityDemandCaseDecisionRead, err, map[string]interface{}{"demand_case_id": in.DemandCaseID})
	}
	if _, err := s.traces.AppendEvent(traceID, &ai.AppendEventInput{EventType: "capability_call", Content: CapabilityDemandCaseDecisionRead, Payload: mustJSON(map[string]interface{}{"demand_case_id": in.DemandCaseID, "risk": "read", "status": "succeeded"})}); err != nil {
		return nil, s.failTrace(traceID, err)
	}
	contextPayload := struct {
		Question     string      `json:"question"`
		DemandCaseID int64       `json:"demand_case_id"`
		Detail       interface{} `json:"detail"`
		DecisionCard interface{} `json:"decision_card"`
		Capabilities []string    `json:"capabilities"`
	}{message, in.DemandCaseID, detail, card, []string{CapabilityDemandCaseRead, CapabilityDemandCaseDecisionRead}}
	inputJSON, err := json.Marshal(contextPayload)
	if err != nil {
		return nil, err
	}
	snapshotHashes := make(map[int64]string, len(detail.Snapshots))
	for _, snapshot := range detail.Snapshots {
		snapshotHashes[snapshot.ID] = snapshot.RawSHA256
	}
	for _, ev := range detail.Evidence {
		observedAt := ""
		if ev.ObservedAt != nil {
			observedAt = ev.ObservedAt.UTC().Format(time.RFC3339)
		}
		payload := map[string]interface{}{"demand_case_id": in.DemandCaseID, "kind": ev.Kind, "dimension": ev.Dimension, "truth_status": ev.TruthStatus, "source_uri": ev.SourceURI, "run_id": ev.RunID, "snapshot_id": ev.SnapshotID, "observed_at": observedAt}
		if hash := snapshotHashes[ev.SnapshotID]; hash != "" {
			payload["snapshot_sha256"] = hash
		}
		if _, err := s.traces.AddEvidence(traceID, &ai.AddEvidenceInput{SourceType: "demand_evidence", SourceID: fmt.Sprint(ev.ID), Title: ev.Title, Summary: ev.TruthStatus, Payload: mustJSON(payload)}); err != nil {
			return nil, s.failTrace(traceID, err)
		}
	}

	if s.provider.Name() == "stub" {
		answer := "这是模拟回答，不能作为可信经营建议。当前案件裁决为「" + card.Verdict + "」；请查看案件证据和未知项。"
		final := &MessageResponse{TraceID: traceID, AgentID: AgentID, Mode: "read_only_v1", TargetType: TargetDemandCase, DemandCaseID: in.DemandCaseID, Answer: answer, TruthStatus: TruthMock, Trusted: false, Provider: "stub", Model: "stub-v1"}
		s.addGrounding(final, detail, card)
		if err := s.complete(traceID, final, "completed"); err != nil {
			return nil, &RunError{TraceID: traceID, Err: err}
		}
		return final, nil
	}

	request := &ai.LLMRequest{
		System:   "你是凌镜的小Q。只能根据提供的候选市场案件与决策卡回答。明确区分有来源事实、推断和未知；不得改变系统裁决，不得声称已成交、已盈利或可自动执行。用简明中文说明当前结论、最强反证/缺口和下一步。",
		Messages: []ai.LLMMessage{{Role: "user", Content: string(inputJSON)}}, MaxTokens: 800,
		Metadata: map[string]interface{}{"agent_id": AgentID, "demand_case_id": in.DemandCaseID},
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
	final := &MessageResponse{TraceID: traceID, AgentID: AgentID, Mode: "read_only_v1", TargetType: TargetDemandCase, DemandCaseID: in.DemandCaseID, Answer: resp.Answer, TruthStatus: TruthInferred, Trusted: false, Provider: s.provider.Name(), Model: resp.Model, TokensIn: resp.TokensIn, TokensOut: resp.TokensOut, LatencyMs: resp.LatencyMs}
	s.addGrounding(final, detail, card)
	if _, err := s.traces.AppendEvent(traceID, &ai.AppendEventInput{EventType: "model_response", Content: "provider response received", Payload: mustJSON(map[string]interface{}{"provider": final.Provider, "model": final.Model, "tokens_in": final.TokensIn, "tokens_out": final.TokensOut, "truth_status": final.TruthStatus})}); err != nil {
		return nil, s.failTrace(traceID, err)
	}
	if err := s.complete(traceID, final, "completed"); err != nil {
		return nil, &RunError{TraceID: traceID, Err: err}
	}
	return final, nil
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
		System:   "你是凌镜的小Q。只能根据提供的经营实验案件与闸门状态回答。严格保留 actual/quoted/estimated/unknown/mock/inferred；不得新增或核验证据、通过闸门、改变实验状态，也不得把待定利润或现金说成已最终确认。用简明中文说明当前阶段、已有证据、阻断项、未知和下一步。",
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
		{Label: "小Q 执行记录", Href: "/api/v1/xiao-q/traces/" + response.TraceID},
	}
	response.Provenance = Provenance{Provider: response.Provider, Model: response.Model, TokensIn: response.TokensIn, TokensOut: response.TokensOut, LatencyMs: response.LatencyMs}
}

func (s *Service) addGrounding(response *MessageResponse, detail *demandcase.Detail, card *demandcase.OwnerDecisionCard) {
	response.Evidence = make([]EvidenceItem, 0, len(detail.Evidence))
	response.Unknowns = make([]string, 0)
	snapshotHashes := make(map[int64]string, len(detail.Snapshots))
	for _, snapshot := range detail.Snapshots {
		snapshotHashes[snapshot.ID] = snapshot.RawSHA256
	}
	for _, item := range detail.Evidence {
		observedAt := ""
		if item.ObservedAt != nil {
			observedAt = item.ObservedAt.UTC().Format(time.RFC3339)
		}
		response.Evidence = append(response.Evidence, EvidenceItem{
			ID: item.ID, Title: item.Title, TruthStatus: item.TruthStatus,
			SourceURL: item.SourceURI, ObservedAt: observedAt,
			Summary: item.Kind + "/" + item.Dimension, RunID: item.RunID,
			SnapshotID: item.SnapshotID, SnapshotSHA256: snapshotHashes[item.SnapshotID],
		})
		if item.TruthStatus == demandcase.TruthUnknown || item.TruthStatus == demandcase.TruthMock || item.TruthStatus == demandcase.TruthInferred {
			response.Unknowns = append(response.Unknowns, item.Title+"（"+item.TruthStatus+"）")
		}
	}
	if strings.TrimSpace(card.NotProven) != "" {
		response.Unknowns = append(response.Unknowns, card.NotProven)
	}
	response.Links = []ResponseLink{
		{Label: "候选市场案件", Href: "/demand-cases/" + fmt.Sprint(response.DemandCaseID)},
		{Label: "Owner 决策卡", Href: "/api/v1/demand-cases/" + fmt.Sprint(response.DemandCaseID) + "/decision-card"},
		{Label: "小Q 执行记录", Href: "/api/v1/xiao-q/traces/" + response.TraceID},
	}
	response.Provenance = Provenance{Provider: response.Provider, Model: response.Model, TokensIn: response.TokensIn, TokensOut: response.TokensOut, LatencyMs: response.LatencyMs}
}

func (s *Service) complete(traceID string, response *MessageResponse, status string) error {
	out, _ := json.Marshal(response)
	_, err := s.traces.Complete(traceID, &ai.CompleteTraceInput{FinalOutput: out, RiskLevel: "low", TokenCount: response.TokensIn + response.TokensOut, Status: status})
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
