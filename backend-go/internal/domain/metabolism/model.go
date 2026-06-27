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
