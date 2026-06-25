// Package evolution provides the Autonomous Evolution platform for AIOS —
// A/B testing, threshold optimization, and behavior auditing for agents.
//
// This extends the existing domain/evolution (trust-score nudges) with
// proactive self-improvement capabilities: structured experimentation,
// automated parameter tuning, and periodic behavior analysis.
package evolution

import (
	"fmt"
	"math"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ABTest defines an A/B experiment comparing two agent behavior variants.
// The test splits agent requests between VariantA and VariantB according to
// TrafficA, collects metrics per variant, and declares a winner when ended.
type ABTest struct {
	// ID is a unique identifier for this experiment.
	ID string `json:"id"`

	// AgentID identifies the agent being experimented on.
	AgentID string `json:"agent_id"`

	// VariantA is the control variant — a prompt version, config key, or
	// behavior template identifier.
	VariantA string `json:"variant_a"`

	// VariantB is the treatment variant being compared.
	VariantB string `json:"variant_b"`

	// TrafficA is the fraction (0.0–1.0) of requests routed to VariantA.
	// VariantB receives 1-TrafficA. A TrafficA of 0.5 means equal split.
	TrafficA float64 `json:"traffic_a"`

	// StartedAt is when the experiment began.
	StartedAt time.Time `json:"started_at"`

	// EndedAt is set when the experiment concludes. Nil while running.
	EndedAt *time.Time `json:"ended_at,omitempty"`

	// Metrics lists the metrics this test collects and compares.
	// Supported: "adoption_rate", "accuracy", "latency", "confidence".
	Metrics []string `json:"metrics"`

	// Winner records the result: "A", "B", or "tie". Nil while running.
	Winner *string `json:"winner,omitempty"`

	// observations stores per-variant metric values collected during the test.
	observations map[string][]float64 `json:"-"`
}

// ExperimentResult is the computed outcome of a completed A/B experiment.
type ExperimentResult struct {
	VariantA   string  `json:"variant_a"`
	VariantB   string  `json:"variant_b"`
	MetricA    float64 `json:"metric_a"`
	MetricB    float64 `json:"metric_b"`
	Lift       float64 `json:"lift"`
	SampleSize int     `json:"sample_size"`
	Duration   string  `json:"duration"`
	Winner     string  `json:"winner"`
}

// ABTestManager manages the lifecycle of A/B experiments. It is safe for
// concurrent use once created via NewABTestManager.
type ABTestManager struct {
	logger *zap.Logger
	mu     sync.RWMutex
	tests  map[string]*ABTest
}

// NewABTestManager creates a new ABTestManager.
func NewABTestManager(logger *zap.Logger) *ABTestManager {
	return &ABTestManager{
		logger: logger,
		tests:  make(map[string]*ABTest),
	}
}

// StartTest begins a new A/B experiment and returns the created test.
// The ID must be unique; an error is returned if it already exists.
func (m *ABTestManager) StartTest(id, agentID, variantA, variantB string, trafficA float64, metrics []string) (*ABTest, error) {
	if id == "" {
		return nil, fmt.Errorf("abtest: id must not be empty")
	}
	if trafficA < 0 || trafficA > 1 {
		return nil, fmt.Errorf("abtest: trafficA must be between 0.0 and 1.0, got %f", trafficA)
	}
	if len(metrics) == 0 {
		return nil, fmt.Errorf("abtest: at least one metric is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tests[id]; exists {
		return nil, fmt.Errorf("abtest: test %q already exists", id)
	}

	test := &ABTest{
		ID:           id,
		AgentID:      agentID,
		VariantA:     variantA,
		VariantB:     variantB,
		TrafficA:     trafficA,
		StartedAt:    time.Now(),
		Metrics:      metrics,
		observations: make(map[string][]float64),
	}
	m.tests[id] = test

	m.logger.Info("ab test started",
		zap.String("id", id),
		zap.String("agent_id", agentID),
		zap.Float64("traffic_a", trafficA),
		zap.Strings("metrics", metrics))

	return test, nil
}

// RecordResult records a single observation for the specified variant in an
// ongoing experiment. The metric value is appended to that variant's list.
// Returns an error if the test does not exist, has already ended, or the
// variant is unrecognised.
func (m *ABTestManager) RecordResult(testID, variant string, metricValue float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	test, ok := m.tests[testID]
	if !ok {
		return fmt.Errorf("abtest: test %q not found", testID)
	}
	if test.EndedAt != nil {
		return fmt.Errorf("abtest: test %q has already ended", testID)
	}
	if variant != test.VariantA && variant != test.VariantB {
		return fmt.Errorf("abtest: variant %q is not part of test %q (A=%q, B=%q)",
			variant, testID, test.VariantA, test.VariantB)
	}
	if math.IsNaN(metricValue) || math.IsInf(metricValue, 0) {
		return fmt.Errorf("abtest: metric value must be finite, got %f", metricValue)
	}

	test.observations[variant] = append(test.observations[variant], metricValue)
	return nil
}

// EndTest concludes an experiment, computes the result, and declares a winner.
// The test's EndedAt and Winner fields are set. The returned ExperimentResult
// summarises the outcome. Returns an error if the test does not exist or has
// already ended.
func (m *ABTestManager) EndTest(testID string) (*ExperimentResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	test, ok := m.tests[testID]
	if !ok {
		return nil, fmt.Errorf("abtest: test %q not found", testID)
	}
	if test.EndedAt != nil {
		return nil, fmt.Errorf("abtest: test %q has already ended", testID)
	}

	now := time.Now()
	duration := now.Sub(test.StartedAt)
	test.EndedAt = &now

	obsA := test.observations[test.VariantA]
	obsB := test.observations[test.VariantB]

	metricA := mean(obsA)
	metricB := mean(obsB)
	sampleSize := len(obsA) + len(obsB)

	var lift float64
	var winner string

	if sampleSize == 0 {
		winner = "tie"
		lift = 0
	} else if metricA == metricB {
		winner = "tie"
		lift = 0
	} else if isBetter(metricA, metricB, test.Metrics) {
		if metricB != 0 {
			lift = ((metricA - metricB) / math.Abs(metricB)) * 100
		} else {
			lift = 100
		}
		winner = "A"
	} else {
		if metricA != 0 {
			lift = ((metricB - metricA) / math.Abs(metricA)) * 100
		} else {
			lift = 100
		}
		winner = "B"
	}


	test.Winner = &winner

	result := &ExperimentResult{
		VariantA:   test.VariantA,
		VariantB:   test.VariantB,
		MetricA:    metricA,
		MetricB:    metricB,
		Lift:       math.Round(lift*100) / 100,
		SampleSize: sampleSize,
		Duration:   fmt.Sprintf("%.0fs", duration.Seconds()),
		Winner:     winner,
	}

	m.logger.Info("ab test ended",
		zap.String("id", testID),
		zap.String("winner", winner),
		zap.Float64("lift", result.Lift),
		zap.Int("sample_size", sampleSize))

	return result, nil
}

// GetTest returns a copy of the experiment with the given ID, or nil if not
// found. The returned copy strips internal observation data.
func (m *ABTestManager) GetTest(testID string) *ABTest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	test, ok := m.tests[testID]
	if !ok {
		return nil
	}
	copy := *test
	copy.observations = nil
	return &copy
}

// ListTests returns all experiments, optionally filtered by agent ID.
func (m *ABTestManager) ListTests(agentID string) []*ABTest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*ABTest
	for _, test := range m.tests {
		if agentID != "" && test.AgentID != agentID {
			continue
		}
		copy := *test
		copy.observations = nil
		result = append(result, &copy)
	}
	return result
}

// GetExperimentResult computes and returns the ExperimentResult for an
// already-ended test. Returns an error if the test does not exist or is
// still running.
func (m *ABTestManager) GetExperimentResult(testID string) (*ExperimentResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	test, ok := m.tests[testID]
	if !ok {
		return nil, fmt.Errorf("abtest: test %q not found", testID)
	}
	if test.EndedAt == nil {
		return nil, fmt.Errorf("abtest: test %q is still running", testID)
	}

	obsA := test.observations[test.VariantA]
	obsB := test.observations[test.VariantB]
	sampleSize := len(obsA) + len(obsB)

	var lift float64
	if sampleSize > 0 {
		if *test.Winner == "A" {
			if mean(obsB) != 0 {
				lift = ((mean(obsA) - mean(obsB)) / math.Abs(mean(obsB))) * 100
			} else {
				lift = 100
			}
		} else if *test.Winner == "B" {
			if mean(obsA) != 0 {
				lift = ((mean(obsB) - mean(obsA)) / math.Abs(mean(obsA))) * 100
			} else {
				lift = 100
			}
		}
	}

	return &ExperimentResult{
		VariantA:   test.VariantA,
		VariantB:   test.VariantB,
		MetricA:    mean(obsA),
		MetricB:    mean(obsB),
		Lift:       math.Round(lift*100) / 100,
		SampleSize: sampleSize,
		Duration:   fmt.Sprintf("%.0fs", test.EndedAt.Sub(test.StartedAt).Seconds()),
		Winner:     *test.Winner,
	}, nil
}

// mean returns the arithmetic mean of a float64 slice. Returns 0.0 for empty.
func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// isBetter returns true if metricA beats metricB for the given metric list.
// For latency, lower is better; for all others, higher is better.
func isBetter(a, b float64, metrics []string) bool {
	if len(metrics) > 0 && metrics[0] == "latency" {
		return a < b
	}
	return a > b
}
