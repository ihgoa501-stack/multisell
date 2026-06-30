package agentlearning

import (
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides agent learning loop operations.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new agent learning service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// EvaluateDecision compares a decision's predicted outcome with actual
// results by querying sales orders linked to the decision's product.
//   - For pricing decisions (A3/acos_analysis): compares estimated profit margin
//     with the actual profit margin from completed orders.
//   - For sourcing decisions (A8/sourcing_scan): compares estimated demand
//     with actual units sold.
//   - Other agents use a simplified accuracy check based on order data.
func (s *Service) EvaluateDecision(decisionTraceID int64, period string) error {
	// 1. Find the decision record (pre_listing_decision).
	type preListingRow struct {
		ID                int64
		SkuID             int64
		DecisionPoint     string
		EstimatedProfit   float64
		ProfitMargin      float64
		TraceID           string
	}
	var row preListingRow
	if err := s.db.Raw(`
		SELECT pld.id, pld.sku_id, pld.decision_point, pld.estimated_profit, pld.profit_margin, pld.trace_id
		FROM pre_listing_decision pld
		WHERE pld.id = ?
	`, decisionTraceID).Scan(&row).Error; err != nil {
		return fmt.Errorf("pre_listing_decision lookup: %w", err)
	}
	if row.ID == 0 {
		return fmt.Errorf("decision trace %d not found", decisionTraceID)
	}

	// 2. Resolve sku_id -> product_id to find sales orders.
	var productID int64
	if err := s.db.Raw(`SELECT product_id FROM sku WHERE id = ?`, row.SkuID).Scan(&productID).Error; err != nil {
		return fmt.Errorf("sku lookup: %w", err)
	}

	// 3. Determine agent_id from trace_id.
	var agentID string
	if err := s.db.Raw(`SELECT agent_id FROM ai_trace WHERE trace_id = ?`, row.TraceID).Scan(&agentID).Error; err != nil {
		// Fallback: derive from decision point.
		agentID = s.guessAgent(row.DecisionPoint)
	}
	if agentID == "" {
		agentID = s.guessAgent(row.DecisionPoint)
	}

	// 4. Query actual outcomes from sales_order_item joined with sales_order.
	type actualRow struct {
		TotalSold    int
		ProfitTotal  float64
		OrderCount   int
	}
	var actual actualRow
	s.db.Raw(`
		SELECT
			COALESCE(SUM(soi.quantity), 0) AS total_sold,
			COALESCE(SUM(so.profit_amount), 0) AS profit_total,
			COUNT(DISTINCT so.id) AS order_count
		FROM sales_order_item soi
		JOIN sales_order so ON so.id = soi.order_id
		WHERE soi.product_id = ?
		  AND so.status IN ('delivered','completed')
	`, productID).Scan(&actual)

	// 5. Compute score based on agent type.
	var score float64
	var predictedOutcome, actualOutcome string

	switch agentID {
	case "A3":
		// Pricing: compare predicted profit_margin with actual.
		actualMargin := 0.0
		if actual.ProfitTotal > 0 && actual.TotalSold > 0 {
			avgPrice := s.getAvgPrice(row.SkuID)
			if avgPrice > 0 {
				actualMargin = (actual.ProfitTotal / float64(actual.TotalSold)) / avgPrice
			}
		}
		predictedOutcome = fmt.Sprintf(`{"estimated_profit": %.2f, "profit_margin": %.4f}`, row.EstimatedProfit, row.ProfitMargin)
		actualOutcome = fmt.Sprintf(`{"profit_total": %.2f, "total_sold": %d, "actual_margin": %.4f}`, actual.ProfitTotal, actual.TotalSold, actualMargin)
		score = s.computeMarginAccuracy(row.ProfitMargin, actualMargin)

	case "A8":
		// Sourcing: compare predicted demand with actual units sold.
		predictedDemand := s.getEstimatedDemand(row.TraceID)
		predictedOutcome = fmt.Sprintf(`{"estimated_demand": %d}`, predictedDemand)
		actualOutcome = fmt.Sprintf(`{"total_sold": %d, "order_count": %d}`, actual.TotalSold, actual.OrderCount)
		if predictedDemand > 0 {
			ratio := float64(actual.TotalSold) / float64(predictedDemand)
			if ratio > 1.0 {
				ratio = 1.0 / ratio
			}
			score = math.Round(ratio*100) / 100
		} else {
			score = 0.5
		}

	default:
		// Generic: use order fulfillment as success signal.
		predictedOutcome = fmt.Sprintf(`{"estimated_profit": %.2f}`, row.EstimatedProfit)
		actualOutcome = fmt.Sprintf(`{"total_sold": %d, "profit_total": %.2f}`, actual.TotalSold, actual.ProfitTotal)
		if row.EstimatedProfit > 0 && actual.ProfitTotal > 0 {
			ratio := actual.ProfitTotal / row.EstimatedProfit
			if ratio > 1.0 {
				ratio = 1.0 / ratio
			}
			score = math.Round(ratio*100) / 100
		} else {
			score = 0.5
		}
	}

	// 6. Persist the evaluation.
	now := time.Now()
	ev := DecisionEvaluation{
		DecisionTraceID:  decisionTraceID,
		ProductID:        productID,
		AgentID:          agentID,
		PredictedOutcome: predictedOutcome,
		ActualOutcome:    actualOutcome,
		Score:            score,
		EvaluatedAt:      &now,
		EvaluationType:   period,
	}
	if err := s.db.Create(&ev).Error; err != nil {
		return fmt.Errorf("save evaluation: %w", err)
	}

	s.logger.Info("decision evaluated",
		zap.Int64("decision_trace_id", decisionTraceID),
		zap.String("agent_id", agentID),
		zap.Float64("score", score),
		zap.String("period", period),
	)
	return nil
}

// RecalculateAccuracy rebuilds AgentAccuracy records for a specific agent and period.
// Accuracy = decisions with score >= 0.5 / total decisions.
// Trend is determined by comparing accuracy across time windows
// (e.g., first half vs second half of the period).
func (s *Service) RecalculateAccuracy(agentID, period string) error {
	// Determine lookback duration.
	hours := periodHours(period)
	if hours <= 0 {
		return fmt.Errorf("invalid period: %s", period)
	}
	since := time.Now().Add(-hours)

	// Count total and correct decisions.
	var stats struct {
		Total   int
		Correct int
	}
	s.db.Raw(`
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE score >= 0.5) AS correct
		FROM decision_evaluation
		WHERE agent_id = ?
		  AND created_at >= ?
	`, agentID, since).Scan(&stats)

	accuracyPct := 0.0
	if stats.Total > 0 {
		accuracyPct = math.Round(float64(stats.Correct)/float64(stats.Total)*10000) / 100
	}

	// Determine trend by comparing first half to second half of the period.
	trend := s.computeTrend(agentID, since, hours)

	upsert := map[string]interface{}{
		"agent_id":          agentID,
		"period":            period,
		"total_decisions":   stats.Total,
		"correct_decisions": stats.Correct,
		"accuracy_pct":      accuracyPct,
		"trend":             trend,
	}

	// Upsert: insert or update on conflict (agent_id, period).
	var existing AgentAccuracy
	if err := s.db.Where("agent_id = ? AND period = ?", agentID, period).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return s.db.Create(&AgentAccuracy{
				AgentID:          agentID,
				Period:           period,
				TotalDecisions:   stats.Total,
				CorrectDecisions: stats.Correct,
				AccuracyPct:      accuracyPct,
				Trend:            trend,
			}).Error
		}
		return err
	}
	return s.db.Model(&existing).Updates(upsert).Error
}

// GetAllAccuracy returns all agent accuracy records.
func (s *Service) GetAllAccuracy() ([]AgentAccuracy, error) {
	var records []AgentAccuracy
	if err := s.db.Order("accuracy_pct DESC, agent_id ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

// GetAccuracyByAgent returns accuracy records for a specific agent.
func (s *Service) GetAccuracyByAgent(agentID string) ([]AgentAccuracy, error) {
	var records []AgentAccuracy
	if err := s.db.Where("agent_id = ?", agentID).Order("period ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

// ListEvaluations returns evaluation records, optionally filtered by agent_id and product_id.
func (s *Service) ListEvaluations(agentID string, productID int64) ([]DecisionEvaluation, error) {
	q := s.db.Model(&DecisionEvaluation{}).Order("created_at DESC")
	if agentID != "" {
		q = q.Where("agent_id = ?", agentID)
	}
	if productID > 0 {
		q = q.Where("product_id = ?", productID)
	}
	var evals []DecisionEvaluation
	if err := q.Find(&evals).Error; err != nil {
		return nil, err
	}
	return evals, nil
}

// EvaluateRecentTraces runs evaluations for all pre_listing_decision records
// created more than 30 days ago that don't yet have a matching evaluation.
func (s *Service) EvaluateRecentTraces() error {
	type pendingRow struct {
		ID          int64
		CreatedAt   time.Time
	}
	var pending []pendingRow
	s.db.Raw(`
		SELECT pld.id, pld.created_at
		FROM pre_listing_decision pld
		LEFT JOIN decision_evaluation de ON de.decision_trace_id = pld.id
		WHERE de.id IS NULL
		  AND pld.created_at <= NOW() - INTERVAL '30 days'
		ORDER BY pld.created_at ASC
		LIMIT 100
	`).Scan(&pending)

	for _, p := range pending {
		period := "T+30"
		ageDays := time.Since(p.CreatedAt).Hours() / 24
		if ageDays >= 90 {
			period = "T+90"
		} else if ageDays >= 60 {
			period = "T+60"
		}
		if err := s.EvaluateDecision(p.ID, period); err != nil {
			s.logger.Warn("evaluate decision failed",
				zap.Int64("decision_id", p.ID),
				zap.Error(err),
			)
		}
	}
	return nil
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func (s *Service) guessAgent(decisionPoint string) string {
	switch decisionPoint {
	case "acos_analysis", "price_watch", "profit_watch":
		return "A3"
	case "sourcing_scan":
		return "A8"
	case "listing_optimize":
		return "A2"
	case "discount_risk_check":
		return "G3"
	default:
		return "A3"
	}
}

func (s *Service) getAvgPrice(skuID int64) float64 {
	var price float64
	s.db.Raw(`SELECT COALESCE(price, 0) FROM sku WHERE id = ?`, skuID).Scan(&price)
	return price
}

func (s *Service) getEstimatedDemand(traceID string) int {
	var demand int
	s.db.Raw(`
		SELECT COALESCE((final_output->>'estimated_demand')::int, 0)
		FROM ai_trace WHERE trace_id = ?
	`, traceID).Scan(&demand)
	return demand
}

func (s *Service) computeMarginAccuracy(predicted, actual float64) float64 {
	if predicted == 0 && actual == 0 {
		return 1.0
	}
	if predicted == 0 {
		return 0.0
	}
	ratio := actual / predicted
	if ratio > 1.0 {
		ratio = 1.0 / ratio
	}
	return math.Round(ratio*100) / 100
}

func (s *Service) computeTrend(agentID string, since time.Time, hours time.Duration) string {
	midPoint := since.Add(hours / 2)

	type periodRow struct {
		Correct int
		Total   int
	}

	var first, second periodRow
	s.db.Raw(`
		SELECT COUNT(*) AS total, COUNT(*) FILTER (WHERE score >= 0.5) AS correct
		FROM decision_evaluation
		WHERE agent_id = ? AND created_at >= ? AND created_at < ?
	`, agentID, since, midPoint).Scan(&first)
	s.db.Raw(`
		SELECT COUNT(*) AS total, COUNT(*) FILTER (WHERE score >= 0.5) AS correct
		FROM decision_evaluation
		WHERE agent_id = ? AND created_at >= ?
	`, agentID, midPoint).Scan(&second)

	firstRate := 0.5
	if first.Total > 0 {
		firstRate = float64(first.Correct) / float64(first.Total)
	}
	secondRate := 0.5
	if second.Total > 0 {
		secondRate = float64(second.Correct) / float64(second.Total)
	}

	diff := secondRate - firstRate
	if diff > 0.05 {
		return "improving"
	} else if diff < -0.05 {
		return "declining"
	}
	return "stable"
}

func periodHours(period string) time.Duration {
	switch period {
	case "7d":
		return 7 * 24 * time.Hour
	case "30d":
		return 30 * 24 * time.Hour
	case "90d":
		return 90 * 24 * time.Hour
	default:
		return 0
	}
}
