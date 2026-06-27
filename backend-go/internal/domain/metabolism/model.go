package metabolism

import (
	"time"
)

// ScorableEvent is a normalised event that the scoring engine evaluates.
type ScorableEvent struct {
	ID         int64
	Source     string
	Topic      string
	OpLogCount int
	RefCount   int
	CreatedAt  time.Time
	Status     string
	Priority   int
	Payload    string
}

// ScoreDimensions holds the per-dimension scores for storage and analysis.
type ScoreDimensions struct {
	Impact    float64 `json:"impact"`
	Ref       float64 `json:"ref"`
	Freshness float64 `json:"freshness"`
	Semantic  float64 `json:"semantic"`
}

// MetabolismScore holds the per-dimension and combined scores.
type MetabolismScore struct {
	Combined   float64
	Impact     float64
	Ref        float64
	Freshness  float64
	Semantic   float64
	SemSkipped bool
	Excretable bool
	Reason     string
}

// MetabolismLog persists a scoring result to the database.
type MetabolismLog struct {
	ID             int64     `gorm:"primaryKey;autoIncrement"`
	EventID        int64     `gorm:"not null;index"`
	Source         string    `gorm:"type:varchar(100);not null;index"`
	TotalScore     float64   `gorm:"type:numeric(5,2);not null"`
	ImpactScore    float64   `gorm:"type:numeric(5,2);not null"`
	RefScore       float64   `gorm:"type:numeric(5,2);not null"`
	FreshnessScore float64   `gorm:"type:numeric(5,2);not null"`
	SemanticScore  float64   `gorm:"type:numeric(5,2);not null"`
	SemSkipped     bool      `gorm:"not null;default:false"`
	Dimensions     string    `gorm:"type:jsonb"`
	Excretable     bool      `gorm:"not null;default:false"`
	Reason         string    `gorm:"type:text"`
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
}

// TableName overrides the default table name.
func (MetabolismLog) TableName() string {
	return "metabolism_log"
}

// ---------------------------------------------------------------------------
// Entity-based (M1) excretion types
// ---------------------------------------------------------------------------

// ExcretionTargetType identifies the type of entity being evaluated.
type ExcretionTargetType string

const (
	ExcretionTargetListing ExcretionTargetType = "listing"
	ExcretionTargetAgent   ExcretionTargetType = "agent"
)

// ExcretionItem represents the result of scoring a single entity.
type ExcretionItem struct {
	ID          int64               `json:"id"`
	TargetType  ExcretionTargetType `json:"target_type"`
	TargetID    int64               `json:"target_id"`
	TargetName  string              `json:"target_name"`
	Score       float64             `json:"score"`       // 0-100 combined score
	StaleScore  float64             `json:"stale_score"` // staleness contribution (0-100)
	PerfScore   float64             `json:"perf_score"`  // performance contribution (0-100)
	Action      string              `json:"action"`      // "keep" | "flag" | "excrete"
	Reason      string              `json:"reason"`
	EvaluatedAt time.Time           `json:"evaluated_at"`
}

// M1ExcretionResult summarizes a full M1 execution run.
type M1ExcretionResult struct {
	TotalItems  int             `json:"total_items"`
	Excreted    int             `json:"excreted"`
	Flagged     int             `json:"flagged"`
	DryRun      bool            `json:"dry_run"`
	StartedAt   time.Time       `json:"started_at"`
	CompletedAt time.Time       `json:"completed_at"`
	Items       []ExcretionItem `json:"items"`
}

// M1Config holds thresholds and weights for the M1 entity excretion scoring.
type M1Config struct {
	// StaleDays is the number of days of inactivity that scores as fully stale (100).
	StaleDays int `json:"stale_days"`
	// FlagThreshold: items with score below this are flagged for review.
	FlagThreshold float64 `json:"flag_threshold"`
	// ExcreteThreshold: items with score below this are automatically excreted.
	ExcreteThreshold float64 `json:"excrete_threshold"`
	// StaleWeight is the weight for the staleness component (0-1).
	StaleWeight float64 `json:"stale_weight"`
	// PerfWeight is the weight for the performance component (0-1).
	PerfWeight float64 `json:"perf_weight"`
}

// DefaultM1Config returns the default configuration for M1 excretion scoring.
func DefaultM1Config() M1Config {
	return M1Config{
		StaleDays:         90,
		FlagThreshold:     40,
		ExcreteThreshold:  20,
		StaleWeight:       0.6,
		PerfWeight:        0.4,
	}
}
