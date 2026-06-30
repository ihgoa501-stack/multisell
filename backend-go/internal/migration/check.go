package migration

import (
	"os"
	"regexp"
	"strconv"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Checker queries migration status against the database and filesystem.
type Checker struct {
	db            *gorm.DB
	logger        *zap.Logger
	migrationsDir string
}

// NewChecker creates a new migration Checker.
// Defaults migrations directory to "migrations" relative to the working directory.
func NewChecker(db *gorm.DB, logger *zap.Logger) *Checker {
	return &Checker{
		db:            db,
		logger:        logger,
		migrationsDir: "migrations",
	}
}

// CurrentVersion queries schema_migrations table, returns latest applied version.
// Returns 0 if table doesn't exist (graceful degradation).
func (c *Checker) CurrentVersion() int {
	var version int
	err := c.db.Raw("SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&version).Error
	if err != nil {
		c.logger.Warn("migration: schema_migrations table not found, returning version 0",
			zap.Error(err))
		return 0
	}
	return version
}

// ExpectedVersion returns the latest migration file number from the filesystem.
func (c *Checker) ExpectedVersion() int {
	re := regexp.MustCompile(`^(\d+)_.*\.up\.sql$`)
	maxVersion := 0

	entries, err := os.ReadDir(c.migrationsDir)
	if err != nil {
		c.logger.Warn("migration: cannot read migrations directory",
			zap.String("dir", c.migrationsDir), zap.Error(err))
		return 0
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := re.FindStringSubmatch(entry.Name())
		if len(matches) < 2 {
			continue
		}
		version, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}
		if version > maxVersion {
			maxVersion = version
		}
	}

	if maxVersion == 0 {
		c.logger.Warn("migration: no migration files found", zap.String("dir", c.migrationsDir))
	}

	return maxVersion
}
