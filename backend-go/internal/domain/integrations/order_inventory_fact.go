package integrations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	OrderActionReserve = "reserve"
	OrderActionCommit  = "commit"
	OrderActionRelease = "release"
)

type PlatformOrderIngest struct {
	ID                int64           `gorm:"column:id;primaryKey;autoIncrement"`
	OwnerID           int64           `gorm:"column:owner_id;not null;uniqueIndex:uq_order_event,priority:1"`
	AccountID         int64           `gorm:"column:account_id;not null;uniqueIndex:uq_order_event,priority:2"`
	PlatformCode      string          `gorm:"column:platform_code;not null"`
	ExternalEventID   string          `gorm:"column:external_event_id;not null;uniqueIndex:uq_order_event,priority:3"`
	ExternalOrderID   string          `gorm:"column:external_order_id;not null;index"`
	EventAction       string          `gorm:"column:event_action;not null"`
	TruthStatus       string          `gorm:"column:truth_status;not null"`
	RawPayload        json.RawMessage `gorm:"column:raw_payload;type:jsonb;not null"`
	PayloadSHA256     string          `gorm:"column:payload_sha256;size:64;not null"`
	ObservedAt        time.Time       `gorm:"column:observed_at;not null"`
	NormalizedOrderID *int64          `gorm:"column:normalized_order_id"`
	ProcessingStatus  string          `gorm:"column:processing_status;not null"`
	ErrorMessage      string          `gorm:"column:error_message;not null"`
	CreatedAt         time.Time       `gorm:"column:created_at;autoCreateTime"`
}

func (PlatformOrderIngest) TableName() string { return "platform_order_ingest" }

type PlatformOrderIngestItem struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement"`
	IngestID        int64     `gorm:"column:ingest_id;not null;uniqueIndex:uq_ingest_line,priority:1"`
	LineNumber      int       `gorm:"column:line_number;not null;uniqueIndex:uq_ingest_line,priority:2"`
	ExternalSKUCode string    `gorm:"column:external_sku_code;not null"`
	SkuID           int64     `gorm:"column:sku_id;not null"`
	Quantity        int       `gorm:"column:quantity;not null"`
	UnitPriceMinor  int64     `gorm:"column:unit_price_minor;not null"`
	Currency        string    `gorm:"column:currency;size:3;not null"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (PlatformOrderIngestItem) TableName() string { return "platform_order_ingest_item" }

type OrderInventoryLedger struct {
	ID                   int64     `gorm:"column:id;primaryKey;autoIncrement"`
	OwnerID              int64     `gorm:"column:owner_id;not null"`
	IngestID             int64     `gorm:"column:ingest_id;not null;uniqueIndex:uq_ingest_line_action,priority:1"`
	OrderID              int64     `gorm:"column:order_id;not null"`
	OrderItemID          int64     `gorm:"column:order_item_id;not null;uniqueIndex:uq_ingest_line_action,priority:2"`
	InventoryID          int64     `gorm:"column:inventory_id;not null"`
	SkuID                int64     `gorm:"column:sku_id;not null"`
	Action               string    `gorm:"column:action;not null;uniqueIndex:uq_ingest_line_action,priority:3"`
	Quantity             int       `gorm:"column:quantity;not null"`
	BeforeQuantity       int       `gorm:"column:before_quantity;not null"`
	AfterQuantity        int       `gorm:"column:after_quantity;not null"`
	BeforeLockedQuantity int       `gorm:"column:before_locked_quantity;not null"`
	AfterLockedQuantity  int       `gorm:"column:after_locked_quantity;not null"`
	CreatedAt            time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (OrderInventoryLedger) TableName() string { return "order_inventory_ledger" }

type ExternalOrderLine struct {
	SKUCode        string `json:"sku_code"`
	Quantity       int    `json:"quantity"`
	UnitPriceMinor int64  `json:"unit_price_minor"`
	Currency       string `json:"currency"`
}

type IngestExternalOrderInput struct {
	OwnerID         int64               `json:"-"`
	AccountID       int64               `json:"-"`
	PlatformCode    string              `json:"platform_code" binding:"required"`
	ExternalEventID string              `json:"external_event_id" binding:"required"`
	ExternalOrderID string              `json:"external_order_id" binding:"required"`
	Action          string              `json:"action" binding:"required"`
	TruthStatus     string              `json:"truth_status" binding:"required"`
	ObservedAt      time.Time           `json:"observed_at" binding:"required"`
	RawPayload      json.RawMessage     `json:"raw_payload" binding:"required"`
	Status          string              `json:"status"`
	Lines           []ExternalOrderLine `json:"lines" binding:"required,min=1"`
}

type IngestExternalOrderResult struct {
	IngestID int64
	OrderID  int64
	Replay   bool
}

// IngestExternalOrder deterministically persists one immutable external event,
// normalizes its lines and applies exactly one reversible inventory action.
func (s *Service) IngestExternalOrder(ctx context.Context, in IngestExternalOrderInput) (*IngestExternalOrderResult, error) {
	if in.OwnerID <= 0 || in.AccountID <= 0 {
		return nil, errors.New("owner and platform account are required")
	}
	in.PlatformCode = strings.TrimSpace(in.PlatformCode)
	in.ExternalEventID = strings.TrimSpace(in.ExternalEventID)
	in.ExternalOrderID = strings.TrimSpace(in.ExternalOrderID)
	if in.PlatformCode == "" || in.ExternalEventID == "" || in.ExternalOrderID == "" || len(in.RawPayload) == 0 {
		return nil, errors.New("platform, external event/order identity and raw payload are required")
	}
	if in.Action != OrderActionReserve && in.Action != OrderActionCommit && in.Action != OrderActionRelease {
		return nil, fmt.Errorf("unsupported inventory action %q", in.Action)
	}
	if in.TruthStatus != "external_observed" && in.TruthStatus != "mock" {
		return nil, fmt.Errorf("unsupported truth status %q", in.TruthStatus)
	}
	if in.ObservedAt.IsZero() || len(in.Lines) == 0 {
		return nil, errors.New("observed_at and order lines are required")
	}
	sum := sha256.Sum256(in.RawPayload)
	digest := hex.EncodeToString(sum[:])
	result := &IngestExternalOrderResult{}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var account PlatformIntegrationAccount
		if err := tx.Select("id", "platform_id").First(&account, in.AccountID).Error; err != nil {
			return fmt.Errorf("platform account not found: %w", err)
		}
		var accountAuthority int64
		if err := tx.Table("owner_platform_account_authority").Select("account_id").Where("owner_id = ? AND account_id = ? AND platform_code = ?", in.OwnerID, in.AccountID, in.PlatformCode).Take(&accountAuthority).Error; err != nil {
			return fmt.Errorf("platform account is not verified for owner: %w", err)
		}

		var existing PlatformOrderIngest
		err := tx.Where("owner_id = ? AND account_id = ? AND external_event_id = ?", in.OwnerID, in.AccountID, in.ExternalEventID).First(&existing).Error
		if err == nil {
			if existing.PayloadSHA256 != digest || existing.ExternalOrderID != in.ExternalOrderID || existing.EventAction != in.Action {
				return errors.New("external event id was reused with different content")
			}
			if existing.ProcessingStatus != "applied" || existing.NormalizedOrderID == nil {
				return errors.New("existing external event is not safely applied")
			}
			result.IngestID, result.OrderID, result.Replay = existing.ID, *existing.NormalizedOrderID, true
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		ingest := PlatformOrderIngest{OwnerID: in.OwnerID, AccountID: in.AccountID, PlatformCode: in.PlatformCode, ExternalEventID: in.ExternalEventID, ExternalOrderID: in.ExternalOrderID, EventAction: in.Action, TruthStatus: in.TruthStatus, RawPayload: append(json.RawMessage(nil), in.RawPayload...), PayloadSHA256: digest, ObservedAt: in.ObservedAt, ProcessingStatus: "received", ErrorMessage: ""}
		if err := tx.Create(&ingest).Error; err != nil {
			return err
		}

		type skuRow struct {
			ID, ProductID int64
			Code          string
		}
		skus := make([]skuRow, len(in.Lines))
		seen := map[string]struct{}{}
		for i, line := range in.Lines {
			line.SKUCode = strings.TrimSpace(line.SKUCode)
			line.Currency = strings.ToUpper(strings.TrimSpace(line.Currency))
			if line.SKUCode == "" || line.Quantity <= 0 || line.UnitPriceMinor < 0 || len(line.Currency) != 3 {
				return fmt.Errorf("invalid order line %d", i+1)
			}
			if _, duplicate := seen[line.SKUCode]; duplicate {
				return fmt.Errorf("duplicate sku %q in event", line.SKUCode)
			}
			seen[line.SKUCode] = struct{}{}
			if err := tx.Table("sku").Select("id, product_id, code").Where("code = ?", line.SKUCode).Take(&skus[i]).Error; err != nil {
				return fmt.Errorf("unknown sku %q: %w", line.SKUCode, err)
			}
			var authorizedSKU int64
			if err := tx.Table("sourcing_sku_mapping").Select("internal_sku_id").Where("owner_id = ? AND internal_sku_id = ?", in.OwnerID, skus[i].ID).Take(&authorizedSKU).Error; err != nil {
				return fmt.Errorf("sku %q is not authoritative for owner: %w", line.SKUCode, err)
			}
			factLine := PlatformOrderIngestItem{IngestID: ingest.ID, LineNumber: i + 1, ExternalSKUCode: line.SKUCode, SkuID: skus[i].ID, Quantity: line.Quantity, UnitPriceMinor: line.UnitPriceMinor, Currency: line.Currency}
			if err := tx.Create(&factLine).Error; err != nil {
				return err
			}
		}

		var orderID int64
		if in.Action == OrderActionReserve {
			var count int64
			if err := tx.Table("platform_order_ingest").Where("owner_id = ? AND account_id = ? AND external_order_id = ? AND event_action = ? AND id <> ?", in.OwnerID, in.AccountID, in.ExternalOrderID, OrderActionReserve, ingest.ID).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return errors.New("external order already has a reserve event")
			}
			order := map[string]any{"order_no": fmt.Sprintf("%s:%d:%s", in.PlatformCode, in.AccountID, in.ExternalOrderID), "platform_id": account.PlatformID, "status": in.Status}
			if strings.TrimSpace(in.Status) == "" {
				order["status"] = "pending"
			}
			if err := tx.Table("sales_order").Create(order).Error; err != nil {
				return err
			}
			if err := tx.Table("sales_order").Select("id").Where("order_no = ?", order["order_no"]).Take(&orderID).Error; err != nil {
				return err
			}
			for i, line := range in.Lines {
				var productName string
				if err := tx.Table("product").Select("name").Where("id = ?", skus[i].ProductID).Take(&productName).Error; err != nil {
					return fmt.Errorf("sku product missing: %w", err)
				}
				// Legacy sales_order_item money fields are only a display projection.
				// The immutable minor-unit ingest line remains the exact authority.
				projectedUnitPrice := float64(line.UnitPriceMinor) / 100
				row := map[string]any{"order_id": orderID, "sku_id": skus[i].ID, "product_id": skus[i].ProductID, "product_name": productName, "sku_code": skus[i].Code, "unit_price": projectedUnitPrice, "quantity": line.Quantity, "subtotal": projectedUnitPrice * float64(line.Quantity)}
				if err := tx.Table("sales_order_item").Create(row).Error; err != nil {
					return err
				}
			}
		} else {
			var prior PlatformOrderIngest
			if err := tx.Where("owner_id = ? AND account_id = ? AND external_order_id = ? AND event_action = ? AND processing_status = ?", in.OwnerID, in.AccountID, in.ExternalOrderID, OrderActionReserve, "applied").First(&prior).Error; err != nil {
				return errors.New("order has no applied reserve event")
			}
			if prior.NormalizedOrderID == nil {
				return errors.New("reserve event has no normalized order")
			}
			orderID = *prior.NormalizedOrderID
		}

		var orderItems []struct {
			ID, SkuID int64
			Quantity  int
		}
		if err := tx.Table("sales_order_item").Select("id, sku_id, quantity").Where("order_id = ?", orderID).Order("id").Scan(&orderItems).Error; err != nil {
			return err
		}
		if len(orderItems) != len(in.Lines) {
			return errors.New("event lines do not match normalized order")
		}
		bySKU := make(map[int64]struct {
			ID       int64
			Quantity int
		}, len(orderItems))
		for _, item := range orderItems {
			bySKU[item.SkuID] = struct {
				ID       int64
				Quantity int
			}{item.ID, item.Quantity}
		}
		for i, line := range in.Lines {
			item, ok := bySKU[skus[i].ID]
			if !ok || item.Quantity != line.Quantity {
				return fmt.Errorf("event line for sku %q differs from normalized order", line.SKUCode)
			}
			var inv struct {
				ID                       int64
				Quantity, LockedQuantity int
			}
			if err := tx.Table("inventory").Clauses(clause.Locking{Strength: "UPDATE"}).Select("id, quantity, locked_quantity").Where("sku_id = ?", skus[i].ID).Take(&inv).Error; err != nil {
				return fmt.Errorf("inventory missing for sku %q: %w", line.SKUCode, err)
			}
			beforeQty, beforeLocked := inv.Quantity, inv.LockedQuantity
			switch in.Action {
			case OrderActionReserve:
				if inv.Quantity-inv.LockedQuantity < line.Quantity {
					return fmt.Errorf("insufficient available inventory for sku %q", line.SKUCode)
				}
				inv.LockedQuantity += line.Quantity
			case OrderActionCommit:
				if inv.LockedQuantity < line.Quantity || inv.Quantity < line.Quantity {
					return fmt.Errorf("insufficient reserved inventory for sku %q", line.SKUCode)
				}
				inv.LockedQuantity -= line.Quantity
				inv.Quantity -= line.Quantity
			case OrderActionRelease:
				if inv.LockedQuantity < line.Quantity {
					return fmt.Errorf("insufficient reserved inventory to release sku %q", line.SKUCode)
				}
				inv.LockedQuantity -= line.Quantity
			}
			if err := tx.Table("inventory").Where("id = ?", inv.ID).Updates(map[string]any{"quantity": inv.Quantity, "locked_quantity": inv.LockedQuantity, "updated_at": time.Now()}).Error; err != nil {
				return err
			}
			ledger := OrderInventoryLedger{OwnerID: in.OwnerID, IngestID: ingest.ID, OrderID: orderID, OrderItemID: item.ID, InventoryID: inv.ID, SkuID: skus[i].ID, Action: in.Action, Quantity: line.Quantity, BeforeQuantity: beforeQty, AfterQuantity: inv.Quantity, BeforeLockedQuantity: beforeLocked, AfterLockedQuantity: inv.LockedQuantity}
			if err := tx.Create(&ledger).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&ingest).Updates(map[string]any{"normalized_order_id": orderID, "processing_status": "applied"}).Error; err != nil {
			return err
		}
		result.IngestID, result.OrderID = ingest.ID, orderID
		return nil
	})
	return result, err
}
