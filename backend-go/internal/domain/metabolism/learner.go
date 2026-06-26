package metabolism

import (
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Persistence models
// ---------------------------------------------------------------------------

// LearnerRecord stores each scoring decision so the learner can analyze
// excretion accuracy and adjust parameters over time.
type LearnerRecord struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	EventID      int64     `gorm:"not null;index"`
	Source       string    `gorm:"type:varchar(100);not null;index"`
	Score        float64   `gorm:"type:numeric(5,2);not null"`
	Excreted     bool      `gorm:"not null"`
	WasRecovered bool      `gorm:"not null;default:false"`
	CreatedAt    time.Time `gorm:"not null"`
}

// TableName overrides the default table name for LearnerRecord.
func (LearnerRecord) TableName() string {
	return "learner_record"
}

// LearningWeight stores a single tunable parameter keyed by name.
type LearningWeight struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	Key       string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	Value     float64   `gorm:"type:numeric(6,4);not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

// TableName overrides the default table name for LearningWeight.
func (LearningWeight) TableName() string {
	return "learning_weight"
}

// ---------------------------------------------------------------------------
// Weight keys
// ---------------------------------------------------------------------------

const (
	// WeightKeyThreshold is the excretion threshold that the learner tunes.
	WeightKeyThreshold = "excretion_threshold"
	// WeightKeyErrorRate is the cached error_rate (recovery_requests / excretions).
	WeightKeyErrorRate = "error_rate"
	// WeightKeySampleCount is the number of excretions used in the last analysis.
	WeightKeySampleCount = "sample_count"
	// WeightKeyConfidence is a heuristic confidence level (1.0 - error_rate).
	WeightKeyConfidence = "confidence"
)

const (
	// MinThreshold is the floor for the excretion threshold.
	MinThreshold = 0.50
	// MaxThreshold is the ceiling for the excretion threshold.
	MaxThreshold = 0.85
	// DefaultThreshold is the initial value.
	DefaultThreshold = 0.70
	// ThresholdStep is the amount added or removed per TuneWeights cycle.
	ThresholdStep = 0.02
	// TuneWindow is the number of recent excretions to examine.
	TuneWindow = 100
	// MaxErrorRate is the upper bound for error_rate that triggers a decrease.
	MaxErrorRate = 0.05
	// MinErrorRate is the lower bound for error_rate that triggers an increase.
	MinErrorRate = 0.01
)

// ---------------------------------------------------------------------------
// Learner
// ---------------------------------------------------------------------------

// Learner adaptively tunes metabolism scoring parameters using Bayesian-style
// updates based on observed recovery_requests / excretions.
type Learner struct {
	db     *gorm.DB
	logger *zap.Logger
	mu     sync.RWMutex
	// In-memory cache of current weight values keyed by WeightKey*.
	metrics map[string]float64
}

// NewLearner creates a new Learner and loads persisted weights from the DB.
func NewLearner(db *gorm.DB, logger *zap.Logger) *Learner {
	l := &Learner{
		db:      db,
		logger:  logger.Named("learner"),
		metrics: make(map[string]float64),
	}
	l.loadWeights()
	return l
}

// loadWeights reads all persisted LearningWeight rows into the in-memory map.
func (l *Learner) loadWeights() {
	var rows []LearningWeight
	if err := l.db.Find(&rows).Error; err != nil {
		l.logger.Warn("failed to load learner weights", zap.Error(err))
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, r := range rows {
		l.metrics[r.Key] = r.Value
	}
	// Ensure default threshold exists.
	if _, ok := l.metrics[WeightKeyThreshold]; !ok {
		l.metrics[WeightKeyThreshold] = DefaultThreshold
	}
}

// persistWeight writes a single weight to both the DB and the in-memory map.
func (l *Learner) persistWeight(key string, value float64) {
	l.mu.Lock()
	l.metrics[key] = value
	l.mu.Unlock()

	err := l.db.Exec(
		`INSERT INTO learning_weight (key, value, updated_at)
		 VALUES (?, ?, NOW())
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
		key, value,
	).Error
	if err != nil {
		l.logger.Warn("failed to persist learner weight",
			zap.String("key", key),
			zap.Float64("value", value),
			zap.Error(err),
		)
	}
}

// RecordDecision logs a single scoring decision for future analysis.
func (l *Learner) RecordDecision(eventID int64, source string, score float64, excreted bool, wasRecovered bool) {
	rec := LearnerRecord{
		EventID:      eventID,
		Source:       source,
		Score:        score,
		Excreted:     excreted,
		WasRecovered: wasRecovered,
		CreatedAt:    time.Now(),
	}
	if err := l.db.Create(&rec).Error; err != nil {
		l.logger.Warn("failed to record learner decision",
			zap.Int64("event_id", eventID),
			zap.Error(err),
		)
	}
}

// TuneWeights analyzes recent decisions and adjusts the excretion threshold
// using a Bayesian-inspired update:
//   - Count recovery_requests / total_excretions in the last TuneWindow records
//   - error_rate >  5% → decrease threshold by ThresholdStep (min MinThreshold)
//   - error_rate <  1% → increase threshold by ThresholdStep (max MaxThreshold)
//   - Otherwise → leave unchanged
func (l *Learner) TuneWeights() {
	l.logger.Info("learner: TuneWeights starting")

	type countResult struct {
		TotalExcretions int64
		TotalRecovered  int64
	}
	var cr countResult
	if err := l.db.Raw(`
		SELECT
			COUNT(*) FILTER (WHERE excreted = true) AS total_excretions,
			COUNT(*) FILTER (WHERE excreted = true AND was_recovered = true) AS total_recovered
		FROM learner_record
		WHERE created_at >= NOW() - INTERVAL '7 days'
	`).Scan(&cr).Error; err != nil {
		l.logger.Warn("learner: failed to query records", zap.Error(err))
		return
	}

	if cr.TotalExcretions < int64(TuneWindow) {
		l.logger.Info("learner: insufficient data for tuning",
			zap.Int64("excretions", cr.TotalExcretions),
			zap.Int("required", TuneWindow),
		)
		return
	}

	errorRate := float64(cr.TotalRecovered) / float64(cr.TotalExcretions)

	l.logger.Info("learner: analysis",
		zap.Int64("excretions", cr.TotalExcretions),
		zap.Int64("recovered", cr.TotalRecovered),
		zap.Float64("error_rate", errorRate),
	)

	// Update persisted error rate and sample count for reporting.
	l.persistWeight(WeightKeyErrorRate, errorRate)
	l.persistWeight(WeightKeySampleCount, float64(cr.TotalExcretions))
	l.persistWeight(WeightKeyConfidence, 1.0-errorRate)

	// Load current threshold.
	l.mu.RLock()
	currentThreshold := l.metrics[WeightKeyThreshold]
	l.mu.RUnlock()

	newThreshold := currentThreshold
	if errorRate > MaxErrorRate {
		newThreshold = currentThreshold - ThresholdStep
		if newThreshold < MinThreshold {
			newThreshold = MinThreshold
		}
		l.logger.Info("learner: decreasing threshold",
			zap.Float64("from", currentThreshold),
			zap.Float64("to", newThreshold),
		)
	} else if errorRate < MinErrorRate {
		newThreshold = currentThreshold + ThresholdStep
		if newThreshold > MaxThreshold {
			newThreshold = MaxThreshold
		}
		l.logger.Info("learner: increasing threshold",
			zap.Float64("from", currentThreshold),
			zap.Float64("to", newThreshold),
		)
	} else {
		l.logger.Info("learner: error rate within bounds, threshold unchanged",
			zap.Float64("threshold", currentThreshold),
		)
	}

	l.persistWeight(WeightKeyThreshold, newThreshold)
	l.logger.Info("learner: TuneWeights completed")
}

// GetWeightReport returns the current weights and their confidence as a
// flat map suitable for API responses.
func (l *Learner) GetWeightReport() map[string]float64 {
	l.mu.RLock()
	defer l.mu.RUnlock()

	report := make(map[string]float64, len(l.metrics))
	for k, v := range l.metrics {
		report[k] = v
	}
	// Ensure defaults.
	if _, ok := report[WeightKeyThreshold]; !ok {
		report[WeightKeyThreshold] = DefaultThreshold
	}
	if _, ok := report[WeightKeyConfidence]; !ok {
		report[WeightKeyConfidence] = 0
	}
	if _, ok := report[WeightKeyErrorRate]; !ok {
		report[WeightKeyErrorRate] = 0
	}
	return report
}

// CurrentThreshold returns the current excretion threshold value.
func (l *Learner) CurrentThreshold() float64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if v, ok := l.metrics[WeightKeyThreshold]; ok {
		return v
	}
	return DefaultThreshold
}
