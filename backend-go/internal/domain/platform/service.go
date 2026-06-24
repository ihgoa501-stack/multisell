package platform

import (
	"errors"

	"github.com/lingmirror/backend-go/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides platform & store business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new platform service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// ---------- Platform CRUD ----------

// ListPlatforms returns paginated platforms with optional search.
func (s *Service) ListPlatforms(c *common.Pagination, search string) ([]Platform, int64, error) {
	var items []Platform
	var total int64

	q := s.db.Model(&Platform{})
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("name ILIKE ? OR code ILIKE ?", like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("sort_order ASC, id DESC").
		Offset(c.Offset()).Limit(c.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetPlatform returns a single platform by id.
func (s *Service) GetPlatform(id int64) (*Platform, error) {
	var p Platform
	if err := s.db.First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &p, nil
}

// CreatePlatform inserts a new platform.
func (s *Service) CreatePlatform(in *CreatePlatformInput) (*Platform, error) {
	p := Platform{
		Name:        in.Name,
		Code:        in.Code,
		APIBaseURL:  in.APIBaseURL,
		APIKey:      in.APIKey,
		ClientID:    in.ClientID,
		ExtraConfig: in.ExtraConfig,
	}
	if in.Status != nil {
		p.Status = *in.Status
	} else {
		p.Status = 1
	}
	if in.SortOrder != nil {
		p.SortOrder = *in.SortOrder
	}
	if err := s.db.Create(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdatePlatform patches a platform by id.
func (s *Service) UpdatePlatform(id int64, in *UpdatePlatformInput) (*Platform, error) {
	var p Platform
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.Name != nil {
		updates["name"] = *in.Name
	}
	if in.Code != nil {
		updates["code"] = *in.Code
	}
	if in.APIBaseURL != nil {
		updates["api_base_url"] = *in.APIBaseURL
	}
	if in.APIKey != nil {
		updates["api_key"] = *in.APIKey
	}
	if in.ClientID != nil {
		updates["client_id"] = *in.ClientID
	}
	if in.ExtraConfig != nil {
		updates["extra_config"] = *in.ExtraConfig
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if in.SortOrder != nil {
		updates["sort_order"] = *in.SortOrder
	}
	if len(updates) == 0 {
		return &p, nil
	}
	if err := s.db.Model(&p).Updates(updates).Error; err != nil {
		return nil, err
	}
	// reload
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// DeletePlatform removes a platform by id.
func (s *Service) DeletePlatform(id int64) error {
	res := s.db.Delete(&Platform{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ---------- Store CRUD ----------

// ListStores returns paginated stores with optional platform filter.
func (s *Service) ListStores(c *common.Pagination, platformID *int64, search string) ([]Store, int64, error) {
	var items []Store
	var total int64

	q := s.db.Model(&Store{})
	if platformID != nil {
		q = q.Where("platform_id = ?", *platformID)
	}
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("name ILIKE ?", like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id DESC").Offset(c.Offset()).Limit(c.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetStore returns a single store by id.
func (s *Service) GetStore(id int64) (*Store, error) {
	var st Store
	if err := s.db.First(&st, id).Error; err != nil {
		return nil, err
	}
	return &st, nil
}

// CreateStore inserts a new store.
func (s *Service) CreateStore(in *CreateStoreInput) (*Store, error) {
	st := Store{
		UserID:            in.UserID,
		Name:              in.Name,
		PlatformID:        in.PlatformID,
		PlatformAccountID: in.PlatformAccountID,
	}
	if in.Status != nil {
		st.Status = *in.Status
	} else {
		st.Status = 1
	}
	if err := s.db.Create(&st).Error; err != nil {
		return nil, err
	}
	return &st, nil
}

// UpdateStore patches a store by id.
func (s *Service) UpdateStore(id int64, in *UpdateStoreInput) (*Store, error) {
	var st Store
	if err := s.db.First(&st, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.UserID != nil {
		updates["user_id"] = *in.UserID
	}
	if in.Name != nil {
		updates["name"] = *in.Name
	}
	if in.PlatformID != nil {
		updates["platform_id"] = *in.PlatformID
	}
	if in.PlatformAccountID != nil {
		updates["platform_account_id"] = *in.PlatformAccountID
	}
	if in.Status != nil {
		updates["status"] = *in.Status
	}
	if len(updates) == 0 {
		return &st, nil
	}
	if err := s.db.Model(&st).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&st, id).Error; err != nil {
		return nil, err
	}
	return &st, nil
}

// DeleteStore removes a store by id.
func (s *Service) DeleteStore(id int64) error {
	res := s.db.Delete(&Store{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
