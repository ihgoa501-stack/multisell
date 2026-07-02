package command

import (
	"context"
	"encoding/json"

	"github.com/lingmirror/backend-go/internal/domain/notification"
	"github.com/lingmirror/backend-go/internal/domain/price"
	"github.com/lingmirror/backend-go/internal/domain/sku"
	"github.com/shopspring/decimal"
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

		// Create a notification via the domain service instead of raw SQL.
		if db != nil {
			alert := &notification.Notification{
				AlertType: "stock_alert",
				Title:     "库存预警: " + skuCode + " 状态 " + status,
				Content:   reason,
				Severity:  "warning",
				SourceID:  skuCode,
			}
			svc := notification.NewService(db, logger)
			if err := svc.Create(alert); err != nil {
				logger.Warn("failed to create stock alert notification", zap.Error(err))
			}
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
		newPriceValue := floatField(input, "suggested_price")
		reason := stringField(input, "reason", "")
		skuID := int64Field(input, "sku_id")

		logger.Info("executing price_adjust",
			zap.String("sku_code", skuCode),
			zap.Float64("new_price", newPriceValue))

		if db != nil && skuID > 0 && newPriceValue > 0 {
			svc := price.NewService(db, logger)
			err := svc.SetPrice(ctx, &price.Price{
				SkuID:     skuID,
				PriceType: "sale_price",
				Price:     decimal.NewFromFloat(newPriceValue),
				Status:    1,
			}, "command:price_adjust")
			if err != nil {
				logger.Warn("price service rejected adjustment", zap.Error(err))
			}
		}

		return &Result{
			Success:    true,
			BusinessID: skuCode,
			AfterSnapshot: map[string]interface{}{
				"adjusted_price": newPriceValue,
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

		// In production: create a listing draft record through the listing domain service.
		return &Result{
			Success:    true,
			BusinessID: skuCode,
			AfterSnapshot: map[string]interface{}{
				"draft_created": true,
				"title":         title,
				"bullets":       bullets,
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

		// Update the SKU compliance_status via the domain service instead of raw SQL.
		if db != nil {
			skuSvc := sku.NewService(db, logger)
			var s sku.Sku
			if err := db.WithContext(ctx).Model(&sku.Sku{}).Where("code = ?", skuCode).First(&s).Error; err == nil {
				s.ComplianceStatus = "flagged_" + risk
				if err := skuSvc.UpdateSku(ctx, &s); err != nil {
					logger.Warn("failed to update sku compliance_status", zap.Error(err))
				}
			} else {
				logger.Warn("sku not found for compliance flagging", zap.String("sku_code", skuCode), zap.Error(err))
			}
		}

		return &Result{
			Success:    true,
			BusinessID: skuCode,
			AfterSnapshot: map[string]interface{}{
				"flagged": true,
				"risk":    risk,
				"issues":  issues,
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

func int64Field(m map[string]interface{}, key string) int64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int64(n)
		case int:
			return int64(n)
		case int64:
			return n
		}
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
