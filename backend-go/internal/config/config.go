package config

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const defaultDevelopmentJWTSecret = "dev-secret-change-in-production"

// Config holds all application configuration.
type Config struct {
	Server        ServerConfig      `mapstructure:"server"`
	Database      DatabaseConfig    `mapstructure:"database"`
	Redis         RedisConfig       `mapstructure:"redis"`
	CORS          CORSConfig        `mapstructure:"cors"`
	Metrics       MetricsConfig     `mapstructure:"metrics"`
	JWT           JWTConfig         `mapstructure:"jwt"`
	Log           LogConfig         `mapstructure:"log"`
	Sentry        SentryConfig      `mapstructure:"sentry"`
	LLM           LLMConfig         `mapstructure:"llm"`
	SchemaDrift   SchemaDriftConfig `mapstructure:"schemadrift"`
	EncryptionKey string            `mapstructure:"encryption_key"`
}

// LLMConfig holds LLM-related settings.
type LLMConfig struct {
	DailyBudgetUSD float64 `mapstructure:"daily_budget_usd"`
	Provider       string  `mapstructure:"provider"`
	APIKey         string  `mapstructure:"api_key"`
}

// ServerConfig holds server-specific settings.
type ServerConfig struct {
	Port                  int           `mapstructure:"port"`
	Mode                  string        `mapstructure:"mode"`
	DeploymentEnvironment string        `mapstructure:"deployment_environment"`
	Version               string        `mapstructure:"version"`
	ReadHeaderTimeout     time.Duration `mapstructure:"read_header_timeout"`
	ReadTimeout           time.Duration `mapstructure:"read_timeout"`
	WriteTimeout          time.Duration `mapstructure:"write_timeout"`
	IdleTimeout           time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout       time.Duration `mapstructure:"shutdown_timeout"`
	MaxHeaderBytes        int           `mapstructure:"max_header_bytes"`
	MaxRequestBodyBytes   int64         `mapstructure:"max_request_body_bytes"`
	SwaggerEnabled        bool          `mapstructure:"swagger_enabled"`
}

func (s ServerConfig) EffectiveDeploymentEnvironment() string {
	if s.DeploymentEnvironment != "" {
		return s.DeploymentEnvironment
	}
	if s.Mode == "release" {
		return "production"
	}
	return "development"
}

// SchemaDriftConfig holds schema drift detection settings.
type SchemaDriftConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	OnDrift string `mapstructure:"on_drift"` // "warn" | "panic" | "log_only"
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// JWTConfig holds JWT authentication settings.
type JWTConfig struct {
	Secret              string `mapstructure:"secret"`
	KeyID               string `mapstructure:"key_id"`
	PreviousKeysJSON    string `mapstructure:"previous_keys_json"`
	ExpiryHours         int    `mapstructure:"expiry_hours"`
	RefreshExpiryHours  int    `mapstructure:"refresh_expiry_hours"`
	RegistrationEnabled bool   `mapstructure:"registration_enabled"`
}

func (c JWTConfig) EffectiveKeyID() string {
	if strings.TrimSpace(c.KeyID) == "" {
		return "current"
	}
	return strings.TrimSpace(c.KeyID)
}

func (c JWTConfig) PreviousKeys() (map[string]string, error) {
	keys := map[string]string{}
	if strings.TrimSpace(c.PreviousKeysJSON) == "" {
		return keys, nil
	}
	if err := json.Unmarshal([]byte(c.PreviousKeysJSON), &keys); err != nil {
		return nil, fmt.Errorf("jwt.previous_keys_json must be a JSON object: %w", err)
	}
	return keys, nil
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// CORSConfig holds Cross-Origin Resource Sharing settings.
type CORSConfig struct {
	AllowedOrigins string `mapstructure:"allowed_origins"` // 逗号分隔，空值或"*"表示允许所有来源
}

// MetricsConfig holds Prometheus metrics settings.
type MetricsConfig struct {
	Enabled bool `mapstructure:"enabled"` // 是否开启 metrics 端点
}

// SentryConfig holds Sentry error tracking settings.
type SentryConfig struct {
	DSN string `mapstructure:"dsn"`
}

// DSN returns the PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

// Load reads configuration from file and environment variables.
func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")

	// Environment variable overrides
	v.SetEnvPrefix("")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Bind specific env vars
	v.BindEnv("database.host", "DB_HOST")
	v.BindEnv("database.port", "DB_PORT")
	v.BindEnv("database.user", "DB_USER")
	v.BindEnv("database.password", "DB_PASSWORD")
	v.BindEnv("database.dbname", "DB_NAME")
	v.BindEnv("database.sslmode", "DB_SSLMODE")
	v.BindEnv("database.max_idle_conns", "DB_MAX_IDLE_CONNS")
	v.BindEnv("database.max_open_conns", "DB_MAX_OPEN_CONNS")
	v.BindEnv("database.conn_max_lifetime", "DB_CONN_MAX_LIFETIME")
	v.BindEnv("database.conn_max_idle_time", "DB_CONN_MAX_IDLE_TIME")
	v.BindEnv("redis.addr", "REDIS_ADDR")
	v.BindEnv("redis.password", "REDIS_PASSWORD")
	v.BindEnv("jwt.secret", "JWT_SECRET")
	v.BindEnv("jwt.key_id", "JWT_KEY_ID")
	v.BindEnv("jwt.previous_keys_json", "JWT_PREVIOUS_KEYS_JSON")
	v.BindEnv("jwt.expiry_hours", "JWT_EXPIRY_HOURS")
	v.BindEnv("jwt.refresh_expiry_hours", "JWT_REFRESH_EXPIRY_HOURS")
	v.BindEnv("jwt.registration_enabled", "AUTH_REGISTRATION_ENABLED")
	v.BindEnv("server.port", "SERVER_PORT")
	v.BindEnv("server.mode", "SERVER_MODE")
	v.BindEnv("server.deployment_environment", "DEPLOYMENT_ENVIRONMENT")
	v.BindEnv("server.read_header_timeout", "SERVER_READ_HEADER_TIMEOUT")
	v.BindEnv("server.read_timeout", "SERVER_READ_TIMEOUT")
	v.BindEnv("server.write_timeout", "SERVER_WRITE_TIMEOUT")
	v.BindEnv("server.idle_timeout", "SERVER_IDLE_TIMEOUT")
	v.BindEnv("server.shutdown_timeout", "SERVER_SHUTDOWN_TIMEOUT")
	v.BindEnv("server.max_header_bytes", "SERVER_MAX_HEADER_BYTES")
	v.BindEnv("server.max_request_body_bytes", "SERVER_MAX_REQUEST_BODY_BYTES")
	v.BindEnv("server.swagger_enabled", "SWAGGER_ENABLED")
	v.BindEnv("sentry.dsn", "SENTRY_DSN")
	v.BindEnv("cors.allowed_origins", "CORS_ALLOWED_ORIGINS")
	v.BindEnv("metrics.enabled", "METRICS_ENABLED")
	v.BindEnv("log.level", "LOG_LEVEL")
	v.BindEnv("log.format", "LOG_FORMAT")
	v.BindEnv("llm.provider", "LLM_PROVIDER")
	v.BindEnv("llm.api_key", "LLM_API_KEY")
	v.BindEnv("llm.daily_budget_usd", "LLM_DAILY_BUDGET_USD")
	v.BindEnv("encryption_key", "ENCRYPTION_KEY")
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// Validate rejects configuration that would make the server unsafe or unable
// to operate predictably. It is part of Load so every executable gets the same
// startup gate instead of relying on checks in cmd/server only.
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if c.Server.Mode != "debug" && c.Server.Mode != "release" && c.Server.Mode != "test" {
		return fmt.Errorf("server.mode must be debug, release, or test")
	}
	if environment := c.Server.EffectiveDeploymentEnvironment(); environment != "development" && environment != "acceptance" && environment != "production" {
		return fmt.Errorf("server.deployment_environment must be development, acceptance, or production")
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}
	if c.Server.ReadHeaderTimeout <= 0 || c.Server.ReadTimeout < c.Server.ReadHeaderTimeout || c.Server.WriteTimeout < 0 || c.Server.IdleTimeout <= 0 || c.Server.ShutdownTimeout <= 0 {
		return fmt.Errorf("server timeouts require positive read-header/read/idle/shutdown values, read >= read-header, and write >= 0")
	}
	if c.Server.MaxHeaderBytes < 8*1024 || c.Server.MaxHeaderBytes > 2*1024*1024 {
		return fmt.Errorf("server.max_header_bytes must be between 8 KiB and 2 MiB")
	}
	if c.Server.MaxRequestBodyBytes < 1024*1024 || c.Server.MaxRequestBodyBytes > 100*1024*1024 {
		return fmt.Errorf("server.max_request_body_bytes must be between 1 MiB and 100 MiB")
	}
	if strings.TrimSpace(c.Database.Host) == "" || c.Database.Port < 1 || c.Database.Port > 65535 || strings.TrimSpace(c.Database.User) == "" || strings.TrimSpace(c.Database.DBName) == "" {
		return fmt.Errorf("database host, valid port, user, and dbname are required")
	}
	if c.Database.MaxIdleConns < 0 || c.Database.MaxOpenConns <= 0 || c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return fmt.Errorf("database pool requires 0 <= max_idle_conns <= max_open_conns")
	}
	if c.Database.ConnMaxLifetime <= 0 || c.Database.ConnMaxIdleTime <= 0 || c.Database.ConnMaxIdleTime > c.Database.ConnMaxLifetime {
		return fmt.Errorf("database pool requires 0 < conn_max_idle_time <= conn_max_lifetime")
	}
	validSSLMode := map[string]bool{"disable": true, "allow": true, "prefer": true, "require": true, "verify-ca": true, "verify-full": true}
	if !validSSLMode[c.Database.SSLMode] {
		return fmt.Errorf("database.sslmode is invalid")
	}
	if strings.TrimSpace(c.JWT.Secret) == "" {
		return fmt.Errorf("jwt.secret is required")
	}
	previousKeys, err := c.JWT.PreviousKeys()
	if err != nil {
		return err
	}
	if _, duplicate := previousKeys[c.JWT.EffectiveKeyID()]; duplicate {
		return fmt.Errorf("jwt current key_id must not also appear in previous_keys_json")
	}
	for keyID, secret := range previousKeys {
		if strings.TrimSpace(keyID) == "" || strings.TrimSpace(secret) == "" {
			return fmt.Errorf("jwt previous key IDs and secrets must be non-empty")
		}
		if c.Server.Mode == "release" && len(secret) < 32 {
			return fmt.Errorf("release mode requires previous JWT secrets of at least 32 characters")
		}
	}
	if c.JWT.ExpiryHours <= 0 || c.JWT.RefreshExpiryHours <= c.JWT.ExpiryHours {
		return fmt.Errorf("JWT expiry must be positive and refresh expiry must exceed access expiry")
	}
	if c.Log.Level != "debug" && c.Log.Level != "info" && c.Log.Level != "warn" && c.Log.Level != "error" {
		return fmt.Errorf("log.level must be debug, info, warn, or error")
	}
	if c.Log.Format != "console" && c.Log.Format != "json" {
		return fmt.Errorf("log.format must be console or json")
	}
	if c.SchemaDrift.OnDrift != "warn" && c.SchemaDrift.OnDrift != "panic" && c.SchemaDrift.OnDrift != "log_only" {
		return fmt.Errorf("schemadrift.on_drift must be warn, panic, or log_only")
	}
	if c.LLM.DailyBudgetUSD < 0 {
		return fmt.Errorf("llm.daily_budget_usd cannot be negative")
	}
	if c.Server.Mode == "release" {
		if c.JWT.Secret == defaultDevelopmentJWTSecret || len(c.JWT.Secret) < 32 {
			return fmt.Errorf("release mode requires a non-default JWT secret of at least 32 characters")
		}
		origins := strings.TrimSpace(c.CORS.AllowedOrigins)
		if origins == "" || origins == "*" {
			return fmt.Errorf("release mode requires explicit CORS allowed origins")
		}
	}
	if c.Server.Mode == "release" && c.JWT.RegistrationEnabled {
		return fmt.Errorf("public registration must be disabled in release mode")
	}
	if c.Server.Mode == "release" && c.Server.SwaggerEnabled {
		return fmt.Errorf("public Swagger must be disabled in release mode")
	}
	return nil
}

// NewLogger creates a new zap logger based on configuration.
func NewLogger(cfg *Config) (*zap.Logger, error) {
	var level zapcore.Level
	switch cfg.Log.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	var encoding string
	if cfg.Log.Format == "json" {
		encoding = "json"
	} else {
		encoding = "console"
	}

	logCfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Encoding:         encoding,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig:    zap.NewProductionEncoderConfig(),
	}

	if encoding == "console" {
		logCfg.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	}

	return logCfg.Build()
}
