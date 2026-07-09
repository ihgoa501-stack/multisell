package integrations

import (
	"encoding/json"
	"time"
)

// RawEvent stores a raw event from any platform before AI mapping.
type RawEvent struct {
	ID            int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PlatformCode  string          `gorm:"column:platform_code;not null;index" json:"platform_code"`
	EventType     string          `gorm:"column:event_type;not null" json:"event_type"`
	RawPayload    json.RawMessage `gorm:"column:raw_payload;type:jsonb" json:"raw_payload"`
	MappingStatus string          `gorm:"column:mapping_status;default:pending" json:"mapping_status"`
	MappedResult  json.RawMessage `gorm:"column:mapped_result;type:jsonb" json:"mapped_result,omitempty"`
	Confidence    float64         `gorm:"column:confidence;default:0" json:"confidence"`
	MappedAt      *time.Time      `gorm:"column:mapped_at" json:"mapped_at,omitempty"`
	ErrorMessage  string          `gorm:"column:error_message" json:"error_message,omitempty"`
	CreatedAt     time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (RawEvent) TableName() string { return "platform_raw_event" }
