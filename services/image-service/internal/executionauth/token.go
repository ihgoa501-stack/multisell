package executionauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"time"
)

const Audience = "lingmirror-image-service-execution"

type Claims struct {
	ApprovalExecutionID string `json:"approval_execution_id"`
	TaskID              string `json:"task_id"`
	TaskVersion         int64  `json:"task_version"`
	OwnerID             int64  `json:"owner_id"`
	JobID               string `json:"job_id"`
	ManifestHash        string `json:"manifest_hash"`
	Operation           string `json:"operation"`
	Processor           string `json:"processor"`
	MaxCost             string `json:"max_cost"`
	Currency            string `json:"currency"`
	Nonce               string `json:"nonce"`
	IssuedAt            int64  `json:"iat"`
	NotBefore           int64  `json:"nbf"`
	ExpiresAt           int64  `json:"exp"`
	Audience            string `json:"aud"`
}

func Verify(token string, key []byte, now time.Time) (Claims, error) {
	var claims Claims
	parts := strings.Split(token, ".")
	if len(parts) != 3 || len(key) < 32 {
		return claims, errors.New("invalid execution authorization")
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(sig, mac.Sum(nil)) {
		return claims, errors.New("invalid execution authorization")
	}
	header, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || string(header) != `{"alg":"HS256","typ":"JWT"}` {
		return claims, errors.New("invalid execution authorization")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || json.Unmarshal(payload, &claims) != nil {
		return claims, errors.New("invalid execution authorization")
	}
	n := now.Unix()
	cost, costOK := new(big.Rat).SetString(claims.MaxCost)
	if claims.Audience != Audience || claims.ApprovalExecutionID == "" || claims.OwnerID <= 0 || claims.TaskID == "" || claims.TaskVersion <= 0 || claims.JobID == "" || claims.ManifestHash == "" || claims.Operation == "" || claims.Processor == "" || !costOK || cost.Sign() <= 0 || !allowedCurrency(claims.Currency) || claims.Nonce == "" || claims.IssuedAt <= 0 || claims.NotBefore <= 0 || claims.ExpiresAt <= claims.NotBefore || claims.ExpiresAt-claims.IssuedAt > 300 || n < claims.NotBefore || n >= claims.ExpiresAt || claims.IssuedAt > n+30 {
		return Claims{}, errors.New("invalid execution authorization")
	}
	return claims, nil
}

func allowedCurrency(v string) bool {
	switch v {
	case "USD", "EUR", "CNY", "GBP", "JPY":
		return true
	}
	return false
}
