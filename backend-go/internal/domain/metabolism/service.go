package metabolism

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	// GrayZoneLower is the lower bound (inclusive) of the gray zone that
	// triggers LLM-based semantic scoring.
	GrayZoneLower = 0.40

	// GrayZoneUpper is the upper bound (exclusive) of the gray zone.
	GrayZoneUpper = 0.75

	// ExcretionThreshold is the combined score at or above which an event
	// is considered excretable.
	ExcretionThreshold = 0.70

	// DefaultTTL is the time after which an event is considered fully stale
	// (freshness = 1.0).
	DefaultTTL = 7 * 24 * time.Hour // 7 days
)

// Scoring weights used when semantic scoring is NOT active.
// When semantic is active, the semantic portion adds to the combined score.
const (
	WImpactNoSem = 0.40
	WRefNoSem    = 0.30
	WFreshNoSem  = 0.30
)

// SemanticBlendWeight is the weight given to the semantic score when it is
// blended into the final combined score.
const SemanticBlendWeight = 0.25

// ---------------------------------------------------------------------------
// Pure scoring functions
// ---------------------------------------------------------------------------

// ImpactScore computes the impact dimension score based on the number of
// related operation_log records.
func ImpactScore(count int) float64 {
	if count <= 0 {
		return 0.0
	}
	if count >= 5 {
		return 1.0
	}
	return 0.3 + (float64(count-1)/3.0)*0.7
}

// ReferenceScore computes the reference dimension score based on the number
// of active agent references.
func ReferenceScore(count int) float64 {
	if count <= 0 {
		return 0.0
	}
	if count >= 3 {
		return 1.0
	}
	if count == 1 {
		return 0.3
	}
	return 0.65
}

// FreshnessScore computes the freshness dimension score.
// Returns 0.0 for just-created events, 1.0 for events past the DefaultTTL.
func FreshnessScore(createdAt, now time.Time) float64 {
	elapsed := now.Sub(createdAt)
	if elapsed <= 0 {
		return 0.0
	}
	ratio := float64(elapsed) / float64(DefaultTTL)
	if ratio >= 1.0 {
		return 1.0
	}
	return ratio
}

// clamp01 clamps a float64 value to the [0, 1] range.
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// ---------------------------------------------------------------------------
// MetabolismService
// ---------------------------------------------------------------------------

// MetabolismService handles scoring and excretion lifecycle.
type MetabolismService struct {
	adapter        ScoringAdapter
	semanticScorer SemanticScorer
	db             *gorm.DB
	logger         *zap.Logger
}

// NewService creates a new MetabolismService.
func NewService(db *gorm.DB, logger *zap.Logger, adapter ScoringAdapter, scorer SemanticScorer) *MetabolismService {
	return &MetabolismService{
		adapter:        adapter,
		semanticScorer: scorer,
		db:             db,
		logger:         logger,
	}
}

// scoreAt is the pure scoring engine. It evaluates a ScorableEvent and returns
// a MetabolismScore with computed dimensions, combined score, and excretability.
func (s *MetabolismService) scoreAt(ev ScorableEvent, now time.Time) MetabolismScore {
	if s.logger == nil {
		s.logger = zap.NewNop()
	}
	if s.semanticScorer == nil {
		s.semanticScorer = &noopSemanticScorer{}
	}

	impact := ImpactScore(ev.OpLogCount)
	ref := ReferenceScore(ev.RefCount)
	fresh := FreshnessScore(ev.CreatedAt, now)

	// Compute the combined score without semantic (for gray zone check).
	combinedNoSem := WImpactNoSem*impact + WRefNoSem*ref + WFreshNoSem*fresh

	score := MetabolismScore{
		Impact:     impact,
		Ref:        ref,
		Freshness:  fresh,
		Semantic:   0,
		SemSkipped: true,
		Combined:   combinedNoSem,
		Reason:     fmt.Sprintf("impact=%.2f ref=%.2f fresh=%.2f combined=%.4f", impact, ref, fresh, combinedNoSem),
	}

	// Check if semantic scoring should be triggered (gray zone).
	if combinedNoSem >= GrayZoneLower && combinedNoSem < GrayZoneUpper {
		semScore, err := s.semanticScorer.Score(ev)
		if err == nil {
			semScore = clamp01(semScore)
			score.Semantic = semScore
			score.SemSkipped = false
			score.Combined = combinedNoSem + SemanticBlendWeight*semScore
			score.Reason = fmt.Sprintf(
				"impact=%.2f ref=%.2f fresh=%.2f sem=%.2f combined=%.4f",
				impact, ref, fresh, semScore, score.Combined,
			)
		}
	}

	score.Excretable = score.Combined >= ExcretionThreshold
	return score
}

// Score evaluates a single event and returns the scoring result.
func (s *MetabolismService) Score(eventID int64, source string, opLogCount, refCount int, createdAt, now time.Time) MetabolismScore {
	ev := ScorableEvent{
		ID:         eventID,
		Source:     source,
		OpLogCount: opLogCount,
		RefCount:   refCount,
		CreatedAt:  createdAt,
	}
	return s.scoreAt(ev, now)
}

// Execute runs the scoring pipeline against the registered adapter.
// If dryRun is true, no actual excretion is performed — only scoring is logged.
func (s *MetabolismService) Execute(dryRun bool) error {
	s.logger.Info("metabolism: M1 Execute starting", zap.Bool("dry_run", dryRun))
	now := time.Now()

	if s.adapter == nil {
		return fmt.Errorf("metabolism: scoring not configured — adapter is nil")
	}

	events, err := s.adapter.ScorableEvents("")
	if err != nil {
		s.logger.Error("metabolism: adapter.ScorableEvents error", zap.Error(err))
		return err
	}

	for _, ev := range events {
		ms := s.scoreAt(ev, now)
		s.logger.Info("metabolism: scored",
			zap.Int64("event_id", ev.ID),
			zap.Float64("combined", ms.Combined),
			zap.Bool("excretable", ms.Excretable),
		)

		// Persist scoring result.
		dims := ScoreDimensions{
			Impact:    ms.Impact,
			Ref:       ms.Ref,
			Freshness: ms.Freshness,
			Semantic:  ms.Semantic,
		}
		dimsJSON, _ := json.Marshal(dims)
		logEntry := MetabolismLog{
			EventID:        ev.ID,
			Source:         ev.Source,
			TotalScore:     ms.Combined,
			ImpactScore:    ms.Impact,
			RefScore:       ms.Ref,
			FreshnessScore: ms.Freshness,
			SemanticScore:  ms.Semantic,
			SemSkipped:     ms.SemSkipped,
			Dimensions:     string(dimsJSON),
			Excretable:     ms.Excretable,
			Reason:         ms.Reason,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := s.db.Create(&logEntry).Error; err != nil {
			s.logger.Error("metabolism: failed to persist score", zap.Error(err))
			continue
		}

		// In non-dry-run mode, mark as excreted.
		if !dryRun && ms.Excretable {
			if err := s.adapter.MarkExcreted(ev.ID, ms.Reason); err != nil {
				s.logger.Error("metabolism: mark excreted failed", zap.Error(err))
			} else {
				s.logger.Info("metabolism: event marked excreted",
					zap.Int64("event_id", ev.ID))
			}
		}
	}

	s.logger.Info("metabolism: M1 Execute completed")
	return nil
}

// ---------------------------------------------------------------------------
// noopSemanticScorer — fallback when no real scorer is configured
// ---------------------------------------------------------------------------

type noopSemanticScorer struct{}

func (n *noopSemanticScorer) Score(_ ScorableEvent) (float64, error) {
	return 0, nil
}

// ---------------------------------------------------------------------------
// ListLogs and GetLog — query helpers
// ---------------------------------------------------------------------------

// ListLogs returns paginated metabolism log entries.
func (s *MetabolismService) ListLogs(page, pageSize int) ([]MetabolismLog, int64, error) {
	var logs []MetabolismLog
	var total int64

	query := s.db.Model(&MetabolismLog{})
	query.Count(&total)

	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// GetLog returns a single metabolism log entry by ID.
func (s *MetabolismService) GetLog(id int64) (*MetabolismLog, error) {
	var log MetabolismLog
	if err := s.db.First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// ---------------------------------------------------------------------------
// Entity-based M1 excretion scoring (listings, agents)
// ---------------------------------------------------------------------------

// ScoreStaleness computes a staleness score (0-100) based on days since last
// activity. 0 = very active (just updated), 100 = fully stale (>StaleDays).
func ScoreStaleness(daysSinceActivity int) float64 {
	if daysSinceActivity <= 0 {
		return 0
	}
	cfg := DefaultM1Config()
	ratio := float64(daysSinceActivity) / float64(cfg.StaleDays)
	if ratio >= 1.0 {
		return 100.0
	}
	// Logarithmic scaling: early days decay steeply, then flatten.
	// score = 100 * (1 - e^(-2 * ratio))
	return 100.0 * (1.0 - math.Exp(-2.0*ratio))
}

// ScorePerformance computes a performance score (0-100) based on sales velocity
// and profit margin. Higher values mean better performance.
//   - salesVelocity: units sold per day (0 = no sales)
//   - profitMargin: profit margin as a fraction (0.0 = breakeven, 0.3 = 30%)
func ScorePerformance(salesVelocity, profitMargin float64) float64 {
	// Velocity component: 0 at 0 sales, 80 at 10+ units/day, logarithmic in between.
	velScore := 0.0
	if salesVelocity > 0 {
		velRatio := salesVelocity / 10.0
		if velRatio >= 1.0 {
			velScore = 80.0
		} else {
			velScore = 80.0 * (1.0 - math.Exp(-3.0*velRatio))
		}
	}

	// Margin component: 0 for <=0 margin, up to 20 for 30%+ margin.
	marginScore := 0.0
	if profitMargin > 0 {
		marginRatio := profitMargin / 0.30
		if marginRatio >= 1.0 {
			marginScore = 20.0
		} else {
			marginScore = 20.0 * marginRatio
		}
	}

	return velScore + marginScore
}

// ScoreEntity combines staleness and performance into a single health score (0-100).
// A high score means the entity is healthy and should be kept.
// A low score means the entity is a candidate for flagging or excretion.
//   - staleScore: 0 = fresh, 100 = fully stale (contributor to low health)
//   - perfScore: 0 = no sales, 100 = high sales/good margin (contributor to high health)
func (s *MetabolismService) ScoreEntity(staleScore, perfScore float64) float64 {
	cfg := DefaultM1Config()
	total := cfg.StaleWeight*(100.0-staleScore) + cfg.PerfWeight*perfScore
	if total < 0 {
		return 0
	}
	if total > 100 {
		return 100
	}
	return total
}

// classifyAction determines what action to take based on the entity score.
func classifyAction(score float64) string {
	cfg := DefaultM1Config()
	if score < cfg.ExcreteThreshold {
		return "excrete"
	}
	if score < cfg.FlagThreshold {
		return "flag"
	}
	return "keep"
}

// listingRow is a minimal projection of product_listing for M1 scoring.
type listingRow struct {
	ID         int64
	ProductID  int64
	Status     string
	LastSyncAt *time.Time
	UpdatedAt  time.Time
	CreatedAt  time.Time
}

func (listingRow) TableName() string { return "product_listing" }

// ScoreAndExcreteEntities runs the full M1 entity excretion pipeline against
// listings. It scores each listing, flags/excretes as appropriate, and returns
// the summary result. In dry-run mode, no actual mutations are performed.
func (s *MetabolismService) ScoreAndExcreteEntities(dryRun bool) (*M1ExcretionResult, error) {
	s.logger.Info("metabolism: M1 entity excretion starting", zap.Bool("dry_run", dryRun))
	now := time.Now()
	result := &M1ExcretionResult{
		DryRun:    dryRun,
		StartedAt: now,
	}

	// 1. Score listings
	listingItems := s.scoreListings(now)

	// 2. Score agents
	agentItems := s.scoreAgents(now)

	allItems := append(listingItems, agentItems...)
	result.Items = allItems
	result.TotalItems = len(allItems)

	// 3. Execute actions (unless dry run)
	if !dryRun {
		for i, item := range allItems {
			switch item.Action {
			case "excrete":
				if err := s.excreteEntity(item.TargetType, item.TargetID, item.Reason); err != nil {
					s.logger.Error("metabolism: excrete failed",
						zap.String("target_type", string(item.TargetType)),
						zap.Int64("target_id", item.TargetID),
						zap.Error(err),
					)
				} else {
					allItems[i].Action = "excreted" // confirmed
					result.Excreted++
				}
			case "flag":
				result.Flagged++
			}
		}
	} else {
		for _, item := range allItems {
			switch item.Action {
			case "excrete":
				result.Excreted++
			case "flag":
				result.Flagged++
			}
		}
	}

	result.CompletedAt = time.Now()
	s.logger.Info("metabolism: M1 entity excretion completed",
		zap.Int("total", result.TotalItems),
		zap.Int("excreted", result.Excreted),
		zap.Int("flagged", result.Flagged),
		zap.Bool("dry_run", dryRun),
	)
	return result, nil
}

// scoreListings evaluates all active listings and returns ExcretionItems.
func (s *MetabolismService) scoreListings(now time.Time) []ExcretionItem {
	var rows []listingRow
	if err := s.db.Where("status NOT IN ('archived', 'deleted')").Find(&rows).Error; err != nil {
		s.logger.Error("metabolism: query listings failed", zap.Error(err))
		return nil
	}

	items := make([]ExcretionItem, 0, len(rows))
	for _, row := range rows {
		// Determine last activity time.
		lastActive := row.UpdatedAt
		if row.LastSyncAt != nil && row.LastSyncAt.After(lastActive) {
			lastActive = *row.LastSyncAt
		}

		daysSinceActivity := int(now.Sub(lastActive).Hours() / 24.0)
		if daysSinceActivity < 0 {
			daysSinceActivity = 0
		}

		staleScore := ScoreStaleness(daysSinceActivity)

		// Performance: query sales velocity and profit margin for this listing's product.
		salesVel := s.querySalesVelocity(row.ProductID)
		profitMargin := s.queryProfitMargin(row.ProductID)
		perfScore := ScorePerformance(salesVel, profitMargin)

		totalScore := s.ScoreEntity(staleScore, perfScore)
		action := classifyAction(totalScore)

		name := fmt.Sprintf("listing_%d", row.ID)
		if row.Status != "" {
			name = fmt.Sprintf("listing_%d(%s)", row.ID, row.Status)
		}

		reason := fmt.Sprintf("stale=%.1f perf=%.1f score=%.1f days_since_activity=%d vel=%.3f margin=%.2f",
			staleScore, perfScore, totalScore, daysSinceActivity, salesVel, profitMargin)

		items = append(items, ExcretionItem{
			TargetType:  ExcretionTargetListing,
			TargetID:    row.ID,
			TargetName:  name,
			Score:       totalScore,
			StaleScore:  staleScore,
			PerfScore:   perfScore,
			Action:      action,
			Reason:      reason,
			EvaluatedAt: now,
		})
	}
	return items
}

// scoreAgents evaluates agents and returns ExcretionItems.
// Agent evaluation checks how recently a given agent was invoked via ai_trace.
func (s *MetabolismService) scoreAgents(now time.Time) []ExcretionItem {
	// Query the last execution time per agent from ai_trace.
	type agentLastRun struct {
		AgentID   string
		LastRunAt time.Time
	}
	var runs []agentLastRun
	if err := s.db.Raw(`
		SELECT agent_id, MAX(created_at) AS last_run_at
		FROM ai_trace
		GROUP BY agent_id
	`).Scan(&runs).Error; err != nil {
		s.logger.Error("metabolism: query agent traces failed", zap.Error(err))
		return nil
	}

	lastRunByAgent := make(map[string]time.Time, len(runs))
	for _, r := range runs {
		lastRunByAgent[r.AgentID] = r.LastRunAt
	}

	// Known agent IDs from the registry.
	knownAgents := []string{"A1", "A2", "A3", "A4", "A5", "A6", "A7",
		"A8", "A9", "A10", "A11",
		"G0", "G1", "G2", "G3"}

	items := make([]ExcretionItem, 0, len(knownAgents))
	for _, agentID := range knownAgents {
		lastRun, found := lastRunByAgent[agentID]
		if !found {
			// Never run — score as very stale.
			daysSinceActivity := 999
			staleScore := ScoreStaleness(daysSinceActivity)
			totalScore := s.ScoreEntity(staleScore, 0)
			action := classifyAction(totalScore)
			reason := fmt.Sprintf("never_executed stale=%.1f score=%.1f", staleScore, totalScore)
			items = append(items, ExcretionItem{
				TargetType:  ExcretionTargetAgent,
				TargetID:    0,
				TargetName:  fmt.Sprintf("agent_%s(never_run)", agentID),
				Score:       totalScore,
				StaleScore:  staleScore,
				PerfScore:   0,
				Action:      action,
				Reason:      reason,
				EvaluatedAt: now,
			})
			continue
		}

		daysSinceActivity := int(now.Sub(lastRun).Hours() / 24.0)
		if daysSinceActivity < 0 {
			daysSinceActivity = 0
		}

		staleScore := ScoreStaleness(daysSinceActivity)
		// Agents don't have sales velocity or margin; use trace count as a proxy.
		traceCount := s.queryAgentTraceCount(agentID, 30)
		perfScore := ScorePerformance(float64(traceCount)/30.0, 0)
		totalScore := s.ScoreEntity(staleScore, perfScore)
		action := classifyAction(totalScore)

		reason := fmt.Sprintf("stale=%.1f perf=%.1f score=%.1f days_since_run=%d traces_30d=%d",
			staleScore, perfScore, totalScore, daysSinceActivity, traceCount)

		items = append(items, ExcretionItem{
			TargetType:  ExcretionTargetAgent,
			TargetID:    0,
			TargetName:  fmt.Sprintf("agent_%s", agentID),
			Score:       totalScore,
			StaleScore:  staleScore,
			PerfScore:   perfScore,
			Action:      action,
			Reason:      reason,
			EvaluatedAt: now,
		})
	}
	return items
}

// querySalesVelocity returns average daily sales for a product (units/day over last 30 days).
func (s *MetabolismService) querySalesVelocity(productID int64) float64 {
	type result struct {
		TotalQty float64
	}
	var r result
	if err := s.db.Raw(`
		SELECT COALESCE(SUM(oi.quantity), 0) AS total_qty
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		WHERE oi.product_id = ? AND o.created_at >= NOW() - INTERVAL '30 days'
	`, productID).Scan(&r).Error; err != nil {
		return 0
	}
	return r.TotalQty / 30.0
}

// queryProfitMargin returns the profit margin fraction for a product.
func (s *MetabolismService) queryProfitMargin(productID int64) float64 {
	type result struct {
		Margin *float64
	}
	var r result
	// Try product_analysis first, fall back to order level.
	if err := s.db.Raw(`
		SELECT estimated_profit_margin AS margin
		FROM product_analysis
		WHERE sourcing_product_id = (
			SELECT sourcing_product_id FROM products WHERE id = ?
		)
		ORDER BY created_at DESC LIMIT 1
	`, productID).Scan(&r).Error; err != nil || r.Margin == nil {
		// Fallback: compute margin from recent orders.
		return s.queryMarginFromOrders(productID)
	}
	margin := *r.Margin
	if margin < 0 {
		return 0
	}
	return margin / 100.0 // stored as percentage, convert to fraction
}

// queryMarginFromOrders computes profit margin from recent order data.
func (s *MetabolismService) queryMarginFromOrders(productID int64) float64 {
	type result struct {
		TotalRevenue float64
		TotalCost    float64
	}
	var r result
	if err := s.db.Raw(`
		SELECT
			COALESCE(SUM(oi.unit_price * oi.quantity), 0) AS total_revenue,
			COALESCE(SUM(oi.cost_price * oi.quantity), 0) AS total_cost
		FROM order_items oi
		WHERE oi.product_id = ? AND oi.cost_price > 0
	`, productID).Scan(&r).Error; err != nil || r.TotalRevenue <= 0 {
		return 0
	}
	if r.TotalRevenue <= 0 {
		return 0
	}
	margin := (r.TotalRevenue - r.TotalCost) / r.TotalRevenue
	if margin < 0 {
		return 0
	}
	return margin
}

// excreteEntity archives or deactivates the target entity.
func (s *MetabolismService) excreteEntity(targetType ExcretionTargetType, targetID int64, reason string) error {
	switch targetType {
	case ExcretionTargetListing:
		return s.db.Model(&listingRow{ID: targetID}).
			Where("id = ?", targetID).
			Update("status", "archived").Error
	case ExcretionTargetAgent:
		// Log the excretion reason for this agent; we don't deactivate agents.
		s.logger.Warn("metabolism: agent excretion proposed",
			zap.String("agent_id", fmt.Sprintf("agent_%d", targetID)),
			zap.String("reason", reason),
		)
		return nil
	default:
		return fmt.Errorf("metabolism: unknown target type %s", targetType)
	}
}

// queryAgentTraceCount returns the number of AI traces for an agent in the last N days.
func (s *MetabolismService) queryAgentTraceCount(agentID string, days int) int {
	type result struct {
		Count int
	}
	var r result
	s.db.Raw(`
		SELECT COUNT(*) AS count
		FROM ai_trace
		WHERE agent_id = ? AND created_at >= NOW() - CAST(? AS INTEGER) * INTERVAL '1 day'
	`, agentID, days).Scan(&r)
	return r.Count
}
