package schemadrift

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MigrationFile represents a migration file found on disk.
type MigrationFile struct {
	Version int
	Name    string // filename without path and version prefix
}

// MigrationHealth holds the results of a migration version health check.
type MigrationHealth struct {
	CurrentVersion      int             // highest version in schema_migrations
	ExpectedVersion     int             // highest version among migration files
	FileCount           int             // total unique migration files found
	AppliedInDB         int             // versions found in both files and DB
	UnappliedMigrations []MigrationFile // files not in schema_migrations
	MissingMigrations   []MigrationFile // files that should exist but don't (gaps)
	DuplicateVersions   []int           // versions appearing in more than one file
	Dirty               bool            // golang-migrate interrupted/dirty state
}

// MigrationChecker queries migration status against the database and filesystem.
type MigrationChecker struct {
	db            *gorm.DB
	logger        *zap.Logger
	migrationsDir string
	fileRegex     *regexp.Regexp
}

// NewMigrationChecker creates a new MigrationChecker.
func NewMigrationChecker(db *gorm.DB, logger *zap.Logger, migrationsDir string) *MigrationChecker {
	return &MigrationChecker{
		db:            db,
		logger:        logger,
		migrationsDir: migrationsDir,
		fileRegex:     regexp.MustCompile(`^(\d+)_(.+)\.(up|down)\.sql$`),
	}
}

// Check performs the migration health check and returns the results.
func (c *MigrationChecker) Check() MigrationHealth {
	files, err := c.readMigrationFiles()
	if err != nil {
		c.logger.Warn("migration: cannot read migrations directory",
			zap.String("dir", c.migrationsDir), zap.Error(err))
		return MigrationHealth{}
	}

	currentVersion, dirty := c.getMigrationState()

	return c.analyze(files, currentVersion, dirty)
}

// readMigrationFiles reads all migration files from disk and extracts version+name.
// Returns ALL .up.sql files, including duplicates (same version from different files).
func (c *MigrationChecker) readMigrationFiles() ([]MigrationFile, error) {
	entries, err := os.ReadDir(c.migrationsDir)
	if err != nil {
		return nil, err
	}

	var files []MigrationFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		matches := c.fileRegex.FindStringSubmatch(name)
		if matches == nil {
			continue
		}
		version, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}
		if matches[3] == "up" {
			files = append(files, MigrationFile{
				Version: version,
				Name:    matches[2],
			})
		}
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].Version != files[j].Version {
			return files[i].Version < files[j].Version
		}
		return files[i].Name < files[j].Name
	})

	return files, nil
}

// getMigrationState reads golang-migrate's single-row version ledger.
func (c *MigrationChecker) getMigrationState() (int, bool) {
	var state struct {
		Version int
		Dirty   bool
	}
	if err := c.db.Raw("SELECT version, dirty FROM schema_migrations LIMIT 1").Scan(&state).Error; err != nil {
		c.logger.Warn("migration: cannot query schema_migrations — table may not exist",
			zap.Error(err))
		return 0, false
	}
	return state.Version, state.Dirty
}

// analyze compares migration files against golang-migrate's current version.
// files may contain duplicate entries for the same version (from different files).
func (c *MigrationChecker) analyze(files []MigrationFile, currentVersion int, dirty bool) MigrationHealth {
	health := MigrationHealth{CurrentVersion: currentVersion, Dirty: dirty}

	// Count occurrences of each version across all files
	versionCount := make(map[int]int)
	for _, f := range files {
		versionCount[f.Version]++
	}
	for v, count := range versionCount {
		if count > 1 {
			health.DuplicateVersions = append(health.DuplicateVersions, v)
		}
	}

	// Build map of unique file versions (map deduplicates naturally)
	fileVersions := make(map[int]MigrationFile, len(files))
	for _, f := range files {
		if _, exists := fileVersions[f.Version]; !exists {
			fileVersions[f.Version] = f
		}
	}

	// Expected version = max file version
	for v := range fileVersions {
		if v > health.ExpectedVersion {
			health.ExpectedVersion = v
		}
	}
	health.FileCount = len(fileVersions)

	for version := range fileVersions {
		if version <= currentVersion {
			health.AppliedInDB++
		}
	}

	// Find unapplied migrations: unique file versions not in schema_migrations
	for version, f := range fileVersions {
		if version > currentVersion {
			health.UnappliedMigrations = append(health.UnappliedMigrations, f)
		}
	}

	// Version numbers are identifiers, not a required contiguous sequence.
	// The only missing-file condition provable from golang-migrate's ledger is
	// that its current version has no corresponding migration file.
	if currentVersion > 0 {
		if _, exists := fileVersions[currentVersion]; !exists {
			health.MissingMigrations = append(health.MissingMigrations, MigrationFile{
				Version: currentVersion,
				Name:    "(current database version has no migration file)",
			})
		}
	}

	// Sort results
	sort.Slice(health.UnappliedMigrations, func(i, j int) bool {
		return health.UnappliedMigrations[i].Version < health.UnappliedMigrations[j].Version
	})
	sort.Slice(health.MissingMigrations, func(i, j int) bool {
		return health.MissingMigrations[i].Version < health.MissingMigrations[j].Version
	})
	sort.Ints(health.DuplicateVersions)

	return health
}

// FormatSummary returns a human-readable summary string.
func (h MigrationHealth) FormatSummary() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("migration health: current=%d expected=%d files=%d applied=%d",
		h.CurrentVersion, h.ExpectedVersion, h.FileCount, h.AppliedInDB))
	if h.Dirty {
		b.WriteString(" dirty=true")
	}

	if len(h.UnappliedMigrations) > 0 {
		b.WriteString(fmt.Sprintf("\n  unapplied (%d):", len(h.UnappliedMigrations)))
		for _, m := range h.UnappliedMigrations {
			b.WriteString(fmt.Sprintf("\n    %06d_%s", m.Version, m.Name))
		}
	}
	if len(h.MissingMigrations) > 0 {
		b.WriteString(fmt.Sprintf("\n  missing version gaps (%d):", len(h.MissingMigrations)))
		for _, m := range h.MissingMigrations {
			b.WriteString(fmt.Sprintf("\n    %06d %s", m.Version, m.Name))
		}
	}
	if len(h.DuplicateVersions) > 0 {
		b.WriteString(fmt.Sprintf("\n  duplicate versions: %v", h.DuplicateVersions))
	}
	return b.String()
}
