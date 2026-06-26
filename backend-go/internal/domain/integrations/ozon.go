package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	OzonAPIBase        = "https://api-seller.ozon.ru"
	OzonDefaultTimeout = 30 * time.Second
)

// OzonAdapter implements PlatformAdapter for the Ozon seller API.
//
// Deprecated: Use OzonRealAdapter instead. OzonAdapter is a stub that only
// works with hardcoded placeholder credentials and fake API responses.
// New code should register OzonRealAdapter via InitRealAdapters.
type OzonAdapter struct {
	httpClient *http.Client
}

func NewOzonAdapter() *OzonAdapter {
	return &OzonAdapter{
		httpClient: &http.Client{Timeout: OzonDefaultTimeout},
	}
}

type ozonAuth struct {
	ClientID string
	APIKey   string
	BaseURL  string
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
	auth := &ozonAuth{BaseURL: OzonAPIBase, ClientID: "?", APIKey: "?"}
	if len(input.SKUs) == 0 {
		return nil, fmt.Errorf("ozon publish: no SKUs")
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
	auth := &ozonAuth{BaseURL: OzonAPIBase, ClientID: "?", APIKey: "?"}
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
	// Requires auth resolution from DB. Return stub until full auth wiring.
	return false, fmt.Errorf("ozon ValidateCredentials not implemented without auth resolution")
}

func (a *OzonAdapter) SyncInventory(ctx context.Context, input *SyncInventoryInput) (bool, error) {
	auth := &ozonAuth{BaseURL: OzonAPIBase, ClientID: "?", APIKey: "?"}
	sku := input.PlatformSKU
	if sku == "" {
		sku = input.SkuCode
	}
	payload := map[string]interface{}{"stocks": []map[string]interface{}{{"sku": sku, "stock": input.Quantity}}}
	_, err := a.do(ctx, http.MethodPost, "/v4/product/import/stocks", auth, payload)
	return err == nil, err
}

func (a *OzonAdapter) PushTracking(ctx context.Context, input *PushTrackingInput) (bool, error) {
	auth := &ozonAuth{BaseURL: OzonAPIBase, ClientID: "?", APIKey: "?"}
	p := map[string]interface{}{"posting_number": input.OrderSN, "tracking_number": input.TrackingNumber}
	if input.CarrierCode != "" {
		p["carrier_code"] = input.CarrierCode
	}
	_, err := a.do(ctx, http.MethodPost, "/v3/posting/fbs/ship", auth, p)
	return err == nil, err
}

func (a *OzonAdapter) FetchOrders(ctx context.Context, input *FetchOrdersInput) ([]*PlatformOrder, error) {
	auth := &ozonAuth{BaseURL: OzonAPIBase, ClientID: "?", APIKey: "?"}
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
	auth := &ozonAuth{BaseURL: OzonAPIBase, ClientID: "?", APIKey: "?"}
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
	auth := &ozonAuth{BaseURL: OzonAPIBase, ClientID: "?", APIKey: "?"}
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
