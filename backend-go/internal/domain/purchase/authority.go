package purchase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	AuthorityRequested         = "requested"
	AuthorityOwnerApproved     = "owner_approved"
	AuthorityExternalSubmitted = "external_submitted"
	AuthorityOrdered           = "ordered"
	AuthorityFailed            = "failed"
	AuthorityPartiallyReceived = "partially_received"
	AuthorityFullyReceived     = "fully_received"
)

var (
	ErrAuthorityInvalid  = errors.New("invalid purchase authority input")
	ErrAuthorityConflict = errors.New("purchase authority conflict")
	ErrLegacyWriteFrozen = errors.New("legacy purchase writes are frozen; use purchase authority endpoints")
)

type Authority struct {
	ID                  int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OwnerID             int64      `gorm:"column:owner_id;not null" json:"owner_id"`
	SupplierID          int64      `gorm:"column:supplier_id;not null" json:"supplier_id"`
	SKUMappingID        int64      `gorm:"column:sku_mapping_id;not null" json:"sku_mapping_id"`
	InternalSKUID       int64      `gorm:"column:internal_sku_id;not null" json:"internal_sku_id"`
	CostVersionID       int64      `gorm:"column:cost_version_id;not null" json:"cost_version_id"`
	InventoryID         int64      `gorm:"column:inventory_id;not null" json:"inventory_id"`
	Quantity            int        `gorm:"column:quantity;not null" json:"quantity"`
	UnitAmountMinor     int64      `gorm:"column:unit_amount_minor;not null" json:"unit_amount_minor"`
	TotalAmountMinor    int64      `gorm:"column:total_amount_minor;not null" json:"total_amount_minor"`
	Currency            string     `gorm:"column:currency;not null" json:"currency"`
	Status              string     `gorm:"column:status;not null" json:"status"`
	RequestSHA256       string     `gorm:"column:request_sha256;not null" json:"request_sha256"`
	IdempotencyKey      string     `gorm:"column:idempotency_key;not null" json:"-"`
	OwnerDecisionID     *int64     `gorm:"column:owner_decision_id" json:"owner_decision_id,omitempty"`
	ApprovedAt          *time.Time `gorm:"column:approved_at" json:"approved_at,omitempty"`
	ExternalSubmittedAt *time.Time `gorm:"column:external_submitted_at" json:"external_submitted_at,omitempty"`
	ExternalOrderedAt   *time.Time `gorm:"column:external_ordered_at" json:"external_ordered_at,omitempty"`
	ExternalFailedAt    *time.Time `gorm:"column:external_failed_at" json:"external_failed_at,omitempty"`
	ReceivedQuantity    int        `gorm:"column:received_quantity;not null" json:"received_quantity"`
	CreatedAt           time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Authority) TableName() string { return "purchase_authority" }

type ExternalFact struct {
	ID               int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OwnerID          int64           `gorm:"column:owner_id;not null" json:"owner_id"`
	PurchaseID       int64           `gorm:"column:purchase_id;not null" json:"purchase_id"`
	EventType        string          `gorm:"column:event_type;not null" json:"event_type"`
	ExternalEventID  string          `gorm:"column:external_event_id;not null" json:"external_event_id"`
	ExternalOrderID  string          `gorm:"column:external_order_id;not null" json:"external_order_id"`
	ReceivedQuantity int             `gorm:"column:received_quantity;not null" json:"received_quantity"`
	TruthStatus      string          `gorm:"column:truth_status;not null" json:"truth_status"`
	RawPayload       json.RawMessage `gorm:"column:raw_payload;type:jsonb;not null" json:"-"`
	PayloadSHA256    string          `gorm:"column:payload_sha256;not null" json:"payload_sha256"`
	ObservedAt       time.Time       `gorm:"column:observed_at;not null" json:"observed_at"`
	CreatedAt        time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (ExternalFact) TableName() string { return "purchase_external_fact" }

type InventoryReceiptLedger struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OwnerID        int64     `gorm:"column:owner_id;not null" json:"owner_id"`
	PurchaseID     int64     `gorm:"column:purchase_id;not null" json:"purchase_id"`
	ExternalFactID int64     `gorm:"column:external_fact_id;not null" json:"external_fact_id"`
	InventoryID    int64     `gorm:"column:inventory_id;not null" json:"inventory_id"`
	SKUID          int64     `gorm:"column:sku_id;not null" json:"sku_id"`
	Quantity       int       `gorm:"column:quantity;not null" json:"quantity"`
	BeforeQuantity int       `gorm:"column:before_quantity;not null" json:"before_quantity"`
	AfterQuantity  int       `gorm:"column:after_quantity;not null" json:"after_quantity"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (InventoryReceiptLedger) TableName() string { return "purchase_inventory_receipt_ledger" }

type CreateAuthorityInput struct {
	SupplierID     int64  `json:"supplier_id"`
	SKUMappingID   int64  `json:"sku_mapping_id"`
	CostVersionID  int64  `json:"cost_version_id"`
	InventoryID    int64  `json:"inventory_id"`
	Quantity       int    `json:"quantity"`
	IdempotencyKey string `json:"idempotency_key"`
}
type ApproveAuthorityInput struct {
	OwnerDecisionID int64 `json:"owner_decision_id"`
}
type ExternalFactInput struct {
	ExternalEventID  string          `json:"external_event_id"`
	ExternalOrderID  string          `json:"external_order_id"`
	ReceivedQuantity int             `json:"received_quantity"`
	RawPayload       json.RawMessage `json:"raw_payload"`
	ObservedAt       time.Time       `json:"observed_at"`
}
type AuthorityDetail struct {
	Authority Authority                `json:"purchase"`
	Facts     []ExternalFact           `json:"external_facts"`
	Ledger    []InventoryReceiptLedger `json:"inventory_ledger"`
}

type authoritySupplier struct {
	ID, OwnerID int64
	Status      int16
	TruthStatus string
}

func (authoritySupplier) TableName() string { return "supplier" }

type authorityMapping struct{ ID, OwnerID, InternalSKUID int64 }

func (authorityMapping) TableName() string { return "sourcing_sku_mapping" }

type authorityCost struct {
	ID, OwnerID, SKUMappingID int64
	TargetCurrency            string
}

func (authorityCost) TableName() string { return "sourcing_cost_version" }

type authorityCostLine struct {
	CostType              string
	NormalizedAmountMinor int64
	TruthStatus           string
}

func (authorityCostLine) TableName() string { return "sourcing_cost_line" }

type authorityInventory struct {
	ID, SkuID int64
	Quantity  int
}

func (authorityInventory) TableName() string { return "inventory" }

type authorityDecision struct {
	ID, OwnerID                                                            int64
	Decision, CapabilityID, CommandType, TargetType, TargetID, InputSHA256 string
}

func (authorityDecision) TableName() string { return "business_owner_decision" }

func authorityDigest(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func (s *Service) CreateAuthority(ctx context.Context, owner int64, in CreateAuthorityInput) (*Authority, error) {
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if owner <= 0 || in.SupplierID <= 0 || in.SKUMappingID <= 0 || in.CostVersionID <= 0 || in.InventoryID <= 0 || in.Quantity <= 0 || in.IdempotencyKey == "" {
		return nil, ErrAuthorityInvalid
	}
	var out Authority
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var old Authority
		if e := tx.Where("owner_id=? AND idempotency_key=?", owner, in.IdempotencyKey).Take(&old).Error; e == nil {
			if old.SupplierID != in.SupplierID || old.SKUMappingID != in.SKUMappingID || old.CostVersionID != in.CostVersionID || old.InventoryID != in.InventoryID || old.Quantity != in.Quantity {
				return ErrAuthorityConflict
			}
			out = old
			return nil
		} else if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		var supplier authoritySupplier
		if e := tx.Where("id=? AND owner_id=?", in.SupplierID, owner).Take(&supplier).Error; e != nil || supplier.Status != 1 || (supplier.TruthStatus != "quoted" && supplier.TruthStatus != "actual") {
			return ErrAuthorityInvalid
		}
		var mapping authorityMapping
		if e := tx.Where("id=? AND owner_id=?", in.SKUMappingID, owner).Take(&mapping).Error; e != nil {
			return ErrAuthorityInvalid
		}
		var cost authorityCost
		if e := tx.Where("id=? AND owner_id=? AND sku_mapping_id=?", in.CostVersionID, owner, in.SKUMappingID).Take(&cost).Error; e != nil {
			return ErrAuthorityInvalid
		}
		var purchaseLine authorityCostLine
		if e := tx.Where("cost_version_id=? AND cost_type='purchase'", cost.ID).Take(&purchaseLine).Error; e != nil || (purchaseLine.TruthStatus != "quoted" && purchaseLine.TruthStatus != "actual") || purchaseLine.NormalizedAmountMinor < 0 {
			return ErrAuthorityInvalid
		}
		var inventory authorityInventory
		if e := tx.Where("id=? AND sku_id=?", in.InventoryID, mapping.InternalSKUID).Take(&inventory).Error; e != nil {
			return ErrAuthorityInvalid
		}
		if purchaseLine.NormalizedAmountMinor > 0 && int64(in.Quantity) > (int64(^uint64(0)>>1)/purchaseLine.NormalizedAmountMinor) {
			return ErrAuthorityInvalid
		}
		out = Authority{OwnerID: owner, SupplierID: in.SupplierID, SKUMappingID: in.SKUMappingID, InternalSKUID: mapping.InternalSKUID, CostVersionID: cost.ID, InventoryID: inventory.ID, Quantity: in.Quantity, UnitAmountMinor: purchaseLine.NormalizedAmountMinor, TotalAmountMinor: purchaseLine.NormalizedAmountMinor * int64(in.Quantity), Currency: cost.TargetCurrency, Status: AuthorityRequested, IdempotencyKey: in.IdempotencyKey}
		out.RequestSHA256 = authorityDigest(struct {
			OwnerID, SupplierID, SKUMappingID, CostVersionID, InventoryID int64
			Quantity                                                      int
			UnitAmountMinor, TotalAmountMinor                             int64
			Currency                                                      string
		}{owner, out.SupplierID, out.SKUMappingID, out.CostVersionID, out.InventoryID, out.Quantity, out.UnitAmountMinor, out.TotalAmountMinor, out.Currency})
		return tx.Create(&out).Error
	})
	return &out, err
}

func (s *Service) ApproveAuthority(ctx context.Context, owner, id int64, in ApproveAuthorityInput) (*Authority, error) {
	var out Authority
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND owner_id=?", id, owner).Take(&out).Error; e != nil {
			return e
		}
		if out.Status == AuthorityOwnerApproved && out.OwnerDecisionID != nil && *out.OwnerDecisionID == in.OwnerDecisionID {
			return nil
		}
		if out.Status != AuthorityRequested || in.OwnerDecisionID <= 0 {
			return ErrAuthorityConflict
		}
		var d authorityDecision
		if e := tx.Where("id=? AND owner_id=?", in.OwnerDecisionID, owner).Take(&d).Error; e != nil || d.Decision != "selected" || d.CapabilityID != "purchase.authority.execute" || d.CommandType != "purchase.submit" || d.TargetType != "purchase_authority" || d.TargetID != strconv.FormatInt(id, 10) || d.InputSHA256 != out.RequestSHA256 {
			return ErrAuthorityInvalid
		}
		now := time.Now().UTC()
		out.Status, out.OwnerDecisionID, out.ApprovedAt = AuthorityOwnerApproved, &d.ID, &now
		return tx.Model(&Authority{}).Where("id=? AND owner_id=? AND status=?", id, owner, AuthorityRequested).Updates(map[string]any{"status": out.Status, "owner_decision_id": d.ID, "approved_at": now}).Error
	})
	return &out, err
}

func (s *Service) RecordExternalFact(ctx context.Context, owner, id int64, eventType string, in ExternalFactInput) (*AuthorityDetail, error) {
	eventType = strings.TrimSpace(eventType)
	in.ExternalEventID = strings.TrimSpace(in.ExternalEventID)
	in.ExternalOrderID = strings.TrimSpace(in.ExternalOrderID)
	if owner <= 0 || id <= 0 || in.ExternalEventID == "" || in.ExternalOrderID == "" || len(in.RawPayload) == 0 || !json.Valid(in.RawPayload) || in.ObservedAt.IsZero() || in.ObservedAt.After(time.Now().Add(5*time.Minute)) {
		return nil, ErrAuthorityInvalid
	}
	if eventType != "submitted" && eventType != "ordered" && eventType != "failed" && eventType != "received" {
		return nil, ErrAuthorityInvalid
	}
	if (eventType == "received") != (in.ReceivedQuantity > 0) {
		return nil, ErrAuthorityInvalid
	}
	h := sha256.Sum256(in.RawPayload)
	hash := hex.EncodeToString(h[:])
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p Authority
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND owner_id=?", id, owner).Take(&p).Error; e != nil {
			return e
		}
		var old ExternalFact
		if e := tx.Where("owner_id=? AND external_event_id=?", owner, in.ExternalEventID).Take(&old).Error; e == nil {
			if old.PurchaseID != id || old.EventType != eventType || old.PayloadSHA256 != hash {
				return ErrAuthorityConflict
			}
			return nil
		} else if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		var prior ExternalFact
		if e := tx.Where("purchase_id=? AND owner_id=?", id, owner).Order("id").Take(&prior).Error; e == nil && prior.ExternalOrderID != in.ExternalOrderID {
			return ErrAuthorityConflict
		} else if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		allowed := (eventType == "submitted" && p.Status == AuthorityOwnerApproved) || ((eventType == "ordered" || eventType == "failed") && p.Status == AuthorityExternalSubmitted) || (eventType == "received" && (p.Status == AuthorityOrdered || p.Status == AuthorityPartiallyReceived))
		if !allowed {
			return ErrAuthorityConflict
		}
		fact := ExternalFact{OwnerID: owner, PurchaseID: id, EventType: eventType, ExternalEventID: in.ExternalEventID, ExternalOrderID: in.ExternalOrderID, ReceivedQuantity: in.ReceivedQuantity, TruthStatus: "external_observed", RawPayload: append(json.RawMessage(nil), in.RawPayload...), PayloadSHA256: hash, ObservedAt: in.ObservedAt.UTC()}
		if e := tx.Create(&fact).Error; e != nil {
			return e
		}
		now := in.ObservedAt.UTC()
		updates := map[string]any{}
		switch eventType {
		case "submitted":
			p.Status = AuthorityExternalSubmitted
			updates["external_submitted_at"] = now
		case "ordered":
			p.Status = AuthorityOrdered
			updates["external_ordered_at"] = now
		case "failed":
			p.Status = AuthorityFailed
			updates["external_failed_at"] = now
		case "received":
			if p.ReceivedQuantity+in.ReceivedQuantity > p.Quantity {
				return ErrAuthorityInvalid
			}
			var inv authorityInventory
			if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND sku_id=?", p.InventoryID, p.InternalSKUID).Take(&inv).Error; e != nil {
				return e
			}
			ledger := InventoryReceiptLedger{OwnerID: owner, PurchaseID: id, ExternalFactID: fact.ID, InventoryID: inv.ID, SKUID: p.InternalSKUID, Quantity: in.ReceivedQuantity, BeforeQuantity: inv.Quantity, AfterQuantity: inv.Quantity + in.ReceivedQuantity}
			if e := tx.Model(&authorityInventory{}).Where("id=? AND quantity=?", inv.ID, inv.Quantity).Update("quantity", ledger.AfterQuantity).Error; e != nil {
				return e
			}
			if e := tx.Create(&ledger).Error; e != nil {
				return e
			}
			p.ReceivedQuantity += in.ReceivedQuantity
			if p.ReceivedQuantity == p.Quantity {
				p.Status = AuthorityFullyReceived
			} else {
				p.Status = AuthorityPartiallyReceived
			}
			updates["received_quantity"] = p.ReceivedQuantity
		}
		updates["status"] = p.Status
		return tx.Model(&Authority{}).Where("id=?", id).Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetAuthority(ctx, owner, id)
}

func (s *Service) GetAuthority(ctx context.Context, owner, id int64) (*AuthorityDetail, error) {
	var d AuthorityDetail
	if e := s.db.WithContext(ctx).Where("id=? AND owner_id=?", id, owner).Take(&d.Authority).Error; e != nil {
		return nil, e
	}
	if e := s.db.WithContext(ctx).Where("purchase_id=? AND owner_id=?", id, owner).Order("id").Find(&d.Facts).Error; e != nil {
		return nil, e
	}
	if e := s.db.WithContext(ctx).Where("purchase_id=? AND owner_id=?", id, owner).Order("id").Find(&d.Ledger).Error; e != nil {
		return nil, e
	}
	return &d, nil
}

func (s *Service) ListAuthorities(ctx context.Context, owner int64) ([]Authority, error) {
	if owner <= 0 {
		return nil, ErrAuthorityInvalid
	}
	var rows []Authority
	return rows, s.db.WithContext(ctx).Where("owner_id=?", owner).Order("id DESC").Find(&rows).Error
}

func (s *Service) LegacyWriteFrozen() error { return ErrLegacyWriteFrozen }
