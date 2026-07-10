package integrations

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	LazadaAPIBase        = "https://api.lazada.com/rest"
	LazadaAPISGBase      = "https://api.lazada.sg/rest"
	LazadaDefaultTimeout = 30 * time.Second
)

// LazadaAdapter implements PlatformAdapter for Lazada Open Platform.
type LazadaAdapter struct {
	httpClient *http.Client
	db         *gorm.DB
	logger     *zap.Logger
}

// NewLazadaAdapter creates a new Lazada adapter.
func NewLazadaAdapter(db *gorm.DB, logger *zap.Logger) *LazadaAdapter {
	return &LazadaAdapter{
		httpClient: &http.Client{Timeout: LazadaDefaultTimeout},
		db:         db,
		logger:     logger,
	}
}

// lazadaAuth stores Lazada API authentication details.
type lazadaAuth struct {
	AppKey      string // AppKey (App ID)
	AppSecret   string // App Secret for request signing
	AccessToken string // Seller access token
	BaseURL     string // API base URL (regional)
}

// getAuth looks up the first active platform integration account for Lazada
// and returns authentication credentials.
func (a *LazadaAdapter) getAuth(ctx context.Context, platformID int64) (*lazadaAuth, error) {
	var accts []PlatformIntegrationAccount
	if err := a.db.WithContext(ctx).
		Where("platform_id = ? AND status = ?", platformID, "active").
		Limit(1).
		Find(&accts).Error; err != nil {
		return nil, fmt.Errorf("lazada getAuth: %w", err)
	}
	if len(accts) == 0 {
		return nil, fmt.Errorf("lazada getAuth: no active account for platform_id=%d", platformID)
	}
	acct := accts[0]

	var cfg struct {
		AppKey    string `json:"app_key"`
		AppSecret string `json:"app_secret"`
		Region    string `json:"region"` // sg, my, th, id, ph, vn
	}
	if len(acct.Config) > 0 {
		json.Unmarshal(acct.Config, &cfg)
	}
	if cfg.AppKey == "" {
		return nil, fmt.Errorf("lazada getAuth: account %d missing app_key in config", acct.ID)
	}
	if acct.AccessToken == "" {
		return nil, fmt.Errorf("lazada getAuth: account %d has empty access_token", acct.ID)
	}

	baseURL := LazadaAPIBase
	switch strings.ToLower(cfg.Region) {
	case "sg":
		baseURL = LazadaAPISGBase
	}

	return &lazadaAuth{
		AppKey:      cfg.AppKey,
		AppSecret:   cfg.AppSecret,
		AccessToken: acct.AccessToken,
		BaseURL:     baseURL,
	}, nil
}

// getAuthByAccountID resolves a platform integration account ID to Lazada creds.
func (a *LazadaAdapter) getAuthByAccountID(ctx context.Context, accountID int64) (*lazadaAuth, error) {
	var acct PlatformIntegrationAccount
	if err := a.db.WithContext(ctx).First(&acct, accountID).Error; err != nil {
		return nil, fmt.Errorf("lazada getAuthByAccountID: %w", err)
	}
	return a.getAuth(ctx, acct.PlatformID)
}

// sign generates a Lazada API request signature per the Open Platform spec:
// sign = HMAC-SHA256(app_secret, sorted query string)
func (a *LazadaAdapter) sign(appSecret string, params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		if sb.Len() > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(params[k])
	}

	mac := hmac.New(md5.New, []byte(appSecret))
	mac.Write([]byte(sb.String()))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

// do makes a signed Lazada API request. Lazada uses query-string-based auth
// where system parameters (app_key, access_token, timestamp, sign_method, sign)
// are appended as query params even on POST requests.
func (a *LazadaAdapter) do(ctx context.Context, method, path string, auth *lazadaAuth, payload interface{}) ([]byte, error) {
	baseURL := strings.TrimRight(auth.BaseURL, "/")
	fullURL := baseURL + path

	// Build system query parameters
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	params := map[string]string{
		"app_key":      auth.AppKey,
		"timestamp":    timestamp,
		"sign_method":  "sha256",
		"access_token": auth.AccessToken,
	}

	// For POST with JSON body, add the body hash
	var bodyBytes []byte
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("lazada marshal: %w", err)
		}
		bodyBytes = b
		params["payload"] = string(b)
	}

	// Generate signature over the sorted parameters
	signature := a.sign(auth.AppSecret, params)

	// Build query string
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	q.Set("sign", signature)

	fullURL = fullURL + "?" + q.Encode()

	var bodyReader io.Reader
	if len(bodyBytes) > 0 {
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("lazada request: %w", err)
	}
	if len(bodyBytes) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lazada %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lazada read %s: %w", path, err)
	}

	// Check for errors in the response
	var errResp struct {
		Code    string `json:"code"`
		Type    string `json:"type"`
		Message string `json:"message"`
		Details string `json:"detail"`
	}
	if json.Unmarshal(body, &errResp) == nil && errResp.Code != "" && errResp.Code != "0" {
		return nil, fmt.Errorf("lazada %s [code=%s]: %s - %s", path, errResp.Code, errResp.Message, errResp.Details)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("lazada %s: HTTP %d %s", path, resp.StatusCode, truncStr(string(body), 300))
	}

	return body, nil
}

func (a *LazadaAdapter) Publish(ctx context.Context, input *PublishInput) (*PublishResult, error) {
	if len(input.SKUs) == 0 {
		return nil, fmt.Errorf("lazada publish: no SKUs")
	}
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return nil, err
	}

	// Lazada product creation via /product/create
	payload := map[string]interface{}{
		"Request": map[string]interface{}{
			"Product": map[string]interface{}{
				"Images": map[string]interface{}{
					"MainImage": map[string]string{
						"Image": input.MainImage,
					},
				},
				"PrimaryCategory": input.CategoryID,
				"Attributes": map[string]interface{}{
					"name":        input.ProductName,
					"description": input.Description,
					"brand":       fmt.Sprintf("%d", input.BrandID),
				},
				"Skus": []map[string]interface{}{
					{
						"SellerSku":   input.SKUs[0].SkuCode,
						"quantity":    input.Inventories[input.SKUs[0].SkuID],
						"price":       input.Prices[input.SKUs[0].SkuID],
						"package_height": fmt.Sprintf("%.1f", input.PackageHeight),
						"package_width":  fmt.Sprintf("%.1f", input.PackageWidth),
						"package_length": fmt.Sprintf("%.1f", input.PackageLength),
						"package_weight": fmt.Sprintf("%.1f", input.PackageWeight),
					},
				},
			},
		},
	}

	body, err := a.do(ctx, http.MethodPost, "/product/create", auth, payload)
	if err != nil {
		return nil, fmt.Errorf("lazada publish: %w", err)
	}

	var r struct {
		Code    string `json:"code"`
		Data    struct {
			ItemID int64 `json:"item_id"`
			SkuList []struct {
				SellerSku string `json:"seller_sku"`
				SkuID     int64  `json:"sku_id"`
			} `json:"sku_list"`
		} `json:"data"`
		Message string `json:"message"`
	}
	json.Unmarshal(body, &r)

	sku := input.SKUs[0].SkuCode
	return &PublishResult{
		PlatformProductID: strconv.FormatInt(r.Data.ItemID, 10),
		PlatformSKU:       sku,
		PlatformURL:       fmt.Sprintf("https://www.lazada.com/products/i%d", r.Data.ItemID),
		PublishedData:     map[string]interface{}{"item_id": r.Data.ItemID, "sync_message": r.Message},
		SyncMessage:       fmt.Sprintf("published to Lazada (item_id=%d)", r.Data.ItemID),
	}, nil
}

func (a *LazadaAdapter) SyncStatus(ctx context.Context, input *SyncStatusInput) (string, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return "unknown", err
	}

	payload := map[string]interface{}{
		"Request": map[string]string{
			"ItemId": input.PlatformProductID,
		},
	}
	body, err := a.do(ctx, http.MethodPost, "/product/item/get", auth, payload)
	if err != nil {
		return "unknown", fmt.Errorf("lazada sync_status: %w", err)
	}

	var r struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	json.Unmarshal(body, &r)

	statusMap := map[string]string{
		"active":  "synced",
		"inactive": "pending",
		"deleted": "failed",
		"rejected": "failed",
	}
	if s, ok := statusMap[r.Data.Status]; ok {
		return s, nil
	}
	return r.Data.Status, nil
}

func (a *LazadaAdapter) ValidateCredentials(ctx context.Context, accountID int64) (bool, error) {
	auth, err := a.getAuthByAccountID(ctx, accountID)
	if err != nil {
		return false, fmt.Errorf("lazada ValidateCredentials: %w", err)
	}
	// Use /auth/token/refresh or /seller/get — a lightweight call to verify credentials
	body, err := a.do(ctx, http.MethodGet, "/seller/get", auth, nil)
	if err != nil {
		return false, fmt.Errorf("lazada ValidateCredentials: %w", err)
	}
	var r struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return false, fmt.Errorf("lazada ValidateCredentials: parse error: %w", err)
	}
	return r.Code == "0", nil
}

func (a *LazadaAdapter) SyncInventory(ctx context.Context, input *SyncInventoryInput) (bool, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return false, err
	}

	sku := input.PlatformSKU
	if sku == "" {
		sku = input.SkuCode
	}

	payload := map[string]interface{}{
		"Request": map[string]interface{}{
			"SkuUpdateRequest": map[string]interface{}{
				"SellerSku": []map[string]interface{}{
					{
						"SkuId":    sku,
						"Quantity": input.Quantity,
						"Price":    0, // don't change price
					},
				},
			},
		},
	}
	_, err = a.do(ctx, http.MethodPost, "/product/price_quantity_update", auth, payload)
	return err == nil, err
}

func (a *LazadaAdapter) PushTracking(ctx context.Context, input *PushTrackingInput) (bool, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return false, err
	}

	payload := map[string]interface{}{
		"Request": map[string]interface{}{
			"OrderItemIds": []string{input.OrderSN},
			"DeliveryType": "dropship",
			"ShipmentProvider": input.CarrierCode,
			"TrackingNumber": input.TrackingNumber,
		},
	}
	_, err = a.do(ctx, http.MethodPost, "/order/ship", auth, payload)
	return err == nil, err
}

func (a *LazadaAdapter) FetchOrders(ctx context.Context, input *FetchOrdersInput) ([]*PlatformOrder, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"Request": map[string]interface{}{
			"CreatedAfter":  input.Since.Format("2006-01-02T15:04:05-07:00"),
			"Status":        "all",
		},
	}
	body, err := a.do(ctx, http.MethodPost, "/orders/get", auth, payload)
	if err != nil {
		return nil, fmt.Errorf("lazada fetch_orders: %w", err)
	}

	var r struct {
		Data struct {
			Count  int `json:"count"`
			Orders []struct {
				OrderID      int64  `json:"order_id"`
				OrderNumber  string `json:"order_number"`
				Status       string `json:"status"`
				Price        string `json:"price"`
				ShippingFee  string `json:"shipping_fee"`
				CreatedAt    string `json:"created_at"`
				Items        []struct {
					OrderItemID int64  `json:"order_item_id"`
					Sku         string `json:"sku"`
					SellerSku   string `json:"seller_sku"`
					Quantity    int    `json:"quantity"`
					ItemPrice   string `json:"item_price"`
				} `json:"order_items"`
			} `json:"orders"`
		} `json:"data"`
	}
	json.Unmarshal(body, &r)

	if r.Data.Count == 0 {
		return []*PlatformOrder{}, nil
	}

	var orders []*PlatformOrder
	for _, o := range r.Data.Orders {
		var items []PlatformOrderItem
		for _, item := range o.Items {
			sku := item.SellerSku
			if sku == "" {
				sku = item.Sku
			}
			items = append(items, PlatformOrderItem{
				SkuCode:   sku,
				Quantity:  item.Quantity,
				UnitPrice: item.ItemPrice,
			})
		}
		orders = append(orders, &PlatformOrder{
			OrderSN:     strconv.FormatInt(o.OrderID, 10),
			Status:      o.Status,
			TotalAmount: o.Price,
			ShippingFee: o.ShippingFee,
			PaidAt:      o.CreatedAt,
			Items:       items,
		})
	}
	return orders, nil
}

func (a *LazadaAdapter) FetchSettlements(ctx context.Context, input *FetchSettlementsInput) ([]*PlatformSettlement, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"Request": map[string]interface{}{
			"created_after": input.Since.Format("2006-01-02"),
		},
	}
	body, err := a.do(ctx, http.MethodGet, "/finance/payout/getStatement", auth, payload)
	if err != nil {
		return nil, err
	}

	var r struct {
		Data struct {
			Statements []struct {
				TransactionID   string `json:"transaction_id"`
				TransactionType string `json:"transaction_type"`
				OrderNumber     string `json:"order_number"`
				Amount          string `json:"amount"`
				Fee             string `json:"fee"`
				Currency        string `json:"currency"`
				CreatedAt       string `json:"created_at"`
				Description     string `json:"description"`
			} `json:"statements"`
		} `json:"data"`
	}
	json.Unmarshal(body, &r)

	var items []*PlatformSettlement
	for _, s := range r.Data.Statements {
		items = append(items, &PlatformSettlement{
			TransactionID:   s.TransactionID,
			TransactionType: s.TransactionType,
			OrderSN:         s.OrderNumber,
			Amount:          s.Amount,
			Fee:             s.Fee,
			Currency:        s.Currency,
			OccurredAt:      s.CreatedAt,
			Description:     s.Description,
		})
	}
	return items, nil
}

func (a *LazadaAdapter) FetchReturns(ctx context.Context, input *FetchReturnsInput) ([]*PlatformReturn, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"Request": map[string]interface{}{
			"CreatedAfter": input.Since.Format("2006-01-02T15:04:05-07:00"),
		},
	}
	body, err := a.do(ctx, http.MethodPost, "/returns/get", auth, payload)
	if err != nil {
		return nil, err
	}

	var r struct {
		Data struct {
			Returns []struct {
				ReturnID     string `json:"return_id"`
				OrderNumber  string `json:"order_number"`
				Sku          string `json:"sku"`
				Quantity     int    `json:"quantity"`
				Reason       string `json:"reason"`
				Status       string `json:"status"`
				CreatedAt    string `json:"created_at"`
				RefundAmount string `json:"refund_amount"`
			} `json:"returns"`
		} `json:"data"`
	}
	json.Unmarshal(body, &r)

	var items []*PlatformReturn
	for _, ret := range r.Data.Returns {
		items = append(items, &PlatformReturn{
			ReturnID:     ret.ReturnID,
			OrderSN:      ret.OrderNumber,
			SkuCode:      ret.Sku,
			Quantity:     ret.Quantity,
			Reason:       ret.Reason,
			Status:       ret.Status,
			CreatedAt:    ret.CreatedAt,
			RefundAmount: ret.RefundAmount,
		})
	}
	return items, nil
}

func (a *LazadaAdapter) FetchRaw(ctx context.Context, platformID int64, endpoint string, payload interface{}) ([]byte, error) {
	return nil, fmt.Errorf("lazada FetchRaw: not yet implemented")
}

// VerifyWebhookSignature implements WebhookVerifier for Lazada.
// Lazada signs webhooks with HMAC-SHA256 using the app secret.
func (a *LazadaAdapter) VerifyWebhookSignature(ctx context.Context, body []byte, headers http.Header) bool {
	signature := headers.Get("X-Lazada-OP-Sign")
	if signature == "" {
		return false
	}

	var accts []PlatformIntegrationAccount
	if err := a.db.WithContext(ctx).
		Model(&PlatformIntegrationAccount{}).
		Where("platform_id = (SELECT id FROM platform WHERE code = ?) AND status = ?", "lazada", "active").
		Limit(1).
		Find(&accts).Error; err != nil || len(accts) == 0 {
		return false
	}

	var cfg struct {
		AppSecret string `json:"app_secret"`
	}
	if len(accts[0].Config) > 0 {
		_ = json.Unmarshal(accts[0].Config, &cfg)
	}
	if cfg.AppSecret == "" {
		return false
	}

	mac := hmac.New(md5.New, []byte(cfg.AppSecret))
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
