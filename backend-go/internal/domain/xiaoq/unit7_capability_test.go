package xiaoq

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/businessdecision"
	"github.com/lingmirror/backend-go/internal/domain/integrations"
)

type fakeOperatingReader struct{ owner, order int64 }

func (f *fakeOperatingReader) ReadOwnerOperatingView(_ context.Context, owner, order int64) (*integrations.OwnerOperatingView, error) {
	f.owner, f.order = owner, order
	return &integrations.OwnerOperatingView{Order: integrations.OwnerOrderFact{OrderID: order, IngestID: 1, TruthStatus: "external_observed", PayloadSHA256: strings.Repeat("a", 64), ObservedAt: time.Now()}, Evidence: []integrations.OwnerFactEvidence{{SourceType: "platform_order_ingest", SourceID: 1, TruthStatus: "external_observed", SHA256: strings.Repeat("a", 64), ObservedAt: time.Now()}}, Unknowns: []string{"尚无利润"}, Blockers: []string{"售后未终结"}}, nil
}

func TestOperatingFactsUsesSevenOwnerScopedCapabilitiesAndSeparatesBlockers(t *testing.T) {
	db := dbtest.NewDB(t, &ai.AITrace{}, &ai.AITraceEvent{}, &ai.AIEvidenceRef{}, &ai.UnifiedAction{})
	r := &fakeOperatingReader{}
	s := NewService(db, dbtest.NewLogger(t), nil, nil, &fakeProvider{name: "stub"}, ai.NewTraceWriter(db, dbtest.NewLogger(t))).WithOwnerOperatingReader(r)
	got, err := s.SendMessage(context.Background(), 42, MessageInput{Message: "订单怎么样", TargetType: TargetOperatingFacts, OrderID: 9})
	if err != nil {
		t.Fatal(err)
	}
	if r.owner != 42 || r.order != 9 || got.OrderID != 9 || len(got.Unknowns) != 1 || len(got.Blockers) != 1 || got.Trusted {
		t.Fatalf("bad response %#v", got)
	}
	var n int64
	if err = db.Model(&ai.AITraceEvent{}).Where("trace_id=? AND event_type='capability_call'", got.TraceID).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 7 {
		t.Fatalf("calls=%d", n)
	}
}

type fakeDecisionReader struct {
	detail          *businessdecision.Detail
	recommendations int
}

func (f *fakeDecisionReader) Get(_ context.Context, owner, id int64) (*businessdecision.Detail, error) {
	return f.detail, nil
}
func (f *fakeDecisionReader) Recommend(_ context.Context, owner, id int64, in businessdecision.RecommendInput) (*businessdecision.AIRecommendation, error) {
	f.recommendations++
	return &businessdecision.AIRecommendation{ID: 77, DecisionCaseID: id, OwnerID: owner, Recommendation: in.Recommendation, TruthStatus: in.TruthStatus, ManifestSHA256: f.detail.Case.ManifestSHA256}, nil
}

func TestBusinessDecisionStubCannotCreateRecommendationOrOwnerDecision(t *testing.T) {
	db := dbtest.NewDB(t, &ai.AITrace{}, &ai.AITraceEvent{}, &ai.AIEvidenceRef{}, &ai.UnifiedAction{})
	hash := strings.Repeat("b", 64)
	r := &fakeDecisionReader{detail: &businessdecision.Detail{Case: businessdecision.Case{ID: 2, OwnerID: 42, ManifestSHA256: hash}, Snapshot: businessdecision.FactSnapshot{ID: 3, TruthStatus: "external_observed", ObjectType: "platform_order_ingest", ObjectID: 4, PayloadSHA256: hash, SourceObservedAt: time.Now()}}}
	s := NewService(db, dbtest.NewLogger(t), nil, nil, &fakeProvider{name: "stub"}, ai.NewTraceWriter(db, dbtest.NewLogger(t))).WithBusinessDecisionReader(r)
	got, err := s.SendMessage(context.Background(), 42, MessageInput{Message: "建议", TargetType: TargetBusinessDecision, DecisionCaseID: 2, CreateRecommendation: true, IdempotencyKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	if r.recommendations != 0 || got.RecommendationID != 0 || got.TruthStatus != TruthMock {
		t.Fatalf("stub created trusted recommendation: %#v", got)
	}
}

func TestUnit7CapabilitiesActiveAndLegacyExperimentClosureAbsent(t *testing.T) {
	ids := []string{CapabilityOrderFactRead, CapabilityInventoryLedgerRead, CapabilityFulfillmentFactRead, CapabilityAftersalesFactRead, CapabilitySettlementRead, CapabilityProfitFinalRead, CapabilityCashReconciliationRead, CapabilityBusinessDecisionRead, CapabilityBusinessRecommend}
	for _, id := range ids {
		if c, ok := activeCapability(id); !ok || c.Status != "active" {
			t.Fatalf("%s inactive", id)
		}
	}
	if _, ok := activeCapability(CapabilityOrderFulfillmentRead); ok {
		t.Fatal("legacy experiment capability active")
	}
}
