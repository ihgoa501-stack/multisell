package integrations

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ShopeeAPIBase is the default base URL for Shopee Open Platform API v2.
const ShopeeAPIBase = "https://partner.shopeemobile.com"

// ShopeeDefaultTimeout is the HTTP client timeout for Shopee API calls.
const ShopeeDefaultTimeout = 30 * time.Second

// ShopeeAdapter implements PlatformAdapter for the Shopee Open Platform API v2.
//
// Authentication: HMAC-SHA256 signature based on
//
//	Partner ID + API Key + Access Token + Shop ID.
//
// Docs: https://open.shopee.com/documents/v2/
type ShopeeAdapter struct {
	httpClient *http.Client
}

// shopeeAuth holds the credentials and base URL for Shopee API authentication.
type shopeeAuth struct {
	PartnerID   int64
	APIKey      string
	AccessToken string
	ShopID      int64
	BaseURL     string
}

// NewShopeeAdapter creates a new ShopeeAdapter with a default HTTP client.
func NewShopeeAdapter() *ShopeeAdapter {
	return &ShopeeAdapter{
		httpClient: &http.Client{Timeout: ShopeeDefaultTimeout},
	}
}

// ---------------------------------------------------------------------------
// Authentication helpers
// ---------------------------------------------------------------------------

// shopeeSign generates the HMAC-SHA256 signature for Shopee API v2.
//
// Signature base string:
//
//	"{partner_id}|{api_path}|{timestamp}|{access_token}|{shop_id}"
func shopeeSign(apiKey string, partnerID int64, apiPath string, timestamp int64, accessToken string, shopID int64) string {
	raw := fmt.Sprintf("%d|%s|%d|%s|%d", partnerID, apiPath, timestamp, accessToken, shopID)
	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write([]byte(raw))
	return hex.EncodeToString(mac.Sum(nil))
}

// buildAuthQuery builds the URL query string with Shopee auth parameters
// (partner_id, timestamp, access_token, shop_id, sign). The sign is computed
// from apiPath and the auth credentials.
func (a *ShopeeAdapter) buildAuthQuery(auth *shopeeAuth, apiPath string) string {
	ts := time.Now().Unix()
	sig := shopeeSign(auth.APIKey, auth.PartnerID, apiPath, ts, auth.AccessToken, auth.ShopID)
	vals := url.Values{}
	vals.Set("partner_id", strconv.FormatInt(auth.PartnerID, 10))
	vals.Set("timestamp", strconv.FormatInt(ts, 10))
	vals.Set("access_token", auth.AccessToken)
	vals.Set("shop_id", strconv.FormatInt(auth.ShopID, 10))
	vals.Set("sign", sig)
	return vals.Encode()
}

// ---------------------------------------------------------------------------
// Low-level HTTP helpers
// ---------------------------------------------------------------------------

// doGet sends an HTTP GET to path with auth + extra query params merged.
func (a *ShopeeAdapter) doGet(ctx context.Context, path string, auth *shopeeAuth, extraParams map[string]string) ([]byte, error) {
	authQuery := a.buildAuthQuery(auth, path)
	vals, _ := url.ParseQuery(authQuery)
	for k, v := range extraParams {
		vals.Set(k, v)
	}
	fullURL := auth.BaseURL + path + "?" + vals.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("shopee request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return a.doRequest(req, path)
}

// doPost sends an HTTP POST to path. Auth params go in the query string;
// the payload (if any) is serialised as JSON in the request body.
func (a *ShopeeAdapter) doPost(ctx context.Context, path string, auth *shopeeAuth, payload interface{}) ([]byte, error) {
	fullURL := auth.BaseURL + path + "?" + a.buildAuthQuery(auth, path)

	var bodyReader io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("shopee marshal: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("shopee request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return a.doRequest(req, path)
}

// doRequest executes a prepared request and parses the Shopee API response.
// It checks HTTP status codes and Shopee's own error code (0 = success).
func (a *ShopeeAdapter) doRequest(req *http.Request, path string) ([]byte, error) {
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("shopee %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("shopee read %s: %w", path, err)
	}

	if resp.StatusCode >= 400 {
		var e struct {
			Error            int    `json:"error"`
			Message          string `json:"message"`
			ErrorDescription string `json:"error_description"`
		}
		if json.Unmarshal(body, &e) == nil && e.Message != "" {
			return nil, fmt.Errorf("shopee %s [%d]: %s", path, resp.StatusCode, e.Message)
		}
		return nil, fmt.Errorf("shopee %s: HTTP %d %s", path, resp.StatusCode, truncStr(string(body), 300))
	}

	// Shopee returns error=0 on success; any non-zero value is a failure.
	var check struct {
		Error   int    `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &check); err == nil && check.Error != 0 {
		msg := check.Message
		if msg == "" {
			msg = fmt.Sprintf("error_code=%d", check.Error)
		}
		return nil, fmt.Errorf("shopee %s [error=%d]: %s", path, check.Error, msg)
	}

	return body, nil
}

// ---------------------------------------------------------------------------
// Listing domain — Publish
// ---------------------------------------------------------------------------

// Publish creates a product listing on Shopee via POST /api/v2/product/add.
func (a *ShopeeAdapter) Publish(ctx context.Context, input *PublishInput) (*PublishResult, error) {
	if len(input.SKUs) == 0 {
		return nil, fmt.Errorf("shopee publish: no SKUs")
	}
	auth := &shopeeAuth{BaseURL: ShopeeAPIBase}

	var variations []map[string]interface{}
	for _, sku := range input.SKUs {
		price := safeFloat(input.Prices[sku.SkuID])
		qty := input.Inventories[sku.SkuID]

		v := map[string]interface{}{
			"name":     sku.SkuCode,
			"sku_code": sku.SkuCode,
			"price":    fmt.Sprintf("%.2f", price),
			"stock":    qty,
		}
		if input.PackageWeight > 0 {
			v["weight"] = input.PackageWeight
		}
		if input.PackageLength > 0 {
			v["package_length"] = input.PackageLength
		}
		if input.PackageWidth > 0 {
			v["package_width"] = input.PackageWidth
		}
		if input.PackageHeight > 0 {
			v["package_height"] = input.PackageHeight
		}
		variations = append(variations, v)
	}

	payload := map[string]interface{}{
		"name":        truncateRunes(input.ProductName, 120),
		"description": truncateRunes(input.Description, 30000),
		"category_id": input.CategoryID,
		"condition":   "new",
		"variations":  variations,
		"logistics": []map[string]interface{}{
			{"logistic_id": 0, "enabled": true, "shipping_fee": 0, "is_free": false},
		},
	}
	if input.BrandID > 0 {
		payload["brand"] = map[string]interface{}{"brand_id": input.BrandID}
	}
	if input.MainImage != "" {
		images := []string{input.MainImage}
		for _, img := range input.Images {
			if img != input.MainImage && len(images) < 9 {
				images = append(images, img)
			}
		}
		payload["images"] = images
	}

	body, err := a.doPost(ctx, "/api/v2/product/add", auth, payload)
	if err != nil {
		// Handle duplicate product detection for auto-binding
		errStr := err.Error()
		if strings.Contains(errStr, "duplicate") || strings.Contains(errStr, "already exists") {
			sku := input.SKUs[0].SkuCode
			platformProductID := fmt.Sprintf("shopee-existing-%d", input.ProductID)
			return &PublishResult{
				PlatformProductID: platformProductID,
				PlatformSKU:       sku,
				PlatformURL:       fmt.Sprintf("https://shopee.ph/product/%s/", platformProductID),
				PublishedData:     map[string]interface{}{"message": "Auto-bound existing platform product"},
				SyncMessage:       fmt.Sprintf("auto-bound duplicate product: %s", errStr),
			}, nil
		}
		return nil, err
	}

	var r struct {
		Response struct {
			ItemID int64 `json:"item_id"`
		} `json:"response"`
	}
	json.Unmarshal(body, &r)

	platformProductID := fmt.Sprintf("shopee-%d", r.Response.ItemID)
	sku := input.SKUs[0].SkuCode

	return &PublishResult{
		PlatformProductID: platformProductID,
		PlatformSKU:       sku,
		PlatformURL:       fmt.Sprintf("https://shopee.ph/product/%s/", platformProductID),
		PublishedData:     map[string]interface{}{"item_id": r.Response.ItemID},
		SyncMessage:       fmt.Sprintf("published to Shopee (item_id=%d)", r.Response.ItemID),
	}, nil
}

// SyncStatus checks the product sync status on Shopee via POST /api/v2/product/get_item_base_info.
func (a *ShopeeAdapter) SyncStatus(ctx context.Context, input *SyncStatusInput) (string, error) {
	platformProductID := input.PlatformProductID
	itemID := strings.TrimPrefix(platformProductID, "shopee-")
	if itemID == "" || itemID == platformProductID {
		return "unknown", nil
	}
	// Placeholder for auto-bound duplicate items
	if strings.HasPrefix(itemID, "existing-") {
		return "synced", nil
	}

	id, err := strconv.ParseInt(itemID, 10, 64)
	if err != nil {
		return "unknown", nil
	}

	auth := &shopeeAuth{BaseURL: ShopeeAPIBase}
	payload := map[string]interface{}{"item_id": id}
	body, err := a.doPost(ctx, "/api/v2/product/get_item_base_info", auth, payload)
	if err != nil {
		return "unknown", err
	}

	var r struct {
		Response struct {
			ItemStatus string `json:"item_status"`
		} `json:"response"`
	}
	json.Unmarshal(body, &r)

	statusMap := map[string]string{
		"NORMAL":  "synced",
		"UNLIST":  "unlisted",
		"BANNED":  "banned",
		"DELETED": "deleted",
	}
	if s, ok := statusMap[r.Response.ItemStatus]; ok {
		return s, nil
	}
	return "unknown", nil
}

// ValidateCredentials checks the Shopee account by calling GET /api/v2/shop/get_shop_info.
func (a *ShopeeAdapter) ValidateCredentials(ctx context.Context, accountID int64) (bool, error) {
	auth := &shopeeAuth{BaseURL: ShopeeAPIBase}
	body, err := a.doGet(ctx, "/api/v2/shop/get_shop_info", auth, nil)
	if err != nil {
		return false, err
	}
	var r struct {
		Error int `json:"error"`
	}
	json.Unmarshal(body, &r)
	return r.Error == 0, nil
}

// ---------------------------------------------------------------------------
// Inventory domain — SyncInventory
// ---------------------------------------------------------------------------

// SyncInventory updates stock on Shopee via POST /api/v2/product/update_stock.
func (a *ShopeeAdapter) SyncInventory(ctx context.Context, input *SyncInventoryInput) (bool, error) {
	auth := &shopeeAuth{BaseURL: ShopeeAPIBase}

	modelID := int64(0)
	if input.PlatformSKU != "" {
		if id, err := strconv.ParseInt(input.PlatformSKU, 10, 64); err == nil {
			modelID = id
		}
	}

	payload := map[string]interface{}{
		"model_id": modelID,
		"stock":    input.Quantity,
	}
	_, err := a.doPost(ctx, "/api/v2/product/update_stock", auth, payload)
	return err == nil, err
}

// ---------------------------------------------------------------------------
// Logistics domain — PushTracking
// ---------------------------------------------------------------------------

// PushTracking sends tracking information to Shopee via
// POST /api/v2/logistics/update_shipping_document.
func (a *ShopeeAdapter) PushTracking(ctx context.Context, input *PushTrackingInput) (bool, error) {
	auth := &shopeeAuth{BaseURL: ShopeeAPIBase}
	payload := map[string]interface{}{
		"order_sn":        input.OrderSN,
		"tracking_number": input.TrackingNumber,
	}
	_, err := a.doPost(ctx, "/api/v2/logistics/update_shipping_document", auth, payload)
	return err == nil, err
}

// ---------------------------------------------------------------------------
// Orders domain — FetchOrders
// ---------------------------------------------------------------------------

// FetchOrders pulls orders from Shopee within the time window.
//
//  1. GET /api/v2/order/get_order_list — list order SNs.
//  2. GET /api/v2/order/get_order_detail — fetch each order's details.
func (a *ShopeeAdapter) FetchOrders(ctx context.Context, input *FetchOrdersInput) ([]*PlatformOrder, error) {
	auth := &shopeeAuth{BaseURL: ShopeeAPIBase}
	extraParams := map[string]string{
		"time_range_field": "create_time",
		"page_size":        "100",
		"create_time_from": strconv.FormatInt(input.Since.Unix(), 10),
		"create_time_to":   strconv.FormatInt(time.Now().Unix(), 10),
	}

	body, err := a.doGet(ctx, "/api/v2/order/get_order_list", auth, extraParams)
	if err != nil {
		return nil, fmt.Errorf("shopee fetch_orders: %w", err)
	}

	var r struct {
		Response struct {
			OrderList []string `json:"order_list"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("shopee parse order list: %w", err)
	}

	var orders []*PlatformOrder
	for _, orderSN := range r.Response.OrderList {
		order, err := a.fetchOrderDetail(ctx, auth, orderSN)
		if err != nil {
			// Skip problematic orders rather than failing the whole batch.
			continue
		}
		orders = append(orders, order)
	}
	return orders, nil
}

// fetchOrderDetail retrieves and parses the full detail for a single Shopee order.
func (a *ShopeeAdapter) fetchOrderDetail(ctx context.Context, auth *shopeeAuth, orderSN string) (*PlatformOrder, error) {
	extraParams := map[string]string{"order_sn_list": orderSN}
	body, err := a.doGet(ctx, "/api/v2/order/get_order_detail", auth, extraParams)
	if err != nil {
		return nil, fmt.Errorf("shopee order detail %s: %w", orderSN, err)
	}

	var r struct {
		Response struct {
			OrderList []struct {
				OrderStatus string `json:"order_status"`
				TotalAmount struct {
					Value string `json:"value"`
				} `json:"total_amount"`
				PayTime string `json:"pay_time"`
				RecipientAddress struct {
					Name        string `json:"name"`
					Phone       string `json:"phone"`
					FullAddress string `json:"full_address"`
				} `json:"recipient_address"`
				ItemList []struct {
					ItemSKU                 string `json:"item_sku"`
					ModelQuantityPurchased  int    `json:"model_quantity_purchased"`
					ModelOriginalPrice      string `json:"model_original_price"`
				} `json:"item_list"`
			} `json:"order_list"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("shopee parse order detail %s: %w", orderSN, err)
	}
	if len(r.Response.OrderList) == 0 {
		return nil, fmt.Errorf("shopee order %s not found", orderSN)
	}

	detail := r.Response.OrderList[0]
	var items []PlatformOrderItem
	for _, item := range detail.ItemList {
		items = append(items, PlatformOrderItem{
			SkuCode:   item.ItemSKU,
			Quantity:  item.ModelQuantityPurchased,
			UnitPrice: item.ModelOriginalPrice,
		})
	}

	return &PlatformOrder{
		OrderSN:         orderSN,
		Status:          detail.OrderStatus,
		TotalAmount:     detail.TotalAmount.Value,
		ShippingFee:     "0",
		PaidAt:          detail.PayTime,
		RecipientName:   detail.RecipientAddress.Name,
		RecipientPhone:  detail.RecipientAddress.Phone,
		ShippingAddress: detail.RecipientAddress.FullAddress,
		Items:           items,
	}, nil
}

// ---------------------------------------------------------------------------
// Settlements domain — not implemented (Phase 1 stub)
// ---------------------------------------------------------------------------

// FetchSettlements is a stub — Shopee settlement import is not yet implemented.
func (a *ShopeeAdapter) FetchSettlements(ctx context.Context, input *FetchSettlementsInput) ([]*PlatformSettlement, error) {
	return []*PlatformSettlement{}, nil
}

// ---------------------------------------------------------------------------
// Returns domain — not implemented (Phase 1 stub)
// ---------------------------------------------------------------------------

// FetchReturns is a stub — Shopee returns import is not yet implemented.
func (a *ShopeeAdapter) FetchReturns(ctx context.Context, input *FetchReturnsInput) ([]*PlatformReturn, error) {
	return []*PlatformReturn{}, nil
}

// ---------------------------------------------------------------------------
// Helpers (Shopee-specific)
// ---------------------------------------------------------------------------

// truncateRunes truncates a string to at most maxLen runes. Shopee field
// length limits are character-based (runes), so byte-level truncation via
// truncStr would break multi-byte characters such as Chinese.
func truncateRunes(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}
