package support

import (
	"errors"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides customer support business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new support service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// ---------- Conversations ----------

// ListConversations returns paginated conversations with optional filters.
func (s *Service) ListConversations(p *common.Pagination, filter *ConversationFilter) ([]CustomerConversation, int64, error) {
	var items []CustomerConversation
	var total int64

	q := s.db.Model(&CustomerConversation{})
	if filter != nil {
		if filter.Status != "" {
			q = q.Where("status = ?", filter.Status)
		}
		if filter.Priority != "" {
			q = q.Where("priority = ?", filter.Priority)
		}
		if filter.Platform != "" {
			q = q.Where("platform = ?", filter.Platform)
		}
		if filter.Search != "" {
			like := "%" + filter.Search + "%"
			q = q.Where("customer_name ILIKE ? OR customer_email ILIKE ? OR subject ILIKE ?", like, like, like)
		}
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("last_message_at DESC NULLS LAST, created_at DESC").
		Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetConversation returns a single conversation with its messages.
func (s *Service) GetConversation(id int64) (*CustomerConversation, error) {
	var conv CustomerConversation
	if err := s.db.Preload("Messages", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at ASC")
	}).First(&conv, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &conv, nil
}

// CreateConversation creates a new conversation.
func (s *Service) CreateConversation(in *CreateConversationInput) (*CustomerConversation, error) {
	conv := CustomerConversation{
		OrderID:       in.OrderID,
		Platform:      in.Platform,
		CustomerName:  in.CustomerName,
		CustomerEmail: in.CustomerEmail,
		Subject:       in.Subject,
		AssignedTo:    in.AssignedTo,
	}
	if in.Status != "" {
		conv.Status = in.Status
	} else {
		conv.Status = "open"
	}
	if in.Priority != "" {
		conv.Priority = in.Priority
	} else {
		conv.Priority = "medium"
	}
	if err := s.db.Create(&conv).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

// UpdateConversation patches a conversation by id.
func (s *Service) UpdateConversation(id int64, in *UpdateConversationInput) (*CustomerConversation, error) {
	var conv CustomerConversation
	if err := s.db.First(&conv, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.OrderID != nil {
		updates["order_id"] = *in.OrderID
	}
	if in.Platform != nil {
		updates["platform"] = *in.Platform
	}
	if in.CustomerName != nil {
		updates["customer_name"] = *in.CustomerName
	}
	if in.CustomerEmail != nil {
		updates["customer_email"] = *in.CustomerEmail
	}
	if in.Subject != nil {
		updates["subject"] = *in.Subject
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if in.Priority != nil {
		updates["priority"] = *in.Priority
	}
	if in.AssignedTo != nil {
		updates["assigned_to"] = *in.AssignedTo
	}
	if len(updates) == 0 {
		return &conv, nil
	}
	if err := s.db.Model(&conv).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&conv, id).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

// DeleteConversation removes a conversation by id.
func (s *Service) DeleteConversation(id int64) error {
	res := s.db.Delete(&CustomerConversation{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ---------- Messages ----------

// SendReply sends a reply (creates a message and updates the conversation).
func (s *Service) SendReply(conversationID int64, content string, isAuto bool) (*ChatMessage, error) {
	// Verify conversation exists.
	var conv CustomerConversation
	if err := s.db.First(&conv, conversationID).Error; err != nil {
		return nil, err
	}

	msg := ChatMessage{
		ConversationID: conversationID,
		SenderType:     "agent",
		Content:        content,
		AutoReplied:    isAuto,
	}
	if err := s.db.Create(&msg).Error; err != nil {
		return nil, err
	}

	// Update conversation's last_message_at and set status to pending if currently open.
	now := time.Now()
	updates := map[string]interface{}{
		"last_message_at": now,
	}
	if conv.Status == "open" {
		updates["status"] = "pending"
	}
	if err := s.db.Model(&conv).Updates(updates).Error; err != nil {
		return nil, err
	}

	return &msg, nil
}

// GetMessages returns all messages for a conversation.
func (s *Service) GetMessages(conversationID int64) ([]ChatMessage, error) {
	var msgs []ChatMessage
	if err := s.db.Where("conversation_id = ?", conversationID).
		Order("created_at ASC").Find(&msgs).Error; err != nil {
		return nil, err
	}
	return msgs, nil
}

// CloseConversation closes a conversation.
func (s *Service) CloseConversation(id int64) error {
	res := s.db.Model(&CustomerConversation{}).Where("id = ?", id).
		Update("status", "closed")
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ---------- Templates ----------

// ListTemplates returns templates, optionally filtered by category and platform.
func (s *Service) ListTemplates(category, platform string) ([]AutoReplyTemplate, error) {
	var items []AutoReplyTemplate
	q := s.db.Model(&AutoReplyTemplate{}).Where("enabled = ?", true)
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if platform != "" {
		q = q.Where("platform = ?", platform)
	}
	if err := q.Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// GetTemplate returns a single template by id.
func (s *Service) GetTemplate(id int64) (*AutoReplyTemplate, error) {
	var t AutoReplyTemplate
	if err := s.db.First(&t, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &t, nil
}

// CreateTemplate creates a new auto-reply template.
func (s *Service) CreateTemplate(in *CreateTemplateInput) (*AutoReplyTemplate, error) {
	t := AutoReplyTemplate{
		Name:      in.Name,
		Category:  in.Category,
		Content:   in.Content,
		Variables: in.Variables,
		Platform:  in.Platform,
		Enabled:   true,
	}
	if in.Enabled != nil {
		t.Enabled = *in.Enabled
	}
	if err := s.db.Create(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// UpdateTemplate patches a template by id.
func (s *Service) UpdateTemplate(id int64, in *UpdateTemplateInput) (*AutoReplyTemplate, error) {
	var t AutoReplyTemplate
	if err := s.db.First(&t, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.Name != nil {
		updates["name"] = *in.Name
	}
	if in.Category != nil {
		updates["category"] = *in.Category
	}
	if in.Content != nil {
		updates["content"] = *in.Content
	}
	if in.Variables != nil {
		updates["variables"] = *in.Variables
	}
	if in.Platform != nil {
		updates["platform"] = *in.Platform
	}
	if in.Enabled != nil {
		updates["enabled"] = *in.Enabled
	}
	if len(updates) == 0 {
		return &t, nil
	}
	if err := s.db.Model(&t).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// DeleteTemplate removes a template by id.
func (s *Service) DeleteTemplate(id int64) error {
	res := s.db.Delete(&AutoReplyTemplate{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ---------- Blacklist ----------

// AddBlacklist adds an entry to the blacklist.
func (s *Service) AddBlacklist(in *CreateBlacklistInput) (*BlacklistEntry, error) {
	entry := BlacklistEntry{
		CustomerEmail: in.CustomerEmail,
		CustomerName:  in.CustomerName,
		Reason:        in.Reason,
		AddedBy:       in.AddedBy,
	}
	if err := s.db.Create(&entry).Error; err != nil {
		return nil, err
	}
	return &entry, nil
}

// CheckBlacklist checks if an email is in the blacklist.
func (s *Service) CheckBlacklist(email string) bool {
	var count int64
	s.db.Model(&BlacklistEntry{}).Where("customer_email = ?", email).Count(&count)
	return count > 0
}

// ListBlacklist returns all blacklist entries.
func (s *Service) ListBlacklist() ([]BlacklistEntry, error) {
	var items []BlacklistEntry
	if err := s.db.Model(&BlacklistEntry{}).Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// DeleteBlacklist removes a blacklist entry by id.
func (s *Service) DeleteBlacklist(id int64) error {
	res := s.db.Delete(&BlacklistEntry{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
