package consolidation

import (
	"errors"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Defaults & constants
// ---------------------------------------------------------------------------

const (
	// DefaultTimeWindow is the default time window (72 hours = 3 days) for grouping.
	DefaultTimeWindow = 72 * time.Hour
)

// ---------------------------------------------------------------------------
// Discount tier thresholds (weight in kg → discount rate as decimal 0-1)
// ---------------------------------------------------------------------------

type discountTier struct {
	MinWeightKg float64
	DiscountPct float64
	Label       string
}

var discountTiers = []discountTier{
	{MinWeightKg: 500, DiscountPct: 20.0, Label: "20% (>=500kg)"},
	{MinWeightKg: 100, DiscountPct: 10.0, Label: "10% (>=100kg)"},
	{MinWeightKg: 50, DiscountPct: 5.0, Label: "5% (>=50kg)"},
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// ConsolidationService provides consolidation group business logic.
type ConsolidationService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewConsolidationService creates a new consolidation service.
func NewConsolidationService(db *gorm.DB, logger *zap.Logger) *ConsolidationService {
	return &ConsolidationService{db: db, logger: logger}
}

// ---------------------------------------------------------------------------
// Discount calculation
// ---------------------------------------------------------------------------

// CalculateDiscountPct returns the discount percentage based on total weight.
// Tiers:
//   - >= 500 kg → 20%
//   - >= 100 kg → 10%
//   - >= 50 kg  → 5%
//   - < 50 kg   → 0%
func CalculateDiscountPct(totalWeightKg float64) float64 {
	for _, tier := range discountTiers {
		if totalWeightKg >= tier.MinWeightKg {
			return tier.DiscountPct
		}
	}
	return 0
}

// DiscountLabel returns a human-readable label for the applicable discount tier.
func DiscountLabel(totalWeightKg float64) string {
	for _, tier := range discountTiers {
		if totalWeightKg >= tier.MinWeightKg {
			return tier.Label
		}
	}
	return "0% (<50kg)"
}

// ---------------------------------------------------------------------------
// CreateGroup
// ---------------------------------------------------------------------------

// CreateGroup creates a new consolidation group for the given destination.
// Items that ship within the time window can be added later.
func (s *ConsolidationService) CreateGroup(destination string, timeWindowH int) (*ConsolidationGroup, error) {
	if destination == "" {
		return nil, errors.New("consolidation: destination is required")
	}

	group := &ConsolidationGroup{
		Status:      GroupStatusOpen,
		Destination: destination,
	}

	// The time window is advisory metadata (logged/annotated on the group).
	// For MVP, we store it by setting an explicit reference.
	if timeWindowH <= 0 {
		timeWindowH = int(DefaultTimeWindow.Hours())
	}

	if err := s.db.Create(group).Error; err != nil {
		return nil, fmt.Errorf("consolidation: create group: %w", err)
	}

	s.logger.Info("consolidation group created",
		zap.Int64("group_id", group.ID),
		zap.String("destination", destination),
		zap.Int("time_window_h", timeWindowH),
	)

	return group, nil
}

// ---------------------------------------------------------------------------
// AddItem
// ---------------------------------------------------------------------------

// AddItem adds a shipment item to an existing open consolidation group.
func (s *ConsolidationService) AddItem(groupID int64, skuID int64, weightKg float64, volumeM3 float64, destination string) (*ConsolidationItem, error) {
	if weightKg <= 0 {
		return nil, errors.New("consolidation: weight must be positive")
	}
	if destination == "" {
		return nil, errors.New("consolidation: destination is required")
	}

	// Load the group to verify it exists and is open.
	group, err := s.GetGroup(groupID)
	if err != nil {
		return nil, fmt.Errorf("consolidation: group not found: %w", err)
	}
	if group.Status != GroupStatusOpen {
		return nil, fmt.Errorf("consolidation: cannot add items to group with status %s", group.Status)
	}

	// Verify destination matches (same-destination consolidation rule).
	if group.Destination != destination {
		return nil, fmt.Errorf("consolidation: item destination %q does not match group destination %q", destination, group.Destination)
	}

	item := &ConsolidationItem{
		GroupID:     groupID,
		SkuID:       skuID,
		WeightKg:    weightKg,
		VolumeM3:    volumeM3,
		Destination: destination,
		Status:      ItemStatusPending,
	}

	if err := s.db.Create(item).Error; err != nil {
		return nil, fmt.Errorf("consolidation: create item: %w", err)
	}

	// Update group totals.
	if err := s.recalculateGroupTotals(groupID); err != nil {
		s.logger.Warn("consolidation: recalculate group totals failed after add item",
			zap.Int64("group_id", groupID),
			zap.Error(err),
		)
	}

	s.logger.Info("consolidation item added",
		zap.Int64("item_id", item.ID),
		zap.Int64("group_id", groupID),
		zap.Int64("sku_id", skuID),
		zap.Float64("weight_kg", weightKg),
	)

	return item, nil
}

// ---------------------------------------------------------------------------
// RemoveItem
// ---------------------------------------------------------------------------

// RemoveItem removes (marks as removed) an item from its consolidation group.
func (s *ConsolidationService) RemoveItem(itemID int64) error {
	var item ConsolidationItem
	if err := s.db.First(&item, itemID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("consolidation: item %d not found", itemID)
		}
		return fmt.Errorf("consolidation: get item %d: %w", itemID, err)
	}

	if item.Status == ItemStatusRemoved {
		return errors.New("consolidation: item already removed")
	}

	// Confirm the group is still open.
	var group ConsolidationGroup
	if err := s.db.First(&group, item.GroupID).Error; err != nil {
		return fmt.Errorf("consolidation: group %d not found", item.GroupID)
	}
	if group.Status != GroupStatusOpen {
		return fmt.Errorf("consolidation: cannot remove items from group with status %s", group.Status)
	}

	if err := s.db.Model(&item).Update("status", ItemStatusRemoved).Error; err != nil {
		return fmt.Errorf("consolidation: remove item %d: %w", itemID, err)
	}

	// Recalculate group totals.
	if err := s.recalculateGroupTotals(item.GroupID); err != nil {
		s.logger.Warn("consolidation: recalculate group totals failed after remove item",
			zap.Int64("group_id", item.GroupID),
			zap.Error(err),
		)
	}

	s.logger.Info("consolidation item removed",
		zap.Int64("item_id", itemID),
		zap.Int64("group_id", item.GroupID),
	)

	return nil
}

// ---------------------------------------------------------------------------
// Negotiate
// ---------------------------------------------------------------------------

// Negotiate calculates the bulk discount rate for a consolidation group
// based on its total weight and updates the group's discount and status.
func (s *ConsolidationService) Negotiate(groupID int64) (*ConsolidationGroup, error) {
	group, err := s.GetGroup(groupID)
	if err != nil {
		return nil, fmt.Errorf("consolidation: group not found: %w", err)
	}

	if group.Status != GroupStatusOpen && group.Status != GroupStatusNegotiating {
		return nil, fmt.Errorf("consolidation: cannot negotiate group with status %s", group.Status)
	}

	// Recalculate totals from confirmed/pending items.
	if err := s.recalculateGroupTotals(groupID); err != nil {
		return nil, fmt.Errorf("consolidation: recalculate totals: %w", err)
	}

	// Re-fetch the updated group.
	group, err = s.GetGroup(groupID)
	if err != nil {
		return nil, err
	}

	discountPct := CalculateDiscountPct(group.TotalWeightKg)
	updateMap := map[string]interface{}{
		"status":        GroupStatusNegotiating,
		"discount_rate": math.Round(discountPct*100) / 100,
	}

	if err := s.db.Model(group).Updates(updateMap).Error; err != nil {
		return nil, fmt.Errorf("consolidation: negotiate group %d: %w", groupID, err)
	}

	group.Status = GroupStatusNegotiating
	group.DiscountRate = discountPct

	s.logger.Info("consolidation group negotiated",
		zap.Int64("group_id", groupID),
		zap.Float64("total_weight_kg", group.TotalWeightKg),
		zap.Float64("discount_rate", discountPct),
	)

	return group, nil
}

// ---------------------------------------------------------------------------
// GetGroup & ListGroups
// ---------------------------------------------------------------------------

// GetGroup retrieves a consolidation group by ID.
func (s *ConsolidationService) GetGroup(id int64) (*ConsolidationGroup, error) {
	var group ConsolidationGroup
	if err := s.db.First(&group, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("consolidation: group %d not found", id)
		}
		return nil, fmt.Errorf("consolidation: get group %d: %w", id, err)
	}
	return &group, nil
}

// ListGroups returns all consolidation groups, ordered by creation time descending.
func (s *ConsolidationService) ListGroups() ([]ConsolidationGroup, error) {
	var groups []ConsolidationGroup
	if err := s.db.Order("created_at DESC").Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("consolidation: list groups: %w", err)
	}
	return groups, nil
}

// GetItemsByGroup returns all items belonging to a consolidation group.
func (s *ConsolidationService) GetItemsByGroup(groupID int64) ([]ConsolidationItem, error) {
	var items []ConsolidationItem
	if err := s.db.Where("group_id = ?", groupID).Order("created_at ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("consolidation: items for group %d: %w", groupID, err)
	}
	return items, nil
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// recalculateGroupTotals recomputes total_weight_kg and total_volume_m3
// for the group from all non-removed items.
func (s *ConsolidationService) recalculateGroupTotals(groupID int64) error {
	var totals struct {
		TotalWeightKg float64
		TotalVolumeM3 float64
	}
	if err := s.db.Model(&ConsolidationItem{}).
		Select("COALESCE(SUM(weight_kg), 0) AS total_weight_kg, COALESCE(SUM(volume_m3), 0) AS total_volume_m3").
		Where("group_id = ? AND status != ?", groupID, ItemStatusRemoved).
		Scan(&totals).Error; err != nil {
		return err
	}

	return s.db.Model(&ConsolidationGroup{}).
		Where("id = ?", groupID).
		Updates(map[string]interface{}{
			"total_weight_kg": math.Round(totals.TotalWeightKg*100) / 100,
			"total_volume_m3": math.Round(totals.TotalVolumeM3*100) / 100,
		}).Error
}
