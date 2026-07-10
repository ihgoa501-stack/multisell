package aimapper

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Pipeline orchestrates the end-to-end flow: fetch raw event -> map -> persist.
type Pipeline struct {
	mapper *Mapper
	db     *gorm.DB
	logger *zap.Logger
}

// NewPipeline creates a Pipeline.
func NewPipeline(mapper *Mapper, db *gorm.DB, logger *zap.Logger) *Pipeline {
	return &Pipeline{mapper: mapper, db: db, logger: logger}
}

// ProcessRawEvent reads a RawEvent row, runs it through the mapper,
// and persists the result to the appropriate domain table.
// Returns the updated mapping_status.
func (p *Pipeline) ProcessRawEvent(ctx context.Context, eventID int64, platformCode, eventType string, rawPayload json.RawMessage) (string, error) {
	result, err := p.mapper.MapEvent(ctx, platformCode, eventType, rawPayload)
	if err != nil {
		p.logger.Error("pipeline: map event failed",
			zap.Int64("event_id", eventID),
			zap.String("platform", platformCode),
			zap.String("event_type", eventType),
			zap.Error(err),
		)
		p.updateStatus(ctx, eventID, "failed", nil, err.Error())
		return "failed", err
	}

	// Persist to domain table based on target.
	if persistErr := p.persist(ctx, result.TargetTable, result.DomainModel); persistErr != nil {
		p.logger.Error("pipeline: persist failed",
			zap.Int64("event_id", eventID),
			zap.String("target", result.TargetTable),
			zap.Error(persistErr),
		)
		p.updateStatus(ctx, eventID, "failed", nil, persistErr.Error())
		return "failed", persistErr
	}

	// Update raw event as mapped.
	mappedResult, _ := json.Marshal(result.DomainModel)
	p.updateStatus(ctx, eventID, "mapped", mappedResult, "")
	p.logger.Info("pipeline: mapped and persisted",
		zap.Int64("event_id", eventID),
		zap.String("target", result.TargetTable),
		zap.Float64("confidence", result.Confidence),
	)
	return "mapped", nil
}

// persist inserts the domain model into the appropriate table.
// ponytail: JSON marshal/unmarshal + direct table insert to avoid importing
// domain model packages (which would create circular imports via the tool bridge).
// Specific field-level mapping is worth adding when we need field-level updates.
func (p *Pipeline) persist(ctx context.Context, targetTable string, domainModel map[string]interface{}) error {
	// Remove internal fields not meant for the domain table.
	delete(domainModel, "platform_code")
	delete(domainModel, "event_type")

	switch targetTable {
	case "sales_order":
		// Upsert on order_no.
		orderNo, _ := domainModel["order_no"].(string)
		if orderNo == "" {
			return fmt.Errorf("pipeline: sales_order missing order_no")
		}
		return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var count int64
			tx.Table("sales_order").Where("order_no = ?", orderNo).Count(&count)
			if count > 0 {
				return tx.Table("sales_order").Where("order_no = ?", orderNo).Updates(domainModel).Error
			}
			return tx.Table("sales_order").Create(domainModel).Error
		})

	case "settlement_item":
		txnID, _ := domainModel["transaction_id"].(string)
		if txnID == "" {
			return nil // skip if no transaction_id
		}
		var count int64
		p.db.Table("settlement_item").Where("transaction_id = ?", txnID).Count(&count)
		if count > 0 {
			return nil // already imported
		}
		return p.db.WithContext(ctx).Table("settlement_item").Create(domainModel).Error

	case "after_sales_order":
		orderNo, _ := domainModel["order_no"].(string)
		skuCode, _ := domainModel["sku_code"].(string)
		if orderNo == "" || skuCode == "" {
			return nil // skip if no identifying fields
		}
		// Look up local order_id from sales_order by platform order_no.
		var orderID int64
		if err := p.db.Table("sales_order").Select("id").Where("order_no = ?", orderNo).Take(&orderID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				p.logger.Warn("pipeline: after_sales_order: sales_order not found, skipping",
					zap.String("order_no", orderNo))
				return nil
			}
			return fmt.Errorf("pipeline: lookup order: %w", err)
		}
		// Replace order_no with order_id FK for after_sales_order.
		delete(domainModel, "order_no")
		delete(domainModel, "platform_code")
		delete(domainModel, "sku_code")
		domainModel["order_id"] = orderID
		var count int64
		p.db.Table("after_sales_order").Where("order_id = ?", orderID).Count(&count)
		if count > 0 {
			return nil // already imported
		}
		return p.db.WithContext(ctx).Table("after_sales_order").Create(domainModel).Error

	default:
		// Unknown target — store in mapped_result but skip domain persist.
		p.logger.Warn("pipeline: unknown target table, skipping domain persist",
			zap.String("target", targetTable),
		)
		return nil
	}
}

// updateStatus sets mapping_status and optional error_message on the raw event.
func (p *Pipeline) updateStatus(_ context.Context, eventID int64, status string, mappedResult json.RawMessage, errMsg string) {
	updates := map[string]interface{}{
		"mapping_status": status,
	}
	if mappedResult != nil {
		updates["mapped_result"] = mappedResult
		updates["confidence"] = 0.85
		updates["mapped_at"] = time.Now()
	}
	if errMsg != "" {
		updates["error_message"] = errMsg
	}
	p.db.Table("platform_raw_event").Where("id = ?", eventID).Updates(updates)
}
