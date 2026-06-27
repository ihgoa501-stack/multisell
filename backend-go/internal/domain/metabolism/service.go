package metabolism

import (
	"encoding/json"
	"fmt"
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
		s.logger.Warn("metabolism: no adapter registered, Execute is a no-op")
		return nil
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
