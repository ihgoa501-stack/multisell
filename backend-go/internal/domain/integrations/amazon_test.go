package integrations

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAmazonConfigParsing(t *testing.T) {
	raw := json.RawMessage(`{
		"client_id": "amzn-app-123",
		"client_secret": "amzn-secret-456",
		"aws_access_key_id": "AKIAIOSFODNN7EXAMPLE",
		"aws_secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"aws_region": "us-east-1",
		"marketplace_id": "ATVPDKIKX0DER"
	}`)

	var cfg amazonConfig
	err := json.Unmarshal(raw, &cfg)
	assert.NoError(t, err)
	assert.Equal(t, "amzn-app-123", cfg.ClientID)
	assert.Equal(t, "amzn-secret-456", cfg.ClientSecret)
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", cfg.AWSAccessKey)
	assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", cfg.AWSSecretKey)
	assert.Equal(t, "us-east-1", cfg.AWSRegion)
}

func TestAmazonLWAEndpoint(t *testing.T) {
	assert.Equal(t, "https://api.amazon.com/auth/o2/token", amazonLWAEndpoint)
}

func TestAmazonProdURLFormat(t *testing.T) {
	url := fmt.Sprintf(amazonProdURLTemplate, "eu-west-1")
	assert.Equal(t, "https://sellingpartnerapi-eu-west-1.amazon.com", url)
}

func TestAmazonSandboxURL(t *testing.T) {
	assert.Equal(t, "https://sellingpartnerapi-sandbox.amazon.com", amazonSandboxURL)
}

func TestSha256Hex(t *testing.T) {
	// SHA256 of empty
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		sha256Hex(nil))
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		sha256Hex([]byte{}))

	// SHA256 of known string
	assert.Equal(t, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		sha256Hex([]byte("hello")))
}

func TestAmazonConfigFromAccount(t *testing.T) {
	now := time.Now()
	acct := PlatformIntegrationAccount{
		ID:             1,
		PlatformID:     5,
		StoreName:      "My Amazon Store",
		AccountID:      "seller-123",
		RefreshToken:   "Atzr|IwEBINh...",
		TokenExpiresAt: &now,
		Config: json.RawMessage(`{
			"client_id": "amzn1.application-oa2-client.abc123",
			"client_secret": "amzn-secret",
			"aws_access_key_id": "AKIAIOSFODNN7EXAMPLE",
			"aws_secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"aws_region": "eu-west-1",
			"marketplace_id": "A1F83G8C2ARO7P"
		}`),
	}

	var cfg amazonConfig
	err := json.Unmarshal(acct.Config, &cfg)
	assert.NoError(t, err)
	assert.Equal(t, "amzn1.application-oa2-client.abc123", cfg.ClientID)
	assert.Equal(t, "eu-west-1", cfg.AWSRegion)
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", cfg.AWSAccessKey)

	// Verify account metadata is accessible alongside config
	assert.Equal(t, "My Amazon Store", acct.StoreName)
	assert.Equal(t, "Atzr|IwEBINh...", acct.RefreshToken)
}
