package sourcing1688

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UpdateDraft replaces the editable internal draft atomically. It never calls
// a platform adapter and refuses pending/approved drafts.
func (s *Service) UpdateDraft(id int64, in *ConvertInput) (*DraftDetail, error) {
	if err := validateConvert(in); err != nil {
		return nil, err
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var source Sourcing1688Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&source, id).Error; err != nil {
			return err
		}
		if source.Status != StatusDraftCreated || source.LifecycleStatus != LifecycleEditing || source.ProductID == nil || source.SnapshotID == nil || source.DemandCaseID == nil || source.ExperimentID == nil {
			return fmt.Errorf("%w: only an editing internal draft may be updated", ErrWorkflowGate)
		}
		var dc demandCaseRow
		if err := tx.First(&dc, *source.DemandCaseID).Error; err != nil {
			return err
		}
		if dc.OwnerID != in.CreatedBy || dc.Status != "experiment_ready" {
			return fmt.Errorf("%w: update requires workflow Owner", ErrWorkflowGate)
		}
		if strings.TrimSpace(dc.TargetLocale) == "" || !strings.EqualFold(dc.TargetLocale, in.TargetLocale) {
			return fmt.Errorf("%w: draft locale does not match the approved market locale", ErrWorkflowGate)
		}
		var platform platformRow
		if err := tx.First(&platform, in.PlatformID).Error; err != nil {
			return err
		}
		channel := strings.ToLower(dc.SalesChannel)
		if !strings.Contains(channel, strings.ToLower(platform.Code)) && !strings.Contains(channel, strings.ToLower(platform.Name)) {
			return fmt.Errorf("%w: platform does not match approved sales channel", ErrWorkflowGate)
		}
		var draft draftRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("sourcing_product_id = ?", id).First(&draft).Error; err != nil {
			return err
		}
		var listing listingRow
		if err := tx.First(&listing, draft.ListingID).Error; err != nil {
			return err
		}
		if listing.Status != "draft" || draft.ApprovalStatus == "pending" || draft.ApprovalStatus == "approved" {
			return fmt.Errorf("%w: pending or approved draft cannot be edited", ErrWorkflowGate)
		}

		images := make([]string, 0, len(in.Media))
		for _, media := range in.Media {
			images = append(images, media.ProcessedURL)
		}
		imageJSON, _ := json.Marshal(images)
		if err := tx.Model(&productRow{}).Where("id = ?", draft.ProductID).Updates(map[string]any{"name": strings.TrimSpace(in.Title), "description": in.Description, "category_id": in.CategoryID, "unit": in.Unit, "images": imageJSON, "main_image": in.Media[0].ProcessedURL, "updated_at": time.Now().UTC()}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", draft.ProductID).Delete(&skuRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", draft.ProductID).Delete(&mediaRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", draft.ProductID).Delete(&costRow{}).Error; err != nil {
			return err
		}
		for _, sku := range in.SKUVariants {
			row := skuRow{ProductID: draft.ProductID, Code: sku.InternalSKU, SpecDesc: sku.SpecDesc, SpecValues: sku.SpecValues, Price: sku.Price, CostPrice: sku.CostPrice, Weight: sku.Weight, Image: sku.Image, Status: 1}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		for _, media := range in.Media {
			var processed ImageProcessingRecord
			if err := tx.First(&processed, media.ProcessingRecordID).Error; err != nil {
				return err
			}
			expectedURL := fmt.Sprintf("/api/v1/sourcing-1688/processed-images/%d/content", processed.ID)
			if processed.SourcingProductID != source.ID || processed.SnapshotID != *source.SnapshotID || processed.SourceURL != media.SourceURL || processed.ProcessedSHA256 != media.ContentSHA256 || processed.RightsEvidenceURI != media.RightsEvidenceURI || !processed.RightsObservedAt.Equal(media.RightsObservedAt) || processed.ChannelRuleURI != media.ChannelRuleURI || expectedURL != media.ProcessedURL {
				return fmt.Errorf("%w: media processing evidence mismatch", ErrWorkflowGate)
			}
			row := mediaRow{ProductID: draft.ProductID, SourceSnapshotID: *source.SnapshotID, SourceURL: media.SourceURL, ProcessedURL: media.ProcessedURL, MediaRole: media.MediaRole, RightsStatus: media.RightsStatus, RightsEvidenceURI: media.RightsEvidenceURI, Operations: media.Operations, ContentSHA256: media.ContentSHA256, Width: media.Width, Height: media.Height, HasWatermark: media.HasWatermark, HasChineseText: media.HasChineseText, HasBrandMark: media.HasBrandMark, ChannelRuleURI: media.ChannelRuleURI}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		for _, cost := range in.Costs {
			row := costRow{ProductID: draft.ProductID, ExperimentID: *source.ExperimentID, CostType: cost.CostType, Amount: cost.Amount, Currency: cost.Currency, TruthStatus: cost.TruthStatus, SourceURI: cost.SourceURI, ObservedAt: cost.ObservedAt}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		published, err := buildListingDraftPayload(in, *source.SnapshotID)
		if err != nil {
			return err
		}
		if err := tx.Model(&listing).Updates(map[string]any{"platform_id": in.PlatformID, "platform_sku": in.PlatformSKU, "published_data": published}).Error; err != nil {
			return err
		}
		return tx.Model(&draft).Updates(map[string]any{"approval_id": nil, "approval_status": "", "approval_content_sha256": "", "approval_rejection_reason": ""}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetDraft(id)
}
