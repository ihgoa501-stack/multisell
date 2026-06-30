package content

import (
	"encoding/json"
	"fmt"

	"github.com/lingmirror/backend-go/internal/ai"
	"go.uber.org/zap"
)

// ContentGenerator creates product content using AI orchestration.
type ContentGenerator struct {
	aiOrch *ai.Orchestrator
	logger *zap.Logger
}

// NewContentGenerator creates a new ContentGenerator.
func NewContentGenerator(aiOrch *ai.Orchestrator, logger *zap.Logger) *ContentGenerator {
	return &ContentGenerator{aiOrch: aiOrch, logger: logger}
}

// GenerateInput holds the raw product data for content generation.
type GenerateInput struct {
	ProductName    string `json:"product_name"`
	Category       string `json:"category"`
	Brand          string `json:"brand"`
	Specifications string `json:"specifications"`
	TargetLanguage string `json:"target_language"` // zh, en, ru
	Platform       string `json:"platform"`        // ozon, shopee, wb
}

// Generate creates product content using AI orchestration.
func (g *ContentGenerator) Generate(in *GenerateRequest) (*GeneratedContent, error) {
	// Build prompt based on target language and platform
	prompt := g.buildPrompt(in)

	// Call AI orchestrator to generate content
	resp, err := g.aiOrch.Run(&ai.RunAgentRequest{
		AgentID:       "content_ai",
		DecisionPoint: "generate_content",
		Context: map[string]interface{}{
			"prompt":          prompt,
			"product_name":    in.ProductName,
			"target_language": in.TargetLanguage,
			"platform":        in.Platform,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("content generation failed: %w", err)
	}

	// Parse AI response from the Output map.
	var content GeneratedContent
	raw, _ := json.Marshal(resp.Output)
	json.Unmarshal(raw, &content)

	// Use orchestrator confidence if our output doesn't set one.
	if content.Confidence == 0 {
		content.Confidence = resp.Confidence
	}

	// LLM-as-Judge: validate the generated content.
	validator := NewContentValidator(g.aiOrch, g.logger)
	review, err := validator.Validate(&content, &ValidateRequest{
		Title:       content.Title,
		Description: content.Description,
		Language:    in.TargetLanguage,
		Platform:    in.Platform,
	})
	if err == nil {
		content.Confidence = review.AdjustedConfidence
		if review.HasIssues() {
			g.logger.Warn("content validation found issues",
				zap.Any("issues", review.Issues))
		}
	}

	return &content, nil
}

func (g *ContentGenerator) buildPrompt(in *GenerateRequest) string {
	switch in.TargetLanguage {
	case "zh":
		return fmt.Sprintf("为以下跨境电商品生成中文商品标题(最多60字)和描述(最多300字)，适用于%s平台。\n商品名: %s\n品类: %s\n品牌: %s\n规格: %s\n要求: 语言简洁、突出卖点、包含搜索关键词、不含违禁词。",
			in.Platform, in.ProductName, in.Category, in.Brand, in.Specifications)
	case "ru":
		return fmt.Sprintf("Создайте заголовок (до 100 символов) и описание (до 500 символов) для товара на платформе %s. \nТовар: %s\nКатегория: %s\nБренд: %s\nХарактеристики: %s\nТребования: кратко, с выделением преимуществ, с ключевыми словами.",
			in.Platform, in.ProductName, in.Category, in.Brand, in.Specifications)
	default:
		return fmt.Sprintf("Generate a product title (max 80 chars) and description (max 500 chars) for %s platform.\nProduct: %s\nCategory: %s\nBrand: %s\nSpecs: %s",
			in.Platform, in.ProductName, in.Category, in.Brand, in.Specifications)
	}
}
