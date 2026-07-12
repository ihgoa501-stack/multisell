package xiaoq

import (
	"context"
	"testing"

	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/experiment"
)

type fakeBusinessClosureReader struct {
	ownerID      int64
	view         *experiment.OwnerBusinessClosureView
	deadlineSeen bool
}

func (f *fakeBusinessClosureReader) ReadOwnerBusinessClosure(ctx context.Context, ownerID int64, _ string) (*experiment.OwnerBusinessClosureView, error) {
	_, f.deadlineSeen = ctx.Deadline()
	f.ownerID = ownerID
	return f.view, nil
}

func TestBusinessClosureMessageUsesThreeReadCapabilitiesAndGrounding(t *testing.T) {
	db := dbtest.NewDB(t, &ai.AITrace{}, &ai.AITraceEvent{}, &ai.AIEvidenceRef{}, &ai.UnifiedAction{})
	reader := &fakeBusinessClosureReader{view: &experiment.OwnerBusinessClosureView{
		ExperimentID: "exp-1",
		Order:        experiment.OwnerOrderClosure{ID: 10, OrderNoMasked: "****1234", TruthStatus: experiment.TruthUnknown, SourceStatus: "internal_record"},
		Unknowns:     []string{"售后观察期未知", "现金一致性未知"},
		EvidenceRefs: []experiment.OwnerClosureEvidenceRef{{SourceType: "order", SourceID: "10", TruthStatus: "unknown", Summary: "internal_record"}},
	}}
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, &fakeProvider{name: "stub"}, ai.NewTraceWriter(db, dbtest.NewLogger(t))).WithBusinessClosureReader(reader)
	response, err := svc.SendMessage(context.Background(), 42, MessageInput{Message: "经营闭环怎么样", TargetType: TargetBusinessClosure, ExperimentID: "exp-1"})
	if err != nil {
		t.Fatal(err)
	}
	if reader.ownerID != 42 || !reader.deadlineSeen || response.TargetType != TargetBusinessClosure || response.Trusted || len(response.Evidence) != 1 || len(response.Unknowns) != 2 {
		t.Fatalf("unexpected response: %#v", response)
	}
	var calls int64
	if err := db.Model(&ai.AITraceEvent{}).Where("trace_id = ? AND event_type = ?", response.TraceID, "capability_call").Count(&calls).Error; err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("capability calls=%d want=3", calls)
	}
	detail, err := ai.NewTraceWriter(db, dbtest.NewLogger(t)).GetDetail(response.TraceID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Evidence) != 1 || len(response.Links) != 2 {
		t.Fatalf("grounding missing: %#v", response)
	}
}

func TestBusinessClosureCapabilitiesAreActive(t *testing.T) {
	for _, id := range []string{CapabilityOrderFulfillmentRead, CapabilitySettlementRead, CapabilityProfitFinalRead} {
		if capability, ok := activeCapability(id); !ok || capability.Status != "active" {
			t.Fatalf("%s not active", id)
		}
	}
}
