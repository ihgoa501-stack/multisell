package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Ozon Seller API base URL and endpoint paths.
const (
	OzonRealAPIV1BasePath = "https://api.ozon.ru"

	OzonRealTokenEndpoint          = "/v1/credentials/token"
	OzonRealProductImport          = "/v2/product/import"
	OzonRealProductImportInfo      = "/v2/product/import/info"
	OzonRealStockImport            = "/v1/product/import/stocks"
	OzonRealPriceImport            = "/v1/product/import/prices"
	OzonRealFBSList                = "/v2/posting/fbs/list"
	OzonRealFBSGet                 = "/v2/posting/fbs/get"
	OzonRealFBSShip                = "/v3/posting/fbs/ship"
	OzonRealCategoryTree           = "/v1/category/tree"
	OzonRealCategoryAttribute      = "/v1/category/attribute"
	OzonRealProductList            = "/v1/product/list"
	OzonRealProductInfo            = "/v1/product/info"
	OzonRealProductInfoList        = "/v3/product/info/list"
	OzonRealFinanceTransactionList = "/v3/finance/transaction/list"
	OzonRealReturnsList            = "/v3/returns/list"

	ozonRealDefaultTimeout     = 30 * time.Second
	ozonRealPollMaxAttempts    = 10
	ozonRealPollInterval       = 3 * time.Second
	ozonRealMaxRetries         = 3
	ozonRealTokenRefreshLeeway = 5 * time.Minute
)

// OzonRealAdapter implements PlatformAdapter for the real Ozon Seller API.
//
// Authentication uses the token-based flow:
//  1. client_id and client_secret are stored in the PlatformIntegrationAccount
//     config JSON field.
//  2. POST /v1/credentials/token exchanges credentials for a bearer access_token.
//  3. The token is stored in access_token / token_expires_at on the account record.
//  4. Tokens are automatically refreshed before expiry.
//
// For legacy setups, the stored access_token may also be used directly as an
// Api-Key header value (header-based auth), retaining backward compatibility.
type OzonRealAdapter struct {
	httpClient *http.Client
	baseURL    string
	db         *gorm.DB
	logger     *zap.Logger

	authMu sync.Mutex // guards token refresh against concurrent calls
}

// NewOzonRealAdapter creates a new OzonRealAdapter.
func NewOzonRealAdapter(db *gorm.DB, logger *zap.Logger) *OzonRealAdapter {
	return &OzonRealAdapter{
		httpClient: &http.Client{
			Timeout: ozonRealDefaultTimeout,
			Transport: &http.Transport{
				MaxIdleConns:       20,
				IdleConnTimeout:    90 * time.Second,
				DisableCompression: false,
			},
		},
		baseURL: OzonRealAPIV1BasePath,
		db:      db,
		logger:  logger,
	}
}

// ---------------------------------------------------------------------------
// Auth helpers
// ---------------------------------------------------------------------------

// ozonRealCredentials holds the Ozon API client credentials stored in the
// PlatformIntegrationAccount config field.
type ozonRealCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// ozonRealTokenResponse is the response body from POST /v1/credentials/token.
type ozonRealTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// getCredentials parses the client_id / client_secret from the account config.
func (a *OzonRealAdapter) getCredentials(acct *PlatformIntegrationAccount) (*ozonRealCredentials, error) {
	var creds ozonRealCredentials
	if len(acct.Config) > 0 {
		if err := json.Unmarshal(acct.Config, &creds); err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
	}
	if creds.ClientID == "" {
		return nil, fmt.Errorf("client_id not configured in account config")
	}
	return &creds, nil
}

// exchangeToken calls POST /v1/credentials/token and returns the response.
func (a *OzonRealAdapter) exchangeToken(ctx context.Context, clientID, clientSecret string) (*ozonRealTokenResponse, error) {
	payload := map[string]string{
		"client_id":     clientID,
		"client_secret": clientSecret,
	}

	body, err := a.doHTTP(ctx, http.MethodPost, OzonRealTokenEndpoint, "", "", payload)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}

	var tr ozonRealTokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	if tr.AccessToken == "" {
		return nil, fmt.Errorf("empty access_token in token response")
	}
	return &tr, nil
}

// refreshAccountToken exchanges the stored credentials for a new token and
// persists it on the account record. Caller must hold a.authMu.
func (a *OzonRealAdapter) refreshAccountToken(ctx context.Context, acct *PlatformIntegrationAccount) error {
	creds, err := a.getCredentials(acct)
	if err != nil {
		return err
	}

	tr, err := a.exchangeToken(ctx, creds.ClientID, creds.ClientSecret)
	if err != nil {
		return err
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(tr.ExpiresIn) * time.Second)

	updates := map[string]interface{}{
		"access_token":     tr.AccessToken,
		"token_expires_at": &expiresAt,
	}
	if err := a.db.Model(acct).Updates(updates).Error; err != nil {
		return fmt.Errorf("persist token: %w", err)
	}

	acct.AccessToken = tr.AccessToken
	acct.TokenExpiresAt = &expiresAt

	if a.logger != nil {
		a.logger.Info("ozon_real: token refreshed",
			zap.Int64("account_id", acct.ID),
			zap.Int("expires_in_s", tr.ExpiresIn))
	}
	return nil
}

// tokenIsValid reports whether the account's stored bearer token is still
// usable (present and not about to expire).
func (a *OzonRealAdapter) tokenIsValid(acct *PlatformIntegrationAccount) bool {
	if acct.AccessToken == "" {
		return false
	}
	if acct.TokenExpiresAt == nil {
		// No expiry — assume valid (legacy Api-Key mode).
		return true
	}
	return time.Now().Add(ozonRealTokenRefreshLeeway).Before(*acct.TokenExpiresAt)
}

// ensureValidToken checks the stored token and refreshes it if missing or
// expired. Safe for concurrent use.
//
// After acquiring the exclusive lock, it re-reads the account from the database
// to pick up any token that another goroutine may have just refreshed. This
// prevents a thundering-herd of concurrent refreshes for the same account.
func (a *OzonRealAdapter) ensureValidToken(ctx context.Context, acct *PlatformIntegrationAccount) error {
	a.authMu.Lock()
	defer a.authMu.Unlock()

	// Re-read from DB to get the latest token state. Another goroutine may have
	// refreshed it while we were waiting for the lock.
	var current PlatformIntegrationAccount
	if err := a.db.WithContext(ctx).First(&current, acct.ID).Error; err != nil {
		return fmt.Errorf("re-read account: %w", err)
	}

	if a.tokenIsValid(&current) {
		// Propagate the valid token to the caller's account object.
		acct.AccessToken = current.AccessToken
		acct.TokenExpiresAt = current.TokenExpiresAt
		return nil
	}

	if a.logger != nil {
		a.logger.Info("ozon_real: token missing or expired, refreshing",
			zap.Int64("account_id", acct.ID))
	}
	return a.refreshAccountToken(ctx, acct)
}

// ---------------------------------------------------------------------------
// Account lookup
// ---------------------------------------------------------------------------

func (a *OzonRealAdapter) getAccount(ctx context.Context, accountID int64) (*PlatformIntegrationAccount, error) {
	var acct PlatformIntegrationAccount
	if err := a.db.WithContext(ctx).First(&acct, accountID).Error; err != nil {
		return nil, fmt.Errorf("account %d: %w", accountID, err)
	}
	return &acct, nil
}

func (a *OzonRealAdapter) getActiveAccountByPlatform(ctx context.Context, platformID int64) (*PlatformIntegrationAccount, error) {
	var accts []PlatformIntegrationAccount
	if err := a.db.WithContext(ctx).
		Where("platform_id = ? AND status = ?", platformID, "active").
		Limit(1).
		Find(&accts).Error; err != nil {
		return nil, fmt.Errorf("find account: %w", err)
	}
	if len(accts) == 0 {
		return nil, fmt.Errorf("no active account for platform_id=%d", platformID)
	}
	return &accts[0], nil
}

// ---------------------------------------------------------------------------
// OzonRealError
// ---------------------------------------------------------------------------

// OzonRealError is the structured error returned by the Ozon Seller API.
type OzonRealError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
}

func (e *OzonRealError) Error() string {
	return fmt.Sprintf("ozon [%s] HTTP %d: %s", e.Code, e.StatusCode, e.Message)
}

// parseAPIError extracts an OzonRealError from an error response body.
func (a *OzonRealAdapter) parseAPIError(body []byte, statusCode int, _ string) *OzonRealError {
	var raw struct {
		Error   *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	err := &OzonRealError{StatusCode: statusCode}
	if json.Unmarshal(body, &raw) == nil {
		if raw.Error != nil {
			err.Code = raw.Error.Code
			err.Message = raw.Error.Message
		} else if raw.Code != "" {
			err.Code = raw.Code
			err.Message = raw.Message
		} else if raw.Message != "" {
			err.Code = fmt.Sprintf("HTTP_%d", statusCode)
			err.Message = raw.Message
		} else {
			err.Code = fmt.Sprintf("HTTP_%d", statusCode)
			err.Message = truncStr(string(body), 300)
		}
	} else {
		err.Code = fmt.Sprintf("HTTP_%d", statusCode)
		err.Message = truncStr(string(body), 300)
	}
	return err
}

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

// doHTTP performs a raw HTTP request. For the credentials endpoint both
// clientID and token are empty. For regular API calls both are passed; the
// method uses the Api-Key header which accepts either a legacy API key or a
// bearer access_token from the credentials endpoint.
func (a *OzonRealAdapter) doHTTP(ctx context.Context, method, path, clientID, token string, payload interface{}) ([]byte, error) {
	url := a.baseURL + path

	var bodyReader io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if clientID != "" {
		req.Header.Set("Client-Id", clientID)
	}
	if token != "" {
		req.Header.Set("Api-Key", token)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body %s: %w", path, err)
	}

	if resp.StatusCode >= 400 {
		return nil, a.parseAPIError(body, resp.StatusCode, path)
	}
	return body, nil
}

// doAuthenticated makes an API call with automatic token management and retry.
// It validates/refreshes the token before the first attempt and retries once
// on 401 (triggering a token refresh in between). 429 and 5xx are retried
// with exponential backoff via doWithRetry.
func (a *OzonRealAdapter) doAuthenticated(ctx context.Context, method, path string, acct *PlatformIntegrationAccount, payload interface{}) ([]byte, error) {
	// Validate / refresh token before the call.
	if !a.tokenIsValid(acct) {
		if err := a.ensureValidToken(ctx, acct); err != nil {
			return nil, fmt.Errorf("%s: auth: %w", path, err)
		}
	}

	creds, _ := a.getCredentials(acct)
	clientID := ""
	if creds != nil {
		clientID = creds.ClientID
	}

	body, err := a.doWithRetry(ctx, method, path, clientID, acct.AccessToken, payload, 0)
	if err == nil {
		return body, nil
	}

	// 401 -> try once with a fresh token (force refresh, ignore stored expiry).
	var apiErr *OzonRealError
	if isOzonRealError(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized {
		if a.logger != nil {
			a.logger.Warn("ozon_real: 401 received, forcing token refresh and retrying",
				zap.String("path", path), zap.Int64("account_id", acct.ID))
		}
		// Force refresh directly (don't use ensureValidToken which checks expiry).
		// The 401 tells us the server rejected the token regardless of what our
		// stored expiry says.
		a.authMu.Lock()
		rerr := a.refreshAccountToken(ctx, acct)
		a.authMu.Unlock()
		if rerr != nil {
			return nil, fmt.Errorf("%s: 401 + refresh failed: %w (original: %w)", path, rerr, err)
		}
		// Retry once with the new token.
		return a.doWithRetry(ctx, method, path, clientID, acct.AccessToken, payload, 0)
	}

	return nil, err
}

// doWithRetry calls doHTTP with exponential-backoff retry for 429 and 5xx.
func (a *OzonRealAdapter) doWithRetry(ctx context.Context, method, path, clientID, token string, payload interface{}, depth int) ([]byte, error) {
	if depth > 0 {
		// Already retrying — add a small backoff.
		backoff := time.Duration(math.Pow(2, float64(depth-1))) * 500 * time.Millisecond
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	body, err := a.doHTTP(ctx, method, path, clientID, token, payload)
	if err == nil {
		return body, nil
	}

	var apiErr *OzonRealError
	if !isOzonRealError(err, &apiErr) {
		return nil, err
	}

	switch {
	case apiErr.StatusCode == http.StatusTooManyRequests && depth < ozonRealMaxRetries:
		if a.logger != nil {
			a.logger.Warn("ozon_real: rate limited, retrying",
				zap.String("path", path), zap.Int("attempt", depth+1))
		}
		return a.doWithRetry(ctx, method, path, clientID, token, payload, depth+1)

	case apiErr.StatusCode >= http.StatusInternalServerError && depth < ozonRealMaxRetries:
		if a.logger != nil {
			a.logger.Warn("ozon_real: server error, retrying",
				zap.String("path", path), zap.Int("status", apiErr.StatusCode), zap.Int("attempt", depth+1))
		}
		return a.doWithRetry(ctx, method, path, clientID, token, payload, depth+1)

	default:
		return nil, err
	}
}

func isOzonRealError(err error, out **OzonRealError) bool {
	if err == nil || out == nil {
		return false
	}
	e, ok := err.(*OzonRealError)
	if !ok {
		return false
	}
	*out = e
	return true
}

// ---------------------------------------------------------------------------
// PlatformAdapter implementation
// ---------------------------------------------------------------------------

// Publish imports a product to Ozon via the async v2 import API and polls
// until the import task completes or times out.
func (a *OzonRealAdapter) Publish(ctx context.Context, input *PublishInput) (*PublishResult, error) {
	if len(input.SKUs) == 0 {
		return nil, fmt.Errorf("ozon_real publish: no SKUs")
	}

	acct, err := a.getActiveAccountByPlatform(ctx, input.PlatformID)
	if err != nil {
		return nil, fmt.Errorf("ozon_real publish: %w", err)
	}

	// Build product items payload.
	type skuItem struct {
		OfferID      string                   `json:"offer_id"`
		Name         string                   `json:"name"`
		Description  string                   `json:"description,omitempty"`
		CategoryID   int64                    `json:"category_id"`
		Price        string                   `json:"price"`
		CurrencyCode string                   `json:"currency_code,omitempty"`
		Height       string                   `json:"height,omitempty"`
		Width        string                   `json:"width,omitempty"`
		Depth        string                   `json:"depth,omitempty"`
		Weight       string                   `json:"weight,omitempty"`
		Images       []map[string]string      `json:"images,omitempty"`
		Attributes   []map[string]interface{} `json:"attributes,omitempty"`
	}

	items := make([]skuItem, 0, len(input.SKUs))
	for _, sku := range input.SKUs {
		price := safeFloat(input.Prices[sku.SkuID])

		item := skuItem{
			OfferID:      sku.SkuCode,
			Name:         input.ProductName,
			Description:  input.Description,
			CategoryID:   input.CategoryID,
			Price:        fmt.Sprintf("%.2f", price),
			CurrencyCode: "RUB",
			Height:       fmt.Sprintf("%.1f", input.PackageHeight),
			Width:        fmt.Sprintf("%.1f", input.PackageWidth),
			Depth:        fmt.Sprintf("%.1f", input.PackageLength),
			Weight:       fmt.Sprintf("%.1f", input.PackageWeight),
		}

		// Collect images.
		var images []map[string]string
		if input.MainImage != "" {
			images = append(images, map[string]string{"file_name": input.MainImage})
		}
		for _, img := range input.Images {
			images = append(images, map[string]string{"file_name": img})
		}
		if len(images) > 0 {
			item.Images = images
		}

		items = append(items, item)
	}

	// Step 1: submit the import task.
	payload := map[string]interface{}{"items": items}
	body, err := a.doAuthenticated(ctx, http.MethodPost, OzonRealProductImport, acct, payload)
	if err != nil {
		return nil, fmt.Errorf("ozon_real publish import: %w", err)
	}

	var importResult struct {
		Result struct {
			TaskID int64 `json:"task_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &importResult); err != nil {
		return nil, fmt.Errorf("ozon_real parse import result: %w", err)
	}

	taskID := importResult.Result.TaskID
	if taskID == 0 {
		return nil, fmt.Errorf("ozon_real publish: empty task_id in response")
	}

	offerID := input.SKUs[0].SkuCode

	// Step 2: poll the import status.
	status, pollErr := a.pollImportStatus(ctx, acct, taskID, offerID)
	result := &PublishResult{
		PlatformProductID: fmt.Sprintf("%d", taskID),
		PlatformSKU:       offerID,
		PlatformURL:       fmt.Sprintf("https://www.ozon.ru/product/ozon-%s/", offerID),
		PublishedData: map[string]interface{}{
			"task_id": taskID,
			"status":  status,
		},
		SyncMessage: fmt.Sprintf("ozon import task_id=%d status=%s", taskID, status),
	}
	if pollErr != nil {
		result.SyncMessage = fmt.Sprintf("ozon import task_id=%d status=%s error=%v", taskID, status, pollErr)
	}
	return result, pollErr
}

// pollImportStatus polls the v2 import info endpoint until the offer reaches a
// terminal state or the context is cancelled / max attempts exhausted.
func (a *OzonRealAdapter) pollImportStatus(ctx context.Context, acct *PlatformIntegrationAccount, taskID int64, offerID string) (string, error) {
	for attempt := 0; attempt < ozonRealPollMaxAttempts; attempt++ {
		select {
		case <-time.After(ozonRealPollInterval):
		case <-ctx.Done():
			return "cancelled", ctx.Err()
		}

		body, err := a.doAuthenticated(ctx, http.MethodPost, OzonRealProductImportInfo, acct,
			map[string]interface{}{"task_id": taskID})
		if err != nil {
			a.logger.Warn("ozon_real: import info poll error",
				zap.Int64("task_id", taskID), zap.Int("attempt", attempt), zap.Error(err))
			continue
		}

		var info struct {
			Result struct {
				Items []struct {
					OfferID string `json:"offer_id"`
					Status  string `json:"status"`
					Errors  []struct {
						Code    string `json:"code"`
						Message string `json:"message"`
					} `json:"errors,omitempty"`
				} `json:"items"`
			} `json:"result"`
		}
		if err := json.Unmarshal(body, &info); err != nil {
			a.logger.Warn("ozon_real: parse import info poll response",
				zap.Int64("task_id", taskID), zap.Error(err))
			continue
		}

		for _, item := range info.Result.Items {
			if item.OfferID != offerID && item.OfferID != "" {
				continue
			}
			switch item.Status {
			case "imported", "processed":
				return item.Status, nil
			case "failed", "rejected":
				errMsg := ""
				if len(item.Errors) > 0 {
					errMsg = item.Errors[0].Message
				}
				return item.Status, fmt.Errorf("import failed: %s", errMsg)
			default:
				// Still pending — continue polling.
			}
		}
	}

	return "timeout", fmt.Errorf("import poll timed out after %d attempts", ozonRealPollMaxAttempts)
}

// SyncStatus checks the current state of a product on Ozon.
func (a *OzonRealAdapter) SyncStatus(ctx context.Context, input *SyncStatusInput) (string, error) {
	acct, err := a.getActiveAccountByPlatform(ctx, input.PlatformID)
	if err != nil {
		return "unknown", fmt.Errorf("ozon_real sync_status: %w", err)
	}

	body, err := a.doAuthenticated(ctx, http.MethodPost, OzonRealProductInfo, acct,
		map[string]interface{}{"offer_id": input.PlatformProductID})
	if err != nil {
		return "unknown", fmt.Errorf("ozon_real sync_status: %w", err)
	}

	var info struct {
		Result struct {
			State string `json:"state"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return "unknown", fmt.Errorf("ozon_real parse product info: %w", err)
	}

	stateMap := map[string]string{
		"imported":                       "synced",
		"processed":                      "synced",
		"processing":                     "in_progress",
		"created":                        "pending",
		"failed":                         "failed",
		"rejected":                       "failed",
		"state_import_completed":         "synced",
		"state_import_process":           "in_progress",
		"state_import_pending":           "pending",
		"state_import_validation":        "pending",
		"state_import_validation_failed": "failed",
		"state_import_processing":        "in_progress",
		"state_import_error":             "failed",
		"state_import_ready":             "synced",
	}
	if s, ok := stateMap[info.Result.State]; ok {
		return s, nil
	}
	return info.Result.State, nil
}

// ValidateCredentials verifies the stored credentials by calling a
// lightweight API endpoint (product list with page_size=1).
func (a *OzonRealAdapter) ValidateCredentials(ctx context.Context, accountID int64) (bool, error) {
	acct, err := a.getAccount(ctx, accountID)
	if err != nil {
		return false, fmt.Errorf("ozon_real validate: %w", err)
	}

	_, err = a.doAuthenticated(ctx, http.MethodPost, OzonRealProductList, acct,
		map[string]interface{}{"page": 1, "page_size": 1})
	if err != nil {
		return false, fmt.Errorf("ozon_real validate: %w", err)
	}
	return true, nil
}

// SyncInventory pushes stock levels to Ozon via POST /v1/product/import/stocks.
func (a *OzonRealAdapter) SyncInventory(ctx context.Context, input *SyncInventoryInput) (bool, error) {
	acct, err := a.getActiveAccountByPlatform(ctx, input.PlatformID)
	if err != nil {
		return false, fmt.Errorf("ozon_real sync_inventory: %w", err)
	}

	offerID := input.SkuCode
	if input.PlatformSKU != "" {
		offerID = input.PlatformSKU
	}

	payload := map[string]interface{}{
		"stocks": []map[string]interface{}{
			{
				"offer_id": offerID,
				"stock":    input.Quantity,
			},
		},
	}

	_, err = a.doAuthenticated(ctx, http.MethodPost, OzonRealStockImport, acct, payload)
	if err != nil {
		return false, fmt.Errorf("ozon_real sync_inventory: %w", err)
	}
	return true, nil
}

// SyncPrice pushes price updates to Ozon via POST /v1/product/import/prices.
// Not part of the PlatformAdapter interface but available for direct use.
func (a *OzonRealAdapter) SyncPrice(ctx context.Context, acct *PlatformIntegrationAccount, offerID, price string) error {
	payload := map[string]interface{}{
		"prices": []map[string]interface{}{
			{
				"offer_id":      offerID,
				"price":         price,
				"currency_code": "RUB",
			},
		},
	}
	_, err := a.doAuthenticated(ctx, http.MethodPost, OzonRealPriceImport, acct, payload)
	if err != nil {
		return fmt.Errorf("sync_price: %w", err)
	}
	return nil
}

// PushTracking sends tracking / shipping info to Ozon for an FBS posting.
func (a *OzonRealAdapter) PushTracking(ctx context.Context, input *PushTrackingInput) (bool, error) {
	acct, err := a.getActiveAccountByPlatform(ctx, input.PlatformID)
	if err != nil {
		return false, fmt.Errorf("ozon_real push_tracking: %w", err)
	}

	payload := map[string]interface{}{
		"posting_number":  input.OrderSN,
		"tracking_number": input.TrackingNumber,
	}
	if input.CarrierCode != "" {
		payload["carrier_code"] = input.CarrierCode
	}

	_, err = a.doAuthenticated(ctx, http.MethodPost, OzonRealFBSShip, acct, payload)
	if err != nil {
		return false, fmt.Errorf("ozon_real push_tracking: %w", err)
	}
	return true, nil
}

// FetchOrders pulls FBS orders from Ozon since the given timestamp. Uses
// offset-based pagination (limit=1000).
func (a *OzonRealAdapter) FetchOrders(ctx context.Context, input *FetchOrdersInput) ([]*PlatformOrder, error) {
	acct, err := a.getActiveAccountByPlatform(ctx, input.PlatformID)
	if err != nil {
		return nil, fmt.Errorf("ozon_real fetch_orders: %w", err)
	}

	var orders []*PlatformOrder
	offset := 0
	limit := 1000

	for {
		payload := map[string]interface{}{
			"dir": "ASC",
			"filter": map[string]string{
				"since": input.Since.Format("2006-01-02T15:04:05.000Z"),
			},
			"limit":  limit,
			"offset": offset,
		}

		body, err := a.doAuthenticated(ctx, http.MethodPost, OzonRealFBSList, acct, payload)
		if err != nil {
			return nil, fmt.Errorf("ozon_real fetch_orders offset=%d: %w", offset, err)
		}

		var listResult struct {
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
		if err := json.Unmarshal(body, &listResult); err != nil {
			return nil, fmt.Errorf("ozon_real parse posting list: %w", err)
		}

		if len(listResult.Result.Postings) == 0 {
			break
		}

		for _, p := range listResult.Result.Postings {
			var items []PlatformOrderItem
			total := 0.0
			for _, prod := range p.FinancialData.Products {
				price := sf(prod.Price)
				total += price * float64(prod.Quantity)
				items = append(items, PlatformOrderItem{
					SkuCode:   prod.Sku,
					Quantity:  prod.Quantity,
					UnitPrice: prod.Price,
				})
			}
			orders = append(orders, &PlatformOrder{
				OrderSN:     p.PostingNumber,
				Status:      p.Status,
				TotalAmount: ff(total),
				ShippingFee: p.AnalyticsData.DeliveryPrice,
				PaidAt:      p.InProcessAt,
				Items:       items,
			})
		}

		if len(listResult.Result.Postings) < limit {
			break
		}
		offset += limit
	}

	return orders, nil
}

// FetchSettlements pulls finance transaction records from Ozon.
func (a *OzonRealAdapter) FetchSettlements(ctx context.Context, input *FetchSettlementsInput) ([]*PlatformSettlement, error) {
	acct, err := a.getActiveAccountByPlatform(ctx, input.PlatformID)
	if err != nil {
		return nil, fmt.Errorf("ozon_real fetch_settlements: %w", err)
	}

	payload := map[string]interface{}{
		"filter": map[string]interface{}{
			"date": map[string]string{"from": input.Since.Format("2006-01-02T15:04:05.000Z")},
		},
		"page":      1,
		"page_size": 100,
	}

	body, err := a.doAuthenticated(ctx, http.MethodPost, OzonRealFinanceTransactionList, acct, payload)
	if err != nil {
		return nil, fmt.Errorf("ozon_real fetch_settlements: %w", err)
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
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("ozon_real parse settlements: %w", err)
	}

	tm := map[string]string{
		"sale":               "order_sale",
		"refund":             "refund",
		"delivery":           "shipping_fee",
		"commission":         "platform_fee",
		"payment_commission": "payment_fee",
	}
	var items []*PlatformSettlement
	for _, tx := range r.Result.Operations {
		ttype := tm[tx.OperationType]
		if ttype == "" {
			ttype = "other"
		}
		items = append(items, &PlatformSettlement{
			TransactionID:   tx.OperationID,
			TransactionType: ttype,
			OrderSN:         tx.Posting.PostingNumber,
			Amount:          ff(absf(sf(tx.Amount))),
			Currency:        tx.CurrencyCode,
			OccurredAt:      tx.OperationDate,
			Description:     tx.Description,
		})
	}
	return items, nil
}

// FetchReturns pulls return/refund records from Ozon.
func (a *OzonRealAdapter) FetchReturns(ctx context.Context, input *FetchReturnsInput) ([]*PlatformReturn, error) {
	acct, err := a.getActiveAccountByPlatform(ctx, input.PlatformID)
	if err != nil {
		return nil, fmt.Errorf("ozon_real fetch_returns: %w", err)
	}

	payload := map[string]interface{}{
		"filter": map[string]string{
			"last_change_from": input.Since.Format("2006-01-02T15:04:05.000Z"),
		},
		"limit": 100,
	}

	body, err := a.doAuthenticated(ctx, http.MethodPost, OzonRealReturnsList, acct, payload)
	if err != nil {
		return nil, fmt.Errorf("ozon_real fetch_returns: %w", err)
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
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("ozon_real parse returns: %w", err)
	}

	var items []*PlatformReturn
	for _, ret := range r.Result.Returns {
		items = append(items, &PlatformReturn{
			ReturnID:     ret.ReturnID,
			OrderSN:      ret.PostingNumber,
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

// ---------------------------------------------------------------------------
// OzonReal-specific convenience methods (not in PlatformAdapter)
// ---------------------------------------------------------------------------

// OzonRealCategoryNode is a node in the Ozon category tree.
type OzonRealCategoryNode struct {
	ID       int64                  `json:"id"`
	Name     string                 `json:"name"`
	Children []*OzonRealCategoryNode `json:"children"`
}

// GetCategoryTree fetches the full category tree from Ozon.
func (a *OzonRealAdapter) GetCategoryTree(ctx context.Context, accountID int64) ([]*OzonRealCategoryNode, error) {
	acct, err := a.getAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	body, err := a.doAuthenticated(ctx, http.MethodPost, OzonRealCategoryTree, acct, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("get category tree: %w", err)
	}

	var result struct {
		Result []*OzonRealCategoryNode `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse category tree: %w", err)
	}
	return result.Result, nil
}

// OzonRealAttribute describes a category attribute on Ozon.
type OzonRealAttribute struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// GetCategoryAttributes fetches the attributes (required / optional) for a
// given Ozon category.
func (a *OzonRealAdapter) GetCategoryAttributes(ctx context.Context, accountID, categoryID int64) ([]*OzonRealAttribute, error) {
	acct, err := a.getAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	body, err := a.doAuthenticated(ctx, http.MethodPost, OzonRealCategoryAttribute, acct,
		map[string]interface{}{"category_id": categoryID})
	if err != nil {
		return nil, fmt.Errorf("get category attributes: %w", err)
	}

	var result struct {
		Result []*OzonRealAttribute `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse category attributes: %w", err)
	}
	return result.Result, nil
}
