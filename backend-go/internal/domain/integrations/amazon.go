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
	return &PublishResult{
		PlatformProductID: fmt.Sprintf("amazon-pending-%d", input.ProductID),
		PlatformSKU:       input.SKUs[0].SkuCode,
		PlatformURL:       "",
		PublishedData:     map[string]interface{}{"status": "stub"},
		SyncMessage:       "amazon adapter stub: publish not yet implemented",
	}, nil
}

func (a *AmazonAdapter) SyncStatus(_ context.Context, input *SyncStatusInput) (string, error) {
	if input.PlatformProductID == "" {
		return "unknown", fmt.Errorf("amazon sync_status: empty platform product id")
	}
	return "synced", nil
}

func (a *AmazonAdapter) ValidateCredentials(_ context.Context, accountID int64) (bool, error) {
	return false, fmt.Errorf("amazon ValidateCredentials: not yet implemented for account %d", accountID)
}

func (a *AmazonAdapter) SyncInventory(_ context.Context, input *SyncInventoryInput) (bool, error) {
	return false, fmt.Errorf("amazon sync_inventory: not yet implemented for sku %s", input.SkuCode)
}

func (a *AmazonAdapter) PushTracking(_ context.Context, input *PushTrackingInput) (bool, error) {
	return false, fmt.Errorf("amazon push_tracking: not yet implemented for order %s", input.OrderSN)
}

func (a *AmazonAdapter) FetchOrders(_ context.Context, input *FetchOrdersInput) ([]*PlatformOrder, error) {
	return []*PlatformOrder{}, nil
}

func (a *AmazonAdapter) FetchSettlements(_ context.Context, input *FetchSettlementsInput) ([]*PlatformSettlement, error) {
	return []*PlatformSettlement{}, nil
}

func (a *AmazonAdapter) FetchReturns(_ context.Context, input *FetchReturnsInput) ([]*PlatformReturn, error) {
	return []*PlatformReturn{}, nil
}
