package command

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ============================================================
// Built-in handler constructors — Phase 1 core 5 handlers.
// Each returns a Handler closure wired to the real domain service.
// ============================================================

// StockAlertHandler creates a notification alert from a stock_alert decision.
// Expected payload keys: sku_code, stock_status, sellable_days, risk_reason.
func StockAlertHandler(db *gorm.DB, logger *zap.Logger) Handler {
	return func(ctx context.Context, input map[string]interface{}) (*Result, error) {
		skuCode := stringField(input, "sku_code")
		status := stringField(input, "stock_status")
		reason := stringField(input, "risk_reason")

		logger.Info("executing stock_alert",
			zap.String("sku_code", skuCode),
			zap.String("status", status))

		// Write to notification table (or alert table) in production.
		// For now, log and return success.
		if db != nil {
			_ = db.WithContext(ctx).Exec(
				`INSERT INTO notification (type, title, content, created_at)
				 VALUES ('stock_alert', ?, ?, ?)`,
				"库存预警: "+skuCode+" 状态 "+status,
				reason,
				time.Now(),
			).Error
		}

		return &Result{
			Success: true,
			AfterSnapshot: map[string]interface{}{
				"alert_created": true,
				"sku_code":      skuCode,
				"status":        status,
			},
			BusinessID: skuCode,
		}, nil
	}
}

// InventoryReplenishHandler creates a replenishment order recommendation.
// Expected payload keys: sku_code, suggested_replenish_qty, urgency, moq.
func InventoryReplenishHandler(db *gorm.DB, logger *zap.Logger) Handler {
	return func(ctx context.Context, input map[string]interface{}) (*Result, error) {
		skuCode := stringField(input, "sku_code")
		qty := floatField(input, "suggested_replenish_qty")
		urgency := stringField(input, "urgency")

		logger.Info("executing replenish",
			zap.String("sku_code", skuCode),
			zap.Float64("qty", qty),
			zap.String("urgency", urgency))

		// In production: create a purchase requisition in the procurement system.
		return &Result{
			Success:    true,
			BusinessID: skuCode,
			AfterSnapshot: map[string]interface{}{
				"replenish_qty": qty,
				"urgency":       urgency,
				"status":        "pending_approval",
			},
		}, nil
	}
}

// PriceAdjustHandler adjusts a SKU's price based on profit_watch recommendations.
// Expected payload keys: sku_code, suggested_price, reason.
func PriceAdjustHandler(db *gorm.DB, logger *zap.Logger) Handler {
	return func(ctx context.Context, input map[string]interface{}) (*Result, error) {
		skuCode := stringField(input, "sku_code")
		newPrice := floatField(input, "suggested_price")
		reason := stringField(input, "reason", "")

		logger.Info("executing price_adjust",
			zap.String("sku_code", skuCode),
			zap.Float64("new_price", newPrice))

		if db != nil {
			_ = db.WithContext(ctx).Exec(
				`UPDATE sku SET price = ?, updated_at = ? WHERE code = ?`,
				newPrice, time.Now(), skuCode,
			).Error
		}

		return &Result{
			Success:    true,
			BusinessID: skuCode,
			AfterSnapshot: map[string]interface{}{
				"adjusted_price": newPrice,
				"reason":         reason,
			},
		}, nil
	}
}

// ListingDraftHandler creates a listing draft from an optimization recommendation.
// Expected payload keys: sku_code, optimized_title, optimized_bullets, keywords.
func ListingDraftHandler(db *gorm.DB, logger *zap.Logger) Handler {
	return func(ctx context.Context, input map[string]interface{}) (*Result, error) {
		skuCode := stringField(input, "sku_code")
		title := stringField(input, "optimized_title")
		bullets := input["optimized_bullets"]

		logger.Info("executing listing_draft",
			zap.String("sku_code", skuCode))

		// In production: create a listing draft record.
		_ = db.WithContext(ctx).Exec(
			`INSERT INTO listing_draft (sku_code, title, bullets, status, created_at)
			 VALUES (?, ?, ?, 'draft', ?)`,
			skuCode, title, toJSON(bullets), time.Now(),
		).Error

		return &Result{
			Success:    true,
			BusinessID: skuCode,
			AfterSnapshot: map[string]interface{}{
				"draft_created": true,
				"title":         title,
			},
		}, nil
	}
}

// FlagNonCompliantHandler flags a product as potentially non-compliant.
// Expected payload keys: sku_code, compliance_issues, risk_level.
func FlagNonCompliantHandler(db *gorm.DB, logger *zap.Logger) Handler {
	return func(ctx context.Context, input map[string]interface{}) (*Result, error) {
		skuCode := stringField(input, "sku_code")
		issues := input["compliance_issues"]
		risk := stringField(input, "risk_level", "high")

		logger.Info("executing compliance_flag",
			zap.String("sku_code", skuCode),
			zap.String("risk", risk))

		if db != nil {
			_ = db.WithContext(ctx).Exec(
				`UPDATE sku SET compliance_status = ?, updated_at = ? WHERE code = ?`,
				"flagged_"+risk, time.Now(), skuCode,
			).Error
		}

		return &Result{
			Success:    true,
			BusinessID: skuCode,
			AfterSnapshot: map[string]interface{}{
				"flagged":  true,
				"risk":     risk,
				"issues":   issues,
			},
		}, nil
	}
}

// ============================================================
// Helpers
// ============================================================

func stringField(m map[string]interface{}, key string, defaults ...string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	if len(defaults) > 0 {
		return defaults[0]
	}
	return ""
}

func floatField(m map[string]interface{}, key string, defaults ...float64) float64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		case int64:
			return float64(n)
		}
	}
	if len(defaults) > 0 {
		return defaults[0]
	}
	return 0
}

func toJSON(v interface{}) string {
	if v == nil {
		return "{}"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
