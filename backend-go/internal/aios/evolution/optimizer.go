package evolution

import (
	"context"
	"math"

	"go.uber.org/zap"
)

// DecisionSample is a single data point used by the ThresholdOptimizer to
// evaluate decision quality at different confidence thresholds.
type DecisionSample struct {
	// PredictedConfidence is the confidence level the agent assigned (0.0–1.0).
	PredictedConfidence float64 `json:"predicted_confidence"`

	// WasCorrect is true if the decision outcome was correct.
	WasCorrect bool `json:"was_correct"`

	// WasAdopted is true if the user adopted the agent's decision.
	WasAdopted bool `json:"was_adopted"`
}

// OptimizationResult contains the best threshold found and its performance.
type OptimizationResult struct {
	// BestThreshold is the confidence threshold with the highest F1 score.
	BestThreshold float64 `json:"best_threshold"`

	// Precision at the best threshold.
	Precision float64 `json:"precision"`

	// Recall at the best threshold.
	Recall float64 `json:"recall"`

	// F1Score is the harmonic mean of precision and recall at the best threshold.
	F1Score float64 `json:"f1_score"`

	// SampleCount is the number of samples used in the optimization.
	SampleCount int `json:"sample_count"`
}

// ThresholdOptimizer scans a range of confidence thresholds and returns the
// one that maximises F1 score (precision-recall balance) for the given samples.
type ThresholdOptimizer struct {
	logger *zap.Logger
}

// NewThresholdOptimizer creates a ThresholdOptimizer.
func NewThresholdOptimizer(logger *zap.Logger) *ThresholdOptimizer {
	return &ThresholdOptimizer{logger: logger}
}

// OptimizeConfidence scans confidence thresholds from 0.0 to 1.0 in 0.01
// increments and returns the threshold with the best F1 score. Only WasCorrect
// is used for precision/recall computation; WasAdopted is recorded but not
// part of the core F1 calculation.
func (o *ThresholdOptimizer) OptimizeConfidence(ctx context.Context, agentID string, samples []DecisionSample) (bestThreshold float64, result *OptimizationResult) {
	if len(samples) == 0 {
		o.logger.Warn("optimize confidence: no samples provided", zap.String("agent_id", agentID))
		return 0, &OptimizationResult{SampleCount: 0}
	}

	bestF1 := -1.0
	var bestPrecision, bestRecall float64
	var bestThresh float64

	// Scan thresholds from 0.00 to 1.00 in 0.01 steps.
	for t := 0.0; t <= 1.001; t += 0.01 {
		thresh := math.Round(t*100) / 100 // avoid floating-point drift
		precision, recall := computePrecisionRecall(samples, thresh)

		var f1 float64
		if precision+recall > 0 {
			f1 = 2 * (precision * recall) / (precision + recall)
		}

		if f1 > bestF1 {
			bestF1 = f1
			bestPrecision = precision
			bestRecall = recall
			bestThresh = thresh
		}
	}

	o.logger.Info("confidence optimization complete",
		zap.String("agent_id", agentID),
		zap.Float64("best_threshold", bestThresh),
		zap.Float64("f1", bestF1),
		zap.Int("samples", len(samples)))

	return bestThresh, &OptimizationResult{
		BestThreshold: bestThresh,
		Precision:     math.Round(bestPrecision*10000) / 10000,
		Recall:        math.Round(bestRecall*10000) / 10000,
		F1Score:       math.Round(bestF1*10000) / 10000,
		SampleCount:   len(samples),
	}
}

// computePrecisionRecall calculates precision and recall for a given
// confidence threshold.
//
//   - True positive: PredictedConfidence >= threshold AND WasCorrect == true
//   - False positive: PredictedConfidence >= threshold AND WasCorrect == false
//   - False negative: PredictedConfidence < threshold AND WasCorrect == true
func computePrecisionRecall(samples []DecisionSample, threshold float64) (precision, recall float64) {
	var tp, fp, fn float64

	for _, s := range samples {
		predicted := s.PredictedConfidence >= threshold
		if predicted && s.WasCorrect {
			tp++
		} else if predicted && !s.WasCorrect {
			fp++
		} else if !predicted && s.WasCorrect {
			fn++
		}
	}

	if tp+fp > 0 {
		precision = tp / (tp + fp)
	}
	if tp+fn > 0 {
		recall = tp / (tp + fn)
	}

	return precision, recall
}
