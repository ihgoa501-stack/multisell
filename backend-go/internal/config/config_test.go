package config

import (
	"strings"
	"testing"
	"time"
)

func validConfig() *Config {
	return &Config{
		Server:      ServerConfig{Port: 8080, Mode: "debug", ReadHeaderTimeout: 5 * time.Second, ReadTimeout: time.Minute, IdleTimeout: 2 * time.Minute, ShutdownTimeout: 10 * time.Second, MaxHeaderBytes: 1 << 20, MaxRequestBodyBytes: 32 << 20},
		Database:    DatabaseConfig{Host: "localhost", Port: 5432, User: "postgres", DBName: "multisell", SSLMode: "disable", MaxIdleConns: 10, MaxOpenConns: 100, ConnMaxLifetime: 30 * time.Minute, ConnMaxIdleTime: 5 * time.Minute},
		CORS:        CORSConfig{AllowedOrigins: "*"},
		JWT:         JWTConfig{Secret: defaultDevelopmentJWTSecret, ExpiryHours: 24, RefreshExpiryHours: 168},
		Log:         LogConfig{Level: "debug", Format: "console"},
		SchemaDrift: SchemaDriftConfig{Enabled: true, OnDrift: "warn"},
	}
}

func TestValidate_AcceptsDevelopmentDefaults(t *testing.T) {
	if err := validConfig().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_ReleaseRequiresStrongSecretAndExplicitCORS(t *testing.T) {
	cfg := validConfig()
	cfg.Server.Mode = "release"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "JWT secret") {
		t.Fatalf("weak release secret error = %v", err)
	}
	cfg.JWT.Secret = strings.Repeat("s", 32)
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "CORS") {
		t.Fatalf("wildcard release CORS error = %v", err)
	}
	cfg.CORS.AllowedOrigins = "https://owner.example"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid release config rejected: %v", err)
	}
	cfg.JWT.RegistrationEnabled = true
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "registration") {
		t.Fatalf("release registration error = %v", err)
	}
	cfg.JWT.RegistrationEnabled = false
	cfg.Server.SwaggerEnabled = true
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "Swagger") {
		t.Fatalf("release Swagger error = %v", err)
	}
}

func TestValidate_RejectsInvalidOperationalBounds(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Config)
	}{
		{"unknown mode", func(c *Config) { c.Server.Mode = "production" }},
		{"invalid port", func(c *Config) { c.Server.Port = 0 }},
		{"invalid read timeout", func(c *Config) { c.Server.ReadTimeout = time.Second }},
		{"invalid header size", func(c *Config) { c.Server.MaxHeaderBytes = 1024 }},
		{"invalid body size", func(c *Config) { c.Server.MaxRequestBodyBytes = 1024 }},
		{"invalid pool", func(c *Config) { c.Database.MaxIdleConns = 101 }},
		{"invalid pool lifetime", func(c *Config) { c.Database.ConnMaxLifetime = 0 }},
		{"idle exceeds lifetime", func(c *Config) { c.Database.ConnMaxIdleTime = time.Hour }},
		{"invalid ssl mode", func(c *Config) { c.Database.SSLMode = "unsafe" }},
		{"empty jwt", func(c *Config) { c.JWT.Secret = "" }},
		{"invalid jwt lifetime", func(c *Config) { c.JWT.RefreshExpiryHours = c.JWT.ExpiryHours }},
		{"invalid log", func(c *Config) { c.Log.Level = "verbose" }},
		{"invalid drift policy", func(c *Config) { c.SchemaDrift.OnDrift = "ignore" }},
		{"negative budget", func(c *Config) { c.LLM.DailyBudgetUSD = -1 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.edit(cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("invalid config was accepted")
			}
		})
	}
}

func TestValidate_JWTKeyRotationConfig(t *testing.T) {
	cfg := validConfig()
	cfg.JWT.KeyID = "2026-07"
	cfg.JWT.PreviousKeysJSON = `{"2026-06":"old-secret"}`
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid rotation config rejected: %v", err)
	}
	cfg.JWT.PreviousKeysJSON = `{invalid`
	if err := cfg.Validate(); err == nil {
		t.Fatal("invalid previous key JSON was accepted")
	}
	cfg.JWT.PreviousKeysJSON = `{"2026-07":"duplicate-current"}`
	if err := cfg.Validate(); err == nil {
		t.Fatal("current key ID duplicated in previous keys")
	}
}
