package sourcing1688

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInvalidWorkflow = errors.New("invalid sourcing workflow input")
	ErrWorkflowGate    = errors.New("sourcing workflow gate not satisfied")
)

var offerIDPattern = regexp.MustCompile(`/offer/(\d+)\.html`)

func canonical1688URL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || (u.Port() != "" && u.Port() != "443") {
		return "", fmt.Errorf("%w: source_url must be an https 1688 URL", ErrInvalidWorkflow)
	}
	host := strings.ToLower(u.Hostname())
	if host != "1688.com" && !strings.HasSuffix(host, ".1688.com") {
		return "", fmt.Errorf("%w: source_url host is not 1688.com", ErrInvalidWorkflow)
	}
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	if match := offerIDPattern.FindStringSubmatch(u.Path); len(match) == 2 {
		u.Scheme, u.Host, u.Path, u.RawQuery = "https", "detail.1688.com", "/offer/"+match[1]+".html", ""
	}
	return u.String(), nil
}

func offerIDFromURL(canonical string) string {
	match := offerIDPattern.FindStringSubmatch(canonical)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

// Capture stores the exact received payload and its digest. Identical retries
// return the existing record; changed observations create a new immutable snapshot.
func (s *Service) Capture(in *CaptureInput) (*Sourcing1688Product, error) {
	canonicalURL, err := canonical1688URL(in.SourceURL)
	if err != nil {
		return nil, err
	}
	if in.DemandCaseID == 0 || in.ExperimentID == "" || in.CollectedBy == 0 || in.CollectedAt.IsZero() || strings.TrimSpace(in.Driver) == "" || strings.TrimSpace(in.ParserVersion) == "" || strings.TrimSpace(in.SupplierBusinessID) == "" || len(in.RawPayload) == 0 || !json.Valid(in.RawPayload) {
		return nil, fmt.Errorf("%w: demand case, experiment, collector, timestamp, driver and valid raw payload are required", ErrInvalidWorkflow)
	}
	digest := sha256.Sum256(in.RawPayload)
	hash := hex.EncodeToString(digest[:])
	offerID := offerIDFromURL(canonicalURL)
	if offerID == "" {
		return nil, fmt.Errorf("%w: a 1688 product offer URL is required", ErrInvalidWorkflow)
	}
	var result Sourcing1688Product
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var previousSnapshot *Sourcing1688Snapshot
		var dc demandCaseRow
		if err := tx.First(&dc, in.DemandCaseID).Error; err != nil {
			return fmt.Errorf("demand case: %w", err)
		}
		if dc.Status != "experiment_ready" {
			return fmt.Errorf("%w: demand case is not experiment_ready", ErrWorkflowGate)
		}
		var exp experimentRow
		if err := tx.Where("experiment_id = ?", in.ExperimentID).First(&exp).Error; err != nil {
			return fmt.Errorf("experiment: %w", err)
		}
		if exp.Status != "active" || exp.OwnerID != dc.OwnerID || in.CollectedBy != dc.OwnerID {
			return fmt.Errorf("%w: active experiment and Owner identities must match", ErrWorkflowGate)
		}
		if exp.Stage != "product" && exp.Stage != "supply" && exp.Stage != "channel" {
			return fmt.Errorf("%w: experiment must have passed the product opportunity gate", ErrWorkflowGate)
		}
		var opportunityPass int64
		if err := tx.Model(&gateRow{}).Where("experiment_id = ? AND stage = ? AND result = ?", in.ExperimentID, "opportunity", "pass").Count(&opportunityPass).Error; err != nil || opportunityPass == 0 {
			return fmt.Errorf("%w: Owner-approved opportunity pass is required", ErrWorkflowGate)
		}
		var linkCount int64
		if err := tx.Model(&objectLinkRow{}).Where("experiment_id = ? AND object_type = ? AND object_id = ?", in.ExperimentID, "demand_case", strconv.FormatInt(in.DemandCaseID, 10)).Count(&linkCount).Error; err != nil {
			return err
		}
		if linkCount == 0 {
			return fmt.Errorf("%w: experiment is not linked to demand case", ErrWorkflowGate)
		}

		var p Sourcing1688Product
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("source_offer_id = ? OR source_url = ?", offerID, canonicalURL).First(&p).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			p = Sourcing1688Product{SourceURL: canonicalURL, SourceOfferID: offerID, Title: in.Title, Price: in.Price, SupplierName: in.SupplierName, SupplierBusinessID: strings.TrimSpace(in.SupplierBusinessID), Status: StatusPendingReview, LifecycleStatus: LifecyclePendingReview, DemandCaseID: &in.DemandCaseID, ExperimentID: &in.ExperimentID}
			if in.MOQ != nil {
				p.MOQ = *in.MOQ
			} else {
				p.MOQ = 1
			}
			if len(in.Images) > 0 {
				v := json.RawMessage(append([]byte(nil), in.Images...))
				p.Images = &v
			}
			if len(in.SkuVariants) > 0 {
				v := json.RawMessage(append([]byte(nil), in.SkuVariants...))
				p.SkuVariants = &v
			}
			if err := tx.Create(&p).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			if p.SnapshotID != nil {
				var previous Sourcing1688Snapshot
				if err := tx.First(&previous, *p.SnapshotID).Error; err != nil {
					return err
				}
				previousSnapshot = &previous
			}
			p.SourceURL, p.SourceOfferID = canonicalURL, offerID
			p.SupplierBusinessID = strings.TrimSpace(in.SupplierBusinessID)
			if p.DemandCaseID == nil || *p.DemandCaseID != in.DemandCaseID || p.ExperimentID == nil || *p.ExperimentID != in.ExperimentID {
				return fmt.Errorf("%w: source URL already belongs to another approved workflow", ErrWorkflowGate)
			}
			var existing Sourcing1688Snapshot
			if err := tx.Where("sourcing_product_id = ? AND raw_sha256 = ?", p.ID, hash).First(&existing).Error; err == nil {
				result = p
				return nil
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if p.Status == StatusDraftCreated {
				return fmt.Errorf("%w: converted source cannot be recaptured", ErrWorkflowGate)
			}
			p.Status, p.ReviewedBy, p.ReviewedAt, p.SnapshotID = StatusPendingReview, nil, nil, nil
			p.Title, p.Price, p.SupplierName = in.Title, in.Price, in.SupplierName
			if in.MOQ != nil {
				p.MOQ = *in.MOQ
			}
			if len(in.Images) > 0 {
				v := json.RawMessage(append([]byte(nil), in.Images...))
				p.Images = &v
			}
			if len(in.SkuVariants) > 0 {
				v := json.RawMessage(append([]byte(nil), in.SkuVariants...))
				p.SkuVariants = &v
			}
			if err := tx.Save(&p).Error; err != nil {
				return err
			}
		}
		snap := Sourcing1688Snapshot{SourcingProductID: p.ID, SourceURL: canonicalURL, CollectedAt: in.CollectedAt, CollectedBy: in.CollectedBy, Driver: strings.TrimSpace(in.Driver), ParserVersion: strings.TrimSpace(in.ParserVersion), RawPayload: append(json.RawMessage(nil), in.RawPayload...), RawSHA256: hash, ObservedTitle: in.Title, ObservedPrice: in.Price, ObservedMOQ: p.MOQ, ObservedSupplier: in.SupplierName, ObservedSupplierBusinessID: strings.TrimSpace(in.SupplierBusinessID)}
		if err := tx.Create(&snap).Error; err != nil {
			return err
		}
		if err := s.recordIdentityAndChanges(tx, &p, previousSnapshot, &snap); err != nil {
			return err
		}
		if err := tx.Model(&p).Updates(map[string]any{"snapshot_id": snap.ID, "status": StatusPendingReview, "lifecycle_status": LifecyclePendingReview, "lifecycle_actor_id": nil, "lifecycle_reason": "", "lifecycle_updated_at": time.Now().UTC()}).Error; err != nil {
			return err
		}
		p.SnapshotID, p.Status = &snap.ID, StatusPendingReview
		result = p
		return nil
	})
	return &result, err
}

func (s *Service) Review(id int64, in *ReviewInput) (*Sourcing1688Product, error) {
	if in.ReviewedBy == 0 || strings.TrimSpace(in.Notes) == "" {
		return nil, fmt.Errorf("%w: Owner and review notes required", ErrInvalidWorkflow)
	}
	var p Sourcing1688Product
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, id).Error; err != nil {
			return err
		}
		if p.Status != StatusPendingReview || p.SnapshotID == nil {
			return fmt.Errorf("%w: only a snapshotted pending item can be reviewed", ErrWorkflowGate)
		}
		var dc demandCaseRow
		if p.DemandCaseID == nil || tx.First(&dc, *p.DemandCaseID).Error != nil || dc.OwnerID != in.ReviewedBy || dc.Status != "experiment_ready" {
			return fmt.Errorf("%w: review must be performed by the approved market Owner", ErrWorkflowGate)
		}
		now := time.Now().UTC()
		if err := tx.Model(&p).Updates(map[string]any{"status": StatusReviewed, "reviewed_by": in.ReviewedBy, "reviewed_at": now, "review_notes": strings.TrimSpace(in.Notes), "lifecycle_status": LifecycleReadyForProduct, "lifecycle_actor_id": in.ReviewedBy, "lifecycle_reason": strings.TrimSpace(in.Notes), "lifecycle_updated_at": now}).Error; err != nil {
			return err
		}
		p.Status, p.ReviewedBy, p.ReviewedAt, p.ReviewNotes = StatusReviewed, &in.ReviewedBy, &now, strings.TrimSpace(in.Notes)
		return nil
	})
	return &p, err
}

func (s *Service) GetSnapshot(id int64) (*Sourcing1688Snapshot, error) {
	var p Sourcing1688Product
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	if p.SnapshotID == nil {
		return nil, gorm.ErrRecordNotFound
	}
	var snapshot Sourcing1688Snapshot
	if err := s.db.Where("id = ? AND sourcing_product_id = ?", *p.SnapshotID, p.ID).First(&snapshot).Error; err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (s *Service) GetDraft(id int64) (*DraftDetail, error) {
	var detail DraftDetail
	if err := s.db.Where("sourcing_product_id = ?", id).First(&detail.Draft).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&detail.Listing, detail.Draft.ListingID).Error; err != nil {
		return nil, err
	}
	if detail.Listing.Status != "draft" {
		return nil, fmt.Errorf("%w: linked listing is no longer a draft", ErrWorkflowGate)
	}
	if err := s.db.First(&detail.Product, detail.Draft.ProductID).Error; err != nil {
		return nil, err
	}
	if err := s.db.Where("product_id = ?", detail.Draft.ProductID).Order("id").Find(&detail.SKUs).Error; err != nil {
		return nil, err
	}
	if err := s.db.Where("product_id = ?", detail.Draft.ProductID).Order("id").Find(&detail.Media).Error; err != nil {
		return nil, err
	}
	if err := s.db.Where("product_id = ?", detail.Draft.ProductID).Order("cost_type").Find(&detail.Costs).Error; err != nil {
		return nil, err
	}
	return &detail, nil
}

func validateConvert(in *ConvertInput) error {
	if in.CreatedBy == 0 || in.PlatformID == 0 || in.CategoryID == 0 || strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.Description) == "" || strings.TrimSpace(in.Unit) == "" || strings.TrimSpace(in.LocalizedTitle) == "" || strings.TrimSpace(in.LocalizedDescription) == "" || strings.TrimSpace(in.TargetLocale) == "" || strings.TrimSpace(in.ShippingTemplateID) == "" || strings.TrimSpace(in.CategorySchemaURI) == "" || in.CategoryObservedAt.IsZero() || len(in.SKUVariants) == 0 || len(in.Media) == 0 || len(in.Costs) == 0 || !json.Valid(in.ListingPayload) || string(in.ListingPayload) == "{}" {
		return fmt.Errorf("%w: complete product, SKU, media, costs and listing payload required", ErrInvalidWorkflow)
	}
	if err := validateChecks(in.SupplierAssessment, []string{"identity", "operating_history", "transaction_history", "moq", "mixed_batch", "lead_time", "sample", "returns"}, false); err != nil {
		return fmt.Errorf("%w: supplier assessment: %v", ErrWorkflowGate, err)
	}
	if err := validateChecks(in.ComplianceChecks, []string{"brand_ip", "patent", "certification", "dangerous_goods", "material", "labeling_instructions"}, true); err != nil {
		return fmt.Errorf("%w: compliance: %v", ErrWorkflowGate, err)
	}
	seenChannelSKU := map[string]bool{}
	for _, sku := range in.SKUVariants {
		if strings.TrimSpace(sku.SupplierSKU) == "" || strings.TrimSpace(sku.ChannelSKU) == "" || seenChannelSKU[sku.ChannelSKU] || !json.Valid(sku.SpecValues) {
			return fmt.Errorf("%w: invalid or duplicate SKU mapping", ErrInvalidWorkflow)
		}
		seenChannelSKU[sku.ChannelSKU] = true
	}
	hasMain := false
	required := map[string]bool{"purchase": false, "domestic_shipping": false, "packaging": false, "cross_border_shipping": false, "platform_fee": false, "payment_fee": false, "advertising": false, "tax": false, "duty": false, "return_loss": false}
	for _, c := range in.Costs {
		if seen, ok := required[c.CostType]; !ok || seen || c.Amount < 0 || c.ObservedAt.IsZero() || c.SourceURI == "" || (c.TruthStatus != "actual" && c.TruthStatus != "quoted" && c.TruthStatus != "estimated") {
			return fmt.Errorf("%w: invalid cost evidence", ErrInvalidWorkflow)
		}
		required[c.CostType] = true
	}
	for k, ok := range required {
		if !ok {
			return fmt.Errorf("%w: missing cost %s", ErrInvalidWorkflow, k)
		}
	}
	for _, m := range in.Media {
		var operations []any
		if m.RightsStatus != "verified" || m.RightsEvidenceURI == "" || m.SourceURL == "" || m.ProcessedURL == "" || m.ChannelRuleURI == "" || m.Width <= 0 || m.Height <= 0 || m.HasWatermark || m.HasChineseText || m.HasBrandMark || !json.Valid(m.Operations) || json.Unmarshal(m.Operations, &operations) != nil || len(operations) == 0 {
			return fmt.Errorf("%w: every image needs verified rights and processing history", ErrWorkflowGate)
		}
		hasMain = hasMain || m.MediaRole == "main"
	}
	if !hasMain {
		return fmt.Errorf("%w: one processed main image is required", ErrWorkflowGate)
	}
	if err := validateDraftConsistency(in); err != nil {
		return err
	}
	validation := ValidateDraft(in.Validation)
	if !validation.Passed {
		blockers, _ := json.Marshal(validation.Blockers)
		return fmt.Errorf("%w: deterministic draft validation failed: %s", ErrWorkflowGate, blockers)
	}
	return nil
}

func validateDraftConsistency(in *ConvertInput) error {
	if in.Validation.Localization.Locale != in.TargetLocale || in.Validation.Localization.Title != in.LocalizedTitle || in.Validation.Localization.Description != in.LocalizedDescription || in.Validation.Localization.Unit != in.Unit {
		return fmt.Errorf("%w: localization validation input does not match the draft", ErrInvalidWorkflow)
	}
	if in.Validation.Channel.PlatformID != in.PlatformID || in.Validation.ChannelRules.PlatformID != in.PlatformID || in.Validation.Channel.CategoryID != strconv.FormatInt(in.CategoryID, 10) || in.Validation.Channel.CategorySchemaURI != in.CategorySchemaURI || !in.Validation.Channel.CategoryObservedAt.Equal(in.CategoryObservedAt) || in.Validation.Channel.ShippingTemplateID != in.ShippingTemplateID || in.Validation.Channel.ImageCount != len(in.Media) {
		return fmt.Errorf("%w: channel validation input does not match category, shipping or media", ErrInvalidWorkflow)
	}
	if len(in.Validation.SKUs) != len(in.SKUVariants) || len(in.Validation.Images) != len(in.Media) || len(in.Validation.Costs.Costs) != len(in.Costs) {
		return fmt.Errorf("%w: validation item counts do not match the draft", ErrInvalidWorkflow)
	}
	for i, sku := range in.SKUVariants {
		validated := in.Validation.SKUs[i]
		if validated.SupplierSKU != sku.SupplierSKU || validated.InternalSKU != sku.InternalSKU || validated.ChannelSKU != sku.ChannelSKU || validated.Color != sku.Color || validated.Size != sku.Size || validated.Material != sku.Material || validated.Packaging != sku.Packaging {
			return fmt.Errorf("%w: SKU validation mapping does not match draft SKU", ErrInvalidWorkflow)
		}
		var specValues map[string]any
		if json.Unmarshal(sku.SpecValues, &specValues) != nil || fmt.Sprint(specValues["color"]) != sku.Color || fmt.Sprint(specValues["size"]) != sku.Size || fmt.Sprint(specValues["material"]) != sku.Material || fmt.Sprint(specValues["packaging"]) != sku.Packaging {
			return fmt.Errorf("%w: SKU attributes do not match stored spec_values", ErrInvalidWorkflow)
		}
	}
	if !in.Validation.SKURules.RequireColor || !in.Validation.SKURules.RequireSize || !in.Validation.SKURules.RequireMaterial || !in.Validation.SKURules.RequirePackaging {
		return fmt.Errorf("%w: color, size, material and packaging SKU rules are mandatory", ErrInvalidWorkflow)
	}
	for i, media := range in.Media {
		validated := in.Validation.Images[i]
		if validated.Role != media.MediaRole || validated.Width != media.Width || validated.Height != media.Height || validated.TruthStatus != "actual" || validated.SourceURI != "sha256:"+media.ContentSHA256 || validated.HasWatermark != media.HasWatermark || validated.HasChineseText != media.HasChineseText || validated.HasBrandMark != media.HasBrandMark {
			return fmt.Errorf("%w: image validation does not match media record", ErrInvalidWorkflow)
		}
	}
	costs := make(map[string]CostInput, len(in.Costs))
	for _, cost := range in.Costs {
		costs[cost.CostType] = cost
	}
	for _, validated := range in.Validation.Costs.Costs {
		cost, ok := costs[validated.Type]
		if !ok || cost.Amount != validated.Amount || !strings.EqualFold(cost.Currency, validated.Currency) || cost.TruthStatus != validated.TruthStatus || cost.SourceURI != validated.SourceURI || !cost.ObservedAt.Equal(validated.ObservedAt) {
			return fmt.Errorf("%w: cost validation does not match stored cost %s", ErrInvalidWorkflow, validated.Type)
		}
	}
	return nil
}

func validateChecks(checks []EvidenceCheck, required []string, actualOnly bool) error {
	seen := make(map[string]bool, len(required))
	allowed := make(map[string]bool, len(required))
	for _, key := range required {
		allowed[key] = true
	}
	for _, check := range checks {
		validTruth := check.TruthStatus == "actual" || (!actualOnly && check.TruthStatus == "quoted")
		if !allowed[check.CheckType] || seen[check.CheckType] || check.Result != "pass" || !validTruth || strings.TrimSpace(check.SourceURI) == "" || check.ObservedAt.IsZero() {
			return fmt.Errorf("invalid check %q", check.CheckType)
		}
		seen[check.CheckType] = true
	}
	for _, key := range required {
		if !seen[key] {
			return fmt.Errorf("missing check %q", key)
		}
	}
	return nil
}

// Convert creates only internal product/SKU/media/cost records and a listing
// whose status is unconditionally draft. It never calls a platform adapter.
func (s *Service) Convert(id int64, in *ConvertInput) (*ConvertResult, error) {
	if err := validateConvert(in); err != nil {
		return nil, err
	}
	result := &ConvertResult{SourcingProductID: id, Status: "draft"}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var p Sourcing1688Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, id).Error; err != nil {
			return err
		}
		if p.Status == StatusDraftCreated && p.ProductID != nil {
			var d draftRow
			if err := tx.Where("sourcing_product_id = ?", id).First(&d).Error; err != nil {
				return err
			}
			result.SnapshotID, result.ProductID, result.ListingID, result.DraftID = d.SnapshotID, d.ProductID, d.ListingID, d.ID
			var skus []skuRow
			if err := tx.Where("product_id = ?", d.ProductID).Find(&skus).Error; err != nil {
				return err
			}
			for _, v := range skus {
				result.SKUIDs = append(result.SKUIDs, v.ID)
			}
			return nil
		}
		if p.Status != StatusReviewed || p.LifecycleStatus != LifecycleReadyForProduct || p.SnapshotID == nil || p.DemandCaseID == nil || p.ExperimentID == nil || p.ReviewedBy == nil || *p.ReviewedBy != in.CreatedBy {
			return fmt.Errorf("%w: Owner-reviewed source required", ErrWorkflowGate)
		}
		var unresolvedDuplicates int64
		if err := tx.Model(&DuplicateCandidate{}).Where("(source_product_id = ? OR matched_product_id = ?) AND status = ?", p.ID, p.ID, "pending_review").Count(&unresolvedDuplicates).Error; err != nil {
			return err
		}
		if unresolvedDuplicates > 0 {
			return fmt.Errorf("%w: suspected duplicate must be resolved by Owner before product creation", ErrWorkflowGate)
		}
		var confirmedDuplicate int64
		if err := tx.Model(&DuplicateCandidate{}).Where("source_product_id = ? AND status = ?", p.ID, "same_product").Count(&confirmedDuplicate).Error; err != nil {
			return err
		}
		if confirmedDuplicate > 0 {
			return fmt.Errorf("%w: confirmed duplicate must reuse the matched product", ErrWorkflowGate)
		}
		var dc demandCaseRow
		if err := tx.First(&dc, *p.DemandCaseID).Error; err != nil {
			return err
		}
		var exp experimentRow
		if err := tx.Where("experiment_id = ?", *p.ExperimentID).First(&exp).Error; err != nil {
			return err
		}
		if dc.Status != "experiment_ready" || exp.Status != "active" || dc.OwnerID != in.CreatedBy || exp.OwnerID != in.CreatedBy {
			return fmt.Errorf("%w: approved market or active experiment changed", ErrWorkflowGate)
		}
		if exp.Stage != "product" && exp.Stage != "supply" && exp.Stage != "channel" {
			return fmt.Errorf("%w: experiment stage no longer permits draft preparation", ErrWorkflowGate)
		}
		var platform platformRow
		if err := tx.First(&platform, in.PlatformID).Error; err != nil {
			return err
		}
		channel := strings.ToLower(dc.SalesChannel)
		if !strings.Contains(channel, strings.ToLower(platform.Code)) && !strings.Contains(channel, strings.ToLower(platform.Name)) {
			return fmt.Errorf("%w: platform does not match approved sales channel", ErrWorkflowGate)
		}
		images, _ := json.Marshal(func() []string {
			v := []string{}
			for _, m := range in.Media {
				v = append(v, m.ProcessedURL)
			}
			return v
		}())
		prod := productRow{Name: strings.TrimSpace(in.Title), Description: in.Description, CategoryID: in.CategoryID, Unit: in.Unit, Status: 0, Images: images, MainImage: in.Media[0].ProcessedURL}
		if err := tx.Create(&prod).Error; err != nil {
			return err
		}
		result.ProductID = prod.ID
		for _, v := range in.SKUVariants {
			row := skuRow{ProductID: prod.ID, Code: v.InternalSKU, SpecDesc: v.SpecDesc, SpecValues: v.SpecValues, Price: v.Price, CostPrice: v.CostPrice, Weight: v.Weight, Image: v.Image, Status: 1}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			result.SKUIDs = append(result.SKUIDs, row.ID)
		}
		for _, v := range in.Media {
			var processed ImageProcessingRecord
			if err := tx.First(&processed, v.ProcessingRecordID).Error; err != nil {
				return err
			}
			expectedURL := fmt.Sprintf("/api/v1/sourcing-1688/processed-images/%d/content", processed.ID)
			if processed.SourcingProductID != p.ID || processed.SnapshotID != *p.SnapshotID || processed.SourceURL != v.SourceURL || processed.ProcessedSHA256 != v.ContentSHA256 || processed.OutputWidth != v.Width || processed.OutputHeight != v.Height || processed.RightsEvidenceURI != v.RightsEvidenceURI || !processed.RightsObservedAt.Equal(v.RightsObservedAt) || processed.ChannelRuleURI != v.ChannelRuleURI || expectedURL != v.ProcessedURL || string(processed.Operations) != string(v.Operations) {
				return fmt.Errorf("%w: media record does not match actual image processing output", ErrWorkflowGate)
			}
			row := mediaRow{ProductID: prod.ID, SourceSnapshotID: *p.SnapshotID, SourceURL: v.SourceURL, ProcessedURL: v.ProcessedURL, MediaRole: v.MediaRole, RightsStatus: v.RightsStatus, RightsEvidenceURI: v.RightsEvidenceURI, Operations: v.Operations, ContentSHA256: v.ContentSHA256, Width: v.Width, Height: v.Height, HasWatermark: v.HasWatermark, HasChineseText: v.HasChineseText, HasBrandMark: v.HasBrandMark, ChannelRuleURI: v.ChannelRuleURI}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		for _, v := range in.Costs {
			row := costRow{ProductID: prod.ID, ExperimentID: *p.ExperimentID, CostType: v.CostType, Amount: v.Amount, Currency: v.Currency, TruthStatus: v.TruthStatus, SourceURI: v.SourceURI, ObservedAt: v.ObservedAt}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		validationResult := ValidateDraft(in.Validation)
		published, _ := json.Marshal(map[string]any{"localized_title": in.LocalizedTitle, "localized_description": in.LocalizedDescription, "target_locale": in.TargetLocale, "shipping_template_id": in.ShippingTemplateID, "category_schema_uri": in.CategorySchemaURI, "category_observed_at": in.CategoryObservedAt, "supplier_assessment": in.SupplierAssessment, "compliance_checks": in.ComplianceChecks, "media_requirements": in.Media, "validation_result": validationResult, "source_snapshot_id": *p.SnapshotID, "supplier_sku_mapping": in.SKUVariants, "channel_fields": json.RawMessage(in.ListingPayload)})
		listing := listingRow{ProductID: prod.ID, PlatformID: in.PlatformID, PlatformSKU: in.PlatformSKU, Status: "draft", PublishedData: published}
		if err := tx.Create(&listing).Error; err != nil {
			return err
		}
		result.ListingID = listing.ID
		draft := draftRow{SourcingProductID: p.ID, SnapshotID: *p.SnapshotID, ProductID: prod.ID, ListingID: listing.ID, DemandCaseID: *p.DemandCaseID, ExperimentID: *p.ExperimentID, CreatedBy: in.CreatedBy}
		if err := tx.Create(&draft).Error; err != nil {
			return err
		}
		result.DraftID, result.SnapshotID = draft.ID, *p.SnapshotID
		now := time.Now().UTC()
		if err := tx.Model(&p).Updates(map[string]any{"status": StatusDraftCreated, "product_id": prod.ID, "imported_by": strconv.FormatInt(in.CreatedBy, 10), "imported_at": now, "lifecycle_status": LifecycleEditing, "lifecycle_actor_id": in.CreatedBy, "lifecycle_reason": "", "lifecycle_updated_at": now}).Error; err != nil {
			return err
		}
		for typ, obj := range map[string]string{"sourcing_1688": strconv.FormatInt(p.ID, 10), "product": strconv.FormatInt(prod.ID, 10), "listing_draft": strconv.FormatInt(listing.ID, 10)} {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&objectLinkRow{ExperimentID: *p.ExperimentID, ObjectType: typ, ObjectID: obj}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return result, err
}
