package imagegen

import (
	"encoding/json"
	"time"
)

// ProductImageGen maps to the `product_image_gen` table.
type ProductImageGen struct {
	ID              int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID       int64           `gorm:"column:product_id;not null" json:"product_id"`
	Prompt          string          `gorm:"column:prompt;size:2000;not null" json:"prompt"`
	Style           string          `gorm:"column:style;size:50;default:product_white" json:"style"`
	NegativePrompt  string          `gorm:"column:negative_prompt;size:1000;default:''" json:"negative_prompt"`
	Size            string          `gorm:"column:size;size:20;default:1024x1024" json:"size"`
	RequestedCount  int             `gorm:"column:requested_count;default:1" json:"requested_count"`
	Status          string          `gorm:"column:status;size:20;default:pending" json:"status"`
	ImageURLs       json.RawMessage `gorm:"column:image_urls;type:jsonb" json:"image_urls"`
	ErrorMessage    string          `gorm:"column:error_message;size:1000" json:"error_message"`
	CreatedBy       int64           `gorm:"column:created_by" json:"created_by"`
	BatchID         string          `gorm:"column:batch_id;size:36" json:"batch_id"`
	CreatedAt       time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (ProductImageGen) TableName() string {
	return "product_image_gen"
}

// ProductCanvas maps to the `product_canvases` table.
type ProductCanvas struct {
	ID         int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProductID  int64           `gorm:"column:product_id;not null" json:"product_id"`
	Name       string          `gorm:"column:name;size:200;default:未命名画布" json:"name"`
	Layers     json.RawMessage `gorm:"column:layers;type:jsonb" json:"layers"`
	Thumbnail  string          `gorm:"column:thumbnail;type:text" json:"thumbnail"`
	CreatedBy  int64           `gorm:"column:created_by" json:"created_by"`
	CreatedAt  time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (ProductCanvas) TableName() string {
	return "product_canvases"
}

// PromptTemplate maps to the `prompt_template` table.
type PromptTemplate struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name           string    `gorm:"column:name;size:200;not null" json:"name"`
	Description    string    `gorm:"column:description;size:500" json:"description"`
	Prompt         string    `gorm:"column:prompt;size:2000;not null" json:"prompt"`
	NegativePrompt string    `gorm:"column:negative_prompt;size:1000;default:''" json:"negative_prompt"`
	Style          string    `gorm:"column:style;size:50;default:product_white" json:"style"`
	Size           string    `gorm:"column:size;size:20;default:1024x1024" json:"size"`
	PlatformCode   string    `gorm:"column:platform_code;size:50" json:"platform_code"`
	IsShared       int       `gorm:"column:is_shared;smallint;default:1" json:"is_shared"`
	UsageCount     int       `gorm:"column:usage_count;default:0" json:"usage_count"`
	CreatedBy      int64     `gorm:"column:created_by" json:"created_by"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (PromptTemplate) TableName() string {
	return "prompt_template"
}
