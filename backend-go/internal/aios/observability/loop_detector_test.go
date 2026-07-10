package observability

import (
	"crypto/sha256"
	"testing"
)

func TestLoopDetector_PingPong(t *testing.T) {
	ld := NewLoopDetector(Config{PingPongWindow: 3, MaxCostUSD: 1.00})

	// Step 1: Baseline edit
	_, blocked := ld.RecordStep("diffA", "errorA", 1, 0.01)
	if blocked {
		t.Fatal("Unexpected block on first step")
	}

	// Step 2: Second code state
	_, blocked = ld.RecordStep("diffB", "errorB", 1, 0.01)
	if blocked {
		t.Fatal("Unexpected block on second step")
	}

	// Step 3: Revert to first code state (ping-pong oscillation)
	loopType, blocked := ld.RecordStep("diffA", "errorA", 1, 0.01)
	if !blocked || loopType != LoopPingPong {
		t.Fatalf("Expected PingPong loop detection, got loopType=%s blocked=%t", loopType, blocked)
	}
}

func TestLoopDetector_SameErrorStagnation(t *testing.T) {
	ld := NewLoopDetector(Config{StagnationWindow: 3, MaxCostUSD: 1.00})

	// Step 1: Error output containing dynamic pointer and path
	_, blocked := ld.RecordStep("diffA", "error at /Users/lc/multisell/main.go:12: struct 0xc0001", 1, 0.01)
	if blocked {
		t.Fatal("Unexpected block")
	}

	// Step 2: Altered diff, but identical normalized error (line number shifted, pointer changed)
	_, blocked = ld.RecordStep("diffB", "error at /Users/lc/multisell/main.go:15: struct 0xc0002", 1, 0.01)
	if blocked {
		t.Fatal("Unexpected block")
	}

	// Step 3: Altered diff again, identical normalized error
	loopType, blocked := ld.RecordStep("diffC", "error at /Users/lc/multisell/main.go:18: struct 0xc0003", 1, 0.01)
	if !blocked || loopType != LoopErrorStagnation {
		t.Fatalf("Expected Same-Error Stagnation detection, got loopType=%s blocked=%t", loopType, blocked)
	}
}

func TestLoopDetector_ErrorOscillation(t *testing.T) {
	ld := NewLoopDetector(Config{OscillationWindow: 4, MaxCostUSD: 1.00})

	// Step 1: Error A
	_, blocked := ld.RecordStep("diffA", "errorA", 1, 0.01)
	if blocked {
		t.Fatal("Unexpected block")
	}

	// Step 2: Error B
	_, blocked = ld.RecordStep("diffB", "errorB", 1, 0.01)
	if blocked {
		t.Fatal("Unexpected block")
	}

	// Step 3: Error A
	_, blocked = ld.RecordStep("diffC", "errorA", 1, 0.01)
	if blocked {
		t.Fatal("Unexpected block")
	}

	// Step 4: Error B again (oscillation)
	loopType, blocked := ld.RecordStep("diffD", "errorB", 1, 0.01)
	if !blocked || loopType != LoopErrorOscillate {
		t.Fatalf("Expected ErrorOscillate loop detection, got loopType=%s blocked=%t", loopType, blocked)
	}
}

func TestLoopDetector_CostLimit(t *testing.T) {
	ld := NewLoopDetector(Config{MaxCostUSD: 0.50})

	_, blocked := ld.RecordStep("diffA", "errorA", 1, 0.30)
	if blocked {
		t.Fatal("Unexpected block within limit")
	}

	loopType, blocked := ld.RecordStep("diffB", "errorB", 1, 0.25)
	if !blocked || loopType != LoopCostLimit {
		t.Fatalf("Expected CostLimitExceeded loop detection, got loopType=%s blocked=%t", loopType, blocked)
	}
}

func TestLoopDetector_MemorySafeResizing(t *testing.T) {
	ld := NewLoopDetector(Config{PingPongWindow: 3, OscillationWindow: 4})

	// maxWindow = 4. maxWindow * 3 = 12.
	// Push 13 states.
	for i := 0; i < 13; i++ {
		diff := string(rune('A' + i))
		err := string(rune('a' + i))
		_, blocked := ld.RecordStep(diff, err, 1, 0.001)
		if blocked {
			t.Fatalf("Unexpected block on step %d", i)
		}
	}

	// After 13th step, history length should have been resized to maxWindow * 2 = 8
	if len(ld.history) != 8 {
		t.Fatalf("Expected history length to be resized to 8, got %d", len(ld.history))
	}

	// Verify the last elements are preserved correctly
	// The last elements pushed were 'E'..'M' (indices 4..12)
	// E is index 4 (5th element). So indices 5..12 are F..M.
	for idx, i := range []int{5, 6, 7, 8, 9, 10, 11, 12} {
		expectedDiff := string(rune('A' + i))
		expectedHash := sha256.Sum256([]byte(expectedDiff))
		if ld.history[idx].GitDiffHash != expectedHash {
			t.Errorf("At history index %d, expected diff state %q, got hash %x", idx, expectedDiff, ld.history[idx].GitDiffHash)
		}
	}
}
