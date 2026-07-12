package rbac

import (
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides RBAC business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewService creates a new RBAC service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{db: db, logger: logger}
}

// ===================== Roles =====================

// ListRoles returns roles with optional status filter and pagination.
func (s *Service) ListRoles(status int, page, size int) ([]Role, int64, error) {
	var roles []Role
	q := s.db.Model(&Role{})
	if status >= 0 {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)

	offset := (page - 1) * size
	if err := q.Order("id desc").Offset(offset).Limit(size).Find(&roles).Error; err != nil {
		return nil, 0, err
	}
	return roles, total, nil
}

// GetRole fetches a single role by id.
func (s *Service) GetRole(id int64) (*Role, error) {
	var r Role
	if err := s.db.First(&r, id).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

// CreateRole inserts a new role.
func (s *Service) CreateRole(r *Role) error {
	return s.db.Create(r).Error
}

// UpdateRole updates an existing role.
func (s *Service) UpdateRole(r *Role) error {
	return s.db.Save(r).Error
}

// DeleteRole removes a role and its permission associations.
func (s *Service) DeleteRole(id int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", id).Delete(&RolePermission{}).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", id).Delete(&UserRole{}).Error; err != nil {
			return err
		}
		return tx.Delete(&Role{}, id).Error
	})
}

// ===================== Permissions =====================

// ListPermissions returns permissions, optionally filtered by module, with pagination.
func (s *Service) ListPermissions(module string, page, size int) ([]Permission, int64, error) {
	var perms []Permission
	q := s.db.Model(&Permission{})
	if module != "" {
		q = q.Where("module = ?", module)
	}
	var total int64
	q.Count(&total)

	offset := (page - 1) * size
	if err := q.Order("id desc").Offset(offset).Limit(size).Find(&perms).Error; err != nil {
		return nil, 0, err
	}
	return perms, total, nil
}

// GetPermission fetches a single permission by id.
func (s *Service) GetPermission(id int64) (*Permission, error) {
	var p Permission
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// CreatePermission inserts a new permission.
func (s *Service) CreatePermission(p *Permission) error {
	return s.db.Create(p).Error
}

// UpdatePermission updates an existing permission.
func (s *Service) UpdatePermission(p *Permission) error {
	return s.db.Save(p).Error
}

// DeletePermission removes a permission and its role associations.
func (s *Service) DeletePermission(id int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("permission_id = ?", id).Delete(&RolePermission{}).Error; err != nil {
			return err
		}
		return tx.Delete(&Permission{}, id).Error
	})
}

// ===================== User-Role =====================

// GetUserRoles returns the roles assigned to a user.
func (s *Service) GetUserRoles(userID int64) ([]Role, error) {
	var roles []Role
	err := s.db.
		Joins("JOIN user_role ON user_role.role_id = role.id").
		Where("user_role.user_id = ?", userID).
		Order("role.id desc").
		Find(&roles).Error
	return roles, err
}

// AssignUserRoles replaces the set of roles for a user (full sync).
func (s *Service) AssignUserRoles(userID int64, roleIDs []int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&UserRole{}).Error; err != nil {
			return err
		}
		if len(roleIDs) == 0 {
			return nil
		}
		rows := make([]UserRole, 0, len(roleIDs))
		for _, rid := range roleIDs {
			rows = append(rows, UserRole{UserID: userID, RoleID: rid})
		}
		return tx.Create(&rows).Error
	})
}

// ===================== Role-Permission =====================

// GetRolePermissions returns the permissions attached to a role.
func (s *Service) GetRolePermissions(roleID int64) ([]Permission, error) {
	var perms []Permission
	err := s.db.
		Joins("JOIN role_permission ON role_permission.permission_id = permission.id").
		Where("role_permission.role_id = ?", roleID).
		Order("permission.id desc").
		Find(&perms).Error
	return perms, err
}

// AssignRolePermissions replaces the set of permissions for a role (full sync).
func (s *Service) AssignRolePermissions(roleID int64, permissionIDs []int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&RolePermission{}).Error; err != nil {
			return err
		}
		if len(permissionIDs) == 0 {
			return nil
		}
		rows := make([]RolePermission, 0, len(permissionIDs))
		for _, pid := range permissionIDs {
			rows = append(rows, RolePermission{RoleID: roleID, PermissionID: pid})
		}
		return tx.Create(&rows).Error
	})
}

// GetUserPermissions returns the aggregated permission codes for a user.
func (s *Service) GetUserPermissions(userID int64) ([]string, error) {
	var codes []string
	err := s.db.
		Model(&Permission{}).
		Distinct("permission.code").
		Joins("JOIN role_permission ON role_permission.permission_id = permission.id").
		Joins("JOIN user_role ON user_role.role_id = role_permission.role_id").
		Joins("JOIN role ON role.id = user_role.role_id AND role.status = ?", 1).
		Where("user_role.user_id = ?", userID).
		Pluck("permission.code", &codes).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return codes, nil
}
