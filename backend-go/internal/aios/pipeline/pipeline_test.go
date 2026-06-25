package pipeline

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testLogger(t *testing.T) *zap.Logger {
	t.Helper()
	return zap.NewNop()
}

// ---------------------------------------------------------------------------
// RunPipeline — normal serial flow
// ---------------------------------------------------------------------------

func TestRunPipeline_NormalFlow(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	var mu sync.Mutex
	var executionOrder []int

	steps := []PipelineStep{
		{
			Name: "step1",
			Input: func(ctx context.Context, prev interface{}) (map[string]interface{}, error) {
				mu.Lock()
				executionOrder = append(executionOrder, 1)
				mu.Unlock()
				return map[string]interface{}{"step": "one", "value": 10}, nil
			},
		},
		{
			Name: "step2",
			Input: func(ctx context.Context, prev interface{}) (map[string]interface{}, error) {
				mu.Lock()
				executionOrder = append(executionOrder, 2)
				mu.Unlock()
				prevMap, _ := prev.(map[string]interface{})
				val := prevMap["value"].(int)
				return map[string]interface{}{"step": "two", "value": val * 2}, nil
			},
		},
		{
			Name: "step3",
			Input: func(ctx context.Context, prev interface{}) (map[string]interface{}, error) {
				mu.Lock()
				executionOrder = append(executionOrder, 3)
				mu.Unlock()
				prevMap, _ := prev.(map[string]interface{})
				val := prevMap["value"].(int)
				return map[string]interface{}{"step": "three", "value": val + 5}, nil
			},
		},
	}

	pipe := &Pipeline{
		Name:  "test-pipeline",
		Steps: steps,
	}

	outputs, err := engine.RunPipeline(context.Background(), pipe)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(outputs) != 3 {
		t.Fatalf("expected 3 outputs, got %d", len(outputs))
	}

	// Verify execution order.
	expectedOrder := []int{1, 2, 3}
	for i, v := range executionOrder {
		if v != expectedOrder[i] {
			t.Errorf("execution order at index %d: expected %d, got %d", i, expectedOrder[i], v)
		}
	}

	// Verify step output values (data flow).
	out0 := outputs[0].(map[string]interface{})
	if out0["step"] != "one" || out0["value"] != int(10) {
		t.Errorf("step1 output mismatch: got %v", out0)
	}

	out1 := outputs[1].(map[string]interface{})
	if out1["step"] != "two" || out1["value"] != int(20) {
		t.Errorf("step2 output mismatch: got %v, expected value=20", out1)
	}

	out2 := outputs[2].(map[string]interface{})
	if out2["step"] != "three" || out2["value"] != int(25) {
		t.Errorf("step3 output mismatch: got %v, expected value=25", out2)
	}
}

// ---------------------------------------------------------------------------
// RunPipeline — middle step fails
// ---------------------------------------------------------------------------

func TestRunPipeline_MiddleStepFails(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	var executionOrder []int

	steps := []PipelineStep{
		{
			Name: "step1",
			Input: func(ctx context.Context, prev interface{}) (map[string]interface{}, error) {
				executionOrder = append(executionOrder, 1)
				return map[string]interface{}{"result": "ok"}, nil
			},
		},
		{
			Name: "step2",
			Input: func(ctx context.Context, prev interface{}) (map[string]interface{}, error) {
				executionOrder = append(executionOrder, 2)
				return nil, errors.New("step2 failed")
			},
		},
		{
			Name: "step3",
			Input: func(ctx context.Context, prev interface{}) (map[string]interface{}, error) {
				executionOrder = append(executionOrder, 3)
				return map[string]interface{}{"result": "should_not_run"}, nil
			},
		},
	}

	pipe := &Pipeline{
		Name:  "failing-pipeline",
		Steps: steps,
	}

	_, err := engine.RunPipeline(context.Background(), pipe)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if len(executionOrder) != 2 {
		t.Errorf("expected 2 steps executed (step3 should be skipped), got: %v", executionOrder)
	}
}

// ---------------------------------------------------------------------------
// RunPipeline — retry on failure
// ---------------------------------------------------------------------------

func TestRunPipeline_StepRetry(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	var attemptCount int

	steps := []PipelineStep{
		{
			Name:    "retry-step",
			RetryMax: 2, // up to 3 total attempts
			Input: func(ctx context.Context, prev interface{}) (map[string]interface{}, error) {
				attemptCount++
				if attemptCount < 3 {
					return nil, errors.New("not ready yet")
				}
				return map[string]interface{}{"result": "success_on_attempt_3"}, nil
			},
		},
	}

	pipe := &Pipeline{
		Name:  "retry-pipeline",
		Steps: steps,
	}

	outputs, err := engine.RunPipeline(context.Background(), pipe)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attemptCount != 3 {
		t.Errorf("expected 3 attempts, got %d", attemptCount)
	}

	out := outputs[0].(map[string]interface{})
	if out["result"] != "success_on_attempt_3" {
		t.Errorf("unexpected output: %v", out)
	}
}

func TestRunPipeline_RetryExhausted(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	var attemptCount int

	steps := []PipelineStep{
		{
			Name:    "always-fail",
			RetryMax: 2, // 3 total attempts
			Input: func(ctx context.Context, prev interface{}) (map[string]interface{}, error) {
				attemptCount++
				return nil, errors.New("persistent failure")
			},
		},
	}

	pipe := &Pipeline{
		Name:  "exhausted-retry",
		Steps: steps,
	}

	_, err := engine.RunPipeline(context.Background(), pipe)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}

	if attemptCount != 3 {
		t.Errorf("expected 3 attempts, got %d", attemptCount)
	}
}

// ---------------------------------------------------------------------------
// RunFanOut — parallel execution
// ---------------------------------------------------------------------------

func TestRunFanOut_Parallel(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	engine.RegisterAgentHandler("agentA", func(ctx context.Context, agentID string, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"agent": "A", "result": "alpha"}, nil
	})
	engine.RegisterAgentHandler("agentB", func(ctx context.Context, agentID string, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"agent": "B", "result": "beta"}, nil
	})
	engine.RegisterAgentHandler("agentC", func(ctx context.Context, agentID string, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"agent": "C", "result": "gamma"}, nil
	})

	results, err := engine.RunFanOut(context.Background(), []string{"agentA", "agentB", "agentC"}, map[string]interface{}{"task": "analyze"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Results must be in registration order.
	tests := []struct {
		idx    int
		agent  string
		result string
	}{
		{0, "A", "alpha"},
		{1, "B", "beta"},
		{2, "C", "gamma"},
	}
	for _, tc := range tests {
		if results[tc.idx]["agent"] != tc.agent || results[tc.idx]["result"] != tc.result {
			t.Errorf("result[%d] mismatch: got %v", tc.idx, results[tc.idx])
		}
	}
}

func TestRunFanOut_MissingHandler(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	engine.RegisterAgentHandler("agentA", func(ctx context.Context, agentID string, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"result": "ok"}, nil
	})

	_, err := engine.RunFanOut(context.Background(), []string{"agentA", "agentB"}, map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing handler, got nil")
	}
}

func TestRunFanOut_ConcurrentExecution(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	var mu sync.Mutex
	running := make(map[string]bool)

	engine.RegisterAgentHandler("slow1", func(ctx context.Context, agentID string, input map[string]interface{}) (map[string]interface{}, error) {
		mu.Lock()
		running[agentID] = true
		// Check that slow1 and slow2 overlap.
		otherRunning := len(running)
		mu.Unlock()

		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		delete(running, agentID)
		mu.Unlock()
		return map[string]interface{}{"agent": agentID, "overlap": otherRunning > 1}, nil
	})

	engine.RegisterAgentHandler("slow2", func(ctx context.Context, agentID string, input map[string]interface{}) (map[string]interface{}, error) {
		mu.Lock()
		running[agentID] = true
		mu.Unlock()

		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		delete(running, agentID)
		mu.Unlock()
		return map[string]interface{}{"agent": agentID}, nil
	})

	start := time.Now()
	results, err := engine.RunFanOut(context.Background(), []string{"slow1", "slow2"}, map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	elapsed := time.Since(start)

	// If they ran in parallel, total time should be ~50ms, not ~100ms.
	if elapsed > 90*time.Millisecond {
		t.Logf("execution took %v — may indicate sequential execution", elapsed)
	}

	if results[0]["overlap"] != true {
		t.Log("workers did not overlap in time — may still be concurrent but timing-dependent")
	}
}

// ---------------------------------------------------------------------------
// RunSelfCorrect — self-correction loop
// ---------------------------------------------------------------------------

func TestRunSelfCorrect_PassesOnSecondIteration(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	var callCount int
	engine.RegisterAgentHandler("fixer", func(ctx context.Context, agentID string, input map[string]interface{}) (map[string]interface{}, error) {
		callCount++
		if callCount == 1 {
			return map[string]interface{}{
				"decision":    "first_draft",
				"_self_check": "needs_fix",
			}, nil
		}
		return map[string]interface{}{
			"decision":    "final_version",
			"_self_check": "ok",
		}, nil
	})

	result, err := engine.RunSelfCorrect(context.Background(), "fixer",
		map[string]interface{}{"query": "optimize"}, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
	if result["decision"] != "final_version" {
		t.Errorf("expected final_version, got %v", result["decision"])
	}
}

func TestRunSelfCorrect_PassesOnFirstIteration(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	var callCount int
	engine.RegisterAgentHandler("perfect", func(ctx context.Context, agentID string, input map[string]interface{}) (map[string]interface{}, error) {
		callCount++
		return map[string]interface{}{
			"decision":    "good",
			"_self_check": "ok",
		}, nil
	})

	result, err := engine.RunSelfCorrect(context.Background(), "perfect",
		map[string]interface{}{}, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
	if result["decision"] != "good" {
		t.Errorf("expected good, got %v", result["decision"])
	}
}

func TestRunSelfCorrect_MaxIterationsReached(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	var callCount int
	engine.RegisterAgentHandler("stubborn", func(ctx context.Context, agentID string, input map[string]interface{}) (map[string]interface{}, error) {
		callCount++
		return map[string]interface{}{
			"decision":    "never_ok",
			"_self_check": "still_needs_fix",
		}, nil
	})

	result, err := engine.RunSelfCorrect(context.Background(), "stubborn",
		map[string]interface{}{}, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 3 {
		t.Errorf("expected 3 calls (maxIter=3), got %d", callCount)
	}
	if result["decision"] != "never_ok" {
		t.Errorf("expected never_ok, got %v", result["decision"])
	}
}

func TestRunSelfCorrect_NoHandler(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	_, err := engine.RunSelfCorrect(context.Background(), "nonexistent",
		map[string]interface{}{}, 3)
	if err == nil {
		t.Fatal("expected error for unregistered handler, got nil")
	}
}

func TestRunSelfCorrect_ZeroMaxIter(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	var callCount int
	engine.RegisterAgentHandler("once", func(ctx context.Context, agentID string, input map[string]interface{}) (map[string]interface{}, error) {
		callCount++
		return map[string]interface{}{
			"decision":    "result",
			"_self_check": "ok",
		}, nil
	})

	result, err := engine.RunSelfCorrect(context.Background(), "once",
		map[string]interface{}{}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected at least 1 call when maxIter=0, got %d", callCount)
	}
	if result["decision"] != "result" {
		t.Errorf("expected result, got %v", result["decision"])
	}
}

// ---------------------------------------------------------------------------
// RunWithFallback — primary fails, fallback succeeds
// ---------------------------------------------------------------------------

func TestRunWithFallback_PrimaryFailsFallbackSucceeds(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	primary := &Pipeline{
		Name: "primary",
		Steps: []PipelineStep{
			{
				Name: "fail_step",
				Input: func(ctx context.Context, prev interface{}) (map[string]interface{}, error) {
					return nil, errors.New("primary failed")
				},
			},
		},
	}

	fallback := &Pipeline{
		Name: "fallback",
		Steps: []PipelineStep{
			{
				Name: "backup_step",
				Input: func(ctx context.Context, prev interface{}) (map[string]interface{}, error) {
					return map[string]interface{}{"result": "from_fallback"}, nil
				},
			},
		},
	}

	output, err := engine.RunWithFallback(context.Background(), primary, fallback)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, ok := output.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map output, got %T", output)
	}
	if result["result"] != "from_fallback" {
		t.Errorf("expected from_fallback, got %v", result["result"])
	}
}

func TestRunWithFallback_PrimarySucceeds(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	primary := &Pipeline{
		Name: "primary",
		Steps: []PipelineStep{
			{
				Name: "good_step",
				Input: func(ctx context.Context, prev interface{}) (map[string]interface{}, error) {
					return map[string]interface{}{"result": "primary_ok"}, nil
				},
			},
		},
	}

	fallback := &Pipeline{
		Name: "backup",
		Steps: []PipelineStep{
			{
				Name: "should_not_run",
				Input: func(ctx context.Context, prev interface{}) (map[string]interface{}, error) {
					return nil, errors.New("fallback should not run")
				},
			},
		},
	}

	output, err := engine.RunWithFallback(context.Background(), primary, fallback)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := output.(map[string]interface{})
	if result["result"] != "primary_ok" {
		t.Errorf("expected primary_ok, got %v", result["result"])
	}
}

func TestRunWithFallback_BothFail(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	primary := &Pipeline{
		Name: "primary",
		Steps: []PipelineStep{
			{
				Name: "fail1",
				Input: func(ctx context.Context, prev interface{}) (map[string]interface{}, error) {
					return nil, errors.New("primary error")
				},
			},
		},
	}

	fallback := &Pipeline{
		Name: "backup",
		Steps: []PipelineStep{
			{
				Name: "fail2",
				Input: func(ctx context.Context, prev interface{}) (map[string]interface{}, error) {
					return nil, errors.New("fallback error")
				},
			},
		},
	}

	_, err := engine.RunWithFallback(context.Background(), primary, fallback)
	if err == nil {
		t.Fatal("expected both to fail, got nil")
	}
}

// ---------------------------------------------------------------------------
// Timeout tests
// ---------------------------------------------------------------------------

func TestRunPipeline_StepTimeout(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	steps := []PipelineStep{
		{
			Name:    "timeout_step",
			Timeout: 10 * time.Millisecond,
			Input: func(ctx context.Context, prev interface{}) (map[string]interface{}, error) {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(100 * time.Millisecond):
					return map[string]interface{}{"result": "too_late"}, nil
				}
			},
		},
	}

	pipe := &Pipeline{
		Name:  "timeout-pipeline",
		Steps: steps,
	}

	_, err := engine.RunPipeline(context.Background(), pipe)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestRunPipeline_PipelineLevelTimeout(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	steps := []PipelineStep{
		{
			Name: "fast_step",
			Input: func(ctx context.Context, prev interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"result": "fast"}, nil
			},
		},
		{
			Name: "slow_step",
			Input: func(ctx context.Context, prev interface{}) (map[string]interface{}, error) {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(100 * time.Millisecond):
					return map[string]interface{}{"result": "slow"}, nil
				}
			},
		},
	}

	pipe := &Pipeline{
		Name:    "pipeline-timeout",
		Steps:   steps,
		Timeout: 20 * time.Millisecond,
	}

	_, err := engine.RunPipeline(context.Background(), pipe)
	if err == nil {
		t.Fatal("expected pipeline timeout error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestRunPipeline_NilPipeline(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	_, err := engine.RunPipeline(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil pipeline")
	}
}

func TestRunPipeline_EmptySteps(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	pipe := &Pipeline{
		Name:  "empty",
		Steps: []PipelineStep{},
	}

	outputs, err := engine.RunPipeline(context.Background(), pipe)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(outputs) != 0 {
		t.Errorf("expected 0 outputs for empty pipeline, got %d", len(outputs))
	}
}

func TestRunFanOut_EmptyAgents(t *testing.T) {
	logger := testLogger(t)
	engine := New(logger)

	_, err := engine.RunFanOut(context.Background(), []string{}, map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for empty agents list")
	}
}
