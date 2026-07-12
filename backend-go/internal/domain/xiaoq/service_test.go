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

func (f *fakeDemandReader) Get(context.Context, int64, int64) (*demandcase.Detail, error) {
	return f.detail, f.err
}

func (f *fakeDemandReader) DecisionCard(context.Context, int64, int64) (*demandcase.OwnerDecisionCard, error) {
	return f.card, f.err
}

type fakeProvider struct {
	name string
	resp *ai.LLMResponse
	err  error
}

func (f *fakeProvider) Name() string { return f.name }
func (f *fakeProvider) Chat(context.Context, *ai.LLMRequest) (*ai.LLMResponse, error) {
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
	return NewService(db, dbtest.NewLogger(t), &fakeDemandReader{detail: detail, card: card}, provider, ai.NewTraceWriter(db, dbtest.NewLogger(t)))
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
			svc := NewService(db, dbtest.NewLogger(t), &fakeDemandReader{detail: detail, card: card}, &fakeProvider{name: "stub"}, recorder)
			if _, err := svc.SendMessage(context.Background(), 42, MessageInput{Message: "查看", DemandCaseID: 7}); err == nil {
				t.Fatalf("%s failure returned success", failAt)
			}
		})
	}
}
