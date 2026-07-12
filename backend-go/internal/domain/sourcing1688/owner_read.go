package sourcing1688

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"
)

// OwnerView is the deliberately narrow, read-only representation exposed to
// Owner-facing agents. It never contains source raw payloads or listing
// published payloads.
type OwnerView struct {
	Source      OwnerSourceView   `json:"source"`
	Snapshot    OwnerSnapshotView `json:"snapshot"`
	Draft       *OwnerDraftView   `json:"draft,omitempty"`
	Listing     *OwnerListingView `json:"listing,omitempty"`
	Product     *OwnerProductView `json:"product,omitempty"`
	SKUs        []OwnerSKUView    `json:"skus"`
	Media       []OwnerMediaView  `json:"media"`
	Costs       []OwnerCostView   `json:"costs"`
	Limitations []string          `json:"limitations"`
}

type OwnerSourceView struct {
	ID              int64      `json:"id"`
	SourceReference string     `json:"source_reference"`
	Title           *string    `json:"title,omitempty"`
	Price           *float64   `json:"price,omitempty"`
	MOQ             int        `json:"moq"`
	DemandCaseID    int64      `json:"demand_case_id"`
	ExperimentID    string     `json:"experiment_id"`
	SnapshotID      int64      `json:"snapshot_id"`
	LifecycleStatus string     `json:"lifecycle_status"`
	ReviewedBy      *int64     `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
}

type OwnerSnapshotView struct {
	ID              int64     `json:"id"`
	SourceReference string    `json:"source_reference"`
	CollectedAt     time.Time `json:"collected_at"`
	Driver          string    `json:"driver"`
	ParserVersion   string    `json:"parser_version"`
	RawSHA256       string    `json:"raw_sha256"`
	ObservedTitle   *string   `json:"observed_title,omitempty"`
	ObservedPrice   *float64  `json:"observed_price,omitempty"`
	ObservedMOQ     int       `json:"observed_moq"`
}

type OwnerDraftView struct {
	ID             int64  `json:"id"`
	ProductID      int64  `json:"product_id"`
	ListingID      int64  `json:"listing_id"`
	DemandCaseID   int64  `json:"demand_case_id"`
	ExperimentID   string `json:"experiment_id"`
	SnapshotID     int64  `json:"snapshot_id"`
	ApprovalStatus string `json:"approval_status"`
	CreatedBy      int64  `json:"created_by"`
}

type OwnerListingView struct {
	ID         int64  `json:"id"`
	ProductID  int64  `json:"product_id"`
	PlatformID int64  `json:"platform_id"`
	Status     string `json:"status"`
}

type OwnerProductView struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Unit       string `json:"unit"`
	CategoryID int64  `json:"category_id"`
	Status     int16  `json:"status"`
}
type OwnerSKUView struct {
	ID        int64   `json:"id"`
	Code      string  `json:"code"`
	SpecDesc  string  `json:"spec_desc"`
	Price     float64 `json:"price"`
	CostPrice float64 `json:"cost_price"`
	Status    int16   `json:"status"`
}
type OwnerMediaView struct {
	ID              int64      `json:"id"`
	MediaRole       string     `json:"media_role"`
	RightsStatus    string     `json:"rights_status"`
	TruthStatus     string     `json:"truth_status"`
	SourceReference string     `json:"source_reference,omitempty"`
	ContentSHA256   string     `json:"content_sha256,omitempty"`
	ObservedAt      *time.Time `json:"observed_at,omitempty"`
}
type OwnerCostView struct {
	ID              int64     `json:"id"`
	CostType        string    `json:"cost_type"`
	Amount          float64   `json:"amount"`
	Currency        string    `json:"currency"`
	TruthStatus     string    `json:"truth_status"`
	SourceReference string    `json:"source_reference,omitempty"`
	ObservedAt      time.Time `json:"observed_at"`
}

type ownerSourceRow struct {
	OwnerSourceView
	SourceURL string
}
type ownerSnapshotRow struct {
	OwnerSnapshotView
	SourceURL string
}
type ownerMediaRow struct {
	OwnerMediaView
	RightsEvidenceURI string
}
type ownerCostRow struct {
	OwnerCostView
	SourceURI string
}

func ownerReadTxOptions() *sql.TxOptions {
	return &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}
}

func safeSourceReference(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return ""
	}
	return u.Host + u.EscapedPath()
}

// ReadOwnerView reads one controlled sourcing chain inside a single
// transaction. Every child query is constrained by the identities established
// by the Owner-scoped source query.
func (s *Service) ReadOwnerView(ctx context.Context, sourceID, ownerID int64) (*OwnerView, error) {
	if sourceID <= 0 || ownerID <= 0 {
		return nil, ErrWorkflowGate
	}
	var out *OwnerView
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var sourceRow ownerSourceRow
		if err := tx.Table("sourcing_1688_product AS sp").
			Select("sp.id, sp.source_url, sp.title, sp.price, sp.moq, sp.demand_case_id, sp.experiment_id, sp.snapshot_id, sp.lifecycle_status, sp.reviewed_by, sp.reviewed_at").
			Joins("JOIN demand_case dc ON dc.id = sp.demand_case_id").
			Where("sp.id = ? AND dc.owner_id = ?", sourceID, ownerID).Take(&sourceRow).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: source does not belong to authenticated Owner", ErrWorkflowGate)
			}
			return err
		}
		source := sourceRow.OwnerSourceView
		source.SourceReference = safeSourceReference(sourceRow.SourceURL)
		if source.DemandCaseID <= 0 || source.SnapshotID <= 0 || strings.TrimSpace(source.ExperimentID) == "" {
			return fmt.Errorf("%w: source lacks controlled workflow linkage", ErrWorkflowGate)
		}
		var experimentCount int64
		if err := tx.Table("experiment_case").Where("experiment_id = ? AND owner_id = ?", source.ExperimentID, ownerID).Count(&experimentCount).Error; err != nil {
			return err
		}
		if experimentCount != 1 {
			return fmt.Errorf("%w: source experiment does not belong to authenticated Owner", ErrWorkflowGate)
		}

		var snapshotRow ownerSnapshotRow
		if err := tx.Table("sourcing_1688_snapshot").
			Select("id, source_url, collected_at, driver, parser_version, raw_sha256, observed_title, observed_price, observed_moq").
			Where("id = ? AND sourcing_product_id = ?", source.SnapshotID, source.ID).Take(&snapshotRow).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: source snapshot linkage is invalid", ErrWorkflowGate)
			}
			return err
		}
		snapshot := snapshotRow.OwnerSnapshotView
		snapshot.SourceReference = safeSourceReference(snapshotRow.SourceURL)
		view := &OwnerView{Source: source, Snapshot: snapshot, SKUs: []OwnerSKUView{}, Media: []OwnerMediaView{}, Costs: []OwnerCostView{}, Limitations: []string{
			"只读视图：不能发布、采购、批准草稿或改变经营状态",
			"原始采集载荷与平台发布载荷不对小Q开放",
			"价格、成本、权利与合规状态保持其来源真实性，不能视为外部已核验事实",
		}}

		var draft OwnerDraftView
		draftErr := tx.Table("sourcing_listing_draft").
			Select("id, product_id, listing_id, demand_case_id, experiment_id, snapshot_id, approval_status, created_by").
			Where("sourcing_product_id = ?", source.ID).Take(&draft).Error
		if errors.Is(draftErr, gorm.ErrRecordNotFound) {
			view.Limitations = append(view.Limitations, "尚未生成受控内部草稿")
			out = view
			return nil
		}
		if draftErr != nil {
			return draftErr
		}
		if draft.DemandCaseID != source.DemandCaseID || draft.ExperimentID != source.ExperimentID || draft.SnapshotID != source.SnapshotID || draft.ProductID <= 0 || draft.ListingID <= 0 || draft.CreatedBy != ownerID {
			return fmt.Errorf("%w: draft linkage does not match source workflow", ErrWorkflowGate)
		}
		validState := (source.LifecycleStatus == LifecycleEditing && (draft.ApprovalStatus == "" || draft.ApprovalStatus == "editing" || draft.ApprovalStatus == "rejected")) ||
			(source.LifecycleStatus == LifecyclePendingApproval && draft.ApprovalStatus == "pending") ||
			(source.LifecycleStatus == LifecycleApprovedDraft && draft.ApprovalStatus == "approved")
		if !validState {
			return fmt.Errorf("%w: draft lifecycle and approval status disagree", ErrWorkflowGate)
		}
		view.Draft = &draft

		var listing OwnerListingView
		if err := tx.Table("product_listing").Select("id, product_id, platform_id, status").Where("id = ? AND product_id = ?", draft.ListingID, draft.ProductID).Take(&listing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: listing linkage is invalid", ErrWorkflowGate)
			}
			return err
		}
		if listing.Status != "draft" {
			return fmt.Errorf("%w: linked listing is not an internal draft", ErrWorkflowGate)
		}
		view.Listing = &listing
		var product OwnerProductView
		if err := tx.Table("product").Select("id, name, unit, category_id, status").Where("id = ?", draft.ProductID).Take(&product).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: product linkage is invalid", ErrWorkflowGate)
			}
			return err
		}
		view.Product = &product
		if err := tx.Table("sku").Select("id, code, spec_desc, price, cost_price, status").Where("product_id = ?", draft.ProductID).Order("id ASC").Scan(&view.SKUs).Error; err != nil {
			return err
		}
		var mediaRows []ownerMediaRow
		if err := tx.Table("product_media_asset").Select("id, media_role, rights_status, rights_evidence_uri, content_sha256").Where("product_id = ? AND source_snapshot_id = ?", draft.ProductID, source.SnapshotID).Order("id ASC").Scan(&mediaRows).Error; err != nil {
			return err
		}
		for _, row := range mediaRows {
			item := row.OwnerMediaView
			item.TruthStatus = "unknown"
			item.SourceReference = safeSourceReference(row.RightsEvidenceURI)
			view.Media = append(view.Media, item)
		}
		var costRows []ownerCostRow
		if err := tx.Table("product_cost_input").Select("id, cost_type, amount, currency, truth_status, source_uri, observed_at").Where("product_id = ? AND experiment_id = ?", draft.ProductID, source.ExperimentID).Order("id ASC").Scan(&costRows).Error; err != nil {
			return err
		}
		for _, row := range costRows {
			item := row.OwnerCostView
			item.SourceReference = safeSourceReference(row.SourceURI)
			view.Costs = append(view.Costs, item)
		}
		out = view
		return nil
	}, ownerReadTxOptions())
	if err != nil {
		return nil, err
	}
	return out, nil
}
