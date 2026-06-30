package content

// GenerateRequest is the input for AI content generation.
type GenerateRequest struct {
	ProductName    string `json:"product_name" binding:"required"`
	Category       string `json:"category" binding:"required"`
	Brand          string `json:"brand"`
	Specifications string `json:"specifications"`
	TargetLanguage string `json:"target_language" binding:"required"` // zh, en, ru
	Platform       string `json:"platform" binding:"required"`        // ozon, shopee, wb
}

// GeneratedContent is the output of AI content generation.
type GeneratedContent struct {
	Title       string   `json:"title"`
	Subtitle    string   `json:"subtitle,omitempty"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
	Confidence  float64  `json:"confidence"` // 0.0-1.0
}

// ValidateRequest is the input for content validation.
type ValidateRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Language    string `json:"language" binding:"required"`
	Platform    string `json:"platform" binding:"required"`
}


// LocalizeInput is the request payload for the content localization endpoint.
type LocalizeInput struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	Keywords        []string `json:"keywords"`
	SourceLanguage  string `json:"source_language"`
	TargetPlatform  string `json:"target_platform"`
	TargetLocale    string `json:"target_locale"`
	Category        string `json:"category"`
	Brand           string `json:"brand"`
	Specifications  []string `json:"specifications"`
}

// LocalizedContent is the response payload for the content localization endpoint.
type LocalizedContent struct {
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Keywords        []string `json:"keywords"`
	Platform        string   `json:"platform"`
	Locale          string   `json:"locale"`
	AppliedRules    []string `json:"applied_rules"`
	FilteredTerms   []string `json:"filtered_terms"`
	FieldMapping    map[string]string `json:"field_mapping"`
}
