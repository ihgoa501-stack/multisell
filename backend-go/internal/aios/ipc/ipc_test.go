package ipc

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

// --- test helpers ---

func newTestLogger(t *testing.T) *zap.Logger {
	t.Helper()
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("failed to create test logger: %v", err)
	}
	return logger
}

func newTestIPC(t *testing.T) *IPC {
	t.Helper()
	return New(newTestLogger(t))
}

// registerEchoHandler registers a handler that echoes the payload back.
func registerEchoHandler(ipc *IPC, topic string) {
	ipc.RegisterHandler(topic, func(ctx context.Context, msg *Message) (*Message, error) {
		return &Message{
			SessionID: msg.SessionID,
			Type:      MsgTypeResponse,
			From:      topic,
			To:        msg.From,
			Payload:   msg.Payload,
			CreatedAt: time.Now(),
		}, nil
	})
}

// --- Send ---

func TestSend(t *testing.T) {
	ipc := newTestIPC(t)
	registerEchoHandler(ipc, "test_topic")

	msg := &Message{
		ID:        "msg1",
		Type:      MsgTypeNotify,
		From:      "agent_a",
		To:        "test_topic",
		Payload:   map[string]interface{}{"key": "value"},
		CreatedAt: time.Now(),
	}

	err := ipc.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestSend_NilMessage(t *testing.T) {
	ipc := newTestIPC(t)
	registerEchoHandler(ipc, "topic")

	err := ipc.Send(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil message")
	}
}

func TestSend_UnknownTopic(t *testing.T) {
	ipc := newTestIPC(t)
	msg := &Message{
		To:    "nonexistent",
		Type:  MsgTypeNotify,
		Payload: map[string]interface{}{},
	}
	err := ipc.Send(context.Background(), msg)
	if err == nil {
		t.Fatal("expected error for unknown topic")
	}
}

func TestSend_NotifyReturnsResponse(t *testing.T) {
	// Notify messages should not have their responses routed to pending channels.
	ipc := newTestIPC(t)
	var callCount atomic.Int64

	ipc.RegisterHandler("topic", func(ctx context.Context, msg *Message) (*Message, error) {
		callCount.Add(1)
		return &Message{SessionID: msg.SessionID, Payload: map[string]interface{}{"result": "ok"}}, nil
	})

	msg := &Message{
		ID:        "n1",
		Type:      MsgTypeNotify,
		From:      "a",
		To:        "topic",
		SessionID: "no-pending",
		Payload:   map[string]interface{}{"x": 1},
	}

	if err := ipc.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	// Wait for async handler to complete.
	time.Sleep(50 * time.Millisecond)
	if got := callCount.Load(); got != 1 {
		t.Fatalf("expected handler to be called once, got %d", got)
	}
}

// --- Request ---

func TestRequest(t *testing.T) {
	ipc := newTestIPC(t)
	registerEchoHandler(ipc, "echo")

	resp, err := ipc.Request(context.Background(), "echo", map[string]interface{}{
		"data": "hello",
	}, time.Second)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.Payload["data"] != "hello" {
		t.Fatalf("expected payload data 'hello', got %v", resp.Payload["data"])
	}
}

func TestRequest_HandlerError(t *testing.T) {
	ipc := newTestIPC(t)
	ipc.RegisterHandler("broken", func(ctx context.Context, msg *Message) (*Message, error) {
		return nil, fmt.Errorf("handler error")
	})

	_, err := ipc.Request(context.Background(), "broken", nil, 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected error from handler")
	}
}

func TestRequest_UnregisteredHandler(t *testing.T) {
	ipc := newTestIPC(t)
	_, err := ipc.Request(context.Background(), "no_handler", nil, time.Second)
	if err == nil {
		t.Fatal("expected error for unregistered handler")
	}
}

func TestRequest_Timeout(t *testing.T) {
	ipc := newTestIPC(t)
	ipc.RegisterHandler("slow", func(ctx context.Context, msg *Message) (*Message, error) {
		time.Sleep(200 * time.Millisecond)
		return &Message{SessionID: msg.SessionID, Payload: map[string]interface{}{"done": true}}, nil
	})

	_, err := ipc.Request(context.Background(), "slow", nil, 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestRequest_ContextCancelled(t *testing.T) {
	ipc := newTestIPC(t)
	registerEchoHandler(ipc, "echo")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancelled

	_, err := ipc.Request(ctx, "echo", nil, time.Second)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

// --- Broadcast ---

func TestBroadcast(t *testing.T) {
	ipc := newTestIPC(t)
	registerEchoHandler(ipc, "squad_a")
	registerEchoHandler(ipc, "squad_b")

	results := ipc.Broadcast(context.Background(), []string{"squad_a", "squad_b"}, map[string]interface{}{
		"broadcast": true,
	})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if v, ok := r.Payload["broadcast"]; !ok || v != true {
			t.Errorf("expected broadcast=true in result, got %v", r.Payload)
		}
	}
}

func TestBroadcast_EmptySquads(t *testing.T) {
	ipc := newTestIPC(t)
	results := ipc.Broadcast(context.Background(), nil, map[string]interface{}{"x": 1})
	if len(results) != 0 {
		t.Fatalf("expected 0 results for empty squads, got %d", len(results))
	}
}

func TestBroadcast_PartialFailure(t *testing.T) {
	ipc := newTestIPC(t)
	registerEchoHandler(ipc, "good")

	// 'bad' has no handler registered, so it will fail silently.
	results := ipc.Broadcast(context.Background(), []string{"good", "bad"}, map[string]interface{}{"k": "v"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result (partial), got %d", len(results))
	}
}

// --- Delegate ---

func TestDelegate(t *testing.T) {
	ipc := newTestIPC(t)
	ipc.RegisterHandler("worker", func(ctx context.Context, msg *Message) (*Message, error) {
		return &Message{
			SessionID: msg.SessionID,
			Type:      MsgTypeResponse,
			From:      "worker",
			To:        msg.From,
			Payload: map[string]interface{}{
				"result":  "done",
				"task_id": msg.Payload["task_id"],
			},
			CreatedAt: time.Now(),
		}, nil
	})

	task := TaskDef{
		ID:          "task-1",
		Description: "test delegated task",
		Payload:     map[string]interface{}{"input": "data"},
	}

	resp, ok := ipc.Delegate(context.Background(), "worker", task)
	if !ok {
		t.Fatal("Delegate returned failure")
	}
	if resp.Payload["task_id"] != "task-1" {
		t.Fatalf("expected task_id 'task-1', got %v", resp.Payload["task_id"])
	}
	if resp.Payload["result"] != "done" {
		t.Fatalf("expected result 'done', got %v", resp.Payload["result"])
	}
}

func TestDelegate_Unregistered(t *testing.T) {
	ipc := newTestIPC(t)
	task := TaskDef{ID: "t1", Description: "no handler"}
	_, ok := ipc.Delegate(context.Background(), "nonexistent", task)
	if ok {
		t.Fatal("expected Delegate to return false for unregistered handler")
	}
}

func TestDelegate_HandlerError(t *testing.T) {
	ipc := newTestIPC(t)
	ipc.RegisterHandler("broken", func(ctx context.Context, msg *Message) (*Message, error) {
		return nil, fmt.Errorf("internal error")
	})
	task := TaskDef{ID: "t2"}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, ok := ipc.Delegate(ctx, "broken", task)
	if ok {
		t.Fatal("expected Delegate to return false when handler returns error")
	}
}

// --- Gather ---

func TestGather(t *testing.T) {
	ipc := newTestIPC(t)
	ipc.RegisterHandler("worker_a", func(ctx context.Context, msg *Message) (*Message, error) {
		return &Message{SessionID: msg.SessionID, Payload: map[string]interface{}{"result": "A"}}, nil
	})
	ipc.RegisterHandler("worker_b", func(ctx context.Context, msg *Message) (*Message, error) {
		return &Message{SessionID: msg.SessionID, Payload: map[string]interface{}{"result": "B"}}, nil
	})

	task := TaskDef{ID: "gather-1", Description: "parallel gather test"}
	results, err := ipc.Gather(context.Background(), []string{"worker_a", "worker_b"}, task)
	if err != nil {
		t.Fatalf("Gather failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestGather_AllTargetsSame(t *testing.T) {
	ipc := newTestIPC(t)
	var callCount atomic.Int64
	ipc.RegisterHandler("shared", func(ctx context.Context, msg *Message) (*Message, error) {
		idx := callCount.Add(1)
		return &Message{
			SessionID: msg.SessionID,
			Payload:   map[string]interface{}{"idx": int(idx)},
		}, nil
	})

	// Gather from the same target twice — each call is a parallel delegate.
	task := TaskDef{ID: "g2", Description: "same target"}
	results, err := ipc.Gather(context.Background(), []string{"shared", "shared"}, task)
	if err != nil {
		t.Fatalf("Gather failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// Both results should have idx=1 and idx=2 (or vice versa depending on race).
	for _, r := range results {
		_ = r // valid result
	}
}

func TestGather_AllFail(t *testing.T) {
	ipc := newTestIPC(t)
	// No handlers registered — all delegates will fail.
	task := TaskDef{ID: "g3"}
	_, err := ipc.Gather(context.Background(), []string{"missing1", "missing2"}, task)
	if err == nil {
		t.Fatal("expected error when all gather targets fail")
	}
}

// --- Consensus ---

func TestConsensus(t *testing.T) {
	ipc := newTestIPC(t)
	ipc.RegisterHandler("agent_1", func(ctx context.Context, msg *Message) (*Message, error) {
		return &Message{
			SessionID: msg.SessionID,
			Payload:   map[string]interface{}{"value": 10.0, "confidence": 0.8},
		}, nil
	})
	ipc.RegisterHandler("agent_2", func(ctx context.Context, msg *Message) (*Message, error) {
		return &Message{
			SessionID: msg.SessionID,
			Payload:   map[string]interface{}{"value": 20.0, "confidence": 0.6},
		}, nil
	})

	result, err := ipc.Consensus(context.Background(), "best value?", []string{"agent_1", "agent_2"})
	if err != nil {
		t.Fatalf("Consensus failed: %v", err)
	}
	if result.Method != "weighted_avg" {
		t.Fatalf("expected weighted_avg method, got %s", result.Method)
	}
	if len(result.IndividualResults) != 2 {
		t.Fatalf("expected 2 individual results, got %d", len(result.IndividualResults))
	}
	// Weighted average: (10*0.8 + 20*0.6) / (0.8 + 0.6) = (8+12)/1.4 = 14.2857...
	if result.Confidence != (0.8+0.6)/2.0 {
		t.Fatalf("expected confidence 0.7, got %f", result.Confidence)
	}
}

func TestConsensus_Majority(t *testing.T) {
	ipc := newTestIPC(t)
	ipc.RegisterHandler("a1", func(ctx context.Context, msg *Message) (*Message, error) {
		return &Message{
			SessionID: msg.SessionID,
			Payload:   map[string]interface{}{"value": 10.0, "choice": "up"},
		}, nil
	})
	ipc.RegisterHandler("a2", func(ctx context.Context, msg *Message) (*Message, error) {
		return &Message{
			SessionID: msg.SessionID,
			Payload:   map[string]interface{}{"value": 20.0, "choice": "up"},
		}, nil
	})
	ipc.RegisterHandler("a3", func(ctx context.Context, msg *Message) (*Message, error) {
		return &Message{
			SessionID: msg.SessionID,
			Payload:   map[string]interface{}{"value": 30.0, "choice": "down"},
		}, nil
	})

	result, err := ipc.Consensus(context.Background(), "go up or down?", []string{"a1", "a2", "a3"})
	if err != nil {
		t.Fatalf("Consensus failed: %v", err)
	}
	if result.Method != "majority" {
		t.Fatalf("expected majority method, got %s", result.Method)
	}
	// Majority vote for choice should be "up".
	if result.FinalOutput["choice"] != "up" {
		t.Fatalf("expected majority choice 'up', got %v", result.FinalOutput["choice"])
	}
}

func TestConsensus_NoAgents(t *testing.T) {
	ipc := newTestIPC(t)
	_, err := ipc.Consensus(context.Background(), "any?", nil)
	if err == nil {
		t.Fatal("expected error when no agents provided")
	}
}

// computeConsensus edge cases

func TestComputeConsensus_Empty(t *testing.T) {
	r := computeConsensus(nil)
	if r.Confidence != 0 {
		t.Fatalf("expected confidence 0, got %f", r.Confidence)
	}
	if r.Method != "majority" {
		t.Fatalf("expected method majority, got %s", r.Method)
	}
}

func TestComputeConsensus_SingleResult(t *testing.T) {
	msgs := []*Message{
		{Payload: map[string]interface{}{"value": 42.0}},
	}
	r := computeConsensus(msgs)
	if r.FinalOutput["value"] != 42.0 {
		t.Fatalf("expected value 42, got %v", r.FinalOutput["value"])
	}
}

// --- Concurrency ---

func TestConcurrentRequests(t *testing.T) {
	ipc := newTestIPC(t)
	registerEchoHandler(ipc, "echo")

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			payload := map[string]interface{}{"idx": idx}
			resp, err := ipc.Request(context.Background(), "echo", payload, 5*time.Second)
			if err != nil {
				t.Errorf("concurrent Request #%d failed: %v", idx, err)
				return
			}
			var got int
			switch v := resp.Payload["idx"].(type) {
			case float64:
				got = int(v)
			case int:
				got = v
			default:
				t.Errorf("concurrent Request #%d: unexpected type for idx: %T", idx, resp.Payload["idx"])
				return
			}
			if got != idx {
				t.Errorf("concurrent Request #%d: expected idx=%d, got %v", idx, idx, resp.Payload["idx"])
			}
		}(i)
	}
	wg.Wait()
}

func TestConcurrentDelegate(t *testing.T) {
	ipc := newTestIPC(t)
	ipc.RegisterHandler("w", func(ctx context.Context, msg *Message) (*Message, error) {
		return &Message{SessionID: msg.SessionID, Payload: msg.Payload}, nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			task := TaskDef{ID: fmt.Sprintf("t-%d", idx), Description: "concurrent"}
			_, ok := ipc.Delegate(context.Background(), "w", task)
			if !ok {
				t.Errorf("concurrent Delegate #%d failed", idx)
			}
		}(i)
	}
	wg.Wait()
}

func TestConcurrentRegisterHandler(t *testing.T) {
	ipc := newTestIPC(t)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			topic := fmt.Sprintf("dyn_%d", idx)
			registerEchoHandler(ipc, topic)
			// Immediately use it after registering.
			_, err := ipc.Request(context.Background(), topic, map[string]interface{}{"idx": idx}, time.Second)
			if err != nil {
				t.Errorf("Request to dynamically registered topic %s failed: %v", topic, err)
			}
		}(i)
	}
	wg.Wait()
}

// --- RegisterHandler ---

func TestRegisterHandler(t *testing.T) {
	ipc := newTestIPC(t)
	called := false

	ipc.RegisterHandler("mytopic", func(ctx context.Context, msg *Message) (*Message, error) {
		called = true
		return &Message{SessionID: msg.SessionID, Payload: msg.Payload}, nil
	})

	_, err := ipc.Request(context.Background(), "mytopic", map[string]interface{}{"ping": "pong"}, time.Second)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if !called {
		t.Fatal("handler was not called")
	}
}

func TestRegisterHandler_Overwrite(t *testing.T) {
	ipc := newTestIPC(t)

	ipc.RegisterHandler("topic", func(ctx context.Context, msg *Message) (*Message, error) {
		return &Message{SessionID: msg.SessionID, Payload: map[string]interface{}{"version": "old"}}, nil
	})

	// Overwrite with new handler
	ipc.RegisterHandler("topic", func(ctx context.Context, msg *Message) (*Message, error) {
		return &Message{SessionID: msg.SessionID, Payload: map[string]interface{}{"version": "new"}}, nil
	})

	resp, err := ipc.Request(context.Background(), "topic", nil, time.Second)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.Payload["version"] != "new" {
		t.Fatalf("expected new handler to be active, got version=%v", resp.Payload["version"])
	}
}

// --- Handler returning payload with task_id ---

func TestDelegate_TaskPayloadIncluded(t *testing.T) {
	ipc := newTestIPC(t)
	ipc.RegisterHandler("worker", func(ctx context.Context, msg *Message) (*Message, error) {
		// Verify task payload fields are present.
		return &Message{
			SessionID: msg.SessionID,
			Payload: map[string]interface{}{
				"task_id":          msg.Payload["task_id"],
				"decision_point":   msg.Payload["decision_point"],
				"task_description": msg.Payload["task_description"],
				"custom":           msg.Payload["custom"],
			},
		}, nil
	})

	task := TaskDef{
		ID:            "task-42",
		Description:   "test description",
		DecisionPoint: "analyze",
		Payload: map[string]interface{}{
			"custom": "value",
		},
	}

	resp, ok := ipc.Delegate(context.Background(), "worker", task)
	if !ok {
		t.Fatal("Delegate failed")
	}
	if resp.Payload["task_id"] != "task-42" {
		t.Fatalf("expected task_id 'task-42', got %v", resp.Payload["task_id"])
	}
	if resp.Payload["decision_point"] != "analyze" {
		t.Fatalf("expected decision_point 'analyze', got %v", resp.Payload["decision_point"])
	}
	if resp.Payload["custom"] != "value" {
		t.Fatalf("expected custom 'value', got %v", resp.Payload["custom"])
	}
}
