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

func TestLegacyExperimentBusinessClosureIsUnavailable(t *testing.T) {
	db := dbtest.NewDB(t, &ai.AITrace{}, &ai.AITraceEvent{}, &ai.AIEvidenceRef{}, &ai.UnifiedAction{})
	reader := &fakeBusinessClosureReader{view: &experiment.OwnerBusinessClosureView{
		ExperimentID: "exp-1",
		Order:        experiment.OwnerOrderClosure{ID: 10, OrderNoMasked: "****1234", TruthStatus: experiment.TruthUnknown, SourceStatus: "internal_record"},
		Unknowns:     []string{"售后观察期未知", "现金一致性未知"},
		EvidenceRefs: []experiment.OwnerClosureEvidenceRef{{SourceType: "order", SourceID: "10", TruthStatus: "unknown", Summary: "internal_record"}},
	}}
	svc := NewService(db, dbtest.NewLogger(t), nil, nil, &fakeProvider{name: "stub"}, ai.NewTraceWriter(db, dbtest.NewLogger(t))).WithBusinessClosureReader(reader)
	response, err := svc.SendMessage(context.Background(), 42, MessageInput{Message: "经营闭环怎么样", TargetType: TargetBusinessClosure, ExperimentID: "exp-1"})
	if err != ErrCapabilityUnavailable || response != nil {
		t.Fatalf("got response=%#v err=%v", response, err)
	}
	if reader.ownerID != 0 || reader.deadlineSeen {
		t.Fatal("deprecated reader must not be invoked")
	}
}

func TestLegacyExperimentBusinessClosureCapabilityIsNotActive(t *testing.T) {
	if _, ok := activeCapability(CapabilityOrderFulfillmentRead); ok {
		t.Fatal("deprecated experiment-scoped capability must not be active")
	}
}
