package sourcing1688

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"
)

// ValidationBlocker is a machine-readable reason that prevents a listing draft
// from passing a deterministic gate. Validation never performs external I/O.
type ValidationBlocker struct {
	Code    string `json:"code"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationResult struct {
	Passed   bool                `json:"passed"`
	Blockers []ValidationBlocker `json:"blockers"`
}

func newValidationResult() ValidationResult {
	return ValidationResult{Passed: true, Blockers: []ValidationBlocker{}}
}

func (r *ValidationResult) block(code, field, message string) {
	r.Passed = false
	r.Blockers = append(r.Blockers, ValidationBlocker{Code: code, Field: field, Message: message})
}

func (r *ValidationResult) merge(other ValidationResult) {
	if !other.Passed {
		r.Passed = false
	}
	r.Blockers = append(r.Blockers, other.Blockers...)
}

func usableTruthStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "actual", "quoted", "estimated":
		return true
	default:
		return false
	}
}

type RuleEvidence struct {
	TruthStatus string    `json:"truth_status"`
	SourceURI   string    `json:"source_uri"`
	ObservedAt  time.Time `json:"observed_at"`
}

func validateRuleEvidence(r *ValidationResult, field string, e RuleEvidence) {
	if !usableTruthStatus(e.TruthStatus) {
		r.block("untrusted_evidence", field+".truth_status", "规则证据不能是 unknown、mock、inferred 或空值")
	}
	if strings.TrimSpace(e.SourceURI) == "" {
		r.block("missing_source", field+".source_uri", "规则快照缺少来源")
	}
	if e.ObservedAt.IsZero() {
		r.block("missing_observed_at", field+".observed_at", "规则快照缺少观察时间")
	}
}

type LocalizationInput struct {
	Locale       string            `json:"locale"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	BulletPoints []string          `json:"bullet_points"`
	Keywords     []string          `json:"keywords"`
	Attributes   map[string]string `json:"attributes"`
	Unit         string            `json:"unit"`
}

type LocalizationRuleSnapshot struct {
	Evidence        RuleEvidence `json:"evidence"`
	Locale          string       `json:"locale"`
	AllowedScripts  []string     `json:"allowed_scripts"`
	MinTitleLength  int          `json:"min_title_length"`
	MaxTitleLength  int          `json:"max_title_length"`
	MinBulletPoints int          `json:"min_bullet_points"`
	MaxBulletLength int          `json:"max_bullet_length"`
	MinKeywords     int          `json:"min_keywords"`
	AllowedUnits    []string     `json:"allowed_units"`
	ProhibitedWords []string     `json:"prohibited_words"`
}

func ValidateLocalization(in LocalizationInput, rules LocalizationRuleSnapshot) ValidationResult {
	r := newValidationResult()
	validateRuleEvidence(&r, "localization_rules", rules.Evidence)
	if strings.TrimSpace(in.Locale) == "" || !strings.EqualFold(strings.TrimSpace(in.Locale), strings.TrimSpace(rules.Locale)) {
		r.block("locale_mismatch", "locale", "内容语言区域必须与目标市场规则一致")
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		r.block("missing_title", "title", "缺少目标语言标题")
	} else if (rules.MinTitleLength > 0 && len([]rune(title)) < rules.MinTitleLength) || (rules.MaxTitleLength > 0 && len([]rune(title)) > rules.MaxTitleLength) {
		r.block("title_length", "title", "标题长度不符合渠道规则")
	}
	validateLocalizedScript(&r, "title", title, rules.AllowedScripts)
	if strings.TrimSpace(in.Description) == "" {
		r.block("missing_description", "description", "缺少目标语言说明")
	}
	validateLocalizedScript(&r, "description", in.Description, rules.AllowedScripts)
	if len(in.BulletPoints) < rules.MinBulletPoints {
		r.block("missing_bullet_points", "bullet_points", "卖点数量不足")
	}
	for i, bullet := range in.BulletPoints {
		field := fmt.Sprintf("bullet_points[%d]", i)
		if strings.TrimSpace(bullet) == "" {
			r.block("empty_bullet_point", field, "卖点不能为空")
		} else if rules.MaxBulletLength > 0 && len([]rune(strings.TrimSpace(bullet))) > rules.MaxBulletLength {
			r.block("bullet_length", field, "卖点长度超过渠道规则")
		}
		validateLocalizedScript(&r, field, bullet, rules.AllowedScripts)
	}
	if len(in.Keywords) < rules.MinKeywords {
		r.block("missing_keywords", "keywords", "关键词数量不足")
	}
	for i, keyword := range in.Keywords {
		if strings.TrimSpace(keyword) == "" {
			r.block("empty_keyword", fmt.Sprintf("keywords[%d]", i), "关键词不能为空")
		}
		validateLocalizedScript(&r, fmt.Sprintf("keywords[%d]", i), keyword, rules.AllowedScripts)
	}
	for name, value := range in.Attributes {
		if strings.TrimSpace(value) == "" {
			r.block("empty_attribute", "attributes."+name, "本地化属性不能为空")
		}
		validateLocalizedScript(&r, "attributes."+name, value, rules.AllowedScripts)
	}
	if !containsFold(rules.AllowedUnits, in.Unit) {
		r.block("invalid_unit", "unit", "计量单位不在目标渠道允许列表中")
	}
	allParts := append([]string{title, in.Description}, in.BulletPoints...)
	allParts = append(allParts, in.Keywords...)
	for _, value := range in.Attributes {
		allParts = append(allParts, value)
	}
	allText := strings.ToLower(strings.Join(allParts, " "))
	for _, word := range rules.ProhibitedWords {
		if w := strings.ToLower(strings.TrimSpace(word)); w != "" && strings.Contains(allText, w) {
			r.block("prohibited_word", "localized_content", fmt.Sprintf("内容包含禁用词 %q", word))
		}
	}
	return r
}

type ChannelListingInput struct {
	PlatformID         int64             `json:"platform_id"`
	CategoryID         string            `json:"category_id"`
	CategorySchemaURI  string            `json:"category_schema_uri"`
	CategoryObservedAt time.Time         `json:"category_observed_at"`
	Attributes         map[string]string `json:"attributes"`
	VariantDimensions  []string          `json:"variant_dimensions"`
	ImageCount         int               `json:"image_count"`
	ImageWidths        []int             `json:"image_widths"`
	ImageHeights       []int             `json:"image_heights"`
	ShippingTemplateID string            `json:"shipping_template_id"`
}

type ChannelRuleSnapshot struct {
	Evidence                   RuleEvidence `json:"evidence"`
	PlatformID                 int64        `json:"platform_id"`
	CategoryID                 string       `json:"category_id"`
	RequiredAttributes         []string     `json:"required_attributes"`
	RequiredVariantDimensions  []string     `json:"required_variant_dimensions"`
	AllowedVariantDimensions   []string     `json:"allowed_variant_dimensions"`
	MinImages                  int          `json:"min_images"`
	MaxImages                  int          `json:"max_images"`
	MinImageWidth              int          `json:"min_image_width"`
	MinImageHeight             int          `json:"min_image_height"`
	AllowedShippingTemplateIDs []string     `json:"allowed_shipping_template_ids"`
}

func ValidateChannelListing(in ChannelListingInput, rules ChannelRuleSnapshot) ValidationResult {
	r := newValidationResult()
	validateRuleEvidence(&r, "channel_rules", rules.Evidence)
	if in.PlatformID <= 0 || in.PlatformID != rules.PlatformID {
		r.block("platform_mismatch", "platform_id", "渠道规则必须绑定已批准平台")
	}
	if in.CategorySchemaURI != rules.Evidence.SourceURI || !in.CategoryObservedAt.Equal(rules.Evidence.ObservedAt) {
		r.block("category_schema_mismatch", "category_schema", "类目规则来源和观察时间必须与规则快照一致")
	}
	if strings.TrimSpace(in.CategoryID) == "" || in.CategoryID != rules.CategoryID {
		r.block("category_mismatch", "category_id", "类目必须与规则快照一致")
	}
	for _, name := range rules.RequiredAttributes {
		if strings.TrimSpace(in.Attributes[name]) == "" {
			r.block("missing_category_attribute", "attributes."+name, "缺少渠道类目必填属性")
		}
	}
	seenDimensions := make(map[string]bool, len(in.VariantDimensions))
	for i, dimension := range in.VariantDimensions {
		key := strings.ToLower(strings.TrimSpace(dimension))
		if key == "" || seenDimensions[key] {
			r.block("invalid_variant_dimension", fmt.Sprintf("variant_dimensions[%d]", i), "变体维度不能为空或重复")
			continue
		}
		seenDimensions[key] = true
		if len(rules.AllowedVariantDimensions) > 0 && !containsFold(rules.AllowedVariantDimensions, dimension) {
			r.block("unsupported_variant_dimension", fmt.Sprintf("variant_dimensions[%d]", i), "变体维度不被类目规则允许")
		}
	}
	for _, dimension := range rules.RequiredVariantDimensions {
		if !containsFold(in.VariantDimensions, dimension) {
			r.block("missing_variant_dimension", "variant_dimensions", fmt.Sprintf("缺少变体维度 %q", dimension))
		}
	}
	if in.ImageCount != len(in.ImageWidths) || in.ImageCount != len(in.ImageHeights) {
		r.block("image_count_mismatch", "image_count", "图片数量与图片规格记录不一致")
	}
	if in.ImageCount < rules.MinImages || (rules.MaxImages > 0 && in.ImageCount > rules.MaxImages) {
		r.block("invalid_image_count", "image_count", "图片数量不符合渠道规则")
	}
	for i := 0; i < len(in.ImageWidths) && i < len(in.ImageHeights); i++ {
		if in.ImageWidths[i] < rules.MinImageWidth || in.ImageHeights[i] < rules.MinImageHeight {
			r.block("invalid_image_dimensions", fmt.Sprintf("images[%d]", i), "图片尺寸低于渠道最低要求")
		}
	}
	if !containsExact(rules.AllowedShippingTemplateIDs, in.ShippingTemplateID) {
		r.block("invalid_shipping_template", "shipping_template_id", "配送模板未被当前渠道规则允许")
	}
	return r
}

var requiredCostTypes = [...]string{
	"purchase", "domestic_shipping", "packaging", "cross_border_shipping", "platform_fee",
	"payment_fee", "advertising", "tax", "duty", "return_loss",
}

// RequiredSourcingCostTypes returns a defensive copy of the canonical 10-item
// expense checklist used by the 1688 draft gate. Exchange rates are separate
// evidence because they convert amounts rather than add another expense.
func RequiredSourcingCostTypes() []string {
	return append([]string(nil), requiredCostTypes[:]...)
}

type CostLine struct {
	Type        string    `json:"type"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	TruthStatus string    `json:"truth_status"`
	SourceURI   string    `json:"source_uri"`
	ObservedAt  time.Time `json:"observed_at"`
}

type ExchangeRate struct {
	FromCurrency string    `json:"from_currency"`
	ToCurrency   string    `json:"to_currency"`
	Rate         float64   `json:"rate"`
	TruthStatus  string    `json:"truth_status"`
	SourceURI    string    `json:"source_uri"`
	ObservedAt   time.Time `json:"observed_at"`
}

type RevenueInput struct {
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	TruthStatus string    `json:"truth_status"`
	SourceURI   string    `json:"source_uri"`
	ObservedAt  time.Time `json:"observed_at"`
}

type CostValidationInput struct {
	TargetCurrency string         `json:"target_currency"`
	Costs          []CostLine     `json:"costs"`
	ExchangeRates  []ExchangeRate `json:"exchange_rates"`
	Revenue        RevenueInput   `json:"revenue"`
}

type CostValidationResult struct {
	ValidationResult
	TotalCost          float64 `json:"total_cost"`
	EstimatedRevenue   float64 `json:"estimated_revenue"`
	ContributionProfit float64 `json:"contribution_profit"`
	Currency           string  `json:"currency"`
}

func ValidateCosts(in CostValidationInput) CostValidationResult {
	out := CostValidationResult{ValidationResult: newValidationResult(), Currency: strings.ToUpper(strings.TrimSpace(in.TargetCurrency))}
	if out.Currency == "" {
		out.block("missing_target_currency", "target_currency", "缺少统一核算币种")
	}
	seen := make(map[string]bool, len(requiredCostTypes))
	allowed := make(map[string]bool, len(requiredCostTypes))
	for _, typ := range requiredCostTypes {
		allowed[typ] = true
	}
	for i, cost := range in.Costs {
		field := fmt.Sprintf("costs[%d]", i)
		if !allowed[cost.Type] {
			out.block("invalid_cost_type", field+".type", "费用类型不在完整成本清单中")
			continue
		}
		if seen[cost.Type] {
			out.block("duplicate_cost_type", field+".type", "同一费用类型只能出现一次")
			continue
		}
		seen[cost.Type] = true
		validateFinancialEvidence(&out.ValidationResult, field, cost.Amount, cost.Currency, cost.TruthStatus, cost.SourceURI, cost.ObservedAt)
		converted, ok := convertCurrency(cost.Amount, cost.Currency, out.Currency, in.ExchangeRates, &out.ValidationResult, field+".currency")
		if ok {
			out.TotalCost += converted
		}
	}
	for _, typ := range requiredCostTypes {
		if !seen[typ] {
			out.block("missing_cost_type", "costs."+typ, "缺少完整成本项")
		}
	}
	validateFinancialEvidence(&out.ValidationResult, "revenue", in.Revenue.Amount, in.Revenue.Currency, in.Revenue.TruthStatus, in.Revenue.SourceURI, in.Revenue.ObservedAt)
	if in.Revenue.Amount <= 0 {
		out.block("invalid_revenue", "revenue.amount", "预计收入必须大于零")
	}
	if converted, ok := convertCurrency(in.Revenue.Amount, in.Revenue.Currency, out.Currency, in.ExchangeRates, &out.ValidationResult, "revenue.currency"); ok {
		out.EstimatedRevenue = converted
	}
	out.TotalCost = roundMoney(out.TotalCost)
	out.EstimatedRevenue = roundMoney(out.EstimatedRevenue)
	out.ContributionProfit = roundMoney(out.EstimatedRevenue - out.TotalCost)
	return out
}

func validateFinancialEvidence(r *ValidationResult, field string, amount float64, currency, truth, source string, observed time.Time) {
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount < 0 {
		r.block("invalid_amount", field+".amount", "金额必须是非负有限数")
	}
	if strings.TrimSpace(currency) == "" {
		r.block("missing_currency", field+".currency", "金额缺少币种")
	}
	if !usableTruthStatus(truth) {
		r.block("untrusted_evidence", field+".truth_status", "金额证据不能是 unknown、mock、inferred 或空值")
	}
	if strings.TrimSpace(source) == "" {
		r.block("missing_source", field+".source_uri", "金额证据缺少来源")
	}
	if observed.IsZero() {
		r.block("missing_observed_at", field+".observed_at", "金额证据缺少观察时间")
	}
}

func convertCurrency(amount float64, from, to string, rates []ExchangeRate, r *ValidationResult, field string) (float64, bool) {
	from, to = strings.ToUpper(strings.TrimSpace(from)), strings.ToUpper(strings.TrimSpace(to))
	if from == "" || to == "" {
		return 0, false
	}
	if from == to {
		return amount, true
	}
	for i, rate := range rates {
		rf, rt := strings.ToUpper(strings.TrimSpace(rate.FromCurrency)), strings.ToUpper(strings.TrimSpace(rate.ToCurrency))
		if (rf == from && rt == to) || (rf == to && rt == from) {
			prefix := fmt.Sprintf("exchange_rates[%d]", i)
			if rate.Rate <= 0 || math.IsNaN(rate.Rate) || math.IsInf(rate.Rate, 0) {
				r.block("invalid_exchange_rate", prefix+".rate", "汇率必须大于零且为有限数")
				return 0, false
			}
			if !usableTruthStatus(rate.TruthStatus) || strings.TrimSpace(rate.SourceURI) == "" || rate.ObservedAt.IsZero() {
				r.block("untrusted_exchange_rate", prefix, "汇率缺少可信来源、观察时间或证据等级")
				return 0, false
			}
			if rf == from {
				return amount * rate.Rate, true
			}
			return amount / rate.Rate, true
		}
	}
	r.block("missing_exchange_rate", field, fmt.Sprintf("缺少 %s 到 %s 的汇率", from, to))
	return 0, false
}

type ImageValidationInput struct {
	Role           string    `json:"role"`
	Width          int       `json:"width"`
	Height         int       `json:"height"`
	Background     string    `json:"background"`
	Cropped        bool      `json:"cropped"`
	ClarityScore   float64   `json:"clarity_score"`
	HasWatermark   bool      `json:"has_watermark"`
	HasChineseText bool      `json:"has_chinese_text"`
	HasBrandMark   bool      `json:"has_brand_mark"`
	TruthStatus    string    `json:"truth_status"`
	SourceURI      string    `json:"source_uri"`
	ObservedAt     time.Time `json:"observed_at"`
}

type ImageRuleSnapshot struct {
	Evidence           RuleEvidence `json:"evidence"`
	MinMainWidth       int          `json:"min_main_width"`
	MinMainHeight      int          `json:"min_main_height"`
	AllowedBackgrounds []string     `json:"allowed_backgrounds"`
	RequireCrop        bool         `json:"require_crop"`
	MinClarityScore    float64      `json:"min_clarity_score"`
	MinImages          int          `json:"min_images"`
	MaxImages          int          `json:"max_images"`
}

func ValidateImages(images []ImageValidationInput, rules ImageRuleSnapshot) ValidationResult {
	r := newValidationResult()
	validateRuleEvidence(&r, "image_rules", rules.Evidence)
	if len(images) < rules.MinImages || (rules.MaxImages > 0 && len(images) > rules.MaxImages) {
		r.block("invalid_image_count", "images", "图片数量不符合处理规则")
	}
	mainCount := 0
	for i, img := range images {
		field := fmt.Sprintf("images[%d]", i)
		if img.Role == "main" {
			mainCount++
			if img.Width < rules.MinMainWidth || img.Height < rules.MinMainHeight {
				r.block("invalid_main_dimensions", field, "主图尺寸低于规则要求")
			}
		}
		if !containsFold(rules.AllowedBackgrounds, img.Background) {
			r.block("invalid_background", field+".background", "图片背景不符合规则")
		}
		if rules.RequireCrop && !img.Cropped {
			r.block("crop_required", field+".cropped", "图片缺少裁切记录")
		}
		if img.ClarityScore < rules.MinClarityScore || math.IsNaN(img.ClarityScore) || math.IsInf(img.ClarityScore, 0) {
			r.block("insufficient_clarity", field+".clarity_score", "图片清晰度不足")
		}
		if img.HasWatermark {
			r.block("watermark_present", field, "图片仍含水印")
		}
		if img.HasChineseText {
			r.block("chinese_text_present", field, "图片仍含中文")
		}
		if img.HasBrandMark {
			r.block("brand_mark_present", field, "图片仍含品牌标识")
		}
		if strings.ToLower(strings.TrimSpace(img.TruthStatus)) != "actual" {
			r.block("untrusted_evidence", field+".truth_status", "水印、文字、品牌与清晰度检查必须由 Owner 实际核验")
		}
		if strings.TrimSpace(img.SourceURI) == "" {
			r.block("missing_source", field+".source_uri", "图片检查缺少来源")
		}
		if img.ObservedAt.IsZero() {
			r.block("missing_observed_at", field+".observed_at", "图片检查缺少观察时间")
		}
	}
	if mainCount != 1 {
		r.block("invalid_main_image_count", "images", "必须且只能有一张主图")
	}
	return r
}

type SKUValidationInput struct {
	SupplierSKU string    `json:"supplier_sku"`
	InternalSKU string    `json:"internal_sku"`
	ChannelSKU  string    `json:"channel_sku"`
	Color       string    `json:"color"`
	Size        string    `json:"size"`
	Material    string    `json:"material"`
	Packaging   string    `json:"packaging"`
	TruthStatus string    `json:"truth_status"`
	SourceURI   string    `json:"source_uri"`
	ObservedAt  time.Time `json:"observed_at"`
}

type SKUValidationRules struct {
	Evidence         RuleEvidence `json:"evidence"`
	RequireColor     bool         `json:"require_color"`
	RequireSize      bool         `json:"require_size"`
	RequireMaterial  bool         `json:"require_material"`
	RequirePackaging bool         `json:"require_packaging"`
}

func ValidateSKUs(skus []SKUValidationInput, rules SKUValidationRules) ValidationResult {
	r := newValidationResult()
	validateRuleEvidence(&r, "sku_rules", rules.Evidence)
	if len(skus) == 0 {
		r.block("missing_skus", "skus", "至少需要一个 SKU")
	}
	suppliers, internals, channels := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for i, sku := range skus {
		field := fmt.Sprintf("skus[%d]", i)
		validateSKUIdentifier(&r, suppliers, sku.SupplierSKU, field+".supplier_sku", "supplier")
		validateSKUIdentifier(&r, internals, sku.InternalSKU, field+".internal_sku", "internal")
		validateSKUIdentifier(&r, channels, sku.ChannelSKU, field+".channel_sku", "channel")
		if rules.RequireColor && strings.TrimSpace(sku.Color) == "" {
			r.block("missing_color", field+".color", "SKU 缺少颜色")
		}
		if rules.RequireSize && strings.TrimSpace(sku.Size) == "" {
			r.block("missing_size", field+".size", "SKU 缺少尺寸")
		}
		if rules.RequireMaterial && strings.TrimSpace(sku.Material) == "" {
			r.block("missing_material", field+".material", "SKU 缺少材质")
		}
		if rules.RequirePackaging && strings.TrimSpace(sku.Packaging) == "" {
			r.block("missing_packaging", field+".packaging", "SKU 缺少包装")
		}
		if !usableTruthStatus(sku.TruthStatus) {
			r.block("untrusted_evidence", field+".truth_status", "SKU 映射证据不能是 unknown、mock、inferred 或空值")
		}
		if strings.TrimSpace(sku.SourceURI) == "" {
			r.block("missing_source", field+".source_uri", "SKU 映射缺少来源")
		}
		if sku.ObservedAt.IsZero() {
			r.block("missing_observed_at", field+".observed_at", "SKU 映射缺少观察时间")
		}
	}
	return r
}

func validateSKUIdentifier(r *ValidationResult, seen map[string]bool, value, field, kind string) {
	v := strings.TrimSpace(value)
	if v == "" {
		r.block("missing_sku_mapping", field, "供应商、内部和渠道 SKU 必须形成完整映射")
		return
	}
	if seen[v] {
		r.block("duplicate_"+kind+"_sku", field, "SKU 映射值重复")
		return
	}
	seen[v] = true
}

type DraftValidationInput struct {
	Localization      LocalizationInput        `json:"localization"`
	LocalizationRules LocalizationRuleSnapshot `json:"localization_rules"`
	Channel           ChannelListingInput      `json:"channel"`
	ChannelRules      ChannelRuleSnapshot      `json:"channel_rules"`
	Costs             CostValidationInput      `json:"costs"`
	Images            []ImageValidationInput   `json:"images"`
	ImageRules        ImageRuleSnapshot        `json:"image_rules"`
	SKUs              []SKUValidationInput     `json:"skus"`
	SKURules          SKUValidationRules       `json:"sku_rules"`
}

type DraftValidationResult struct {
	ValidationResult `json:"validation"`
	Cost             CostValidationResult `json:"cost"`
}

func ValidateDraft(in DraftValidationInput) DraftValidationResult {
	r := DraftValidationResult{ValidationResult: newValidationResult()}
	r.merge(ValidateLocalization(in.Localization, in.LocalizationRules))
	r.merge(ValidateChannelListing(in.Channel, in.ChannelRules))
	r.Cost = ValidateCosts(in.Costs)
	r.merge(r.Cost.ValidationResult)
	r.merge(ValidateImages(in.Images, in.ImageRules))
	r.merge(ValidateSKUs(in.SKUs, in.SKURules))
	return r
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}

func containsExact(values []string, target string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == strings.TrimSpace(target) && strings.TrimSpace(target) != "" {
			return true
		}
	}
	return false
}

func validateLocalizedScript(r *ValidationResult, field, value string, allowed []string) {
	if strings.TrimSpace(value) == "" || len(allowed) == 0 {
		return
	}
	foundLetter := false
	for _, char := range value {
		if !unicode.IsLetter(char) {
			continue
		}
		foundLetter = true
		matched := false
		for _, script := range allowed {
			var table *unicode.RangeTable
			switch strings.ToLower(strings.TrimSpace(script)) {
			case "latin":
				table = unicode.Latin
			case "cyrillic":
				table = unicode.Cyrillic
			case "han":
				table = unicode.Han
			case "arabic":
				table = unicode.Arabic
			}
			if table != nil && unicode.Is(table, char) {
				matched = true
				break
			}
		}
		if !matched {
			r.block("invalid_locale_script", field, "内容包含目标 locale 不允许的文字脚本")
			return
		}
	}
	if !foundLetter {
		r.block("missing_locale_text", field, "目标语言内容必须包含文字")
	}
}

func roundMoney(value float64) float64 { return math.Round(value*100) / 100 }
