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
	"strconv"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ShopeeOpenAPI base URLs for different regions.
const (
	ShopeeThailand   = "https://partner.shopeemobile.com"
	ShopeePhilippine = "https://partner.shopeemobile.com"
	ShopeeDefault    = "https://partner.shopeemobile.com"
)

// ShopeeAdapter implements PlatformAdapter for Shopee Open API v2.
type ShopeeAdapter struct {
	httpClient *http.Client
	db         *gorm.DB
	logger     *zap.Logger
}

func NewShopeeAdapter(db *gorm.DB, logger *zap.Logger) *ShopeeAdapter {
	return &ShopeeAdapter{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		db:         db,
		logger:     logger,
	}
}

type shopeeAuth struct {
	PartnerID   int64
	APIKey      string
	AccessToken string
	ShopID      int64
	BaseURL     string
}

// getAuth looks up the first active platform integration account for the given
// platform ID and returns Shopee credentials. Config stores partner_id, api_key, and shop_id.
func (a *ShopeeAdapter) getAuth(ctx context.Context, platformID int64) (*shopeeAuth, error) {
	var accts []PlatformIntegrationAccount
	if err := a.db.WithContext(ctx).
		Where("platform_id = ? AND status = ?", platformID, "active").
		Limit(1).
		Find(&accts).Error; err != nil {
		return nil, fmt.Errorf("shopee getAuth: %w", err)
	}
	if len(accts) == 0 {
		return nil, fmt.Errorf("shopee getAuth: no active account for platform_id=%d", platformID)
	}
	acct := accts[0]

	var cfg struct {
		PartnerID int64  `json:"partner_id"`
		APIKey    string `json:"api_key"`
		ShopID    int64  `json:"shop_id"`
	}
	if len(acct.Config) > 0 {
		json.Unmarshal(acct.Config, &cfg)
	}
	if acct.AccessToken == "" {
		return nil, fmt.Errorf("shopee getAuth: account %d has empty access_token", acct.ID)
	}
	return &shopeeAuth{
		PartnerID:   cfg.PartnerID,
		APIKey:      cfg.APIKey,
		AccessToken: acct.AccessToken,
		ShopID:      cfg.ShopID,
		BaseURL:     ShopeeDefault,
	}, nil
}

// getAuthByAccountID resolves a platform integration account ID to Shopee creds.
func (a *ShopeeAdapter) getAuthByAccountID(ctx context.Context, accountID int64) (*shopeeAuth, error) {
	var acct PlatformIntegrationAccount
	if err := a.db.WithContext(ctx).First(&acct, accountID).Error; err != nil {
		return nil, fmt.Errorf("shopee getAuthByAccountID: %w", err)
	}
	return a.getAuth(ctx, acct.PlatformID)
}

func (a *ShopeeAdapter) sign(partnerID int64, apiKey, path, accessToken string, shopID int64, timestamp int64) string {
	base := fmt.Sprintf("%d%s%d%s%d", partnerID, path, timestamp, accessToken, shopID)
	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write([]byte(base))
	return hex.EncodeToString(mac.Sum(nil))
}

func (a *ShopeeAdapter) do(ctx context.Context, method, path string, auth *shopeeAuth, payload interface{}) ([]byte, error) {
	url := auth.BaseURL + path
	ts := time.Now().Unix()

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

	sign := a.sign(auth.PartnerID, auth.APIKey, path, auth.AccessToken, auth.ShopID, ts)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", sign)
	req.Header.Set("Timestamp", strconv.FormatInt(ts, 10))

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("shopee %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("shopee read %s: %w", path, err)
	}

	var shopeeResp struct {
		Error   int    `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &shopeeResp) == nil && shopeeResp.Error != 0 {
		return nil, fmt.Errorf("shopee %s error %d: %s", path, shopeeResp.Error, shopeeResp.Message)
	}
	return body, nil
}

// VerifyWebhookSignature implements WebhookVerifier.
// Shopee signs webhook payloads with Authorization = HMAC-SHA256(body + timestamp, partner_key).
// The timestamp is taken from the Timestamp header, and the partner_key from the
// PlatformIntegrationAccount Config JSON as "api_key".
func (a *ShopeeAdapter) VerifyWebhookSignature(ctx context.Context, body []byte, headers http.Header) bool {
	signature := headers.Get("Authorization")
	if signature == "" {
		return false
	}
	timestamp := headers.Get("Timestamp")
	if timestamp == "" {
		return false
	}

	// Find the first active Shopee account and read its partner key.
	var accts []PlatformIntegrationAccount
	if err := a.db.WithContext(ctx).
		Model(&PlatformIntegrationAccount{}).
		Where("platform_id = (SELECT id FROM platform WHERE code = ?) AND status = ?", "shopee", "active").
		Limit(1).
		Find(&accts).Error; err != nil || len(accts) == 0 {
		return false
	}

	var cfg struct {
		APIKey string `json:"api_key"`
	}
	if len(accts[0].Config) > 0 {
		_ = json.Unmarshal(accts[0].Config, &cfg)
	}
	if cfg.APIKey == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(cfg.APIKey))
	mac.Write(body)
	mac.Write([]byte(timestamp))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (a *ShopeeAdapter) Publish(ctx context.Context, input *PublishInput) (*PublishResult, error) {
	if len(input.SKUs) == 0 {
		return nil, fmt.Errorf("shopee publish: no SKUs")
	}
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"item": map[string]interface{}{
			"name":        input.ProductName,
			"description": input.Description,
			"category_id": input.CategoryID,
			"images":      []string{},
			"item_sku":    input.SKUs[0].SkuCode,
			"price":       input.Prices[input.SKUs[0].SkuID],
			"stock":       input.Inventories[input.SKUs[0].SkuID],
		},
	}
	body, err := a.do(ctx, http.MethodPost, "/api/v2/product/add", auth, payload)
	if err != nil {
		return nil, err
	}
	var r struct {
		Response struct {
			ItemID   int64 `json:"item_id"`
		} `json:"response"`
	}
	json.Unmarshal(body, &r)
	sku := input.SKUs[0].SkuCode
	return &PublishResult{
		PlatformProductID: fmt.Sprintf("shopee-%d", r.Response.ItemID),
		PlatformSKU:       sku,
		PlatformURL:       fmt.Sprintf("https://shopee.ph/product/%d/", r.Response.ItemID),
		PublishedData:     map[string]interface{}{"item_id": r.Response.ItemID},
		SyncMessage:       "published to Shopee",
	}, nil
}

func (a *ShopeeAdapter) SyncStatus(ctx context.Context, input *SyncStatusInput) (string, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return "unknown", err
	}
	payload := map[string]interface{}{
		"item_id": interface{}(nil),
		"item_sku": []string{input.PlatformProductID},
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
	json.Unmarshal(body, &r)
	if len(r.Response.ItemList) == 0 {
		return "unknown", nil
	}
	m := map[string]string{"NORMAL": "synced", "UNLIST": "pending", "BANNED": "failed"}
	if s, ok := m[r.Response.ItemList[0].Status]; ok {
		return s, nil
	}
	return r.Response.ItemList[0].Status, nil
}

func (a *ShopeeAdapter) ValidateCredentials(ctx context.Context, accountID int64) (bool, error) {
	_, err := a.getAuthByAccountID(ctx, accountID)
	if err != nil {
		return false, fmt.Errorf("shopee ValidateCredentials: %w", err)
	}
	return false, fmt.Errorf("shopee ValidateCredentials: not yet implemented, requires OAuth credential setup")
}

func (a *ShopeeAdapter) SyncInventory(ctx context.Context, input *SyncInventoryInput) (bool, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return false, err
	}
	payload := map[string]interface{}{
		"item_id": interface{}(nil),
		"sku":     input.SkuCode,
		"stock":   input.Quantity,
	}
	_, err = a.do(ctx, http.MethodPost, "/api/v2/product/update_stock", auth, payload)
	return err == nil, err
}

func (a *ShopeeAdapter) PushTracking(ctx context.Context, input *PushTrackingInput) (bool, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
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
	_, err = a.do(ctx, http.MethodPost, "/api/v2/logistics/update_shipping_document", auth, payload)
	return err == nil, err
}

func (a *ShopeeAdapter) FetchOrders(ctx context.Context, input *FetchOrdersInput) ([]*PlatformOrder, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return nil, err
	}
	tsStr := strconv.FormatInt(input.Since.Unix(), 10)

	payload := map[string]interface{}{
		"time_from": tsStr,
		"time_to":   strconv.FormatInt(time.Now().Unix(), 10),
		"page_size": 100,
		"cursor":    0,
	}

	body, err := a.do(ctx, http.MethodPost, "/api/v2/order/get_order_list", auth, payload)
	if err != nil {
		return nil, err
	}

	var r struct {
		Response struct {
			OrderList []struct {
				OrderSN string `json:"order_sn"`
				Status  string `json:"order_status"`
			} `json:"order_list"`
		} `json:"response"`
	}
	json.Unmarshal(body, &r)

	var orders []*PlatformOrder
	for _, o := range r.Response.OrderList {
		detailPayload := map[string]interface{}{
			"order_sn":     o.OrderSN,
			"response_optional_fields": "item_list,order_status,shipping_carrier",
		}
		detailBody, err := a.do(ctx, http.MethodPost, "/api/v2/order/get_order_detail", auth, detailPayload)
		if err != nil {
			continue
		}
		var dr struct {
			Response struct {
				OrderSN     string `json:"order_sn"`
				ItemList    []struct {
					ItemName string `json:"item_name"`
					ModelID  int64  `json:"model_id"`
					ModelQuantity int `json:"model_quantity"`
					ModelOriginalPrice float64 `json:"model_original_price"`
				} `json:"item_list"`
				TotalAmount float64 `json:"total_amount"`
				OrderStatus string  `json:"order_status"`
			} `json:"response"`
		}
		json.Unmarshal(detailBody, &dr)

		var items []PlatformOrderItem
		for _, item := range dr.Response.ItemList {
			items = append(items, PlatformOrderItem{
				SkuCode:   fmt.Sprintf("model-%d", item.ModelID),
				Quantity:  item.ModelQuantity,
				UnitPrice: fmt.Sprintf("%.2f", item.ModelOriginalPrice),
			})
		}

		orders = append(orders, &PlatformOrder{
			OrderSN:     dr.Response.OrderSN,
			Status:      dr.Response.OrderStatus,
			TotalAmount: fmt.Sprintf("%.2f", dr.Response.TotalAmount),
			Items:       items,
		})
	}
	if orders == nil {
		orders = []*PlatformOrder{}
	}
	return orders, nil
}

func (a *ShopeeAdapter) FetchSettlements(ctx context.Context, input *FetchSettlementsInput) ([]*PlatformSettlement, error) {
	_, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return nil, err
	}
	// stub: Shopee finance API requires additional auth scope
	return []*PlatformSettlement{}, nil
}

func (a *ShopeeAdapter) FetchReturns(ctx context.Context, input *FetchReturnsInput) ([]*PlatformReturn, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
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
				ReturnSN    string `json:"return_sn"`
				OrderSN     string `json:"order_sn"`
				ItemName    string `json:"item_name"`
				ReturnQty   int    `json:"return_quantity"`
				Reason      string `json:"reason"`
				Status      string `json:"status"`
				CreateTime  int64  `json:"create_time"`
				RefundAmount float64 `json:"refund_amount"`
			} `json:"return_list"`
		} `json:"response"`
	}
	json.Unmarshal(body, &r)

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
