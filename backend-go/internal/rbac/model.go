package rbac

import "time"

// Role maps to the `role` table.
type Role struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"column:name;size:100;not null" json:"name"`
	Code        string    `gorm:"column:code;size:100;not null;uniqueIndex" json:"code"`
	Description string    `gorm:"column:description;size:500" json:"description"`
	Status      int       `gorm:"column:status;smallint;default:1" json:"status"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName overrides the default table name.
func (Role) TableName() string {
	return "role"
}

// Permission maps to the `permission` table.
type Permission struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"column:name;size:100;not null" json:"name"`
	Code        string    `gorm:"column:code;size:100;not null;uniqueIndex" json:"code"`
	Description string    `gorm:"column:description;size:500" json:"description"`
	Module      string    `gorm:"column:module;size:100" json:"module"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName overrides the default table name.
func (Permission) TableName() string {
	return "permission"
}

// UserRole maps to the `user_role` join table.
type UserRole struct {
	ID     int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID int64 `gorm:"column:user_id;not null;index" json:"user_id"`
	RoleID int64 `gorm:"column:role_id;not null;index" json:"role_id"`
}

// TableName overrides the default table name.
func (UserRole) TableName() string {
	return "user_role"
}

// RolePermission maps to the `role_permission` join table.
type RolePermission struct {
	ID           int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RoleID       int64 `gorm:"column:role_id;not null;index" json:"role_id"`
	PermissionID int64 `gorm:"column:permission_id;not null;index" json:"permission_id"`
}

// TableName overrides the default table name.
func (RolePermission) TableName() string {
	return "role_permission"
}
