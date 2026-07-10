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
	LoopPingPong        LoopType = "PingPong"        // Recurrence of code states (oscillation)
	LoopErrorStagnation LoopType = "ErrorStagnation"  // Edits made but error signature remains identical
	LoopErrorOscillate  LoopType = "ErrorOscillate"   // Alternating between two different errors
	LoopCostLimit       LoopType = "CostLimitExceeded" // Total API spending exceeds threshold
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

	// Avoid memory leak when slicing arrays in Go.
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

	// 1. Ping-Pong Loop Check (diff oscillation)
	startIdx := t - ld.cfg.PingPongWindow
	if startIdx < 0 {
		startIdx = 0
	}
	for i := startIdx; i < t; i++ {
		// Only check if we are actually editing code (diff is not empty)
		if ld.history[t].GitDiffHash == ld.history[i].GitDiffHash && ld.history[t].GitDiffHash != [32]byte{} {
			return LoopPingPong, true
		}
	}

	// 2. Error Stagnation Check
	if len(ld.history) >= ld.cfg.StagnationWindow {
		stagnant := true
		firstErrHash := ld.history[t].ErrorSignatureHash
		firstErrCount := ld.history[t].ErrorCount

		if firstErrCount > 0 { // Do not trigger on success states
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

	// 3. Error Oscillation Check (Ping-ponging between two distinct errors)
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
