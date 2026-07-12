package auth

import (
	"time"
)

// User maps to the `user` table.
type User struct {
	ID           int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Username     string     `gorm:"column:username;size:100;not null;uniqueIndex" json:"username"`
	PasswordHash string     `gorm:"column:password_hash;size:500;not null" json:"-"`
	DisplayName  string     `gorm:"column:display_name;size:200" json:"display_name"`
	Role         string     `gorm:"column:role;size:50;default:user" json:"role"`
	Email        string     `gorm:"column:email;size:200" json:"email"`
	Status       int        `gorm:"column:status;smallint;default:1" json:"status"`
	LastLoginAt  *time.Time `gorm:"column:last_login_at" json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// RefreshSession stores only a random token identifier, never the JWT itself.
// Rotation revokes the current row and creates a successor in the same family.
type RefreshSession struct {
	TokenID    string     `gorm:"column:token_id;primaryKey;size:64" json:"-"`
	FamilyID   string     `gorm:"column:family_id;size:64;not null;index" json:"-"`
	UserID     int64      `gorm:"column:user_id;not null;index" json:"-"`
	ExpiresAt  time.Time  `gorm:"column:expires_at;not null;index" json:"-"`
	RevokedAt  *time.Time `gorm:"column:revoked_at;index" json:"-"`
	ReplacedBy string     `gorm:"column:replaced_by;size:64;not null;default:''" json:"-"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime" json:"-"`
}

func (RefreshSession) TableName() string { return "auth_refresh_session" }

// TableName overrides the default table name.
func (User) TableName() string {
	return "user"
}

// UserVO is the user view object returned to clients (no password hash).
type UserVO struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	Role        string     `json:"role"`
	Email       string     `json:"email"`
	Status      int        `json:"status"`
	Permissions []string   `json:"permissions"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ToVO converts a User to a UserVO.
func (u *User) ToVO() *UserVO {
	return &UserVO{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Role:        u.Role,
		Email:       u.Email,
		Status:      u.Status,
		Permissions: []string{},
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
	}
}
