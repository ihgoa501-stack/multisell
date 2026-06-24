package settlement

import (
	"errors"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides settlement business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new settlement service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// List returns paginated settlements with optional filter.
func (s *Service) List(p *common.Pagination, f *SettlementListFilter) ([]Settlement, int64, error) {
	q := s.db.Model(&Settlement{})
	if f != nil {
		if f.Search != "" {
			like := "%" + f.Search + "%"
			q = q.Where("settlement_no ILIKE ?", like)
		}
		if f.Status != "" {
			q = q.Where("status = ?", f.Status)
		}
		if f.PlatformID != nil {
			q = q.Where("platform_id = ?", *f.PlatformID)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Settlement
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Get returns a single settlement with its items.
func (s *Service) Get(id int64) (*SettlementDetail, error) {
	var st Settlement
	if err := s.db.First(&st, id).Error; err != nil {
		return nil, err
	}
	var items []SettlementItem
	if err := s.db.Where("settlement_id = ?", id).Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return &SettlementDetail{Settlement: st, Items: items}, nil
}

// Create inserts a settlement with its items in a transaction.
func (s *Service) Create(in *CreateSettlementInput) (*Settlement, error) {
	if strings.TrimSpace(in.SettlementNo) == "" {
		return nil, errors.New("settlement_no is required")
	}
	status := in.Status
	if status == "" {
		status = "pending"
	}
	currency := in.Currency
	if currency == "" {
		currency = "CNY"
	}
	st := Settlement{
		PlatformID:   in.PlatformID,
		SettlementNo: in.SettlementNo,
		PeriodStart:  in.PeriodStart,
		PeriodEnd:    in.PeriodEnd,
		Currency:     currency,
		Status:       status,
		RawData:      in.RawData,
		ImportedAt:   in.ImportedAt,
	}
	if in.TotalRevenue != nil {
		st.TotalRevenue = *in.TotalRevenue
	}
	if in.TotalFee != nil {
		st.TotalFee = *in.TotalFee
	}
	if in.TotalRefund != nil {
		st.TotalRefund = *in.TotalRefund
	}
	if in.TotalNet != nil {
		st.TotalNet = *in.TotalNet
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&st).Error; err != nil {
			return err
		}
		if len(in.Items) == 0 {
			return nil
		}
		items := make([]SettlementItem, 0, len(in.Items))
		for _, it := range in.Items {
			item := SettlementItem{
				SettlementID:    st.ID,
				TransactionType: it.TransactionType,
				TransactionID:   it.TransactionID,
				OrderNo:         it.OrderNo,
				OrderID:         it.OrderID,
				SkuID:           it.SkuID,
				OccurredAt:      it.OccurredAt,
			}
			if it.Amount != nil {
				item.Amount = *it.Amount
			}
			if it.Fee != nil {
				item.Fee = *it.Fee
			}
			if it.Net != nil {
				item.Net = *it.Net
			}
			if it.Quantity != nil {
				item.Quantity = *it.Quantity
			}
			items = append(items, item)
		}
		return tx.Create(&items).Error
	})
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// Update applies partial updates to a settlement.
func (s *Service) Update(id int64, in *UpdateSettlementInput) (*Settlement, error) {
	var st Settlement
	if err := s.db.First(&st, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.PlatformID != nil {
		updates["platform_id"] = *in.PlatformID
	}
	if in.PeriodStart != nil {
		updates["period_start"] = *in.PeriodStart
	}
	if in.PeriodEnd != nil {
		updates["period_end"] = *in.PeriodEnd
	}
	if in.Currency != nil {
		updates["currency"] = *in.Currency
	}
	if in.TotalRevenue != nil {
		updates["total_revenue"] = *in.TotalRevenue
	}
	if in.TotalFee != nil {
		updates["total_fee"] = *in.TotalFee
	}
	if in.TotalRefund != nil {
		updates["total_refund"] = *in.TotalRefund
	}
	if in.TotalNet != nil {
		updates["total_net"] = *in.TotalNet
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if in.RawData != nil {
		updates["raw_data"] = *in.RawData
	}
	if len(updates) == 0 {
		return &st, nil
	}
	if err := s.db.Model(&st).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&st, id).Error; err != nil {
		return nil, err
	}
	return &st, nil
}

// Delete removes a settlement and cascades its items.
func (s *Service) Delete(id int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("settlement_id = ?", id).Delete(&SettlementItem{}).Error; err != nil {
			return err
		}
		res := tx.Delete(&Settlement{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

// Reconcile updates reconciliation_status of settlement items.
// If ItemID is nil, all pending items of the settlement are updated.
func (s *Service) Reconcile(id int64, in *ReconcileInput) error {
	now := time.Now()
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Verify settlement exists
		var st Settlement
		if err := tx.First(&st, id).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{
			"reconciliation_status": in.ReconciliationStatus,
			"reconciled_at":         now,
			"reconciled_by":         in.ReconciledBy,
		}
		if in.ReconciliationNote != "" {
			updates["reconciliation_note"] = in.ReconciliationNote
		}
		var res *gorm.DB
		if in.ItemID != nil {
			res = tx.Model(&SettlementItem{}).
				Where("id = ? AND settlement_id = ?", *in.ItemID, id).
				Updates(updates)
		} else {
			res = tx.Model(&SettlementItem{}).
				Where("settlement_id = ? AND reconciliation_status = ?", id, "pending").
				Updates(updates)
		}
		if res.Error != nil {
			return res.Error
		}
		// If all items are matched, mark settlement as reconciled
		var pendingCount int64
		if err := tx.Model(&SettlementItem{}).
			Where("settlement_id = ? AND reconciliation_status = ?", id, "pending").
			Count(&pendingCount).Error; err != nil {
			return err
		}
		if pendingCount == 0 && st.Status != "closed" {
			if err := tx.Model(&st).Update("status", "reconciled").Error; err != nil {
				return err
			}
		} else if pendingCount > 0 && st.Status == "pending" {
			if err := tx.Model(&st).Update("status", "reconciling").Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Summary returns aggregation for dashboard.
func (s *Service) Summary() (*SettlementSummary, error) {
	var total int64
	if err := s.db.Model(&Settlement{}).Count(&total).Error; err != nil {
		return nil, err
	}
	type statusCount struct {
		Status string
		Cnt    int64
	}
	var scs []statusCount
	if err := s.db.Model(&Settlement{}).Select("status, COUNT(*) AS cnt").Group("status").Scan(&scs).Error; err != nil {
		return nil, err
	}
	byStatus := make(map[string]int64, len(scs))
	for _, sc := range scs {
		byStatus[sc.Status] = sc.Cnt
	}
	type platformNet struct {
		PlatformID *int64
		TotalNet   float64
	}
	var pns []platformNet
	if err := s.db.Model(&Settlement{}).
		Select("platform_id, COALESCE(SUM(total_net),0) AS total_net").
		Group("platform_id").
		Scan(&pns).Error; err != nil {
		return nil, err
	}
	netByPlatform := make([]PlatformNetTotal, 0, len(pns))
	for _, pn := range pns {
		netByPlatform = append(netByPlatform, PlatformNetTotal{
			PlatformID: pn.PlatformID,
			TotalNet:   pn.TotalNet,
		})
	}
	return &SettlementSummary{Total: total, ByStatus: byStatus, NetByPlatform: netByPlatform}, nil
}

// ---------- Settlement items ----------

// AddItemInput is the payload for POST /settlement/:id/items.
type AddItemInput struct {
	TransactionType string     `json:"transaction_type" binding:"required"`
	TransactionID   string     `json:"transaction_id"`
	OrderNo         string     `json:"order_no"`
	OrderID         *int64     `json:"order_id"`
	SkuID           *int64     `json:"sku_id"`
	Amount          float64    `json:"amount"`
	Fee             float64    `json:"fee"`
	Net             float64    `json:"net"`
	Quantity        int        `json:"quantity"`
	OccurredAt      *time.Time `json:"occurred_at"`
}

// AddItem creates a settlement_item under the given settlement.
func (s *Service) AddItem(settlementID int64, in *AddItemInput) (*SettlementItem, error) {
	// Verify parent exists.
	var count int64
	if err := s.db.Model(&Settlement{}).Where("id = ?", settlementID).Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errors.New("settlement not found")
	}
	item := SettlementItem{
		SettlementID:         settlementID,
		TransactionType:      in.TransactionType,
		TransactionID:        in.TransactionID,
		OrderNo:              in.OrderNo,
		OrderID:              in.OrderID,
		SkuID:                in.SkuID,
		Amount:               in.Amount,
		Fee:                  in.Fee,
		Net:                  in.Net,
		Quantity:             in.Quantity,
		OccurredAt:           in.OccurredAt,
		ReconciliationStatus: "pending",
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// ListItems returns all items for a settlement, optionally filtered by
// reconciliation_status.
func (s *Service) ListItems(settlementID int64, reconciliationStatus string) ([]SettlementItem, error) {
	var items []SettlementItem
	q := s.db.Where("settlement_id = ?", settlementID)
	if reconciliationStatus != "" {
		q = q.Where("reconciliation_status = ?", reconciliationStatus)
	}
	if err := q.Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// UpdateReconciliationInput is the payload for PUT /settlement/items/:item_id/reconciliation.
type UpdateReconciliationInput struct {
	ReconciliationStatus string `json:"reconciliation_status" binding:"required"`
	ReconciliationNote   string `json:"reconciliation_note"`
	ReconciledBy         string `json:"reconciled_by"`
}

// UpdateItemReconciliation updates the reconciliation state of a single item.
func (s *Service) UpdateItemReconciliation(itemID int64, in *UpdateReconciliationInput) (*SettlementItem, error) {
	var item SettlementItem
	if err := s.db.First(&item, itemID).Error; err != nil {
		return nil, err
	}
	validStatuses := map[string]bool{"pending": true, "matched": true, "unmatched": true, "discrepancy": true}
	if !validStatuses[in.ReconciliationStatus] {
		return nil, errors.New("invalid reconciliation_status")
	}
	now := time.Now()
	updates := map[string]interface{}{
		"reconciliation_status": in.ReconciliationStatus,
		"reconciliation_note":   in.ReconciliationNote,
		"reconciled_by":         in.ReconciledBy,
		"reconciled_at":         &now,
	}
	if err := s.db.Model(&item).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&item, itemID).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
