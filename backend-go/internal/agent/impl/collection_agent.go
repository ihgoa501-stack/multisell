package impl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lingmirror/backend-go/internal/domain/candidate"
	"github.com/lingmirror/backend-go/internal/platform/toolbridge"
	"go.uber.org/zap"
)

// CollectionAgent implements A12 Collection Agent logic.
// It drives the browser extension via ToolBridge to scrape product data
// from supplier sites and saves results as CandidateProduct records.
type CollectionAgent struct {
	toolBridge *toolbridge.ToolBridge
	candSvc    *candidate.Service
	logger     *zap.Logger
}

// NewCollectionAgent creates an A12 CollectionAgent.
// toolBridge and candSvc can be nil for test mode.
func NewCollectionAgent(tb *toolbridge.ToolBridge, cs *candidate.Service, logger *zap.Logger) *CollectionAgent {
	return &CollectionAgent{
		toolBridge: tb,
		candSvc:    cs,
		logger:     logger.Named("A12"),
	}
}

// Decide implements the Agent interface.
// Decision points:
//   - "product_collect" — collect products from given URLs
//     params: { "urls": ["..."], "category_id": N, "collected_by": "A12" }
//   - "supplier_scrape" — scrape all products from a supplier catalog
//     params: { "supplier_url": "...", "max_products": 50 }
func (a *CollectionAgent) Decide(ctx context.Context, decisionPoint string, params map[string]interface{}) (map[string]interface{}, float64, string, error) {
	switch decisionPoint {
	case "product_collect":
		return a.collectProducts(ctx, params)
	case "supplier_scrape":
		return a.scrapeSupplier(ctx, params)
	default:
		return nil, 0, "unknown", fmt.Errorf("collection agent: unknown decision point %s", decisionPoint)
	}
}

// supplier_scrape — scrape all products from a supplier catalog
// params: { "supplier_url": "...", "max_products": 50 }
func (a *CollectionAgent) scrapeSupplier(ctx context.Context, params map[string]interface{}) (map[string]interface{}, float64, string, error) {
	url, _ := params["supplier_url"].(string)
	if url == "" {
		return nil, 0, "error", fmt.Errorf("supplier_url required")
	}
	collectProducts := true
	if _, ok := params["collect_products"]; ok {
		collectProducts, _ = params["collect_products"].(bool)
	}

	if collectProducts {
		// For now, treat supplier_url as a product URL and collect.
		// Future: extract catalog URLs first, then collect each.
		pid, err := a.collectSingle(ctx, url, params)
		if err != nil {
			return nil, 0, "error", err
		}
		return map[string]interface{}{
			"collected":     1,
			"product_ids":   []int64{pid},
			"supplier_url":  url,
		}, 1.0, "low", nil
	}
	return map[string]interface{}{
		"scraped":        false,
		"supplier_url":   url,
		"message":        "supplier discovery only (collection deferred)",
	}, 1.0, "low", nil
}

func (a *CollectionAgent) collectProducts(ctx context.Context, params map[string]interface{}) (map[string]interface{}, float64, string, error) {
	urlsRaw, ok := params["urls"]
	if !ok {
		return nil, 0, "error", fmt.Errorf("'urls' required ([]string)")
	}

	urls, ok := urlsRaw.([]interface{})
	if !ok {
		return nil, 0, "error", fmt.Errorf("'urls' must be an array")
	}
	if len(urls) == 0 {
		return map[string]interface{}{
			"collected": 0,
			"message":   "no URLs provided",
		}, 1.0, "low", nil
	}

	var productIDs []int64
	successCount := 0
	failCount := 0

	for i, rawURL := range urls {
		urlStr := fmt.Sprintf("%v", rawURL)
		a.logger.Info("collecting product", zap.Int("index", i), zap.String("url", urlStr))

		// Set per-URL params with category context.
		urlParams := copyParams(params)
		urlParams["url"] = urlStr

		pid, err := a.collectSingle(ctx, urlStr, urlParams)
		if err != nil {
			a.logger.Warn("collect failed", zap.String("url", urlStr), zap.Error(err))
			failCount++
			continue
		}
		productIDs = append(productIDs, pid)
		successCount++
	}

	return map[string]interface{}{
		"collected":    successCount,
		"failed":       failCount,
		"product_ids":  productIDs,
		"total_urls":   len(urls),
		"agent_id":     "A12",
		"decision":     "product_collect",
	}, 1.0, "low", nil
}

func (a *CollectionAgent) collectSingle(ctx context.Context, url string, params map[string]interface{}) (int64, error) {
	var pageData *toolbridge.PageData

	if a.toolBridge != nil {
		var err error
		pageData, err = a.toolBridge.FetchPage(ctx, url)
		if err != nil {
			return 0, fmt.Errorf("fetch page %s: %w", url, err)
		}
	} else {
		// ponytail: test mode — no ToolBridge, just return a stub product.
		pageData = &toolbridge.PageData{
			SourceURL: url,
			Title:     fmt.Sprintf("Collected from %s", url),
			PriceCNY:  0,
		}
	}

	if pageData == nil {
		return 0, fmt.Errorf("no data returned for %s", url)
	}

	// Map PageData → CandidateProduct.
	input := pageDataToCandidate(pageData, params)

	if a.candSvc != nil {
		created, err := a.candSvc.Create(input)
		if err != nil {
			return 0, fmt.Errorf("save candidate: %w", err)
		}
		return created.ID, nil
	}

	return 0, nil
}

// pageDataToCandidate converts scraped PageData to a CreateCandidateInput.
func pageDataToCandidate(pd *toolbridge.PageData, params map[string]interface{}) *candidate.CreateCandidateInput {
	imagesJSON, _ := json.Marshal(pd.Images)

	specJSON := json.RawMessage(nil)
	if len(pd.SpecVariants) > 0 {
		specJSON, _ = json.Marshal(pd.SpecVariants)
	}

	price := pd.PriceCNY
	weight := 0.0
	length := 0.0
	width := 0.0
	height := 0.0

	if pd.WeightKg != nil {
		weight = *pd.WeightKg
	}
	if pd.PackageLengthCm != nil {
		length = *pd.PackageLengthCm
	}
	if pd.PackageWidthCm != nil {
		width = *pd.PackageWidthCm
	}
	if pd.PackageHeightCm != nil {
		height = *pd.PackageHeightCm
	}

	collectedBy := "A12"
	if v, ok := params["collected_by"].(string); ok {
		collectedBy = v
	}

	// Single image = first from array.
	mainImage := ""
	if len(pd.Images) > 0 {
		mainImage = pd.Images[0]
	}

	return &candidate.CreateCandidateInput{
		Title:            pd.Title,
		Description:      pd.Description,
		MainImage:        mainImage,
		Images:           imagesJSON,
		SpecJSON:         specJSON,
		PurchasePrice:    &price,
		PurchaseCurrency: "CNY",
		PackageWeightKg:  &weight,
		PackageLengthCm:  &length,
		PackageWidthCm:   &width,
		PackageHeightCm:  &height,
		OriginCountry:    "CN",
		Status:           "draft",
		CreatedBy:        collectedBy,
		SourceURL:        pd.SourceURL,
		SourcePlatform:   detectPlatform(pd.SourceURL),
		RawPayload:       pd.RawData,
	}
}

// detectPlatform guesses the source platform from a URL.
func detectPlatform(url string) string {
	switch {
	case strings.Contains(url, "1688.com"):
		return "1688"
	case strings.Contains(url, "taobao.com"):
		return "taobao"
	case strings.Contains(url, "tmall.com"):
		return "tmall"
	case strings.Contains(url, "aliexpress.com"):
		return "aliexpress"
	case strings.Contains(url, "alibaba.com"):
		return "alibaba"
	case strings.Contains(url, "pinduoduo.com"):
		return "pinduoduo"
	case strings.Contains(url, "made-in-china.com"):
		return "made-in-china"
	case strings.Contains(url, "yiwugo.com"):
		return "yiwugo"
	default:
		return "unknown"
	}
}

func copyParams(original map[string]interface{}) map[string]interface{} {
	cp := make(map[string]interface{}, len(original))
	for k, v := range original {
		cp[k] = v
	}
	return cp
}
