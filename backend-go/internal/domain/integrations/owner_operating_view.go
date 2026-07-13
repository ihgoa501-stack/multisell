package integrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// OwnerOperatingView is the deliberately narrow, PII-free projection exposed
// to Owner-facing agents. Raw platform, carrier, settlement and bank payloads
// are never included.
type OwnerOperatingView struct {
	Order       OwnerOrderFact        `json:"order"`
	Inventory   []OwnerInventoryFact  `json:"inventory"`
	Fulfillment []OwnerCarrierFact    `json:"fulfillment"`
	Aftersales  []OwnerAftersaleFact  `json:"aftersales"`
	Settlements []OwnerSettlementFact `json:"settlements"`
	Profit      *OwnerProfitFact      `json:"profit,omitempty"`
	Cash        []OwnerCashFact       `json:"cash"`
	Evidence    []OwnerFactEvidence   `json:"evidence"`
	Unknowns    []string              `json:"unknowns"`
	Blockers    []string              `json:"blockers"`
	AsOf        time.Time             `json:"as_of"`
}

type OwnerOrderFact struct {
	OrderID          int64     `json:"order_id"`
	IngestID         int64     `json:"ingest_id"`
	AccountID        int64     `json:"account_id"`
	PlatformCode     string    `json:"platform_code"`
	EventAction      string    `json:"event_action"`
	TruthStatus      string    `json:"truth_status"`
	ProcessingStatus string    `json:"processing_status"`
	PayloadSHA256    string    `json:"payload_sha256"`
	ObservedAt       time.Time `json:"observed_at"`
}
type OwnerInventoryFact struct {
	ID                   int64     `json:"id"`
	IngestID             int64     `json:"ingest_id"`
	OrderItemID          int64     `json:"order_item_id"`
	InventoryID          int64     `json:"inventory_id"`
	SKUID                int64     `json:"sku_id"`
	Action               string    `json:"action"`
	Quantity             int       `json:"quantity"`
	BeforeQuantity       int       `json:"before_quantity"`
	AfterQuantity        int       `json:"after_quantity"`
	BeforeLockedQuantity int       `json:"before_locked_quantity"`
	AfterLockedQuantity  int       `json:"after_locked_quantity"`
	CreatedAt            time.Time `json:"created_at"`
}
type OwnerCarrierFact struct {
	ID            int64     `json:"id"`
	TrackingID    string    `json:"tracking_id"`
	SourceSystem  string    `json:"source_system"`
	Status        string    `json:"status"`
	TruthStatus   string    `json:"truth_status"`
	PayloadSHA256 string    `json:"payload_sha256"`
	OccurredAt    time.Time `json:"occurred_at"`
	ObservedAt    time.Time `json:"observed_at"`
}
type OwnerAftersaleFact struct {
	ID                int64                  `json:"id"`
	Kind              string                 `json:"kind"`
	Status            string                 `json:"status"`
	Currency          string                 `json:"currency"`
	RequestSource     string                 `json:"request_source"`
	RequestEvidenceID string                 `json:"request_evidence_id"`
	RequestedMinor    int64                  `json:"requested_minor"`
	RequestObservedAt time.Time              `json:"request_observed_at"`
	Receipt           *OwnerAftersaleReceipt `gorm:"-" json:"receipt,omitempty"`
}
type OwnerAftersaleReceipt struct {
	ID                int64     `json:"id"`
	Outcome           string    `json:"outcome"`
	SourceType        string    `json:"source_type"`
	EvidenceID        string    `json:"evidence_id"`
	ExternalReceiptID string    `json:"external_receipt_id"`
	Currency          string    `json:"currency"`
	ReceiptSHA256     string    `json:"receipt_sha256"`
	ActualMinor       int64     `json:"actual_minor"`
	ObservedAt        time.Time `json:"observed_at"`
}
type OwnerSettlementFact struct {
	ID            int64                 `json:"id"`
	AccountID     int64                 `json:"account_id"`
	PlatformCode  string                `json:"platform_code"`
	TruthStatus   string                `json:"truth_status"`
	Currency      string                `json:"currency"`
	PayloadSHA256 string                `json:"payload_sha256"`
	ContentSHA256 string                `json:"content_sha256"`
	ObservedAt    time.Time             `json:"observed_at"`
	Lines         []OwnerSettlementLine `gorm:"-" json:"lines"`
}
type OwnerSettlementLine struct {
	ID          int64     `json:"id"`
	Kind        string    `json:"kind"`
	FeeCode     string    `json:"fee_code"`
	Currency    string    `json:"currency"`
	AmountMinor int64     `json:"amount_minor"`
	OccurredAt  time.Time `json:"occurred_at"`
}
type OwnerProfitFact struct {
	ID                   int64     `json:"id"`
	Version              int64     `json:"version"`
	RevenueMinor         int64     `json:"revenue_minor"`
	ProductCostMinor     int64     `json:"product_cost_minor"`
	SettlementFeeMinor   int64     `json:"settlement_fee_minor"`
	FulfillmentFeeMinor  int64     `json:"fulfillment_fee_minor"`
	RefundMinor          int64     `json:"refund_minor"`
	TotalCostMinor       int64     `json:"total_cost_minor"`
	ProfitMinor          int64     `json:"profit_minor"`
	Currency             string    `json:"currency"`
	SourceManifestSHA256 string    `json:"source_manifest_sha256"`
	FinalizedAt          time.Time `json:"finalized_at"`
}
type OwnerCashFact struct {
	ID                      int64      `json:"id"`
	CashReceiptID           int64      `json:"cash_receipt_id"`
	SettlementIngestID      int64      `json:"settlement_ingest_id"`
	AmountMinor             int64      `json:"amount_minor"`
	ExpectedReceivableMinor int64      `json:"expected_receivable_minor"`
	Currency                string     `json:"currency"`
	Status                  string     `json:"status"`
	RequestSHA256           string     `json:"request_sha256"`
	ReconciledAt            *time.Time `json:"reconciled_at,omitempty"`
}
type OwnerFactEvidence struct {
	SourceType  string    `json:"source_type"`
	SourceID    int64     `json:"source_id"`
	TruthStatus string    `json:"truth_status"`
	SHA256      string    `json:"sha256"`
	ObservedAt  time.Time `json:"observed_at"`
	Summary     string    `json:"summary"`
}

// OwnerFactOptions is a PII-free selector projection for Owner workspaces.
// Every row is constrained by owner_id and only authoritative facts are exposed.
type OwnerFactOptions struct {
	Accounts        []OwnerAccountOption        `json:"accounts"`
	Orders          []OwnerOrderOption          `json:"orders"`
	OrderItems      []OwnerOrderItemOption      `json:"order_items"`
	CostVersions    []OwnerCostVersionOption    `json:"cost_versions"`
	Settlements     []OwnerSettlementOption     `json:"settlements"`
	CashReceipts    []OwnerCashOption           `json:"cash_receipts"`
	FinanceAccounts []OwnerFinanceAccountOption `json:"finance_accounts"`
	AftersalesCases []OwnerAftersalesCaseOption `json:"aftersales_cases"`
}
type OwnerAccountOption struct {
	ID           int64  `json:"id"`
	PlatformCode string `json:"platform_code"`
	StoreName    string `json:"store_name"`
}
type OwnerOrderOption struct {
	ID              int64     `json:"id"`
	AccountID       int64     `json:"account_id"`
	ExternalOrderID string    `json:"external_order_id"`
	PlatformCode    string    `json:"platform_code"`
	Currency        string    `json:"currency"`
	ObservedAt      time.Time `json:"observed_at"`
}
type OwnerOrderItemOption struct {
	ID          int64  `json:"id"`
	OrderID     int64  `json:"order_id"`
	SKUID       int64  `json:"sku_id"`
	SKUCode     string `json:"sku_code"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
}
type OwnerCostVersionOption struct {
	ID            int64  `json:"id"`
	InternalSKUID int64  `json:"internal_sku_id"`
	Version       int64  `json:"version"`
	TotalMinor    int64  `json:"total_minor"`
	Currency      string `json:"currency"`
}
type OwnerSettlementOption struct {
	ID                   int64     `json:"id"`
	AccountID            int64     `json:"account_id"`
	ReceivableMinor      int64     `json:"receivable_minor"`
	ExternalSettlementID string    `json:"external_settlement_id"`
	PlatformCode         string    `json:"platform_code"`
	Currency             string    `json:"currency"`
	ObservedAt           time.Time `json:"observed_at"`
}
type OwnerCashOption struct {
	ID                   int64     `json:"id"`
	AmountMinor          int64     `json:"amount_minor"`
	ExternalReceiptID    string    `json:"external_receipt_id"`
	Currency             string    `json:"currency"`
	ReconciliationStatus string    `json:"reconciliation_status"`
	ObservedAt           time.Time `json:"observed_at"`
}
type OwnerFinanceAccountOption struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	AccountType string `json:"account_type"`
	Currency    string `json:"currency"`
	Status      string `json:"status"`
}
type OwnerAftersalesCaseOption struct {
	ID             int64  `json:"id"`
	OrderID        int64  `json:"order_id"`
	Kind           string `json:"kind"`
	Status         string `json:"status"`
	RequestedMinor int64  `json:"requested_minor"`
	Currency       string `json:"currency"`
}

func (s *Service) ListOwnerFactOptions(ctx context.Context, ownerID int64) (*OwnerFactOptions, error) {
	if ownerID <= 0 {
		return nil, errors.New("owner is required")
	}
	out := &OwnerFactOptions{Accounts: []OwnerAccountOption{}, Orders: []OwnerOrderOption{}, OrderItems: []OwnerOrderItemOption{}, CostVersions: []OwnerCostVersionOption{}, Settlements: []OwnerSettlementOption{}, CashReceipts: []OwnerCashOption{}, FinanceAccounts: []OwnerFinanceAccountOption{}, AftersalesCases: []OwnerAftersalesCaseOption{}}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("owner_platform_account_authority a").Select("a.account_id AS id,a.platform_code,COALESCE(p.store_name,'') AS store_name").Joins("JOIN platform_integration_account p ON p.id=a.account_id").Where("a.owner_id=?", ownerID).Order("a.account_id").Scan(&out.Accounts).Error; err != nil {
			return err
		}
		if err := tx.Table("platform_order_ingest").Select("normalized_order_id AS id,account_id,external_order_id,platform_code,COALESCE((SELECT currency FROM platform_order_ingest_item WHERE ingest_id=platform_order_ingest.id ORDER BY line_number LIMIT 1),'') AS currency,observed_at").Where("owner_id=? AND normalized_order_id IS NOT NULL AND event_action='reserve' AND truth_status='external_observed' AND processing_status='applied'", ownerID).Order("observed_at DESC,id DESC").Scan(&out.Orders).Error; err != nil {
			return err
		}
		if err := tx.Table("sales_order_item oi").Select("oi.id,oi.order_id,oi.sku_id,oi.sku_code,oi.product_name,oi.quantity").Joins("JOIN platform_order_ingest p ON p.normalized_order_id=oi.order_id").Where("p.owner_id=? AND p.event_action='reserve' AND p.truth_status='external_observed' AND p.processing_status='applied'", ownerID).Order("oi.order_id DESC,oi.id").Scan(&out.OrderItems).Error; err != nil {
			return err
		}
		if err := tx.Table("sourcing_cost_version cv").Select("cv.id,sm.internal_sku_id,cv.version,cv.total_minor,cv.target_currency AS currency").Joins("JOIN sourcing_sku_mapping sm ON sm.id=cv.sku_mapping_id AND sm.owner_id=cv.owner_id").Where("cv.owner_id=? AND NOT EXISTS (SELECT 1 FROM sourcing_cost_line cl WHERE cl.cost_version_id=cv.id AND cl.truth_status<>'actual') AND EXISTS (SELECT 1 FROM sourcing_cost_line cl WHERE cl.cost_version_id=cv.id)", ownerID).Order("cv.created_at DESC,cv.id DESC").Scan(&out.CostVersions).Error; err != nil {
			return err
		}
		if err := tx.Table("platform_settlement_ingest i").Select("i.id,i.account_id,i.external_settlement_id,i.platform_code,i.currency,i.observed_at,COALESCE(SUM(CASE WHEN l.kind='sale' THEN l.amount_minor ELSE -l.amount_minor END),0) AS receivable_minor").Joins("LEFT JOIN platform_settlement_fact_line l ON l.ingest_id=i.id").Where("i.owner_id=? AND i.truth_status='external_observed'", ownerID).Group("i.id").Order("i.observed_at DESC,i.id DESC").Scan(&out.Settlements).Error; err != nil {
			return err
		}
		if err := tx.Table("cash_receipt").Select("id,amount_minor,external_receipt_id,currency,reconciliation_status,observed_at").Where("owner_id=? AND truth_status='external_observed'", ownerID).Order("observed_at DESC,id DESC").Scan(&out.CashReceipts).Error; err != nil {
			return err
		}
		if err := tx.Table("finance_account").Select("id,name,account_type,currency,status").Where("owner_id=? AND status='active'", ownerID).Order("name,id").Scan(&out.FinanceAccounts).Error; err != nil {
			return err
		}
		return tx.Table("aftersales_resolution_case").Select("id,order_id,kind,status,requested_minor,currency").Where("owner_id=?", ownerID).Order("created_at DESC,id DESC").Scan(&out.AftersalesCases).Error
	})
	return out, err
}

// ReadOwnerOperatingView reads one exact normalized order from the Unit 4-6
// authority tables under an Owner-scoped, repeatable read transaction.
func (s *Service) ReadOwnerOperatingView(ctx context.Context, ownerID, orderID int64) (*OwnerOperatingView, error) {
	if ownerID <= 0 || orderID <= 0 {
		return nil, errors.New("owner and order are required")
	}
	var out *OwnerOperatingView
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		v := &OwnerOperatingView{AsOf: time.Now().UTC()}
		var orders []OwnerOrderFact
		if err := tx.Table("platform_order_ingest").Select("normalized_order_id AS order_id,id AS ingest_id,account_id,platform_code,event_action,truth_status,processing_status,payload_sha256,observed_at").Where("owner_id=? AND normalized_order_id=? AND event_action='reserve' AND truth_status='external_observed' AND processing_status='applied'", ownerID, orderID).Find(&orders).Error; err != nil {
			return err
		}
		if len(orders) != 1 {
			return fmt.Errorf("exactly one applied external Owner order fact required")
		}
		v.Order = orders[0]
		v.Evidence = append(v.Evidence, OwnerFactEvidence{"platform_order_ingest", orders[0].IngestID, "external_observed", orders[0].PayloadSHA256, orders[0].ObservedAt, "applied external order fact"})
		if err := tx.Table("order_inventory_ledger").Where("owner_id=? AND order_id=?", ownerID, orderID).Order("id").Find(&v.Inventory).Error; err != nil {
			return err
		}
		if len(v.Inventory) == 0 {
			v.Unknowns = append(v.Unknowns, "尚无该订单的库存账本动作")
		}

		var trackingIDs []string
		if err := tx.Table("supply_chain_tracking").Where("owner_id=? AND order_id=?", ownerID, strconv.FormatInt(orderID, 10)).Pluck("id", &trackingIDs).Error; err != nil {
			return err
		}
		if len(trackingIDs) > 0 {
			if err := tx.Table("supply_chain_carrier_event").Where("owner_id=? AND tracking_id IN ?", ownerID, trackingIDs).Order("occurred_at,id").Find(&v.Fulfillment).Error; err != nil {
				return err
			}
		}
		if len(v.Fulfillment) == 0 {
			v.Unknowns = append(v.Unknowns, "尚无可信承运商事件；人工物流状态不能升级为外部事实")
		}
		for _, f := range v.Fulfillment {
			v.Evidence = append(v.Evidence, OwnerFactEvidence{"supply_chain_carrier_event", f.ID, f.TruthStatus, f.PayloadSHA256, f.ObservedAt, f.Status})
		}

		if err := tx.Table("aftersales_resolution_case").Where("owner_id=? AND order_id=?", ownerID, orderID).Order("id").Find(&v.Aftersales).Error; err != nil {
			return err
		}
		for i := range v.Aftersales {
			var r OwnerAftersaleReceipt
			err := tx.Table("aftersales_resolution_receipt").Where("owner_id=? AND resolution_id=?", ownerID, v.Aftersales[i].ID).Take(&r).Error
			if err == nil {
				v.Aftersales[i].Receipt = &r
				v.Evidence = append(v.Evidence, OwnerFactEvidence{"aftersales_resolution_receipt", r.ID, "external_observed", r.ReceiptSHA256, r.ObservedAt, r.Outcome})
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if v.Aftersales[i].Status != "succeeded" && v.Aftersales[i].Status != "failed" && v.Aftersales[i].Status != "rejected" {
				v.Blockers = append(v.Blockers, fmt.Sprintf("售后案卷 %d 尚未终结", v.Aftersales[i].ID))
			}
		}

		var settlementIDs []int64
		if err := tx.Table("platform_settlement_fact_line l").Select("DISTINCT l.ingest_id").Joins("JOIN platform_settlement_ingest i ON i.id=l.ingest_id").Where("i.owner_id=? AND l.order_id=?", ownerID, orderID).Pluck("l.ingest_id", &settlementIDs).Error; err != nil {
			return err
		}
		for _, id := range settlementIDs {
			var st OwnerSettlementFact
			if err := tx.Table("platform_settlement_ingest").Select("id,account_id,platform_code,truth_status,currency,payload_sha256,content_sha256,observed_at").Where("id=? AND owner_id=?", id, ownerID).Take(&st).Error; err != nil {
				return err
			}
			if err := tx.Table("platform_settlement_fact_line").Select("id,kind,fee_code,currency,amount_minor,occurred_at").Where("ingest_id=? AND order_id=?", id, orderID).Order("line_number").Find(&st.Lines).Error; err != nil {
				return err
			}
			v.Settlements = append(v.Settlements, st)
			v.Evidence = append(v.Evidence, OwnerFactEvidence{"platform_settlement_ingest", st.ID, st.TruthStatus, st.PayloadSHA256, st.ObservedAt, "immutable settlement fact"})
		}
		if len(v.Settlements) == 0 {
			v.Unknowns = append(v.Unknowns, "尚无该订单的平台结算事实")
		} else if len(v.Settlements) > 1 {
			v.Blockers = append(v.Blockers, "该订单存在多个结算批次，需按事实行核对，不得混合宣称单一结算")
		}

		var p OwnerProfitFact
		err := tx.Table("order_final_profit_version").Where("owner_id=? AND order_id=?", ownerID, orderID).Order("version DESC,id DESC").Take(&p).Error
		if err == nil {
			v.Profit = &p
			v.Evidence = append(v.Evidence, OwnerFactEvidence{"order_final_profit_version", p.ID, "actual", p.SourceManifestSHA256, p.FinalizedAt, "immutable final profit version"})
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			v.Unknowns = append(v.Unknowns, "尚无该订单的最终利润版本")
		} else {
			return err
		}

		cashEligibleSettlementIDs := make([]int64, 0, len(settlementIDs))
		for _, settlementID := range settlementIDs {
			var distinctOrders int64
			if err := tx.Table("platform_settlement_fact_line").Where("ingest_id=?", settlementID).Distinct("order_id").Count(&distinctOrders).Error; err != nil {
				return err
			}
			if distinctOrders == 1 {
				cashEligibleSettlementIDs = append(cashEligibleSettlementIDs, settlementID)
			} else if distinctOrders > 1 {
				v.Blockers = append(v.Blockers, fmt.Sprintf("结算批次 %d 为多订单批次，批次级现金不可归属单订单", settlementID))
			}
		}
		if len(cashEligibleSettlementIDs) > 0 {
			if err := tx.Table("cash_reconciliation c").Select("c.id,c.cash_receipt_id,c.platform_settlement_ingest_id,c.amount_minor,c.expected_receivable_minor,c.currency,c.status,c.request_sha256,c.reconciled_at").Where("c.owner_id=? AND c.platform_settlement_ingest_id IN ?", ownerID, cashEligibleSettlementIDs).Order("c.id").Find(&v.Cash).Error; err != nil {
				return err
			}
		}
		if len(v.Cash) == 0 {
			v.Unknowns = append(v.Unknowns, "尚无与该订单结算绑定的现金对账")
		}
		for _, c := range v.Cash {
			truth := "unknown"
			observed := v.AsOf
			if c.Status == "reconciled" && c.ReconciledAt != nil {
				truth = "reconciled"
				observed = *c.ReconciledAt
			} else {
				v.Blockers = append(v.Blockers, fmt.Sprintf("现金对账 %d 状态为 %s", c.ID, c.Status))
			}
			v.Evidence = append(v.Evidence, OwnerFactEvidence{"cash_reconciliation", c.ID, truth, c.RequestSHA256, observed, c.Status})
		}
		out = v
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return out, err
}
