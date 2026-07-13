package businessfeedback

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"github.com/lingmirror/backend-go/internal/platform/command"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type decisionCase struct {
	ID, OwnerID, ObjectID int64
	ObjectType            string
}

func (decisionCase) TableName() string { return "business_decision_case" }

type ownerDecision struct {
	ID, OwnerID, DecisionCaseID                                            int64
	Decision, CapabilityID, CommandType, TargetType, TargetID, InputSHA256 string
	CreatedAt                                                              time.Time
}

func (ownerDecision) TableName() string { return "business_owner_decision" }

type profitFact struct {
	ID, OwnerID, OrderID int64
	SourceManifestSHA256 string
	FinalizedAt          time.Time
}

func (profitFact) TableName() string { return "order_final_profit_version" }

type orderIngestFact struct {
	ID, OwnerID                                  int64
	NormalizedOrderID                            *int64
	TruthStatus, ProcessingStatus, PayloadSHA256 string
	ObservedAt                                   time.Time
}

func (orderIngestFact) TableName() string { return "platform_order_ingest" }

type allowPolicy struct{}

func (allowPolicy) AuthorizeFor(context.Context, int64, string, string, string, string) error {
	return nil
}
func (allowPolicy) ConsumeFor(context.Context, int64, string, string, string, string) error {
	return nil
}
func (allowPolicy) CompleteFor(context.Context, int64, string) error    { return nil }
func (allowPolicy) FailFor(context.Context, int64, string, error) error { return nil }

type denyPolicy struct{ allowPolicy }

func (denyPolicy) AuthorizeFor(context.Context, int64, string, string, string, string) error {
	return errors.New("approval target mismatch")
}

func setup(t *testing.T) (*Service, int64) {
	db := dbtest.NewDB(t, &decisionCase{}, &ownerDecision{}, &ControlledAction{}, &ActionObservation{}, &NextActionRecommendation{}, &profitFact{}, &orderIngestFact{}, &command.ActionExecution{}, &operationlog.OperationLog{})
	d := command.NewDispatcher(zap.NewNop(), command.WithIdempotencyStore(command.NewGormIdempotencyStore(db, time.Minute)))
	d.Register("price_update", func(context.Context, map[string]interface{}) (*command.Result, error) {
		return &command.Result{Success: true, BusinessID: "price-1"}, nil
	})
	orderID := int64(101)
	ingest := orderIngestFact{OwnerID: 7, NormalizedOrderID: &orderID, TruthStatus: "external_observed", ProcessingStatus: "applied", PayloadSHA256: strings.Repeat("b", 64), ObservedAt: time.Now()}
	if err := db.Create(&ingest).Error; err != nil {
		t.Fatal(err)
	}
	c := decisionCase{OwnerID: 7, ObjectType: "platform_order_ingest", ObjectID: ingest.ID}
	if err := db.Create(&c).Error; err != nil {
		t.Fatal(err)
	}
	payloadHash := sha256.Sum256([]byte(`{}`))
	o := ownerDecision{OwnerID: 7, DecisionCaseID: c.ID, Decision: "selected", CapabilityID: "command.price_update.v1", CommandType: "price_update", TargetType: "sku", TargetID: "1", InputSHA256: hex.EncodeToString(payloadHash[:]), CreatedAt: time.Now()}
	if err := db.Create(&o).Error; err != nil {
		t.Fatal(err)
	}
	return NewService(db, d, allowPolicy{}, []string{"command.price_update.v1"}), o.ID
}

func TestControlledActionRequiresLatestSelectedDecisionAndRegisteredCapabilityCommand(t *testing.T) {
	s, decisionID := setup(t)
	in := CreateActionInput{OwnerDecisionID: decisionID, CapabilityID: "command.price_update.v1", CommandType: "price_update", TargetType: "sku", TargetID: "1", ApprovalID: 9, IdempotencyKey: "decision-1-price", InputPayload: json.RawMessage(`{}`)}
	a, err := s.CreateAction(context.Background(), 7, in)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CreateAction(context.Background(), 8, in); err == nil {
		t.Fatal("cross-owner decision must fail")
	}
	bad := in
	bad.IdempotencyKey = "bad-cap"
	bad.CapabilityID = "unknown"
	if _, err = s.CreateAction(context.Background(), 7, bad); err == nil {
		t.Fatal("unknown capability must fail")
	}
	mismatch := in
	mismatch.IdempotencyKey = "target-mismatch"
	mismatch.TargetID = "2"
	if _, err = s.CreateAction(context.Background(), 7, mismatch); !errors.Is(err, ErrNotAuthorized) {
		t.Fatalf("action outside frozen target must fail: %v", err)
	}
	mismatch = in
	mismatch.IdempotencyKey = "input-mismatch"
	mismatch.InputPayload = json.RawMessage(`{"price":101}`)
	if _, err = s.CreateAction(context.Background(), 7, mismatch); !errors.Is(err, ErrNotAuthorized) {
		t.Fatalf("action outside frozen input must fail: %v", err)
	}
	if a, err = s.Execute(context.Background(), 7, a.ID); err != nil || a.Status != "succeeded" || a.CommandBusinessID != "price-1" {
		t.Fatalf("execute=%+v err=%v", a, err)
	}
}

func TestNewOwnerDecisionRevokesUnexecutedAction(t *testing.T) {
	s, decisionID := setup(t)
	in := CreateActionInput{OwnerDecisionID: decisionID, CapabilityID: "command.price_update.v1", CommandType: "price_update", TargetType: "sku", TargetID: "1", ApprovalID: 9, IdempotencyKey: "revoked", InputPayload: json.RawMessage(`{}`)}
	a, err := s.CreateAction(context.Background(), 7, in)
	if err != nil {
		t.Fatal(err)
	}
	var old ownerDecision
	if err := s.db.First(&old, decisionID).Error; err != nil {
		t.Fatal(err)
	}
	newer := ownerDecision{OwnerID: 7, DecisionCaseID: old.DecisionCaseID, Decision: "paused", CreatedAt: time.Now().Add(time.Second)}
	if err := s.db.Create(&newer).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.Execute(context.Background(), 7, a.ID); err == nil {
		t.Fatal("latest paused decision must revoke execution")
	}
}

func TestDispatchSafeRejectsApprovalNotBoundToExactCommandTargetAndKey(t *testing.T) {
	s, decisionID := setup(t)
	s.policy = denyPolicy{}
	in := CreateActionInput{OwnerDecisionID: decisionID, CapabilityID: "command.price_update.v1", CommandType: "price_update", TargetType: "sku", TargetID: "1", ApprovalID: 9, IdempotencyKey: "approval-mismatch", InputPayload: json.RawMessage(`{}`)}
	a, err := s.CreateAction(context.Background(), 7, in)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Execute(context.Background(), 7, a.ID); !errors.Is(err, command.ErrApprovalRequired) {
		t.Fatalf("expected exact approval rejection, got %v", err)
	}
	if err = s.db.First(&a, a.ID).Error; err != nil || a.Status != "failed" {
		t.Fatalf("pre-dispatch rejection must be a deterministic failure: action=%+v err=%v", a, err)
	}
}

func TestExecuteRequiresReconciliationWhenCommandOutcomeIsUnknown(t *testing.T) {
	s, decisionID := setup(t)
	s.dispatcher.Register("price_update", func(context.Context, map[string]interface{}) (*command.Result, error) {
		return nil, context.DeadlineExceeded
	})
	in := CreateActionInput{OwnerDecisionID: decisionID, CapabilityID: "command.price_update.v1", CommandType: "price_update", TargetType: "sku", TargetID: "1", ApprovalID: 9, IdempotencyKey: "unknown-outcome", InputPayload: json.RawMessage(`{}`)}
	a, err := s.CreateAction(context.Background(), 7, in)
	if err != nil {
		t.Fatal(err)
	}
	a, err = s.Execute(context.Background(), 7, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != "reconcile_required" {
		t.Fatalf("external command without a receipt must require reconciliation, got %+v", a)
	}
	if _, err = s.Execute(context.Background(), 7, a.ID); err != nil {
		t.Fatalf("reconciliation state must block blind retry: %v", err)
	}
}

func TestExecuteFailsClosedWhenPendingAuditCannotBeWritten(t *testing.T) {
	s, decisionID := setup(t)
	called := false
	s.dispatcher.Register("price_update", func(context.Context, map[string]interface{}) (*command.Result, error) {
		called = true
		return &command.Result{Success: true}, nil
	})
	if err := s.db.Callback().Create().Before("gorm:create").Register("test:reject_operation_log", func(tx *gorm.DB) {
		if tx.Statement.Table == "operation_log" {
			tx.AddError(errors.New("audit unavailable"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	in := CreateActionInput{OwnerDecisionID: decisionID, CapabilityID: "command.price_update.v1", CommandType: "price_update", TargetType: "sku", TargetID: "1", ApprovalID: 9, IdempotencyKey: "audit-fail-closed", InputPayload: json.RawMessage(`{}`)}
	a, err := s.CreateAction(context.Background(), 7, in)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Execute(context.Background(), 7, a.ID); err == nil {
		t.Fatal("execution must stop when its pending audit cannot be written")
	}
	if called {
		t.Fatal("external command ran without a durable pending audit")
	}
}

func TestExecuteRecoversInterruptedFinalAuditAsReconcileRequired(t *testing.T) {
	s, decisionID := setup(t)
	called := false
	s.dispatcher.Register("price_update", func(context.Context, map[string]interface{}) (*command.Result, error) {
		called = true
		return &command.Result{Success: true, BusinessID: "remote-1"}, nil
	})
	if err := s.db.Callback().Create().Before("gorm:create").Register("test:reject_final_operation_log", func(tx *gorm.DB) {
		if called && tx.Statement.Table == "operation_log" {
			tx.AddError(errors.New("final audit unavailable"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	in := CreateActionInput{OwnerDecisionID: decisionID, CapabilityID: "command.price_update.v1", CommandType: "price_update", TargetType: "sku", TargetID: "1", ApprovalID: 9, IdempotencyKey: "final-audit-recovery", InputPayload: json.RawMessage(`{}`)}
	a, err := s.CreateAction(context.Background(), 7, in)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Execute(context.Background(), 7, a.ID); err == nil {
		t.Fatal("missing final audit must surface an error")
	}
	if err = s.db.Callback().Create().Remove("test:reject_final_operation_log"); err != nil {
		t.Fatal(err)
	}
	a, err = s.Execute(context.Background(), 7, a.ID)
	if err != nil || a.Status != "reconcile_required" {
		t.Fatalf("interrupted execution must recover without redispatch: action=%+v err=%v", a, err)
	}
}

func TestObservationUsesAuthoritativeFactAndRecommendationRemainsInferred(t *testing.T) {
	s, decisionID := setup(t)
	in := CreateActionInput{OwnerDecisionID: decisionID, CapabilityID: "command.price_update.v1", CommandType: "price_update", TargetType: "sku", TargetID: "1", ApprovalID: 9, IdempotencyKey: "observe", InputPayload: json.RawMessage(`{}`)}
	a, _ := s.CreateAction(context.Background(), 7, in)
	a, _ = s.Execute(context.Background(), 7, a.ID)
	f := profitFact{OwnerID: 7, OrderID: 101, SourceManifestSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", FinalizedAt: time.Now()}
	if err := s.db.Create(&f).Error; err != nil {
		t.Fatal(err)
	}
	o, err := s.CreateObservation(context.Background(), 7, a.ID, CreateObservationInput{EvidenceKind: "counter", SourceObjectType: "order_final_profit_version", SourceObjectID: f.ID, TargetMetric: "profit_minor", TargetValue: "1000", ActualValue: "-200"})
	if err != nil {
		t.Fatal(err)
	}
	if o.TruthStatus != "actual" || o.EvidenceKind != "counter" {
		t.Fatalf("unexpected observation %+v", o)
	}
	other := profitFact{OwnerID: 7, OrderID: 202, SourceManifestSHA256: strings.Repeat("c", 64), FinalizedAt: time.Now()}
	if err := s.db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateObservation(context.Background(), 7, a.ID, CreateObservationInput{EvidenceKind: "support", SourceObjectType: "order_final_profit_version", SourceObjectID: other.ID, TargetMetric: "profit_minor", TargetValue: "1000", ActualValue: "1000"}); err == nil {
		t.Fatal("same-Owner fact from another order must not enter this decision feedback")
	}
	r, err := s.CreateRecommendation(context.Background(), 7, a.ID, CreateRecommendationInput{RecommendationText: "降低价格变量后再由Owner决定", Rationale: "实际利润低于目标"})
	if err != nil {
		t.Fatal(err)
	}
	if r.TruthStatus != "inferred" || r.Status != "proposed" {
		t.Fatalf("recommendation improperly upgraded: %+v", r)
	}
	list, err := s.List(context.Background(), 7, decisionID)
	if err != nil || len(list) != 1 || list[0].ID != a.ID {
		t.Fatalf("Owner action list=%+v err=%v", list, err)
	}
	detail, err := s.Get(context.Background(), 7, a.ID)
	if err != nil || len(detail.Observations) != 1 || len(detail.NextRecommendations) != 1 {
		t.Fatalf("restored detail=%+v err=%v", detail, err)
	}
	if _, err := s.Get(context.Background(), 8, a.ID); err == nil {
		t.Fatal("cross-Owner action detail leaked")
	}
}
