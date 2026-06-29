package producthub

import (
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func newSampleService(t *testing.T) *SampleService {
	t.Helper()
	db := dbtest.NewDB(t, &ProductMaster{}, &SampleRequest{})
	return NewSampleService(db, zap.NewNop())
}

func TestSampleCreateAndEvaluate(t *testing.T) {
	db := dbtest.NewDB(t, &ProductMaster{}, &SampleRequest{})
	ms := NewMasterService(db, zap.NewNop())
	svc := NewSampleService(db, zap.NewNop())

	ctx := t.Context()
	master := &ProductMaster{Name: "Sample Test", OwnerID: 1}
	ms.Create(ctx, master)

	sr := &SampleRequest{ProductMasterID: master.ID, SupplierID: 100, Quantity: 5}
	if err := svc.Create(ctx, sr); err != nil {
		t.Fatal(err)
	}

	if err := svc.RecordEvaluation(ctx, sr.ID, "Looks good", 8.5, "pass", nil); err != nil {
		t.Fatal(err)
	}

	latest, _ := svc.GetLatestByMaster(ctx, master.ID)
	if latest.Decision != "pass" || latest.QualityScore != 8.5 {
		t.Fatalf("expected pass/8.5, got %s/%.1f", latest.Decision, latest.QualityScore)
	}
}
