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
	var expectedVersion int64
	if in.TaskLinkID > 0 {
		current, err := s.GetOwnedTaskDraft(id, in.TaskLinkID, in.CreatedBy)
		if err != nil {
			return nil, err
		}
		if in.EditableVersion <= 0 || in.EditableSHA256 == "" || in.EditableVersion != current.EditableVersion || !strings.EqualFold(in.EditableSHA256, current.EditableSHA256) {
			return nil, fmt.Errorf("%w: draft changed; reload the exact task draft before editing", ErrWorkflowGate)
		}
		expectedVersion = current.EditableVersion
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var source Sourcing1688Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&source, id).Error; err != nil {
			return err
		}
		if source.SnapshotID == nil || source.OwnerID != in.CreatedBy {
			return fmt.Errorf("%w: update requires sourcing Owner", ErrWorkflowGate)
		}
		task, err := requireTaskSourcingAuthority(tx, source.ID, in.CreatedBy, in.TaskLinkID)
		if err != nil {
			return err
		}
		if task.WorkflowStatus != "editing" && task.WorkflowStatus != "converted_to_draft" && !(task.IsPrimary && source.Status == StatusDraftCreated && source.LifecycleStatus == LifecycleEditing && (task.WorkflowStatus == "" || task.WorkflowStatus == "needs_review")) {
			return fmt.Errorf("%w: only this task's editing internal draft may be updated", ErrWorkflowGate)
		}
		var dc demandCaseRow
		if err := tx.First(&dc, task.DemandCaseID).Error; err != nil {
			return err
		}
		if dc.OwnerID != in.CreatedBy {
			return fmt.Errorf("%w: update requires selected market Owner", ErrWorkflowGate)
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
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("sourcing_product_id = ? AND task_link_id = ?", id, task.ID).First(&draft).Error; err != nil {
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
			row := costRow{ProductID: draft.ProductID, ExperimentID: task.ExperimentID, CostType: cost.CostType, Amount: cost.Amount, Currency: cost.Currency, TruthStatus: cost.TruthStatus, SourceURI: cost.SourceURI, ObservedAt: cost.ObservedAt}
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
		updates := map[string]any{"approval_id": nil, "approval_status": "", "approval_content_sha256": "", "approval_rejection_reason": ""}
		query := tx.Model(&draft)
		if expectedVersion > 0 {
			query = query.Where("editable_version = ?", expectedVersion)
			updates["editable_version"] = expectedVersion + 1
		}
		updated := query.Updates(updates)
		if updated.Error != nil {
			return updated.Error
		}
		if expectedVersion > 0 && updated.RowsAffected != 1 {
			return fmt.Errorf("%w: draft was concurrently edited", ErrWorkflowGate)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if in.TaskLinkID > 0 {
		return s.GetTaskDraft(id, in.TaskLinkID)
	}
	return s.GetDraft(id)
}
