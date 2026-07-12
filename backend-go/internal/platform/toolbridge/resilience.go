package toolbridge

import (
	"context"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var toolCallsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "multisell_tool_calls_total",
	Help: "External ToolBridge calls by tool and result.",
}, []string{"tool", "result"})

var toolCallDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "multisell_tool_call_duration_seconds",
	Help:    "External ToolBridge call duration by tool.",
	Buckets: prometheus.DefBuckets,
}, []string{"tool"})

var toolCircuitOpen = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "multisell_tool_circuit_open",
	Help: "ToolBridge circuit state by tool: 1 open, 0 closed.",
}, []string{"tool"})

func executeResilient[T any](ctx context.Context, tool string, attempts int, backoff time.Duration, tracker *ExternalCallTracker, call func(context.Context) (T, error)) (T, error) {
	var zero T
	if attempts < 1 {
		attempts = 1
	}
	if tracker != nil && !tracker.AllowCall(tool) {
		toolCallsTotal.WithLabelValues(tool, "circuit_open").Inc()
		return zero, DegradedErr
	}
	start := time.Now()
	defer func() {
		toolCallDuration.WithLabelValues(tool).Observe(time.Since(start).Seconds())
	}()
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		result, err := call(ctx)
		if tracker != nil {
			tracker.RecordCall(tool, err)
		}
		if err == nil {
			toolCallsTotal.WithLabelValues(tool, "success").Inc()
			return result, nil
		}
		lastErr = err
		toolCallsTotal.WithLabelValues(tool, "failure_attempt_"+strconv.Itoa(attempt+1)).Inc()
		if attempt == attempts-1 {
			break
		}
		timer := time.NewTimer(backoff * time.Duration(attempt+1))
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
		}
	}
	return zero, lastErr
}
