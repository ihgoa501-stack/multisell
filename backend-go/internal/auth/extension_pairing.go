package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const extensionCollectScope = "sourcing1688.collect"

type ExtensionPairing struct {
	ID              int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	NonceHash       string     `gorm:"column:nonce_hash;size:64;not null;uniqueIndex" json:"-"`
	UserID          int64      `gorm:"column:user_id;not null;index" json:"user_id"`
	Environment     string     `gorm:"column:environment;size:40;not null" json:"environment"`
	ExtensionID     string     `gorm:"column:extension_id;size:128" json:"extension_id,omitempty"`
	DeviceID        string     `gorm:"column:device_id;size:128" json:"device_id,omitempty"`
	BrowserLabel    string     `gorm:"column:browser_label;size:120" json:"browser_label,omitempty"`
	ClaimSecretHash string     `gorm:"column:claim_secret_hash;size:64" json:"-"`
	Status          string     `gorm:"column:status;size:24;not null" json:"status"`
	ExpiresAt       time.Time  `gorm:"column:expires_at;not null" json:"expires_at"`
	ConfirmedAt     *time.Time `gorm:"column:confirmed_at" json:"confirmed_at,omitempty"`
	ExchangedAt     *time.Time `gorm:"column:exchanged_at" json:"exchanged_at,omitempty"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (ExtensionPairing) TableName() string { return "extension_pairing" }

type ExtensionDevice struct {
	DeviceID       string     `gorm:"column:device_id;size:128;primaryKey" json:"device_id"`
	InstallationID string     `gorm:"column:installation_id;size:128;not null;uniqueIndex:ux_extension_device_install,priority:3" json:"-"`
	UserID         int64      `gorm:"column:user_id;not null;index;uniqueIndex:ux_extension_device_install,priority:1" json:"user_id"`
	ExtensionID    string     `gorm:"column:extension_id;size:128;not null" json:"extension_id"`
	Environment    string     `gorm:"column:environment;size:40;not null;uniqueIndex:ux_extension_device_install,priority:2" json:"environment"`
	BrowserLabel   string     `gorm:"column:browser_label;size:120;not null" json:"browser_label"`
	SecretHash     string     `gorm:"column:secret_hash;size:64;not null" json:"-"`
	Scope          string     `gorm:"column:scope;size:120;not null" json:"scope"`
	RevokedAt      *time.Time `gorm:"column:revoked_at" json:"revoked_at,omitempty"`
	LastUsedAt     *time.Time `gorm:"column:last_used_at" json:"last_used_at,omitempty"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ExtensionDevice) TableName() string { return "extension_device" }

type ExtensionTokenClaims struct {
	UserID      int64    `json:"user_id"`
	Type        string   `json:"type"`
	DeviceID    string   `json:"device_id"`
	Environment string   `json:"environment"`
	Scopes      []string `json:"scopes"`
	jwt.RegisteredClaims
}

type PairingCreated struct {
	PairingID   int64     `json:"pairing_id"`
	Nonce       string    `json:"nonce"`
	Environment string    `json:"environment"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type DeviceCredential struct {
	AccessToken  string    `json:"access_token"`
	DeviceID     string    `json:"device_id"`
	DeviceSecret string    `json:"device_secret"`
	Scope        string    `json:"scope"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func opaqueSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func secretHash(value string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(digest[:])
}

func validPairingEnvironment(value string) bool {
	return value == "development" || value == "acceptance" || value == "production"
}

func (s *Service) CreateExtensionPairing(userID int64, environment string) (*PairingCreated, error) {
	if userID <= 0 || !validPairingEnvironment(environment) || environment != s.cfg.Server.EffectiveDeploymentEnvironment() {
		return nil, errors.New("invalid extension pairing environment")
	}
	if err := s.requireExtensionOwner(userID); err != nil {
		return nil, err
	}
	nonce, err := opaqueSecret()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	expires := now.Add(10 * time.Minute)
	row := ExtensionPairing{NonceHash: secretHash(nonce), UserID: userID, Environment: environment, Status: "waiting_for_browser", ExpiresAt: expires}
	if err := s.db.Create(&row).Error; err != nil {
		return nil, err
	}
	return &PairingCreated{PairingID: row.ID, Nonce: nonce, Environment: environment, ExpiresAt: expires}, nil
}

func (s *Service) requireExtensionOwner(userID int64) error {
	var user User
	if err := s.db.First(&user, userID).Error; err != nil || user.Status != 1 || (user.Role != "owner" && user.Role != "admin") {
		return errors.New("only an active Owner can manage browser extension devices")
	}
	return nil
}

func (s *Service) ClaimExtensionPairing(nonce, claimSecret, deviceID, extensionID, environment, browserLabel string) error {
	if strings.TrimSpace(claimSecret) == "" || strings.TrimSpace(deviceID) == "" || strings.TrimSpace(extensionID) == "" || strings.TrimSpace(browserLabel) == "" {
		return errors.New("incomplete browser identity")
	}
	if environment != s.cfg.Server.EffectiveDeploymentEnvironment() {
		return errors.New("pairing targets a different deployment environment")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var row ExtensionPairing
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("nonce_hash = ?", secretHash(nonce)).First(&row).Error; err != nil {
			return errors.New("invalid or expired pairing")
		}
		if row.Environment != environment || row.Status != "waiting_for_browser" || !row.ExpiresAt.After(time.Now().UTC()) {
			return errors.New("invalid or expired pairing")
		}
		return tx.Model(&row).Updates(map[string]any{"claim_secret_hash": secretHash(claimSecret), "device_id": strings.TrimSpace(deviceID), "extension_id": strings.TrimSpace(extensionID), "browser_label": strings.TrimSpace(browserLabel), "status": "waiting_for_owner"}).Error
	})
}

func (s *Service) GetExtensionPairing(userID, pairingID int64) (*ExtensionPairing, error) {
	if err := s.requireExtensionOwner(userID); err != nil {
		return nil, err
	}
	var row ExtensionPairing
	err := s.db.Where("id = ? AND user_id = ?", pairingID, userID).First(&row).Error
	return &row, err
}

func (s *Service) ConfirmExtensionPairing(userID, pairingID int64) error {
	if err := s.requireExtensionOwner(userID); err != nil {
		return err
	}
	now := time.Now().UTC()
	result := s.db.Model(&ExtensionPairing{}).Where("id = ? AND user_id = ? AND status = ? AND expires_at > ?", pairingID, userID, "waiting_for_owner", now).
		Updates(map[string]any{"status": "confirmed", "confirmed_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("pairing is not ready for confirmation")
	}
	return nil
}

func (s *Service) signExtensionAccess(device *ExtensionDevice) (string, time.Time, error) {
	now := time.Now().UTC()
	// Security review selected a dedicated short TTL; device credentials handle
	// recovery and can be revoked independently.
	expires := now.Add(15 * time.Minute)
	claims := ExtensionTokenClaims{UserID: device.UserID, Type: "extension_access", DeviceID: device.DeviceID, Environment: device.Environment,
		Scopes: []string{extensionCollectScope}, RegisteredClaims: jwt.RegisteredClaims{ID: device.DeviceID, Issuer: "lingmirror-extension:" + device.Environment,
			Audience: jwt.ClaimStrings{"lingmirror-sourcing1688"}, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expires)}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = s.cfg.JWT.EffectiveKeyID()
	signed, err := token.SignedString([]byte(s.cfg.JWT.Secret))
	return signed, expires, err
}

func (s *Service) ExchangeExtensionPairing(nonce, claimSecret, extensionID string) (*DeviceCredential, error) {
	var credential DeviceCredential
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var row ExtensionPairing
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("nonce_hash = ?", secretHash(nonce)).First(&row).Error; err != nil {
			return errors.New("invalid pairing")
		}
		if row.Status != "confirmed" || row.ClaimSecretHash != secretHash(claimSecret) || !row.ExpiresAt.After(time.Now().UTC()) {
			return errors.New("pairing not confirmed or expired")
		}
		if row.ExtensionID != strings.TrimSpace(extensionID) {
			return errors.New("extension origin does not match pairing")
		}
		if row.Environment != s.cfg.Server.EffectiveDeploymentEnvironment() {
			return errors.New("pairing belongs to a different deployment environment")
		}
		var owner User
		if err := tx.First(&owner, row.UserID).Error; err != nil || owner.Status != 1 || (owner.Role != "owner" && owner.Role != "admin") {
			return errors.New("Owner is no longer allowed to pair extensions")
		}
		deviceSecret, err := opaqueSecret()
		if err != nil {
			return err
		}
		var device ExtensionDevice
		err = tx.Where("user_id = ? AND environment = ? AND installation_id = ?", row.UserID, row.Environment, row.DeviceID).First(&device).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			serverDeviceID, generateErr := opaqueSecret()
			if generateErr != nil {
				return generateErr
			}
			device = ExtensionDevice{DeviceID: serverDeviceID, InstallationID: row.DeviceID, UserID: row.UserID, ExtensionID: row.ExtensionID, Environment: row.Environment,
				BrowserLabel: row.BrowserLabel, SecretHash: secretHash(deviceSecret), Scope: extensionCollectScope}
			if err := tx.Create(&device).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else if err := tx.Model(&device).Updates(map[string]any{"extension_id": row.ExtensionID, "browser_label": row.BrowserLabel, "secret_hash": secretHash(deviceSecret), "revoked_at": nil}).Error; err != nil {
			return err
		}
		token, expires, err := s.signExtensionAccess(&device)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := tx.Model(&row).Updates(map[string]any{"status": "exchanged", "exchanged_at": now}).Error; err != nil {
			return err
		}
		credential = DeviceCredential{AccessToken: token, DeviceID: device.DeviceID, DeviceSecret: deviceSecret, Scope: extensionCollectScope, ExpiresAt: expires}
		return nil
	})
	return &credential, err
}

func (s *Service) RefreshExtensionDevice(deviceID, deviceSecret, environment, extensionID string) (*DeviceCredential, error) {
	if environment != s.cfg.Server.EffectiveDeploymentEnvironment() {
		return nil, errors.New("device belongs to a different deployment environment")
	}
	var device ExtensionDevice
	if err := s.db.Where("device_id = ? AND environment = ? AND revoked_at IS NULL", strings.TrimSpace(deviceID), environment).First(&device).Error; err != nil {
		return nil, errors.New("device credential is invalid or revoked")
	}
	if device.SecretHash != secretHash(deviceSecret) {
		return nil, errors.New("device credential is invalid or revoked")
	}
	if device.ExtensionID != strings.TrimSpace(extensionID) {
		return nil, errors.New("extension origin does not match device")
	}
	var user User
	if err := s.db.First(&user, device.UserID).Error; err != nil || user.Status != 1 || (user.Role != "owner" && user.Role != "admin") {
		return nil, errors.New("Owner account is unavailable")
	}
	token, expires, err := s.signExtensionAccess(&device)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	_ = s.db.Model(&device).Update("last_used_at", now).Error
	return &DeviceCredential{AccessToken: token, DeviceID: device.DeviceID, Scope: extensionCollectScope, ExpiresAt: expires}, nil
}

func (s *Service) RevokeExtensionDevice(userID int64, deviceID string) error {
	if err := s.requireExtensionOwner(userID); err != nil {
		return err
	}
	now := time.Now().UTC()
	result := s.db.Model(&ExtensionDevice{}).Where("device_id = ? AND user_id = ? AND revoked_at IS NULL", deviceID, userID).Update("revoked_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("device not found or already revoked")
	}
	return nil
}

// RevokeAllExtensionDevices makes web logout an immediate stop signal for all
// paired collection browsers. Extension access JWTs are device-checked on
// every request, so revocation takes effect without waiting for token expiry.
func (s *Service) RevokeAllExtensionDevices(userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user identity")
	}
	now := time.Now().UTC()
	return s.db.Model(&ExtensionDevice{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

func (s *Service) ListExtensionDevices(userID int64) ([]ExtensionDevice, error) {
	if err := s.requireExtensionOwner(userID); err != nil {
		return nil, err
	}
	var devices []ExtensionDevice
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&devices).Error
	return devices, err
}
