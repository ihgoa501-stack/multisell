package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lingmirror/backend-go/internal/config"
	"github.com/lingmirror/backend-go/internal/rbac"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Claims is the JWT payload for access and refresh tokens.
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Email    string `json:"email,omitempty"`
	Type     string `json:"type"` // access | refresh
	jwt.RegisteredClaims
}

// Service provides authentication business logic.
type Service struct {
	db     *gorm.DB
	cfg    *config.Config
	logger *zap.Logger
}

// NewService creates a new auth service.
func NewService(db *gorm.DB, cfg *config.Config, logger *zap.Logger) *Service {
	return &Service{db: db, cfg: cfg, logger: logger}
}

// HashPassword hashes a plain password using bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares a plain password against a bcrypt hash.
func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateAccessToken creates a short-lived access JWT.
func (s *Service) GenerateAccessToken(u *User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   u.ID,
		Username: u.Username,
		Role:     u.Role,
		Email:    u.Email,
		Type:     "access",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.cfg.JWT.ExpiryHours) * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = s.cfg.JWT.EffectiveKeyID()
	return token.SignedString([]byte(s.cfg.JWT.Secret))
}

// GenerateRefreshToken creates a long-lived refresh JWT.
func (s *Service) GenerateRefreshToken(u *User) (string, error) {
	now := time.Now()
	tokenID, err := newTokenID()
	if err != nil {
		return "", err
	}
	token, expiresAt, err := s.signRefreshToken(u, tokenID, now)
	if err != nil {
		return "", err
	}
	session := &RefreshSession{TokenID: tokenID, FamilyID: tokenID, UserID: u.ID, ExpiresAt: expiresAt}
	if err := s.db.Create(session).Error; err != nil {
		return "", fmt.Errorf("create refresh session: %w", err)
	}
	return token, nil
}

func newTokenID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Service) signRefreshToken(u *User, tokenID string, now time.Time) (string, time.Time, error) {
	expiresAt := now.Add(time.Duration(s.cfg.JWT.RefreshExpiryHours) * time.Hour)
	claims := Claims{
		UserID:   u.ID,
		Username: u.Username,
		Type:     "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = s.cfg.JWT.EffectiveKeyID()
	signed, err := token.SignedString([]byte(s.cfg.JWT.Secret))
	return signed, expiresAt, err
}

// ParseToken validates a JWT and returns its claims.
func (s *Service) ParseToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, jwt.ErrSignatureInvalid
		}
		keyID, _ := t.Header["kid"].(string)
		if keyID == "" {
			keys := []jwt.VerificationKey{[]byte(s.cfg.JWT.Secret)}
			if previous, parseErr := s.cfg.JWT.PreviousKeys(); parseErr == nil {
				for _, secret := range previous {
					keys = append(keys, []byte(secret))
				}
			}
			return jwt.VerificationKeySet{Keys: keys}, nil
		}
		if keyID == s.cfg.JWT.EffectiveKeyID() {
			return []byte(s.cfg.JWT.Secret), nil
		}
		previous, parseErr := s.cfg.JWT.PreviousKeys()
		if parseErr != nil {
			return nil, jwt.ErrSignatureInvalid
		}
		secret, ok := previous[keyID]
		if !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// Authenticate validates username+password and returns the user on success.
func (s *Service) Authenticate(username, password string) (*User, error) {
	var user User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户名或密码错误")
		}
		return nil, err
	}
	if !CheckPassword(password, user.PasswordHash) {
		return nil, errors.New("用户名或密码错误")
	}
	if user.Status != 1 {
		return nil, errors.New("账号已被禁用")
	}
	now := time.Now()
	user.LastLoginAt = &now
	s.db.Model(&user).Update("last_login_at", now)
	return &user, nil
}

// GetUserByID fetches a user by primary key.
func (s *Service) GetUserByID(id int64) (*User, error) {
	var user User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Register creates a new user account. Returns an error if the username is
// already taken. Default role is "user"; "admin" can only be assigned by an
// existing admin (enforced at handler layer for now).
func (s *Service) Register(username, password, displayName, email, role string) (*User, error) {
	if len(password) < 8 {
		return nil, errors.New("密码至少 8 位")
	}
	// Check for existing username.
	var existing User
	if err := s.db.Where("username = ?", username).First(&existing).Error; err == nil {
		return nil, errors.New("用户名已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	if role == "" {
		role = "user"
	}
	if role != "user" && role != "admin" && role != "operator" {
		role = "user"
	}
	user := User{
		Username:     username,
		PasswordHash: hash,
		DisplayName:  displayName,
		Email:        email,
		Role:         role,
		Status:       1,
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		return assignDefaultRBACRole(tx, user.ID, role)
	}); err != nil {
		return nil, err
	}
	return &user, nil
}

func assignDefaultRBACRole(tx *gorm.DB, userID int64, legacyRole string) error {
	if !tx.Migrator().HasTable(&rbac.Role{}) || !tx.Migrator().HasTable(&rbac.UserRole{}) {
		return nil
	}

	roleCode := ""
	switch legacyRole {
	case "admin":
		roleCode = "admin"
	case "operator":
		roleCode = "ops"
	default:
		return nil
	}

	var role rbac.Role
	if err := tx.Where("code = ? AND status = ?", roleCode, 1).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	return tx.Where(rbac.UserRole{UserID: userID, RoleID: role.ID}).
		FirstOrCreate(&rbac.UserRole{UserID: userID, RoleID: role.ID}).Error
}

// Login authenticates a user and returns access+refresh tokens plus the user VO.
func (s *Service) Login(username, password string) (accessToken, refreshToken string, user *UserVO, err error) {
	u, err := s.Authenticate(username, password)
	if err != nil {
		return "", "", nil, err
	}
	accessToken, err = s.GenerateAccessToken(u)
	if err != nil {
		return "", "", nil, err
	}
	refreshToken, err = s.GenerateRefreshToken(u)
	if err != nil {
		return "", "", nil, err
	}
	return accessToken, refreshToken, u.ToVO(), nil
}

// Refresh validates a refresh token and mints new access+refresh tokens.
func (s *Service) Refresh(refreshToken string) (string, string, *UserVO, error) {
	claims, err := s.ParseToken(refreshToken)
	if err != nil {
		return "", "", nil, errors.New("invalid refresh token")
	}
	if claims.Type != "refresh" {
		return "", "", nil, errors.New("invalid refresh token")
	}
	if claims.ID == "" {
		return "", "", nil, errors.New("invalid refresh token")
	}
	u, err := s.GetUserByID(claims.UserID)
	if err != nil {
		return "", "", nil, errors.New("user not found")
	}
	if u.Status != 1 {
		return "", "", nil, errors.New("账号已被禁用")
	}
	access, err := s.GenerateAccessToken(u)
	if err != nil {
		return "", "", nil, err
	}
	var refresh string
	replayDetected := false
	err = s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		var current RefreshSession
		if err := tx.Where("token_id = ? AND user_id = ?", claims.ID, claims.UserID).First(&current).Error; err != nil {
			return errors.New("invalid refresh token")
		}
		if current.RevokedAt != nil || !current.ExpiresAt.After(now) {
			// Reuse of a rotated token invalidates the whole family so a stolen
			// predecessor cannot coexist with the legitimate successor.
			if current.FamilyID != "" {
				if err := tx.Model(&RefreshSession{}).Where("family_id = ? AND revoked_at IS NULL", current.FamilyID).Update("revoked_at", now).Error; err != nil {
					return err
				}
			}
			replayDetected = true
			return nil
		}
		newID, err := newTokenID()
		if err != nil {
			return err
		}
		updated := tx.Model(&RefreshSession{}).
			Where("token_id = ? AND revoked_at IS NULL AND expires_at > ?", current.TokenID, now).
			Updates(map[string]any{"revoked_at": now, "replaced_by": newID})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			if current.FamilyID != "" {
				if err := tx.Model(&RefreshSession{}).Where("family_id = ? AND revoked_at IS NULL", current.FamilyID).Update("revoked_at", now).Error; err != nil {
					return err
				}
			}
			replayDetected = true
			return nil
		}
		var expiresAt time.Time
		refresh, expiresAt, err = s.signRefreshToken(u, newID, now)
		if err != nil {
			return err
		}
		return tx.Create(&RefreshSession{TokenID: newID, FamilyID: current.FamilyID, UserID: u.ID, ExpiresAt: expiresAt}).Error
	})
	if err != nil {
		return "", "", nil, err
	}
	if replayDetected {
		return "", "", nil, errors.New("invalid refresh token")
	}
	return access, refresh, u.ToVO(), nil
}

// RevokeRefreshFamily invalidates the current device/session family. Repeated
// revocation is idempotent, but the token must be authentic and belong to the
// authenticated user.
func (s *Service) RevokeRefreshFamily(refreshToken string, userID int64) error {
	claims, err := s.ParseToken(refreshToken)
	if err != nil || claims.Type != "refresh" || claims.ID == "" || claims.UserID != userID || userID <= 0 {
		return errors.New("invalid refresh token")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var session RefreshSession
		if err := tx.Where("token_id = ? AND user_id = ?", claims.ID, userID).First(&session).Error; err != nil {
			return errors.New("invalid refresh token")
		}
		now := time.Now()
		return tx.Model(&RefreshSession{}).
			Where("family_id = ? AND user_id = ? AND revoked_at IS NULL", session.FamilyID, userID).
			Update("revoked_at", now).Error
	})
}

// RevokeAllRefreshSessions invalidates every active refresh session for one
// user. Access JWTs remain valid until their short expiry.
func (s *Service) RevokeAllRefreshSessions(userID int64) error {
	if userID <= 0 {
		return errors.New("invalid user identity")
	}
	now := time.Now()
	return s.db.Model(&RefreshSession{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}
