package schemadrift

import (
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config holds schema drift detector configuration.
type Config struct {
	Enabled bool   `mapstructure:"enabled"`
	OnDrift string `mapstructure:"on_drift"` // "warn" | "panic" | "log_only"
}

// DriftDetector checks the database schema against expected tables
// and compares migration definitions against GORM model structs.
type DriftDetector struct {
	db            *gorm.DB
	logger        *zap.Logger
	config        Config
	migrationsDir string
	models        []interface{}
}

// New creates a new DriftDetector.
func New(db *gorm.DB, logger *zap.Logger, cfg Config) *DriftDetector {
	return &DriftDetector{
		db:            db,
		logger:        logger,
		config:        cfg,
		migrationsDir: "migrations",
	}
}

// RegisterModels registers GORM model structs for reflection-based comparison.
func (d *DriftDetector) RegisterModels(models ...interface{}) {
	d.models = append(d.models, models...)
}

// SetMigrationsDir sets the path to the migrations directory.
func (d *DriftDetector) SetMigrationsDir(dir string) {
	d.migrationsDir = dir
}

// Check performs schema drift detection and migration health check.
func (d *DriftDetector) Check() {
	if !d.config.Enabled {
		d.logger.Debug("schemadrift: disabled, skipping check")
		return
	}

	// 1. Parse migration SQL files
	migrationTables, err := d.parseMigrations()
	if err != nil {
		d.logger.Warn("schemadrift: migration parsing failed", zap.Error(err))
	} else {
		d.logger.Info("schemadrift: parsed migrations",
			zap.Int("tables", len(migrationTables)))
	}

	// 2. Reflect GORM models
	modelTables := d.reflectModels()
	d.logger.Info("schemadrift: reflected models",
		zap.Int("tables", len(modelTables)))

	// 3. Compare migration SQL vs GORM models (static analysis)
	if len(migrationTables) > 0 && len(modelTables) > 0 {
		driftDiff := d.detectStatic(migrationTables, modelTables)
		d.reportDriftDiff(driftDiff)
	} else if len(migrationTables) > 0 {
		d.logger.Warn("schemadrift: static model comparison skipped because no models were registered; live database comparison remains active")
	}

	// 4. Migration version health check (requires DB)
	if d.db != nil {
		mh := NewMigrationChecker(d.db, d.logger, d.migrationsDir)
		health := mh.Check()
		d.reportMigrationHealth(health)
	}

	// 5. Live DB schema check (existing functionality)
	if d.db != nil && len(migrationTables) > 0 {
		actual, err := d.getActualTables()
		if err != nil {
			d.logger.Error("schemadrift: failed to query information_schema", zap.Error(err))
			return
		}
		r := d.detect(actual, migrationTables)
		d.report(r)
	}
}

// parseMigrations reads migration SQL files and extracts table definitions.
func (d *DriftDetector) parseMigrations() ([]TableDef, error) {
	parser := NewMigrationParser(d.migrationsDir)
	return parser.ParseAll()
}

// reflectModels extracts table definitions from registered GORM models.
func (d *DriftDetector) reflectModels() []TableDef {
	reflector := NewModelReflector()
	return reflector.Reflect(d.models)
}

// detectStatic compares migration SQL definitions against GORM model definitions.
func (d *DriftDetector) detectStatic(migrations []TableDef, models []TableDef) DiffReport {
	var report DiffReport

	migByName := make(map[string]TableDef, len(migrations))
	for _, t := range migrations {
		migByName[t.Name] = t
	}

	modByName := make(map[string]TableDef, len(models))
	for _, t := range models {
		modByName[t.Name] = t
	}

	report.MigrationCount = len(migrations)
	report.ModelCount = len(models)

	migrationNames := make(map[string]bool, len(migrations))
	for _, t := range migrations {
		migrationNames[t.Name] = true
	}
	modelNames := make(map[string]bool, len(models))
	for _, t := range models {
		modelNames[t.Name] = true
	}

	// Find tables in migrations but missing from models
	for _, t := range migrations {
		if !modelNames[t.Name] {
			report.Diffs = append(report.Diffs, DiffItem{
				TableName:      t.Name,
				DiffType:       "missing_table",
				MigrationValue: "defined in migration SQL",
				ModelValue:     "not found in GORM models",
			})
		}
	}

	// Find tables in models but missing from migrations
	for _, t := range models {
		if !migrationNames[t.Name] {
			report.Diffs = append(report.Diffs, DiffItem{
				TableName:      t.Name,
				DiffType:       "extra_table",
				MigrationValue: "not found in migration SQL",
				ModelValue:     "defined in GORM model",
			})
		}
	}

	// Compare columns for tables that exist in both
	for _, migTbl := range migrations {
		modTbl, ok := modByName[migTbl.Name]
		if !ok {
			continue
		}

		migCols := make(map[string]ColumnDef, len(migTbl.Columns))
		for _, c := range migTbl.Columns {
			migCols[c.Name] = c
		}
		modCols := make(map[string]ColumnDef, len(modTbl.Columns))
		for _, c := range modTbl.Columns {
			modCols[c.Name] = c
		}

		// Columns in migration but missing from model
		for _, mc := range migTbl.Columns {
			if _, ok := modCols[mc.Name]; !ok {
				report.Diffs = append(report.Diffs, DiffItem{
					TableName:      migTbl.Name,
					ColumnName:     mc.Name,
					DiffType:       "missing_column",
					MigrationValue: fmt.Sprintf("type=%s default=%v", mc.Type, mc.Default),
					ModelValue:     "not found in GORM model",
				})
			}
		}

		// Columns in model but missing from migration (extra)
		for _, mc := range modTbl.Columns {
			if _, ok := migCols[mc.Name]; !ok {
				report.Diffs = append(report.Diffs, DiffItem{
					TableName:      modTbl.Name,
					ColumnName:     mc.Name,
					DiffType:       "extra_column",
					MigrationValue: "not found in migration SQL",
					ModelValue:     fmt.Sprintf("type=%s default=%v", mc.Type, mc.Default),
				})
			}
		}

		// Type comparison for columns in both
		for _, mc := range migTbl.Columns {
			modCol, ok := modCols[mc.Name]
			if !ok {
				continue
			}
			migType := mc.Type
			modType := modCol.Type

			if migType != "" && modType != "" && migType != modType {
				report.Diffs = append(report.Diffs, DiffItem{
					TableName:      migTbl.Name,
					ColumnName:     mc.Name,
					DiffType:       "type_mismatch",
					MigrationValue: migType,
					ModelValue:     modType,
				})
			}

			// Default value comparison
			migDefault := NormalizeDefault(mc.Default)
			modDefault := NormalizeDefault(modCol.Default)
			if migDefault != modDefault && (migDefault != "" || modDefault != "") {
				report.Diffs = append(report.Diffs, DiffItem{
					TableName:      migTbl.Name,
					ColumnName:     mc.Name,
					DiffType:       "default_diff",
					MigrationValue: migDefault,
					ModelValue:     modDefault,
				})
			}
		}
	}

	sort.Slice(report.Diffs, func(i, j int) bool {
		if report.Diffs[i].TableName != report.Diffs[j].TableName {
			return report.Diffs[i].TableName < report.Diffs[j].TableName
		}
		return report.Diffs[i].ColumnName < report.Diffs[j].ColumnName
	})

	return report
}

// reportDriftDiff logs the static drift analysis results.
func (d *DriftDetector) reportDriftDiff(r DiffReport) {
	if len(r.Diffs) == 0 {
		d.logger.Info("schemadrift: no static drift detected between migrations and models",
			zap.Int("migration_tables", r.MigrationCount),
			zap.Int("model_tables", r.ModelCount))
		return
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("schema drift between migrations and models: %d issues",
		len(r.Diffs)))

	typeCount := make(map[string]int)
	for _, dif := range r.Diffs {
		typeCount[dif.DiffType]++
	}

	var typeSummaries []string
	for _, t := range []string{"missing_table", "extra_table", "missing_column", "extra_column", "type_mismatch", "default_diff"} {
		if n := typeCount[t]; n > 0 {
			typeSummaries = append(typeSummaries, fmt.Sprintf("%s=%d", t, n))
		}
	}
	b.WriteString(fmt.Sprintf(" [%s]", strings.Join(typeSummaries, " ")))

	for _, dif := range r.Diffs {
		b.WriteString(fmt.Sprintf("\n  %s.%s: %s (migration: %s, model: %s)",
			dif.TableName, dif.ColumnName, dif.DiffType,
			dif.MigrationValue, dif.ModelValue))
	}

	msg := b.String()
	switch strings.ToLower(d.config.OnDrift) {
	case "log_only":
		d.logger.Info(msg)
	case "panic":
		d.logger.Panic(msg)
	default:
		d.logger.Warn(msg)
	}
}

// reportMigrationHealth logs the migration health check results.
func (d *DriftDetector) reportMigrationHealth(h MigrationHealth) {
	d.logger.Info("schemadrift: migration health check",
		zap.Int("current_version", h.CurrentVersion),
		zap.Int("expected_version", h.ExpectedVersion),
		zap.Int("migration_files", h.FileCount),
		zap.Int("applied_in_db", h.AppliedInDB),
		zap.Int("unapplied", len(h.UnappliedMigrations)),
		zap.Int("missing_files", len(h.MissingMigrations)),
		zap.Bool("dirty", h.Dirty),
	)
	if h.Dirty {
		d.logger.Error("schemadrift: migration database is dirty; startup must not be considered production-ready",
			zap.Int("version", h.CurrentVersion))
	}

	if len(h.UnappliedMigrations) > 0 {
		names := make([]string, len(h.UnappliedMigrations))
		for i, m := range h.UnappliedMigrations {
			names[i] = fmt.Sprintf("%06d_%s", m.Version, m.Name)
		}
		d.logger.Warn("schemadrift: unapplied migrations",
			zap.Strings("files", names))
	}
	if len(h.MissingMigrations) > 0 {
		names := make([]string, len(h.MissingMigrations))
		for i, m := range h.MissingMigrations {
			names[i] = fmt.Sprintf("%06d_%s", m.Version, m.Name)
		}
		d.logger.Warn("schemadrift: missing migration files",
			zap.Strings("files", names))
	}
	if len(h.DuplicateVersions) > 0 {
		d.logger.Warn("schemadrift: duplicate migration version numbers",
			zap.Ints("versions", h.DuplicateVersions))
	}
}

// columnInfo holds minimal column metadata from information_schema.
type columnInfo struct {
	TableName  string
	ColumnName string
	DataType   string
	IsNullable string
}

// getActualTables queries PostgreSQL information_schema for all tables in the public schema.
func (d *DriftDetector) getActualTables() (map[string][]columnInfo, error) {
	rows, err := d.db.Raw(`
		SELECT table_name, column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'public'
		ORDER BY table_name, ordinal_position
	`).Rows()
	if err != nil {
		return nil, fmt.Errorf("query information_schema.columns: %w", err)
	}
	defer rows.Close()

	tables := make(map[string][]columnInfo)
	for rows.Next() {
		var ci columnInfo
		if err := rows.Scan(&ci.TableName, &ci.ColumnName, &ci.DataType, &ci.IsNullable); err != nil {
			return nil, fmt.Errorf("scan column row: %w", err)
		}
		tables[ci.TableName] = append(tables[ci.TableName], ci)
	}
	return tables, nil
}

// driftReport summarizes schema differences between migration SQL and live DB.
type driftReport struct {
	MissingTables  []string
	ExtraTables    []string
	MissingColumns map[string][]string
	ColumnMismatch int
}

// detect compares migration SQL definitions against live DB schema.
func (d *DriftDetector) detect(actual map[string][]columnInfo, migrations []TableDef) driftReport {
	var r driftReport

	expectedSet := make(map[string]bool, len(migrations))
	migByName := make(map[string]TableDef, len(migrations))
	for _, t := range migrations {
		expectedSet[t.Name] = true
		migByName[t.Name] = t
	}

	for _, t := range migrations {
		if _, ok := actual[t.Name]; !ok {
			r.MissingTables = append(r.MissingTables, t.Name)
		}
	}

	for name := range actual {
		if knownMigrationLedgerTables[name] {
			continue
		}
		if !expectedSet[name] {
			r.ExtraTables = append(r.ExtraTables, name)
		}
	}

	r.MissingColumns = make(map[string][]string)
	for _, t := range migrations {
		actualCols, ok := actual[t.Name]
		if !ok {
			continue
		}
		actualColMap := make(map[string]bool, len(actualCols))
		actualColType := make(map[string]string, len(actualCols))
		for _, col := range actualCols {
			actualColMap[col.ColumnName] = true
			actualColType[col.ColumnName] = col.DataType
		}

		for _, col := range t.Columns {
			if !actualColMap[col.Name] {
				r.MissingColumns[t.Name] = append(r.MissingColumns[t.Name], col.Name)
			} else {
				actualType := NormalizeType(actualColType[col.Name])
				expectedType := col.Type
				if actualType != "" && expectedType != "" && actualType != expectedType {
					r.ColumnMismatch++
				}
			}
		}
	}

	sort.Strings(r.MissingTables)
	sort.Strings(r.ExtraTables)
	return r
}

// report logs drift findings and reacts per OnDrift policy.
func (d *DriftDetector) report(r driftReport) {
	totalMissingCols := 0
	for _, cols := range r.MissingColumns {
		totalMissingCols += len(cols)
	}

	if len(r.MissingTables) == 0 && len(r.ExtraTables) == 0 && totalMissingCols == 0 && r.ColumnMismatch == 0 {
		d.logger.Info("schemadrift: no live DB drift detected",
			zap.Int("table_count", len(r.MissingTables)))
		return
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("live DB schema drift: %d missing tables, %d extra tables, %d missing columns, %d type mismatches",
		len(r.MissingTables), len(r.ExtraTables), totalMissingCols, r.ColumnMismatch))

	if len(r.MissingTables) > 0 {
		b.WriteString(fmt.Sprintf("\n  missing tables: %s", strings.Join(r.MissingTables, ", ")))
	}
	if len(r.ExtraTables) > 0 {
		b.WriteString(fmt.Sprintf("\n  extra tables: %s", strings.Join(r.ExtraTables, ", ")))
	}
	for table, cols := range r.MissingColumns {
		if len(cols) > 0 {
			b.WriteString(fmt.Sprintf("\n  missing columns in %s: %s", table, strings.Join(cols, ", ")))
		}
	}

	msg := b.String()

	switch strings.ToLower(d.config.OnDrift) {
	case "log_only":
		d.logger.Info(msg)
	case "panic":
		d.logger.Panic(msg)
	default:
		d.logger.Warn(msg)
	}
}
