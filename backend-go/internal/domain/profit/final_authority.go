package profit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrFinalProfitUnknown = errors.New("final profit is unknown")

// OrderProductCostAllocation freezes the exact cost version used by one order
// line. One row is required per normalized order line.
type OrderProductCostAllocation struct {
	ID, OwnerID, OrderID, OrderItemID, SKUID, SourcingCostVersionID, AmountMinor int64
	Currency, ContentSHA256                                                      string
	CreatedAt                                                                    time.Time
}

func (OrderProductCostAllocation) TableName() string { return "order_product_cost_allocation" }

// OrderFinalProfitVersion is append-only. A new calculation creates a new
// version; no previously issued final result is overwritten.
type OrderFinalProfitVersion struct {
	ID, OwnerID, OrderID, Version                                                                                     int64
	Currency                                                                                                          string
	RevenueMinor, ProductCostMinor, SettlementFeeMinor, FulfillmentFeeMinor, RefundMinor, TotalCostMinor, ProfitMinor int64
	SourceManifestSHA256                                                                                              string
	FinalizedAt                                                                                                       time.Time
}

func (OrderFinalProfitVersion) TableName() string { return "order_final_profit_version" }

type AllocateOrderProductCostInput struct{ OrderItemID, SourcingCostVersionID int64 }

type finalOrderAuthority struct {
	ID, OwnerID, NormalizedOrderID             int64
	TruthStatus, ProcessingStatus, EventAction string
}

func (finalOrderAuthority) TableName() string { return "platform_order_ingest" }

type finalOrderItem struct {
	ID       int64 `gorm:"column:id"`
	OrderID  int64 `gorm:"column:order_id"`
	SkuID    int64 `gorm:"column:sku_id"`
	Quantity int   `gorm:"column:quantity"`
}

func (finalOrderItem) TableName() string { return "sales_order_item" }

type finalCostVersion struct {
	ID             int64  `gorm:"column:id"`
	OwnerID        int64  `gorm:"column:owner_id"`
	SKUMappingID   int64  `gorm:"column:sku_mapping_id"`
	TotalMinor     int64  `gorm:"column:total_minor"`
	TargetCurrency string `gorm:"column:target_currency"`
}

func (finalCostVersion) TableName() string { return "sourcing_cost_version" }

type finalCostLine struct {
	CostVersionID int64  `gorm:"column:cost_version_id"`
	TruthStatus   string `gorm:"column:truth_status"`
}

func (finalCostLine) TableName() string { return "sourcing_cost_line" }

type finalSKUMapping struct {
	ID            int64 `gorm:"column:id"`
	OwnerID       int64 `gorm:"column:owner_id"`
	InternalSKUID int64 `gorm:"column:internal_sku_id"`
}

func (finalSKUMapping) TableName() string { return "sourcing_sku_mapping" }

type finalResolution struct {
	ID, OwnerID, OrderID int64
	Status, Currency     string
}

func (finalResolution) TableName() string { return "aftersales_resolution_case" }

type finalReceipt struct {
	ResolutionID, OwnerID, ActualMinor int64
	Outcome, Currency, SourceType      string
}

func (finalReceipt) TableName() string { return "aftersales_resolution_receipt" }

type finalSettlementIngest struct {
	ID, OwnerID                          int64
	TruthStatus, Currency, ContentSHA256 string
}

func (finalSettlementIngest) TableName() string { return "platform_settlement_ingest" }

type finalSettlementLine struct {
	ID, IngestID, OrderID, AmountMinor           int64
	Kind, FeeCode, Currency, IngestContentSHA256 string
}

func (finalSettlementLine) TableName() string { return "platform_settlement_fact_line" }

func requireExternalOrderAuthority(tx *gorm.DB, ownerID, orderID int64) error {
	var rows []finalOrderAuthority
	if err := tx.Where("owner_id=? AND normalized_order_id=? AND event_action=? AND truth_status=? AND processing_status=?", ownerID, orderID, "reserve", "external_observed", "applied").Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) != 1 {
		return fmt.Errorf("%w: order lacks exactly one applied external Owner fact", ErrFinalProfitUnknown)
	}
	return nil
}

func (s *Service) AllocateOrderProductCost(ctx context.Context, ownerID, orderID int64, in AllocateOrderProductCostInput) (*OrderProductCostAllocation, error) {
	var out OrderProductCostAllocation
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := requireExternalOrderAuthority(tx, ownerID, orderID); err != nil {
			return err
		}
		var item finalOrderItem
		if err := tx.Where("id=? AND order_id=?", in.OrderItemID, orderID).First(&item).Error; err != nil {
			return fmt.Errorf("%w: order line missing", ErrFinalProfitUnknown)
		}
		var cv finalCostVersion
		if err := tx.Where("id=? AND owner_id=?", in.SourcingCostVersionID, ownerID).First(&cv).Error; err != nil {
			return fmt.Errorf("%w: exact Owner cost version missing", ErrFinalProfitUnknown)
		}
		var mapping finalSKUMapping
		if err := tx.Where("id=? AND owner_id=? AND internal_sku_id=?", cv.SKUMappingID, ownerID, item.SkuID).First(&mapping).Error; err != nil {
			return fmt.Errorf("%w: cost version does not authorize order SKU", ErrFinalProfitUnknown)
		}
		var totalLines, actualLines int64
		if err := tx.Model(&finalCostLine{}).Where("cost_version_id=?", cv.ID).Count(&totalLines).Error; err != nil {
			return err
		}
		if err := tx.Model(&finalCostLine{}).Where("cost_version_id=? AND truth_status=?", cv.ID, "actual").Count(&actualLines).Error; err != nil {
			return err
		}
		if totalLines == 0 || actualLines != totalLines {
			return fmt.Errorf("%w: estimated or quoted product cost cannot become final", ErrFinalProfitUnknown)
		}
		if item.Quantity <= 0 || cv.TotalMinor > math.MaxInt64/int64(item.Quantity) {
			return fmt.Errorf("invalid cost allocation")
		}
		amount := cv.TotalMinor * int64(item.Quantity)
		h := sha256.Sum256([]byte(fmt.Sprintf("%d|%d|%d|%d|%d|%s", ownerID, orderID, item.ID, cv.ID, amount, cv.TargetCurrency)))
		hash := hex.EncodeToString(h[:])
		q := tx.Where("owner_id=? AND order_item_id=?", ownerID, item.ID).First(&out)
		if q.Error == nil {
			if out.ContentSHA256 != hash {
				return fmt.Errorf("order cost allocation is already frozen")
			}
			return nil
		}
		if !errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return q.Error
		}
		out = OrderProductCostAllocation{OwnerID: ownerID, OrderID: orderID, OrderItemID: item.ID, SKUID: item.SkuID, SourcingCostVersionID: cv.ID, AmountMinor: amount, Currency: cv.TargetCurrency, ContentSHA256: hash}
		return tx.Create(&out).Error
	})
	return &out, err
}

func (s *Service) FinalizeOrderProfit(ctx context.Context, ownerID, orderID int64) (*OrderFinalProfitVersion, error) {
	var out OrderFinalProfitVersion
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := requireExternalOrderAuthority(tx, ownerID, orderID); err != nil {
			return err
		}
		var lines []finalSettlementLine
		if err := tx.Table("platform_settlement_fact_line AS l").Select("l.*, i.content_sha256 AS ingest_content_sha256").Joins("JOIN platform_settlement_ingest AS i ON i.id=l.ingest_id").Where("i.owner_id=? AND i.truth_status=? AND l.order_id=?", ownerID, "external_observed", orderID).Order("l.id").Scan(&lines).Error; err != nil {
			return err
		}
		var revenue, settlementFee, fulfillmentFee, settlementRefund int64
		hasSale, hasFulfillment := false, false
		currency := ""
		manifest := make([]string, 0)
		for _, line := range lines {
			if line.AmountMinor < 0 {
				return fmt.Errorf("%w: negative settlement fact", ErrFinalProfitUnknown)
			}
			if currency == "" {
				currency = line.Currency
			} else if currency != line.Currency {
				return fmt.Errorf("%w: mixed currency", ErrFinalProfitUnknown)
			}
			manifest = append(manifest, "settlement-line:"+fmt.Sprint(line.ID)+":"+line.IngestContentSHA256)
			switch line.Kind {
			case "sale":
				revenue += line.AmountMinor
				hasSale = true
			case "refund":
				settlementRefund += line.AmountMinor
			case "fee", "commission":
				if line.FeeCode == "" {
					return fmt.Errorf("%w: unclassified settlement fee", ErrFinalProfitUnknown)
				}
				if line.FeeCode == "fulfillment_fee" {
					fulfillmentFee += line.AmountMinor
					hasFulfillment = true
				} else {
					settlementFee += line.AmountMinor
				}
			default:
				return fmt.Errorf("%w: unsupported settlement kind", ErrFinalProfitUnknown)
			}
		}
		if !hasSale || !hasFulfillment || revenue <= 0 {
			return fmt.Errorf("%w: observed sale and classified fulfillment fee required", ErrFinalProfitUnknown)
		}
		var items []finalOrderItem
		if err := tx.Where("order_id=?", orderID).Find(&items).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return fmt.Errorf("%w: order lines missing", ErrFinalProfitUnknown)
		}
		var allocations []OrderProductCostAllocation
		if err := tx.Where("owner_id=? AND order_id=?", ownerID, orderID).Order("order_item_id").Find(&allocations).Error; err != nil {
			return err
		}
		if len(allocations) != len(items) {
			return fmt.Errorf("%w: every order line requires exact cost allocation", ErrFinalProfitUnknown)
		}
		productCost := int64(0)
		seen := map[int64]bool{}
		for _, a := range allocations {
			if a.Currency != currency || seen[a.OrderItemID] {
				return fmt.Errorf("%w: cost allocation currency or identity mismatch", ErrFinalProfitUnknown)
			}
			seen[a.OrderItemID] = true
			productCost += a.AmountMinor
			manifest = append(manifest, "cost:"+fmt.Sprint(a.ID)+":"+a.ContentSHA256)
		}
		for _, i := range items {
			if !seen[i.ID] {
				return fmt.Errorf("%w: order line cost missing", ErrFinalProfitUnknown)
			}
		}
		var resolutions []finalResolution
		if err := tx.Where("owner_id=? AND order_id=?", ownerID, orderID).Find(&resolutions).Error; err != nil {
			return err
		}
		refund := int64(0)
		for _, rc := range resolutions {
			if rc.Status != "succeeded" && rc.Status != "failed" && rc.Status != "rejected" {
				return fmt.Errorf("%w: aftersales is not terminal", ErrFinalProfitUnknown)
			}
			if rc.Status == "succeeded" || rc.Status == "failed" {
				var receipt finalReceipt
				if err := tx.Where("owner_id=? AND resolution_id=?", ownerID, rc.ID).First(&receipt).Error; err != nil {
					return fmt.Errorf("%w: terminal aftersales receipt missing", ErrFinalProfitUnknown)
				}
				if receipt.Currency != currency || (receipt.SourceType != "platform_receipt" && receipt.SourceType != "controlled_reconciliation") {
					return fmt.Errorf("%w: refund receipt mismatch", ErrFinalProfitUnknown)
				}
				if receipt.Outcome == "succeeded" {
					refund += receipt.ActualMinor
				}
				manifest = append(manifest, "refund:"+fmt.Sprint(receipt.ResolutionID))
			}
		}
		if refund != settlementRefund {
			return fmt.Errorf("%w: settlement refund and aftersales terminal receipts disagree", ErrFinalProfitUnknown)
		}
		total := productCost + settlementFee + fulfillmentFee + refund
		profitMinor := revenue - total
		sort.Strings(manifest)
		mh := sha256.Sum256([]byte(strings.Join(manifest, "|")))
		var latest int64
		if err := tx.Model(&OrderFinalProfitVersion{}).Where("owner_id=? AND order_id=?", ownerID, orderID).Select("COALESCE(MAX(version),0)").Scan(&latest).Error; err != nil {
			return err
		}
		out = OrderFinalProfitVersion{OwnerID: ownerID, OrderID: orderID, Version: latest + 1, Currency: currency, RevenueMinor: revenue, ProductCostMinor: productCost, SettlementFeeMinor: settlementFee, FulfillmentFeeMinor: fulfillmentFee, RefundMinor: refund, TotalCostMinor: total, ProfitMinor: profitMinor, SourceManifestSHA256: hex.EncodeToString(mh[:]), FinalizedAt: time.Now().UTC()}
		var existing OrderFinalProfitVersion
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("owner_id=? AND order_id=? AND source_manifest_sha256=?", ownerID, orderID, out.SourceManifestSHA256).First(&existing).Error; err == nil {
			out = existing
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(&out).Error
	})
	return &out, err
}

func (s *Service) ListFinalOrderProfitVersions(ctx context.Context, ownerID, orderID int64) ([]OrderFinalProfitVersion, error) {
	if err := requireExternalOrderAuthority(s.db.WithContext(ctx), ownerID, orderID); err != nil {
		return nil, err
	}
	var rows []OrderFinalProfitVersion
	err := s.db.WithContext(ctx).Where("owner_id=? AND order_id=?", ownerID, orderID).Order("version DESC").Find(&rows).Error
	return rows, err
}
