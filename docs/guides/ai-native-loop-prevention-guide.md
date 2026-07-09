# AI Development Loop Monitor & Prevention Guide

> **Purpose**: Guide AI coding agents on how to monitor autonomous edit cycles, normalize compiler/test errors, detect oscillations or stagnations, and handle rollback-to-owner triggers.

---

## 1. The Loop Problem
When autonomous coding agents work on complex tasks, they can get trapped in repetitive edit loops (e.g. *Fix Error A -> breaks B -> Fix B -> breaks A*).

To prevent this from wasting API tokens and CPU time, the **AIOS Observability** module tracks development steps, compiling a history of codebase signatures, and halts execution when loop rules are triggered.

---

## 2. Code State Space Formula
At each iteration step $t$, the system computes a state signature:

$$\text{State}_t = \langle \text{GitDiffHash}_t, \text{ErrorSignatureHash}_t, \text{ErrorCount}_t \rangle$$

1. **`GitDiffHash`**: SHA-256 hash of the current git diff compared to a *fixed baseline commit* established at the start of the session. If the agent reverts back, the diff is empty, hashing consistently.
2. **`ErrorSignatureHash`**: SHA-256 hash of the normalized output of BOTH compilers (linter/vet) and test runner failure logs.
3. **`ErrorCount`**: Total number of compilation errors and failing unit/E2E test assertions.

---

## 3. Regular Expression Normalization
Raw compiler and test output contain dynamic variables (line offsets, absolute file paths, container IDs, pointers, dates) that change on every run.

To create a stable hash, the normalizer strips these variables using regex replacements in the following sequence:

| Step | Pattern | Match Example | Replacement |
|---|---|---|---|
| 1 | `\` $\rightarrow$ `/` | `\Users\lc\multisell` | `/Users/lc/multisell` |
| 2 | Timestamps | `2026-07-09T20:25:00Z` | `<timestamp>` |
| 3 | Log Times | `20:25:00.123` | `<time>` |
| 4 | UUIDs | `824097d7-89d5-4f03-887e-ae77e17156e2` | `<uuid>` |
| 5 | Sandbox Containers | `backend-pr-123` | `<sandbox-service>` |
| 6 | Hex Pointers | `0xc0000a60f0` | `<ptr>` |
| 7 | Absolute Paths | `/Users/lc/multisell/main.go` | `<path>` |
| 8 | Line & Column Numbers | `:127:32` | `:<line>` |

---

## 4. Loop Detection Algorithms

### 4.1 Ping-Pong Loop (Oscillation Check)
Checks if the current code diff matches a diff state visited within the sliding window of the last 6 steps:
$$\exists i \in [t-W, t-1] \text{ s.t. } \text{GitDiffHash}_t == \text{GitDiffHash}_i$$
*Note: This check must ignore empty diff states (success runs) to prevent false positives.*

### 4.2 Error Stagnation (Definition of Insanity Check)
Checks if the agent is making code modifications, but the compiler or test runner produces the exact same normalized error signature for 3 consecutive edits:
$$\text{GitDiffHash}_t \neq \text{GitDiffHash}_{t-1} \quad \text{AND} \quad \text{ErrorSignatureHash}_t == \text{ErrorSignatureHash}_{t-1} == \text{ErrorSignatureHash}_{t-2}$$

### 4.3 Error Oscillation (Alternating Error Check)
Checks if the agent is caught ping-ponging between two different errors (fixing A causes B, fixing B causes A):
$$E_t == E_{t-2} \quad \text{AND} \quad E_{t-1} == E_{t-3} \quad \text{AND} \quad E_t \neq E_{t-1}$$

---

## 5. Loop Detector Implementation (`observability/loop_detector.go`)

```go
package observability

import (
	"crypto/sha256"
	"regexp"
	"strings"
	"sync"
)

type LoopType string

const (
	LoopNone            LoopType = "None"
	LoopPingPong        LoopType = "PingPong"        // Recurrence of code states
	LoopErrorStagnation LoopType = "ErrorStagnation"  // Edits made but error remains identical
	LoopErrorOscillate  LoopType = "ErrorOscillate"   // Alternating between two different errors
	LoopCostLimit       LoopType = "CostLimitExceeded" // Cumulative cost threshold reached
)

type CodeState struct {
	GitDiffHash        [32]byte
	ErrorSignatureHash [32]byte
	ErrorCount         int
}

type Config struct {
	MaxCostUSD         float64
	PingPongWindow     int
	StagnationWindow   int
	OscillationWindow  int
}

type LoopDetector struct {
	mu             sync.Mutex
	cfg            Config
	history        []CodeState
	cumulativeCost float64

	pathRegex      *regexp.Regexp
	ptrRegex       *regexp.Regexp
	lineRegex      *regexp.Regexp
	timestampRegex *regexp.Regexp
	timeRegex      *regexp.Regexp
	uuidRegex      *regexp.Regexp
	sandboxRegex   *regexp.Regexp
}

func NewLoopDetector(cfg Config) *LoopDetector {
	if cfg.PingPongWindow <= 0 {
		cfg.PingPongWindow = 6
	}
	if cfg.StagnationWindow <= 0 {
		cfg.StagnationWindow = 3
	}
	if cfg.OscillationWindow <= 0 {
		cfg.OscillationWindow = 4
	}
	if cfg.MaxCostUSD <= 0 {
		cfg.MaxCostUSD = 2.00
	}

	return &LoopDetector{
		cfg:            cfg,
		history:        make([]CodeState, 0),
		cumulativeCost: 0.0,
		pathRegex:      regexp.MustCompile(`/[a-zA-Z0-9_\-\.\+]+(?:/[a-zA-Z0-9_\-\.\+]+)+`),
		ptrRegex:       regexp.MustCompile(`\b0x[0-9a-fA-F]+\b`),
		lineRegex:      regexp.MustCompile(`(?::\d+)+`),
		timestampRegex: regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?\b`),
		timeRegex:      regexp.MustCompile(`\b\d{2}:\d{2}:\d{2}(?:\.\d+)?\b`),
		uuidRegex:      regexp.MustCompile(`\b[0-9a-fA-F]{8}(?:-[0-9a-fA-F]{4}){3}-[0-9a-fA-F]{12}\b`),
		sandboxRegex:   regexp.MustCompile(`\b(?:db|backend|frontend|migrate|seed|e2e)-[a-zA-Z0-9_\-]+\b`),
	}
}

func (ld *LoopDetector) NormalizeError(errStr string) string {
	if errStr == "" {
		return ""
	}
	s := strings.ReplaceAll(errStr, "\\", "/")
	s = ld.timestampRegex.ReplaceAllString(s, "<timestamp>")
	s = ld.timeRegex.ReplaceAllString(s, "<time>")
	s = ld.uuidRegex.ReplaceAllString(s, "<uuid>")
	s = ld.sandboxRegex.ReplaceAllString(s, "<sandbox-service>")
	s = ld.ptrRegex.ReplaceAllString(s, "<ptr>")
	s = ld.pathRegex.ReplaceAllString(s, "<path>")
	s = ld.lineRegex.ReplaceAllString(s, ":<line>")
	return s
}

func (ld *LoopDetector) RecordStep(gitDiff string, errOutput string, errorCount int, stepCost float64) (LoopType, bool) {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	ld.cumulativeCost += stepCost
	if ld.cumulativeCost > ld.cfg.MaxCostUSD {
		return LoopCostLimit, true
	}

	diffHash := sha256.Sum256([]byte(gitDiff))
	normalizedErr := ld.NormalizeError(errOutput)
	errHash := sha256.Sum256([]byte(normalizedErr))

	state := CodeState{
		GitDiffHash:        diffHash,
		ErrorSignatureHash: errHash,
		ErrorCount:         errorCount,
	}

	ld.history = append(ld.history, state)
	t := len(ld.history) - 1

	// Avoid memory leak when slicing history slice
	maxWindow := ld.cfg.PingPongWindow
	if ld.cfg.OscillationWindow > maxWindow {
		maxWindow = ld.cfg.OscillationWindow
	}
	if len(ld.history) > maxWindow*3 {
		newHistory := make([]CodeState, maxWindow*2)
		copy(newHistory, ld.history[len(ld.history)-maxWindow*2:])
		ld.history = newHistory
		t = len(ld.history) - 1
	}

	if t < 1 {
		return LoopNone, false
	}

	// 1. Ping-Pong Loop Check
	startIdx := t - ld.cfg.PingPongWindow
	if startIdx < 0 {
		startIdx = 0
	}
	for i := startIdx; i < t; i++ {
		if ld.history[t].GitDiffHash == ld.history[i].GitDiffHash && ld.history[t].GitDiffHash != [32]byte{} {
			return LoopPingPong, true
		}
	}

	// 2. Error Stagnation Check
	if len(ld.history) >= ld.cfg.StagnationWindow {
		stagnant := true
		firstErrHash := ld.history[t].ErrorSignatureHash
		firstErrCount := ld.history[t].ErrorCount

		if firstErrCount > 0 {
			for i := t - ld.cfg.StagnationWindow + 1; i <= t; i++ {
				if ld.history[i].ErrorSignatureHash != firstErrHash {
					stagnant = false
					break
				}
				if i > t-ld.cfg.StagnationWindow+1 {
					if ld.history[i].GitDiffHash == ld.history[i-1].GitDiffHash {
						stagnant = false
						break
					}
				}
			}
			if stagnant {
				return LoopErrorStagnation, true
			}
		}
	}

	// 3. Error Oscillation Check
	if len(ld.history) >= 4 {
		currErr := ld.history[t].ErrorSignatureHash
		prevErr := ld.history[t-1].ErrorSignatureHash
		twoBackErr := ld.history[t-2].ErrorSignatureHash
		threeBackErr := ld.history[t-3].ErrorSignatureHash

		if currErr == twoBackErr && prevErr == threeBackErr && currErr != prevErr {
			if ld.history[t].GitDiffHash != ld.history[t-1].GitDiffHash &&
				ld.history[t-1].GitDiffHash != ld.history[t-2].GitDiffHash {
				if ld.history[t].ErrorCount > 0 && ld.history[t-1].ErrorCount > 0 {
					return LoopErrorOscillate, true
				}
			}
		}
	}

	return LoopNone, false
}
```

---

## 6. Fallback and Owner Alerting Protocol
When a loop is detected, the executor halts the agent and runs the recovery loop:
1. **Preservation**: Run `git diff > .loop/stuck-agent-diff.patch` to save progress, and then hard reset: `git reset --hard && git clean -fd`.
2. **Escalate**: Render loop diagnostic logs (affected files, normalized errors, patch file links) and output the warning for Owner decision.
