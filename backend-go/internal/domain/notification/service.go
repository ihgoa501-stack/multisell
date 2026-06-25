package notification

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ListFilter holds notification list query parameters.
type ListFilter struct {
	UserID    int64
	AlertType string
	Severity  string
	IsRead    *int // nil = all, 0 = unread, 1 = read
}

// Service provides notification business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new notification service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// List returns notifications matching the filter with pagination.
func (s *Service) List(f ListFilter, page, size int) ([]Notification, int64, error) {
	q := s.db.Model(&Notification{})
	if f.UserID > 0 {
		q = q.Where("user_id = ?", f.UserID)
	}
	if f.AlertType != "" {
		q = q.Where("alert_type = ?", f.AlertType)
	}
	if f.Severity != "" {
		q = q.Where("severity = ?", f.Severity)
	}
	if f.IsRead != nil {
		q = q.Where("is_read = ?", *f.IsRead)
	}
	var total int64
	q.Count(&total)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	var items []Notification
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID returns a single notification.
func (s *Service) GetByID(id int64) (*Notification, error) {
	var n Notification
	if err := s.db.First(&n, id).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

// Create inserts a new notification (triggered by alerts or agent actions).
func (s *Service) Create(n *Notification) error {
	return s.db.Create(n).Error
}

// MarkAsRead marks a single notification as read.
func (s *Service) MarkAsRead(id int64) error {
	return s.db.Model(&Notification{}).Where("id = ?", id).Update("is_read", 1).Error
}

// MarkAllRead marks all unread notifications for a user as read.
func (s *Service) MarkAllRead(userID int64) error {
	return s.db.Model(&Notification{}).Where("user_id = ? AND is_read = 0", userID).Update("is_read", 1).Error
}

// Delete removes a notification.
func (s *Service) Delete(id int64) error {
	return s.db.Delete(&Notification{}, id).Error
}

// UnreadCount returns the number of unread notifications for a user.
func (s *Service) UnreadCount(userID int64) (int64, error) {
	var count int64
	err := s.db.Model(&Notification{}).Where("user_id = ? AND is_read = 0", userID).Count(&count).Error
	return count, err
}

// ===================== Alert Rules =====================

// ListAlertRules returns all alert rules.
func (s *Service) ListAlertRules() ([]AlertRule, error) {
	var rules []AlertRule
	if err := s.db.Order("id desc").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// GetAlertRule returns a single alert rule.
func (s *Service) GetAlertRule(id int64) (*AlertRule, error) {
	var r AlertRule
	if err := s.db.First(&r, id).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

// CreateAlertRule inserts a new alert rule.
func (s *Service) CreateAlertRule(r *AlertRule) error {
	return s.db.Create(r).Error
}

// UpdateAlertRule updates an existing alert rule.
func (s *Service) UpdateAlertRule(r *AlertRule) error {
	return s.db.Save(r).Error
}

// DeleteAlertRule removes an alert rule.
func (s *Service) DeleteAlertRule(id int64) error {
	return s.db.Delete(&AlertRule{}, id).Error
}
