package integrations

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// ── AES-GCM token encryption ──
// Tokens are transparently encrypted at rest using AES-256-GCM before
// persisting to the database, and decrypted on read. Configure the
// encryption key via the PLATFORM_TOKEN_ENCRYPTION_KEY env var
// (32-byte key, base64-encoded). In development, a static fallback key
// is used (NOT production-safe — always set PLATFORM_TOKEN_ENCRYPTION_KEY
// in production).

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

// ── GORM hooks for transparent token encryption ──

// BeforeCreate encrypts tokens before insert.
func (a *PlatformIntegrationAccount) BeforeCreate(tx *gorm.DB) error {
	return a.encryptTokens()
}

// BeforeUpdate encrypts tokens before update (if they changed).
func (a *PlatformIntegrationAccount) BeforeUpdate(tx *gorm.DB) error {
	return a.encryptTokens()
}

// AfterFind decrypts tokens after read.
func (a *PlatformIntegrationAccount) AfterFind(tx *gorm.DB) error {
	return a.decryptTokens()
}

func (a *PlatformIntegrationAccount) encryptTokens() error {
	if a.AccessToken != "" && !isEncrypted(a.AccessToken) {
		enc, err := encrypt(a.AccessToken)
		if err != nil {
			return err
		}
		a.AccessToken = enc
	}
	if a.RefreshToken != "" && !isEncrypted(a.RefreshToken) {
		enc, err := encrypt(a.RefreshToken)
		if err != nil {
			return err
		}
		a.RefreshToken = enc
	}
	return nil
}

func (a *PlatformIntegrationAccount) decryptTokens() error {
	if a.AccessToken != "" && isEncrypted(a.AccessToken) {
		dec, err := decrypt(a.AccessToken)
		if err != nil {
			return err
		}
		a.AccessToken = dec
	}
	if a.RefreshToken != "" && isEncrypted(a.RefreshToken) {
		dec, err := decrypt(a.RefreshToken)
		if err != nil {
			return err
		}
		a.RefreshToken = dec
	}
	return nil
}

// isEncrypted checks if a token value is base64-encoded ciphertext
// by verifying it's valid base64 and at least the minimum ciphertext length.
func isEncrypted(s string) bool {
	if len(s) < 48 { // nonce(12) + min ciphertext + tag(16) in base64
		return false
	}
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

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
