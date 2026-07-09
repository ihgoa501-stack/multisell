package settlement

import (
	"context"
	"fmt"
	"math"

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

// NewRecalculator creates a new Recalculator.
func NewRecalculator(db *gorm.DB, logger *zap.Logger) *Recalculator {
	return &Recalculator{db: db, logger: logger}
}

// RecalculateProfit computes real profit from settlement items for the given order.
// Updates sales_order.profit_amount and profit_margin, and upserts order_profit_record.
func (r *Recalculator) RecalculateProfit(ctx context.Context, orderNo string) error {
	var items []SettlementItem
	if err := r.db.WithContext(ctx).
		Where("order_no = ?", orderNo).
		Find(&items).Error; err != nil {
		return fmt.Errorf("query settlement items: %w", err)
	}
	if len(items) == 0 {
		return nil
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

	r2 := func(v float64) float64 { return math.Round(v*100) / 100 }

	if err := r.db.WithContext(ctx).Model(&o).Updates(map[string]interface{}{
		"profit_amount": r2(record.Profit),
		"profit_margin": record.Margin,
	}).Error; err != nil {
		return fmt.Errorf("update sales_order profit: %w", err)
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
	var orderSaleAmt, orderSaleFee, platformFee, paymentFee, shippingFee, refundAmt float64

	for _, item := range items {
		switch item.TransactionType {
		case "order_sale":
			orderSaleAmt += item.Amount
			orderSaleFee += item.Fee
		case "platform_fee":
			platformFee += item.Amount
		case "payment_fee":
			paymentFee += item.Amount
		case "shipping_fee":
			shippingFee += item.Amount
		case "refund":
			// Refund amounts are positive; they reduce profit
			refundAmt += math.Abs(item.Amount)
		}
	}

	revenue := orderSaleAmt - orderSaleFee // net sale revenue after commission
	if revenue == 0 {
		return nil
	}

	totalCost := productCost + orderSaleFee + platformFee + paymentFee + shippingFee + refundAmt
	profitVal := revenue - totalCost

	margin := 0.0
	if revenue > 0 {
		margin = (profitVal / revenue) * 100
		margin = math.Round(margin*100) / 100
	}

	r2 := func(v float64) float64 {
		return math.Round(v*100) / 100
	}

	return &profit.OrderProfitRecord{
		Revenue:      r2(revenue),
		Cost:         r2(productCost),
		ShippingCost: r2(shippingFee),
		PlatformFee:  r2(orderSaleFee + platformFee),
		PaymentFee:   r2(paymentFee),
		TariffCost:   0, // tariff not available from settlement data
		TotalCost:    r2(totalCost),
		Profit:       r2(profitVal),
		Margin:       margin,
	}
}
