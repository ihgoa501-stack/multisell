package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	ShopifyAPIVersion     = "2026-01"
	ShopifyDefaultTimeout = 30 * time.Second
)

// ShopifyAdapter implements PlatformAdapter for the Shopify REST Admin API.
type ShopifyAdapter struct {
	httpClient *http.Client
	db         *gorm.DB
	logger     *zap.Logger
}

func NewShopifyAdapter(db *gorm.DB, logger *zap.Logger) *ShopifyAdapter {
	return &ShopifyAdapter{
		httpClient: &http.Client{Timeout: ShopifyDefaultTimeout},
		db:         db,
		logger:     logger,
	}
}

type shopifyAuth struct {
	ShopName    string
	AccessToken string
	BaseURL     string
	APIVersion  string
}

// getAuth looks up the first active platform integration account for the given
// platform ID and returns Shopify credentials. Config stores shop_name and
// optionally api_version. The access_token is the Shopify OAuth or private app token.
func (a *ShopifyAdapter) getAuth(ctx context.Context, platformID int64) (*shopifyAuth, error) {
	var accts []PlatformIntegrationAccount
	if err := a.db.WithContext(ctx).
		Where("platform_id = ? AND status = ?", platformID, "active").
		Limit(1).
		Find(&accts).Error; err != nil {
		return nil, fmt.Errorf("shopify getAuth: %w", err)
	}
	if len(accts) == 0 {
		return nil, fmt.Errorf("shopify getAuth: no active account for platform_id=%d", platformID)
	}
	acct := accts[0]

	var cfg struct {
		ShopName   string `json:"shop_name"`
		APIVersion string `json:"api_version"`
	}
	if len(acct.Config) > 0 {
		json.Unmarshal(acct.Config, &cfg)
	}
	if acct.AccessToken == "" {
		return nil, fmt.Errorf("shopify getAuth: account %d has empty access_token", acct.ID)
	}
	if cfg.ShopName == "" {
		return nil, fmt.Errorf("shopify getAuth: account %d missing shop_name in config", acct.ID)
	}
	apiVer := cfg.APIVersion
	if apiVer == "" {
		apiVer = ShopifyAPIVersion
	}
	return &shopifyAuth{
		ShopName:    cfg.ShopName,
		AccessToken: acct.AccessToken,
		BaseURL:     fmt.Sprintf("https://%s.myshopify.com/admin/api/%s", cfg.ShopName, apiVer),
		APIVersion:  apiVer,
	}, nil
}

// getAuthByAccountID resolves a platform integration account ID to Shopify creds.
func (a *ShopifyAdapter) getAuthByAccountID(ctx context.Context, accountID int64) (*shopifyAuth, error) {
	var acct PlatformIntegrationAccount
	if err := a.db.WithContext(ctx).First(&acct, accountID).Error; err != nil {
		return nil, fmt.Errorf("shopify getAuthByAccountID: %w", err)
	}
	return a.getAuth(ctx, acct.PlatformID)
}

func (a *ShopifyAdapter) do(ctx context.Context, method, path string, auth *shopifyAuth, payload interface{}) ([]byte, error) {
	url := auth.BaseURL + path
	var bodyReader io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("shopify marshal: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("shopify request: %w", err)
	}
	req.Header.Set("X-Shopify-Access-Token", auth.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("shopify %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("shopify read %s: %w", path, err)
	}

	if resp.StatusCode >= 400 {
		var e struct {
			Errors interface{} `json:"errors"`
		}
		msg := string(body)
		if json.Unmarshal(body, &e) == nil && e.Errors != nil {
			switch v := e.Errors.(type) {
			case string:
				msg = v
			default:
				b, _ := json.Marshal(v)
				msg = string(b)
			}
		}
		return nil, fmt.Errorf("shopify %s: HTTP %d %s", path, resp.StatusCode, truncStr(msg, 300))
	}
	return body, nil
}

// ─── Publish ───

func (a *ShopifyAdapter) Publish(ctx context.Context, input *PublishInput) (*PublishResult, error) {
	if len(input.SKUs) == 0 {
		return nil, fmt.Errorf("shopify publish: no SKUs")
	}
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return nil, err
	}

	// Build variants from SKUs
	type variantPayload struct {
		Price string `json:"price"`
		Sku   string `json:"sku"`
	}
	var variants []variantPayload
	for _, sku := range input.SKUs {
		price := input.Prices[sku.SkuID]
		if price == "" {
			price = "0.00"
		}
		variants = append(variants, variantPayload{
			Price: price,
			Sku:   sku.SkuCode,
		})
	}
	if len(variants) == 0 {
		variants = []variantPayload{{Price: "0.00", Sku: ""}}
	}

	payload := map[string]interface{}{
		"product": map[string]interface{}{
			"title":     input.ProductName,
			"body_html": input.Description,
			"status":    "draft",
			"variants":  variants,
		},
	}

	body, err := a.do(ctx, http.MethodPost, "/products.json", auth, payload)
	if err != nil {
		return nil, err
	}
	var r struct {
		Product struct {
			ID       int64  `json:"id"`
			Title    string `json:"title"`
			Handle   string `json:"handle"`
			Variants []struct {
				ID  int64  `json:"id"`
				Sku string `json:"sku"`
			} `json:"variants"`
		} `json:"product"`
	}
	json.Unmarshal(body, &r)

	sku := input.SKUs[0].SkuCode
	return &PublishResult{
		PlatformProductID: fmt.Sprintf("shopify-%d", r.Product.ID),
		PlatformSKU:       sku,
		PlatformURL:       fmt.Sprintf("https://%s.myshopify.com/products/%s", auth.ShopName, r.Product.Handle),
		PublishedData:     map[string]interface{}{"product_id": r.Product.ID, "handle": r.Product.Handle},
		SyncMessage:       fmt.Sprintf("published to Shopify as draft (product_id=%d)", r.Product.ID),
	}, nil
}

// ─── SyncStatus ───

func (a *ShopifyAdapter) SyncStatus(ctx context.Context, input *SyncStatusInput) (string, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return "unknown", err
	}
	productID := strings.TrimPrefix(input.PlatformProductID, "shopify-")
	body, err := a.do(ctx, http.MethodGet, "/products/"+productID+".json", auth, nil)
	if err != nil {
		return "unknown", err
	}
	var r struct {
		Product struct {
			Status string `json:"status"`
		} `json:"product"`
	}
	json.Unmarshal(body, &r)
	m := map[string]string{"active": "synced", "draft": "pending", "archived": "failed"}
	if s, ok := m[r.Product.Status]; ok {
		return s, nil
	}
	return r.Product.Status, nil
}

// ─── ValidateCredentials ───

func (a *ShopifyAdapter) ValidateCredentials(ctx context.Context, accountID int64) (bool, error) {
	auth, err := a.getAuthByAccountID(ctx, accountID)
	if err != nil {
		return false, fmt.Errorf("shopify ValidateCredentials: %w", err)
	}
	// Hit /shop.json (lightweight) to verify the token works.
	_, err = a.do(ctx, http.MethodGet, "/shop.json", auth, nil)
	if err != nil {
		return false, fmt.Errorf("shopify ValidateCredentials: %w", err)
	}
	return true, nil
}

// ─── SyncInventory ───

func (a *ShopifyAdapter) SyncInventory(ctx context.Context, input *SyncInventoryInput) (bool, error) {
	return false, fmt.Errorf("shopify SyncInventory: not yet implemented — requires location_id and inventory_item_id resolution")
}

// ─── PushTracking ───

func (a *ShopifyAdapter) PushTracking(ctx context.Context, input *PushTrackingInput) (bool, error) {
	return false, fmt.Errorf("shopify PushTracking: not yet implemented — requires fulfillment order flow")
}

// ─── FetchOrders ───

// linkHeaderRE matches Shopify Link header page_info cursors.
var linkHeaderRE = regexp.MustCompile(`<[^>]*page_info=([^&>]+)[^>]*>;\s*rel="(\w+)"`)

func (a *ShopifyAdapter) FetchOrders(ctx context.Context, input *FetchOrdersInput) ([]*PlatformOrder, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return nil, err
	}
	var orders []*PlatformOrder
	pageInfo := ""
	for {
		path := "/orders.json?status=any&limit=250&updated_at_min=" + input.Since.Format(time.RFC3339)
		if pageInfo != "" {
			path += "&page_info=" + pageInfo
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, auth.BaseURL+path, nil)
		if err != nil {
			return nil, fmt.Errorf("shopify fetch_orders req: %w", err)
		}
		req.Header.Set("X-Shopify-Access-Token", auth.AccessToken)

		resp, err := a.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("shopify fetch_orders: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("shopify fetch_orders read: %w", err)
		}
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("shopify fetch_orders: HTTP %d %s", resp.StatusCode, truncStr(string(body), 200))
		}

		var r struct {
			Orders []struct {
				ID                int64  `json:"id"`
				Name              string `json:"name"`
				CreatedAt         string `json:"created_at"`
				UpdatedAt         string `json:"updated_at"`
				TotalPrice        string `json:"total_price"`
				TotalShipping     string `json:"total_shipping"`
				FinancialStatus   string `json:"financial_status"`
				FulfillmentStatus string `json:"fulfillment_status"`
				ShippingAddress   *struct {
					Name     string `json:"name"`
					Phone    string `json:"phone"`
					Address1 string `json:"address1"`
					Address2 string `json:"address2"`
					City     string `json:"city"`
					Province string `json:"province"`
					Zip      string `json:"zip"`
					Country  string `json:"country"`
				} `json:"shipping_address"`
				LineItems []struct {
					Sku      string `json:"sku"`
					Name     string `json:"name"`
					Quantity int    `json:"quantity"`
					Price    string `json:"price"`
				} `json:"line_items"`
			} `json:"orders"`
		}
		json.Unmarshal(body, &r)

		for _, o := range r.Orders {
			var items []PlatformOrderItem
			for _, li := range o.LineItems {
				skuCode := li.Sku
				if skuCode == "" {
					skuCode = li.Name
				}
				items = append(items, PlatformOrderItem{
					SkuCode:   skuCode,
					Quantity:  li.Quantity,
					UnitPrice: li.Price,
				})
			}
			var addr string
			if o.ShippingAddress != nil {
				var parts []string
				if o.ShippingAddress.Address1 != "" {
					parts = append(parts, o.ShippingAddress.Address1)
				}
				if o.ShippingAddress.Address2 != "" {
					parts = append(parts, o.ShippingAddress.Address2)
				}
				if o.ShippingAddress.City != "" {
					parts = append(parts, o.ShippingAddress.City)
				}
				if o.ShippingAddress.Province != "" {
					parts = append(parts, o.ShippingAddress.Province)
				}
				if o.ShippingAddress.Zip != "" {
					parts = append(parts, o.ShippingAddress.Zip)
				}
				if o.ShippingAddress.Country != "" {
					parts = append(parts, o.ShippingAddress.Country)
				}
				addr = strings.Join(parts, ", ")
			}
			shippingFee := o.TotalShipping
			if shippingFee == "" {
				shippingFee = "0.00"
			}
			orders = append(orders, &PlatformOrder{
				OrderSN:         strconv.FormatInt(o.ID, 10),
				Status:          o.FinancialStatus,
				TotalAmount:     o.TotalPrice,
				ShippingFee:     shippingFee,
				PaidAt:          o.CreatedAt,
				RecipientName:   orderRecipientName(o.ShippingAddress),
				RecipientPhone:  orderRecipientPhone(o.ShippingAddress),
				ShippingAddress: addr,
				Items:           items,
			})
		}

		// Paginate via Link header
		linkHeader := resp.Header.Get("Link")
		pageInfo = ""
		if linkHeader != "" {
			matches := linkHeaderRE.FindAllStringSubmatch(linkHeader, -1)
			for _, m := range matches {
				if len(m) >= 3 && m[2] == "next" {
					pageInfo = m[1]
					break
				}
			}
		}
		if pageInfo == "" || len(r.Orders) == 0 {
			break
		}
	}
	if orders == nil {
		orders = []*PlatformOrder{}
	}
	return orders, nil
}

func orderRecipientName(addr *struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Address1 string `json:"address1"`
	Address2 string `json:"address2"`
	City     string `json:"city"`
	Province string `json:"province"`
	Zip      string `json:"zip"`
	Country  string `json:"country"`
}) string {
	if addr == nil {
		return ""
	}
	return addr.Name
}

func orderRecipientPhone(addr *struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Address1 string `json:"address1"`
	Address2 string `json:"address2"`
	City     string `json:"city"`
	Province string `json:"province"`
	Zip      string `json:"zip"`
	Country  string `json:"country"`
}) string {
	if addr == nil {
		return ""
	}
	return addr.Phone
}

// ─── FetchSettlements ───

func (a *ShopifyAdapter) FetchSettlements(ctx context.Context, input *FetchSettlementsInput) ([]*PlatformSettlement, error) {
	return nil, fmt.Errorf("shopify FetchSettlements: not yet implemented — requires Shopify payouts API")
}

// ─── FetchReturns ───

func (a *ShopifyAdapter) FetchReturns(ctx context.Context, input *FetchReturnsInput) ([]*PlatformReturn, error) {
	return nil, fmt.Errorf("shopify FetchReturns: not yet implemented — requires iterating orders for refunds")
}
