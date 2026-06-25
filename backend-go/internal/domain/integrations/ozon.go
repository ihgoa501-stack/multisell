package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	OzonAPIBase        = "https://api-seller.ozon.ru"
	OzonDefaultTimeout = 30 * time.Second
)

// OzonAdapter implements PlatformAdapter for the Ozon seller API.
type OzonAdapter struct {
	httpClient *http.Client
	db         *gorm.DB
	logger     *zap.Logger
}

// NewOzonAdapter creates an Ozon adapter with a DB handle for credential lookup.
func NewOzonAdapter(db *gorm.DB, logger *zap.Logger) *OzonAdapter {
	return &OzonAdapter{
		httpClient: &http.Client{Timeout: OzonDefaultTimeout},
		db:         db,
		logger:     logger,
	}
}

// ozonAuth stores Ozon API authentication details.
type ozonAuth struct {
	ClientID string
	APIKey   string
	BaseURL  string
}

// getAuth looks up the first active platform integration account for the given
// platform ID and returns Ozon credentials. access_token = API key,
// config.client_id = Ozon Client-Id.
func (a *OzonAdapter) getAuth(ctx context.Context, platformID int64) (*ozonAuth, error) {
	var accts []PlatformIntegrationAccount
	if err := a.db.WithContext(ctx).
		Where("platform_id = ? AND status = ?", platformID, "active").
		Limit(1).
		Find(&accts).Error; err != nil {
		return nil, fmt.Errorf("ozon getAuth: %w", err)
	}
	if len(accts) == 0 {
		return nil, fmt.Errorf("ozon getAuth: no active account for platform_id=%d", platformID)
	}
	acct := accts[0]

	var cfg struct {
		ClientID string `json:"client_id"`
	}
	if len(acct.Config) > 0 {
		json.Unmarshal(acct.Config, &cfg)
	}
	if acct.AccessToken == "" {
		return nil, fmt.Errorf("ozon getAuth: account %d has empty access_token", acct.ID)
	}
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("ozon getAuth: account %d missing client_id in config", acct.ID)
	}
	return &ozonAuth{
		ClientID: cfg.ClientID,
		APIKey:   acct.AccessToken,
		BaseURL:  OzonAPIBase,
	}, nil
}

// getAuthByAccountID resolves a platform integration account ID to Ozon creds.
func (a *OzonAdapter) getAuthByAccountID(ctx context.Context, accountID int64) (*ozonAuth, error) {
	var acct PlatformIntegrationAccount
	if err := a.db.WithContext(ctx).First(&acct, accountID).Error; err != nil {
		return nil, fmt.Errorf("ozon getAuthByAccountID: %w", err)
	}
	return a.getAuth(ctx, acct.PlatformID)
}

func (a *OzonAdapter) do(ctx context.Context, method, path string, auth *ozonAuth, payload interface{}) ([]byte, error) {
	url := auth.BaseURL + path
	var bodyReader io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("ozon marshal: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("ozon request: %w", err)
	}
	req.Header.Set("Client-Id", auth.ClientID)
	req.Header.Set("Api-Key", auth.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ozon %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ozon read %s: %w", path, err)
	}

	if resp.StatusCode >= 400 {
		var e struct {
			Error *struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &e) == nil {
			if e.Error != nil {
				return nil, fmt.Errorf("ozon %s [%s]: %s", path, e.Error.Code, e.Error.Message)
			}
			if e.Message != "" {
				return nil, fmt.Errorf("ozon %s: %s", path, e.Message)
			}
		}
		return nil, fmt.Errorf("ozon %s: HTTP %d %s", path, resp.StatusCode, truncStr(string(body), 300))
	}
	return body, nil
}

func (a *OzonAdapter) Publish(ctx context.Context, input *PublishInput) (*PublishResult, error) {
	if len(input.SKUs) == 0 {
		return nil, fmt.Errorf("ozon publish: no SKUs")
	}
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return nil, err
	}
	prices := make(map[int64]float64)
	for k, v := range input.Prices {
		prices[k] = safeFloat(v)
	}
	payload := map[string]interface{}{
		"items": []map[string]interface{}{{
			"name": input.ProductName, "description": input.Description,
			"category_id": input.CategoryID, "offer_id": input.SKUs[0].SkuCode,
			"currency_code": "RUB",
			"height": fmt.Sprintf("%.1f", input.PackageHeight),
			"width":  fmt.Sprintf("%.1f", input.PackageWidth),
			"depth":  fmt.Sprintf("%.1f", input.PackageLength),
			"weight": fmt.Sprintf("%.1f", input.PackageWeight),
			"sku_data": []map[string]interface{}{{
				"offer_id": input.SKUs[0].SkuCode,
				"price":    fmt.Sprintf("%.2f", prices[input.SKUs[0].SkuID]),
				"stock":    map[string]int{"present": input.Inventories[input.SKUs[0].SkuID], "reserved": 0},
			}},
		}},
	}
	body, err := a.do(ctx, http.MethodPost, "/v4/product/import", auth, payload)
	if err != nil {
		return nil, err
	}
	var r struct {
		Result struct {
			TaskID string `json:"task_id"`
		} `json:"result"`
	}
	json.Unmarshal(body, &r)
	sku := input.SKUs[0].SkuCode
	return &PublishResult{
		PlatformProductID: "ozon-" + sku,
		PlatformSKU:       sku,
		PlatformURL:       fmt.Sprintf("https://www.ozon.ru/product/ozon-%s/", sku),
		PublishedData:     map[string]interface{}{"task_id": r.Result.TaskID},
		SyncMessage:       fmt.Sprintf("published to Ozon (task_id=%s)", r.Result.TaskID),
	}, nil
}

func (a *OzonAdapter) SyncStatus(ctx context.Context, input *SyncStatusInput) (string, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return "unknown", err
	}
	offerID := input.PlatformProductID
	if len(offerID) > 4 && offerID[:4] == "ozon-" {
		offerID = offerID[5:]
	}
	body, err := a.do(ctx, http.MethodPost, "/v4/product/info", auth, map[string]interface{}{"offer_id": offerID, "sku": nil})
	if err != nil {
		return "unknown", err
	}
	var r struct {
		Result struct {
			Items []struct {
				State string `json:"state"`
			} `json:"items"`
		} `json:"result"`
	}
	json.Unmarshal(body, &r)
	if len(r.Result.Items) == 0 {
		return "unknown", nil
	}
	m := map[string]string{"imported": "synced", "processed": "synced", "processing": "in_progress", "created": "pending", "failed": "failed", "rejected": "failed"}
	if s, ok := m[r.Result.Items[0].State]; ok {
		return s, nil
	}
	return r.Result.Items[0].State, nil
}

func (a *OzonAdapter) ValidateCredentials(ctx context.Context, accountID int64) (bool, error) {
	auth, err := a.getAuthByAccountID(ctx, accountID)
	if err != nil {
		return false, fmt.Errorf("ozon ValidateCredentials: %w", err)
	}
	// Hit the Ozon /v1/product/list endpoint (lightweight) to verify creds.
	body, err := a.do(ctx, http.MethodPost, "/v1/product/list", auth, map[string]interface{}{"page": 1, "page_size": 1})
	if err != nil {
		return false, fmt.Errorf("ozon ValidateCredentials: %w", err)
	}
	var r struct {
		Result struct {
			Total int64 `json:"total"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return false, fmt.Errorf("ozon ValidateCredentials: parse error: %w", err)
	}
	return true, nil
}

func (a *OzonAdapter) SyncInventory(ctx context.Context, input *SyncInventoryInput) (bool, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return false, err
	}
	sku := input.PlatformSKU
	if sku == "" {
		sku = input.SkuCode
	}
	payload := map[string]interface{}{"stocks": []map[string]interface{}{{"sku": sku, "stock": input.Quantity}}}
	_, err = a.do(ctx, http.MethodPost, "/v4/product/import/stocks", auth, payload)
	return err == nil, err
}

func (a *OzonAdapter) PushTracking(ctx context.Context, input *PushTrackingInput) (bool, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return false, err
	}
	p := map[string]interface{}{"posting_number": input.OrderSN, "tracking_number": input.TrackingNumber}
	if input.CarrierCode != "" {
		p["carrier_code"] = input.CarrierCode
	}
	_, err = a.do(ctx, http.MethodPost, "/v3/posting/fbs/ship", auth, p)
	return err == nil, err
}

func (a *OzonAdapter) FetchOrders(ctx context.Context, input *FetchOrdersInput) ([]*PlatformOrder, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return nil, err
	}
	var orders []*PlatformOrder
	page := 1
	for {
		payload := map[string]interface{}{
			"dir": "ASC",
			"filter": map[string]string{
				"since": input.Since.Format("2006-01-02T15:04:05.000Z"),
			},
			"limit": 100,
			"page":  page,
		}
		body, err := a.do(ctx, http.MethodPost, "/v3/posting/fbs/list", auth, payload)
		if err != nil {
			return nil, fmt.Errorf("ozon fetch_orders page %d: %w", page, err)
		}
		var r struct {
			Result struct {
				Postings []struct {
					PostingNumber string `json:"posting_number"`
					Status        string `json:"status"`
					InProcessAt   string `json:"in_process_at"`
					AnalyticsData struct {
						DeliveryPrice string `json:"delivery_price"`
					} `json:"analytics_data"`
					FinancialData struct {
						Products []struct {
							Sku      string `json:"sku"`
							Quantity int    `json:"quantity"`
							Price    string `json:"price"`
						} `json:"products"`
					} `json:"financial_data"`
				} `json:"postings"`
			} `json:"result"`
		}
		json.Unmarshal(body, &r)
		if len(r.Result.Postings) == 0 {
			break
		}
		for _, p := range r.Result.Postings {
			var items []PlatformOrderItem
			total := 0.0
			for _, prod := range p.FinancialData.Products {
				price := sf(prod.Price)
				total += price * float64(prod.Quantity)
				items = append(items, PlatformOrderItem{SkuCode: prod.Sku, Quantity: prod.Quantity, UnitPrice: prod.Price})
			}
			orders = append(orders, &PlatformOrder{
				OrderSN: p.PostingNumber, Status: p.Status,
				TotalAmount: ff(total), ShippingFee: p.AnalyticsData.DeliveryPrice,
				PaidAt: p.InProcessAt, Items: items,
			})
		}
		page++
	}
	return orders, nil
}

func (a *OzonAdapter) FetchSettlements(ctx context.Context, input *FetchSettlementsInput) ([]*PlatformSettlement, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"filter":   map[string]interface{}{"date": map[string]string{"from": input.Since.Format("2006-01-02T15:04:05.000Z")}},
		"page":     1,
		"page_size": 100,
	}
	body, err := a.do(ctx, http.MethodPost, "/v3/finance/transaction/list", auth, payload)
	if err != nil {
		return nil, err
	}
	var r struct {
		Result struct {
			Operations []struct {
				OperationID   string `json:"operation_id"`
				OperationType string `json:"operation_type"`
				Amount        string `json:"amount"`
				CurrencyCode  string `json:"currency_code"`
				OperationDate string `json:"operation_date"`
				Description   string `json:"description"`
				Posting       struct {
					PostingNumber string `json:"posting_number"`
				} `json:"posting"`
			} `json:"operations"`
		} `json:"result"`
	}
	json.Unmarshal(body, &r)
	tm := map[string]string{"sale": "order_sale", "refund": "refund", "delivery": "shipping_fee", "commission": "platform_fee", "payment_commission": "payment_fee"}
	var items []*PlatformSettlement
	for _, tx := range r.Result.Operations {
		ttype := tm[tx.OperationType]
		if ttype == "" {
			ttype = "other"
		}
		items = append(items, &PlatformSettlement{
			TransactionID: tx.OperationID, TransactionType: ttype,
			OrderSN: tx.Posting.PostingNumber, Amount: ff(absf(sf(tx.Amount))),
			Currency: tx.CurrencyCode, OccurredAt: tx.OperationDate,
			Description: tx.Description,
		})
	}
	return items, nil
}

func (a *OzonAdapter) FetchReturns(ctx context.Context, input *FetchReturnsInput) ([]*PlatformReturn, error) {
	auth, err := a.getAuth(ctx, input.PlatformID)
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"filter": map[string]string{"last_change_from": input.Since.Format("2006-01-02T15:04:05.000Z")},
		"limit":  100,
	}
	body, err := a.do(ctx, http.MethodPost, "/v3/returns/list", auth, payload)
	if err != nil {
		return nil, err
	}
	var r struct {
		Result struct {
			Returns []struct {
				ReturnID     string `json:"return_id"`
				PostingNumber string `json:"posting_number"`
				Sku          string `json:"sku"`
				Quantity     int    `json:"quantity"`
				Reason       string `json:"reason"`
				Status       string `json:"status"`
				CreatedAt    string `json:"created_at"`
				RefundAmount string `json:"refund_amount"`
			} `json:"returns"`
		} `json:"result"`
	}
	json.Unmarshal(body, &r)
	var items []*PlatformReturn
	for _, ret := range r.Result.Returns {
		items = append(items, &PlatformReturn{
			ReturnID: ret.ReturnID, OrderSN: ret.PostingNumber,
			SkuCode: ret.Sku, Quantity: ret.Quantity,
			Reason: ret.Reason, Status: ret.Status,
			CreatedAt: ret.CreatedAt, RefundAmount: ret.RefundAmount,
		})
	}
	return items, nil
}

// ─── Ozon-specific product listing (not part of PlatformAdapter) ───

// OzonProduct is a simplified product view from Ozon.
type OzonProduct struct {
	OfferID    string  `json:"offer_id"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	OldPrice   float64 `json:"old_price,omitempty"`
	Stock      int     `json:"stock"`
	CategoryID int64   `json:"category_id"`
	State      string  `json:"state"`
	ImageURL   string  `json:"image_url,omitempty"`
}

// ListProducts fetches the product catalog from Ozon via /v1/product/list
// and /v3/product/info/list. Returns a flat list of simplified products.
func (a *OzonAdapter) ListProducts(ctx context.Context, platformID int64) ([]OzonProduct, error) {
	auth, err := a.getAuth(ctx, platformID)
	if err != nil {
		return nil, err
	}

	// Step 1: list all offer IDs (paginated)
	var allOffers []string
	page := 1
	for {
		body, err := a.do(ctx, http.MethodPost, "/v1/product/list", auth, map[string]interface{}{
			"page": page, "page_size": 100,
		})
		if err != nil {
			return nil, fmt.Errorf("ozon list_products page %d: %w", page, err)
		}
		var r struct {
			Result struct {
				Items []struct {
					OfferID string `json:"offer_id"`
				} `json:"items"`
				Total int `json:"total"`
			} `json:"result"`
		}
		if err := json.Unmarshal(body, &r); err != nil {
			return nil, fmt.Errorf("ozon list_products parse: %w", err)
		}
		for _, item := range r.Result.Items {
			allOffers = append(allOffers, item.OfferID)
		}
		if len(allOffers) >= r.Result.Total || len(r.Result.Items) == 0 {
			break
		}
		page++
	}
	if len(allOffers) == 0 {
		return nil, nil
	}

	// Step 2: fetch details in batches (Ozon /v3/product/info/list allows up to 1000 per call)
	var products []OzonProduct
	batchSize := 100
	for i := 0; i < len(allOffers); i += batchSize {
		end := i + batchSize
		if end > len(allOffers) {
			end = len(allOffers)
		}
		batch := allOffers[i:end]

		body, err := a.do(ctx, http.MethodPost, "/v3/product/info/list", auth, map[string]interface{}{
			"offer_id": batch,
		})
		if err != nil {
			a.logger.Warn("ozon product info batch failed, skipping",
				zap.Int("from", i), zap.Int("count", len(batch)), zap.Error(err))
			continue
		}
		var r struct {
			Result struct {
				Items []struct {
					OfferID     string  `json:"offer_id"`
					Name        string  `json:"name"`
					Price       float64 `json:"price"`
					OldPrice    float64 `json:"old_price,omitempty"`
					Stock       int     `json:"stock"`
					CategoryID  int64   `json:"category_id"`
					State       string  `json:"state"`
					PrimaryImg  string  `json:"primary_image"`
				} `json:"items"`
			} `json:"result"`
		}
		if err := json.Unmarshal(body, &r); err != nil {
			a.logger.Warn("ozon product info parse failed, skipping batch",
				zap.Int("from", i), zap.Error(err))
			continue
		}
		for _, item := range r.Result.Items {
			products = append(products, OzonProduct{
				OfferID:    item.OfferID,
				Name:       item.Name,
				Price:      item.Price,
				OldPrice:   item.OldPrice,
				Stock:      item.Stock,
				CategoryID: item.CategoryID,
				State:      item.State,
				ImageURL:   item.PrimaryImg,
			})
		}
	}
	return products, nil
}

// --- helpers ---

func sf(s string) float64 {
	var f float64
	json.Unmarshal([]byte(s), &f)
	return f
}

func absf(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func ff(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

func safeFloat(s string) float64 {
	var f float64
	if s == "" {
		return 0
	}
	json.Unmarshal([]byte(s), &f)
	return f
}

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
