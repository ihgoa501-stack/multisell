package platform

import (
	"encoding/json"
	"time"
)

// Platform maps to the "platform" table — e-commerce platform configuration.
type Platform struct {
	ID          int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name        string          `gorm:"column:name;not null" json:"name"`
	Code        string          `gorm:"column:code;not null;uniqueIndex" json:"code"`
	APIBaseURL  string          `gorm:"column:api_base_url" json:"api_base_url"`
	APIKey      string          `gorm:"column:api_key" json:"api_key,omitempty"`
	ClientID    string          `gorm:"column:client_id" json:"client_id"`
	ExtraConfig json.RawMessage `gorm:"column:extra_config;type:jsonb" json:"extra_config,omitempty"`
	Status      int16           `gorm:"column:status;default:1" json:"status"`
	SortOrder   int             `gorm:"column:sort_order;default:0" json:"sort_order"`
	CreatedAt   time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName explicitly sets the table name (no AutoMigrate).
func (Platform) TableName() string { return "platform" }

// Store maps to the "stores" table.
type Store struct {
	ID               int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID           int64     `gorm:"column:user_id;not null" json:"user_id"`
	Name             string    `gorm:"column:name;not null" json:"name"`
	PlatformID       *int64    `gorm:"column:platform_id" json:"platform_id,omitempty"`
	PlatformAccountID string   `gorm:"column:platform_account_id" json:"platform_account_id"`
	Status           int16     `gorm:"column:status;default:1" json:"status"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName explicitly sets the table name.
func (Store) TableName() string { return "stores" }

// CreatePlatformInput is the payload for creating a platform.
type CreatePlatformInput struct {
	Name        string          `json:"name" binding:"required"`
	Code        string          `json:"code" binding:"required"`
	APIBaseURL  string          `json:"api_base_url"`
	APIKey      string          `json:"api_key"`
	ClientID    string          `json:"client_id"`
	ExtraConfig json.RawMessage `json:"extra_config"`
	Status      *int16          `json:"status"`
	SortOrder   *int            `json:"sort_order"`
}

// UpdatePlatformInput is the payload for updating a platform.
type UpdatePlatformInput struct {
	Name        *string          `json:"name"`
	Code        *string          `json:"code"`
	APIBaseURL  *string          `json:"api_base_url"`
	APIKey      *string          `json:"api_key"`
	ClientID    *string          `json:"client_id"`
	ExtraConfig *json.RawMessage `json:"extra_config"`
	Status      *int16           `json:"status"`
	SortOrder   *int             `json:"sort_order"`
}

// CreateStoreInput is the payload for creating a store.
type CreateStoreInput struct {
	UserID            int64  `json:"user_id" binding:"required"`
	Name              string `json:"name" binding:"required"`
	PlatformID        *int64 `json:"platform_id"`
	PlatformAccountID string `json:"platform_account_id"`
	Status            *int16 `json:"status"`
}

// UpdateStoreInput is the payload for updating a store.
type UpdateStoreInput struct {
	UserID            *int64  `json:"user_id"`
	Name              *string `json:"name"`
	PlatformID        *int64  `json:"platform_id"`
	PlatformAccountID *string `json:"platform_account_id"`
	Status            *int16  `json:"status"`
}
