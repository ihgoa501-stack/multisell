package settlement

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/order"
	"github.com/lingmirror/backend-go/internal/domain/profit"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Recalculator computes real profit from settlement data for each order.
// Uses settlement_item transaction types: order_sale, platform_fee, payment_fee,
// shipping_fee, refund — grouped by order_no — to calculate actual profit.
type Recalculator struct {
	db     *gorm.DB
	logger *zap.Logger
}

type profitSettlementItem struct {
	SettlementItem
	SettlementStatus   string     `gorm:"column:settlement_status"`
	SettlementSource   string     `gorm:"column:settlement_source"`
	SettlementImported *time.Time `gorm:"column:settlement_imported_at"`
}

// NewRecalculator creates a new Recalculator.
func NewRecalculator(db *gorm.DB, logger *zap.Logger) *Recalculator {
	return &Recalculator{db: db, logger: logger}
}

// RecalculateProfit computes real profit from settlement items for the given order.
// Updates sales_order.profit_amount and profit_margin, and upserts order_profit_record.
func (r *Recalculator) RecalculateProfit(ctx context.Context, orderNo string) error {
	var rows []profitSettlementItem
	if err := r.db.WithContext(ctx).
		Table("settlement_item AS si").
		Select("si.*, s.status AS settlement_status, s.source_type AS settlement_source, s.imported_at AS settlement_imported_at").
		Joins("JOIN settlement AS s ON s.id = si.settlement_id").
		Where("si.order_no = ?", orderNo).
		Scan(&rows).Error; err != nil {
		return fmt.Errorf("query settlement items: %w", err)
	}
	if len(rows) == 0 {
		return nil
	}
	items := make([]SettlementItem, 0, len(rows))
	settlementMissing := make([]string, 0)
	for _, row := range rows {
		items = append(items, row.SettlementItem)
		if row.SettlementStatus != "reconciled" && row.SettlementStatus != "closed" {
			settlementMissing = append(settlementMissing, "settlement_not_completed")
		}
		if row.SettlementImported == nil || (row.SettlementSource != "platform_import" && row.SettlementSource != "api_sync") {
			settlementMissing = append(settlementMissing, "settlement_source_unverified")
		}
		if row.ReconciliationStatus != "matched" || row.ReconciledAt == nil || strings.TrimSpace(row.ReconciledBy) == "" {
			settlementMissing = append(settlementMissing, "settlement_item_unreconciled")
		}
	}

	var o order.Order
	if err := r.db.WithContext(ctx).Where("order_no = ?", orderNo).First(&o).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Order may arrive later; settlement items alone can't compute profit
			return nil
		}
		return fmt.Errorf("find order: %w", err)
	}

	record := computeProfit(items, o.ProductCost)
	if record == nil {
		return nil
	}
	if len(settlementMissing) > 0 {
		record.ProfitStatus = "provisional"
		record.MissingCosts = mergeMissingReasons(record.MissingCosts, settlementMissing)
	}

	r2 := func(v float64) float64 { return math.Round(v*100) / 100 }

	// sales_order profit fields are treated as final figures elsewhere. Do not
	// populate them while a critical cost is still unsupported by evidence.
	if record.ProfitStatus == "final" {
		if err := r.db.WithContext(ctx).Model(&o).Updates(map[string]interface{}{
			"profit_amount": r2(record.Profit),
			"profit_margin": record.Margin,
		}).Error; err != nil {
			return fmt.Errorf("update sales_order profit: %w", err)
		}
	}

	// Upsert order_profit_record
	record.OrderID = o.ID
	var existing profit.OrderProfitRecord
	if err := r.db.WithContext(ctx).Where("order_id = ?", o.ID).First(&existing).Error; err == nil {
		record.ID = existing.ID
		if err := r.db.WithContext(ctx).Save(record).Error; err != nil {
			return fmt.Errorf("update order_profit_record: %w", err)
		}
	} else {
		if err := r.db.WithContext(ctx).Create(record).Error; err != nil {
			return fmt.Errorf("create order_profit_record: %w", err)
		}
	}

	return nil
}

func mergeMissingReasons(existing string, reasons []string) string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(reasons)+1)
	for _, reason := range append(strings.Split(existing, ","), reasons...) {
		reason = strings.TrimSpace(reason)
		if reason != "" && !seen[reason] {
			seen[reason] = true
			out = append(out, reason)
		}
	}
	return strings.Join(out, ",")
}

// RecalculateAllProfit recalculates profit for every order that has settlement items.
func (r *Recalculator) RecalculateAllProfit(ctx context.Context) error {
	var orderNos []string
	if err := r.db.WithContext(ctx).
		Model(&SettlementItem{}).
		Select("DISTINCT order_no").
		Where("order_no IS NOT NULL AND order_no != ''").
		Pluck("order_no", &orderNos).Error; err != nil {
		return fmt.Errorf("query distinct order_nos: %w", err)
	}

	for _, on := range orderNos {
		if err := r.RecalculateProfit(ctx, on); err != nil {
			r.logger.Error("profit recalculation failed",
				zap.String("order_no", on), zap.Error(err))
		}
	}
	return nil
}

// computeProfit aggregates settlement items and returns the OrderProfitRecord.
// Returns nil when no meaningful data exists.
func computeProfit(items []SettlementItem, productCost float64) *profit.OrderProfitRecord {
	var orderSaleAmt, orderSaleFee, platformFee, paymentFee, shippingFee, tariffFee, refundAmt float64
	var hasPlatformFee, hasPaymentFee, hasShippingFee, hasTariffFee bool

	for _, item := range items {
		switch item.TransactionType {
		case "order_sale":
			orderSaleAmt += item.Amount
			orderSaleFee += item.Fee
		case "platform_fee":
			platformFee += item.Amount
			hasPlatformFee = true
		case "payment_fee":
			paymentFee += item.Amount
			hasPaymentFee = true
		case "shipping_fee":
			shippingFee += item.Amount
			hasShippingFee = true
		case "tariff_fee":
			tariffFee += item.Amount
			hasTariffFee = true
		case "refund":
			// Refund amounts are positive; they reduce profit
			refundAmt += math.Abs(item.Amount)
		}
	}

	// Revenue is gross settlement sale proceeds. Commission belongs in costs;
	// subtracting it here as well would double-count it.
	revenue := orderSaleAmt
	if revenue == 0 {
		return nil
	}

	totalCost := productCost + orderSaleFee + platformFee + paymentFee + shippingFee + tariffFee + refundAmt
	profitVal := revenue - totalCost

	margin := 0.0
	if revenue > 0 {
		margin = (profitVal / revenue) * 100
		margin = math.Round(margin*100) / 100
	}

	r2 := func(v float64) float64 {
		return math.Round(v*100) / 100
	}

	missing := make([]string, 0, 5)
	if productCost <= 0 {
		missing = append(missing, "product_cost")
	}
	if orderSaleFee == 0 && !hasPlatformFee {
		missing = append(missing, "platform_fee")
	}
	if !hasPaymentFee {
		missing = append(missing, "payment_fee")
	}
	if !hasShippingFee {
		missing = append(missing, "shipping_fee")
	}
	if !hasTariffFee {
		missing = append(missing, "tariff_fee")
	}
	status := "final"
	if len(missing) > 0 {
		status = "provisional"
	}

	return &profit.OrderProfitRecord{
		Revenue:      r2(revenue),
		Cost:         r2(productCost),
		ShippingCost: r2(shippingFee),
		PlatformFee:  r2(orderSaleFee + platformFee),
		PaymentFee:   r2(paymentFee),
		TariffCost:   r2(tariffFee),
		TotalCost:    r2(totalCost),
		Profit:       r2(profitVal),
		Margin:       margin,
		ProfitStatus: status,
		MissingCosts: strings.Join(missing, ","),
	}
}
