package support

import (
	"encoding/json"
	"time"
)

// CustomerConversation maps to the "customer_conversations" table.
type CustomerConversation struct {
	ID            int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OrderID       *int64     `gorm:"column:order_id" json:"order_id,omitempty"`
	Platform      string     `gorm:"column:platform;not null" json:"platform"`
	CustomerName  string     `gorm:"column:customer_name;not null" json:"customer_name"`
	CustomerEmail string     `gorm:"column:customer_email;not null" json:"customer_email"`
	Subject       string     `gorm:"column:subject;not null" json:"subject"`
	Status        string     `gorm:"column:status;not null;default:open" json:"status"`
	Priority      string     `gorm:"column:priority;not null;default:medium" json:"priority"`
	AssignedTo    *string    `gorm:"column:assigned_to" json:"assigned_to,omitempty"`
	LastMessageAt *time.Time `gorm:"column:last_message_at" json:"last_message_at,omitempty"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	// Relations (not persisted)
	Messages []ChatMessage `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
}

// TableName explicitly sets the table name.
func (CustomerConversation) TableName() string { return "customer_conversations" }

// ChatMessage maps to the "chat_messages" table.
type ChatMessage struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ConversationID int64     `gorm:"column:conversation_id;not null;index" json:"conversation_id"`
	SenderType     string    `gorm:"column:sender_type;not null" json:"sender_type"`
	Content        string    `gorm:"column:content;not null" json:"content"`
	AutoReplied    bool      `gorm:"column:auto_replied;default:false" json:"auto_replied"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName explicitly sets the table name.
func (ChatMessage) TableName() string { return "chat_messages" }

// AutoReplyTemplate maps to the "auto_reply_templates" table.
type AutoReplyTemplate struct {
	ID        int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name      string          `gorm:"column:name;not null" json:"name"`
	Category  string          `gorm:"column:category;not null" json:"category"`
	Content   string          `gorm:"column:content;not null" json:"content"`
	Variables json.RawMessage `gorm:"column:variables;type:jsonb" json:"variables,omitempty"`
	Platform  string          `gorm:"column:platform" json:"platform"`
	Enabled   bool            `gorm:"column:enabled;default:true" json:"enabled"`
	CreatedAt time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName explicitly sets the table name.
func (AutoReplyTemplate) TableName() string { return "auto_reply_templates" }

// BlacklistEntry maps to the "blacklist_entries" table.
type BlacklistEntry struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CustomerEmail string    `gorm:"column:customer_email;not null;index" json:"customer_email"`
	CustomerName  string    `gorm:"column:customer_name" json:"customer_name"`
	Reason        string    `gorm:"column:reason" json:"reason"`
	AddedBy       string    `gorm:"column:added_by" json:"added_by"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName explicitly sets the table name.
func (BlacklistEntry) TableName() string { return "blacklist_entries" }

// ---------- Input structs ----------

// CreateConversationInput is the payload for creating a conversation.
type CreateConversationInput struct {
	OrderID       *int64  `json:"order_id"`
	Platform      string  `json:"platform" binding:"required"`
	CustomerName  string  `json:"customer_name" binding:"required"`
	CustomerEmail string  `json:"customer_email" binding:"required"`
	Subject       string  `json:"subject" binding:"required"`
	Status        string  `json:"status"`
	Priority      string  `json:"priority"`
	AssignedTo    *string `json:"assigned_to"`
}

// UpdateConversationInput is the payload for updating a conversation.
type UpdateConversationInput struct {
	OrderID       *int64  `json:"order_id"`
	Platform      *string `json:"platform"`
	CustomerName  *string `json:"customer_name"`
	CustomerEmail *string `json:"customer_email"`
	Subject       *string `json:"subject"`
	Status        *string `json:"status"`
	Priority      *string `json:"priority"`
	AssignedTo    *string `json:"assigned_to"`
}

// SendReplyInput is the payload for sending a reply.
type SendReplyInput struct {
	Content string `json:"content" binding:"required"`
	IsAuto  bool   `json:"is_auto"`
}

// CreateTemplateInput is the payload for creating an auto-reply template.
type CreateTemplateInput struct {
	Name      string          `json:"name" binding:"required"`
	Category  string          `json:"category" binding:"required"`
	Content   string          `json:"content" binding:"required"`
	Variables json.RawMessage `json:"variables"`
	Platform  string          `json:"platform"`
	Enabled   *bool           `json:"enabled"`
}

// UpdateTemplateInput is the payload for updating a template.
type UpdateTemplateInput struct {
	Name      *string          `json:"name"`
	Category  *string          `json:"category"`
	Content   *string          `json:"content"`
	Variables *json.RawMessage `json:"variables"`
	Platform  *string          `json:"platform"`
	Enabled   *bool            `json:"enabled"`
}

// CreateBlacklistInput is the payload for adding a blacklist entry.
type CreateBlacklistInput struct {
	CustomerEmail string `json:"customer_email" binding:"required"`
	CustomerName  string `json:"customer_name"`
	Reason        string `json:"reason"`
	AddedBy       string `json:"added_by"`
}

// ConversationFilter holds optional filters for listing conversations.
type ConversationFilter struct {
	Status   string `json:"status" form:"status"`
	Priority string `json:"priority" form:"priority"`
	Platform string `json:"platform" form:"platform"`
	Search   string `json:"search" form:"search"`
}
