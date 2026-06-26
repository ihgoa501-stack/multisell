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
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Shopee Open Platform v2 base URL.
const ShopeeBaseURL = "https://partner.shopeemobile.com"

// ShopeeRealAdapter implements PlatformAdapter for Shopee Open Platform v2
// with real API calls, credential management from PlatformIntegrationAccount,
// and full order sync.
type ShopeeRealAdapter struct {
	httpClient *http.Client
	db         *gorm.DB
	logger     *zap.Logger
	partnerID  int64
	partnerKey string
}

// NewShopeeRealAdapter creates a new ShopeeRealAdapter.
// Partner credentials are read from environment variables SHOPEE_PARTNER_ID
// and SHOPEE_PARTNER_KEY if not explicitly provided.
func NewShopeeRealAdapter(db *gorm.DB, logger *zap.Logger) *ShopeeRealAdapter {
	partnerID, _ := strconv.ParseInt(os.Getenv("SHOPEE_PARTNER_ID"), 10, 64)
	partnerKey := os.Getenv("SHOPEE_PARTNER_KEY")

	return &ShopeeRealAdapter{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		db:         db,
		logger:     logger,
		partnerID:  partnerID,
		partnerKey: partnerKey,
	}
}

// sign computes the HMAC-SHA256 signature for a Shopee API request.
// base_string = partner_id + path + timestamp + access_token + shop_id
func (a *ShopeeRealAdapter) sign(partnerID int64, apiKey, path, accessToken string, shopID int64, timestamp int64) string {
	base := fmt.Sprintf("%d%s%d%s%d", partnerID, path, timestamp, accessToken, shopID)
	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write([]byte(base))
	return hex.EncodeToString(mac.Sum(nil))
}

// buildSignedURL constructs the full URL with Shopee auth query parameters.
func (a *ShopeeRealAdapter) buildSignedURL(baseURL, path, accessToken string, partnerID, shopID, timestamp int64) string {
	sign := a.sign(partnerID, a.partnerKey, path, accessToken, shopID, timestamp)
	return fmt.Sprintf("%s%s?partner_id=%d&timestamp=%d&sign=%s&access_token=%s&shop_id=%d",
		baseURL, path, partnerID, timestamp, sign, accessToken, shopID)
}

// resolveAuth fetches the PlatformIntegrationAccount from the database and
// resolves the Shopee auth parameters (partner_id, partner_key, access_token,
// shop_id). Falls back to the adapter-level partner credentials from env vars;
// per-account overrides are read from the Config JSON column.
func (a *ShopeeRealAdapter) resolveAuth(ctx context.Context, accountID int64) (*shopeeAuth, error) {
	var acct PlatformIntegrationAccount
	if err := a.db.WithContext(ctx).First(&acct, accountID).Error; err != nil {
		return nil, fmt.Errorf("shopee account %d not found: %w", accountID, err)
	}

	partnerID := a.partnerID
	partnerKey := a.partnerKey

	// Allow per-account override of partner credentials via Config JSON.
	if acct.Config != nil {
		var cfg struct {
			PartnerID  int64  `json:"partner_id"`
			PartnerKey string `json:"partner_key"`
		}
		if json.Unmarshal(acct.Config, &cfg) == nil {
			if cfg.PartnerID != 0 {
				partnerID = cfg.PartnerID
			}
			if cfg.PartnerKey != "" {
				partnerKey = cfg.PartnerKey
			}
		}
	}

	if partnerID == 0 || partnerKey == "" {
		return nil, fmt.Errorf("shopee partner credentials not configured for account %d", accountID)
	}

	shopID, _ := strconv.ParseInt(acct.AccountID, 10, 64)

	return &shopeeAuth{
		PartnerID:   partnerID,
		APIKey:      partnerKey,
		AccessToken: acct.AccessToken,
		ShopID:      shopID,
		BaseURL:     ShopeeBaseURL,
	}, nil
}

// do performs an authenticated HTTP request to the Shopee Open Platform v2 API.
// It constructs the signed URL with query parameters, sends the request, and
// parses the Shopee response envelope (error/message at top level).
func (a *ShopeeRealAdapter) do(ctx context.Context, method, path string, auth *shopeeAuth, payload interface{}) ([]byte, error) {
	if auth.AccessToken == "" {
		return nil, fmt.Errorf("shopee %s: access token is empty", path)
	}

	ts := time.Now().Unix()
	url := a.buildSignedURL(auth.BaseURL, path, auth.AccessToken, auth.PartnerID, auth.ShopID, ts)

	var bodyReader io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("shopee marshal: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("shopee req: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("shopee %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("shopee read %s: %w", path, err)
	}

	// Check Shopee error envelope (top-level "error" / "message" fields).
	var shopeeResp struct {
		Error   int    `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &shopeeResp) == nil && shopeeResp.Error != 0 {
		return nil, fmt.Errorf("shopee %s error %d: %s", path, shopeeResp.Error, shopeeResp.Message)
	}
	return body, nil
}

// checkTokenExpiry returns true if the account's token is expired or about to
// expire within the next 5 minutes.
func (a *ShopeeRealAdapter) checkTokenExpiry(acct *PlatformIntegrationAccount) bool {
	if acct.TokenExpiresAt == nil {
		return true // no expiry known — assume expired
	}
	return time.Now().After(acct.TokenExpiresAt.Add(-5 * time.Minute))
}

// ExchangeCode exchanges an authorization code for access and refresh tokens
// via POST /api/v2/auth/access_token/get. Stores the tokens in the account.
func (a *ShopeeRealAdapter) ExchangeCode(ctx context.Context, accountID int64, authCode string) error {
	if a.partnerID == 0 || a.partnerKey == "" {
		return fmt.Errorf("shopee partner credentials not configured (set SHOPEE_PARTNER_ID and SHOPEE_PARTNER_KEY)")
	}

	ts := time.Now().Unix()
	// Auth endpoint sign: partner_id + path + timestamp (no access_token, shop_id=0)
	base := fmt.Sprintf("%d%s%d%s%d", a.partnerID, "/api/v2/auth/access_token/get", ts, "", 0)
	mac := hmac.New(sha256.New, []byte(a.partnerKey))
	mac.Write([]byte(base))
	sign := hex.EncodeToString(mac.Sum(nil))

	payload := map[string]interface{}{
		"code":       authCode,
		"partner_id": a.partnerID,
		"shop_id":    0,
		"sign":       sign,
		"timestamp":  ts,
	}

	url := a.partnerBaseURL() + "/api/v2/auth/access_token/get?" +
		fmt.Sprintf("partner_id=%d&timestamp=%d&sign=%s", a.partnerID, ts, sign)

	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("shopee exchange marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("shopee exchange req: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("shopee exchange do: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("shopee exchange read: %w", err)
	}

	var r struct {
		Error   int    `json:"error"`
		Message string `json:"message"`
		Response *struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpireIn     int64  `json:"expire_in"`
			ShopID       int64  `json:"shop_id"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return fmt.Errorf("shopee exchange parse: %w", err)
	}
	if r.Error != 0 {
		return fmt.Errorf("shopee exchange error %d: %s", r.Error, r.Message)
	}
	if r.Response == nil || r.Response.AccessToken == "" {
		return fmt.Errorf("shopee exchange: empty access_token in response")
	}

	expiresAt := time.Now().Add(time.Duration(r.Response.ExpireIn) * time.Second)

	return a.db.WithContext(ctx).Model(&PlatformIntegrationAccount{}).
		Where("id = ?", accountID).
		Updates(map[string]interface{}{
			"account_id":       strconv.FormatInt(r.Response.ShopID, 10),
			"access_token":     r.Response.AccessToken,
			"refresh_token":    r.Response.RefreshToken,
			"token_expires_at": expiresAt,
		}).Error
}

// RefreshToken refreshes the access token via POST /api/v2/auth/refresh_token/get.
func (a *ShopeeRealAdapter) RefreshToken(ctx context.Context, accountID int64) error {
	var acct PlatformIntegrationAccount
	if err := a.db.WithContext(ctx).First(&acct, accountID).Error; err != nil {
		return fmt.Errorf("shopee refresh: account %d not found: %w", accountID, err)
	}
	if acct.RefreshToken == "" {
		return fmt.Errorf("shopee refresh: account %d has no refresh token", accountID)
	}

	partnerID := a.partnerID
	partnerKey := a.partnerKey

	// Resolve per-account override if configured.
	if acct.Config != nil {
		var cfg struct {
			PartnerID  int64  `json:"partner_id"`
			PartnerKey string `json:"partner_key"`
		}
		if json.Unmarshal(acct.Config, &cfg) == nil {
			if cfg.PartnerID != 0 {
				partnerID = cfg.PartnerID
			}
			if cfg.PartnerKey != "" {
				partnerKey = cfg.PartnerKey
			}
		}
	}
	if partnerID == 0 || partnerKey == "" {
		return fmt.Errorf("shopee refresh: partner credentials not configured")
	}

	shopID, _ := strconv.ParseInt(acct.AccountID, 10, 64)

	ts := time.Now().Unix()
	base := fmt.Sprintf("%d%s%d%s%d", partnerID, "/api/v2/auth/refresh_token/get", ts, acct.RefreshToken, shopID)
	mac := hmac.New(sha256.New, []byte(partnerKey))
	mac.Write([]byte(base))
	sign := hex.EncodeToString(mac.Sum(nil))

	payload := map[string]interface{}{
		"refresh_token": acct.RefreshToken,
		"partner_id":    partnerID,
		"shop_id":       shopID,
	}

	url := a.partnerBaseURL() + "/api/v2/auth/refresh_token/get?" +
		fmt.Sprintf("partner_id=%d&timestamp=%d&sign=%s&access_token=%s&shop_id=%d",
			partnerID, ts, sign, acct.RefreshToken, shopID)

	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("shopee refresh marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("shopee refresh req: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("shopee refresh do: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("shopee refresh read: %w", err)
	}

	var r struct {
		Error   int    `json:"error"`
		Message string `json:"message"`
		Response *struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpireIn     int64  `json:"expire_in"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return fmt.Errorf("shopee refresh parse: %w", err)
	}
	if r.Error != 0 {
		return fmt.Errorf("shopee refresh error %d: %s", r.Error, r.Message)
	}
	if r.Response == nil || r.Response.AccessToken == "" {
		return fmt.Errorf("shopee refresh: empty access_token in response")
	}

	expiresAt := time.Now().Add(time.Duration(r.Response.ExpireIn) * time.Second)

	return a.db.WithContext(ctx).Model(&PlatformIntegrationAccount{}).
		Where("id = ?", accountID).
		Updates(map[string]interface{}{
			"access_token":     r.Response.AccessToken,
			"refresh_token":    r.Response.RefreshToken,
			"token_expires_at": expiresAt,
		}).Error
}

// partnerBaseURL returns the default Shopee partner URL.
func (a *ShopeeRealAdapter) partnerBaseURL() string {
	return ShopeeBaseURL
}

// ──────────────────────────────────────────────
//  Shopee order status → LingMirror status map
// ──────────────────────────────────────────────

var shopeeStatusMap = map[string]string{
	"UNPAID":              "pending",
	"READY_TO_SHIP":       "confirmed",
	"PROCESSED":           "processing",
	"SHIPPED":             "shipped",
	"COMPLETED":           "completed",
	"CANCELLED":           "cancelled",
	"IN_CANCEL":           "cancelling",
	"TO_CONFIRM_RECEIVE":  "shipped",
	"TO_RETURN":           "returning",
	"INVOICE_PENDING":     "pending",
}

// mapStatus converts a Shopee order_status to the LingMirror internal status.
func mapStatus(shopeeStatus string) string {
	if s, ok := shopeeStatusMap[shopeeStatus]; ok {
		return s
	}
	return strings.ToLower(shopeeStatus)
}

// ──────────────────────────────────────────────
//  PlatformAdapter interface implementation
// ──────────────────────────────────────────────

// Publish publishes a product to the Shopee platform.
func (a *ShopeeRealAdapter) Publish(ctx context.Context, input *PublishInput) (*PublishResult, error) {
	auth, err := a.resolveAuth(ctx, input.AccountID)
	if err != nil {
		return nil, err
	}
	if len(input.SKUs) == 0 {
		return nil, fmt.Errorf("shopee publish: no SKUs")
	}

	images := input.Images
	if len(images) == 0 && input.MainImage != "" {
		images = []string{input.MainImage}
	}
	if images == nil {
		images = []string{}
	}

	payload := map[string]interface{}{
		"item_name":      input.ProductName,
		"description":    input.Description,
		"category_id":    input.CategoryID,
		"image":          images,
		"item_sku":       input.SKUs[0].SkuCode,
		"price":          input.Prices[input.SKUs[0].SkuID],
		"stock":          input.Inventories[input.SKUs[0].SkuID],
		"weight":         fmt.Sprintf("%.2f", input.PackageWeight),
		"package_length": int(input.PackageLength),
		"package_width":  int(input.PackageWidth),
		"package_height": int(input.PackageHeight),
	}

	body, err := a.do(ctx, http.MethodPost, "/api/v2/product/add", auth, payload)
	if err != nil {
		return nil, err
	}

	var r struct {
		Response struct {
			ItemID int64 `json:"item_id"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("shopee publish parse: %w", err)
	}

	return &PublishResult{
		PlatformProductID: fmt.Sprintf("shopee-%d", r.Response.ItemID),
		PlatformSKU:       input.SKUs[0].SkuCode,
		PlatformURL:       fmt.Sprintf("https://shopee.ph/product/%d/", r.Response.ItemID),
		PublishedData:     map[string]interface{}{"item_id": r.Response.ItemID},
		SyncMessage:       "published to Shopee",
	}, nil
}

// SyncStatus checks the current publish status of a product on Shopee.
func (a *ShopeeRealAdapter) SyncStatus(ctx context.Context, input *SyncStatusInput) (string, error) {
	// SyncStatusInput does not carry AccountID, so we use GetItemList endpoint
	// which only needs product-level auth.
	auth := &shopeeAuth{BaseURL: ShopeeBaseURL}

	payload := map[string]interface{}{
		"item_id":   nil,
		"item_sku":  []string{input.PlatformProductID},
	}

	body, err := a.do(ctx, http.MethodPost, "/api/v2/product/get_item_list", auth, payload)
	if err != nil {
		return "unknown", err
	}

	var r struct {
		Response struct {
			ItemList []struct {
				Status string `json:"status"`
			} `json:"item_list"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return "unknown", nil
	}
	if len(r.Response.ItemList) == 0 {
		return "unknown", nil
	}

	m := map[string]string{"NORMAL": "synced", "UNLIST": "pending", "BANNED": "failed"}
	if s, ok := m[r.Response.ItemList[0].Status]; ok {
		return s, nil
	}
	return r.Response.ItemList[0].Status, nil
}

// ValidateCredentials verifies that the stored Shopee tokens are still valid
// by attempting a lightweight authenticated call.
func (a *ShopeeRealAdapter) ValidateCredentials(ctx context.Context, accountID int64) (bool, error) {
	auth, err := a.resolveAuth(ctx, accountID)
	if err != nil {
		return false, err
	}

	// Lightweight call to check token validity: get shop info.
	payload := map[string]interface{}{
		"shop_id": auth.ShopID,
	}
	_, err = a.do(ctx, http.MethodPost, "/api/v2/shop/get_shop_detail", auth, payload)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// SyncInventory pushes a local inventory level to the Shopee platform.
func (a *ShopeeRealAdapter) SyncInventory(ctx context.Context, input *SyncInventoryInput) (bool, error) {
	// SyncInventoryInput does not carry AccountID, so we attempt with minimal auth.
	auth := &shopeeAuth{BaseURL: ShopeeBaseURL}

	payload := map[string]interface{}{
		"item_id": nil,
		"sku":     input.SkuCode,
		"stock":   input.Quantity,
	}
	_, err := a.do(ctx, http.MethodPost, "/api/v2/product/update_stock", auth, payload)
	return err == nil, err
}

// PushTracking sends shipping tracking information to Shopee.
// Uses /api/v2/logistics/ship_order to confirm shipment.
func (a *ShopeeRealAdapter) PushTracking(ctx context.Context, input *PushTrackingInput) (bool, error) {
	auth, err := a.resolveAuth(ctx, input.PlatformID)
	if err != nil {
		return false, err
	}

	payload := map[string]interface{}{
		"order_sn":        input.OrderSN,
		"tracking_number": input.TrackingNumber,
	}
	if input.CarrierCode != "" {
		payload["carrier_code"] = input.CarrierCode
	}

	_, err = a.do(ctx, http.MethodPost, "/api/v2/logistics/ship_order", auth, payload)
	return err == nil, err
}

// FetchOrders pulls orders from Shopee since the given timestamp.
// Uses cursor-based pagination via /api/v2/order/get_order_list and fetches
// full detail via /api/v2/order/get_order_detail for each order.
func (a *ShopeeRealAdapter) FetchOrders(ctx context.Context, input *FetchOrdersInput) ([]*PlatformOrder, error) {
	auth, err := a.resolveAuth(ctx, input.PlatformID)
	if err != nil {
		return nil, err
	}

	// Refresh token if expired.
	var acct PlatformIntegrationAccount
	if err := a.db.WithContext(ctx).First(&acct, input.PlatformID).Error; err == nil && a.checkTokenExpiry(&acct) {
		if acct.RefreshToken != "" {
			if refreshErr := a.RefreshToken(ctx, input.PlatformID); refreshErr != nil {
				a.logger.Warn("shopee token refresh failed, continuing with existing token",
					zap.Int64("account", input.PlatformID), zap.Error(refreshErr))
			} else {
				// Re-resolve auth with fresh token.
				auth, err = a.resolveAuth(ctx, input.PlatformID)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	var allOrders []*PlatformOrder
	cursor := ""
	timeFrom := input.Since.Unix()
	timeTo := time.Now().Unix()
	pageSize := 100

	for {
		listPayload := map[string]interface{}{
			"time_from": timeFrom,
			"time_to":   timeTo,
			"page_size": pageSize,
			"cursor":    cursor,
		}

		body, err := a.do(ctx, http.MethodPost, "/api/v2/order/get_order_list", auth, listPayload)
		if err != nil {
			return allOrders, err
		}

		var orderListResp struct {
			Response struct {
				OrderList []struct {
					OrderSN string `json:"order_sn"`
					Status  string `json:"order_status"`
				} `json:"order_list"`
				More   bool   `json:"more"`
				Cursor string `json:"next_cursor"`
			} `json:"response"`
		}
		if err := json.Unmarshal(body, &orderListResp); err != nil {
			return allOrders, fmt.Errorf("shopee order list parse: %w", err)
		}

		for _, o := range orderListResp.Response.OrderList {
			detailBody, err := a.do(ctx, http.MethodPost, "/api/v2/order/get_order_detail", auth, map[string]interface{}{
				"order_sn":                 o.OrderSN,
				"response_optional_fields": "item_list,order_status,shipping_carrier,recipient_address,total_amount,shipping_fee,actual_shipping_fee,paid_time",
			})
			if err != nil {
				a.logger.Warn("shopee order detail fetch failed, skipping",
					zap.String("order_sn", o.OrderSN), zap.Error(err))
				continue
			}

			order := a.parseOrderDetail(detailBody)
			if order != nil {
				allOrders = append(allOrders, order)
			}
		}

		if !orderListResp.Response.More || orderListResp.Response.Cursor == "" {
			break
		}
		cursor = orderListResp.Response.Cursor
	}

	if allOrders == nil {
		allOrders = []*PlatformOrder{}
	}
	return allOrders, nil
}

// parseOrderDetail parses a Shopee order detail response into a PlatformOrder.
func (a *ShopeeRealAdapter) parseOrderDetail(body []byte) *PlatformOrder {
	var dr struct {
		Response struct {
			OrderSN      string `json:"order_sn"`
			OrderStatus  string `json:"order_status"`
			TotalAmount  float64 `json:"total_amount"`
			ShippingFee  float64 `json:"shipping_fee"`
			PaidTime     int64   `json:"paid_time"`
			ShipByDate   int64   `json:"ship_by_date"`
			CreateTime   int64   `json:"create_time"`
			RecipientAddress *struct {
				Name       string `json:"name"`
				Phone      string `json:"phone"`
				FullAddr   string `json:"full_address"`
			} `json:"recipient_address"`
			ItemList []struct {
				ItemName     string  `json:"item_name"`
				ModelID      int64   `json:"model_id"`
				ModelName    string  `json:"model_name"`
				ItemSKU      string  `json:"item_sku"`
				ModelSKU     string  `json:"model_sku"`
				ModelQty    int     `json:"model_quantity_purchased"`
				ModelOrigPrice float64 `json:"model_original_price"`
				ModelDiscPrice float64 `json:"model_discounted_price"`
			} `json:"item_list"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &dr); err != nil || dr.Response.OrderSN == "" {
		return nil
	}

	r := dr.Response
	status := mapStatus(r.OrderStatus)

	var paidAt string
	if r.PaidTime > 0 {
		paidAt = time.Unix(r.PaidTime, 0).Format(time.RFC3339)
	}

	var recipientName, recipientPhone, shippingAddr string
	if r.RecipientAddress != nil {
		recipientName = r.RecipientAddress.Name
		recipientPhone = r.RecipientAddress.Phone
		shippingAddr = r.RecipientAddress.FullAddr
	}

	var items []PlatformOrderItem
	for _, item := range r.ItemList {
		skuCode := item.ItemSKU
		if skuCode == "" {
			skuCode = item.ModelSKU
		}
		if skuCode == "" {
			skuCode = fmt.Sprintf("model-%d", item.ModelID)
		}

		items = append(items, PlatformOrderItem{
			SkuCode:   skuCode,
			Quantity:  item.ModelQty,
			UnitPrice: fmt.Sprintf("%.2f", item.ModelOrigPrice),
		})
	}

	if items == nil {
		items = []PlatformOrderItem{}
	}

	return &PlatformOrder{
		OrderSN:         r.OrderSN,
		Status:          status,
		TotalAmount:     fmt.Sprintf("%.2f", r.TotalAmount),
		ShippingFee:     fmt.Sprintf("%.2f", r.ShippingFee),
		PaidAt:          paidAt,
		RecipientName:   recipientName,
		RecipientPhone:  recipientPhone,
		ShippingAddress: shippingAddr,
		Items:           items,
	}
}

// FetchSettlements pulls settlement records from Shopee.
// Shopee finance/settlement API requires additional auth scope; returns empty
// for now as a stub.
func (a *ShopeeRealAdapter) FetchSettlements(ctx context.Context, input *FetchSettlementsInput) ([]*PlatformSettlement, error) {
	return []*PlatformSettlement{}, nil
}

// FetchReturns pulls return/refund requests from Shopee.
func (a *ShopeeRealAdapter) FetchReturns(ctx context.Context, input *FetchReturnsInput) ([]*PlatformReturn, error) {
	auth, err := a.resolveAuth(ctx, input.PlatformID)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"page_size": 100,
		"page_no":   1,
	}

	body, err := a.do(ctx, http.MethodPost, "/api/v2/returns/get_return_list", auth, payload)
	if err != nil {
		return nil, err
	}

	var r struct {
		Response struct {
			ReturnList []struct {
				ReturnSN     string  `json:"return_sn"`
				OrderSN      string  `json:"order_sn"`
				ItemName     string  `json:"item_name"`
				ReturnQty    int     `json:"return_quantity"`
				Reason       string  `json:"reason"`
				Status       string  `json:"status"`
				CreateTime   int64   `json:"create_time"`
				RefundAmount float64 `json:"refund_amount"`
			} `json:"return_list"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("shopee returns parse: %w", err)
	}

	var items []*PlatformReturn
	for _, ret := range r.Response.ReturnList {
		items = append(items, &PlatformReturn{
			ReturnID:     ret.ReturnSN,
			OrderSN:      ret.OrderSN,
			SkuCode:      ret.ItemName,
			Quantity:     ret.ReturnQty,
			Reason:       ret.Reason,
			Status:       ret.Status,
			CreatedAt:    time.Unix(ret.CreateTime, 0).Format(time.RFC3339),
			RefundAmount: fmt.Sprintf("%.2f", ret.RefundAmount),
		})
	}
	if items == nil {
		items = []*PlatformReturn{}
	}
	return items, nil
}

// ──────────────────────────────────────────────
//  Convenience helpers
// ──────────────────────────────────────────────

// ConfirmShipment confirms shipment for an order via /api/v2/logistics/ship_order.
func (a *ShopeeRealAdapter) ConfirmShipment(ctx context.Context, accountID int64, orderSN string) error {
	auth, err := a.resolveAuth(ctx, accountID)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"order_sn": orderSN,
	}
	_, err = a.do(ctx, http.MethodPost, "/api/v2/logistics/ship_order", auth, payload)
	return err
}

// UpdateOrderNote updates the note for an order via /api/v2/order/note/update.
func (a *ShopeeRealAdapter) UpdateOrderNote(ctx context.Context, accountID int64, orderSN, note string) error {
	auth, err := a.resolveAuth(ctx, accountID)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"order_sn": orderSN,
		"note":     note,
	}
	_, err = a.do(ctx, http.MethodPost, "/api/v2/order/note/update", auth, payload)
	return err
}
