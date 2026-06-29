package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	CORS     CORSConfig     `mapstructure:"cors"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
	Sentry   SentryConfig   `mapstructure:"sentry"`
	LLM      LLMConfig      `mapstructure:"llm"`
	Prism    PrismConfig    `mapstructure:"prism"`
	EncryptionKey string         `mapstructure:"encryption_key"`
}

// LLMConfig holds LLM-related settings.
type LLMConfig struct {
	DailyBudgetUSD float64 `mapstructure:"daily_budget_usd"`
	Provider       string  `mapstructure:"provider"`
	APIKey         string  `mapstructure:"api_key"`
}

// ServerConfig holds server-specific settings.
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	SSLMode      string `mapstructure:"sslmode"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// JWTConfig holds JWT authentication settings.
type JWTConfig struct {
	Secret             string `mapstructure:"secret"`
	ExpiryHours        int    `mapstructure:"expiry_hours"`
	RefreshExpiryHours int    `mapstructure:"refresh_expiry_hours"`
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

// PrismConfig holds Prism image generation engine settings.
type PrismConfig struct {
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"`
	Timeout int    `mapstructure:"timeout"` // seconds
	Enabled bool   `mapstructure:"enabled"`
	Strict  bool   `mapstructure:"strict"` // block on service error vs warn+continue
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
	v.BindEnv("redis.addr", "REDIS_ADDR")
	v.BindEnv("redis.password", "REDIS_PASSWORD")
	v.BindEnv("jwt.secret", "JWT_SECRET")
	v.BindEnv("server.port", "SERVER_PORT")
	v.BindEnv("sentry.dsn", "SENTRY_DSN")
	v.BindEnv("cors.allowed_origins", "CORS_ALLOWED_ORIGINS")
	v.BindEnv("metrics.enabled", "METRICS_ENABLED")
	v.BindEnv("prism.base_url", "PRISM_BASE_URL")
	v.BindEnv("prism.api_key", "PRISM_API_KEY")
	v.BindEnv("prism.timeout", "PRISM_TIMEOUT")
	v.BindEnv("prism.enabled", "PRISM_ENABLED")
	v.BindEnv("prism.strict", "PRISM_STRICT")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
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
