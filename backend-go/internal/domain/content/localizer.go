package content

import (
	"fmt"
	"strings"
)

// Platform constants
const (
	PlatformShopee      = "shopee"
	PlatformOzon        = "ozon"
	PlatformLazada      = "lazada"
	PlatformWildberries = "wb"
)

// Locale constants
const (
	LocalePH = "PH"
	LocaleRU = "RU"
	LocaleTH = "TH"
	LocaleID = "ID"
	LocaleMY = "MY"
	LocaleVN = "VN"
)

// localizeRule holds platform+locale-specific localization rules.
type localizeRule struct {
	platform       string
	locale         string
	maxTitleLen    int
	maxDescLen     int
	titleTemplate  string
	descTemplate   string
	forbiddenTerms []string
	seoKeywords    map[string][]string // category -> recommended keywords
	fieldMappings  map[string]string   // generic field -> platform field name
}

// PlatformLocalizer manages localization rules for all supported platforms.
type PlatformLocalizer struct {
	rules map[string]localizeRule // key: "{platform}:{locale}"
}

// NewPlatformLocalizer creates a PlatformLocalizer with built-in rules.
func NewPlatformLocalizer() *PlatformLocalizer {
	pl := &PlatformLocalizer{rules: make(map[string]localizeRule)}
	pl.registerBuiltinRules()
	return pl
}

// registerBuiltinRules populates the built-in platform+locale localization rules.
func (pl *PlatformLocalizer) registerBuiltinRules() {
	// ---- Shopee PH (Philippines) ----
	// English title with Tagalog keywords (hybrid). Max 120 chars.
	pl.rules[key(PlatformShopee, LocalePH)] = localizeRule{
		platform:      PlatformShopee,
		locale:        LocalePH,
		maxTitleLen:   120,
		maxDescLen:    2000,
		titleTemplate: "{brand} {product_name} - {key_features} | {keywords}",
		descTemplate:  "{description}\n\n{specifications}\n\nBrand New | Authentic | Free Shipping within Metro Manila",
		forbiddenTerms: []string{
			"best", "cheap", "guaranteed", "100%", "#1", "cheapest", "free", "perfect",
			"unbeatable", "miracle", "cure", "instant", "magic",
		},
		seoKeywords: map[string][]string{
			"default":     {"brand new", "authentic", "free shipping", "sale", "mura", "quality", "original"},
			"electronics": {"brand new", "authentic", "free shipping", "sulit", "quality", "original", "warranty"},
			"fashion":     {"brand new", "authentic", "free shipping", "mura", "trending", "stylish", "quality"},
			"home":        {"brand new", "durable", "quality", "sulit", "free shipping"},
		},
		fieldMappings: map[string]string{
			"brand":       "brand",
			"category":    "category_id",
			"description": "description",
			"images":      "images",
			"variations":  "variations",
		},
	}

	// ---- Ozon RU (Russia) ----
	// Full Russian localization. Max 100 chars title. Attribute-based structure.
	pl.rules[key(PlatformOzon, LocaleRU)] = localizeRule{
		platform:      PlatformOzon,
		locale:        LocaleRU,
		maxTitleLen:   100,
		maxDescLen:    3000,
		titleTemplate: "{brand} {product_name}, {key_attributes}",
		descTemplate:  "Характеристики:\n{specifications}\n\nОписание:\n{description}\n\nПроизводитель: {brand}\nСтрана производства: Китай",
		forbiddenTerms: []string{
			"лучший", "best", "дешевый", "дешёвый", "бесплатный",
			"№1", "100%", "гарантированный", "супер", "хит",
			"эксклюзивный", "уникальный", "невероятный",
		},
		seoKeywords: map[string][]string{
			"default":     {"купить", "цена", "доставка", "оригинал", "качество"},
			"electronics": {"купить", "цена", "доставка", "оригинал", "гарантия", "техника"},
			"clothing":    {"купить", "цена", "доставка", "оригинал", "размер", "ткань"},
			"home":        {"купить", "цена", "доставка", "качество", "материал"},
		},
		fieldMappings: map[string]string{
			"brand":       "brand",
			"category":    "category_id",
			"description": "description",
			"images":      "images",
			"attributes":  "characteristics",
			"variations":  "variations",
			"barcode":     "barcode",
		},
	}

	// ---- Lazada TH (Thailand) ----
	// Thai language localization. Max 200 chars. Category-specific attributes.
	pl.rules[key(PlatformLazada, LocaleTH)] = localizeRule{
		platform:      PlatformLazada,
		locale:        LocaleTH,
		maxTitleLen:   200,
		maxDescLen:    2500,
		titleTemplate: "{brand}{product_name} {key_features} {keywords}",
		descTemplate:  "{description}\n\nคุณสมบัติ:\n{specifications}\n\nแบรนด์: {brand}\n\nจัดส่งด่วน | สินค้าพร้อมส่ง",
		forbiddenTerms: []string{
			"ดีที่สุด", "รับประกัน", "100%", "#1", "ถูกที่สุด",
			"miraculous", "cure", "instant",
			"best", "cheapest", "guaranteed",
		},
		seoKeywords: map[string][]string{
			"default":     {"จัดส่งด่วน", "สินค้าพร้อมส่ง", "แบรนด์แท้", "ราคาพิเศษ", "คุณภาพดี"},
			"electronics": {"จัดส่งด่วน", "ของแท้", "รับประกัน", "ราคาพิเศษ", "แบรนด์เนม"},
			"fashion":     {"จัดส่งด่วน", "ของแท้", "สวย", "เก๋", "คุณภาพดี", "ราคาถูก"},
			"home":        {"จัดส่งด่วน", "คุณภาพดี", "ทนทาน", "ของแท้", "ราคาพิเศษ"},
		},
		fieldMappings: map[string]string{
			"brand":       "brand",
			"category":    "primary_category",
			"description": "description",
			"images":      "images",
			"attributes":  "product_attributes",
			"variations":  "sku_list",
		},
	}

	// ---- Lazada ID (Indonesia) ----
	pl.rules[key(PlatformLazada, LocaleID)] = localizeRule{
		platform:      PlatformLazada,
		locale:        LocaleID,
		maxTitleLen:   200,
		maxDescLen:    2000,
		titleTemplate: "{brand} {product_name} - {key_features} | {keywords}",
		descTemplate:  "{description}\n\nSpesifikasi:\n{specifications}\n\nMerk: {brand}\n\nGratis Ongkir | Barang Original",
		forbiddenTerms: []string{
			"terbaik", "dijamin", "100%", "#1", "termurah",
			"best", "cheapest", "guaranteed",
		},
		seoKeywords: map[string][]string{
			"default": {"gratis ongkir", "barang original", "berkualitas", "diskon", "murah"},
		},
		fieldMappings: map[string]string{
			"brand":       "brand",
			"category":    "primary_category",
			"description": "description",
			"images":      "images",
		},
	}

	// ---- Shopee MY (Malaysia) ----
	pl.rules[key(PlatformShopee, LocaleMY)] = localizeRule{
		platform:      PlatformShopee,
		locale:        LocaleMY,
		maxTitleLen:   120,
		maxDescLen:    2000,
		titleTemplate: "{brand} {product_name} {key_features} | {keywords}",
		descTemplate:  "{description}\n\n{specifications}\n\nBrand New | Authentic | Free Shipping",
		forbiddenTerms: []string{
			"guaranteed", "100%", "#1", "cheapest", "free", "perfect",
			"miracle", "cure", "best",
		},
		seoKeywords: map[string][]string{
			"default": {"brand new", "authentic", "free shipping", "sale", "murah", "quality"},
		},
		fieldMappings: map[string]string{
			"brand":    "brand",
			"category": "category_id",
		},
	}
}

// key builds the composite map key for platform:locale.
func key(platform, locale string) string {
	return strings.ToLower(platform) + ":" + strings.ToUpper(locale)
}

// GetRule returns the localization rule for a given platform and locale.
// Returns nil and an error if no rule is found.
func (pl *PlatformLocalizer) GetRule(platform, locale string) (*localizeRule, error) {
	r, ok := pl.rules[key(platform, locale)]
	if !ok {
		return nil, fmt.Errorf("no localization rule for %s/%s", platform, locale)
	}
	return &r, nil
}

// LocalizeContent applies platform+locale specific localization rules to input content.
// It does NOT call any AI API -- purely rule-based transformation.
func LocalizeContent(input *LocalizeInput, localizer *PlatformLocalizer) (*LocalizedContent, error) {
	rule, err := localizer.GetRule(input.TargetPlatform, input.TargetLocale)
	if err != nil {
		return nil, err
	}

	result := &LocalizedContent{
		Platform:     input.TargetPlatform,
		Locale:       input.TargetLocale,
		AppliedRules: []string{},
		Keywords:     make([]string, 0),
	}

	// 1. Filter forbidden terms from source content.
	filteredTitle, titleFiltered := filterForbiddenTerms(input.Title, rule.forbiddenTerms)
	filteredDesc, descFiltered := filterForbiddenTerms(input.Description, rule.forbiddenTerms)
	result.FilteredTerms = append(result.FilteredTerms, titleFiltered...)
	result.FilteredTerms = append(result.FilteredTerms, descFiltered...)
	if len(result.FilteredTerms) > 0 {
		result.AppliedRules = append(result.AppliedRules, "forbidden_term_filter")
	}

	// 2. Build title using the platform-specific template.
	result.Title = buildLocalizedTitle(filteredTitle, input, rule)
	if len([]rune(result.Title)) > rule.maxTitleLen {
		result.Title = truncateByRunes(result.Title, rule.maxTitleLen)
		result.AppliedRules = append(result.AppliedRules, "title_truncation")
	}
	result.AppliedRules = append(result.AppliedRules, "title_template")

	// 3. Build description using the platform-specific template.
	result.Description = buildLocalizedDescription(filteredDesc, input, rule)
	if len([]rune(result.Description)) > rule.maxDescLen {
		result.Description = truncateByRunes(result.Description, rule.maxDescLen)
		result.AppliedRules = append(result.AppliedRules, "description_truncation")
	}
	result.AppliedRules = append(result.AppliedRules, "description_template")

	// 4. Map SEO keywords by category.
	seoOpt := NewSEOKeywordOptimizer()
	seoKeywords, seoRules := seoOpt.Optimize(input, rule)
	result.Keywords = append(result.Keywords, seoKeywords...)
	result.AppliedRules = append(result.AppliedRules, seoRules...)

	// 5. Map generic field names to platform-specific field names.
	result.FieldMapping = rule.fieldMappings
	result.AppliedRules = append(result.AppliedRules, "field_mapping")

	return result, nil
}

// buildLocalizedTitle constructs the title following the platform-specific template.
func buildLocalizedTitle(sourceTitle string, input *LocalizeInput, rule *localizeRule) string {
	t := rule.titleTemplate
	t = strings.ReplaceAll(t, "{brand}", input.Brand)
	t = strings.ReplaceAll(t, "{product_name}", sourceTitle)

	// Extract key features from specifications (first 50 chars).
	keyFeatures := extractKeyFeatures(strings.Join(input.Specifications, ", "))
	t = strings.ReplaceAll(t, "{key_features}", keyFeatures)

	// Add relevant SEO keywords.
	kw := pickCategoryKeywords(input.Category, rule.seoKeywords, 2)
	t = strings.ReplaceAll(t, "{keywords}", strings.Join(kw, " "))

	// Clean up double spaces and trim.
	t = strings.Join(strings.Fields(t), " ")
	return t
}

// buildLocalizedDescription constructs the description following the platform-specific template.
func buildLocalizedDescription(sourceDesc string, input *LocalizeInput, rule *localizeRule) string {
	t := rule.descTemplate
	t = strings.ReplaceAll(t, "{description}", sourceDesc)
	t = strings.ReplaceAll(t, "{specifications}", strings.Join(input.Specifications, ", "))
	t = strings.ReplaceAll(t, "{brand}", input.Brand)

	// Clean up double spaces and trim.
	t = strings.Join(strings.Fields(t), " ")
	return t
}

// filterForbiddenTerms removes platform-specific forbidden terms from text and
// returns the filtered text and a list of removed terms.
func filterForbiddenTerms(text string, forbidden []string) (string, []string) {
	if text == "" || len(forbidden) == 0 {
		return text, nil
	}

	lower := strings.ToLower(text)
	var removed []string
	filtered := text

	for _, term := range forbidden {
		lowerTerm := strings.ToLower(term)
		if strings.Contains(lower, lowerTerm) {
			removed = append(removed, term)
			// Case-insensitive replacement using multiple case variants.
			filtered = strings.ReplaceAll(filtered, term, "")
			filtered = strings.ReplaceAll(filtered, lowerTerm, "")
			filtered = strings.ReplaceAll(filtered, strings.ToUpper(term), "")
			if len(term) > 0 {
				cap := strings.ToUpper(string(term[0])) + term[1:]
				filtered = strings.ReplaceAll(filtered, cap, "")
			}
		}
	}

	// Clean up double spaces from removal.
	filtered = strings.Join(strings.Fields(filtered), " ")
	return filtered, removed
}

// extractKeyFeatures extracts the first meaningful segment from specifications.
func extractKeyFeatures(specs string) string {
	if specs == "" {
		return ""
	}
	// Take up to first 50 chars that are meaningful.
	runes := []rune(specs)
	if len(runes) > 50 {
		return string(runes[:50]) + "..."
	}
	return specs
}

// pickCategoryKeywords returns up to n keywords for the given category.
func pickCategoryKeywords(category string, km map[string][]string, n int) []string {
	keywords, ok := km[strings.ToLower(category)]
	if !ok || len(keywords) == 0 {
		keywords = km["default"]
	}
	if keywords == nil {
		return nil
	}
	if len(keywords) > n {
		return keywords[:n]
	}
	return keywords
}

// truncateByRunes truncates a string to at most maxLen runes.
func truncateByRunes(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

// ==============================================================
// SEOKeywordOptimizer -- platform-specific keyword optimization
// ==============================================================

// SEOKeywordOptimizer optimizes content keywords based on platform search algorithms.
type SEOKeywordOptimizer struct{}

// NewSEOKeywordOptimizer creates a new SEO keyword optimizer.
func NewSEOKeywordOptimizer() *SEOKeywordOptimizer {
	return &SEOKeywordOptimizer{}
}

// Optimize returns SEO-optimized keywords for the given input and platform rule.
// Returns the keyword list and the list of applied optimization rules.
func (s *SEOKeywordOptimizer) Optimize(input *LocalizeInput, rule *localizeRule) ([]string, []string) {
	var keywords []string
	var rulesApplied []string

	// 1. Start with category-specific SEO keywords.
	catKeywords := pickCategoryKeywords(input.Category, rule.seoKeywords, 3)
	keywords = append(keywords, catKeywords...)
	rulesApplied = append(rulesApplied, "category_seo_keywords")

	// 2. Add input keywords (filtering forbidden terms).
	if len(input.Keywords) > 0 {
		for _, kw := range input.Keywords {
			clean, _ := filterForbiddenTerms(kw, rule.forbiddenTerms)
			if clean != "" {
				keywords = append(keywords, clean)
			}
		}
		rulesApplied = append(rulesApplied, "input_keyword_passthrough")
	}

	// 3. Add brand as keyword if present.
	if input.Brand != "" {
		keywords = append(keywords, input.Brand)
		rulesApplied = append(rulesApplied, "brand_keyword")
	}

	// Remove duplicates while preserving order.
	keywords = dedupe(keywords)

	// 4. Cap keyword count per platform best practices.
	maxKw := maxKeywords(rule.platform)
	if len(keywords) > maxKw {
		keywords = keywords[:maxKw]
		rulesApplied = append(rulesApplied, "keyword_count_cap")
	}

	return keywords, rulesApplied
}

// SEOOptimizeContent runs the SEO optimizer directly against a LocalizeInput.
// Convenience wrapper for standalone usage.
func (s *SEOKeywordOptimizer) SEOOptimizeContent(input *LocalizeInput, localizer *PlatformLocalizer) ([]string, error) {
	rule, err := localizer.GetRule(input.TargetPlatform, input.TargetLocale)
	if err != nil {
		return nil, err
	}
	kws, _ := s.Optimize(input, rule)
	return kws, nil
}

// maxKeywords returns the recommended max keyword count per platform.
func maxKeywords(platform string) int {
	switch strings.ToLower(platform) {
	case PlatformShopee:
		return 10
	case PlatformOzon:
		return 15
	case PlatformLazada:
		return 12
	case PlatformWildberries:
		return 10
	default:
		return 10
	}
}

// dedupe removes duplicate strings while preserving order.
func dedupe(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		lower := strings.ToLower(item)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		result = append(result, item)
	}
	return result
}
