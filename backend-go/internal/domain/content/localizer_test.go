package content

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/response"
)

// ==============================================================
// PlatformLocalizer tests
// ==============================================================

func TestNewPlatformLocalizer(t *testing.T) {
	pl := NewPlatformLocalizer()
	if pl == nil {
		t.Fatal("expected non-nil PlatformLocalizer")
	}

	// Verify all expected rules are registered.
	cases := []struct {
		platform, locale string
	}{
		{PlatformShopee, LocalePH},
		{PlatformShopee, LocaleMY},
		{PlatformOzon, LocaleRU},
		{PlatformLazada, LocaleTH},
		{PlatformLazada, LocaleID},
	}

	for _, tc := range cases {
		rule, err := pl.GetRule(tc.platform, tc.locale)
		if err != nil {
			t.Errorf("expected rule for %s/%s, got error: %v", tc.platform, tc.locale, err)
		}
		if rule == nil {
			t.Errorf("expected non-nil rule for %s/%s", tc.platform, tc.locale)
		}
	}

	// Unknown platform should error.
	_, err := pl.GetRule("nonexistent", "XX")
	if err == nil {
		t.Error("expected error for unknown platform/locale")
	}
}

func TestPlatformLocalizer_GetRule(t *testing.T) {
	pl := NewPlatformLocalizer()

	rule, err := pl.GetRule(PlatformShopee, LocalePH)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rule.platform != PlatformShopee {
		t.Errorf("expected platform shopee, got %s", rule.platform)
	}
	if rule.maxTitleLen != 120 {
		t.Errorf("expected maxTitleLen 120, got %d", rule.maxTitleLen)
	}
	if len(rule.forbiddenTerms) == 0 {
		t.Error("expected non-empty forbidden terms for Shopee PH")
	}
	if len(rule.fieldMappings) == 0 {
		t.Error("expected non-empty field mappings for Shopee PH")
	}

	// Ozon RU
	rule, err = pl.GetRule(PlatformOzon, LocaleRU)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.maxTitleLen != 100 {
		t.Errorf("expected maxTitleLen 100 for Ozon RU, got %d", rule.maxTitleLen)
	}
	if rule.titleTemplate == "" {
		t.Error("expected non-empty title template for Ozon RU")
	}
	hasRussianForbidden := false
	for _, t := range rule.forbiddenTerms {
		if t == "лучший" {
			hasRussianForbidden = true
			break
		}
	}
	if !hasRussianForbidden {
		t.Error("expected Russian forbidden terms for Ozon RU")
	}

	// Lazada TH
	rule, err = pl.GetRule(PlatformLazada, LocaleTH)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.maxTitleLen != 200 {
		t.Errorf("expected maxTitleLen 200 for Lazada TH, got %d", rule.maxTitleLen)
	}
	if len(rule.fieldMappings) == 0 {
		t.Error("expected non-empty field mappings for Lazada TH")
	}
}

// ==============================================================
// Forbidden Term Filter tests
// ==============================================================

func TestFilterForbiddenTerms(t *testing.T) {
	forbidden := []string{"best", "100%", "guaranteed"}

	// No forbidden terms.
	filtered, removed := filterForbiddenTerms("High quality product", forbidden)
	if len(removed) != 0 {
		t.Errorf("expected 0 removed, got %d: %v", len(removed), removed)
	}
	if filtered != "High quality product" {
		t.Errorf("expected unchanged text, got %q", filtered)
	}

	// Single forbidden term.
	filtered, removed = filterForbiddenTerms("This is the best product ever", forbidden)
	if len(removed) != 1 || removed[0] != "best" {
		t.Errorf("expected 1 removed term 'best', got %v", removed)
	}
	if strings.Contains(strings.ToLower(filtered), "best") {
		t.Errorf("filtered text should not contain 'best', got: %q", filtered)
	}

	// Multiple forbidden terms.
	filtered, removed = filterForbiddenTerms("Best 100% guaranteed product", forbidden)
	if len(removed) != 3 {
		t.Errorf("expected 3 removed terms, got %d: %v", len(removed), removed)
	}

	// Empty text.
	filtered, removed = filterForbiddenTerms("", forbidden)
	if filtered != "" || len(removed) != 0 {
		t.Errorf("expected empty result for empty input")
	}

	// Empty forbidden list.
	filtered, removed = filterForbiddenTerms("some text", nil)
	if filtered != "some text" || len(removed) != 0 {
		t.Errorf("expected unchanged text with nil forbidden list")
	}
}

// ==============================================================
// LocalizeContent tests
// ==============================================================

func TestLocalizeContent_ShopeePH(t *testing.T) {
	pl := NewPlatformLocalizer()
	input := &LocalizeInput{
		Title:          "Wireless Bluetooth Earbuds with Charging Case",
		Description:    "High quality wireless earbuds with noise cancellation and 24hr battery life",
		Keywords:       []string{"earbuds", "wireless", "audio"},
		SourceLanguage: "en",
		TargetPlatform: PlatformShopee,
		TargetLocale:   LocalePH,
		Category:       "electronics",
		Brand:          "SoundPro",
		Specifications: []string{"Bluetooth 5.3, 24hr battery, IPX5 waterproof"},
	}

	result, err := LocalizeContent(input, pl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Platform != PlatformShopee {
		t.Errorf("expected platform shopee, got %s", result.Platform)
	}
	if result.Locale != LocalePH {
		t.Errorf("expected locale PH, got %s", result.Locale)
	}

	// Title should not exceed 120 chars.
	if len([]rune(result.Title)) > 120 {
		t.Errorf("title exceeds 120 chars: %d", len([]rune(result.Title)))
	}

	// Title should contain brand.
	if !strings.Contains(result.Title, "SoundPro") {
		t.Errorf("title should contain brand, got: %s", result.Title)
	}

	// Title should contain product name.
	if !strings.Contains(result.Title, "Wireless Bluetooth Earbuds") {
		t.Errorf("title should contain product name, got: %s", result.Title)
	}

	// Should have applied rules.
	if len(result.AppliedRules) == 0 {
		t.Error("expected at least one applied rule")
	}

	// Should have field mappings.
	if len(result.FieldMapping) == 0 {
		t.Error("expected field mappings")
	}

	// Should have keywords.
	if len(result.Keywords) == 0 {
		t.Error("expected keywords")
	}
}

func TestLocalizeContent_OzonRU(t *testing.T) {
	pl := NewPlatformLocalizer()
	input := &LocalizeInput{
		Title:          "Wireless Bluetooth Earbuds with Charging Case",
		Description:    "High quality wireless earbuds with noise cancellation and 24hr battery life",
		Keywords:       []string{"earbuds", "wireless", "audio"},
		SourceLanguage: "en",
		TargetPlatform: PlatformOzon,
		TargetLocale:   LocaleRU,
		Category:       "electronics",
		Brand:          "SoundPro",
		Specifications: []string{"Bluetooth 5.3, 24hr battery, IPX5 waterproof"},
	}

	result, err := LocalizeContent(input, pl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Title should not exceed 100 chars (Ozon RU limit).
	if len([]rune(result.Title)) > 100 {
		t.Errorf("title exceeds 100 chars: %d", len([]rune(result.Title)))
	}

	// Should have Ozon-specific field mapping (characteristics).
	if attr, ok := result.FieldMapping["attributes"]; !ok || attr != "characteristics" {
		t.Errorf("expected characteristics field mapping, got %v", result.FieldMapping)
	}

	if result.Locale != LocaleRU {
		t.Errorf("expected locale RU, got %s", result.Locale)
	}
}

func TestLocalizeContent_LazadaTH(t *testing.T) {
	pl := NewPlatformLocalizer()
	input := &LocalizeInput{
		Title:          "Wireless Bluetooth Earbuds with Charging Case",
		Description:    "High quality wireless earbuds with noise cancellation",
		Keywords:       []string{"earbuds", "wireless"},
		SourceLanguage: "en",
		TargetPlatform: PlatformLazada,
		TargetLocale:   LocaleTH,
		Category:       "electronics",
		Brand:          "SoundPro",
		Specifications: []string{"Bluetooth 5.3, 24hr battery"},
	}

	result, err := LocalizeContent(input, pl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Title should not exceed 200 chars (Lazada TH limit).
	if len([]rune(result.Title)) > 200 {
		t.Errorf("title exceeds 200 chars: %d", len([]rune(result.Title)))
	}

	if result.Platform != PlatformLazada {
		t.Errorf("expected platform lazada, got %s", result.Platform)
	}
	if result.Locale != LocaleTH {
		t.Errorf("expected locale TH, got %s", result.Locale)
	}

	// Description should include brand.
	if !strings.Contains(result.Description, "SoundPro") {
		t.Errorf("description should contain brand, got: %s", result.Description)
	}
}

func TestLocalizeContent_ForbiddenTermFiltering(t *testing.T) {
	pl := NewPlatformLocalizer()

	testCases := []struct {
		name     string
		platform string
		locale   string
		title    string
	}{
		{
			name:     "Shopee PH filters 'guaranteed' and '100%'",
			platform: PlatformShopee,
			locale:   LocalePH,
			title:    "100% guaranteed authentic product",
		},
		{
			name:     "Ozon RU filters Russian forbidden terms",
			platform: PlatformOzon,
			locale:   LocaleRU,
			title:    "лучший товар по лучшей цене",
		},
		{
			name:     "Lazada TH filters 'ดีที่สุด'",
			platform: PlatformLazada,
			locale:   LocaleTH,
			title:    "ดีที่สุด guaranteed product",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := &LocalizeInput{
				Title:          tc.title,
				Description:    "product description content",
				Keywords:       []string{"test"},
				SourceLanguage: "en",
				TargetPlatform: tc.platform,
				TargetLocale:   tc.locale,
				Brand:          "TestBrand",
			}

			result, err := LocalizeContent(input, pl)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.FilteredTerms) == 0 {
				t.Errorf("expected filtered terms for title: %q", tc.title)
			}

			hasForbiddenRule := false
			for _, r := range result.AppliedRules {
				if r == "forbidden_term_filter" {
					hasForbiddenRule = true
					break
				}
			}
			if !hasForbiddenRule {
				t.Error("expected forbidden_term_filter in applied rules")
			}
		})
	}
}

func TestLocalizeContent_ErrorUnsupported(t *testing.T) {
	pl := NewPlatformLocalizer()
	input := &LocalizeInput{
		Title:          "Test Product",
		SourceLanguage: "en",
		TargetPlatform: "unknown_platform",
		TargetLocale:   "XX",
	}

	_, err := LocalizeContent(input, pl)
	if err == nil {
		t.Error("expected error for unsupported platform/locale")
	}
}

// ==============================================================
// SEOKeywordOptimizer tests
// ==============================================================

func TestSEOKeywordOptimizer_Shopee(t *testing.T) {
	pl := NewPlatformLocalizer()
	rule, _ := pl.GetRule(PlatformShopee, LocalePH)
	optimizer := NewSEOKeywordOptimizer()

	input := &LocalizeInput{
		Keywords:       []string{"earbuds", "wireless", "best", "cheap"},
		Category:       "electronics",
		Brand:          "SoundPro",
		TargetPlatform: PlatformShopee,
		TargetLocale:   LocalePH,
	}

	keywords, rules := optimizer.Optimize(input, rule)

	if len(keywords) == 0 {
		t.Error("expected non-empty keywords")
	}

	// "best" and "cheap" should be filtered out (forbidden terms).
	for _, kw := range keywords {
		if kw == "best" || kw == "cheap" {
			t.Errorf("forbidden term '%s' should not appear in keywords", kw)
		}
	}

	// Brand should appear.
	hasBrand := false
	for _, kw := range keywords {
		if kw == "SoundPro" {
			hasBrand = true
			break
		}
	}
	if !hasBrand {
		t.Error("expected brand 'SoundPro' in keywords")
	}

	if len(rules) == 0 {
		t.Error("expected non-empty optimization rules")
	}
}

func TestSEOKeywordOptimizer_Ozon(t *testing.T) {
	pl := NewPlatformLocalizer()
	rule, _ := pl.GetRule(PlatformOzon, LocaleRU)
	optimizer := NewSEOKeywordOptimizer()

	input := &LocalizeInput{
		Keywords:       []string{"купить", "наушники"},
		Category:       "electronics",
		Brand:          "SoundPro",
		TargetPlatform: PlatformOzon,
		TargetLocale:   LocaleRU,
	}

	keywords, rules := optimizer.Optimize(input, rule)

	if len(keywords) == 0 {
		t.Error("expected non-empty keywords")
	}

	if len(rules) == 0 {
		t.Error("expected non-empty optimization rules")
	}

	// Ozon supports up to 15 keywords.
	if len(keywords) > 15 {
		t.Errorf("expected <= 15 keywords for Ozon, got %d", len(keywords))
	}
}

func TestSEOKeywordOptimizer_Deduplication(t *testing.T) {
	pl := NewPlatformLocalizer()
	rule, _ := pl.GetRule(PlatformShopee, LocalePH)
	optimizer := NewSEOKeywordOptimizer()

	input := &LocalizeInput{
		Keywords:       []string{"earbuds", "earbuds", "wireless", "WIRELESS"},
		Category:       "electronics",
		Brand:          "SoundPro",
		TargetPlatform: PlatformShopee,
		TargetLocale:   LocalePH,
	}

	keywords, _ := optimizer.Optimize(input, rule)

	// Count occurrences of "earbuds" (case-insensitive).
	count := 0
	for _, kw := range keywords {
		if strings.ToLower(kw) == "earbuds" {
			count++
		}
	}
	if count > 1 {
		t.Errorf("expected 'earbuds' deduplicated, found %d occurrences", count)
	}

	// Count occurrences of "wireless" (case-insensitive).
	count = 0
	for _, kw := range keywords {
		if strings.ToLower(kw) == "wireless" {
			count++
		}
	}
	if count > 1 {
		t.Errorf("expected 'wireless' deduplicated, found %d occurrences", count)
	}
}

func TestSEOKeywordOptimizer_StandaloneWrapper(t *testing.T) {
	pl := NewPlatformLocalizer()
	optimizer := NewSEOKeywordOptimizer()

	input := &LocalizeInput{
		Keywords:       []string{"test"},
		Category:       "fashion",
		Brand:          "Nike",
		TargetPlatform: PlatformShopee,
		TargetLocale:   LocalePH,
	}

	keywords, err := optimizer.SEOOptimizeContent(input, pl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(keywords) == 0 {
		t.Error("expected non-empty keywords from standalone wrapper")
	}
}

// ==============================================================
// Truncation and edge case tests
// ==============================================================

func TestTruncateByRunes(t *testing.T) {
	cases := []struct {
		input  string
		maxLen int
		expect string
	}{
		{"hello", 10, "hello"},
		{"hello", 3, "hel"},
		{"", 5, ""},
		{"世界hello", 4, "世界he"},
		{"hello", 0, ""},
	}

	for _, tc := range cases {
		result := truncateByRunes(tc.input, tc.maxLen)
		if result != tc.expect {
			t.Errorf("truncateByRunes(%q, %d) = %q, want %q", tc.input, tc.maxLen, result, tc.expect)
		}
	}
}

func TestExtractKeyFeatures(t *testing.T) {
	cases := []struct {
		input  string
		maxLen int // implicit: we check it's at most 53 (50 chars + "...")
	}{
		{"", 0},
		{"short", 5},
		{"a very long specification string that exceeds fifty characters in display length", 53},
	}

	for _, tc := range cases {
		result := extractKeyFeatures(tc.input)
		if len([]rune(result)) > 53 {
			t.Errorf("extractKeyFeatures result too long: %d chars: %q", len([]rune(result)), result)
		}
		if tc.input == "" && result != "" {
			t.Errorf("expected empty result for empty input")
		}
	}
}

func TestDedupe(t *testing.T) {
	cases := []struct {
		input  []string
		expect []string
	}{
		{[]string{"a", "b", "a", "c"}, []string{"a", "b", "c"}},
		{[]string{"A", "a"}, []string{"A"}},
		{[]string{}, []string{}},
		{nil, nil},
	}

	for _, tc := range cases {
		result := dedupe(tc.input)
		if len(result) != len(tc.expect) {
			t.Errorf("dedupe(%v) = %v, want %v", tc.input, result, tc.expect)
			continue
		}
		for i := range result {
			if tc.expect[i] != "" && strings.ToLower(result[i]) != strings.ToLower(tc.expect[i]) {
				t.Errorf("dedupe(%v) = %v, want %v", tc.input, result, tc.expect)
				break
			}
		}
	}
}

// ==============================================================
// Handler test (HTTP integration)
// ==============================================================

func TestHandler_LocalizeContent(t *testing.T) {
	// Setup Gin test mode
	gin.SetMode(gin.TestMode)

	h := &Handler{service: nil}

	t.Run("valid localization request", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{
			"title": "Wireless Bluetooth Earbuds",
			"description": "High quality earbuds",
			"source_language": "en",
			"target_platform": "shopee",
			"target_locale": "PH",
			"category": "electronics",
			"brand": "SoundPro"
		}`
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/content/localize", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		h.LocalizeContent(c)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp response.Result
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if resp.Code != 0 {
			t.Errorf("expected code 0, got %d: %s", resp.Code, resp.Message)
		}
	})

	t.Run("unsupported platform returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{
			"title": "Test",
			"source_language": "en",
			"target_platform": "nonexistent",
			"target_locale": "XX"
		}`
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/content/localize", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		h.LocalizeContent(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for unsupported platform, got %d", w.Code)
		}
	})

	t.Run("missing required fields returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := `{}`
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/content/localize", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")

		h.LocalizeContent(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for missing fields, got %d", w.Code)
		}
	})
}

// ==============================================================
// Integration test: full localization pipeline
// ==============================================================

func TestFullLocalizationPipeline(t *testing.T) {
	pl := NewPlatformLocalizer()

	testCases := []struct {
		name     string
		platform string
		locale   string
	}{
		{"Shopee PH", PlatformShopee, LocalePH},
		{"Ozon RU", PlatformOzon, LocaleRU},
		{"Lazada TH", PlatformLazada, LocaleTH},
		{"Lazada ID", PlatformLazada, LocaleID},
		{"Shopee MY", PlatformShopee, LocaleMY},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := &LocalizeInput{
				Title:          "Premium Wireless Bluetooth Earbuds Noise Cancelling",
				Description:    "Experience premium sound quality with our latest wireless earbuds featuring active noise cancellation, 24-hour battery life, and comfortable ergonomic design.",
				Keywords:       []string{"earbuds", "wireless", "bluetooth", "noise cancelling", "audio"},
				SourceLanguage: "en",
				TargetPlatform: tc.platform,
				TargetLocale:   tc.locale,
				Category:       "electronics",
				Brand:          "SoundPro",
				Specifications: []string{"Bluetooth 5.3, 24hr battery, IPX5 waterproof, USB-C charging, 10mm drivers"},
			}

			result, err := LocalizeContent(input, pl)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify basic invariants.
			if result.Platform != tc.platform {
				t.Errorf("platform mismatch: got %s, want %s", result.Platform, tc.platform)
			}
			if result.Locale != tc.locale {
				t.Errorf("locale mismatch: got %s, want %s", result.Locale, tc.locale)
			}
			if result.Title == "" {
				t.Error("title should not be empty")
			}
			if result.Description == "" {
				t.Error("description should not be empty")
			}
			if len(result.AppliedRules) < 2 {
				t.Errorf("expected at least 2 applied rules, got %d", len(result.AppliedRules))
			}
			if result.FieldMapping == nil {
				t.Error("field mapping should not be nil")
			}
		})
	}
}
