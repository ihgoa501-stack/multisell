package sourcing1688

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PrivateCollectInput is the trusted Owner-facing seam used by the browser
// extension. It creates an unverified private bookmark, not an approved
// opportunity or listing draft.
type PrivateCollectInput struct {
	OwnerID            int64           `json:"-"`
	SchemaVersion      string          `json:"schema_version" binding:"required"`
	PageOfferID        string          `json:"page_offer_id" binding:"required"`
	PriceModel         string          `json:"price_model" binding:"required"`
	RequestID          string          `json:"request_id" binding:"required"`
	SourceURL          string          `json:"source_url" binding:"required"`
	ObservedAt         time.Time       `json:"observed_at" binding:"required"`
	ParserVersion      string          `json:"parser_version" binding:"required"`
	ExtensionVersion   string          `json:"extension_version" binding:"required"`
	RawPayload         json.RawMessage `json:"raw_payload" binding:"required"`
	Title              *string         `json:"title" binding:"required"`
	Price              *float64        `json:"price"`
	MOQ                *int            `json:"moq"`
	SupplierName       string          `json:"supplier_name"`
	SupplierBusinessID string          `json:"supplier_business_id"`
	Images             json.RawMessage `json:"images"`
	SkuVariants        json.RawMessage `json:"sku_variants"`
	Attributes         json.RawMessage `json:"attributes"`
	FieldStatuses      json.RawMessage `json:"field_statuses" binding:"required"`
	ObservationIntent  string          `json:"observation_intent,omitempty"`
}

const ObservationIntentNew = "save_new_observation"

// DuplicatePrivateCollectionError requires an explicit Owner decision before
// another immutable observation is appended to an existing private product.
type DuplicatePrivateCollectionError struct {
	RecordID   int64
	SnapshotID int64
	Existing   ExistingPrivateCollectionSummary
}

func (e *DuplicatePrivateCollectionError) Error() string {
	return "this 1688 product already exists; Owner choice is required"
}

// ExistingPrivateCollectionSummary is the deliberately small, Owner-scoped
// read model returned for duplicate comparison. It must never contain the
// immutable raw payload, page HTML, cookies or extension credentials.
type ExistingPrivateCollectionSummary struct {
	Title        *string   `json:"title"`
	Price        *float64  `json:"price"`
	MOQ          *int      `json:"moq"`
	SupplierName string    `json:"supplier_name"`
	SKUCount     int       `json:"sku_count"`
	ImageCount   int       `json:"image_count"`
	ObservedAt   time.Time `json:"observed_at"`
}

func duplicateSummary(snapshot *Sourcing1688Snapshot) ExistingPrivateCollectionSummary {
	summary := ExistingPrivateCollectionSummary{
		Title: snapshot.ObservedTitle, Price: snapshot.ObservedPrice,
		SupplierName: snapshot.ObservedSupplier, ObservedAt: snapshot.CollectedAt,
	}
	var raw struct {
		MOQ           *int              `json:"min_order_qty"`
		Images        []json.RawMessage `json:"images"`
		SpecVariants  []json.RawMessage `json:"spec_variants"`
		FieldStatuses map[string]string `json:"field_statuses"`
	}
	if json.Unmarshal(snapshot.RawPayload, &raw) == nil {
		if raw.FieldStatuses["moq"] == "observed" {
			summary.MOQ = raw.MOQ
		}
		summary.ImageCount = len(raw.Images)
		summary.SKUCount = len(raw.SpecVariants)
	}
	return summary
}

func duplicateSummaryFromProduct(product *Sourcing1688Product) ExistingPrivateCollectionSummary {
	summary := ExistingPrivateCollectionSummary{
		Title: product.Title, Price: product.Price, SupplierName: product.SupplierName,
		ObservedAt: product.UpdatedAt,
	}
	if summary.ObservedAt.IsZero() {
		summary.ObservedAt = product.CreatedAt
	}
	if product.MOQ > 0 {
		moq := product.MOQ
		summary.MOQ = &moq
	}
	if product.Images != nil {
		var images []json.RawMessage
		if json.Unmarshal(*product.Images, &images) == nil {
			summary.ImageCount = len(images)
		}
	}
	if product.SkuVariants != nil {
		var variants []json.RawMessage
		if json.Unmarshal(*product.SkuVariants, &variants) == nil {
			summary.SKUCount = len(variants)
		}
	}
	return summary
}

type PrivateCollectResult struct {
	Product          Sourcing1688Product  `json:"product"`
	Snapshot         Sourcing1688Snapshot `json:"snapshot"`
	IdempotentReplay bool                 `json:"idempotent_replay"`
	NewObservation   bool                 `json:"new_observation"`
}

type PrivateCollectHTTPResult struct {
	Status           string `json:"status"`
	RecordID         int64  `json:"record_id"`
	SnapshotID       int64  `json:"snapshot_id"`
	RequestID        string `json:"request_id"`
	IdempotentReplay bool   `json:"idempotent_replay"`
	NewObservation   bool   `json:"new_observation"`
	FailureCode      string `json:"failure_code,omitempty"`
	SafeMessage      string `json:"safe_message,omitempty"`
}

type Sourcing1688TaskLink struct {
	ID                    int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SourcingProductID     int64      `gorm:"column:sourcing_product_id;not null;uniqueIndex:ux_sourcing_task_link" json:"sourcing_product_id"`
	DemandCaseID          int64      `gorm:"column:demand_case_id;not null" json:"demand_case_id"`
	ExperimentID          string     `gorm:"column:experiment_id;not null;uniqueIndex:ux_sourcing_task_link" json:"experiment_id"`
	OwnerID               int64      `gorm:"column:owner_id;not null;index" json:"owner_id"`
	ProductOpportunityID  *int64     `gorm:"column:product_opportunity_id" json:"product_opportunity_id,omitempty"`
	OpportunityDecisionID *int64     `gorm:"column:opportunity_decision_id" json:"opportunity_decision_id,omitempty"`
	AuthorityKind         string     `gorm:"column:authority_kind;size:24;not null;default:legacy_experiment" json:"authority_kind"`
	Status                string     `gorm:"column:status;size:24;not null;default:linked" json:"status"`
	IsPrimary             bool       `gorm:"column:is_primary;not null;default:false" json:"is_primary"`
	BlockedReason         string     `gorm:"column:blocked_reason;size:500;not null;default:''" json:"blocked_reason,omitempty"`
	SamplePolicy          string     `gorm:"column:sample_policy;size:16;not null;default:required" json:"sample_policy"`
	SampleWaiverReason    string     `gorm:"column:sample_waiver_reason;type:text;not null;default:''" json:"sample_waiver_reason,omitempty"`
	SampleWaivedBy        *int64     `gorm:"column:sample_waived_by" json:"sample_waived_by,omitempty"`
	SampleWaivedAt        *time.Time `gorm:"column:sample_waived_at" json:"sample_waived_at,omitempty"`
	WorkflowStatus        string     `gorm:"column:workflow_status;size:32;not null;default:needs_review" json:"workflow_status"`
	DraftID               *int64     `gorm:"column:draft_id" json:"draft_id,omitempty"`
	WorkflowUpdatedAt     time.Time  `gorm:"column:workflow_updated_at;autoUpdateTime" json:"workflow_updated_at"`
	CreatedAt             time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (Sourcing1688TaskLink) TableName() string { return "sourcing_1688_task_link" }

type LinkPrivateTaskInput struct {
	OwnerID              int64  `json:"-"`
	DemandCaseID         int64  `json:"demand_case_id" binding:"required"`
	ExperimentID         string `json:"experiment_id" binding:"required"`
	ProductOpportunityID int64  `json:"product_opportunity_id" binding:"required"`
}

type LinkPrivateTaskResult struct {
	Product Sourcing1688Product  `json:"product"`
	Link    Sourcing1688TaskLink `json:"link"`
}

type PrivateWorkcopyInput struct {
	OwnerID           int64     `json:"-"`
	ExpectedUpdatedAt time.Time `json:"expected_updated_at" binding:"required"`
	Title             string    `json:"title" binding:"required"`
	Price             *float64  `json:"price"`
	MOQ               *int      `json:"moq"`
	SupplierName      *string   `json:"supplier_name"`
	Notes             string    `json:"notes"`
}

func (s *Service) UpdatePrivateWorkcopy(productID int64, in *PrivateWorkcopyInput) (*Sourcing1688Product, error) {
	if productID <= 0 || in == nil || in.OwnerID <= 0 || in.ExpectedUpdatedAt.IsZero() || strings.TrimSpace(in.Title) == "" || len([]rune(in.Title)) > 500 || len([]rune(in.Notes)) > 4000 {
		return nil, ErrInvalidWorkflow
	}
	var updated Sourcing1688Product
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var product Sourcing1688Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND owner_id = ?", productID, in.OwnerID).First(&product).Error; err != nil {
			return fmt.Errorf("%w: private source not found", ErrWorkflowGate)
		}
		if product.ExperimentID != nil || product.DemandCaseID != nil || (product.LifecycleStatus != LifecycleUnverifiedLead && product.LifecycleStatus != LifecycleNeedsReview) {
			return fmt.Errorf("%w: linked or governed source must be edited through its controlled draft", ErrWorkflowGate)
		}
		if !product.UpdatedAt.UTC().Equal(in.ExpectedUpdatedAt.UTC()) {
			return fmt.Errorf("%w: private workcopy changed; reload before saving", ErrWorkflowGate)
		}
		now := time.Now().UTC()
		values := map[string]any{"title": strings.TrimSpace(in.Title), "review_notes": strings.TrimSpace(in.Notes),
			"lifecycle_status": LifecycleNeedsReview, "lifecycle_actor_id": in.OwnerID, "lifecycle_reason": "private_workcopy_edited", "lifecycle_updated_at": now, "updated_at": now}
		if in.Price != nil {
			if *in.Price < 0 {
				return ErrInvalidWorkflow
			}
			values["price"] = *in.Price
		}
		if in.MOQ != nil {
			if *in.MOQ < 0 {
				return ErrInvalidWorkflow
			}
			values["moq"] = *in.MOQ
		}
		if in.SupplierName != nil {
			values["supplier_name"] = strings.TrimSpace(*in.SupplierName)
		}
		result := tx.Model(&Sourcing1688Product{}).Where("id = ? AND owner_id = ? AND updated_at = ?", productID, in.OwnerID, product.UpdatedAt).Updates(values)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("%w: private workcopy changed; reload before saving", ErrWorkflowGate)
		}
		return tx.First(&updated, productID).Error
	})
	return &updated, err
}

// SetPrivateArchive changes only an unlinked Owner bookmark. Immutable
// observations stay intact; anything that entered a task or draft chain is
// deliberately ineligible for this ordinary采集箱 action.
func (s *Service) SetPrivateArchive(productID, ownerID int64, archived bool) (*Sourcing1688Product, error) {
	if productID <= 0 || ownerID <= 0 {
		return nil, ErrInvalidWorkflow
	}
	var updated Sourcing1688Product
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var product Sourcing1688Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND owner_id = ?", productID, ownerID).First(&product).Error; err != nil {
			return fmt.Errorf("%w: private source not found", ErrWorkflowGate)
		}
		var taskLinks, drafts int64
		if err := tx.Model(&Sourcing1688TaskLink{}).Where("sourcing_product_id = ?", productID).Count(&taskLinks).Error; err != nil {
			return err
		}
		if err := tx.Model(&draftRow{}).Where("sourcing_product_id = ?", productID).Count(&drafts).Error; err != nil {
			return err
		}
		if product.DemandCaseID != nil || product.ExperimentID != nil || product.ProductID != nil || taskLinks > 0 || drafts > 0 {
			return fmt.Errorf("%w: linked or governed source can only be archived through its controlled workflow", ErrWorkflowGate)
		}
		if archived {
			if product.LifecycleStatus != LifecycleUnverifiedLead && product.LifecycleStatus != LifecycleNeedsReview {
				return fmt.Errorf("%w: only an active private bookmark can be archived", ErrWorkflowGate)
			}
		} else if product.LifecycleStatus != LifecycleArchived {
			return fmt.Errorf("%w: only an archived private bookmark can be restored", ErrWorkflowGate)
		}
		now := time.Now().UTC()
		next := LifecycleNeedsReview
		reason := "private_bookmark_restored"
		if archived {
			next, reason = LifecycleArchived, "private_bookmark_archived"
		}
		if err := tx.Model(&product).Updates(map[string]any{
			"status": next, "lifecycle_status": next, "lifecycle_actor_id": ownerID,
			"lifecycle_reason": reason, "lifecycle_updated_at": now, "updated_at": now,
		}).Error; err != nil {
			return err
		}
		return tx.First(&updated, productID).Error
	})
	return &updated, err
}

type Sourcing1688TaskLinkView struct {
	Sourcing1688TaskLink
	Label          string `json:"label"`
	CurrentStatus  string `json:"current_status"`
	CurrentBlocker string `json:"current_blocker,omitempty"`
}

type EligibleSourcingTask struct {
	ExperimentID         string `json:"experiment_id"`
	DemandCaseID         int64  `json:"demand_case_id"`
	Region               string `json:"region"`
	Consumer             string `json:"consumer"`
	NeedScenario         string `json:"need_scenario"`
	SalesChannel         string `json:"sales_channel"`
	Stage                string `json:"stage"`
	Label                string `json:"label"`
	ProductOpportunityID int64  `json:"product_opportunity_id"`
	OpportunityTitle     string `json:"opportunity_title"`
}

type sourcingOpportunityRow struct {
	ID, OwnerID, DemandCaseID, MarketDecisionID, Version int64
	Title, TargetChannel, Status, ContentHash            string
}

func (sourcingOpportunityRow) TableName() string { return "product_opportunity" }

type sourcingOpportunityDecisionRow struct {
	ID, OpportunityID, OwnerID, Version int64
	Decision, ContentHash               string
}

func (sourcingOpportunityDecisionRow) TableName() string { return "product_opportunity_decision" }

type sourcingMarketDecisionRow struct {
	ID, DemandCaseID, OwnerID int64
	Decision                  string
}

func (sourcingMarketDecisionRow) TableName() string { return "market_owner_decision" }

func requireSourcingAuthority(tx *gorm.DB, ownerID, demandCaseID, opportunityID int64) (*sourcingOpportunityDecisionRow, error) {
	var opportunity sourcingOpportunityRow
	if err := tx.Where("id = ? AND owner_id = ? AND demand_case_id = ?", opportunityID, ownerID, demandCaseID).First(&opportunity).Error; err != nil {
		return nil, fmt.Errorf("%w: approved product opportunity not found", ErrWorkflowGate)
	}
	if opportunity.Status != "approved" {
		return nil, fmt.Errorf("%w: product opportunity is not approved", ErrWorkflowGate)
	}
	var latestMarket sourcingMarketDecisionRow
	if err := tx.Where("demand_case_id = ? AND owner_id = ?", demandCaseID, ownerID).Order("id DESC").First(&latestMarket).Error; err != nil || latestMarket.ID != opportunity.MarketDecisionID || latestMarket.Decision != "selected" {
		return nil, fmt.Errorf("%w: latest Owner market decision is not selected", ErrWorkflowGate)
	}
	var decision sourcingOpportunityDecisionRow
	if err := tx.Where("opportunity_id = ? AND owner_id = ? AND version = ? AND content_hash = ? AND decision = 'approved'", opportunity.ID, ownerID, opportunity.Version, opportunity.ContentHash).Order("id DESC").First(&decision).Error; err != nil {
		return nil, fmt.Errorf("%w: frozen Owner opportunity approval is required", ErrWorkflowGate)
	}
	var dc demandCaseRow
	if err := tx.First(&dc, demandCaseID).Error; err != nil || dc.OwnerID != ownerID || !strings.EqualFold(strings.TrimSpace(dc.SalesChannel), strings.TrimSpace(opportunity.TargetChannel)) {
		return nil, fmt.Errorf("%w: opportunity channel and selected market must match", ErrWorkflowGate)
	}
	return &decision, nil
}

func requirePrimarySourcingAuthority(tx *gorm.DB, ownerID, demandCaseID int64, experimentID string) error {
	var link Sourcing1688TaskLink
	if err := tx.Where("owner_id = ? AND demand_case_id = ? AND experiment_id = ? AND is_primary = ? AND authority_kind = ?", ownerID, demandCaseID, experimentID, true, "product_opportunity").First(&link).Error; err != nil || link.ProductOpportunityID == nil || link.OpportunityDecisionID == nil {
		return fmt.Errorf("%w: approved product opportunity authority is required", ErrWorkflowGate)
	}
	decision, err := requireSourcingAuthority(tx, ownerID, demandCaseID, *link.ProductOpportunityID)
	if err != nil {
		return err
	}
	if decision.ID != *link.OpportunityDecisionID {
		return fmt.Errorf("%w: frozen opportunity approval changed", ErrWorkflowGate)
	}
	return nil
}

// requireTaskSourcingAuthority resolves the exact task that owns a conversion.
// taskLinkID=0 is the compatibility path for old callers and resolves only the
// frozen primary link; new callers must send the explicit task link identity.
func findOwnedTaskLink(tx *gorm.DB, sourceID, ownerID, taskLinkID int64) (*Sourcing1688TaskLink, error) {
	if sourceID <= 0 || ownerID <= 0 || taskLinkID < 0 {
		return nil, ErrInvalidWorkflow
	}
	query := tx.Where("sourcing_product_id = ? AND owner_id = ?", sourceID, ownerID)
	if taskLinkID > 0 {
		query = query.Where("id = ?", taskLinkID)
	} else {
		query = query.Where("is_primary = ?", true)
	}
	var link Sourcing1688TaskLink
	if err := query.First(&link).Error; err != nil {
		return nil, fmt.Errorf("%w: sourcing task link does not belong to authenticated Owner", ErrWorkflowGate)
	}
	return &link, nil
}

func requireTaskSourcingAuthority(tx *gorm.DB, sourceID, ownerID, taskLinkID int64) (*Sourcing1688TaskLink, error) {
	link, err := findOwnedTaskLink(tx, sourceID, ownerID, taskLinkID)
	if err != nil {
		return nil, err
	}
	if link.Status == "archived" || link.WorkflowStatus == "archived" {
		return nil, fmt.Errorf("%w: sourcing task link is archived", ErrWorkflowGate)
	}
	if link.AuthorityKind != "product_opportunity" || link.ProductOpportunityID == nil || link.OpportunityDecisionID == nil {
		return nil, fmt.Errorf("%w: approved product opportunity authority is required", ErrWorkflowGate)
	}
	var dc demandCaseRow
	if err := tx.First(&dc, link.DemandCaseID).Error; err != nil || dc.OwnerID != ownerID {
		return nil, fmt.Errorf("%w: selected market is no longer eligible for sourcing", ErrWorkflowGate)
	}
	decision, err := requireSourcingAuthority(tx, ownerID, link.DemandCaseID, *link.ProductOpportunityID)
	if err != nil {
		return nil, err
	}
	if decision.ID != *link.OpportunityDecisionID {
		return nil, fmt.Errorf("%w: frozen opportunity approval changed", ErrWorkflowGate)
	}
	return link, nil
}

func validatePrivateCollectInput(in *PrivateCollectInput) (string, string, string, string, string, error) {
	if in == nil || in.OwnerID <= 0 || strings.TrimSpace(in.RequestID) == "" ||
		in.ObservedAt.IsZero() || strings.TrimSpace(in.ParserVersion) == "" ||
		strings.TrimSpace(in.ExtensionVersion) == "" || len(in.RawPayload) == 0 ||
		!json.Valid(in.RawPayload) || !json.Valid(in.FieldStatuses) || in.Title == nil || strings.TrimSpace(*in.Title) == "" {
		return "", "", "", "", "", fmt.Errorf("%w: owner, request, page identity, title, versions, timestamp and raw payload are required", ErrInvalidWorkflow)
	}
	if in.SchemaVersion != "sourcing1688.private.v1" || !strings.HasPrefix(in.ExtensionVersion, "0.2.") {
		return "", "", "", "", "", fmt.Errorf("%w: unsupported extension or schema version", ErrInvalidWorkflow)
	}
	if in.PriceModel != "fixed" && in.PriceModel != "range" && in.PriceModel != "tiered" && in.PriceModel != "sku" && in.PriceModel != "unknown" {
		return "", "", "", "", "", fmt.Errorf("%w: invalid price model", ErrInvalidWorkflow)
	}
	if intent := strings.TrimSpace(in.ObservationIntent); intent != "" && intent != ObservationIntentNew {
		return "", "", "", "", "", fmt.Errorf("%w: invalid observation intent", ErrInvalidWorkflow)
	}
	if err := validatePrivatePayloadLimits(in); err != nil {
		return "", "", "", "", "", err
	}
	if !strings.HasPrefix(strings.TrimSpace(in.RequestID), "collect_") {
		return "", "", "", "", "", fmt.Errorf("%w: extension collection request must start with collect_", ErrInvalidWorkflow)
	}
	canonicalURL, err := canonical1688URL(in.SourceURL)
	if err != nil {
		return "", "", "", "", "", err
	}
	offerID := offerIDFromURL(canonicalURL)
	if offerID == "" {
		return "", "", "", "", "", fmt.Errorf("%w: a 1688 product offer URL is required", ErrInvalidWorkflow)
	}
	if strings.TrimSpace(in.PageOfferID) != offerID {
		return "", "", "", "", "", fmt.Errorf("%w: URL and page offer identity conflict", ErrInvalidWorkflow)
	}
	var rawObject map[string]json.RawMessage
	if json.Unmarshal(in.RawPayload, &rawObject) != nil {
		return "", "", "", "", "", fmt.Errorf("%w: raw payload must be a structured object", ErrInvalidWorkflow)
	}
	for key := range rawObject {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if normalized == "raw_html" || normalized == "raw_text" || strings.Contains(normalized, "password") || strings.Contains(normalized, "cookie") || strings.Contains(normalized, "authorization") || strings.Contains(normalized, "secret") {
			return "", "", "", "", "", fmt.Errorf("%w: private collection raw payload contains a forbidden field", ErrInvalidWorkflow)
		}
	}
	var rawIdentity struct {
		SchemaVersion      string          `json:"schema_version"`
		OfferIDURL         string          `json:"offer_id_url"`
		OfferIDPage        string          `json:"offer_id_page"`
		SourceURL          string          `json:"source_url"`
		Title              string          `json:"title"`
		Price              *float64        `json:"price_1688"`
		PriceModel         string          `json:"price_model"`
		MOQ                *int            `json:"min_order_qty"`
		SupplierName       string          `json:"supplier_name"`
		SupplierBusinessID string          `json:"supplier_business_id"`
		Images             json.RawMessage `json:"images"`
		SKUVariants        json.RawMessage `json:"spec_variants"`
		Attributes         json.RawMessage `json:"attributes"`
		FieldStatuses      json.RawMessage `json:"field_statuses"`
	}
	if json.Unmarshal(in.RawPayload, &rawIdentity) != nil || rawIdentity.SchemaVersion != in.SchemaVersion || rawIdentity.OfferIDURL != offerID || rawIdentity.OfferIDPage != offerID || rawIdentity.PriceModel != in.PriceModel || strings.TrimSpace(rawIdentity.Title) != strings.TrimSpace(*in.Title) {
		return "", "", "", "", "", fmt.Errorf("%w: raw page identity and structured fields conflict", ErrInvalidWorkflow)
	}
	rawCanonicalURL, rawURLErr := canonical1688URL(rawIdentity.SourceURL)
	if rawURLErr != nil || rawCanonicalURL != canonicalURL || len(rawIdentity.FieldStatuses) == 0 {
		return "", "", "", "", "", fmt.Errorf("%w: raw page URL or field status conflict", ErrInvalidWorkflow)
	}
	if in.Price != nil && (rawIdentity.Price == nil || *rawIdentity.Price != *in.Price) || in.MOQ != nil && (rawIdentity.MOQ == nil || *rawIdentity.MOQ != *in.MOQ) ||
		strings.TrimSpace(rawIdentity.SupplierName) != strings.TrimSpace(in.SupplierName) || strings.TrimSpace(rawIdentity.SupplierBusinessID) != strings.TrimSpace(in.SupplierBusinessID) {
		return "", "", "", "", "", fmt.Errorf("%w: raw and structured commercial fields conflict", ErrInvalidWorkflow)
	}
	sameJSON := func(left, right json.RawMessage) bool {
		var a, b any
		if len(left) == 0 {
			left = []byte("null")
		}
		if len(right) == 0 {
			right = []byte("null")
		}
		if json.Unmarshal(left, &a) != nil || json.Unmarshal(right, &b) != nil {
			return false
		}
		aa, _ := json.Marshal(a)
		bb, _ := json.Marshal(b)
		return string(aa) == string(bb)
	}
	if len(in.Images) > 0 && !sameJSON(rawIdentity.Images, in.Images) || len(in.SkuVariants) > 0 && !sameJSON(rawIdentity.SKUVariants, in.SkuVariants) || len(in.Attributes) > 0 && !sameJSON(rawIdentity.Attributes, in.Attributes) {
		return "", "", "", "", "", fmt.Errorf("%w: raw and structured collection arrays conflict", ErrInvalidWorkflow)
	}
	var statuses map[string]string
	if json.Unmarshal(in.FieldStatuses, &statuses) != nil {
		return "", "", "", "", "", ErrInvalidWorkflow
	}
	var rawStatuses map[string]string
	if json.Unmarshal(rawIdentity.FieldStatuses, &rawStatuses) != nil || len(rawStatuses) != len(statuses) {
		return "", "", "", "", "", fmt.Errorf("%w: raw and structured field statuses conflict", ErrInvalidWorkflow)
	}
	for key, value := range statuses {
		if rawStatuses[key] != value {
			return "", "", "", "", "", fmt.Errorf("%w: raw and structured field statuses conflict", ErrInvalidWorkflow)
		}
	}
	for _, key := range []string{"title", "price", "moq", "supplier", "images", "sku"} {
		value := statuses[key]
		if value != "observed" && value != "unknown" && value != "parse_failed" && value != "no_sku" {
			return "", "", "", "", "", fmt.Errorf("%w: invalid field status for %s", ErrInvalidWorkflow, key)
		}
	}
	if statuses["title"] != "observed" || (statuses["price"] == "observed") != (in.Price != nil) ||
		(statuses["moq"] == "observed") != (in.MOQ != nil && *in.MOQ > 0) ||
		(statuses["supplier"] == "observed") != (strings.TrimSpace(in.SupplierName) != "" || strings.TrimSpace(in.SupplierBusinessID) != "") ||
		(statuses["images"] == "observed") != (len(in.Images) > 0) {
		return "", "", "", "", "", fmt.Errorf("%w: structured values do not match field statuses", ErrInvalidWorkflow)
	}
	if statuses["sku"] == "no_sku" && len(in.SkuVariants) > 0 || statuses["sku"] == "observed" && len(in.SkuVariants) == 0 {
		return "", "", "", "", "", fmt.Errorf("%w: SKU values do not match field status", ErrInvalidWorkflow)
	}
	digest := sha256.Sum256(in.RawPayload)
	rawHash := hex.EncodeToString(digest[:])
	canonical := func(raw json.RawMessage) any {
		var value any
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &value)
		}
		return value
	}
	structuredBytes, _ := json.Marshal(struct {
		Title                                          *string  `json:"title"`
		Price                                          *float64 `json:"price"`
		PriceModel                                     string   `json:"price_model"`
		MOQ                                            *int     `json:"moq"`
		SupplierName                                   string   `json:"supplier_name"`
		SupplierBusinessID                             string   `json:"supplier_business_id"`
		Images, SKUVariants, Attributes, FieldStatuses any
	}{
		in.Title, in.Price, in.PriceModel, in.MOQ, strings.TrimSpace(in.SupplierName), strings.TrimSpace(in.SupplierBusinessID), canonical(in.Images), canonical(in.SkuVariants), canonical(in.Attributes), canonical(in.FieldStatuses)})
	structuredDigest := sha256.Sum256(structuredBytes)
	structuredHash := hex.EncodeToString(structuredDigest[:])
	envelopeBytes, _ := json.Marshal(struct {
		OwnerID                                                                                                                               int64
		RequestID, SourceURL, OfferID, SchemaVersion, ParserVersion, ExtensionVersion, ObservedAt, RawHash, StructuredHash, ObservationIntent string
	}{
		in.OwnerID, strings.TrimSpace(in.RequestID), canonicalURL, offerID, in.SchemaVersion, strings.TrimSpace(in.ParserVersion), strings.TrimSpace(in.ExtensionVersion), in.ObservedAt.UTC().Format(time.RFC3339Nano), rawHash, structuredHash, strings.TrimSpace(in.ObservationIntent)})
	envelopeDigest := sha256.Sum256(envelopeBytes)
	return canonicalURL, offerID, rawHash, structuredHash, hex.EncodeToString(envelopeDigest[:]), nil
}

func validatePrivatePayloadLimits(in *PrivateCollectInput) error {
	if len(in.RawPayload) > 1<<20 || len(in.SkuVariants) > 512<<10 || len(in.Attributes) > 256<<10 || len(in.Images) > 128<<10 || len(in.FieldStatuses) > 4<<10 ||
		len(in.SourceURL) > 2048 || len(in.RequestID) > 80 || len([]rune(*in.Title)) > 500 || len([]rune(in.SupplierName)) > 300 || len(in.SupplierBusinessID) > 300 {
		return fmt.Errorf("%w: private collection payload exceeds safe field limits", ErrInvalidWorkflow)
	}
	if len(in.Images) > 0 {
		var images []string
		if json.Unmarshal(in.Images, &images) != nil || len(images) > 100 {
			return fmt.Errorf("%w: invalid or excessive image list", ErrInvalidWorkflow)
		}
		for _, image := range images {
			parsed, err := url.Parse(image)
			host := strings.ToLower(parsed.Hostname())
			if err != nil || parsed.Scheme != "https" || len(image) > 2048 || !(host == "1688.com" || strings.HasSuffix(host, ".1688.com") || host == "alicdn.com" || strings.HasSuffix(host, ".alicdn.com")) {
				return fmt.Errorf("%w: unapproved image URL", ErrInvalidWorkflow)
			}
		}
	}
	if len(in.SkuVariants) > 0 {
		var variants []map[string]any
		if json.Unmarshal(in.SkuVariants, &variants) != nil || len(variants) > 2000 {
			return fmt.Errorf("%w: invalid or excessive SKU list", ErrInvalidWorkflow)
		}
		for _, variant := range variants {
			if spec, _ := variant["spec"].(string); len([]rune(spec)) > 500 {
				return fmt.Errorf("%w: SKU text too long", ErrInvalidWorkflow)
			}
		}
	}
	if len(in.Attributes) > 0 {
		var attributes map[string]string
		if json.Unmarshal(in.Attributes, &attributes) != nil || len(attributes) > 500 {
			return fmt.Errorf("%w: invalid or excessive attributes", ErrInvalidWorkflow)
		}
		for key, value := range attributes {
			if len([]rune(key)) > 200 || len([]rune(value)) > 2000 {
				return fmt.Errorf("%w: attribute text too long", ErrInvalidWorkflow)
			}
		}
	}
	return nil
}

// CollectPrivate stores one Owner-triggered 1688 page observation. It does not
// require or create a demand-case/experiment link; those gates remain on the
// later promotion into the controlled sourcing draft workflow.
func (s *Service) CollectPrivate(in *PrivateCollectInput) (*PrivateCollectResult, error) {
	canonicalURL, offerID, rawHash, structuredHash, envelopeHash, err := validatePrivateCollectInput(in)
	if err != nil {
		// Validation failures happen before the collection transaction. Preserve a
		// separate safe operational record when the request envelope is sufficient;
		// never replace the original validation error if audit persistence fails.
		if failure := privateCollectFailureInput(in, err); failure != nil {
			_, _, _ = s.RecordPrivateCaptureFailure(failure)
		}
		// RecordPrivateCaptureFailure also creates the durable not_saved receipt
		// when enough request identity is present.
		return nil, err
	}
	receipt, err := s.beginPrivateCollectionRequest(in.OwnerID, in.RequestID, envelopeHash)
	if err != nil {
		return nil, err
	}
	if receipt.Status == PrivateRequestSaved {
		var snapshot Sourcing1688Snapshot
		if err := s.db.First(&snapshot, receipt.SnapshotID).Error; err != nil {
			return nil, err
		}
		var product Sourcing1688Product
		if err := s.db.First(&product, receipt.RecordID).Error; err != nil {
			return nil, err
		}
		return &PrivateCollectResult{Product: product, Snapshot: snapshot, IdempotentReplay: true}, nil
	}

	var result PrivateCollectResult
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var replay Sourcing1688Snapshot
		err := tx.Where("collection_request_id = ?", strings.TrimSpace(in.RequestID)).First(&replay).Error
		if err == nil {
			if replay.CollectedBy != in.OwnerID || replay.RequestEnvelopeSHA256 != envelopeHash {
				return fmt.Errorf("%w: collection request identity or payload conflict", ErrInvalidWorkflow)
			}
			var product Sourcing1688Product
			if err := tx.First(&product, replay.SourcingProductID).Error; err != nil {
				return err
			}
			result = PrivateCollectResult{Product: product, Snapshot: replay, IdempotentReplay: true}
			return markPrivateRequest(tx, in.OwnerID, in.RequestID, PrivateRequestSaved, "", "", &product.ID, &replay.ID)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var product Sourcing1688Product
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("owner_id = ? AND source_offer_id = ?", in.OwnerID, offerID).
			First(&product).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			product = Sourcing1688Product{
				OwnerID: in.OwnerID, SourceURL: canonicalURL, SourceOfferID: offerID,
				Title: in.Title, Price: in.Price, MOQ: privateMOQ(in.MOQ),
				SupplierName:       strings.TrimSpace(in.SupplierName),
				SupplierBusinessID: strings.TrimSpace(in.SupplierBusinessID),
				Status:             StatusUnverifiedLead, LifecycleStatus: LifecycleUnverifiedLead,
			}
			setPrivateCollectionJSON(&product, in)
			if err := tx.Create(&product).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			if strings.TrimSpace(in.ObservationIntent) != ObservationIntentNew {
				var latest Sourcing1688Snapshot
				latestErr := tx.Where("sourcing_product_id = ? AND collected_by = ?", product.ID, in.OwnerID).Order("id DESC").First(&latest).Error
				if latestErr == nil {
					return &DuplicatePrivateCollectionError{RecordID: product.ID, SnapshotID: latest.ID, Existing: duplicateSummary(&latest)}
				}
				if !errors.Is(latestErr, gorm.ErrRecordNotFound) {
					return latestErr
				}
				return &DuplicatePrivateCollectionError{RecordID: product.ID, Existing: duplicateSummaryFromProduct(&product)}
			}
			// A later approved workflow is never mutated or downgraded by a private
			// recapture. The new observation is stored, but its governed snapshot
			// pointer and working fields remain frozen on the controlled workflow.
			if product.DemandCaseID == nil && product.ExperimentID == nil {
				product.SourceURL, product.Title = canonicalURL, in.Title
				product.Price, product.MOQ = in.Price, privateMOQ(in.MOQ)
				product.SupplierName = strings.TrimSpace(in.SupplierName)
				product.SupplierBusinessID = strings.TrimSpace(in.SupplierBusinessID)
				setPrivateCollectionJSON(&product, in)
				product.Status, product.LifecycleStatus = StatusUnverifiedLead, LifecycleUnverifiedLead
				if err := tx.Save(&product).Error; err != nil {
					return err
				}
			}
		}

		fingerprint, err := productFingerprint(strings.TrimSpace(*in.Title), in.SkuVariants)
		if err != nil {
			return err
		}
		product.SourceProductFingerprint = fingerprint
		snapshot := Sourcing1688Snapshot{
			SourcingProductID: product.ID, SourceURL: canonicalURL,
			CollectedAt: in.ObservedAt.UTC(), CollectedBy: in.OwnerID,
			Driver: "chrome_extension", ParserVersion: strings.TrimSpace(in.ParserVersion),
			ExtensionVersion: strings.TrimSpace(in.ExtensionVersion),
			SchemaVersion:    strings.TrimSpace(in.SchemaVersion),
			CaptureMode:      CaptureModeExtensionClick, CollectionRequestID: strings.TrimSpace(in.RequestID),
			RawPayload: append(json.RawMessage(nil), in.RawPayload...), RawSHA256: rawHash,
			StructuredDataSHA256: structuredHash, RequestEnvelopeSHA256: envelopeHash,
			ObservedTitle: in.Title, ObservedPrice: in.Price, ObservedMOQ: privateMOQ(in.MOQ),
			ObservedSupplier:           strings.TrimSpace(in.SupplierName),
			ObservedSupplierBusinessID: strings.TrimSpace(in.SupplierBusinessID), ProductFingerprint: fingerprint,
		}
		if err := tx.Create(&snapshot).Error; err != nil {
			return err
		}
		if product.DemandCaseID == nil && product.ExperimentID == nil {
			if err := tx.Model(&product).Updates(map[string]any{
				"snapshot_id": snapshot.ID, "source_product_fingerprint": fingerprint,
			}).Error; err != nil {
				return err
			}
			product.SnapshotID = &snapshot.ID
			product.SourceProductFingerprint = fingerprint
		}
		result = PrivateCollectResult{Product: product, Snapshot: snapshot, NewObservation: true}
		return markPrivateRequest(tx, in.OwnerID, in.RequestID, PrivateRequestSaved, "", "", &product.ID, &snapshot.ID)
	})
	if err != nil {
		// A concurrent retry may have committed the same immutable request first.
		// Re-read the receipt before classifying the outcome as uncertain.
		if state, stateErr := s.GetPrivateCollectionRequest(in.OwnerID, in.RequestID); stateErr == nil && state.Status == PrivateRequestSaved {
			var snapshot Sourcing1688Snapshot
			if loadErr := s.db.First(&snapshot, state.SnapshotID).Error; loadErr == nil && snapshot.RequestEnvelopeSHA256 == envelopeHash {
				var product Sourcing1688Product
				if loadErr = s.db.First(&product, state.RecordID).Error; loadErr == nil {
					return &PrivateCollectResult{Product: product, Snapshot: snapshot, IdempotentReplay: true}, nil
				}
			}
		}
		s.markPrivateRequestFailure(in, err)
	}
	return &result, err
}

func privateMOQ(value *int) int {
	if value == nil || *value < 0 {
		return 0
	}
	return *value
}

func setPrivateCollectionJSON(product *Sourcing1688Product, in *PrivateCollectInput) {
	if len(in.Images) > 0 {
		value := json.RawMessage(append([]byte(nil), in.Images...))
		product.Images = &value
	}
	if len(in.SkuVariants) > 0 {
		value := json.RawMessage(append([]byte(nil), in.SkuVariants...))
		product.SkuVariants = &value
	}
	if len(in.Attributes) > 0 {
		value := json.RawMessage(append([]byte(nil), in.Attributes...))
		product.Attributes = &value
	}
}

// LinkPrivateToTask promotes a private bookmark into the existing governed
// sourcing workflow. Collection itself stays easy; the business gate lives at
// this later boundary.
func (s *Service) LinkPrivateToTask(productID int64, in *LinkPrivateTaskInput) (*LinkPrivateTaskResult, error) {
	if productID <= 0 || in == nil || in.OwnerID <= 0 || in.DemandCaseID <= 0 || in.ProductOpportunityID <= 0 || strings.TrimSpace(in.ExperimentID) == "" {
		return nil, ErrInvalidWorkflow
	}
	var result LinkPrivateTaskResult
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var product Sourcing1688Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND owner_id = ?", productID, in.OwnerID).First(&product).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: private source does not belong to authenticated Owner", ErrWorkflowGate)
			}
			return err
		}

		var dc demandCaseRow
		if err := tx.First(&dc, in.DemandCaseID).Error; err != nil {
			return fmt.Errorf("demand case: %w", err)
		}
		var exp experimentRow
		if err := tx.Where("experiment_id = ?", in.ExperimentID).First(&exp).Error; err != nil {
			return fmt.Errorf("experiment: %w", err)
		}
		if dc.OwnerID != in.OwnerID || dc.Status != "experiment_ready" || exp.OwnerID != in.OwnerID || exp.Status != "active" ||
			(exp.Stage != "product" && exp.Stage != "supply" && exp.Stage != "channel") {
			return fmt.Errorf("%w: selected task is not eligible for sourcing", ErrWorkflowGate)
		}
		var gateCount, objectLinkCount int64
		if err := tx.Model(&gateRow{}).Where("experiment_id = ? AND stage = ? AND result = ?", in.ExperimentID, "opportunity", "pass").Count(&gateCount).Error; err != nil {
			return err
		}
		if err := tx.Model(&objectLinkRow{}).
			Where("experiment_id = ? AND object_type = ? AND object_id = ?", in.ExperimentID, "demand_case", strconv.FormatInt(in.DemandCaseID, 10)).
			Count(&objectLinkCount).Error; err != nil {
			return err
		}
		if gateCount == 0 || objectLinkCount == 0 {
			return fmt.Errorf("%w: experiment trace is incomplete; it does not authorize sourcing", ErrWorkflowGate)
		}
		approval, err := requireSourcingAuthority(tx, in.OwnerID, in.DemandCaseID, in.ProductOpportunityID)
		if err != nil {
			return err
		}

		link := Sourcing1688TaskLink{
			SourcingProductID: product.ID, DemandCaseID: in.DemandCaseID,
			ExperimentID: strings.TrimSpace(in.ExperimentID), OwnerID: in.OwnerID,
			Status: "linked", IsPrimary: product.ExperimentID == nil, WorkflowStatus: "needs_review",
			ProductOpportunityID: &in.ProductOpportunityID, OpportunityDecisionID: &approval.ID, AuthorityKind: "product_opportunity",
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&link).Error; err != nil {
			return err
		}
		if link.ID == 0 {
			if err := tx.Where("sourcing_product_id = ? AND experiment_id = ?", product.ID, in.ExperimentID).First(&link).Error; err != nil {
				return err
			}
		}
		// The product's legacy single-task fields represent only the first active
		// governed working copy. Additional task links are independent evidence
		// associations and must never overwrite or downgrade that working copy.
		if product.ExperimentID == nil {
			link.Status = "active_workflow"
			if err := tx.Model(&link).Updates(map[string]any{"status": link.Status, "is_primary": true}).Error; err != nil {
				return err
			}
			product.DemandCaseID, product.ExperimentID = &in.DemandCaseID, &in.ExperimentID
			product.Status, product.LifecycleStatus = StatusPendingReview, LifecyclePendingReview
			product.LifecycleUpdatedAt = pointerTime(time.Now().UTC())
			if err := tx.Save(&product).Error; err != nil {
				return err
			}
		} else if product.ReviewedBy != nil && product.SnapshotID != nil && (product.Status == StatusReviewed || product.Status == StatusDraftCreated) {
			link.WorkflowStatus = "ready_for_draft"
			if err := tx.Model(&link).Updates(map[string]any{"workflow_status": link.WorkflowStatus, "workflow_updated_at": time.Now().UTC()}).Error; err != nil {
				return err
			}
		}
		result = LinkPrivateTaskResult{Product: product, Link: link}
		return nil
	})
	return &result, err
}

func (s *Service) ListPrivateTaskLinks(productID, ownerID int64) ([]Sourcing1688TaskLinkView, error) {
	if productID <= 0 || ownerID <= 0 {
		return nil, ErrInvalidWorkflow
	}
	var owned int64
	if err := s.db.Model(&Sourcing1688Product{}).Where("id = ? AND owner_id = ?", productID, ownerID).Count(&owned).Error; err != nil {
		return nil, err
	}
	if owned != 1 {
		return nil, fmt.Errorf("%w: private source does not belong to authenticated Owner", ErrWorkflowGate)
	}
	var links []Sourcing1688TaskLink
	if err := s.db.Where("sourcing_product_id = ? AND owner_id = ?", productID, ownerID).Order("is_primary DESC, created_at ASC").Find(&links).Error; err != nil {
		return nil, err
	}
	views := make([]Sourcing1688TaskLinkView, 0, len(links))
	for _, link := range links {
		view := Sourcing1688TaskLinkView{Sourcing1688TaskLink: link, CurrentStatus: link.WorkflowStatus}
		var dc demandCaseRow
		var exp experimentRow
		var gateCount, objectCount int64
		dcErr := s.db.First(&dc, link.DemandCaseID).Error
		expErr := s.db.Where("experiment_id = ?", link.ExperimentID).First(&exp).Error
		_ = s.db.Model(&gateRow{}).Where("experiment_id = ? AND stage = ? AND result = ?", link.ExperimentID, "opportunity", "pass").Count(&gateCount).Error
		_ = s.db.Model(&objectLinkRow{}).Where("experiment_id = ? AND object_type = ? AND object_id = ?", link.ExperimentID, "demand_case", strconv.FormatInt(link.DemandCaseID, 10)).Count(&objectCount).Error
		if link.Status == "archived" || link.WorkflowStatus == "archived" {
			view.CurrentStatus, view.CurrentBlocker = "archived", "该任务关联已归档"
		} else if link.AuthorityKind != "product_opportunity" || link.ProductOpportunityID == nil || link.OpportunityDecisionID == nil {
			view.CurrentStatus, view.CurrentBlocker = "blocked", "历史实验关联仅供追踪，缺少已冻结的商品机会批准"
		} else if dcErr != nil || expErr != nil || dc.OwnerID != ownerID || exp.OwnerID != ownerID {
			view.CurrentStatus, view.CurrentBlocker = "blocked", "关联任务已不存在或不再属于当前Owner"
		} else if dc.Status != "experiment_ready" || exp.Status != "active" || gateCount == 0 || objectCount == 0 {
			view.CurrentStatus, view.CurrentBlocker = "blocked", "选品任务当前未通过机会闸门或已停止"
		} else if decision, err := requireSourcingAuthority(s.db, ownerID, link.DemandCaseID, *link.ProductOpportunityID); err != nil || decision.ID != *link.OpportunityDecisionID {
			view.CurrentStatus, view.CurrentBlocker = "blocked", "商品机会批准已失效或与冻结决定不一致"
		}
		parts := []string{strings.TrimSpace(dc.Region), strings.TrimSpace(dc.SalesChannel), strings.TrimSpace(dc.NeedScenario)}
		for _, part := range parts {
			if part != "" {
				if view.Label != "" {
					view.Label += " · "
				}
				view.Label += part
			}
		}
		if view.Label == "" {
			view.Label = link.ExperimentID
		}
		views = append(views, view)
	}
	return views, nil
}

func pointerTime(value time.Time) *time.Time { return &value }

func (s *Service) ListEligibleTasks(ownerID int64) ([]EligibleSourcingTask, error) {
	if ownerID <= 0 {
		return nil, ErrWorkflowGate
	}
	var tasks []EligibleSourcingTask
	err := s.db.Raw(`
		SELECT e.experiment_id, dc.id AS demand_case_id, dc.region, dc.consumer,
		       dc.need_scenario, dc.sales_channel, e.stage,
		       po.id AS product_opportunity_id, po.title AS opportunity_title
		FROM experiment_case e
		JOIN experiment_object_link ol
		  ON ol.experiment_id = e.experiment_id AND ol.object_type = 'demand_case'
		JOIN demand_case dc ON CAST(dc.id AS TEXT) = ol.object_id
		JOIN experiment_gate_decision gd
		  ON gd.experiment_id = e.experiment_id AND gd.stage = 'opportunity' AND gd.result = 'pass'
		JOIN product_opportunity po
		  ON po.demand_case_id = dc.id AND po.owner_id = dc.owner_id AND po.status = 'approved'
		JOIN product_opportunity_decision pod
		  ON pod.opportunity_id = po.id AND pod.owner_id = po.owner_id
		 AND pod.version = po.version AND pod.content_hash = po.content_hash AND pod.decision = 'approved'
		JOIN market_owner_decision md
		  ON md.id = po.market_decision_id AND md.demand_case_id = dc.id
		 AND md.owner_id = dc.owner_id AND md.decision = 'selected'
		WHERE e.owner_id = ? AND dc.owner_id = ? AND e.status = 'active'
		  AND dc.status = 'experiment_ready' AND e.stage IN ('product', 'supply', 'channel')
		  AND NOT EXISTS (
		    SELECT 1 FROM market_owner_decision newer
		    WHERE newer.demand_case_id = dc.id AND newer.owner_id = dc.owner_id AND newer.id > md.id
		  )
		ORDER BY dc.id, e.experiment_id`, ownerID, ownerID).Scan(&tasks).Error
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		parts := []string{strings.TrimSpace(tasks[i].OpportunityTitle), strings.TrimSpace(tasks[i].Region), strings.TrimSpace(tasks[i].SalesChannel), strings.TrimSpace(tasks[i].NeedScenario)}
		var visible []string
		for _, part := range parts {
			if part != "" {
				visible = append(visible, part)
			}
		}
		tasks[i].Label = strings.Join(visible, " · ")
		if tasks[i].Label == "" {
			tasks[i].Label = tasks[i].ExperimentID
		}
	}
	return tasks, nil
}
