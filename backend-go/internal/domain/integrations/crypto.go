package integrations

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
)

// encryptionKey returns the AES-GCM key used for token encryption.
// Configured via PLATFORM_TOKEN_ENCRYPTION_KEY env var (32 bytes, base64-encoded).
// Falls back to a dev-only key when the env var is unset (NOT production-safe).
func encryptionKey() ([]byte, error) {
	keyStr := os.Getenv("PLATFORM_TOKEN_ENCRYPTION_KEY")
	if keyStr == "" {
		// Dev-only fallback: NOT safe for production.
		return []byte("0123456789abcdef0123456789abcdef"), nil
	}
	return base64.StdEncoding.DecodeString(keyStr)
}

// encrypt encrypts plaintext using AES-256-GCM and returns base64-encoded ciphertext.
func encrypt(plaintext string) (string, error) {
	key, err := encryptionKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts a base64-encoded AES-256-GCM ciphertext.
func decrypt(cipherB64 string) (string, error) {
	key, err := encryptionKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(cipherB64)
	if err != nil {
		return "", err
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
