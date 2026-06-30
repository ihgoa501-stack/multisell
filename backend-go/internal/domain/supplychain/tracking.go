package supplychain

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DefaultPage is the default page number for pagination.
const DefaultPage = 1

// DefaultSize is the default page size for pagination.
const DefaultSize = 20

// SupplyChainTracking maps to the PostgreSQL "supply_chain_tracking" table.
type SupplyChainTracking struct {
	ID                string           `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	FlowID            string           `gorm:"column:flow_id;type:uuid;index:idx_tracking_flow_id" json:"flow_id"`
	OrderID           string           `gorm:"column:order_id;type:varchar(100);index:idx_tracking_order_id" json:"order_id"`
	CarrierCode       string           `gorm:"column:carrier_code;type:varchar(50);index:idx_tracking_carrier_code" json:"carrier_code"`
	TrackingNo        string           `gorm:"column:tracking_no;type:varchar(200);index:idx_tracking_tracking_no" json:"tracking_no"`
	Status            string           `gorm:"column:status;type:varchar(30);default:pending;index:idx_tracking_status" json:"status"`
	EstimatedDelivery *time.Time       `gorm:"column:estimated_delivery" json:"estimated_delivery,omitempty"`
	ActualDelivery    *time.Time       `gorm:"column:actual_delivery" json:"actual_delivery,omitempty"`
	StatusHistory     *json.RawMessage `gorm:"column:status_history;type:jsonb;default:'[]'" json:"status_history,omitempty"`
	CreatedAt         time.Time        `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time        `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name for SupplyChainTracking.
func (SupplyChainTracking) TableName() string { return "supply_chain_tracking" }

// TrackingEvent is a single entry in the status_history JSONB array.
type TrackingEvent struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Location  string    `json:"location,omitempty"`
	Message   string    `json:"message,omitempty"`
}

// ValidTrackingStatuses enumerates the allowed status values.
var ValidTrackingStatuses = []string{
	"pending", "picked_up", "outbound", "transit",
	"customs", "last_mile", "delivered", "exception",
}

// isTrackingStatusValid checks whether the given status is one of the allowed values.
func isTrackingStatusValid(status string) bool {
	for _, s := range ValidTrackingStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// ErrInvalidTrackingStatus returns an error indicating an unrecognized tracking status.
func ErrInvalidTrackingStatus(status string) error {
	return &invalidTrackingStatusError{status: status}
}

type invalidTrackingStatusError struct {
	status string
}

func (e *invalidTrackingStatusError) Error() string {
	return "invalid tracking status: " + e.status
}

// ---------- Request / Response structs ----------

// CreateTrackingRequest is the payload for POST /supplychain/tracking.
type CreateTrackingRequest struct {
	FlowID            string     `json:"flow_id" binding:"required"`
	OrderID           string     `json:"order_id"`
	CarrierCode       string     `json:"carrier_code" binding:"required"`
	TrackingNo        string     `json:"tracking_no" binding:"required"`
	Status            string     `json:"status"`
	EstimatedDelivery *time.Time `json:"estimated_delivery"`
}

// UpdateTrackingRequest is the payload for PUT /supplychain/tracking/:id.
type UpdateTrackingRequest struct {
	Status            string     `json:"status" binding:"required"`
	Location          string     `json:"location"`
	Message           string     `json:"message"`
	EstimatedDelivery *time.Time `json:"estimated_delivery,omitempty"`
	ActualDelivery    *time.Time `json:"actual_delivery,omitempty"`
}

// ListTrackingRequest is the query for listing tracking records.
type ListTrackingRequest struct {
	Page        int    `form:"page"`
	Size        int    `form:"size"`
	FlowID      string `form:"flow_id"`
	OrderID     string `form:"order_id"`
	Status      string `form:"status"`
	CarrierCode string `form:"carrier_code"`
}

// TrackingService provides supply chain tracking business logic.
type TrackingService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewTrackingService creates a new supply chain tracking service.
func NewTrackingService(db *gorm.DB, logger *zap.Logger) *TrackingService {
	return &TrackingService{db: db, logger: logger}
}

// Create inserts a new tracking record.
func (s *TrackingService) Create(ctx context.Context, req *CreateTrackingRequest) (*SupplyChainTracking, error) {
	status := req.Status
	if status == "" {
		status = "pending"
	}
	if !isTrackingStatusValid(status) {
		return nil, ErrInvalidTrackingStatus(status)
	}

	tracking := SupplyChainTracking{
		FlowID:            req.FlowID,
		OrderID:           req.OrderID,
		CarrierCode:       req.CarrierCode,
		TrackingNo:        req.TrackingNo,
		Status:            status,
		EstimatedDelivery: req.EstimatedDelivery,
	}

	// Initialize status_history with the creation event.
	history := []TrackingEvent{{
		Status:    status,
		Timestamp: time.Now(),
		Location:  "",
		Message:   "Tracking record created",
	}}
	raw, err := json.Marshal(history)
	if err != nil {
		return nil, err
	}
	tracking.StatusHistory = (*json.RawMessage)(&raw)

	if err := s.db.WithContext(ctx).Create(&tracking).Error; err != nil {
		return nil, err
	}
	return &tracking, nil
}

// GetByID retrieves a single tracking record by ID.
func (s *TrackingService) GetByID(ctx context.Context, id string) (*SupplyChainTracking, error) {
	var t SupplyChainTracking
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// UpdateStatus updates the tracking record's status and appends an event to
// status_history. It validates that the status value is one of the allowed values.
func (s *TrackingService) UpdateStatus(ctx context.Context, id string, req *UpdateTrackingRequest) (*SupplyChainTracking, error) {
	if !isTrackingStatusValid(req.Status) {
		return nil, ErrInvalidTrackingStatus(req.Status)
	}

	var t SupplyChainTracking
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&t).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, err
	}

	// Append to status_history.
	var history []TrackingEvent
	if t.StatusHistory != nil && len(*t.StatusHistory) > 0 {
		if err := json.Unmarshal(*t.StatusHistory, &history); err != nil {
			history = []TrackingEvent{}
		}
	}
	history = append(history, TrackingEvent{
		Status:    req.Status,
		Timestamp: time.Now(),
		Location:  req.Location,
		Message:   req.Message,
	})
	raw, err := json.Marshal(history)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"status":         req.Status,
		"status_history": (*json.RawMessage)(&raw),
	}
	if req.EstimatedDelivery != nil {
		updates["estimated_delivery"] = req.EstimatedDelivery
	}
	if req.ActualDelivery != nil {
		updates["actual_delivery"] = req.ActualDelivery
	}

	if err := s.db.WithContext(ctx).Model(&t).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Reload the updated record.
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// AddHistoryEntry appends an arbitrary event to the status_history without
// changing the current status. Useful for adding informational milestones
// (e.g., "Package arrived at sorting center").
func (s *TrackingService) AddHistoryEntry(ctx context.Context, id string, event TrackingEvent) (*SupplyChainTracking, error) {
	var t SupplyChainTracking
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&t).Error; err != nil {
		return nil, err
	}

	var history []TrackingEvent
	if t.StatusHistory != nil && len(*t.StatusHistory) > 0 {
		if err := json.Unmarshal(*t.StatusHistory, &history); err != nil {
			history = []TrackingEvent{}
		}
	}
	history = append(history, event)
	raw, err := json.Marshal(history)
	if err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Model(&t).Update("status_history", (*json.RawMessage)(&raw)).Error; err != nil {
		return nil, err
	}

	// Reload the updated record.
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// GetByFlowID returns all tracking records for a given flow ID.
func (s *TrackingService) GetByFlowID(ctx context.Context, flowID string) ([]SupplyChainTracking, error) {
	var items []SupplyChainTracking
	if err := s.db.WithContext(ctx).
		Where("flow_id = ?", flowID).
		Order("created_at DESC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// List returns a paginated list of tracking records with optional filters.
func (s *TrackingService) List(ctx context.Context, req *ListTrackingRequest) ([]SupplyChainTracking, int64, error) {
	var items []SupplyChainTracking
	var total int64

	q := s.db.WithContext(ctx).Model(&SupplyChainTracking{})
	if req.FlowID != "" {
		q = q.Where("flow_id = ?", req.FlowID)
	}
	if req.OrderID != "" {
		q = q.Where("order_id = ?", req.OrderID)
	}
	if req.Status != "" {
		q = q.Where("status = ?", req.Status)
	}
	if req.CarrierCode != "" {
		q = q.Where("carrier_code = ?", req.CarrierCode)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	size := req.Size
	if size <= 0 {
		size = 20
	}
	offset := (page - 1) * size

	if err := q.Order("created_at DESC").Offset(offset).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
