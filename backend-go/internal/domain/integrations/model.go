package integrations

import (
	"encoding/json"
	"time"
)

// PlatformIntegrationAccount maps to "platform_integration_account".
type PlatformIntegrationAccount struct {
	ID              int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PlatformID      int64           `gorm:"column:platform_id;not null;index" json:"platform_id"`
	StoreName       string          `gorm:"column:store_name" json:"store_name"`
	AccountID       string          `gorm:"column:account_id" json:"account_id"`
	AccessToken     string          `gorm:"column:access_token" json:"-"`
	RefreshToken    string          `gorm:"column:refresh_token" json:"-"`
	TokenExpiresAt  *time.Time      `gorm:"column:token_expires_at" json:"token_expires_at,omitempty"`
	Status          string          `gorm:"column:status;default:active" json:"status"`
	LastSyncAt      *time.Time      `gorm:"column:last_sync_at" json:"last_sync_at,omitempty"`
	SyncStatus      string          `gorm:"column:sync_status;default:idle" json:"sync_status"`
	LastError       string          `gorm:"column:last_error" json:"last_error"`
	Config          json.RawMessage `gorm:"column:config;type:jsonb" json:"config,omitempty"`
	CreatedAt       time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (PlatformIntegrationAccount) TableName() string { return "platform_integration_account" }

// PlatformCategoryMapping maps to "platform_category_mapping".
type PlatformCategoryMapping struct {
	ID                  int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	AccountID           int64     `gorm:"column:account_id;not null;index" json:"account_id"`
	LocalCategoryID     int64     `gorm:"column:local_category_id;not null" json:"local_category_id"`
	PlatformCategoryID  string    `gorm:"column:platform_category_id" json:"platform_category_id"`
	PlatformCategoryName string   `gorm:"column:platform_category_name" json:"platform_category_name"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (PlatformCategoryMapping) TableName() string { return "platform_category_mapping" }

// PlatformAttributeMapping maps to "platform_attribute_mapping".
type PlatformAttributeMapping struct {
	ID                int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	AccountID         int64     `gorm:"column:account_id;not null;index" json:"account_id"`
	LocalAttrName     string    `gorm:"column:local_attr_name;not null" json:"local_attr_name"`
	PlatformAttrID    string    `gorm:"column:platform_attr_id" json:"platform_attr_id"`
	PlatformAttrName  string    `gorm:"column:platform_attr_name" json:"platform_attr_name"`
	Required          bool      `gorm:"column:required;default:false" json:"required"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (PlatformAttributeMapping) TableName() string { return "platform_attribute_mapping" }

// ---------- Input / DTO structs ----------

// CreateAccountInput is the payload for POST /platform-integrations.
type CreateAccountInput struct {
	PlatformID     int64           `json:"platform_id" binding:"required"`
	StoreName      string          `json:"store_name"`
	AccountID      string          `json:"account_id"`
	AccessToken    string          `json:"access_token"`
	RefreshToken   string          `json:"refresh_token"`
	TokenExpiresAt *time.Time      `json:"token_expires_at"`
	Status         string          `json:"status"`
	Config         json.RawMessage `json:"config"`
}

// UpdateAccountInput allows partial updates.
type UpdateAccountInput struct {
	StoreName      *string          `json:"store_name"`
	AccountID      *string          `json:"account_id"`
	AccessToken    *string          `json:"access_token"`
	RefreshToken   *string          `json:"refresh_token"`
	TokenExpiresAt *time.Time       `json:"token_expires_at"`
	Status         *string          `json:"status"`
	SyncStatus     *string          `json:"sync_status"`
	LastError      *string          `json:"last_error"`
	Config         *json.RawMessage `json:"config"`
}

// AccountListFilter captures query parameters.
type AccountListFilter struct {
	Search     string
	PlatformID *int64
	Status     string
}

// CreateCategoryMappingInput is the payload for POST /platform-integrations/:id/categories.
type CreateCategoryMappingInput struct {
	LocalCategoryID      int64  `json:"local_category_id" binding:"required"`
	PlatformCategoryID   string `json:"platform_category_id"`
	PlatformCategoryName string `json:"platform_category_name"`
}

// CreateAttributeMappingInput is the payload for POST /platform-integrations/:id/attributes.
type CreateAttributeMappingInput struct {
	LocalAttrName    string `json:"local_attr_name" binding:"required"`
	PlatformAttrID   string `json:"platform_attr_id"`
	PlatformAttrName string `json:"platform_attr_name"`
	Required         bool   `json:"required"`
}

// TestConnectionResult is the response for POST /platform-integrations/:id/test.
type TestConnectionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
