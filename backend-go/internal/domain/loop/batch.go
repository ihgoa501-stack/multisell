package loop

import (
	"go.uber.org/zap"
)

// BatchEvaluateInput is the payload for triggering a batch evaluation.
type BatchEvaluateInput struct {
	ProductIDs  []int64 `json:"product_ids" binding:"required"`
	TriggeredBy string  `json:"triggered_by"`
}

// BatchEvaluate runs the full Evaluate pipeline for each product.
// If one product fails, it continues with the others and includes the error in the result.
func (s *Service) BatchEvaluate(productIDs []int64, triggeredBy string) []*EvaluateResult {
	if triggeredBy == "" {
		triggeredBy = "system"
	}
	results := make([]*EvaluateResult, 0, len(productIDs))
	for _, pid := range productIDs {
		result, err := s.Evaluate(pid, triggeredBy)
		if err != nil {
			results = append(results, &EvaluateResult{
				ProductID: pid,
				Error:     err.Error(),
			})
			s.logger.Error("batch evaluate product failed",
				zap.Int64("product_id", pid),
				zap.Error(err),
			)
			continue
		}
		results = append(results, result)
	}
	return results
}
