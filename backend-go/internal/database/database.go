package database

import (
	"fmt"

	"github.com/lingmirror/backend-go/internal/config"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect establishes a connection to PostgreSQL using GORM.
func Connect(cfg *config.Config, log *zap.Logger) (*gorm.DB, error) {
	dsn := cfg.Database.DSN()

	// Determine log level based on server mode
	var logLevel logger.LogLevel
	switch cfg.Server.Mode {
	case "debug":
		logLevel = logger.Info
	case "release":
		logLevel = logger.Warn
	default:
		logLevel = logger.Silent
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.Database.ConnMaxIdleTime)

	log.Info("database connected successfully")
	return db, nil
}
