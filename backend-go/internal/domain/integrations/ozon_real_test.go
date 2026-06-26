package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// testOzonServer is a mock Ozon Seller API server for tests.
type testOzonServer struct {
	*httptest.Server
	tokenHits atomic.Int64 // count of token endpoint calls
}

// newTestOzonServer creates a mocked Ozon API server.
// handlers maps endpoint paths to their handler funcs.
// If a path is not in handlers, a 200 with "{}" is returned.
func newTestOzonServer(t testing.TB, handlers map[string]func(w http.ResponseWriter, r *http.Request)) *testOzonServer {
	t.Helper()
	s := &testOzonServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Track token endpoint hits.
		if r.URL.Path == OzonRealTokenEndpoint {
			s.tokenHits.Add(1)
		}

		if h, ok := handlers[r.URL.Path]; ok {
			h(w, r)
			return
		}
		// Default: empty result.
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	return s
}

func (s *testOzonServer) Close() {
	s.Server.Close()
}

// testRealAdapter creates an OzonRealAdapter with a given test server URL and DB.
func testRealAdapter(t testing.TB, serverURL string, db *gorm.DB) *OzonRealAdapter {
	t.Helper()
	adapter := NewOzonRealAdapter(db, zap.NewNop())
	adapter.baseURL = serverURL
	// Disable transport pooling for test isolation.
	adapter.httpClient = &http.Client{Timeout: 5 * time.Second}
	return adapter
}

// createTestAcct inserts a PlatformIntegrationAccount for testing.
func createTestAcct(t testing.TB, db *gorm.DB, platformID int64, token string, expiresIn time.Duration, configJSON string, status ...string) *PlatformIntegrationAccount {
	t.Helper()
	var expiresAt *time.Time
	st := "active"
	if len(status) > 0 {
		st = status[0]
	}
	// When expiresIn is non-zero, set the expiry. Negative means already expired.
	if expiresIn != 0 {
		tm := time.Now().Add(expiresIn)
		expiresAt = &tm
	}
	acct := &PlatformIntegrationAccount{
		PlatformID:     platformID,
		StoreName:      "test-store",
		AccountID:      "test-acc",
		AccessToken:    token,
		TokenExpiresAt: expiresAt,
		Status:         st,
	}
	if configJSON != "" {
		acct.Config = json.RawMessage(configJSON)
	}
	if err := db.Create(acct).Error; err != nil {
		t.Fatalf("create test account: %v", err)
	}
	// Re-read to get AfterFind-decrypted token.
	var reloaded PlatformIntegrationAccount
	if err := db.First(&reloaded, acct.ID).Error; err != nil {
		t.Fatalf("re-read test account: %v", err)
	}
	return &reloaded
}

// ---------------------------------------------------------------------------
// TestAuthFlow
// ---------------------------------------------------------------------------

func TestRealAdapter_AuthFlow(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	var tokenCalled bool
	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			tokenCalled = true
			// Verify request body.
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, `{"error":{"code":"BAD_REQUEST","message":"invalid json"}}`, 400)
				return
			}
			if body["client_id"] != "test-client" || body["client_secret"] != "test-secret" {
				http.Error(w, `{"error":{"code":"AUTH_FAILED","message":"invalid credentials"}}`, 401)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"fresh-token","token_type":"Bearer","expires_in":86400}`))
		},
		OzonRealProductList: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			// Verify token is being sent.
			if r.Header.Get("Api-Key") == "" {
				http.Error(w, `{"error":{"code":"NO_AUTH","message":"missing token"}}`, 401)
				return
			}
			if r.Header.Get("Api-Key") != "fresh-token" {
				http.Error(w, `{"error":{"code":"BAD_TOKEN","message":"token mismatch"}}`, 401)
				return
			}
			if r.Header.Get("Client-Id") != "test-client" {
				http.Error(w, `{"error":{"code":"BAD_CLIENT","message":"missing client-id"}}`, 401)
				return
			}
			w.Write([]byte(`{"result":{"items":[],"total":0}}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	_ = createTestAcct(t, db, 1, "", 0, `{"client_id":"test-client","client_secret":"test-secret"}`)

	// This should trigger token exchange then use the fresh token.
	ok, err := adapter.ValidateCredentials(context.Background(), 1)
	if err != nil {
		t.Fatalf("ValidateCredentials: %v", err)
	}
	if !ok {
		t.Fatal("expected ValidateCredentials to succeed")
	}
	if !tokenCalled {
		t.Fatal("expected token endpoint to be called")
	}

	// Verify the token was persisted.
	var acct PlatformIntegrationAccount
	if err := db.First(&acct, 1).Error; err != nil {
		t.Fatalf("read account: %v", err)
	}
	if acct.AccessToken != "fresh-token" {
		t.Fatalf("expected stored token 'fresh-token', got %q", acct.AccessToken)
	}
	if acct.TokenExpiresAt == nil {
		t.Fatal("expected token_expires_at to be set")
	}
}

// ---------------------------------------------------------------------------
// TestTokenExpiry (auto-refresh on expired token)
// ---------------------------------------------------------------------------

func TestRealAdapter_TokenExpiry(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	var tokenCalls atomic.Int64
	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			tokenCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"new-token","token_type":"Bearer","expires_in":3600}`))
		},
		OzonRealProductList: func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("Api-Key")
			if apiKey == "" {
				http.Error(w, `{"error":{"code":"NO_AUTH"}}`, 401)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result":{"items":[],"total":0}}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	// Expired token — set expiry in the past.
	_ = createTestAcct(t, db, 1, "expired-token", -1*time.Hour, `{"client_id":"test-client","client_secret":"test-secret"}`)

	ok, err := adapter.ValidateCredentials(context.Background(), 1)
	if err != nil {
		t.Fatalf("ValidateCredentials with expired token: %v", err)
	}
	if !ok {
		t.Fatal("expected ValidateCredentials to succeed after refresh")
	}

	// Token should have been refreshed.
	calls := tokenCalls.Load()
	if calls == 0 {
		t.Fatal("expected token refresh on expired token")
	}

	var acct PlatformIntegrationAccount
	if err := db.First(&acct, 1).Error; err != nil {
		t.Fatalf("read account: %v", err)
	}
	if acct.AccessToken != "new-token" {
		t.Fatalf("expected refreshed token 'new-token', got %q", acct.AccessToken)
	}
}

// ---------------------------------------------------------------------------
// TestPublish (product import + polling)
// ---------------------------------------------------------------------------

func TestRealAdapter_Publish(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	var importCalled, pollCalled bool
	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`))
		},
		OzonRealProductImport: func(w http.ResponseWriter, r *http.Request) {
			importCalled = true
			// Verify request body.
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			items, ok := body["items"].([]interface{})
			if !ok || len(items) != 1 {
				http.Error(w, `{"error":{"code":"BAD_REQUEST","message":"invalid items"}}`, 400)
				return
			}
			item := items[0].(map[string]interface{})
			if item["offer_id"] != "test-sku" {
				http.Error(w, `{"error":{"code":"BAD_REQUEST","message":"unexpected offer_id"}}`, 400)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result":{"task_id":12345}}`))
		},
		OzonRealProductImportInfo: func(w http.ResponseWriter, r *http.Request) {
			pollCalled = true
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result":{"items":[{"offer_id":"test-sku","status":"imported"}]}}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	_ = createTestAcct(t, db, 1, "test-token", 1*time.Hour, `{"client_id":"test-client","client_secret":"test-secret"}`)

	result, err := adapter.Publish(context.Background(), &PublishInput{
		ProductID:   100,
		PlatformID:  1,
		ProductName: "Test Product",
		Description: "A test product for Ozon",
		CategoryID:  12345,
		SKUs:        []PublishSKU{{SkuID: 1, SkuCode: "test-sku"}},
		Prices:      map[int64]string{1: "1999.99"},
		Inventories: map[int64]int{1: 50},
		MainImage:   "https://example.com/img.jpg",
		Images:      []string{"https://example.com/img2.jpg"},
	})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if !importCalled {
		t.Fatal("expected product import endpoint to be called")
	}
	if !pollCalled {
		t.Fatal("expected import info poll to be called")
	}
	if result.PlatformSKU != "test-sku" {
		t.Fatalf("expected PlatformSKU 'test-sku', got %q", result.PlatformSKU)
	}
	if result.PublishedData["task_id"] != int64(12345) {
		t.Fatalf("expected task_id 12345, got %v", result.PublishedData["task_id"])
	}
}

// ---------------------------------------------------------------------------
// TestPublish_ImportFailed
// ---------------------------------------------------------------------------

func TestRealAdapter_Publish_ImportFailed(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`))
		},
		OzonRealProductImport: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result":{"task_id":666}}`))
		},
		OzonRealProductImportInfo: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result":{"items":[{"offer_id":"bad-sku","status":"failed","errors":[{"code":"VALIDATION_ERROR","message":"invalid price"}]}]}}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	_ = createTestAcct(t, db, 1, "test-token", 1*time.Hour, `{"client_id":"test-client","client_secret":"test-secret"}`)

	_, err := adapter.Publish(context.Background(), &PublishInput{
		ProductID:   100,
		PlatformID:  1,
		ProductName: "Bad Product",
		CategoryID:  123,
		SKUs:        []PublishSKU{{SkuID: 1, SkuCode: "bad-sku"}},
		Prices:      map[int64]string{1: "0"},
		Inventories: map[int64]int{1: 0},
	})
	if err == nil {
		t.Fatal("expected error for failed import")
	}
	if !strings.Contains(err.Error(), "import failed") {
		t.Fatalf("expected 'import failed' error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestSyncInventory
// ---------------------------------------------------------------------------

func TestRealAdapter_SyncInventory(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	var stockCalled bool
	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`))
		},
		OzonRealStockImport: func(w http.ResponseWriter, r *http.Request) {
			stockCalled = true
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			stocks, ok := body["stocks"].([]interface{})
			if !ok || len(stocks) != 1 {
				http.Error(w, `{"error":{"code":"BAD_REQUEST"}}`, 400)
				return
			}
			entry := stocks[0].(map[string]interface{})
			if entry["offer_id"] != "test-sku" || entry["stock"] != float64(150) {
				http.Error(w, `{"error":{"code":"BAD_REQUEST"}}`, 400)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result":true}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	_ = createTestAcct(t, db, 1, "test-token", 1*time.Hour, `{"client_id":"test-client","client_secret":"test-secret"}`)

	ok, err := adapter.SyncInventory(context.Background(), &SyncInventoryInput{
		PlatformID: 1,
		SkuCode:    "test-sku",
		Quantity:   150,
	})
	if err != nil {
		t.Fatalf("SyncInventory: %v", err)
	}
	if !ok {
		t.Fatal("expected SyncInventory to succeed")
	}
	if !stockCalled {
		t.Fatal("expected stock endpoint to be called")
	}
}

// ---------------------------------------------------------------------------
// TestFetchOrders
// ---------------------------------------------------------------------------

func TestRealAdapter_FetchOrders(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`))
		},
		OzonRealFBSList: func(w http.ResponseWriter, r *http.Request) {
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			offset := int(body["offset"].(float64))
			limit := int(body["limit"].(float64))

			w.Header().Set("Content-Type", "application/json")
			if offset == 0 {
				w.Write([]byte(`{
					"result": {
						"postings": [
							{
								"posting_number": "posting-001",
								"status": "delivered",
								"in_process_at": "2026-06-01T10:00:00.000Z",
								"analytics_data": {"delivery_price": "500.00"},
								"financial_data": {
									"products": [
										{"sku": "sku-1", "quantity": 2, "price": "1999.99"},
										{"sku": "sku-2", "quantity": 1, "price": "999.50"}
									]
								}
							}
						]
					}
				}`))
			} else {
				// Second (empty) page signals end of pagination.
				w.Write([]byte(`{"result":{"postings":[]}}`))
			}
			_ = limit // used
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	_ = createTestAcct(t, db, 1, "test-token", 1*time.Hour, `{"client_id":"test-client","client_secret":"test-secret"}`)

	since := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	orders, err := adapter.FetchOrders(context.Background(), &FetchOrdersInput{
		PlatformID: 1,
		Since:      since,
	})
	if err != nil {
		t.Fatalf("FetchOrders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(orders))
	}
	if orders[0].OrderSN != "posting-001" {
		t.Fatalf("expected OrderSN 'posting-001', got %q", orders[0].OrderSN)
	}
	if orders[0].Status != "delivered" {
		t.Fatalf("expected Status 'delivered', got %q", orders[0].Status)
	}
	if orders[0].ShippingFee != "500.00" {
		t.Fatalf("expected ShippingFee '500.00', got %q", orders[0].ShippingFee)
	}
	if len(orders[0].Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(orders[0].Items))
	}
}

// ---------------------------------------------------------------------------
// TestValidateCredentials
// ---------------------------------------------------------------------------

func TestRealAdapter_ValidateCredentials(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	var listCalled bool
	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"valid-token","token_type":"Bearer","expires_in":86400}`))
		},
		OzonRealProductList: func(w http.ResponseWriter, r *http.Request) {
			listCalled = true
			// Verify auth headers.
			if r.Header.Get("Api-Key") == "" {
				http.Error(w, `{"error":{"code":"NO_AUTH"}}`, 401)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result":{"items":[],"total":0}}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	_ = createTestAcct(t, db, 1, "valid-token", 1*time.Hour, `{"client_id":"test-client","client_secret":"test-secret"}`)

	ok, err := adapter.ValidateCredentials(context.Background(), 1)
	if err != nil {
		t.Fatalf("ValidateCredentials: %v", err)
	}
	if !ok {
		t.Fatal("expected ValidateCredentials to succeed")
	}
	if !listCalled {
		t.Fatal("expected product list endpoint to be called")
	}
}

// TestValidateCredentialsWithTokenOnly tests that the adapter works with just
// a valid token, no credentials config needed.
func TestRealAdapter_ValidateCredentials_TokenOnly(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	var listCalled bool
	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealProductList: func(w http.ResponseWriter, r *http.Request) {
			listCalled = true
			if r.Header.Get("Api-Key") != "valid-token" {
				http.Error(w, `{"error":{"code":"BAD_TOKEN","message":"wrong token"}}`, 401)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result":{"items":[],"total":0}}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	// Account with valid token but no config (legacy mode).
	_ = createTestAcct(t, db, 1, "valid-token", 1*time.Hour, "")

	ok, err := adapter.ValidateCredentials(context.Background(), 1)
	if err != nil {
		t.Fatalf("ValidateCredentials with token only: %v", err)
	}
	if !ok {
		t.Fatal("expected ValidateCredentials to succeed")
	}
	if !listCalled {
		t.Fatal("expected product list endpoint to be called")
	}
}

// TestValidateCredentialsWithEmptyConfig tests that the adapter works with
// empty config (no client_id/secret) when a valid token is already stored.
func TestRealAdapter_ValidateCredentials_EmptyConfig(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealProductList: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result":{"items":[],"total":0}}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	_ = createTestAcct(t, db, 1, "some-token", 1*time.Hour, `{}`)

	ok, err := adapter.ValidateCredentials(context.Background(), 1)
	if err != nil {
		t.Fatalf("ValidateCredentials with empty config: %v", err)
	}
	if !ok {
		t.Fatal("expected ValidateCredentials to succeed")
	}
}

func TestRealAdapter_RetryOn429(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	var callCount atomic.Int64
	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`))
		},
		OzonRealProductList: func(w http.ResponseWriter, r *http.Request) {
			n := callCount.Add(1)
			if n == 1 {
				// First call: rate limited.
				http.Error(w, `{"error":{"code":"TOO_MANY_REQUESTS","message":"rate limited"}}`, http.StatusTooManyRequests)
				return
			}
			// Second call: succeed.
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result":{"items":[],"total":0}}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	_ = createTestAcct(t, db, 1, "test-token", 1*time.Hour, `{"client_id":"test-client","client_secret":"test-secret"}`)

	ok, err := adapter.ValidateCredentials(context.Background(), 1)
	if err != nil {
		t.Fatalf("ValidateCredentials after 429 retry: %v", err)
	}
	if !ok {
		t.Fatal("expected success after 429 retry")
	}
	if callCount.Load() != 2 {
		t.Fatalf("expected 2 calls (1 failed, 1 retry), got %d", callCount.Load())
	}
}

// ---------------------------------------------------------------------------
// TestRetryOn5xx
// ---------------------------------------------------------------------------

func TestRealAdapter_RetryOn5xx(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	var callCount atomic.Int64
	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`))
		},
		OzonRealProductList: func(w http.ResponseWriter, r *http.Request) {
			n := callCount.Add(1)
			if n < 3 {
				http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"server error"}}`, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result":{"items":[],"total":0}}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	_ = createTestAcct(t, db, 1, "test-token", 1*time.Hour, `{"client_id":"test-client","client_secret":"test-secret"}`)

	ok, err := adapter.ValidateCredentials(context.Background(), 1)
	if err != nil {
		t.Fatalf("ValidateCredentials after 5xx retries: %v", err)
	}
	if !ok {
		t.Fatal("expected success after 5xx retries")
	}
	if callCount.Load() != 3 {
		t.Fatalf("expected 3 calls (2 failed, 1 retry), got %d", callCount.Load())
	}
}

// ---------------------------------------------------------------------------
// Test401AutoReAuth (401 triggers token refresh and retry)
// ---------------------------------------------------------------------------

func TestRealAdapter_401AutoReAuth(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	var tokenCalls atomic.Int64
	var apiCalls atomic.Int64
	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			tokenCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"new-token","token_type":"Bearer","expires_in":3600}`))
		},
		OzonRealProductList: func(w http.ResponseWriter, r *http.Request) {
			n := apiCalls.Add(1)
			if n == 1 {
				// First call: token rejected.
				http.Error(w, `{"error":{"code":"INVALID_TOKEN","message":"token expired"}}`, http.StatusUnauthorized)
				return
			}
			// Second call (after refresh): succeed.
			if r.Header.Get("Api-Key") != "new-token" {
				http.Error(w, `{"error":{"code":"BAD_TOKEN","message":"wrong token"}}`, 401)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result":{"items":[],"total":0}}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	_ = createTestAcct(t, db, 1, "expired-token", 1*time.Hour, `{"client_id":"test-client","client_secret":"test-secret"}`)

	ok, err := adapter.ValidateCredentials(context.Background(), 1)
	if err != nil {
		t.Fatalf("ValidateCredentials after 401 re-auth: %v", err)
	}
	if !ok {
		t.Fatal("expected success after 401 re-auth")
	}

	// Token should have been refreshed at least once (initial token exchange + 401 trigger).
	if tokenCalls.Load() < 1 {
		t.Fatalf("expected at least 1 token refresh, got %d", tokenCalls.Load())
	}
}

// ---------------------------------------------------------------------------
// TestSyncStatus
// ---------------------------------------------------------------------------

func TestRealAdapter_SyncStatus(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`))
		},
		OzonRealProductInfo: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result":{"id":123,"offer_id":"test-sku","state":"imported","name":"Test"}}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	_ = createTestAcct(t, db, 1, "test-token", 1*time.Hour, `{"client_id":"test-client","client_secret":"test-secret"}`)

	status, err := adapter.SyncStatus(context.Background(), &SyncStatusInput{
		PlatformID:        1,
		PlatformProductID: "test-sku",
	})
	if err != nil {
		t.Fatalf("SyncStatus: %v", err)
	}
	if status != "synced" {
		t.Fatalf("expected status 'synced', got %q", status)
	}
}

// ---------------------------------------------------------------------------
// TestPushTracking
// ---------------------------------------------------------------------------

func TestRealAdapter_PushTracking(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	var shipCalled bool
	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`))
		},
		OzonRealFBSShip: func(w http.ResponseWriter, r *http.Request) {
			shipCalled = true
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			if body["posting_number"] != "post-001" || body["tracking_number"] != "TRACK123" {
				http.Error(w, `{"error":{"code":"BAD_REQUEST"}}`, 400)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result":true}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	_ = createTestAcct(t, db, 1, "test-token", 1*time.Hour, `{"client_id":"test-client","client_secret":"test-secret"}`)

	ok, err := adapter.PushTracking(context.Background(), &PushTrackingInput{
		PlatformID:     1,
		OrderSN:        "post-001",
		TrackingNumber: "TRACK123",
		CarrierCode:    "RUSSIAN_POST",
	})
	if err != nil {
		t.Fatalf("PushTracking: %v", err)
	}
	if !ok {
		t.Fatal("expected PushTracking to succeed")
	}
	if !shipCalled {
		t.Fatal("expected ship endpoint to be called")
	}
}

// ---------------------------------------------------------------------------
// TestPublish_NoSKUs
// ---------------------------------------------------------------------------

func TestRealAdapter_Publish_NoSKUs(t *testing.T) {
	adapter := NewOzonRealAdapter(nil, zap.NewNop())
	_, err := adapter.Publish(context.Background(), &PublishInput{})
	if err == nil {
		t.Fatal("expected error for empty SKUs")
	}
	if !strings.Contains(err.Error(), "no SKUs") {
		t.Fatalf("expected 'no SKUs' error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestPublish_NoAccount
// ---------------------------------------------------------------------------

func TestRealAdapter_Publish_NoAccount(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})
	adapter := testRealAdapter(t, "http://localhost:1", db)
	_, err := adapter.Publish(context.Background(), &PublishInput{
		PlatformID: 999,
		SKUs:       []PublishSKU{{SkuID: 1, SkuCode: "sku1"}},
	})
	if err == nil {
		t.Fatal("expected error for non-existent account")
	}
}

// ---------------------------------------------------------------------------
// TestGetCategoryTree
// ---------------------------------------------------------------------------

func TestRealAdapter_GetCategoryTree(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`))
		},
		OzonRealCategoryTree: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"result": [
					{"id": 17000000, "name": "Электроника", "children": [
						{"id": 17001000, "name": "Смартфоны", "children": []}
					]},
					{"id": 18000000, "name": "Одежда", "children": []}
				]
			}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	_ = createTestAcct(t, db, 1, "test-token", 1*time.Hour, `{"client_id":"test-client","client_secret":"test-secret"}`)

	nodes, err := adapter.GetCategoryTree(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetCategoryTree: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 top-level categories, got %d", len(nodes))
	}
	if nodes[0].ID != 17000000 {
		t.Fatalf("expected first category ID 17000000, got %d", nodes[0].ID)
	}
	if len(nodes[0].Children) != 1 {
		t.Fatalf("expected 1 child for electronics, got %d", len(nodes[0].Children))
	}
	if nodes[0].Children[0].Name != "Смартфоны" {
		t.Fatalf("expected child name 'Смартфоны', got %q", nodes[0].Children[0].Name)
	}
}

// ---------------------------------------------------------------------------
// TestGetCategoryAttributes
// ---------------------------------------------------------------------------

func TestRealAdapter_GetCategoryAttributes(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`))
		},
		OzonRealCategoryAttribute: func(w http.ResponseWriter, r *http.Request) {
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			cid := int(body["category_id"].(float64))
			if cid != 17001000 {
				http.Error(w, `{"error":{"code":"BAD_REQUEST","message":"unexpected category"}}`, 400)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"result": [
					{"id": 1001, "name": "Бренд", "required": true, "type": "string"},
					{"id": 1002, "name": "Цвет", "required": false, "type": "string"}
				]
			}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	_ = createTestAcct(t, db, 1, "test-token", 1*time.Hour, `{"client_id":"test-client","client_secret":"test-secret"}`)

	attrs, err := adapter.GetCategoryAttributes(context.Background(), 1, 17001000)
	if err != nil {
		t.Fatalf("GetCategoryAttributes: %v", err)
	}
	if len(attrs) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(attrs))
	}
	if attrs[0].ID != 1001 || attrs[0].Name != "Бренд" || !attrs[0].Required {
		t.Fatalf("unexpected first attribute: %+v", attrs[0])
	}
	if attrs[1].ID != 1002 || attrs[1].Name != "Цвет" || attrs[1].Required {
		t.Fatalf("unexpected second attribute: %+v", attrs[1])
	}
}

// ---------------------------------------------------------------------------
// TestFetchSettlements
// ---------------------------------------------------------------------------

func TestRealAdapter_FetchSettlements(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`))
		},
		OzonRealFinanceTransactionList: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"result": {
					"operations": [
						{
							"operation_id": "tx-001",
							"operation_type": "sale",
							"amount": "2500.00",
							"currency_code": "RUB",
							"operation_date": "2026-06-15T12:00:00Z",
							"description": "Order #123",
							"posting": {"posting_number": "post-001"}
						},
						{
							"operation_id": "tx-002",
							"operation_type": "commission",
							"amount": "-250.00",
							"currency_code": "RUB",
							"operation_date": "2026-06-15T12:00:00Z",
							"description": "Commission",
							"posting": {"posting_number": "post-001"}
						}
					]
				}
			}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	_ = createTestAcct(t, db, 1, "test-token", 1*time.Hour, `{"client_id":"test-client","client_secret":"test-secret"}`)

	since := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	items, err := adapter.FetchSettlements(context.Background(), &FetchSettlementsInput{
		PlatformID: 1,
		Since:      since,
	})
	if err != nil {
		t.Fatalf("FetchSettlements: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 settlements, got %d", len(items))
	}
	if items[0].TransactionType != "order_sale" {
		t.Fatalf("expected type 'order_sale', got %q", items[0].TransactionType)
	}
	if items[1].TransactionType != "platform_fee" {
		t.Fatalf("expected type 'platform_fee', got %q", items[1].TransactionType)
	}
}

// ---------------------------------------------------------------------------
// TestFetchReturns
// ---------------------------------------------------------------------------

func TestRealAdapter_FetchReturns(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`))
		},
		OzonRealReturnsList: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"result": {
					"returns": [
						{
							"return_id": "ret-001",
							"posting_number": "post-001",
							"sku": "sku-1",
							"quantity": 1,
							"reason": "defective",
							"status": "accepted",
							"created_at": "2026-06-20T10:00:00Z",
							"refund_amount": "1999.99"
						}
					]
				}
			}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	_ = createTestAcct(t, db, 1, "test-token", 1*time.Hour, `{"client_id":"test-client","client_secret":"test-secret"}`)

	since := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	items, err := adapter.FetchReturns(context.Background(), &FetchReturnsInput{
		PlatformID: 1,
		Since:      since,
	})
	if err != nil {
		t.Fatalf("FetchReturns: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 return, got %d", len(items))
	}
	if items[0].ReturnID != "ret-001" {
		t.Fatalf("expected ReturnID 'ret-001', got %q", items[0].ReturnID)
	}
	if items[0].SkuCode != "sku-1" {
		t.Fatalf("expected SkuCode 'sku-1', got %q", items[0].SkuCode)
	}
	if items[0].Reason != "defective" {
		t.Fatalf("expected Reason 'defective', got %q", items[0].Reason)
	}
}

// ---------------------------------------------------------------------------
// TestInitRealAdapters
// ---------------------------------------------------------------------------

func TestInitRealAdapters_ReplacesStubs(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	// Before InitRealAdapters, GetAdapter returns the stub.
	adapter, ok := GetAdapter("ozon")
	if !ok {
		t.Fatal("expected 'ozon' adapter to be available before InitRealAdapters")
	}
	_, isStub := adapter.(*OzonAdapter)
	if !isStub {
		t.Fatal("expected 'ozon' adapter to be *OzonAdapter (stub) before InitRealAdapters")
	}

	// InitRealAdapters replaces stubs with real adapters.
	InitRealAdapters(db, zap.NewNop())

	adapter, ok = GetAdapter("ozon")
	if !ok {
		t.Fatal("expected 'ozon' adapter after InitRealAdapters")
	}
	_, isReal := adapter.(*OzonRealAdapter)
	if !isReal {
		t.Fatal("expected 'ozon' adapter to be *OzonRealAdapter after InitRealAdapters")
	}

	// Second call is a no-op (should not panic).
	InitRealAdapters(db, zap.NewNop())
}

// ---------------------------------------------------------------------------
// TestOzonRealError
// ---------------------------------------------------------------------------

func TestOzonRealError_Format(t *testing.T) {
	err := &OzonRealError{Code: "ACCESS_DENIED", Message: "invalid token", StatusCode: 403}
	want := "ozon [ACCESS_DENIED] HTTP 403: invalid token"
	if got := err.Error(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

// ---------------------------------------------------------------------------
// TestExchangeTokenError
// ---------------------------------------------------------------------------

func TestRealAdapter_ExchangeTokenError(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"error":{"code":"AUTH_FAILED","message":"bad credentials"}}`, 401)
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	_ = createTestAcct(t, db, 1, "", 0, `{"client_id":"test-client","client_secret":"bad-secret"}`)

	_, err := adapter.ValidateCredentials(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for bad credentials")
	}
	if !strings.Contains(err.Error(), "AUTH_FAILED") {
		t.Fatalf("expected error to contain 'AUTH_FAILED', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestNoCredentials
// ---------------------------------------------------------------------------
// TestSyncPrice (convenience method)
// ---------------------------------------------------------------------------

func TestRealAdapter_SyncPrice(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	var priceCalled bool
	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`))
		},
		OzonRealPriceImport: func(w http.ResponseWriter, r *http.Request) {
			priceCalled = true
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			prices, ok := body["prices"].([]interface{})
			if !ok || len(prices) != 1 {
				http.Error(w, `{"error":{"code":"BAD_REQUEST"}}`, 400)
				return
			}
			entry := prices[0].(map[string]interface{})
			if entry["offer_id"] != "test-sku" || entry["price"] != "2999.00" {
				http.Error(w, `{"error":{"code":"BAD_REQUEST"}}`, 400)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result":true}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	acct := createTestAcct(t, db, 1, "test-token", 1*time.Hour, `{"client_id":"test-client","client_secret":"test-secret"}`)

	err := adapter.SyncPrice(context.Background(), acct, "test-sku", "2999.00")
	if err != nil {
		t.Fatalf("SyncPrice: %v", err)
	}
	if !priceCalled {
		t.Fatal("expected price endpoint to be called")
	}
}

// ---------------------------------------------------------------------------
// TestConcurrentAccess (safe token refresh under load)
// ---------------------------------------------------------------------------

func TestRealAdapter_ConcurrentAccess(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	var tokenCalls atomic.Int64
	srv := newTestOzonServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			tokenCalls.Add(1)
			// Simulate slow token endpoint.
			time.Sleep(50 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"concurrent-token","token_type":"Bearer","expires_in":3600}`))
		},
		OzonRealProductList: func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("Api-Key")
			if apiKey == "" {
				http.Error(w, `{"error":{"code":"NO_AUTH"}}`, 401)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result":{"items":[],"total":0}}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(t, srv.URL, db)
	// No token — will trigger exchange on first call.
	_ = createTestAcct(t, db, 1, "", 0, `{"client_id":"test-client","client_secret":"test-secret"}`)

	// Fire 10 concurrent validation requests.
	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := adapter.ValidateCredentials(context.Background(), 1)
			errs <- err
		}()
	}

	for i := 0; i < 10; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent validate: %v", err)
		}
	}

	// Token should only have been exchanged once (due to authMu).
	if n := tokenCalls.Load(); n != 1 {
		t.Fatalf("expected exactly 1 token exchange, got %d", n)
	}
}

// ---------------------------------------------------------------------------
// Test that InitRealAdapters double-call doesn't panic
// ---------------------------------------------------------------------------

func TestInitRealAdapters_Idempotent(t *testing.T) {
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{})

	// Multiple calls should not panic.
	InitRealAdapters(db, zap.NewNop())
	InitRealAdapters(db, zap.NewNop())
	InitRealAdapters(db, zap.NewNop())

	adapter, ok := GetAdapter("ozon")
	if !ok {
		t.Fatal("expected 'ozon' adapter")
	}
	if _, ok := adapter.(*OzonRealAdapter); !ok {
		t.Fatal("expected *OzonRealAdapter")
	}
}

// ---------------------------------------------------------------------------
// Benchmark (lightweight — just HTTP call overhead)
// ---------------------------------------------------------------------------

func BenchmarkRealAdapter_ValidateCredentials(b *testing.B) {
	db := dbtest.NewDB(b, &PlatformIntegrationAccount{})

	srv := newTestOzonServer(b, map[string]func(w http.ResponseWriter, r *http.Request){
		OzonRealTokenEndpoint: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"bench-token","token_type":"Bearer","expires_in":86400}`))
		},
		OzonRealProductList: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result":{"items":[],"total":0}}`))
		},
	})
	defer srv.Close()

	adapter := testRealAdapter(b, srv.URL, db)
	_ = createTestAcct(b, db, 1, "bench-token", 1*time.Hour, `{"client_id":"bench-client","client_secret":"bench-secret"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.ValidateCredentials(context.Background(), 1)
	}
}

// ---------------------------------------------------------------------------
//  Edge case: empty config JSON
// ensure the test file belongs to the integrations package and compiles
var _ = fmt.Sprintf("test build marker")
