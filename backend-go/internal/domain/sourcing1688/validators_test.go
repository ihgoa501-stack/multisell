package sourcing1688

import (
	"strings"
	"testing"
	"time"
)

var validatorNow = time.Date(2026, 7, 12, 8, 0, 0, 0, time.UTC)

func ruleEvidence() RuleEvidence {
	return RuleEvidence{TruthStatus: "quoted", SourceURI: "https://rules.example/snapshot", ObservedAt: validatorNow}
}

func TestValidateLocalization(t *testing.T) {
	rules := LocalizationRuleSnapshot{Evidence: ruleEvidence(), Locale: "ru-RU", AllowedScripts: []string{"cyrillic"}, MinTitleLength: 5, MaxTitleLength: 80, MinBulletPoints: 2, MaxBulletLength: 100, MinKeywords: 2, AllowedUnits: []string{"шт"}, ProhibitedWords: []string{"лучший"}}
	valid := LocalizationInput{Locale: "ru-RU", Title: "Игрушка для кошек", Description: "Безопасная игрушка", BulletPoints: []string{"Безопасный материал", "Размер 20 см"}, Keywords: []string{"кошка", "игрушка"}, Attributes: map[string]string{"материал": "хлопок"}, Unit: "шт"}
	if got := ValidateLocalization(valid, rules); !got.Passed {
		t.Fatalf("expected pass, blockers=%+v", got.Blockers)
	}
	valid.Title = "Лучший товар"
	valid.Unit = "件"
	got := ValidateLocalization(valid, rules)
	assertBlocker(t, got, "prohibited_word")
	assertBlocker(t, got, "invalid_unit")
	valid.Title = "Chinese 商品"
	assertBlocker(t, ValidateLocalization(valid, rules), "invalid_locale_script")
}

func TestValidateChannelListing(t *testing.T) {
	rules := ChannelRuleSnapshot{Evidence: ruleEvidence(), PlatformID: 3, CategoryID: "pet-toys", RequiredAttributes: []string{"material", "age"}, RequiredVariantDimensions: []string{"color", "size"}, AllowedVariantDimensions: []string{"color", "size"}, MinImages: 2, MaxImages: 4, MinImageWidth: 1000, MinImageHeight: 1000, AllowedShippingTemplateIDs: []string{"ship-1"}}
	in := ChannelListingInput{PlatformID: 3, CategoryID: "pet-toys", CategorySchemaURI: ruleEvidence().SourceURI, CategoryObservedAt: validatorNow, Attributes: map[string]string{"material": "cotton", "age": "adult"}, VariantDimensions: []string{"color", "size"}, ImageCount: 2, ImageWidths: []int{1200, 1000}, ImageHeights: []int{1200, 1000}, ShippingTemplateID: "ship-1"}
	if got := ValidateChannelListing(in, rules); !got.Passed {
		t.Fatalf("expected pass, blockers=%+v", got.Blockers)
	}
	in.Attributes["age"] = ""
	in.VariantDimensions = []string{"color"}
	in.ImageWidths[1] = 800
	in.ShippingTemplateID = "wrong"
	got := ValidateChannelListing(in, rules)
	for _, code := range []string{"missing_category_attribute", "missing_variant_dimension", "invalid_image_dimensions", "invalid_shipping_template"} {
		assertBlocker(t, got, code)
	}
	in.VariantDimensions = []string{"color", "color", "unsupported"}
	got = ValidateChannelListing(in, rules)
	assertBlocker(t, got, "invalid_variant_dimension")
	assertBlocker(t, got, "unsupported_variant_dimension")
}

func completeCosts() CostValidationInput {
	required := RequiredSourcingCostTypes()
	costs := make([]CostLine, 0, len(required))
	for _, typ := range required {
		costs = append(costs, CostLine{Type: typ, Amount: 1, Currency: "USD", TruthStatus: "quoted", SourceURI: "https://evidence.example/" + typ, ObservedAt: validatorNow})
	}
	return CostValidationInput{TargetCurrency: "USD", Costs: costs, Revenue: RevenueInput{Amount: 20, Currency: "USD", TruthStatus: "estimated", SourceURI: "https://evidence.example/revenue", ObservedAt: validatorNow}}
}

func TestValidateCostsCompleteAndProfit(t *testing.T) {
	got := ValidateCosts(completeCosts())
	if !got.Passed {
		t.Fatalf("expected pass, blockers=%+v", got.Blockers)
	}
	if got.TotalCost != 10 || got.EstimatedRevenue != 20 || got.ContributionProfit != 10 {
		t.Fatalf("unexpected totals: %+v", got)
	}
}

func TestValidateCostsCurrencyConversionAndEvidenceBlockers(t *testing.T) {
	in := completeCosts()
	in.Costs[0].Currency = "CNY"
	in.Costs[0].Amount = 7
	in.ExchangeRates = []ExchangeRate{{FromCurrency: "USD", ToCurrency: "CNY", Rate: 7, TruthStatus: "actual", SourceURI: "https://fx.example/snapshot", ObservedAt: validatorNow}}
	got := ValidateCosts(in)
	if !got.Passed || got.TotalCost != 10 {
		t.Fatalf("inverse conversion should pass: %+v", got)
	}
	in.Costs = in.Costs[:len(in.Costs)-1]
	in.Revenue.TruthStatus = "inferred"
	in.ExchangeRates[0].TruthStatus = "mock"
	got = ValidateCosts(in)
	for _, code := range []string{"missing_cost_type", "untrusted_evidence", "untrusted_exchange_rate"} {
		assertBlocker(t, got.ValidationResult, code)
	}
}

func TestValidateImages(t *testing.T) {
	rules := ImageRuleSnapshot{Evidence: ruleEvidence(), MinMainWidth: 1000, MinMainHeight: 1000, AllowedBackgrounds: []string{"white"}, RequireCrop: true, MinClarityScore: 0.8, MinImages: 1, MaxImages: 3}
	images := []ImageValidationInput{{Role: "main", Width: 1200, Height: 1200, Background: "white", Cropped: true, ClarityScore: .95, TruthStatus: "actual", SourceURI: "sha256:abc", ObservedAt: validatorNow}}
	if got := ValidateImages(images, rules); !got.Passed {
		t.Fatalf("expected pass, blockers=%+v", got.Blockers)
	}
	images[0].Width = 800
	images[0].Background = "busy"
	images[0].Cropped = false
	images[0].ClarityScore = .2
	images[0].HasWatermark, images[0].HasChineseText, images[0].HasBrandMark = true, true, true
	images[0].TruthStatus = "unknown"
	got := ValidateImages(images, rules)
	for _, code := range []string{"invalid_main_dimensions", "invalid_background", "crop_required", "insufficient_clarity", "watermark_present", "chinese_text_present", "brand_mark_present", "untrusted_evidence"} {
		assertBlocker(t, got, code)
	}
}

func TestValidateSKUs(t *testing.T) {
	rules := SKUValidationRules{Evidence: ruleEvidence(), RequireColor: true, RequireSize: true, RequireMaterial: true, RequirePackaging: true}
	skus := []SKUValidationInput{{SupplierSKU: "S-1", InternalSKU: "I-1", ChannelSKU: "C-1", Color: "red", Size: "M", Material: "cotton", Packaging: "box", TruthStatus: "actual", SourceURI: "https://detail.1688.com/offer/1.html", ObservedAt: validatorNow}}
	if got := ValidateSKUs(skus, rules); !got.Passed {
		t.Fatalf("expected pass, blockers=%+v", got.Blockers)
	}
	skus = append(skus, SKUValidationInput{SupplierSKU: "S-1", InternalSKU: "", ChannelSKU: "C-1", TruthStatus: "mock"})
	got := ValidateSKUs(skus, rules)
	for _, code := range []string{"duplicate_supplier_sku", "missing_sku_mapping", "duplicate_channel_sku", "missing_color", "missing_size", "missing_material", "missing_packaging", "untrusted_evidence"} {
		assertBlocker(t, got, code)
	}
}

func TestUntrustedRuleSnapshotBlocksEveryValidator(t *testing.T) {
	bad := ruleEvidence()
	bad.TruthStatus = "inferred"
	if got := ValidateLocalization(LocalizationInput{}, LocalizationRuleSnapshot{Evidence: bad}); !hasBlocker(got.Blockers, "untrusted_evidence") {
		t.Fatal("localization rules should be blocked")
	}
	if got := ValidateChannelListing(ChannelListingInput{}, ChannelRuleSnapshot{Evidence: bad}); !hasBlocker(got.Blockers, "untrusted_evidence") {
		t.Fatal("channel rules should be blocked")
	}
	if got := ValidateImages(nil, ImageRuleSnapshot{Evidence: bad}); !hasBlocker(got.Blockers, "untrusted_evidence") {
		t.Fatal("image rules should be blocked")
	}
	if got := ValidateSKUs(nil, SKUValidationRules{Evidence: bad}); !hasBlocker(got.Blockers, "untrusted_evidence") {
		t.Fatal("sku rules should be blocked")
	}
}

func TestValidateDraftAggregatesStructuredBlockers(t *testing.T) {
	got := ValidateDraft(DraftValidationInput{})
	if got.Passed || len(got.Blockers) < 5 {
		t.Fatalf("expected aggregate blockers, got %+v", got)
	}
	for _, blocker := range got.Blockers {
		if strings.TrimSpace(blocker.Code) == "" || strings.TrimSpace(blocker.Field) == "" || strings.TrimSpace(blocker.Message) == "" {
			t.Fatalf("blocker must be structured: %+v", blocker)
		}
	}
}

func assertBlocker(t *testing.T, got ValidationResult, code string) {
	t.Helper()
	if !hasBlocker(got.Blockers, code) {
		t.Fatalf("expected blocker %q, got %+v", code, got.Blockers)
	}
}

func hasBlocker(blockers []ValidationBlocker, code string) bool {
	for _, blocker := range blockers {
		if blocker.Code == code {
			return true
		}
	}
	return false
}
