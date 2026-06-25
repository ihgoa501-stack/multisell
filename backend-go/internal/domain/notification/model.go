package notification

import (
	"encoding/json"
	"time"
)

// Notification maps to the `notification` table.
type Notification struct {
	ID        int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID    int64           `gorm:"column:user_id;not null;index" json:"user_id"`
	AlertType string          `gorm:"column:alert_type;size:50;not null" json:"alert_type"`
	Title     string          `gorm:"column:title;size:200;not null" json:"title"`
	Content   string          `gorm:"column:content;type:text" json:"content"`
	LinkURL   string          `gorm:"column:link_url;size:500" json:"link_url"`
	Severity  string          `gorm:"column:severity;size:20;default:info" json:"severity"`
	IsRead    int             `gorm:"column:is_read;smallint;default:0" json:"is_read"`
	SourceID  string          `gorm:"column:source_id;size:100" json:"source_id"`
	CreatedAt time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the default table name.
func (Notification) TableName() string {
	return "notification"
}

// AlertRule maps to the `alert_rule` table.
type AlertRule struct {
	ID          int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name        string          `gorm:"column:name;size:200;not null" json:"name"`
	AlertType   string          `gorm:"column:alert_type;size:50;not null;uniqueIndex" json:"alert_type"`
	Enabled     int             `gorm:"column:enabled;smallint;default:1" json:"enabled"`
	Config      json.RawMessage `gorm:"column:config;type:jsonb" json:"config"`
	Description string          `gorm:"column:description;size:500" json:"description"`
	CreatedAt   time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (AlertRule) TableName() string {
	return "alert_rule"
}
