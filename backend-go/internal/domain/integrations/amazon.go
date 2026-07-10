package integrations

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	amazonLWAEndpoint     = "https://api.amazon.com/auth/o2/token"
	amazonProdURLTemplate = "https://sellingpartnerapi-%s.amazon.com"
	amazonSandboxURL      = "https://sellingpartnerapi-sandbox.amazon.com"
	amazonService         = "execute-api"
	amazonDefaultTimeout  = 30 * time.Second
)

// AmazonAdapter implements PlatformAdapter for Amazon SP-API / Selling Partner API.
type AmazonAdapter struct {
	httpClient *http.Client
	db         *gorm.DB
	logger     *zap.Logger
}

// NewAmazonAdapter creates a new Amazon adapter.
func NewAmazonAdapter(db *gorm.DB, logger *zap.Logger) *AmazonAdapter {
	return &AmazonAdapter{
		httpClient: &http.Client{Timeout: amazonDefaultTimeout},
		db:         db,
		logger:     logger,
	}
}

// amazonConfig maps the per-account JSON config stored in PlatformIntegrationAccount.Config.
type amazonConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AWSAccessKey string `json:"aws_access_key_id"`
	AWSSecretKey string `json:"aws_secret_access_key"`
	AWSRegion    string `json:"aws_region"`
	// ponytail: marketplace_id is stored in config for symmetry with other
	// platforms; callers pass it in payloads when needed.
}

// amazonAuth holds resolved credentials for one request batch.
type amazonAuth struct {
	AccessToken  string
	RefreshToken string
	ClientID     string
	ClientSecret string
	Creds        aws.Credentials
	Region       string
	BaseURL      string
}

// getAuth resolves credentials for a platform integration account from the database.
// Auto-refreshes the LWA access token when expired.
func (a *AmazonAdapter) getAuth(ctx context.Context, platformID int64, mode ExecutionMode) (*amazonAuth, error) {
	var accts []PlatformIntegrationAccount
	if err := a.db.WithContext(ctx).
		Where("platform_id = ? AND status = ?", platformID, "active").
		Limit(1).
		Find(&accts).Error; err != nil {
		return nil, fmt.Errorf("amazon getAuth: %w", err)
	}
	if len(accts) == 0 {
		return nil, fmt.Errorf("amazon getAuth: no active account for platform_id=%d", platformID)
	}
	acct := accts[0]

	var cfg amazonConfig
	if len(acct.Config) > 0 {
		json.Unmarshal(acct.Config, &cfg)
	}
	if acct.RefreshToken == "" {
		return nil, fmt.Errorf("amazon getAuth: account %d has empty refresh_token", acct.ID)
	}
	if cfg.AWSRegion == "" {
		return nil, fmt.Errorf("amazon getAuth: account %d missing aws_region in config", acct.ID)
	}

	// Auto-refresh LWA access token if expired or absent.
	accessToken := acct.AccessToken
	if accessToken == "" || acct.TokenExpiresAt == nil || time.Now().After(*acct.TokenExpiresAt) {
		newToken, expiresIn, err := a.refreshLWAToken(ctx, cfg.ClientID, cfg.ClientSecret, acct.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("amazon getAuth: refresh LWA: %w", err)
		}
		accessToken = newToken
		expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
		// ponytail: best-effort persist — a failed write still lets this
		// request proceed; the next request will retry the refresh.
		a.db.Model(&acct).Updates(map[string]interface{}{
			"access_token":     accessToken,
			"token_expires_at": &expiresAt,
		})
	}

	baseURL := fmt.Sprintf(amazonProdURLTemplate, cfg.AWSRegion)
	if mode == ExecutionModeSandbox {
		baseURL = amazonSandboxURL
	}

	return &amazonAuth{
		AccessToken:  accessToken,
		RefreshToken: acct.RefreshToken,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Creds: aws.Credentials{
			AccessKeyID:     cfg.AWSAccessKey,
			SecretAccessKey: cfg.AWSSecretKey,
		},
		Region:  cfg.AWSRegion,
		BaseURL: baseURL,
	}, nil
}

// refreshLWAToken exchanges a refresh token for a new LWA access token.
func (a *AmazonAdapter) refreshLWAToken(ctx context.Context, clientID, clientSecret, refreshToken string) (string, int, error) {
	payload := map[string]string{
		"grant_type":    "refresh_token",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"refresh_token": refreshToken,
	}
	b, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, amazonLWAEndpoint, bytes.NewReader(b))
	if err != nil {
		return "", 0, fmt.Errorf("amazon LWA request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("amazon LWA: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("amazon LWA read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("amazon LWA: HTTP %d %s", resp.StatusCode, truncStr(string(body), 300))
	}

	var r struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return "", 0, fmt.Errorf("amazon LWA parse: %w", err)
	}
	return r.AccessToken, r.ExpiresIn, nil
}

// FetchRaw performs a low-level HTTP request to the Amazon SP-API.
// It resolves auth from the DB, auto-refreshes the LWA token if expired,
// signs the request with AWS SigV4, and handles a single 401 retry.
func (a *AmazonAdapter) FetchRaw(ctx context.Context, platformID int64, endpoint string, payload interface{}) ([]byte, error) {
	mode := ExecutionModeFromCtx(ctx)
	auth, err := a.getAuth(ctx, platformID, mode)
	if err != nil {
		return nil, err
	}
	return a.do(ctx, http.MethodPost, endpoint, auth, payload)
}

// do is the internal HTTP helper. It signs each request with AWS SigV4 and
// retries once on 401 (stale LWA token).
// ponytail: retry loop bounded at 2 — only retries on 401.
func (a *AmazonAdapter) do(ctx context.Context, method, path string, auth *amazonAuth, payload interface{}) ([]byte, error) {
	var bodyBytes []byte
	if payload != nil {
		var err error
		bodyBytes, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("amazon marshal: %w", err)
		}
	}
	payloadHash := sha256Hex(bodyBytes)

	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, auth.BaseURL+path, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("amazon request: %w", err)
		}
		req.Header.Set("x-amz-access-token", auth.AccessToken)
		req.Header.Set("Content-Type", "application/json")

		signer := v4.NewSigner()
		if err := signer.SignHTTP(ctx, auth.Creds, req, payloadHash, amazonService, auth.Region, time.Now()); err != nil {
			return nil, fmt.Errorf("amazon sign: %w", err)
		}

		resp, err := a.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("amazon %s: %w", path, err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("amazon read %s: %w", path, readErr)
		}

		if resp.StatusCode == http.StatusUnauthorized && attempt == 0 {
			newToken, _, refreshErr := a.refreshLWAToken(ctx, auth.ClientID, auth.ClientSecret, auth.RefreshToken)
			if refreshErr != nil {
				return nil, fmt.Errorf("amazon %s: HTTP 401 and LWA refresh failed: %w", path, refreshErr)
			}
			auth.AccessToken = newToken
			continue // retry once with fresh token
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, fmt.Errorf("amazon %s: rate limited (retry-after: %s)", path, resp.Header.Get("Retry-After"))
		}

		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("amazon %s: HTTP %d %s", path, resp.StatusCode, truncStr(string(body), 300))
		}
		return body, nil
	}
	return nil, fmt.Errorf("amazon %s: unexpected retry exhaustion", path)
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// --- Stub methods (keep existing return signatures) ---

func (a *AmazonAdapter) Publish(ctx context.Context, input *PublishInput) (*PublishResult, error) {
	if len(input.SKUs) == 0 {
		return nil, fmt.Errorf("amazon publish: no SKUs")
	}
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return nil, err
	}

	// Amazon uses the Listings API (feed-based or direct) to create/update items.
	// The direct REST API: PUT /listings/2021-08-01/items/{sellerId}/{sku}
	sku := input.SKUs[0].SkuCode
	path := fmt.Sprintf("/listings/2021-08-01/items/%s/%s", auth.SellerID, sku)

	payload := map[string]interface{}{
		"productType": "PRODUCT",
		"attributes": map[string]interface{}{
			"item_name": []map[string]interface{}{
				{"value": input.ProductName, "language_tag": "en_US"},
			},
			"product_description": []map[string]interface{}{
				{"value": input.Description, "language_tag": "en_US"},
			},
			"brand": []map[string]interface{}{
				{"value": fmt.Sprintf("%d", input.BrandID)},
			},
			"main_image": []map[string]interface{}{
				{"value": input.MainImage},
			},
			"package_weight": []map[string]interface{}{
				{"value": fmt.Sprintf("%.2f", input.PackageWeight), "unit": "kilograms"},
			},
			"package_dimensions": []map[string]interface{}{
				{
					"height": map[string]interface{}{"value": input.PackageHeight, "unit": "centimeters"},
					"width":  map[string]interface{}{"value": input.PackageWidth, "unit": "centimeters"},
					"length": map[string]interface{}{"value": input.PackageLength, "unit": "centimeters"},
				},
			},
			"purchasable_offer": []map[string]interface{}{
				{
					"currency": "USD",
					"our_price": []map[string]interface{}{
						{
							"schedule": []map[string]interface{}{
								{
									"value_withholding_tax": map[string]interface{}{"value": input.Prices[input.SKUs[0].SkuID]},
								},
							},
						},
					},
				},
			},
			"fulfillment_availability": []map[string]interface{}{
				{
					"fulfillment_channel_code": "DEFAULT",
					"quantity":                input.Inventories[input.SKUs[0].SkuID],
				},
			},
		},
	}

	body, err := a.do(ctx, http.MethodPut, path, auth, payload)
	if err != nil {
		return nil, fmt.Errorf("amazon publish: %w", err)
	}

	var r struct {
		Sku string `json:"sku"`
		Status string `json:"status"`
	}
	json.Unmarshal(body, &r)

	return &PublishResult{
		PlatformProductID: sku,
		PlatformSKU:       sku,
		PlatformURL:       fmt.Sprintf("https://sellercentral.amazon.com/product/%s", sku),
		PublishedData:     map[string]interface{}{"status": r.Status, "sku": sku},
		SyncMessage:       fmt.Sprintf("published to Amazon (sku=%s)", sku),
	}, nil
}

func (a *AmazonAdapter) SyncStatus(ctx context.Context, input *SyncStatusInput) (string, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return "unknown", err
	}

	sku := strings.TrimPrefix(input.PlatformProductID, "amazon-")
	path := fmt.Sprintf("/listings/2021-08-01/items/%s/%s?marketplaceIds=%s", auth.SellerID, sku, auth.MarketplaceID)

	body, err := a.do(ctx, http.MethodGet, path, auth, nil)
	if err != nil {
		return "unknown", fmt.Errorf("amazon sync_status: %w", err)
	}

	var r struct {
		Sku      string `json:"sku"`
		Status   string `json:"status"`
		Sumaries []struct {
			Status string `json:"status"`
		} `json:"summaries"`
	}
	json.Unmarshal(body, &r)

	if len(r.Sumaries) > 0 {
		statusMap := map[string]string{
			"ACTIVE":   "synced",
			"INACTIVE": "pending",
			"CREATING": "in_progress",
		}
		if s, ok := statusMap[r.Sumaries[0].Status]; ok {
			return s, nil
		}
		return r.Sumaries[0].Status, nil
	}
	return r.Status, nil
}

func (a *AmazonAdapter) ValidateCredentials(ctx context.Context, accountID int64) (bool, error) {
	auth, err := a.getAuthByAccountID(ctx, accountID)
	if err != nil {
		return false, fmt.Errorf("amazon ValidateCredentials: %w", err)
	}
	// Lightweight call: get marketplace participation
	path := "/authorization/v1/authorizationCode"
	payload := map[string]interface{}{
		"sellingPartnerId": auth.SellerID,
		"marketplaceIds":   []string{auth.MarketplaceID},
	}
	_, err = a.do(ctx, http.MethodPost, path, auth, payload)
	if err != nil {
		return false, fmt.Errorf("amazon ValidateCredentials: %w", err)
	}
	return true, nil
}

func (a *AmazonAdapter) SyncInventory(ctx context.Context, input *SyncInventoryInput) (bool, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return false, err
	}

	sku := input.PlatformSKU
	if sku == "" {
		sku = input.SkuCode
	}

	path := fmt.Sprintf("/listings/2021-08-01/items/%s/%s", auth.SellerID, sku)
	payload := map[string]interface{}{
		"productType": "PRODUCT",
		"attributes": map[string]interface{}{
			"fulfillment_availability": []map[string]interface{}{
				{
					"fulfillment_channel_code": "DEFAULT",
					"quantity":                input.Quantity,
				},
			},
		},
	}
	_, err = a.do(ctx, http.MethodPatch, path, auth, payload)
	return err == nil, err
}

func (a *AmazonAdapter) PushTracking(ctx context.Context, input *PushTrackingInput) (bool, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return false, err
	}

	payload := map[string]interface{}{
		"orderId":       input.OrderSN,
		"carrierCode":   input.CarrierCode,
		"trackingNumber": input.TrackingNumber,
	}
	_, err = a.do(ctx, http.MethodPost, "/orders/v0/orders/shipment", auth, payload)
	return err == nil, err
}

func (a *AmazonAdapter) FetchOrders(ctx context.Context, input *FetchOrdersInput) ([]*PlatformOrder, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf(
		"/orders/v0/orders?MarketplaceIds=%s&CreatedAfter=%s&MaxResultsPerPage=100",
		auth.MarketplaceID,
		input.Since.Format("2006-01-02T15:04:05.000Z"),
	)

	body, err := a.do(ctx, http.MethodGet, path, auth, nil)
	if err != nil {
		return nil, fmt.Errorf("amazon fetch_orders: %w", err)
	}

	var r struct {
		Payload struct {
			Orders []struct {
				AmazonOrderID string `json:"amazon_order_id"`
				OrderStatus   string `json:"order_status"`
				OrderTotal    struct {
					Amount       string `json:"amount"`
					CurrencyCode string `json:"currency_code"`
				} `json:"order_total"`
				PurchaseDate string `json:"purchase_date"`
				ShippingAddress struct {
					Name  string `json:"name"`
					Phone string `json:"phone"`
				} `json:"shipping_address"`
			} `json:"orders"`
		} `json:"payload"`
	}
	json.Unmarshal(body, &r)

	if len(r.Payload.Orders) == 0 {
		return []*PlatformOrder{}, nil
	}

	var orders []*PlatformOrder
	for _, o := range r.Payload.Orders {
		items, _ := a.fetchOrderItems(ctx, auth, o.AmazonOrderID)
		orders = append(orders, &PlatformOrder{
			OrderSN:         o.AmazonOrderID,
			Status:          o.OrderStatus,
			TotalAmount:     o.OrderTotal.Amount,
			RecipientName:   o.ShippingAddress.Name,
			RecipientPhone:  o.ShippingAddress.Phone,
			PaidAt:          o.PurchaseDate,
			Items:           items,
		})
	}
	return orders, nil
}

// fetchOrderItems retrieves line items for a specific Amazon order.
func (a *AmazonAdapter) fetchOrderItems(ctx context.Context, auth *amazonAuth, orderID string) ([]PlatformOrderItem, error) {
	path := fmt.Sprintf("/orders/v0/orders/%s/orderItems", orderID)
	body, err := a.do(ctx, http.MethodGet, path, auth, nil)
	if err != nil {
		return nil, err
	}

	var r struct {
		Payload struct {
			Items []struct {
				SellerSKU     string `json:"seller_sku"`
				Quantity      int    `json:"quantity_ordered"`
				ItemPrice     struct {
					Amount string `json:"amount"`
				} `json:"item_price"`
			} `json:"order_items"`
		} `json:"payload"`
	}
	json.Unmarshal(body, &r)

	var items []PlatformOrderItem
	for _, item := range r.Payload.Items {
		items = append(items, PlatformOrderItem{
			SkuCode:   item.SellerSKU,
			Quantity:  item.Quantity,
			UnitPrice: item.ItemPrice.Amount,
		})
	}
	return items, nil
}

func (a *AmazonAdapter) FetchSettlements(ctx context.Context, input *FetchSettlementsInput) ([]*PlatformSettlement, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return nil, err
	}

	// Amazon SP-API Financials API: GET /finances/v0/financialEvents
	path := fmt.Sprintf(
		"/finances/v0/financialEvents?PostedAfter=%s&MaxResultsPerPage=100",
		input.Since.Format("2006-01-02T15:04:05.000Z"),
	)

	body, err := a.do(ctx, http.MethodGet, path, auth, nil)
	if err != nil {
		return nil, err
	}

	var r struct {
		Payload struct {
			Events []struct {
				TransactionID string `json:"transaction_id"`
				EventType     string `json:"financial_event_type"`
				PostedDate    string `json:"posted_date"`
				TotalAmount   struct {
					CurrencyAmount string `json:"currency_amount"`
					CurrencyCode   string `json:"currency_code"`
				} `json:"total_amount"`
			} `json:"financial_events"`
		} `json:"payload"`
	}
	json.Unmarshal(body, &r)

	var items []*PlatformSettlement
	for _, e := range r.Payload.Events {
		items = append(items, &PlatformSettlement{
			TransactionID:   e.TransactionID,
			TransactionType: e.EventType,
			Amount:          e.TotalAmount.CurrencyAmount,
			Currency:        e.TotalAmount.CurrencyCode,
			OccurredAt:      e.PostedDate,
		})
	}
	return items, nil
}

func (a *AmazonAdapter) FetchReturns(ctx context.Context, input *FetchReturnsInput) ([]*PlatformReturn, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return nil, err
	}

	// Amazon uses the FBA Returns API via the Fulfillment Outbound API
	path := fmt.Sprintf(
		"/returns/v2022-04-01/returns?marketplaceId=%s&updatedAfter=%s",
		auth.MarketplaceID,
		input.Since.Format("2006-01-02"),
	)

	body, err := a.do(ctx, http.MethodGet, path, auth, nil)
	if err != nil {
		// Fall back to fetching orders with return status
		items, fallbackErr := a.fetchReturnsFromOrders(ctx, auth, input)
		if fallbackErr != nil {
			return nil, err // return the primary error
		}
		return items, nil
	}

	var r struct {
		Returns []struct {
			ReturnID      string `json:"return_id"`
			OrderID       string `json:"order_id"`
			Sku           string `json:"sku"`
			Quantity      int    `json:"quantity"`
			ReturnReason  string `json:"return_reason"`
			Status        string `json:"status"`
			CreatedAt     string `json:"created_at"`
			RefundAmount  string `json:"refund_amount"`
		} `json:"returns"`
	}
	json.Unmarshal(body, &r)

	var items []*PlatformReturn
	for _, ret := range r.Returns {
		items = append(items, &PlatformReturn{
			ReturnID:     ret.ReturnID,
			OrderSN:      ret.OrderID,
			SkuCode:      ret.Sku,
			Quantity:     ret.Quantity,
			Reason:       ret.ReturnReason,
			Status:       ret.Status,
			CreatedAt:    ret.CreatedAt,
			RefundAmount: ret.RefundAmount,
		})
	}
	return items, nil
}

// fetchReturnsFromOrders is a fallback: scan recent orders for cancelled/returned items.
func (a *AmazonAdapter) fetchReturnsFromOrders(ctx context.Context, auth *amazonAuth, input *FetchReturnsInput) ([]*PlatformReturn, error) {
	orders, err := a.FetchOrders(ctx, &FetchOrdersInput{PlatformID: input.PlatformID, Since: input.Since})
	if err != nil {
		return nil, err
	}

	var items []*PlatformReturn
	for _, o := range orders {
		if o.Status == "Canceled" || o.Status == "Returned" {
			for _, item := range o.Items {
				items = append(items, &PlatformReturn{
					ReturnID: fmt.Sprintf("amz-%s-%s", o.OrderSN, item.SkuCode),
					OrderSN:  o.OrderSN,
					SkuCode:  item.SkuCode,
					Quantity: item.Quantity,
					Reason:   "return",
					Status:   "initiated",
					RefundAmount: o.TotalAmount,
				})
			}
		}
	}
	return items, nil
}

// VerifyWebhookSignature implements WebhookVerifier for Amazon.
// Amazon SP-API uses SQS for notifications, but direct webhooks use HMAC-SHA256
// with the client secret (for developer notifications via SP-API Notifications API).
func (a *AmazonAdapter) VerifyWebhookSignature(ctx context.Context, body []byte, headers http.Header) bool {
	// Amazon's SP-API notification webhooks use HMAC-SHA256 with the client secret
	signature := headers.Get("x-amz-notification-signature")
	if signature == "" {
		// Amazon SNS notifications use a different signature format
		signature = headers.Get("x-amz-sns-signature")
	}
	if signature == "" {
		return false
	}

	var accts []PlatformIntegrationAccount
	if err := a.db.WithContext(ctx).
		Model(&PlatformIntegrationAccount{}).
		Where("platform_id = (SELECT id FROM platform WHERE code = ?) AND status = ?", "amazon", "active").
		Limit(1).
		Find(&accts).Error; err != nil || len(accts) == 0 {
		return false
	}

	var cfg struct {
		ClientSecret string `json:"client_secret"`
	}
	if len(accts[0].Config) > 0 {
		_ = json.Unmarshal(accts[0].Config, &cfg)
	}
	if cfg.ClientSecret == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(cfg.ClientSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// GetServiceStatus checks the Amazon SP-API service health.
func (a *AmazonAdapter) GetServiceStatus(ctx context.Context, platformID int64) (string, error) {
	auth, err := a.getAuth(ctx, platformID)
	if err != nil {
		return "unknown", err
	}
	body, err := a.do(ctx, http.MethodGet, "/sellers/v1/marketplaceParticipations", auth, nil)
	if err != nil {
		return "unavailable", err
	}
	var r struct {
		Payload []struct {
			Marketplace struct {
				Name string `json:"name"`
			} `json:"marketplace"`
		} `json:"payload"`
	}
	json.Unmarshal(body, &r)
	if len(r.Payload) > 0 {
		return "active", nil
	}
	return "available", nil
}
