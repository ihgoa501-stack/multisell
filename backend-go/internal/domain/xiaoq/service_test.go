package xiaoq

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/demandcase"
	"github.com/lingmirror/backend-go/internal/domain/experiment"
	"github.com/lingmirror/backend-go/internal/domain/sourcing1688"
	"gorm.io/gorm"
)

type failingTraceRecorder struct {
	failAt string
	trace  *ai.TraceWriter
}

func (f *failingTraceRecorder) Start(in *ai.CreateTraceInput) (string, error) {
	if f.failAt == "start" {
		return "", errors.New("start failed")
	}
	return f.trace.Start(in)
}
func (f *failingTraceRecorder) AppendEvent(id string, in *ai.AppendEventInput) (*ai.AITraceEvent, error) {
	if f.failAt == "append" {
		return nil, errors.New("append failed")
	}
	return f.trace.AppendEvent(id, in)
}
func (f *failingTraceRecorder) AddEvidence(id string, in *ai.AddEvidenceInput) (*ai.AIEvidenceRef, error) {
	if f.failAt == "evidence" {
		return nil, errors.New("evidence failed")
	}
	return f.trace.AddEvidence(id, in)
}
func (f *failingTraceRecorder) Complete(id string, in *ai.CompleteTraceInput) (*ai.AITrace, error) {
	if f.failAt == "complete" {
		return nil, errors.New("complete failed")
	}
	return f.trace.Complete(id, in)
}
func (f *failingTraceRecorder) GetDetail(id string) (*ai.TraceDetail, error) {
	return f.trace.GetDetail(id)
}

type fakeDemandReader struct {
	detail *demandcase.Detail
	card   *demandcase.OwnerDecisionCard
	err    error
}

type fakeExperimentReader struct {
	detail      *experiment.Detail
	summary     *experiment.OwnerSummary
	ownerIDSeen int64
	detailErr   error
	summaryErr  error
}

type fakeSourcingReader struct {
	view         *sourcing1688.OwnerView
	err          error
	ownerIDSeen  int64
	sourceIDSeen int64
}

func (f *fakeSourcingReader) ReadOwnerView(_ context.Context, sourceID, ownerID int64) (*sourcing1688.OwnerView, error) {
	f.sourceIDSeen, f.ownerIDSeen = sourceID, ownerID
	return f.view, f.err
}

func (f *fakeExperimentReader) GetDetail(_ context.Context, _ string, ownerID int64) (*experiment.Detail, error) {
	f.ownerIDSeen = ownerID
	return f.detail, f.detailErr
}

func (f *fakeExperimentReader) OwnerSummary(_ context.Context, _ string, ownerID int64) (*experiment.OwnerSummary, error) {
	f.ownerIDSeen = ownerID
	return f.summary, f.summaryErr
}

func (f *fakeDemandReader) Get(context.Context, int64, int64) (*demandcase.Detail, error) {
	return f.detail, f.err
}

func (f *fakeDemandReader) DecisionCard(context.Context, int64, int64) (*demandcase.OwnerDecisionCard, error) {
	return f.card, f.err
}

type fakeProvider struct {
	name         string
	resp         *ai.LLMResponse
	err          error
	req          *ai.LLMRequest
	deadlineSeen bool
}

func (f *fakeProvider) Name() string { return f.name }
func (f *fakeProvider) Chat(ctx context.Context, req *ai.LLMRequest) (*ai.LLMResponse, error) {
	f.req = req
	_, f.deadlineSeen = ctx.Deadline()
	return f.resp, f.err
}
func (f *fakeProvider) ChatStream(context.Context, *ai.LLMRequest) (<-chan ai.LLMChunk, error) {
	return nil, errors.New("not used")
}

func testDemandData() (*demandcase.Detail, *demandcase.OwnerDecisionCard) {
	observed := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)
	detail := &demandcase.Detail{
		Case: demandcase.DemandCase{ID: 7, OwnerID: 42, Region: "测试地区", Consumer: "目标消费者", NeedScenario: "测试场景", SalesChannel: "测试渠道", Status: demandcase.VerdictEvidenceMissing},
		Evidence: []demandcase.DemandEvidence{
			{ID: 9, DemandCaseID: 7, Dimension: demandcase.DimensionDemand, Kind: demandcase.EvidenceSupport, TruthStatus: demandcase.TruthQuoted, Title: "公开来源线索", SourceURI: "https://example.com/source", ObservedAt: &observed, RunID: "run-1", SnapshotID: 3},
			{ID: 10, DemandCaseID: 7, Dimension: demandcase.DimensionProfit, Kind: demandcase.EvidenceSupport, TruthStatus: demandcase.TruthInferred, Title: "尚未核验利润", RunID: "run-1"},
		},
		Verdict:   &demandcase.DemandVerdict{DemandCaseID: 7, Status: demandcase.VerdictEvidenceMissing, Blockers: []string{"missing:competition"}},
		Snapshots: []demandcase.ResearchSnapshot{{ID: 3, RunID: "run-1", RawSHA256: strings.Repeat("a", 64)}},
	}
	card := &demandcase.OwnerDecisionCard{DemandCaseID: 7, Verdict: demandcase.VerdictEvidenceMissing, Proven: "只有公开研究线索", NotProven: "未证明真实成交与利润", StrongestCounterevidence: "尚无独立反证", NextAuthorityOrCost: "继续补证"}
	return detail, card
}

func newTestService(t *testing.T, provider ai.LLMProvider) *Service {
	t.Helper()
	detail, card := testDemandData()
	db := dbtest.NewDB(t, &ai.AITrace{}, &ai.AITraceEvent{}, &ai.AIEvidenceRef{}, &ai.UnifiedAction{})
	return NewService(db, dbtest.NewLogger(t), &fakeDemandReader{detail: detail, card: card}, nil, provider, ai.NewTraceWriter(db, dbtest.NewLogger(t)))
}

func testExperimentData() (*experiment.Detail, *experiment.OwnerSummary) {
	observed := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	detail := &experiment.Detail{
		Case:  experiment.ExperimentCase{ExperimentID: "exp_test", OwnerID: 42, Name: "真实商品实验", Stage: experiment.StageOrder, Status: experiment.StatusActive, FinalProfitStatus: experiment.ProfitPending, CashRecoveryStatus: experiment.CashPending},
		Gates: []experiment.GateDecision{{ID: 11, ExperimentID: "exp_test", Stage: experiment.StageOpportunity, GateCode: "demand_evidence", Result: experiment.ResultPass}},
		Evidence: []experiment.EvidenceRecord{
			{ID: 21, ExperimentID: "exp_test", Stage: experiment.StageOrder, EvidenceKind: "support", TruthStatus: experiment.TruthActual, Title: "Owner 已核验付款凭证", SourceURI: "internal://orders/8", ObservedAt: &observed, VerifiedBy: 42, VerifiedAt: &observed},
			{ID: 22, ExperimentID: "exp_test", Stage: experiment.StageOrder, EvidenceKind: "counter", TruthStatus: experiment.TruthUnknown, Title: "买家关联关系尚未核验"},
		},
	}
	summary := &experiment.OwnerSummary{ExperimentID: "exp_test", Stage: experiment.StageOrder, PassedGates: 1, Blockers: []string{"order:gate_missing", "profit:gate_missing"}, FinalProfitStatus: experiment.ProfitPending, CashRecoveryStatus: experiment.CashPending}
	return detail, summary
}

func TestSendMessageReadsOwnerScopedExperimentWithoutMutation(t *testing.T) {
	detail, summary := testExperimentData()
	reader := &fakeExperimentReader{detail: detail, summary: summary}
	db := dbtest.NewDB(t, &ai.AITrace{}, &ai.AITraceEvent{}, &ai.AIEvidenceRef{}, &ai.UnifiedAction{})
	svc := NewService(db, dbtest.NewLogger(t), nil, reader, &fakeProvider{name: "stub"}, ai.NewTraceWriter(db, dbtest.NewLogger(t)))

	got, err := svc.SendMessage(context.Background(), 42, MessageInput{Message: "这个实验卡在哪里？", TargetType: TargetExperiment, ExperimentID: "exp_test"})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if reader.ownerIDSeen != 42 {
		t.Fatalf("experiment reader owner = %d, want 42", reader.ownerIDSeen)
	}
	if got.TargetType != TargetExperiment || got.ExperimentID != "exp_test" || got.DemandCaseID != 0 {
		t.Fatalf("wrong target grounding: %#v", got)
	}
	if len(got.Evidence) != 2 || got.Evidence[0].TruthStatus != experiment.TruthActual {
		t.Fatalf("experiment evidence not preserved: %#v", got.Evidence)
	}
	if got.Evidence[0].VerifiedBy != 42 || got.Evidence[0].VerifiedAt == "" {
		t.Fatalf("experiment verification provenance missing: %#v", got.Evidence[0])
	}
	if !strings.Contains(strings.Join(got.Unknowns, "|"), "买家关联关系尚未核验") || !strings.Contains(strings.Join(got.Unknowns, "|"), "order:gate_missing") {
		t.Fatalf("missing experiment unknowns/blockers: %#v", got.Unknowns)
	}
	if len(got.Links) != 3 || !strings.Contains(got.Links[0].Href, "exp_test") {
		t.Fatalf("missing experiment links: %#v", got.Links)
	}
	trace, traceErr := svc.GetTrace(context.Background(), 42, got.TraceID)
	if traceErr != nil || len(trace.Evidence) != 2 {
		t.Fatalf("experiment trace evidence = %#v, err=%v", trace, traceErr)
	}
}

func TestSendMessageReadsControlledSourcingDraftWithTraceAndBoundaries(t *testing.T) {
	view := &sourcing1688.OwnerView{
		Source:      sourcing1688.OwnerSourceView{ID: 8, DemandCaseID: 7, ExperimentID: "exp-owner", SnapshotID: 9, LifecycleStatus: "editing"},
		Snapshot:    sourcing1688.OwnerSnapshotView{ID: 9, RawSHA256: strings.Repeat("a", 64), SourceReference: "detail.1688.com/offer/8.html"},
		Draft:       &sourcing1688.OwnerDraftView{ID: 12, ProductID: 10, ListingID: 11, DemandCaseID: 7, ExperimentID: "exp-owner", SnapshotID: 9, ApprovalStatus: "editing"},
		Costs:       []sourcing1688.OwnerCostView{{ID: 15, CostType: "purchase", Amount: 12.5, Currency: "CNY", TruthStatus: "quoted", SourceReference: ""}},
		Limitations: []string{"只读视图", "成本尚未外部核验"},
	}
	reader := &fakeSourcingReader{view: view}
	db := dbtest.NewDB(t, &ai.AITrace{}, &ai.AITraceEvent{}, &ai.AIEvidenceRef{}, &ai.UnifiedAction{})
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, &fakeProvider{name: "stub"}, ai.NewTraceWriter(db, dbtest.NewLogger(t))).WithSourcingReader(reader)

	got, err := svc.SendMessage(context.Background(), 42, MessageInput{Message: "这个1688草稿是否可用？", TargetType: TargetSourcing1688, SourceID: 8})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if reader.ownerIDSeen != 42 || reader.sourceIDSeen != 8 || got.TargetType != TargetSourcing1688 || got.SourceID != 8 {
		t.Fatalf("owner/target not preserved: reader=%#v response=%#v", reader, got)
	}
	if got.TruthStatus != TruthMock || got.Trusted || len(got.Evidence) < 2 || !strings.Contains(strings.Join(got.Unknowns, "|"), "成本尚未外部核验") {
		t.Fatalf("grounding boundary missing: %#v", got)
	}
	trace, traceErr := svc.GetTrace(context.Background(), 42, got.TraceID)
	if traceErr != nil || len(trace.Evidence) < 2 || len(trace.Events) == 0 || trace.Events[0].Content != CapabilitySourcing1688Read {
		t.Fatalf("sourcing trace=%#v err=%v", trace, traceErr)
	}
}

func TestSourcingProviderGetsRedactedPayloadAndFiveSecondDeadline(t *testing.T) {
	view := &sourcing1688.OwnerView{
		Source:      sourcing1688.OwnerSourceView{ID: 8, SourceReference: "detail.1688.com/offer/8.html", DemandCaseID: 7, ExperimentID: "exp-owner", SnapshotID: 9, LifecycleStatus: "editing"},
		Snapshot:    sourcing1688.OwnerSnapshotView{ID: 9, SourceReference: "detail.1688.com/offer/8.html", RawSHA256: strings.Repeat("a", 64)},
		Draft:       &sourcing1688.OwnerDraftView{ID: 12, DemandCaseID: 7, ExperimentID: "exp-owner", SnapshotID: 9, ProductID: 10, ListingID: 11, CreatedBy: 42},
		Media:       []sourcing1688.OwnerMediaView{{ID: 13, MediaRole: "main", RightsStatus: "verified", TruthStatus: "unknown"}},
		Limitations: []string{"只读"},
	}
	provider := &fakeProvider{name: "openai", resp: &ai.LLMResponse{Answer: "分析", Model: "test"}}
	db := dbtest.NewDB(t, &ai.AITrace{}, &ai.AITraceEvent{}, &ai.AIEvidenceRef{}, &ai.UnifiedAction{})
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, provider, ai.NewTraceWriter(db, dbtest.NewLogger(t))).WithSourcingReader(&fakeSourcingReader{view: view})
	if _, err := svc.SendMessage(context.Background(), 42, MessageInput{Message: "读取", TargetType: TargetSourcing1688, SourceID: 8}); err != nil {
		t.Fatal(err)
	}
	if provider.req == nil || !provider.deadlineSeen {
		t.Fatalf("provider request/deadline missing: %#v", provider)
	}
	payload := provider.req.Messages[0].Content
	for _, forbidden := range []string{"secret", "?", "description", "supplier", "platform_sku", "internal://", "processed_url", "rights_evidence_uri"} {
		if strings.Contains(strings.ToLower(payload), forbidden) {
			t.Fatalf("provider payload leaked %q: %s", forbidden, payload)
		}
	}
}

func TestSourcingReadDoesNotCrossOwnerBoundary(t *testing.T) {
	reader := &fakeSourcingReader{err: sourcing1688.ErrWorkflowGate}
	db := dbtest.NewDB(t, &ai.AITrace{}, &ai.AITraceEvent{}, &ai.AIEvidenceRef{}, &ai.UnifiedAction{})
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, &fakeProvider{name: "stub"}, ai.NewTraceWriter(db, dbtest.NewLogger(t))).WithSourcingReader(reader)
	_, err := svc.SendMessage(context.Background(), 99, MessageInput{Message: "读取", TargetType: TargetSourcing1688, SourceID: 8})
	var runErr *RunError
	if !errors.As(err, &runErr) || !errors.Is(err, sourcing1688.ErrWorkflowGate) || reader.ownerIDSeen != 99 {
		t.Fatalf("cross-owner sourcing error=%T %v owner=%d", err, err, reader.ownerIDSeen)
	}
}

func TestExperimentReadDoesNotCrossOwnerBoundary(t *testing.T) {
	reader := &fakeExperimentReader{detailErr: gorm.ErrRecordNotFound}
	db := dbtest.NewDB(t, &ai.AITrace{}, &ai.AITraceEvent{}, &ai.AIEvidenceRef{}, &ai.UnifiedAction{})
	svc := NewService(db, dbtest.NewLogger(t), nil, reader, &fakeProvider{name: "stub"}, ai.NewTraceWriter(db, dbtest.NewLogger(t)))
	_, err := svc.SendMessage(context.Background(), 99, MessageInput{Message: "读取", TargetType: TargetExperiment, ExperimentID: "exp_owner_42"})
	if !errors.Is(err, gorm.ErrRecordNotFound) || reader.ownerIDSeen != 99 {
		t.Fatalf("cross-owner read err=%v owner=%d", err, reader.ownerIDSeen)
	}
}

func TestExperimentReadUsesRealDomainOwnerIsolation(t *testing.T) {
	db := dbtest.NewDB(t, &experiment.ExperimentCase{}, &experiment.GateDecision{}, &experiment.EvidenceRecord{}, &experiment.ObjectLink{}, &ai.AITrace{}, &ai.AITraceEvent{}, &ai.AIEvidenceRef{}, &ai.UnifiedAction{})
	logger := dbtest.NewLogger(t)
	experimentService := experiment.NewService(db, logger)
	c := &experiment.ExperimentCase{Name: "Owner 42 experiment", Stage: experiment.StageOpportunity, OwnerID: 42}
	if err := experimentService.Create(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, logger, nil, experimentService, &fakeProvider{name: "stub"}, ai.NewTraceWriter(db, logger))
	_, err := svc.SendMessage(context.Background(), 99, MessageInput{Message: "读取", TargetType: TargetExperiment, ExperimentID: c.ExperimentID})
	var runErr *RunError
	if !errors.As(err, &runErr) || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("real domain cross-owner error=%T %v", err, err)
	}
	trace, traceErr := svc.GetTrace(context.Background(), 99, runErr.TraceID)
	if traceErr != nil || trace.Trace.Status != "failed" {
		t.Fatalf("cross-owner attempt trace=%#v err=%v", trace, traceErr)
	}
}

func TestExperimentDomainReadFailurePersistsFailedTrace(t *testing.T) {
	reader := &fakeExperimentReader{detailErr: errors.New("experiment store unavailable")}
	db := dbtest.NewDB(t, &ai.AITrace{}, &ai.AITraceEvent{}, &ai.AIEvidenceRef{}, &ai.UnifiedAction{})
	svc := NewService(db, dbtest.NewLogger(t), nil, reader, &fakeProvider{name: "stub"}, ai.NewTraceWriter(db, dbtest.NewLogger(t)))
	_, err := svc.SendMessage(context.Background(), 42, MessageInput{Message: "读取", TargetType: TargetExperiment, ExperimentID: "exp_test"})
	var runErr *RunError
	if !errors.As(err, &runErr) || runErr.TraceID == "" {
		t.Fatalf("expected traced RunError, got %T %v", err, err)
	}
	trace, getErr := svc.GetTrace(context.Background(), 42, runErr.TraceID)
	if getErr != nil || trace.Trace.Status != "failed" || len(trace.Events) != 1 || trace.Events[0].EventType != "capability_failed" || trace.Events[0].Content != CapabilityExperimentRead {
		t.Fatalf("failed domain trace=%#v err=%v", trace, getErr)
	}
	if strings.Contains(string(trace.Trace.InputContext), "store unavailable") || strings.Contains(string(trace.Trace.InputContext), "detail") {
		t.Fatalf("trace input must contain only request target: %s", trace.Trace.InputContext)
	}
}

func TestExperimentSummaryFailureNamesGateCapability(t *testing.T) {
	detail, _ := testExperimentData()
	reader := &fakeExperimentReader{detail: detail, summaryErr: errors.New("summary unavailable")}
	db := dbtest.NewDB(t, &ai.AITrace{}, &ai.AITraceEvent{}, &ai.AIEvidenceRef{}, &ai.UnifiedAction{})
	svc := NewService(db, dbtest.NewLogger(t), nil, reader, &fakeProvider{name: "stub"}, ai.NewTraceWriter(db, dbtest.NewLogger(t)))
	_, err := svc.SendMessage(context.Background(), 42, MessageInput{Message: "读取", TargetType: TargetExperiment, ExperimentID: "exp_test"})
	var runErr *RunError
	if !errors.As(err, &runErr) {
		t.Fatalf("expected RunError, got %v", err)
	}
	trace, getErr := svc.GetTrace(context.Background(), 42, runErr.TraceID)
	if getErr != nil || len(trace.Events) != 2 || trace.Events[1].EventType != "capability_failed" || trace.Events[1].Content != CapabilityExperimentGateRead {
		t.Fatalf("summary failure trace=%#v err=%v", trace, getErr)
	}
}

func TestExperimentProviderFailureMarksTraceFailed(t *testing.T) {
	detail, summary := testExperimentData()
	reader := &fakeExperimentReader{detail: detail, summary: summary}
	db := dbtest.NewDB(t, &ai.AITrace{}, &ai.AITraceEvent{}, &ai.AIEvidenceRef{}, &ai.UnifiedAction{})
	svc := NewService(db, dbtest.NewLogger(t), nil, reader, &fakeProvider{name: "openai", err: errors.New("provider unavailable")}, ai.NewTraceWriter(db, dbtest.NewLogger(t)))
	_, err := svc.SendMessage(context.Background(), 42, MessageInput{Message: "读取", TargetType: TargetExperiment, ExperimentID: "exp_test"})
	var runErr *RunError
	if !errors.As(err, &runErr) {
		t.Fatalf("expected RunError, got %v", err)
	}
	trace, getErr := svc.GetTrace(context.Background(), 42, runErr.TraceID)
	if getErr != nil || trace.Trace.Status != "failed" || len(trace.Evidence) != 2 {
		t.Fatalf("provider failure trace=%#v err=%v", trace, getErr)
	}
}

func TestExperimentTraceWriteFailuresCannotReturnSuccess(t *testing.T) {
	for _, failAt := range []string{"append", "evidence", "complete"} {
		t.Run(failAt, func(t *testing.T) {
			detail, summary := testExperimentData()
			db := dbtest.NewDB(t, &ai.AITrace{}, &ai.AITraceEvent{}, &ai.AIEvidenceRef{}, &ai.UnifiedAction{})
			writer := ai.NewTraceWriter(db, dbtest.NewLogger(t))
			recorder := &failingTraceRecorder{failAt: failAt, trace: writer}
			svc := NewService(db, dbtest.NewLogger(t), nil, &fakeExperimentReader{detail: detail, summary: summary}, &fakeProvider{name: "stub"}, recorder)
			if _, err := svc.SendMessage(context.Background(), 42, MessageInput{Message: "读取", TargetType: TargetExperiment, ExperimentID: "exp_test"}); err == nil {
				t.Fatalf("experiment %s trace failure returned success", failAt)
			}
		})
	}
}

func TestLegacyDemandCaseInputRemainsCompatible(t *testing.T) {
	svc := newTestService(t, &fakeProvider{name: "stub"})
	got, err := svc.SendMessage(context.Background(), 42, MessageInput{Message: "查看案件", DemandCaseID: 7})
	if err != nil || got.TargetType != TargetDemandCase || got.DemandCaseID != 7 {
		t.Fatalf("legacy demand case input got=%#v err=%v", got, err)
	}
}

func TestSendMessageRejectsConflictingTargetFields(t *testing.T) {
	svc := newTestService(t, &fakeProvider{name: "stub"})
	for _, in := range []MessageInput{
		{Message: "冲突", TargetType: TargetExperiment, ExperimentID: "exp_test", DemandCaseID: 7},
		{Message: "冲突", TargetType: TargetDemandCase, DemandCaseID: 7, ExperimentID: "exp_test"},
	} {
		if _, err := svc.SendMessage(context.Background(), 42, in); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("conflicting input %#v error=%v", in, err)
		}
	}
}

func TestSendMessageStubIsExplicitMockAndNotTrusted(t *testing.T) {
	svc := newTestService(t, &fakeProvider{name: "stub", resp: &ai.LLMResponse{Answer: "固定演示回答", Model: "stub-v1", TokensIn: 12, TokensOut: 4}})

	got, err := svc.SendMessage(context.Background(), 42, MessageInput{Message: "这个候选市场目前如何？", DemandCaseID: 7})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if got.AgentID != AgentID || got.TruthStatus != TruthMock || got.Trusted {
		t.Fatalf("got identity/truth/trusted = %q/%q/%v", got.AgentID, got.TruthStatus, got.Trusted)
	}
	if got.Mode != "read_only_v1" || len(got.Evidence) != 2 || len(got.Unknowns) == 0 {
		t.Fatalf("missing structured grounding: %#v", got)
	}
	if len(got.Links) != 3 || got.Links[0].Href == "" || got.Links[1].Href == "" || got.Links[2].Href == "" {
		t.Fatalf("missing traceable links: %#v", got.Links)
	}
	if got.Provenance.Provider != "stub" || got.Provenance.Model != "stub-v1" {
		t.Fatalf("missing provenance: %#v", got.Provenance)
	}
	b, _ := json.Marshal(got)
	var shape map[string]interface{}
	_ = json.Unmarshal(b, &shape)
	links, ok := shape["links"].([]interface{})
	if !ok || len(links) != 3 {
		t.Fatalf("links JSON must be an array of 3 label/href objects: %s", b)
	}
	first := links[0].(map[string]interface{})
	if first["label"] == "" || first["href"] == "" {
		t.Fatalf("invalid link shape: %#v", first)
	}
	if !strings.Contains(got.Answer, "模拟") {
		t.Fatalf("mock answer must be explicit, got %q", got.Answer)
	}
	trace, err := svc.GetTrace(context.Background(), 42, got.TraceID)
	if err != nil || trace.Trace.Status != "completed" || trace.Trace.UserID == nil || *trace.Trace.UserID != 42 {
		t.Fatalf("trace = %#v, err=%v", trace, err)
	}
}

func TestSendMessageRealProviderReturnsGroundedAnswer(t *testing.T) {
	svc := newTestService(t, &fakeProvider{name: "openai", resp: &ai.LLMResponse{Answer: "当前证据不足，应继续补齐竞争与独立反证。", Model: "test-model", TokensIn: 80, TokensOut: 20, LatencyMs: 15}})

	got, err := svc.SendMessage(context.Background(), 42, MessageInput{Message: "现在可以实验吗？", DemandCaseID: 7})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if got.TruthStatus != TruthInferred || got.Trusted {
		t.Fatalf("truth/trusted = %q/%v", got.TruthStatus, got.Trusted)
	}
	if got.DemandCaseID != 7 || got.Provider != "openai" || got.Model != "test-model" {
		t.Fatalf("unexpected provenance: %#v", got)
	}
	if got.Provenance.TokensIn != 80 || got.Provenance.TokensOut != 20 || got.Evidence[0].SourceURL == "" {
		t.Fatalf("incomplete structured response: %#v", got)
	}
	if got.Evidence[0].RunID != "run-1" || got.Evidence[0].SnapshotID != 3 || got.Evidence[0].SnapshotSHA256 != strings.Repeat("a", 64) {
		t.Fatalf("missing evidence provenance: %#v", got.Evidence[0])
	}
	trace, err := svc.GetTrace(context.Background(), 42, got.TraceID)
	if err != nil || len(trace.Evidence) == 0 {
		t.Fatalf("missing trace evidence: %#v err=%v", trace, err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(trace.Evidence[0].Payload, &payload); err != nil || payload["run_id"] != "run-1" || payload["snapshot_sha256"] != strings.Repeat("a", 64) || payload["observed_at"] == "" {
		t.Fatalf("incomplete trace evidence payload: %s err=%v", trace.Evidence[0].Payload, err)
	}
}

func TestSendMessageProviderFailureMarksTraceFailedWithoutFallback(t *testing.T) {
	svc := newTestService(t, &fakeProvider{name: "openai", err: errors.New("provider unavailable")})

	_, err := svc.SendMessage(context.Background(), 42, MessageInput{Message: "现在可以实验吗？", DemandCaseID: 7})
	if err == nil {
		t.Fatal("expected provider failure")
	}
	var runErr *RunError
	if !errors.As(err, &runErr) || runErr.TraceID == "" {
		t.Fatalf("expected RunError with trace id, got %T %v", err, err)
	}
	trace, getErr := svc.GetTrace(context.Background(), 42, runErr.TraceID)
	if getErr != nil || trace.Trace.Status != "failed" {
		t.Fatalf("failed trace not persisted: %#v, err=%v", trace, getErr)
	}
}

func TestGetTraceIsOwnerScoped(t *testing.T) {
	svc := newTestService(t, &fakeProvider{name: "stub", resp: &ai.LLMResponse{Answer: "demo", Model: "stub-v1"}})
	got, err := svc.SendMessage(context.Background(), 42, MessageInput{Message: "查看案件", DemandCaseID: 7})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetTrace(context.Background(), 99, got.TraceID); !errors.Is(err, ErrTraceNotFound) {
		t.Fatalf("cross-owner GetTrace error = %v", err)
	}
}

func TestSendMessageRejectsInvalidInput(t *testing.T) {
	svc := newTestService(t, &fakeProvider{name: "stub", resp: &ai.LLMResponse{}})
	for _, in := range []MessageInput{
		{Message: "", DemandCaseID: 7},
		{Message: "问题", DemandCaseID: 0},
		{Message: strings.Repeat("字", MaxMessageRunes+1), DemandCaseID: 7},
	} {
		if _, err := svc.SendMessage(context.Background(), 42, in); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("input %#v error = %v", in, err)
		}
	}
}

func TestCapabilitiesAreCompleteActiveLongTermContracts(t *testing.T) {
	for _, c := range Capabilities() {
		if c.Status != "active" || c.Version == "" || c.Domain == "" || c.InputSchema == nil || c.OutputSchema == nil || c.RequiredPermission != "agent.write" || c.ApprovalRequired || c.ExternalSideEffects || c.IdempotencyRequired || len(c.ExecutionModes) != 1 || c.ExecutionModes[0] != "read_only" || c.TimeoutSeconds <= 0 || c.RetryLimit < 0 || c.AuditActionType == "" || c.OwnerExplanation == "" || c.EvidencePolicy == "" {
			t.Fatalf("incomplete capability contract: %#v", c)
		}
	}
}

func TestTraceWriteFailuresCannotReturnSuccess(t *testing.T) {
	for _, failAt := range []string{"start", "append", "evidence", "complete"} {
		t.Run(failAt, func(t *testing.T) {
			detail, card := testDemandData()
			db := dbtest.NewDB(t, &ai.AITrace{}, &ai.AITraceEvent{}, &ai.AIEvidenceRef{}, &ai.UnifiedAction{})
			writer := ai.NewTraceWriter(db, dbtest.NewLogger(t))
			recorder := &failingTraceRecorder{failAt: failAt, trace: writer}
			svc := NewService(db, dbtest.NewLogger(t), &fakeDemandReader{detail: detail, card: card}, nil, &fakeProvider{name: "stub"}, recorder)
			if _, err := svc.SendMessage(context.Background(), 42, MessageInput{Message: "查看", DemandCaseID: 7}); err == nil {
				t.Fatalf("%s failure returned success", failAt)
			}
		})
	}
}
