package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"github.com/lingmirror/backend-go/internal/domain/integrations/aimapper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides integrations business logic.
type Service struct {
	db          *gorm.DB
	logger      *zap.Logger
	approvalSvc *approval.Service
}

// NewService creates a new integrations service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// WithApproval sets the optional approval service used by write-back gates.
func (s *Service) WithApproval(svc *approval.Service) *Service {
	s.approvalSvc = svc
	return s
}

// List returns paginated integration accounts with optional filter.
func (s *Service) List(p *common.Pagination, f *AccountListFilter) ([]PlatformIntegrationAccount, int64, error) {
	q := s.db.Model(&PlatformIntegrationAccount{})
	if f != nil {
		if f.Search != "" {
			like := "%" + f.Search + "%"
			q = q.Where("store_name ILIKE ? OR account_id ILIKE ?", like, like)
		}
		if f.PlatformID != nil {
			q = q.Where("platform_id = ?", *f.PlatformID)
		}
		if f.Status != "" {
			q = q.Where("status = ?", f.Status)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []PlatformIntegrationAccount
	if err := q.Order("id DESC").Offset(p.Offset()).Limit(p.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	s.enrichPlatformNames(items)
	return items, total, nil
}

// enrichPlatformNames populates PlatformName for each account via batch lookup.
func (s *Service) enrichPlatformNames(items []PlatformIntegrationAccount) {
	if len(items) == 0 {
		return
	}
	ids := make([]int64, len(items))
	for i, it := range items {
		ids[i] = it.PlatformID
	}
	type platRow struct {
		ID   int64
		Name string
	}
	var plats []platRow
	_ = s.db.Table("platform").Where("id IN ?", ids).Select("id, name").Find(&plats).Error
	m := make(map[int64]string, len(plats))
	for _, p := range plats {
		m[p.ID] = p.Name
	}
	for i := range items {
		items[i].PlatformName = m[ids[i]]
	}
}

// Get returns a single integration account.
func (s *Service) Get(id int64) (*PlatformIntegrationAccount, error) {
	var a PlatformIntegrationAccount
	if err := s.db.First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// Create inserts a new integration account.
func (s *Service) Create(in *CreateAccountInput) (*PlatformIntegrationAccount, error) {
	status := in.Status
	if status == "" {
		status = "active"
	}
	a := PlatformIntegrationAccount{
		PlatformID:     in.PlatformID,
		StoreName:      in.StoreName,
		AccountID:      in.AccountID,
		AccessToken:    in.AccessToken,
		RefreshToken:   in.RefreshToken,
		TokenExpiresAt: in.TokenExpiresAt,
		Status:         status,
		Config:         in.Config,
	}
	if err := s.db.Create(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// Update applies partial updates to an integration account.
func (s *Service) Update(id int64, in *UpdateAccountInput) (*PlatformIntegrationAccount, error) {
	var a PlatformIntegrationAccount
	if err := s.db.First(&a, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.StoreName != nil {
		updates["store_name"] = *in.StoreName
	}
	if in.AccountID != nil {
		updates["account_id"] = *in.AccountID
	}
	if in.AccessToken != nil {
		updates["access_token"] = *in.AccessToken
	}
	if in.RefreshToken != nil {
		updates["refresh_token"] = *in.RefreshToken
	}
	if in.TokenExpiresAt != nil {
		updates["token_expires_at"] = *in.TokenExpiresAt
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if in.SyncStatus != nil {
		updates["sync_status"] = *in.SyncStatus
	}
	if in.LastError != nil {
		updates["last_error"] = *in.LastError
	}
	if in.Config != nil {
		updates["config"] = *in.Config
	}
	if len(updates) == 0 {
		return &a, nil
	}
	if err := s.db.Model(&a).Updates(updates).Error; err != nil {
		return nil, err
	}
	// reload to reflect updates
	if err := s.db.First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// Delete removes an integration account and its mappings.
func (s *Service) Delete(id int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("account_id = ?", id).Delete(&PlatformCategoryMapping{}).Error; err != nil {
			return err
		}
		if err := tx.Where("account_id = ?", id).Delete(&PlatformAttributeMapping{}).Error; err != nil {
			return err
		}
		return tx.Delete(&PlatformIntegrationAccount{}, id).Error
	})
}

// TestConnection tests the connection to the platform. Simplified: always ok if account exists.
func (s *Service) TestConnection(id int64) (*TestConnectionResult, error) {
	var a PlatformIntegrationAccount
	if err := s.db.First(&a, id).Error; err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.AccessToken) == "" {
		return &TestConnectionResult{Success: false, Message: "access token is empty"}, nil
	}
	return &TestConnectionResult{Success: true, Message: "ok"}, nil
}

// ListOzonProducts fetches the product catalog from Ozon for the given
// integration account ID by delegating to the Ozon adapter.
func (s *Service) ListOzonProducts(ctx context.Context, accountID int64) ([]OzonProduct, error) {
	var a PlatformIntegrationAccount
	if err := s.db.First(&a, accountID).Error; err != nil {
		return nil, err
	}
	adapter, ok := GetAdapter("ozon")
	if !ok {
		return nil, errors.New("ozon adapter not registered")
	}
	ozon, ok := adapter.(*OzonAdapter)
	if !ok {
		return nil, errors.New("ozon adapter type assertion failed")
	}
	return ozon.ListProducts(ctx, a.PlatformID)
}

// TriggerSync marks the account as syncing.
func (s *Service) TriggerSync(id int64) (*PlatformIntegrationAccount, error) {
	var a PlatformIntegrationAccount
	if err := s.db.First(&a, id).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	if err := s.db.Model(&a).Updates(map[string]interface{}{
		"sync_status":  "syncing",
		"last_sync_at": now,
		"last_error":   "",
	}).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// GetMode returns the execution mode for an integration account.
func (s *Service) GetMode(id int64) (int8, error) {
	var a PlatformIntegrationAccount
	if err := s.db.Select("execution_mode").First(&a, id).Error; err != nil {
		return 0, err
	}
	return a.ExecutionMode, nil
}

// UpdateMode sets the execution mode for an integration account.
func (s *Service) UpdateMode(id int64, mode int8) error {
	if mode < int8(ExecutionModeDryRun) || mode > int8(ExecutionModeProduction) {
		return fmt.Errorf("unknown execution mode: %d", mode)
	}
	res := s.db.Model(&PlatformIntegrationAccount{}).Where("id = ?", id).
		Update("execution_mode", mode)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ListCategoryMappings returns the category mappings for an account.
func (s *Service) ListCategoryMappings(accountID int64) ([]PlatformCategoryMapping, error) {
	var items []PlatformCategoryMapping
	if err := s.db.Where("account_id = ?", accountID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// CreateCategoryMapping adds a category mapping for an account.
func (s *Service) CreateCategoryMapping(accountID int64, in *CreateCategoryMappingInput) (*PlatformCategoryMapping, error) {
	if accountID <= 0 {
		return nil, errors.New("account_id is required")
	}
	m := PlatformCategoryMapping{
		AccountID:            accountID,
		LocalCategoryID:      in.LocalCategoryID,
		PlatformCategoryID:   in.PlatformCategoryID,
		PlatformCategoryName: in.PlatformCategoryName,
	}
	if err := s.db.Create(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// ListAttributeMappings returns the attribute mappings for an account.
func (s *Service) ListAttributeMappings(accountID int64) ([]PlatformAttributeMapping, error) {
	var items []PlatformAttributeMapping
	if err := s.db.Where("account_id = ?", accountID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// CreateAttributeMapping adds an attribute mapping for an account.
func (s *Service) CreateAttributeMapping(accountID int64, in *CreateAttributeMappingInput) (*PlatformAttributeMapping, error) {
	if accountID <= 0 {
		return nil, errors.New("account_id is required")
	}
	m := PlatformAttributeMapping{
		AccountID:        accountID,
		LocalAttrName:    in.LocalAttrName,
		PlatformAttrID:   in.PlatformAttrID,
		PlatformAttrName: in.PlatformAttrName,
		Required:         in.Required,
	}
	if err := s.db.Create(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// PublishToOzonInput is the request payload for publishing a product to Ozon.
type PublishToOzonInput struct {
	ProductID    int64   `json:"product_id" binding:"required"`
	AccountID    int64   `json:"account_id" binding:"required"`
	Price        float64 `json:"price" binding:"required"`
	CurrencyCode string  `json:"currency_code"`
	ImageURL     string  `json:"image_url,omitempty"`
	ApprovalID   *int64  `json:"approval_id,omitempty"`
}

// ErrImageReleaseAttestationRequired permanently closes the legacy URL-based
// Ozon publisher for real writes. Production publication must use the
// productimage controlled byte publisher, which consumes a release
// attestation and does not trust caller-supplied image URLs.
var ErrImageReleaseAttestationRequired = errors.New("IMAGE_RELEASE_ATTESTATION_REQUIRED")

// PublishToOzon publishes a local product to the Ozon platform.
func (s *Service) PublishToOzon(ctx context.Context, in *PublishToOzonInput) (*PublishResult, error) {
	// This legacy endpoint accepts an arbitrary image_url and therefore can
	// never perform an external write. Keep dry-run/sandbox useful as explicit,
	// state-free mocks; every other/unknown mode fails before DB or adapter use.
	mode, explicitlySet := ctx.Value(modeCtxKey{}).(ExecutionMode)
	if !explicitlySet {
		return nil, ErrImageReleaseAttestationRequired
	}
	switch mode {
	case ExecutionModeDryRun, ExecutionModeSandbox:
		s.logger.Info("mock: skipping legacy Ozon platform write",
			zap.String("operation", "publish_ozon"),
			zap.String("execution_mode", mode.String()),
		)
		return &PublishResult{
			PlatformProductID: mode.String() + "-simulated",
			PlatformURL:       mode.String() + "://simulated",
			SyncMessage:       mode.String() + ": mock only; no platform call or state change was made",
		}, nil
	default:
		return nil, ErrImageReleaseAttestationRequired
	}
}

// checkWriteMode guards write operations based on the execution mode from context.
// Returns:
//   - (mockResult, nil) for dry-run mode (caller should return mockResult immediately)
//   - (nil, nil) for production/sandbox mode (caller should proceed)
//   - (nil, error) if mode validation fails
func (s *Service) checkWriteMode(ctx context.Context, op string, in interface{}) (*PublishResult, error) {
	mode := ExecutionModeFromCtx(ctx)
	switch mode {
	case ExecutionModeDryRun:
		s.logger.Info("dry-run: skipping platform write",
			zap.String("operation", op),
			zap.Any("input", in),
		)
		// Return a mock success result so the caller can proceed without errors.
		return &PublishResult{
			PlatformProductID: "dry-run-simulated",
			PlatformURL:       "dry-run://simulated",
			SyncMessage:       "dry_run: no platform call was made",
		}, nil
	case ExecutionModeSandbox:
		// Sandbox: pass through — the adapter uses sandbox API endpoints.
		return nil, nil
	case ExecutionModeApprovalRequired:
		if _, ok := ApprovalIDFromCtx(ctx); !ok {
			return nil, errors.New("execution mode is approval_required: approval context is required but not provided")
		}
		return nil, nil
	case ExecutionModeProduction:
		if _, ok := ApprovalIDFromCtx(ctx); !ok {
			return nil, errors.New("execution mode is production: approval context is required but not provided")
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown execution mode: %d (%s)", mode, mode.String())
	}
}

// SyncOzonOrders fetches new orders from all active Ozon accounts via the adapter.
// Called by the scheduler on a 15-minute interval.
func (s *Service) SyncOzonOrders(ctx context.Context) error {
	type accountRow struct {
		ID         int64
		PlatformID int64
	}
	var accounts []accountRow
	if err := s.db.Table("platform_integration_account AS a").
		Select("a.id, a.platform_id").
		Joins("JOIN platform p ON p.id = a.platform_id").
		Where("p.code = ? AND a.status = ?", "ozon", "active").
		Find(&accounts).Error; err != nil {
		return fmt.Errorf("sync ozon: %w", err)
	}
	if len(accounts) == 0 {
		return nil
	}
	adapter, ok := GetAdapter("ozon")
	if !ok {
		return fmt.Errorf("sync ozon: no adapter")
	}
	ozon, ok := adapter.(*OzonAdapter)
	if !ok {
		return fmt.Errorf("sync ozon: type error")
	}
	since := time.Now().Add(-72 * time.Hour)
	for _, acct := range accounts {
		orders, err := ozon.FetchOrders(ctx, &FetchOrdersInput{PlatformID: acct.PlatformID, Since: since})
		if err != nil {
			s.db.Table("platform_integration_account").Where("id = ?", acct.ID).Updates(
				map[string]interface{}{"last_error": err.Error(), "last_sync_at": time.Now(), "sync_status": "error"})
			s.logger.Warn("ozon sync error", zap.Int64("account", acct.ID), zap.Error(err))
			continue
		}
		s.logger.Info("ozon sync done", zap.Int64("account", acct.ID), zap.Int("orders", len(orders)))
		s.db.Table("platform_integration_account").Where("id = ?", acct.ID).Updates(
			map[string]interface{}{"last_sync_at": time.Now(), "last_error": "", "sync_status": "idle"})
	}
	return nil
}

// SyncOzonOrdersRaw fetches raw order data from Ozon, stores as RawEvent records,
// and triggers the AI mapping pipeline for each posting.
// Runs alongside the existing SyncOzonOrders (which logs structured order counts).
func (s *Service) SyncOzonOrdersRaw(ctx context.Context, pipeline *aimapper.Pipeline) error {
	type accountRow struct {
		ID         int64
		PlatformID int64
	}
	var accounts []accountRow
	if err := s.db.Table("platform_integration_account AS a").
		Select("a.id, a.platform_id").
		Joins("JOIN platform p ON p.id = a.platform_id").
		Where("p.code = ? AND a.status = ?", "ozon", "active").
		Find(&accounts).Error; err != nil {
		return fmt.Errorf("sync ozon raw: %w", err)
	}
	if len(accounts) == 0 {
		return nil
	}
	adapter, ok := GetAdapter("ozon")
	if !ok {
		return fmt.Errorf("sync ozon raw: no adapter")
	}
	ozon, ok := adapter.(*OzonAdapter)
	if !ok {
		return fmt.Errorf("sync ozon raw: type error")
	}
	since := time.Now().Add(-72 * time.Hour)
	for _, acct := range accounts {
		payload := map[string]interface{}{
			"dir": "ASC",
			"filter": map[string]string{
				"since": since.Format("2006-01-02T15:04:05.000Z"),
			},
			"limit": 100,
		}
		rawBody, err := ozon.FetchRaw(ctx, acct.PlatformID, "/v3/posting/fbs/list", payload)
		if err != nil {
			s.logger.Warn("ozon sync raw fetch error", zap.Int64("account", acct.ID), zap.Error(err))
			continue
		}
		var r struct {
			Result struct {
				Postings []map[string]interface{} `json:"postings"`
			} `json:"result"`
		}
		if err := json.Unmarshal(rawBody, &r); err != nil {
			s.logger.Warn("ozon sync raw unmarshal error", zap.Int64("account", acct.ID), zap.Error(err))
			continue
		}
		for _, posting := range r.Result.Postings {
			postingBytes, _ := json.Marshal(posting)
			rawEvent := RawEvent{
				PlatformCode:  "ozon",
				EventType:     "order",
				RawPayload:    postingBytes,
				MappingStatus: "pending",
			}
			if err := s.db.Create(&rawEvent).Error; err != nil {
				s.logger.Warn("ozon sync raw: failed to store raw event", zap.Error(err))
				continue
			}
			if pipeline != nil {
				go func(eid int64, rp []byte) {
					_, err := pipeline.ProcessRawEvent(context.Background(), eid, "ozon", "order", rp)
					if err != nil {
						s.logger.Error("ozon sync raw pipeline failed",
							zap.Int64("event_id", eid),
							zap.Error(err))
					}
				}(rawEvent.ID, postingBytes)
			}
		}
		s.logger.Info("ozon sync raw done",
			zap.Int64("account", acct.ID),
			zap.Int("postings", len(r.Result.Postings)),
		)
	}
	return nil
}
