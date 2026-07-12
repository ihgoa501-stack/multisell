package experiment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// OwnerBusinessClosureView is a redacted, read-only projection rooted in one
// owner-scoped experiment. It deliberately does not expose customer PII, raw
// settlement payloads, account names/balances, or free-text finance details.
type OwnerBusinessClosureView struct {
	ExperimentID string                    `json:"experiment_id"`
	Order        OwnerOrderClosure         `json:"order"`
	Aftersales   OwnerAftersalesClosure    `json:"aftersales"`
	Settlement   OwnerSettlementClosure    `json:"settlement"`
	Profit       OwnerProfitClosure        `json:"profit"`
	Cash         OwnerCashClosure          `json:"cash"`
	Blockers     []string                  `json:"blockers"`
	Unknowns     []string                  `json:"unknowns"`
	EvidenceRefs []OwnerClosureEvidenceRef `json:"evidence_refs"`
	AsOf         time.Time                 `json:"as_of"`
}

type OwnerOrderClosure struct {
	ID               int64      `json:"id"`
	PlatformID       int64      `json:"platform_id,omitempty"`
	OrderNoMasked    string     `json:"order_no_masked"`
	Status           string     `json:"status"`
	PaidAt           *time.Time `json:"paid_at,omitempty"`
	DeliveredAt      *time.Time `json:"delivered_at,omitempty"`
	PaymentRecorded  bool       `json:"payment_recorded"`
	DeliveryRecorded bool       `json:"delivery_recorded"`
	TruthStatus      string     `json:"truth_status"`
	SourceStatus     string     `json:"source_status"`
}
type OwnerAftersalesClosure struct {
	RecordedCount           int64  `json:"recorded_count"`
	ObservationWindowStatus string `json:"observation_window_status"`
}
type OwnerSettlementClosure struct {
	ID                 int64      `json:"id"`
	SettlementNoMasked string     `json:"settlement_no_masked"`
	Currency           string     `json:"currency"`
	Status             string     `json:"status"`
	SourceType         string     `json:"source_type"`
	ImportedAt         *time.Time `json:"imported_at,omitempty"`
	ItemCount          int64      `json:"item_count"`
	MatchedItemCount   int64      `json:"matched_item_count"`
	Trusted            bool       `json:"trusted"`
	FullyReconciled    bool       `json:"fully_reconciled"`
}
type OwnerProfitClosure struct {
	ID           int64   `json:"id"`
	OrderID      int64   `json:"order_id"`
	Revenue      float64 `json:"revenue"`
	TotalCost    float64 `json:"total_cost"`
	Profit       float64 `json:"profit"`
	Currency     string  `json:"currency"`
	Status       string  `json:"status"`
	MissingCosts string  `json:"missing_costs"`
	Final        bool    `json:"final"`
}
type OwnerCashClosure struct {
	RecordedCount     int64      `json:"recorded_count"`
	Amount            float64    `json:"amount"`
	Currency          string     `json:"currency"`
	TransactionDate   *time.Time `json:"transaction_date,omitempty"`
	ConsistencyStatus string     `json:"consistency_status"`
}
type OwnerClosureEvidenceRef struct {
	SourceType  string `json:"source_type"`
	SourceID    string `json:"source_id"`
	TruthStatus string `json:"truth_status"`
	Summary     string `json:"summary"`
}

type closureOrderRow struct {
	ID                                                                      int64 `gorm:"primaryKey"`
	OrderNo, Status, RecipientName, RecipientPhone, ShippingAddress, Remark string
	PlatformID                                                              *int64
	PayAmount                                                               float64
	PaidAt, DeliveredAt, CancelledAt                                        *time.Time
}

func (closureOrderRow) TableName() string { return "sales_order" }

type closureSettlementRow struct {
	ID                           int64 `gorm:"primaryKey"`
	SettlementNo                 string
	PlatformID                   int64
	Currency, Status, SourceType string
	ImportedAt                   *time.Time
	RawData                      string
}

func (closureSettlementRow) TableName() string { return "settlement" }

type closureSettlementItemRow struct {
	ID                                                           int64 `gorm:"primaryKey"`
	SettlementID                                                 int64
	OrderID                                                      *int64
	OrderNo, TransactionType, ReconciliationStatus, ReconciledBy string
	ReconciledAt                                                 *time.Time
}

func (closureSettlementItemRow) TableName() string { return "settlement_item" }

type closureProfitRow struct {
	ID, OrderID                int64
	Revenue, TotalCost, Profit float64
	ProfitStatus, MissingCosts string
	CalculatedAt               time.Time
}

func (closureProfitRow) TableName() string { return "order_profit_record" }

type closureAftersaleRow struct {
	ID, OrderID int64
	Status      string
}

func (closureAftersaleRow) TableName() string { return "after_sales_order" }

type closureCashRow struct {
	ID, AccountID         int64
	TransactionType       string
	AccountType           string
	Amount                float64
	Currency              string
	OrderID, SettlementID *int64
	TransactionDate       *time.Time
}

func (closureCashRow) TableName() string { return "finance_transaction" }

type closureAccountRow struct {
	ID                    int64
	AccountType, Currency string
	Status                string
}

func (closureAccountRow) TableName() string { return "finance_account" }

// ReadOwnerBusinessClosure reads only objects explicitly linked to an experiment
// that belongs to ownerID. Missing downstream facts are returned as unknowns;
// an absent or ambiguous order link is rejected because it changes the subject.
func (s *Service) ReadOwnerBusinessClosure(ctx context.Context, ownerID int64, experimentID string) (*OwnerBusinessClosureView, error) {
	var view *OwnerBusinessClosureView
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		scoped := *s
		scoped.db = tx
		var readErr error
		view, readErr = scoped.readOwnerBusinessClosure(ctx, ownerID, experimentID)
		return readErr
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return view, err
}

func (s *Service) readOwnerBusinessClosure(ctx context.Context, ownerID int64, experimentID string) (*OwnerBusinessClosureView, error) {
	if err := s.requireOwner(ctx, experimentID, ownerID); err != nil {
		return nil, err
	}
	orderID, err := s.uniqueLinkedID(ctx, experimentID, "order", true)
	if err != nil {
		return nil, err
	}
	var order closureOrderRow
	if err := s.db.WithContext(ctx).Where("id = ?", orderID).First(&order).Error; err != nil {
		return nil, err
	}
	view := &OwnerBusinessClosureView{ExperimentID: experimentID, AsOf: time.Now().UTC()}
	view.Order = OwnerOrderClosure{ID: order.ID, OrderNoMasked: maskIdentifier(order.OrderNo), Status: order.Status, PaidAt: order.PaidAt, DeliveredAt: order.DeliveredAt, TruthStatus: TruthUnknown, SourceStatus: "internal_record"}
	view.Order.PaymentRecorded = order.PaidAt != nil && order.PayAmount > 0 && order.CancelledAt == nil
	view.Order.DeliveryRecorded = order.DeliveredAt != nil && (order.Status == "delivered" || order.Status == "completed")
	if order.PlatformID != nil {
		view.Order.PlatformID = *order.PlatformID
	}
	view.Unknowns = append(view.Unknowns, "订单付款与签收仅为内部记录，缺少可核验的外部来源，真实性保持 unknown")
	view.EvidenceRefs = append(view.EvidenceRefs, OwnerClosureEvidenceRef{"order", strconv.FormatInt(order.ID, 10), TruthUnknown, "internal_record"})

	if err := s.readAftersalesClosure(ctx, orderID, view); err != nil {
		return nil, err
	}
	settlementID, settlementErr := s.uniqueLinkedID(ctx, experimentID, "settlement", false)
	if settlementErr != nil {
		view.Blockers = append(view.Blockers, settlementErr.Error())
	} else if settlementID == 0 {
		view.Unknowns = append(view.Unknowns, "尚未关联结算")
	} else {
		if err := s.readSettlementClosure(ctx, order, settlementID, view); err != nil {
			return nil, err
		}
	}
	profitID, profitErr := s.uniqueLinkedID(ctx, experimentID, "profit_record", false)
	if profitErr != nil {
		view.Blockers = append(view.Blockers, profitErr.Error())
	} else if profitID == 0 {
		view.Unknowns = append(view.Unknowns, "尚未关联订单利润记录")
	} else {
		if err := s.readProfitClosure(ctx, orderID, profitID, view); err != nil {
			return nil, err
		}
	}
	if err := s.detectMixedSettlements(ctx, order, settlementID, view); err != nil {
		return nil, err
	}
	if err := s.readCashClosure(ctx, experimentID, orderID, settlementID, view); err != nil {
		return nil, err
	}
	return view, nil
}

func (s *Service) uniqueLinkedID(ctx context.Context, experimentID, objectType string, required bool) (int64, error) {
	var links []ObjectLink
	if err := s.db.WithContext(ctx).Where("experiment_id = ? AND object_type = ?", experimentID, objectType).Find(&links).Error; err != nil {
		return 0, err
	}
	if len(links) == 0 {
		if required {
			return 0, fmt.Errorf("%s link required", objectType)
		}
		return 0, nil
	}
	if len(links) != 1 {
		return 0, fmt.Errorf("%s 存在多个对象关联，无法确定唯一事实", objectType)
	}
	id, err := strconv.ParseInt(strings.TrimSpace(links[0].ObjectID), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid %s link", objectType)
	}
	return id, nil
}

func (s *Service) readAftersalesClosure(ctx context.Context, orderID int64, view *OwnerBusinessClosureView) error {
	q := s.db.WithContext(ctx).Model(&closureAftersaleRow{}).Where("order_id = ?", orderID)
	if err := q.Count(&view.Aftersales.RecordedCount).Error; err != nil {
		return fmt.Errorf("read aftersales count: %w", err)
	}
	view.Aftersales.ObservationWindowStatus = TruthUnknown
	view.Unknowns = append(view.Unknowns, "售后/争议观察期未知；无记录不代表观察期已关闭")
	return nil
}

func (s *Service) readSettlementClosure(ctx context.Context, order closureOrderRow, id int64, view *OwnerBusinessClosureView) error {
	var st closureSettlementRow
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&st).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("read settlement: %w", err)
		}
		view.Blockers = append(view.Blockers, "关联结算不存在")
		return nil
	}
	view.Settlement = OwnerSettlementClosure{ID: st.ID, SettlementNoMasked: maskIdentifier(st.SettlementNo), Currency: st.Currency, Status: st.Status, SourceType: st.SourceType, ImportedAt: st.ImportedAt}
	q := s.db.WithContext(ctx).Model(&closureSettlementItemRow{}).Where("settlement_id = ?", id)
	if err := q.Count(&view.Settlement.ItemCount).Error; err != nil {
		return fmt.Errorf("count settlement items: %w", err)
	}
	if err := q.Where("reconciliation_status = ? AND reconciled_at IS NOT NULL AND reconciled_by <> ''", "matched").Count(&view.Settlement.MatchedItemCount).Error; err != nil {
		return fmt.Errorf("count matched settlement items: %w", err)
	}
	var orderMatches int64
	if err := s.db.WithContext(ctx).Model(&closureSettlementItemRow{}).Where("settlement_id = ? AND (order_id = ? OR order_no = ?)", id, order.ID, order.OrderNo).Count(&orderMatches).Error; err != nil {
		return fmt.Errorf("count order settlement items: %w", err)
	}
	view.Settlement.Trusted = isTrustedSettlement(st.Status, st.SourceType, st.ImportedAt)
	view.Settlement.FullyReconciled = isFullyReconciledSettlement(view.Settlement.ItemCount, view.Settlement.ItemCount-view.Settlement.MatchedItemCount, orderMatches)
	if !view.Settlement.Trusted || !view.Settlement.FullyReconciled {
		view.Blockers = append(view.Blockers, "关联结算不可信、未全部对账或未关联实验订单")
	}
	view.EvidenceRefs = append(view.EvidenceRefs, OwnerClosureEvidenceRef{"settlement", strconv.FormatInt(id, 10), TruthUnknown, st.SourceType})
	return nil
}

func (s *Service) readProfitClosure(ctx context.Context, orderID, id int64, view *OwnerBusinessClosureView) error {
	var p closureProfitRow
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&p).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("read profit record: %w", err)
		}
		view.Blockers = append(view.Blockers, "关联利润记录不存在")
		return nil
	}
	view.Profit = OwnerProfitClosure{ID: p.ID, OrderID: p.OrderID, Revenue: p.Revenue, TotalCost: p.TotalCost, Profit: p.Profit, Currency: view.Settlement.Currency, Status: p.ProfitStatus, MissingCosts: p.MissingCosts}
	view.Profit.Final = isFinalProfitForOrder(p.OrderID, orderID, p.ProfitStatus, p.MissingCosts) && view.Settlement.Trusted && view.Settlement.FullyReconciled
	if !view.Profit.Final {
		view.Blockers = append(view.Blockers, "利润记录不是该订单的完整最终利润")
	}
	view.EvidenceRefs = append(view.EvidenceRefs, OwnerClosureEvidenceRef{"order_profit_record", strconv.FormatInt(id, 10), TruthUnknown, p.ProfitStatus})
	return nil
}

func (s *Service) detectMixedSettlements(ctx context.Context, order closureOrderRow, linkedSettlementID int64, view *OwnerBusinessClosureView) error {
	var ids []int64
	if err := s.db.WithContext(ctx).Model(&closureSettlementItemRow{}).Where("order_id = ? OR order_no = ?", order.ID, order.OrderNo).Distinct("settlement_id").Pluck("settlement_id", &ids).Error; err != nil {
		return fmt.Errorf("detect mixed settlements: %w", err)
	}
	if len(ids) > 1 {
		view.Blockers = append(view.Blockers, "同一订单存在多个结算来源，利润可能混合计算")
		view.Profit.Final = false
	}
	if len(ids) == 1 && linkedSettlementID > 0 && ids[0] != linkedSettlementID {
		view.Blockers = append(view.Blockers, "利润数据来源与实验关联结算不一致")
		view.Profit.Final = false
	}
	return nil
}

func (s *Service) readCashClosure(ctx context.Context, experimentID string, orderID, settlementID int64, view *OwnerBusinessClosureView) error {
	view.Cash.ConsistencyStatus = TruthUnknown
	if settlementID <= 0 {
		view.Unknowns = append(view.Unknowns, "没有实验关联结算，不能接受现金回收记录")
		return nil
	}
	transactionID, err := s.uniqueLinkedID(ctx, experimentID, "cash_transaction", false)
	if err != nil {
		view.Blockers = append(view.Blockers, err.Error())
		return nil
	}
	if transactionID == 0 {
		view.Unknowns = append(view.Unknowns, "尚未关联现金收款记录；现金回收未知")
		return nil
	}
	var row closureCashRow
	if err := s.db.WithContext(ctx).Table("finance_transaction AS ft").Select("ft.*, fa.account_type").Joins("JOIN finance_account AS fa ON fa.id = ft.account_id").Where("ft.id = ?", transactionID).First(&row).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("read cash transaction: %w", err)
		}
		view.Unknowns = append(view.Unknowns, "关联现金收款记录不存在")
		return nil
	}
	if row.OrderID == nil || *row.OrderID != orderID || row.SettlementID == nil || *row.SettlementID != settlementID || row.TransactionType != "revenue" || row.Amount <= 0 || row.TransactionDate == nil || (row.AccountType != "bank" && row.AccountType != "cash") {
		view.Blockers = append(view.Blockers, "现金记录未同时关联实验订单与结算")
		return nil
	}
	view.Cash.RecordedCount, view.Cash.Amount, view.Cash.Currency, view.Cash.TransactionDate = 1, row.Amount, row.Currency, row.TransactionDate
	view.Unknowns = append(view.Unknowns, "现金记录的金额和币种尚未与结算及最终利润完成一致性核验")
	return nil
}

func maskIdentifier(value string) string {
	v := strings.TrimSpace(value)
	if len(v) <= 4 {
		return "****"
	}
	return "****" + v[len(v)-4:]
}
