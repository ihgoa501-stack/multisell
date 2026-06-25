package integrations

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides integrations business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new integrations service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
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
	return items, total, nil
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
		"sync_status": "syncing",
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
