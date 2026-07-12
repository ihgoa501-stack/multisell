package toolbridge

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"go.uber.org/zap"
)

type countingIdempotentDriver struct {
	calls atomic.Int32
	fail  atomic.Bool
}

func TestExecuteCallConsumesRealApprovalOnceAndReplaysSameKey(t *testing.T) {
	db := dbtest.NewDB(t, &ToolExecution{}, &approval.ApprovalRequest{}, &approval.ApprovalExecution{})
	approvalSvc := approval.NewService(db, dbtest.NewLogger(t), nil)
	future := time.Now().Add(time.Hour)
	req, err := approvalSvc.Create(&approval.CreateApprovalInput{ProductID: 42, RequestType: "publish", Requester: "owner", TargetType: "listing", TargetID: 42, ExpiresAt: &future})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := approvalSvc.Review(req.ID, &approval.ReviewApprovalInput{Action: "approve", Reviewer: "owner"}); err != nil {
		t.Fatal(err)
	}
	driver := &countingIdempotentDriver{}
	b := NewToolBridge(nil, time.Second, zap.NewNop(), WithApprovalVerifier(approval.NewApprovalPolicyChecker(approvalSvc)), WithIdempotencyStore(NewGormToolIdempotencyStore(db, time.Minute)))
	b.RegisterTool("auto_publish", driver)
	call := ToolCall{ToolName: "auto_publish", Category: ToolCategoryMutation, Mode: ModeProduction, ApprovalID: &req.ID, IdempotencyKey: "tool-publish-42", TargetType: "listing", TargetID: "42", Input: map[string]interface{}{"value": "publish"}}
	first, err := b.ExecuteCall(context.Background(), call)
	if err != nil {
		t.Fatal(err)
	}
	second, err := b.ExecuteCall(context.Background(), call)
	if err != nil || first.Data["key"] != second.Data["key"] || driver.calls.Load() != 1 {
		t.Fatalf("first=%+v second=%+v calls=%d err=%v", first, second, driver.calls.Load(), err)
	}
	call.IdempotencyKey = "tool-publish-42-other"
	if _, err := b.ExecuteCall(context.Background(), call); !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("different-key approval reuse error=%v", err)
	}
}

func (d *countingIdempotentDriver) FetchPage(context.Context, string) (*PageData, error) {
	return nil, errors.New("unsupported")
}
func (d *countingIdempotentDriver) Health() (bool, time.Duration, error) { return true, 0, nil }
func (d *countingIdempotentDriver) Category() ToolCategory               { return ToolCategoryMutation }
func (d *countingIdempotentDriver) Execute(context.Context, map[string]interface{}) (*ToolResult, error) {
	return nil, errors.New("non-idempotent path used")
}
func (d *countingIdempotentDriver) ExecuteIdempotent(_ context.Context, input map[string]interface{}, key string) (*ToolResult, error) {
	d.calls.Add(1)
	if d.fail.Load() {
		return nil, errors.New("temporary provider failure")
	}
	return &ToolResult{Success: true, Data: map[string]interface{}{"key": key, "value": input["value"]}}, nil
}

func productionToolCall(key string) ToolCall {
	approvalID := int64(7)
	return ToolCall{ToolName: "publish_listing", Category: ToolCategoryMutation, Mode: ModeProduction, ApprovalID: &approvalID, IdempotencyKey: key, TargetType: "listing", TargetID: "42", Input: map[string]interface{}{"value": "one"}}
}

func newIdempotentBridge(t *testing.T, driver ToolDriver) *ToolBridge {
	t.Helper()
	db := dbtest.NewDB(t, &ToolExecution{})
	b := NewToolBridge(nil, time.Second, zap.NewNop(), WithApprovalVerifier(toolApprovalVerifier{approved: true}), WithIdempotencyStore(NewGormToolIdempotencyStore(db, time.Minute)))
	b.RegisterTool("publish_listing", driver)
	return b
}

func TestExecuteCall_ReplaysDurableToolResult(t *testing.T) {
	driver := &countingIdempotentDriver{}
	b := newIdempotentBridge(t, driver)
	call := productionToolCall("publish:durable:1")
	first, err := b.ExecuteCall(context.Background(), call)
	if err != nil {
		t.Fatal(err)
	}
	second, err := b.ExecuteCall(context.Background(), call)
	if err != nil {
		t.Fatal(err)
	}
	if driver.calls.Load() != 1 || first.Data["key"] != call.IdempotencyKey || second.Data["value"] != "one" {
		t.Fatalf("calls=%d first=%+v second=%+v", driver.calls.Load(), first, second)
	}
}

func TestExecuteCall_RejectsKeyReuseForDifferentInput(t *testing.T) {
	b := newIdempotentBridge(t, &countingIdempotentDriver{})
	call := productionToolCall("publish:durable:2")
	if _, err := b.ExecuteCall(context.Background(), call); err != nil {
		t.Fatal(err)
	}
	call.Input["value"] = "different"
	if _, err := b.ExecuteCall(context.Background(), call); err == nil {
		t.Fatal("idempotency key reuse for a different call was accepted")
	}
}

func TestExecuteCall_FailedExecutionCanRetry(t *testing.T) {
	driver := &countingIdempotentDriver{}
	driver.fail.Store(true)
	b := newIdempotentBridge(t, driver)
	call := productionToolCall("publish:durable:3")
	if _, err := b.ExecuteCall(context.Background(), call); err == nil {
		t.Fatal("provider failure was hidden")
	}
	driver.fail.Store(false)
	if _, err := b.ExecuteCall(context.Background(), call); err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if driver.calls.Load() != 3 {
		t.Fatalf("driver calls=%d, want 3 (two bounded attempts then one successful logical retry)", driver.calls.Load())
	}
}

func TestExecuteCall_ProductionMutationFailsClosedWithoutDurability(t *testing.T) {
	call := productionToolCall("publish:durable:4")
	withoutStore := NewToolBridge(nil, time.Second, zap.NewNop(), WithApprovalVerifier(toolApprovalVerifier{approved: true}))
	withoutStore.RegisterTool(call.ToolName, &countingIdempotentDriver{})
	if _, err := withoutStore.ExecuteCall(context.Background(), call); !errors.Is(err, ErrIdempotencyUnavailable) {
		t.Fatalf("without store error=%v", err)
	}

	withStore := newIdempotentBridge(t, &nonIdempotentMutationDriver{})
	if _, err := withStore.ExecuteCall(context.Background(), call); !errors.Is(err, ErrIdempotencyUnavailable) {
		t.Fatalf("without provider idempotency error=%v", err)
	}
}

type nonIdempotentMutationDriver struct{}

func (*nonIdempotentMutationDriver) FetchPage(context.Context, string) (*PageData, error) {
	return nil, nil
}
func (*nonIdempotentMutationDriver) Health() (bool, time.Duration, error) { return true, 0, nil }
func (*nonIdempotentMutationDriver) Category() ToolCategory               { return ToolCategoryMutation }
func (*nonIdempotentMutationDriver) Execute(context.Context, map[string]interface{}) (*ToolResult, error) {
	return &ToolResult{Success: true}, nil
}
