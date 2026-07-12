package sourcing1688

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
	"gorm.io/gorm"
)

const (
	AcceptancePassed  = "passed"
	AcceptanceBlocked = "blocked"
	AcceptanceUnknown = "unknown"
)

// AcceptanceEvidence is deliberately metadata-only. The report proves which
// persisted records were inspected without returning raw HTML, credentials or
// frozen platform request bodies.
type AcceptanceEvidence struct {
	Kind        string     `json:"kind"`
	ID          string     `json:"id"`
	TruthStatus string     `json:"truth_status,omitempty"`
	ObservedAt  *time.Time `json:"observed_at,omitempty"`
}

type AcceptanceItem struct {
	Number   int                  `json:"number"`
	Code     string               `json:"code"`
	Title    string               `json:"title"`
	Status   string               `json:"status"`
	Summary  string               `json:"summary"`
	Evidence []AcceptanceEvidence `json:"evidence"`
	Blockers []string             `json:"blockers"`
}

type AcceptanceReport struct {
	SourcingProductID int64            `json:"sourcing_product_id"`
	GeneratedAt       time.Time        `json:"generated_at"`
	Ready             bool             `json:"ready"`
	Status            string           `json:"status"`
	Items             []AcceptanceItem `json:"items"`
	Disclaimer        string           `json:"disclaimer"`
}

type acceptanceListingEvidence struct {
	LocalizedTitle       string               `json:"localized_title"`
	LocalizedDescription string               `json:"localized_description"`
	TargetLocale         string               `json:"target_locale"`
	ShippingTemplateID   string               `json:"shipping_template_id"`
	CategorySchemaURI    string               `json:"category_schema_uri"`
	CategoryObservedAt   time.Time            `json:"category_observed_at"`
	SupplierAssessment   []EvidenceCheck      `json:"supplier_assessment"`
	ComplianceChecks     []EvidenceCheck      `json:"compliance_checks"`
	MediaRequirements    []MediaInput         `json:"media_requirements"`
	ValidationInput      DraftValidationInput `json:"validation_input"`
	SourceSnapshotID     int64                `json:"source_snapshot_id"`
	SupplierSKUMapping   []DraftSKUInput      `json:"supplier_sku_mapping"`
}

func acceptanceItem(number int, code, title string) AcceptanceItem {
	return AcceptanceItem{Number: number, Code: code, Title: title, Status: AcceptanceUnknown, Evidence: []AcceptanceEvidence{}, Blockers: []string{}}
}

func (i *AcceptanceItem) pass(summary string) {
	i.Status, i.Summary, i.Blockers = AcceptancePassed, summary, []string{}
}

func (i *AcceptanceItem) block(summary string, blockers ...string) {
	i.Status, i.Summary = AcceptanceBlocked, summary
	i.Blockers = append(i.Blockers, blockers...)
}

func (i *AcceptanceItem) unknown(summary string, blockers ...string) {
	i.Status, i.Summary = AcceptanceUnknown, summary
	i.Blockers = append(i.Blockers, blockers...)
}

func evidence(kind string, id any, truth string, observed time.Time) AcceptanceEvidence {
	identifier := fmt.Sprint(id)
	if _, ok := id.(string); ok {
		sum := sha256.Sum256([]byte(identifier))
		identifier = "sha256:" + hex.EncodeToString(sum[:8])
	}
	ref := AcceptanceEvidence{Kind: kind, ID: identifier, TruthStatus: truth}
	if !observed.IsZero() {
		t := observed.UTC()
		ref.ObservedAt = &t
	}
	return ref
}

func realDriver(driver string) bool {
	v := strings.ToLower(strings.TrimSpace(driver))
	for _, allowed := range []string{"plugin", "playwright", "api1688"} {
		if v == allowed || strings.HasPrefix(v, allowed+"@") || strings.HasPrefix(v, allowed+"-") || strings.HasPrefix(v, allowed+"_") {
			return true
		}
	}
	return false
}

func validChecks(checks []EvidenceCheck, required []string, actualOnly bool) (bool, []AcceptanceEvidence, []string) {
	seen := map[string]bool{}
	refs := []AcceptanceEvidence{}
	problems := []string{}
	for _, check := range checks {
		if seen[check.CheckType] {
			problems = append(problems, "重复检查项: "+check.CheckType)
			continue
		}
		seen[check.CheckType] = true
		validTruth := check.TruthStatus == "actual" || (!actualOnly && check.TruthStatus == "quoted")
		if check.Result != "pass" || !validTruth || strings.TrimSpace(check.SourceURI) == "" || check.ObservedAt.IsZero() {
			problems = append(problems, "检查证据未通过: "+check.CheckType)
		}
		refs = append(refs, evidence("check:"+check.CheckType, check.SourceURI, check.TruthStatus, check.ObservedAt))
	}
	for _, key := range required {
		if !seen[key] {
			problems = append(problems, "缺少检查项: "+key)
		}
	}
	return len(problems) == 0 && len(seen) == len(required), refs, problems
}

func validationProblems(result ValidationResult) []string {
	out := make([]string, 0, len(result.Blockers))
	for _, blocker := range result.Blockers {
		out = append(out, blocker.Code+": "+blocker.Field)
	}
	return out
}

// BuildAcceptanceReport reads one Owner-scoped chain and evaluates the 15
// business requirements from persisted evidence. Code paths, tests and module
// existence are intentionally not inputs to this decision.
func (s *Service) BuildAcceptanceReport(ctx context.Context, sourceID, ownerID int64) (*AcceptanceReport, error) {
	if sourceID <= 0 || ownerID <= 0 {
		return nil, ErrWorkflowGate
	}
	report := &AcceptanceReport{SourcingProductID: sourceID, GeneratedAt: time.Now().UTC(), Status: AcceptanceUnknown, Items: make([]AcceptanceItem, 15), Disclaimer: "该报告只裁决此商品持久化证据链；工程模块或自动测试通过不能代替真实商品验收。"}
	titles := [][2]string{
		{"market_prerequisite", "目标市场前提"}, {"raw_source_evidence", "原始采集证据"}, {"dedupe_and_updates", "去重与更新"}, {"supplier_assessment", "供应商判断"}, {"sku_mapping", "SKU/变体映射"}, {"full_cost", "完整成本与汇率"}, {"image_rights", "图片权利"}, {"image_processing", "图片处理标准"}, {"compliance", "合规检查"}, {"localization", "内容本地化"}, {"category_rules", "类目与渠道属性"}, {"lifecycle_approval", "状态与审批"}, {"traceability", "全链路追溯"}, {"publish_safety", "独立发布安全"}, {"real_product_acceptance", "真实商品端到端验收"},
	}
	for idx, pair := range titles {
		report.Items[idx] = acceptanceItem(idx+1, pair[0], pair[1])
	}
	controlledRaw := false

	readReport := func(tx *gorm.DB) error {
		var source Sourcing1688Product
		if err := tx.Table("sourcing_1688_product AS sp").Joins("JOIN demand_case dc ON dc.id = sp.demand_case_id").Where("sp.id = ? AND dc.owner_id = ?", sourceID, ownerID).Select("sp.*").Take(&source).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: source does not belong to authenticated Owner", ErrWorkflowGate)
			}
			return err
		}

		var dc demandCaseRow
		var exp experimentRow
		market := &report.Items[0]
		if source.DemandCaseID == nil || source.ExperimentID == nil {
			market.unknown("尚未关联已批准市场与实验", "缺少 demand_case_id 或 experiment_id")
		} else if err := tx.First(&dc, *source.DemandCaseID).Error; err != nil {
			return err
		} else if err := tx.Where("experiment_id = ?", *source.ExperimentID).First(&exp).Error; err != nil {
			return err
		} else {
			var gateCount, demandLinkCount, sourceLinkCount int64
			if err := tx.Model(&gateRow{}).Where("experiment_id = ? AND stage = 'opportunity' AND result = 'pass'", *source.ExperimentID).Count(&gateCount).Error; err != nil {
				return err
			}
			if err := tx.Model(&objectLinkRow{}).Where("experiment_id = ? AND object_type = 'demand_case' AND object_id = ?", *source.ExperimentID, strconv.FormatInt(*source.DemandCaseID, 10)).Count(&demandLinkCount).Error; err != nil {
				return err
			}
			if err := tx.Model(&objectLinkRow{}).Where("experiment_id = ? AND object_type = 'sourcing_1688' AND object_id = ?", *source.ExperimentID, strconv.FormatInt(source.ID, 10)).Count(&sourceLinkCount).Error; err != nil {
				return err
			}
			market.Evidence = append(market.Evidence, evidence("demand_case", dc.ID, "actual", dc.UpdatedAt), evidence("experiment", exp.ExperimentID, "actual", time.Time{}), evidence("opportunity_gate", exp.ExperimentID, "actual", time.Time{}))
			validStage := exp.Stage == "product" || exp.Stage == "supply" || exp.Stage == "channel"
			if strings.TrimSpace(dc.Region) == "" || strings.TrimSpace(dc.Consumer) == "" || strings.TrimSpace(dc.SalesChannel) == "" || dc.Status != "experiment_ready" || dc.OwnerID != ownerID || exp.OwnerID != ownerID || exp.Status != "active" || !validStage || gateCount != 1 || demandLinkCount != 1 || sourceLinkCount != 1 {
				market.block("市场 tuple、Owner、实验或机会闸门不完整", "必须同时满足国家/地区、目标消费者、销售渠道、experiment_ready、active experiment 和 opportunity pass")
			} else {
				market.pass("已持久化国家/地区 × 目标消费者 × 销售渠道，并通过同一 Owner 的机会闸门")
			}
		}

		var snapshots []Sourcing1688Snapshot
		if err := tx.Where("sourcing_product_id = ?", source.ID).Order("id").Find(&snapshots).Error; err != nil {
			return err
		}
		rawItem := &report.Items[1]
		if source.SnapshotID == nil || len(snapshots) == 0 {
			rawItem.unknown("尚无不可变原始快照", "缺少当前 snapshot")
		} else {
			var current *Sourcing1688Snapshot
			for idx := range snapshots {
				if snapshots[idx].ID == *source.SnapshotID {
					current = &snapshots[idx]
					break
				}
			}
			if current == nil {
				rawItem.block("当前来源指向不存在的快照", "snapshot_id 与来源不匹配")
			} else {
				var page toolbridge.PageData
				sum := sha256.Sum256(current.RawPayload)
				canonicalSource, sourceErr := canonical1688URL(current.SourceURL)
				var canonicalPage string
				var pageErr error
				decodeErr := json.Unmarshal(current.RawPayload, &page)
				// Recompute page URL only after successful decoding.
				if decodeErr == nil {
					canonicalPage, pageErr = canonical1688URL(page.SourceURL)
				}
				controlledRaw = current.CaptureMode == CaptureModeControlledFetch && decodeErr == nil && sourceErr == nil && pageErr == nil && canonicalSource == canonicalPage && strings.TrimSpace(page.RawHTML) != "" && realDriver(current.Driver) && current.Driver == page.Driver && current.ParserVersion == page.ParserVersion && !current.CollectedAt.IsZero() && len(current.RawPayload) > 0 && hex.EncodeToString(sum[:]) == current.RawSHA256
				rawItem.Evidence = append(rawItem.Evidence, evidence("snapshot", current.ID, "actual", current.CollectedAt))
				if !controlledRaw {
					rawItem.block("当前快照不能证明一次真实受控页面采集", "需由 controlled_fetch 入口产生，并同时具备 1688 原链接、采集时间、白名单驱动、解析版本、raw_html、原始载荷和正确 SHA-256")
				} else {
					rawItem.pass("真实 1688 页面结构、原链接、采集时间、驱动、解析版本和原始哈希均已持久化")
				}
			}
		}

		identity := &report.Items[2]
		var changes []SourcingChangeEvent
		var duplicates []DuplicateCandidate
		if err := tx.Where("sourcing_product_id = ?", source.ID).Find(&changes).Error; err != nil {
			return err
		}
		if err := tx.Where("source_product_id = ? OR matched_product_id = ?", source.ID, source.ID).Find(&duplicates).Error; err != nil {
			return err
		}
		pending := 0
		identityProblems := []string{}
		for _, candidate := range duplicates {
			identity.Evidence = append(identity.Evidence, evidence("duplicate_candidate", candidate.ID, "actual", candidate.CreatedAt))
			if candidate.Status == "pending_review" {
				pending++
			} else if (candidate.Status != "same_product" && candidate.Status != "different_product") || candidate.ReviewedBy == nil || candidate.ReviewedAt == nil {
				identityProblems = append(identityProblems, "重复候选缺少有效 Owner 裁决")
			}
		}
		changeIndex := map[string]bool{}
		for _, change := range changes {
			identity.Evidence = append(identity.Evidence, evidence("change_event:"+change.ChangeType, change.ID, "actual", change.CreatedAt))
			changeIndex[fmt.Sprintf("%d:%s", change.CurrentSnapshotID, change.ChangeType)] = true
		}
		for idx := 1; idx < len(snapshots); idx++ {
			before, after := snapshots[idx-1], snapshots[idx]
			expected := []string{}
			if fmt.Sprint(optionalFloatValue(before.ObservedPrice)) != fmt.Sprint(optionalFloatValue(after.ObservedPrice)) {
				expected = append(expected, "price")
			}
			if before.ObservedMOQ != after.ObservedMOQ {
				expected = append(expected, "moq")
			}
			if before.ObservedSupplier != after.ObservedSupplier {
				expected = append(expected, "supplier")
			}
			if before.ObservedSupplierBusinessID != after.ObservedSupplierBusinessID {
				expected = append(expected, "supplier_business_id")
			}
			if before.SourceURL != after.SourceURL {
				expected = append(expected, "source_url")
			}
			if before.ProductFingerprint != after.ProductFingerprint {
				expected = append(expected, "product_identity")
			}
			for _, typ := range expected {
				if !changeIndex[fmt.Sprintf("%d:%s", after.ID, typ)] {
					identityProblems = append(identityProblems, fmt.Sprintf("快照 %d 缺少 %s 变化记录", after.ID, typ))
				}
			}
		}
		if source.SourceOfferID == "" || source.SourceProductFingerprint == "" || len(snapshots) == 0 {
			identity.unknown("尚不能证明去重身份已建立", "缺少 offer ID、内容指纹或快照")
		} else if pending > 0 {
			identity.block("存在尚未由 Owner 裁决的疑似同款", fmt.Sprintf("%d 个重复候选待复核", pending))
		} else if len(identityProblems) > 0 {
			identity.block("去重裁决或结构化变化记录不完整", identityProblems...)
		} else {
			identity.pass(fmt.Sprintf("已建立 offer/内容指纹，%d 个快照、%d 条变化、%d 个已裁决/无重复候选", len(snapshots), len(changes), len(duplicates)))
		}

		var draft draftRow
		var listing listingRow
		var product productRow
		var payload acceptanceListingEvidence
		draftFound := true
		if err := tx.Where("sourcing_product_id = ?", source.ID).Take(&draft).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				draftFound = false
			} else {
				return err
			}
		}
		payloadValid := false
		if draftFound {
			if err := tx.First(&listing, draft.ListingID).Error; err != nil {
				return err
			}
			if err := tx.First(&product, draft.ProductID).Error; err != nil {
				return err
			}
			payloadValid = json.Unmarshal(listing.PublishedData, &payload) == nil && payload.SourceSnapshotID > 0
		}
		if !draftFound || !payloadValid {
			for _, idx := range []int{3, 4, 5, 6, 7, 8, 9, 10} {
				report.Items[idx].unknown("尚无可审计的冻结草稿证据", "需先完成真实采集、复核和转草稿")
			}
		} else {
			supplier := &report.Items[3]
			ok, refs, problems := validChecks(payload.SupplierAssessment, []string{"identity", "operating_history", "transaction_history", "moq", "mixed_batch", "lead_time", "sample", "returns"}, false)
			supplier.Evidence = append(supplier.Evidence, refs...)
			if source.SupplierBusinessID == "" {
				ok = false
				problems = append(problems, "缺少供应商业务身份")
			}
			if ok {
				supplier.pass("供应商身份、经营年限、成交、MOQ、混批、交期、样品和退换货 8 项均有来源证据")
			} else {
				supplier.block("供应商 8 项判断不完整", problems...)
			}

			var skus []skuRow
			if err := tx.Where("product_id = ?", product.ID).Order("id").Find(&skus).Error; err != nil {
				return err
			}
			skuItem := &report.Items[4]
			skuValidation := ValidateSKUs(payload.ValidationInput.SKUs, payload.ValidationInput.SKURules)
			skuProblems := validationProblems(skuValidation)
			if len(payload.SupplierSKUMapping) != len(skus) || len(payload.ValidationInput.SKUs) != len(skus) {
				skuProblems = append(skuProblems, "冻结映射数量与持久化 SKU 数量不一致")
			}
			codes := map[string]bool{}
			for _, sku := range skus {
				codes[sku.Code] = true
				skuItem.Evidence = append(skuItem.Evidence, evidence("sku", sku.ID, "quoted", time.Time{}))
			}
			for _, mapping := range payload.SupplierSKUMapping {
				if !codes[mapping.InternalSKU] || mapping.SupplierSKU == "" || mapping.ChannelSKU == "" || mapping.Color == "" || mapping.Size == "" || mapping.Material == "" || mapping.Packaging == "" {
					skuProblems = append(skuProblems, "供应商/内部/渠道 SKU 或颜色、尺寸、材质、包装映射不完整")
				}
			}
			if len(skuProblems) == 0 {
				skuItem.pass("供应商 SKU、内部 SKU、渠道 SKU 及四项变体属性形成完整持久化映射")
			} else {
				skuItem.block("SKU/变体映射未通过", skuProblems...)
			}

			var costs []costRow
			if err := tx.Where("product_id = ? AND experiment_id = ?", product.ID, draft.ExperimentID).Find(&costs).Error; err != nil {
				return err
			}
			costItem := &report.Items[5]
			costResult := ValidateCosts(payload.ValidationInput.Costs)
			costProblems := validationProblems(costResult.ValidationResult)
			if len(costs) != len(requiredCostTypes) {
				costProblems = append(costProblems, "持久化费用不是完整 10 项")
			}
			if len(payload.ValidationInput.Costs.ExchangeRates) == 0 {
				costProblems = append(costProblems, "缺少独立汇率证据")
			}
			persistedCost := map[string]costRow{}
			for _, cost := range costs {
				persistedCost[cost.CostType] = cost
				costItem.Evidence = append(costItem.Evidence, evidence("cost:"+cost.CostType, cost.ID, cost.TruthStatus, cost.ObservedAt))
			}
			for _, line := range payload.ValidationInput.Costs.Costs {
				row, ok := persistedCost[line.Type]
				if !ok || row.Amount != line.Amount || !strings.EqualFold(row.Currency, line.Currency) || row.TruthStatus != line.TruthStatus || row.SourceURI != line.SourceURI || !row.ObservedAt.Equal(line.ObservedAt) {
					costProblems = append(costProblems, "冻结成本与持久化成本不一致: "+line.Type)
				}
			}
			for idx, rate := range payload.ValidationInput.Costs.ExchangeRates {
				costItem.Evidence = append(costItem.Evidence, evidence("exchange_rate", idx+1, rate.TruthStatus, rate.ObservedAt))
				if rate.Rate <= 0 || !usableTruthStatus(rate.TruthStatus) || rate.SourceURI == "" || rate.ObservedAt.IsZero() {
					costProblems = append(costProblems, "汇率证据无效")
				}
			}
			if len(costProblems) == 0 {
				costItem.pass("采购至退货损失 10 项成本、独立汇率和收入证据均可复算")
			} else {
				costItem.block("完整成本证据未通过", costProblems...)
			}

			var media []mediaRow
			var processing []ImageProcessingRecord
			if err := tx.Where("product_id = ?", product.ID).Order("id").Find(&media).Error; err != nil {
				return err
			}
			if err := tx.Where("sourcing_product_id = ? AND snapshot_id = ?", source.ID, draft.SnapshotID).Find(&processing).Error; err != nil {
				return err
			}
			rightsItem, processItem := &report.Items[6], &report.Items[7]
			rightsProblems, processProblems := []string{}, validationProblems(ValidateImages(payload.ValidationInput.Images, payload.ValidationInput.ImageRules))
			processedByHash := map[string]ImageProcessingRecord{}
			for _, record := range processing {
				processedByHash[record.ProcessedSHA256] = record
				processItem.Evidence = append(processItem.Evidence, evidence("image_processing", record.ID, record.RightsTruthStatus, record.RightsObservedAt))
			}
			if len(media) == 0 || len(media) != len(payload.ValidationInput.Images) {
				rightsProblems = append(rightsProblems, "图片记录或图片核验记录数量不完整")
			}
			for _, asset := range media {
				rightsItem.Evidence = append(rightsItem.Evidence, evidence("media_asset", asset.ID, "actual", time.Time{}))
				record, ok := processedByHash[asset.ContentSHA256]
				if asset.RightsStatus != "verified" || asset.RightsEvidenceURI == "" || asset.HasWatermark || asset.HasChineseText || asset.HasBrandMark {
					rightsProblems = append(rightsProblems, "图片权利或水印/中文/品牌核验未通过")
				}
				var requirement *MediaInput
				for idx := range payload.MediaRequirements {
					if payload.MediaRequirements[idx].ContentSHA256 == asset.ContentSHA256 {
						requirement = &payload.MediaRequirements[idx]
						break
					}
				}
				if !ok || requirement == nil || record.RightsTruthStatus != "actual" || record.RightsEvidenceURI != asset.RightsEvidenceURI || record.RightsObservedAt.IsZero() || requirement.RightsObservedAt.IsZero() || !record.RightsObservedAt.Equal(requirement.RightsObservedAt) || record.ProcessedBy != ownerID {
					rightsProblems = append(rightsProblems, "图片缺少 Owner actual 权利与处理证据绑定")
				}
				if !ok || len(record.ProcessedBytes) == 0 || record.SourceSHA256 == "" || record.ProcessedSHA256 == "" || record.ProcessorVersion == "" || len(record.Operations) == 0 || record.OutputWidth != asset.Width || record.OutputHeight != asset.Height || record.ChannelRuleURI != asset.ChannelRuleURI {
					processProblems = append(processProblems, "图片处理输出、版本、尺寸、操作或渠道规则绑定不完整")
				}
			}
			if len(rightsProblems) == 0 {
				rightsItem.pass("图片使用权及水印、中文、品牌标识均由 Owner actual 证据核验")
			} else {
				rightsItem.block("图片权利证据未通过", rightsProblems...)
			}
			if len(processProblems) == 0 {
				processItem.pass("图片真实执行裁切、缩放、白底处理，并绑定清晰度、数量和渠道规则")
			} else {
				processItem.block("图片处理标准未通过", processProblems...)
			}

			compliance := &report.Items[8]
			ok, refs, problems = validChecks(payload.ComplianceChecks, []string{"brand_ip", "patent", "certification", "dangerous_goods", "material", "labeling_instructions"}, true)
			compliance.Evidence = append(compliance.Evidence, refs...)
			if ok {
				compliance.pass("品牌、专利、认证、危险品、材质、标签/说明书 6 项均为 actual 通过")
			} else {
				compliance.block("合规 6 项 actual 证据未通过", problems...)
			}

			localization := &report.Items[9]
			localProblems := validationProblems(ValidateLocalization(payload.ValidationInput.Localization, payload.ValidationInput.LocalizationRules))
			localization.Evidence = append(localization.Evidence, evidence("localization_rule", payload.ValidationInput.LocalizationRules.Evidence.SourceURI, payload.ValidationInput.LocalizationRules.Evidence.TruthStatus, payload.ValidationInput.LocalizationRules.Evidence.ObservedAt))
			if payload.LocalizedTitle != payload.ValidationInput.Localization.Title || payload.LocalizedDescription != payload.ValidationInput.Localization.Description || payload.TargetLocale != payload.ValidationInput.Localization.Locale {
				localProblems = append(localProblems, "冻结本地化内容与规则输入不一致")
			}
			if len(localProblems) == 0 {
				localization.pass("标题、说明、卖点、属性、关键词、单位、语言和禁用词均按冻结规则校验")
			} else {
				localization.block("本地化证据未通过", localProblems...)
			}

			category := &report.Items[10]
			categoryProblems := validationProblems(ValidateChannelListing(payload.ValidationInput.Channel, payload.ValidationInput.ChannelRules))
			category.Evidence = append(category.Evidence, evidence("category_rule", payload.CategorySchemaURI, payload.ValidationInput.ChannelRules.Evidence.TruthStatus, payload.CategoryObservedAt))
			if payload.ValidationInput.Channel.PlatformID != listing.PlatformID || payload.CategorySchemaURI != payload.ValidationInput.Channel.CategorySchemaURI || payload.ShippingTemplateID != payload.ValidationInput.Channel.ShippingTemplateID {
				categoryProblems = append(categoryProblems, "类目、平台或配送模板绑定不一致")
			}
			var platform platformRow
			if err := tx.First(&platform, listing.PlatformID).Error; err != nil {
				return err
			}
			channel := strings.ToLower(dc.SalesChannel)
			if !strings.Contains(channel, strings.ToLower(platform.Code)) && !strings.Contains(channel, strings.ToLower(platform.Name)) {
				categoryProblems = append(categoryProblems, "草稿平台不属于已批准销售渠道")
			}
			if len(categoryProblems) == 0 {
				category.pass("必填属性、变体结构、图片规格与配送模板绑定当前渠道规则")
			} else {
				category.block("类目与渠道规则未通过", categoryProblems...)
			}
		}

		lifecycle := &report.Items[11]
		if !draftFound {
			lifecycle.unknown("尚未进入草稿审批", "需完成 editing → pending_approval → approved_draft")
		} else if source.LifecycleStatus != LifecycleApprovedDraft || source.ReviewedBy == nil || *source.ReviewedBy != ownerID || source.ReviewedAt == nil || draft.ApprovalID == nil || draft.ApprovalStatus != approval.StatusApproved {
			lifecycle.block("草稿尚未完成独立 Owner 批准", "当前生命周期必须为 approved_draft，草稿审批必须为 approved")
		} else {
			var req approval.ApprovalRequest
			if err := tx.First(&req, *draft.ApprovalID).Error; err != nil {
				return err
			}
			var auditCount int64
			if err := tx.Table("operation_log").Where("action = 'approval.review' AND (resource_id = ? OR entity_id = ?) AND operator = ? AND result IN ('approved','success')", strconv.FormatInt(req.ID, 10), draft.ID, strconv.FormatInt(ownerID, 10)).Count(&auditCount).Error; err != nil {
				return err
			}
			lifecycle.Evidence = append(lifecycle.Evidence, evidence("draft_approval", req.ID, "actual", req.UpdatedAt))
			contentErr := validateDraftApprovalContent(tx, &draft, &req)
			if req.RequestType != DraftApprovalRequestType || req.TargetType != DraftApprovalTargetType || req.TargetID != draft.ID || req.Status != approval.StatusApproved || req.ReviewerUserID == nil || *req.ReviewerUserID != ownerID || listing.Status != "draft" || contentErr != nil || auditCount == 0 {
				lifecycle.block("草稿审批对象、Owner 或内部草稿状态不一致", "批准不得越级且不得把 listing 变为已发布")
			} else {
				lifecycle.pass("已按状态机完成 Owner 草稿审批，批准后仍保持内部 draft")
			}
		}

		trace := &report.Items[12]
		if !draftFound {
			trace.unknown("尚无产品与草稿可追溯链", "缺少 draft/product/listing")
		} else {
			traceProblems := []string{}
			if source.SnapshotID == nil || source.ExperimentID == nil || source.DemandCaseID == nil || source.ProductID == nil {
				traceProblems = append(traceProblems, "source 缺少 snapshot/experiment/demand/product 关联")
			} else if draft.SnapshotID != *source.SnapshotID || draft.ProductID != product.ID || draft.ListingID != listing.ID || draft.ExperimentID != *source.ExperimentID || draft.DemandCaseID != *source.DemandCaseID || *source.ProductID != product.ID {
				traceProblems = append(traceProblems, "source/snapshot/draft/product/listing/experiment 主键链不一致")
			}
			var badMedia, badCosts int64
			if err := tx.Model(&mediaRow{}).Where("product_id = ? AND source_snapshot_id <> ?", product.ID, draft.SnapshotID).Count(&badMedia).Error; err != nil {
				return err
			}
			if err := tx.Model(&costRow{}).Where("product_id = ? AND experiment_id <> ?", product.ID, draft.ExperimentID).Count(&badCosts).Error; err != nil {
				return err
			}
			if badMedia > 0 || badCosts > 0 {
				traceProblems = append(traceProblems, "图片快照或成本实验关联不一致")
			}
			for typ, objectID := range map[string]string{"demand_case": strconv.FormatInt(draft.DemandCaseID, 10), "sourcing_1688": strconv.FormatInt(source.ID, 10), "product": strconv.FormatInt(product.ID, 10), "listing_draft": strconv.FormatInt(listing.ID, 10)} {
				var count int64
				if err := tx.Model(&objectLinkRow{}).Where("experiment_id = ? AND object_type = ? AND object_id = ?", draft.ExperimentID, typ, objectID).Count(&count).Error; err != nil {
					return err
				}
				if count != 1 {
					traceProblems = append(traceProblems, "缺少实验对象关联: "+typ)
				}
			}
			trace.Evidence = append(trace.Evidence, evidence("source", source.ID, "actual", source.UpdatedAt), evidence("snapshot", draft.SnapshotID, "actual", time.Time{}), evidence("product", product.ID, "actual", product.UpdatedAt), evidence("listing", listing.ID, "actual", time.Time{}), evidence("draft", draft.ID, "actual", draft.CreatedAt))
			if len(traceProblems) == 0 {
				trace.pass("采集、快照、产品、图片、草稿及实验对象关联使用同一事实链")
			} else {
				trace.block("可追溯关系未通过", traceProblems...)
			}
		}

		publish := &report.Items[13]
		if !draftFound {
			publish.unknown("尚无内部草稿，无法判断发布隔离", "先生成受控草稿")
		} else {
			var attempts []PublishAttempt
			if err := tx.Where("sourcing_product_id = ?", source.ID).Order("id").Find(&attempts).Error; err != nil {
				return err
			}
			publishProblems := []string{}
			if len(attempts) == 0 {
				if listing.Status != "draft" || listing.PlatformProductID != "" || listing.PlatformURL != "" || listing.LastSyncAt != nil {
					publishProblems = append(publishProblems, "未申请发布但 listing 已出现外部发布结果")
				}
			} else {
				for _, attempt := range attempts {
					publish.Evidence = append(publish.Evidence, evidence("publish_attempt", attempt.ID, "actual", attempt.RequestedAt))
					if attempt.ApprovalID == nil || draft.ApprovalID == nil || *attempt.ApprovalID == *draft.ApprovalID || attempt.DraftID != draft.ID || attempt.ProductID != product.ID || attempt.ListingID != listing.ID || attempt.ExperimentID != draft.ExperimentID || attempt.RequestedBy != ownerID || len(attempt.RequestPayload) == 0 {
						publishProblems = append(publishProblems, "发布申请未与草稿隔离或冻结请求不完整")
						continue
					}
					hash := sha256.Sum256(attempt.RequestPayload)
					if hex.EncodeToString(hash[:]) != attempt.RequestSHA256 {
						publishProblems = append(publishProblems, "发布请求哈希不匹配")
					}
					var req approval.ApprovalRequest
					if err := tx.First(&req, *attempt.ApprovalID).Error; err != nil {
						return err
					}
					if req.RequestType != PublishApprovalRequestType || req.TargetType != PublishApprovalTargetType || req.TargetID != attempt.ID || req.RiskLevel != "high" {
						publishProblems = append(publishProblems, "发布审批不是独立高风险审批")
					}
					switch attempt.Status {
					case PublishStatusPending:
						if req.Status != approval.StatusPending {
							publishProblems = append(publishProblems, "待审批发布申请与审批状态不一致")
						}
					case PublishStatusRejected:
						if req.Status != approval.StatusRejected {
							publishProblems = append(publishProblems, "已拒绝发布申请与审批状态不一致")
						}
					default:
						if req.Status != approval.StatusApproved || req.ReviewerUserID == nil || *req.ReviewerUserID != ownerID {
							publishProblems = append(publishProblems, "执行/结果缺少 Owner 发布批准")
						}
					}
					if attempt.Status == PublishStatusExecuting {
						publishProblems = append(publishProblems, "发布仍处于 executing，结果尚未落账")
					}
					if attempt.Status == PublishStatusExecuting || attempt.Status == PublishStatusSubmitted || attempt.Status == PublishStatusReconcile || attempt.Status == PublishStatusSucceeded || attempt.Status == PublishStatusFailed {
						if len(attempt.AdapterRequestPayload) == 0 {
							publishProblems = append(publishProblems, "已执行发布缺少冻结适配器请求")
						}
					}
					if attempt.Status == PublishStatusSubmitted || attempt.Status == PublishStatusSucceeded || attempt.Status == PublishStatusFailed {
						if attempt.ExecutedAt == nil || attempt.CompletedAt == nil || len(attempt.ResponsePayload) == 0 || attempt.ResponseSHA256 == "" {
							publishProblems = append(publishProblems, "平台终态缺少执行时间或返回结果")
						} else {
							responseHash := sha256.Sum256(attempt.ResponsePayload)
							if hex.EncodeToString(responseHash[:]) != attempt.ResponseSHA256 {
								publishProblems = append(publishProblems, "平台返回结果哈希不匹配")
							}
						}
					}
				}
			}
			if len(publishProblems) == 0 {
				publish.pass("草稿与真实发布分离；任何发布请求均使用第二次高风险审批并保留冻结请求/平台结果")
			} else {
				publish.block("独立发布安全证据未通过", publishProblems...)
			}
		}

		return nil
	}
	var err error
	if s.db.Dialector.Name() == "postgres" {
		err = s.db.WithContext(ctx).Transaction(readReport, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	} else {
		err = s.db.WithContext(ctx).Transaction(readReport)
	}
	if err != nil {
		return nil, err
	}

	real := &report.Items[14]
	allPriorPassed := true
	for idx := 0; idx < 14; idx++ {
		if report.Items[idx].Status != AcceptancePassed {
			allPriorPassed = false
			break
		}
	}
	if controlledRaw && allPriorPassed {
		real.pass("该真实 1688 商品已从市场闸门、页面采集、Owner 复核、产品/图片/成本到内部批准草稿逐项通过")
	} else if !controlledRaw {
		real.unknown("尚未完成真实 1688 商品验收", "必须由 controlled fetch 保存非 mock raw_html 后再逐项验收")
	} else {
		real.block("真实商品链仍有未通过项", "前 14 项必须全部 passed")
	}

	report.Ready = true
	for _, item := range report.Items {
		if item.Status != AcceptancePassed {
			report.Ready = false
			break
		}
	}
	if report.Ready {
		report.Status = AcceptancePassed
	} else {
		report.Status = AcceptanceUnknown
		for _, item := range report.Items {
			if item.Status == AcceptanceBlocked {
				report.Status = AcceptanceBlocked
				break
			}
		}
	}
	return report, nil
}
