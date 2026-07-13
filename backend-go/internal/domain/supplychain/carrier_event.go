package supplychain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrCarrierEventConflict = errors.New("carrier event idempotency conflict")
	ErrTrackingNotOwned     = errors.New("tracking record not found for owner")
)

// IngestCarrierEvent saves an immutable, Owner-scoped carrier observation.
// Replaying the same source event and payload is idempotent. Reusing its
// identity with different facts fails closed.
func (s *TrackingService) IngestCarrierEvent(ctx context.Context, ownerID int64, trackingID string, req *IngestCarrierEventRequest) (*CarrierEvent, bool, error) {
	if ownerID <= 0 || strings.TrimSpace(trackingID) == "" {
		return nil, false, ErrTrackingNotOwned
	}
	if req == nil || strings.TrimSpace(req.SourceSystem) == "" || strings.TrimSpace(req.ExternalEventID) == "" || !isTrackingStatusValid(req.Status) || req.OccurredAt.IsZero() || req.ObservedAt.IsZero() || len(req.RawPayload) == 0 || !json.Valid(req.RawPayload) {
		return nil, false, fmt.Errorf("invalid carrier event")
	}
	if req.ObservedAt.Before(req.OccurredAt) {
		return nil, false, fmt.Errorf("observed_at must not precede occurred_at")
	}
	digestBytes := sha256.Sum256(req.RawPayload)
	digest := hex.EncodeToString(digestBytes[:])

	var out CarrierEvent
	replayed := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tracking SupplyChainTracking
		if err := tx.Where("id = ? AND owner_id = ?", trackingID, ownerID).First(&tracking).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTrackingNotOwned
			}
			return err
		}

		source := strings.TrimSpace(req.SourceSystem)
		externalID := strings.TrimSpace(req.ExternalEventID)
		lookup := tx.Where("owner_id = ? AND source_system = ? AND external_event_id = ?", ownerID, source, externalID).First(&out)
		if lookup.Error == nil {
			if out.TrackingID != trackingID || out.PayloadSHA256 != digest || out.Status != req.Status || !out.OccurredAt.Equal(req.OccurredAt.UTC()) {
				return ErrCarrierEventConflict
			}
			replayed = true
			return nil
		}
		if !errors.Is(lookup.Error, gorm.ErrRecordNotFound) {
			return lookup.Error
		}

		out = CarrierEvent{
			OwnerID: ownerID, TrackingID: trackingID, SourceSystem: source,
			ExternalEventID: externalID, Status: req.Status,
			OccurredAt: req.OccurredAt.UTC(), ObservedAt: req.ObservedAt.UTC(),
			Location: strings.TrimSpace(req.Location), Message: strings.TrimSpace(req.Message),
			RawPayload: append(json.RawMessage(nil), req.RawPayload...), PayloadSHA256: digest, TruthStatus: "external_observed",
		}
		if err := tx.Create(&out).Error; err != nil {
			return err
		}

		updates := map[string]any{"status": req.Status}
		if req.Status == "delivered" {
			// This projection is allowed only because the immutable source event is
			// external_observed. Manual and mock status updates cannot set it.
			occurred := req.OccurredAt.UTC()
			updates["actual_delivery"] = &occurred
		}
		return tx.Model(&tracking).Updates(updates).Error
	})
	return &out, replayed, err
}

func (s *TrackingService) ListCarrierEvents(ctx context.Context, ownerID int64, trackingID string) ([]CarrierEvent, error) {
	var items []CarrierEvent
	if err := s.db.WithContext(ctx).Where("owner_id = ? AND tracking_id = ?", ownerID, trackingID).Order("occurred_at ASC, id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// CreateForOwner is the HTTP boundary for new records. Legacy callers may use
// Create internally, but all authenticated API-created records are isolated.
func (s *TrackingService) CreateForOwner(ctx context.Context, ownerID int64, req *CreateTrackingRequest) (*SupplyChainTracking, error) {
	if ownerID <= 0 {
		return nil, ErrTrackingNotOwned
	}
	if strings.TrimSpace(req.OrderID) != "" {
		var count int64
		err := s.db.WithContext(ctx).Table("platform_order_ingest").
			Where("owner_id = ? AND external_order_id = ? AND truth_status = ? AND processing_status = ? AND normalized_order_id IS NOT NULL", ownerID, strings.TrimSpace(req.OrderID), "external_observed", "applied").Count(&count).Error
		if err != nil || count == 0 {
			return nil, fmt.Errorf("order_id lacks Owner-scoped external order fact")
		}
	}
	return s.create(ctx, ownerID, req)
}

func (s *TrackingService) GetByIDForOwner(ctx context.Context, ownerID int64, id string) (*SupplyChainTracking, error) {
	var item SupplyChainTracking
	err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", id, ownerID).First(&item).Error
	return &item, err
}
