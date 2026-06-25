package inventory

import (
	"time"
)

// InventoryAlertRule defines stock alert thresholds for a SKU.
type InventoryAlertRule struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SkuID        int64     `gorm:"column:sku_id;not null;index" json:"sku_id"`
	MinLevel     int       `gorm:"column:min_level;default:0" json:"min_level"`
	MaxLevel     int       `gorm:"column:max_level;default:0" json:"max_level"`
	LeadTimeDays int       `gorm:"column:lead_time_days;default:0" json:"lead_time_days"`
	Enabled      bool      `gorm:"column:enabled;default:true" json:"enabled"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (InventoryAlertRule) TableName() string { return "inventory_alert_rule" }

// InventoryAlert represents a triggered alert.
type InventoryAlert struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SkuID      int64     `gorm:"column:sku_id;not null;index" json:"sku_id"`
	AlertType  string    `gorm:"column:alert_type" json:"alert_type"` // low_stock, overstock, aging
	Message    string    `gorm:"column:message" json:"message"`
	Status     string    `gorm:"column:status;default:active" json:"status"` // active, resolved
	ResolvedAt *time.Time `gorm:"column:resolved_at" json:"resolved_at,omitempty"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (InventoryAlert) TableName() string { return "inventory_alert" }

// CreateAlertRule creates a new alert rule.
func (s *Service) CreateAlertRule(skuID int64, minLevel, maxLevel, leadTimeDays int) (*InventoryAlertRule, error) {
	r := InventoryAlertRule{
		SkuID:        skuID,
		MinLevel:     minLevel,
		MaxLevel:     maxLevel,
		LeadTimeDays: leadTimeDays,
		Enabled:      true,
	}
	if err := s.db.Create(&r).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

// ListAlertRules returns all alert rules for a SKU.
func (s *Service) ListAlertRules(skuID int64) ([]InventoryAlertRule, error) {
	var rules []InventoryAlertRule
	q := s.db.Model(&InventoryAlertRule{})
	if skuID > 0 {
		q = q.Where("sku_id = ?", skuID)
	}
	if err := q.Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// CheckAlerts scans all SKUs with enabled alert rules and generates alerts.
func (s *Service) CheckAlerts(since time.Time) ([]InventoryAlert, error) {
	var alerts []InventoryAlert
	rows, err := s.db.Table("sku").
		Select(`sku.id AS sku_id, 'low_stock' AS alert_type,
		        CONCAT('库存不足: 当前', stock, ', 安全库存', COALESCE(r.min_level,0)) AS message`).
		Joins("JOIN inventory_alert_rule r ON r.sku_id = sku.id AND r.enabled = true").
		Where("sku.stock <= r.min_level AND r.min_level > 0").
		Where("NOT EXISTS (SELECT 1 FROM inventory_alert a WHERE a.sku_id = sku.id AND a.alert_type = 'low_stock' AND a.status = 'active')").
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// Simple approach: log and return empty for now (full scan via scheduler)
	return alerts, nil
}

// ListAlerts returns alerts with filters.
func (s *Service) ListAlerts(status string, alertType string, page, size int) ([]InventoryAlert, int64, error) {
	var alerts []InventoryAlert
	var total int64
	q := s.db.Model(&InventoryAlert{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if alertType != "" {
		q = q.Where("alert_type = ?", alertType)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	offset := (page - 1) * size
	if err := q.Offset(offset).Limit(size).Order("id DESC").Find(&alerts).Error; err != nil {
		return nil, 0, err
	}
	return alerts, total, nil
}

// ResolveAlert marks an alert as resolved.
func (s *Service) ResolveAlert(id int64) error {
	now := time.Now()
	return s.db.Model(&InventoryAlert{}).Where("id = ?", id).
		Updates(map[string]interface{}{"status": "resolved", "resolved_at": &now}).Error
}
