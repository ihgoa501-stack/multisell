package sourcing1688

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	WatchRunPendingBrowser = "pending_browser"
	WatchRunEvaluated      = "evaluated"
)

// SourcingWatchSubscription is an explicit Owner choice. It never authorizes
// unattended scraping; a pending run means the Owner browser must provide a
// new immutable observation.
type SourcingWatchSubscription struct {
	ID                int64      `gorm:"column:id;primaryKey" json:"id"`
	OwnerID           int64      `gorm:"column:owner_id;not null;uniqueIndex:ux_sourcing_watch_owner_source,priority:1" json:"owner_id"`
	SourcingProductID int64      `gorm:"column:sourcing_product_id;not null;uniqueIndex:ux_sourcing_watch_owner_source,priority:2" json:"sourcing_product_id"`
	Enabled           bool       `gorm:"column:enabled;not null" json:"enabled"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DisabledAt        *time.Time `gorm:"column:disabled_at" json:"disabled_at,omitempty"`
}

func (SourcingWatchSubscription) TableName() string { return "sourcing_1688_watch_subscription" }

type SourcingWatchRefreshRun struct {
	ID                 int64      `gorm:"column:id;primaryKey" json:"id"`
	OwnerID            int64      `gorm:"column:owner_id;not null;uniqueIndex:ux_sourcing_watch_run_request,priority:1" json:"owner_id"`
	SourcingProductID  int64      `gorm:"column:sourcing_product_id;not null" json:"sourcing_product_id"`
	RequestID          string     `gorm:"column:request_id;size:80;not null;uniqueIndex:ux_sourcing_watch_run_request,priority:2" json:"request_id"`
	Status             string     `gorm:"column:status;size:32;not null" json:"status"`
	PreviousSnapshotID *int64     `gorm:"column:previous_snapshot_id" json:"previous_snapshot_id,omitempty"`
	CurrentSnapshotID  *int64     `gorm:"column:current_snapshot_id" json:"current_snapshot_id,omitempty"`
	AlertCount         int        `gorm:"column:alert_count;not null" json:"alert_count"`
	FailureCode        string     `gorm:"column:failure_code;size:60;not null" json:"failure_code,omitempty"`
	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	CompletedAt        *time.Time `gorm:"column:completed_at" json:"completed_at,omitempty"`
}

func (SourcingWatchRefreshRun) TableName() string { return "sourcing_1688_watch_refresh_run" }

// SourcingWatchAlert is append-only and references observations only. It has
// deliberately no draft/listing foreign key and no code path that mutates one.
type SourcingWatchAlert struct {
	ID                 int64           `gorm:"column:id;primaryKey" json:"id"`
	OwnerID            int64           `gorm:"column:owner_id;not null" json:"owner_id"`
	SourcingProductID  int64           `gorm:"column:sourcing_product_id;not null" json:"sourcing_product_id"`
	RefreshRunID       int64           `gorm:"column:refresh_run_id;not null;uniqueIndex:ux_sourcing_watch_alert_run_type,priority:1" json:"refresh_run_id"`
	PreviousSnapshotID int64           `gorm:"column:previous_snapshot_id;not null" json:"previous_snapshot_id"`
	CurrentSnapshotID  int64           `gorm:"column:current_snapshot_id;not null" json:"current_snapshot_id"`
	ChangeType         string          `gorm:"column:change_type;size:40;not null;uniqueIndex:ux_sourcing_watch_alert_run_type,priority:2" json:"change_type"`
	BeforeValue        json.RawMessage `gorm:"column:before_value;type:jsonb;not null" json:"before_value"`
	AfterValue         json.RawMessage `gorm:"column:after_value;type:jsonb;not null" json:"after_value"`
	ContentHash        string          `gorm:"column:content_hash;size:64;not null" json:"content_hash"`
	CreatedAt          time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (SourcingWatchAlert) TableName() string { return "sourcing_1688_watch_alert" }

type SetSourcingWatchInput struct {
	Enabled bool `json:"enabled"`
}
type CreateSourcingWatchRunInput struct {
	RequestID string `json:"request_id" binding:"required"`
}
type EvaluateSourcingWatchRunInput struct {
	PreviousSnapshotID int64 `json:"previous_snapshot_id" binding:"required"`
	CurrentSnapshotID  int64 `json:"current_snapshot_id" binding:"required"`
}

func (s *Service) SetSourcingWatch(ownerID, sourceID int64, enabled bool) (*SourcingWatchSubscription, error) {
	if err := s.RequireSourceOwner(sourceID, ownerID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	row := SourcingWatchSubscription{OwnerID: ownerID, SourcingProductID: sourceID, Enabled: enabled}
	updates := map[string]any{"enabled": enabled, "updated_at": now, "disabled_at": nil}
	if !enabled {
		updates["disabled_at"] = now
	}
	if err := s.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "owner_id"}, {Name: "sourcing_product_id"}}, DoUpdates: clause.Assignments(updates)}).Create(&row).Error; err != nil {
		return nil, err
	}
	if err := s.db.Where("owner_id = ? AND sourcing_product_id = ?", ownerID, sourceID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) GetSourcingWatch(ownerID, sourceID int64) (*SourcingWatchSubscription, error) {
	if err := s.RequireSourceOwner(sourceID, ownerID); err != nil {
		return nil, err
	}
	var row SourcingWatchSubscription
	if err := s.db.Where("owner_id = ? AND sourcing_product_id = ?", ownerID, sourceID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) CreateSourcingWatchRun(ownerID, sourceID int64, requestID string) (*SourcingWatchRefreshRun, error) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" || len(requestID) > 80 {
		return nil, fmt.Errorf("%w: valid request_id required", ErrInvalidWorkflow)
	}
	if err := s.RequireSourceOwner(sourceID, ownerID); err != nil {
		return nil, err
	}
	var watch SourcingWatchSubscription
	if err := s.db.Where("owner_id = ? AND sourcing_product_id = ? AND enabled = ?", ownerID, sourceID, true).First(&watch).Error; err != nil {
		return nil, fmt.Errorf("%w: source watch is not enabled", ErrWorkflowGate)
	}
	row := SourcingWatchRefreshRun{OwnerID: ownerID, SourcingProductID: sourceID, RequestID: requestID, Status: WatchRunPendingBrowser}
	if err := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return nil, err
	}
	if err := s.db.Where("owner_id = ? AND request_id = ?", ownerID, requestID).First(&row).Error; err != nil {
		return nil, err
	}
	if row.SourcingProductID != sourceID {
		return nil, fmt.Errorf("%w: request_id belongs to another source", ErrWorkflowGate)
	}
	return &row, nil
}

func (s *Service) GetSourcingWatchRun(ownerID, sourceID, runID int64) (*SourcingWatchRefreshRun, error) {
	if err := s.RequireSourceOwner(sourceID, ownerID); err != nil {
		return nil, err
	}
	var row SourcingWatchRefreshRun
	if err := s.db.Where("id = ? AND owner_id = ? AND sourcing_product_id = ?", runID, ownerID, sourceID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) ListSourcingWatchRuns(ownerID, sourceID int64) ([]SourcingWatchRefreshRun, error) {
	if err := s.RequireSourceOwner(sourceID, ownerID); err != nil {
		return nil, err
	}
	var rows []SourcingWatchRefreshRun
	err := s.db.Where("owner_id = ? AND sourcing_product_id = ?", ownerID, sourceID).Order("id DESC").Find(&rows).Error
	return rows, err
}

func (s *Service) EvaluateSourcingWatchRun(ownerID, sourceID, runID int64, in EvaluateSourcingWatchRunInput) (*SourcingWatchRefreshRun, error) {
	if err := s.RequireSourceOwner(sourceID, ownerID); err != nil {
		return nil, err
	}
	var result SourcingWatchRefreshRun
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var run SourcingWatchRefreshRun
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND owner_id = ? AND sourcing_product_id = ?", runID, ownerID, sourceID).First(&run).Error; err != nil {
			return err
		}
		if run.Status == WatchRunEvaluated {
			if run.PreviousSnapshotID == nil || run.CurrentSnapshotID == nil || *run.PreviousSnapshotID != in.PreviousSnapshotID || *run.CurrentSnapshotID != in.CurrentSnapshotID {
				return fmt.Errorf("%w: evaluated run is immutable", ErrWorkflowGate)
			}
			result = run
			return nil
		}
		if run.Status != WatchRunPendingBrowser {
			return fmt.Errorf("%w: run cannot be evaluated", ErrWorkflowGate)
		}
		var before, after Sourcing1688Snapshot
		if err := tx.Where("id = ? AND sourcing_product_id = ?", in.PreviousSnapshotID, sourceID).First(&before).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ? AND sourcing_product_id = ?", in.CurrentSnapshotID, sourceID).First(&after).Error; err != nil {
			return err
		}
		if before.ID == after.ID || after.CollectedAt.Before(run.CreatedAt) || !after.CollectedAt.After(before.CollectedAt) {
			return fmt.Errorf("%w: current observation must be a new browser observation for this run", ErrInvalidWorkflow)
		}
		changes := watchObservationDiff(before, after)
		for typ, values := range changes {
			b, _ := json.Marshal(values[0])
			a, _ := json.Marshal(values[1])
			h := sha256.Sum256([]byte(fmt.Sprintf("%d|%d|%s|%s|%s", before.ID, after.ID, typ, b, a)))
			alert := SourcingWatchAlert{OwnerID: ownerID, SourcingProductID: sourceID, RefreshRunID: run.ID, PreviousSnapshotID: before.ID, CurrentSnapshotID: after.ID, ChangeType: typ, BeforeValue: b, AfterValue: a, ContentHash: hex.EncodeToString(h[:])}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&alert).Error; err != nil {
				return err
			}
		}
		now := time.Now().UTC()
		previousID, currentID := before.ID, after.ID
		updated := tx.Model(&run).Where("status = ?", WatchRunPendingBrowser).Updates(map[string]any{"status": WatchRunEvaluated, "previous_snapshot_id": previousID, "current_snapshot_id": currentID, "alert_count": len(changes), "completed_at": now})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return fmt.Errorf("%w: watch run was concurrently evaluated", ErrWorkflowGate)
		}
		if err := tx.First(&result, run.ID).Error; err != nil {
			return err
		}
		return nil
	})
	return &result, err
}

func (s *Service) ListSourcingWatchAlerts(ownerID, sourceID int64) ([]SourcingWatchAlert, error) {
	if err := s.RequireSourceOwner(sourceID, ownerID); err != nil {
		return nil, err
	}
	var rows []SourcingWatchAlert
	err := s.db.Where("owner_id = ? AND sourcing_product_id = ?", ownerID, sourceID).Order("id DESC").Find(&rows).Error
	return rows, err
}

type watchObservation struct {
	Price           any            `json:"price"`
	MOQ             int            `json:"moq"`
	Supplier        string         `json:"supplier"`
	SKUSet          []string       `json:"sku_set"`
	QuotedInventory map[string]any `json:"quoted_inventory"`
	OfferState      string         `json:"offer_state"`
}

func watchObservationDiff(before, after Sourcing1688Snapshot) map[string][2]any {
	b, a := snapshotWatchObservation(before), snapshotWatchObservation(after)
	changes := map[string][2]any{}
	if fmt.Sprint(b.Price) != fmt.Sprint(a.Price) {
		changes["price"] = [2]any{b.Price, a.Price}
	}
	if b.MOQ != a.MOQ {
		changes["moq"] = [2]any{b.MOQ, a.MOQ}
	}
	if b.Supplier != a.Supplier {
		changes["supplier"] = [2]any{b.Supplier, a.Supplier}
	}
	if canonicalJSON(b.SKUSet) != canonicalJSON(a.SKUSet) {
		changes["sku_set"] = [2]any{b.SKUSet, a.SKUSet}
	}
	if canonicalJSON(b.QuotedInventory) != canonicalJSON(a.QuotedInventory) {
		changes["quoted_inventory"] = [2]any{b.QuotedInventory, a.QuotedInventory}
	}
	if b.OfferState != a.OfferState {
		changes["offer_state"] = [2]any{b.OfferState, a.OfferState}
	}
	return changes
}

func snapshotWatchObservation(s Sourcing1688Snapshot) watchObservation {
	o := watchObservation{Price: optionalFloatValue(s.ObservedPrice), MOQ: s.ObservedMOQ, Supplier: s.ObservedSupplier, QuotedInventory: map[string]any{}, OfferState: "unknown"}
	var raw any
	if json.Unmarshal(s.RawPayload, &raw) != nil {
		return o
	}
	// The current extension schema uses spec_variants. Historical controlled
	// captures used sku_variants/skuVariants, so monitoring must understand both.
	variants := findJSONKey(raw, "spec_variants", "sku_variants", "skuVariants", "skus")
	if list, ok := variants.([]any); ok {
		for i, value := range list {
			m, ok := value.(map[string]any)
			if !ok {
				continue
			}
			id := firstString(m, "supplier_sku", "sku_id", "skuId", "spec_id", "specId", "spec", "name")
			if id == "" {
				id = fmt.Sprintf("variant-%d", i+1)
			}
			o.SKUSet = append(o.SKUSet, id)
			if stock := firstValue(m, "stock", "inventory", "quantity", "canBookCount"); stock != nil {
				o.QuotedInventory[id] = stock
			}
		}
	}
	sort.Strings(o.SKUSet)
	if v := findJSONKey(raw, "offer_state", "offerStatus", "offer_status", "availability"); v != nil {
		o.OfferState = strings.ToLower(strings.TrimSpace(fmt.Sprint(v)))
	}
	if v := findJSONKey(raw, "is_delisted", "isDelisted"); v == true {
		o.OfferState = "delisted"
	}
	return o
}

func findJSONKey(v any, keys ...string) any {
	wanted := map[string]bool{}
	for _, k := range keys {
		wanted[k] = true
	}
	var walk func(any) any
	walk = func(node any) any {
		switch n := node.(type) {
		case map[string]any:
			for k, v := range n {
				if wanted[k] {
					return v
				}
			}
			for _, v := range n {
				if found := walk(v); found != nil {
					return found
				}
			}
		case []any:
			for _, v := range n {
				if found := walk(v); found != nil {
					return found
				}
			}
		}
		return nil
	}
	return walk(v)
}

func firstString(m map[string]any, keys ...string) string {
	if v := firstValue(m, keys...); v != nil {
		return strings.TrimSpace(fmt.Sprint(v))
	}
	return ""
}
func firstValue(m map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v
		}
	}
	return nil
}
func canonicalJSON(v any) string { b, _ := json.Marshal(v); return string(b) }
