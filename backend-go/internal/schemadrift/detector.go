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

// DriftDetector checks the database schema against expected tables.
type DriftDetector struct {
	db     *gorm.DB
	logger *zap.Logger
	config Config
}

// columnInfo holds minimal column metadata from information_schema.
type columnInfo struct {
	TableName  string
	ColumnName string
	DataType   string
	IsNullable string
}

// New creates a new DriftDetector.
func New(db *gorm.DB, logger *zap.Logger, cfg Config) *DriftDetector {
	return &DriftDetector{
		db:     db,
		logger: logger,
		config: cfg,
	}
}

// Check queries information_schema and compares against the known table baseline.
func (d *DriftDetector) Check() {
	if !d.config.Enabled {
		d.logger.Debug("schemadrift: disabled, skipping check")
		return
	}

	actual, err := d.getActualTables()
	if err != nil {
		d.logger.Error("schemadrift: failed to query information_schema", zap.Error(err))
		return
	}

	r := d.detect(actual)
	d.report(r)
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

// driftReport summarizes schema differences.
type driftReport struct {
	MissingTables  []string
	ExtraTables    []string
	ColumnMismatch int
}

// detect compares actual tables against the known baseline.
func (d *DriftDetector) detect(actual map[string][]columnInfo) driftReport {
	var r driftReport

	// Build set of expected table names.
	expectedSet := make(map[string]bool, len(expectedTables))
	for _, name := range expectedTables {
		expectedSet[name] = true
	}

	// Check for missing tables.
	for _, name := range expectedTables {
		if _, ok := actual[name]; !ok {
			r.MissingTables = append(r.MissingTables, name)
		}
	}

	// Check for extra tables.
	for name := range actual {
		if !expectedSet[name] {
			r.ExtraTables = append(r.ExtraTables, name)
		}
	}

	// Column-level comparison for known tables.
	for name, cols := range actual {
		known, ok := knownColumnBaseline[name]
		if !ok {
			continue
		}
		for _, col := range cols {
			exp, found := known[col.ColumnName]
			if !found {
				r.ColumnMismatch++
				continue
			}
			if exp.DataType != "" && exp.DataType != col.DataType {
				r.ColumnMismatch++
			}
		}
	}

	sort.Strings(r.MissingTables)
	sort.Strings(r.ExtraTables)
	return r
}

// report logs drift findings and reacts per OnDrift policy.
func (d *DriftDetector) report(r driftReport) {
	if len(r.MissingTables) == 0 && len(r.ExtraTables) == 0 && r.ColumnMismatch == 0 {
		d.logger.Info("schemadrift: no drift detected",
			zap.Int("table_count", len(expectedTables)))
		return
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("schema drift: %d missing, %d extra tables, %d column mismatches",
		len(r.MissingTables), len(r.ExtraTables), r.ColumnMismatch))

	if len(r.MissingTables) > 0 {
		b.WriteString(fmt.Sprintf("\n  missing tables: %s", strings.Join(r.MissingTables, ", ")))
	}
	if len(r.ExtraTables) > 0 {
		b.WriteString(fmt.Sprintf("\n  extra tables: %s", strings.Join(r.ExtraTables, ", ")))
	}

	msg := b.String()

	switch strings.ToLower(d.config.OnDrift) {
	case "log_only":
		d.logger.Info(msg)
	case "panic":
		d.logger.Panic(msg)
	default:
		// "warn" or unknown value defaults to warn-level logging
		d.logger.Warn(msg)
	}
}

// expectedTables is the baseline of tables that should exist in the database.
// Derived from CREATE TABLE statements in backend-go/migrations/*.up.sql.
var expectedTables = []string{
	"after_sales_order",
	"agent_action",
	"agent_decision",
	"agent_episode",
	"agent_evolution_config",
	"agent_nudge",
	"agent_pending_action",
	"agent_trust_score",
	"agentos_action_proposal",
	"agentos_approval_request",
	"agentos_command_execution",
	"agentos_operation_log",
	"agentos_outcome_review",
	"ai_evidence_ref",
	"ai_trace",
	"ai_trace_event",
	"alert_rule",
	"allocation_rule",
	"analysis_feedback",
	"approval_policy_rule",
	"brand",
	"category",
	"cost_allocation_batch",
	"cost_allocation_item",
	"event_outbox",
	"exception_item",
	"exchange_rate",
	"finance_account",
	"finance_ledger_entry",
	"finance_transaction",
	"honcho_profile",
	"import_batch",
	"import_batch_row",
	"inventory",
	"inventory_log",
	"inventory_warehouse",
	"listing_task",
	"listing_task_item",
	"llm_cost_logs",
	"metabolism_log",
	"notification",
	"operation_log",
	"order_import",
	"order_import_batch",
	"order_import_item",
	"permission",
	"personal_rule",
	"platform",
	"platform_attribute_mapping",
	"platform_category_mapping",
	"platform_fee_rule",
	"platform_integration_account",
	"platform_settlement_batch",
	"platform_settlement_item",
	"pre_listing_decision",
	"price",
	"price_change_log",
	"product",
	"product_analysis",
	"product_canvases",
	"product_image_gen",
	"product_listing",
	"product_supplier",
	"prompt_template",
	"role",
	"role_permission",
	"rule_conflict",
	"rule_mark_change",
	"sales_order",
	"sales_order_item",
	"sales_order_shipping_snapshot",
	"sales_order_status_log",
	"settlement",
	"settlement_item",
	"shipping_bill_batch",
	"shipping_bill_item",
	"shipping_channel",
	"shipping_provider",
	"shipping_quote_rule",
	"shipping_zone",
	"sku",
	"sku_return_stats",
	"sourcing_1688_product",
	"spc_control_limit",
	"spec_name",
	"spec_value",
	"stores",
	"supplier",
	"supply_chain_flow",
	"supply_chain_tracking",
	"system_config",
	"tariff_rule",
	"unified_action",
	"user",
	"user_role",
	"warehouse",
}

// knownColumnBaseline holds expected column types for key tables.
// An empty string for DataType means skip type checking (name-only match).
type columnBaseline struct {
	DataType   string
	IsNullable bool
}

var knownColumnBaseline = map[string]map[string]columnBaseline{
	"product": {
		"id":          {DataType: "bigint"},
		"sku_id":      {DataType: "bigint"},
		"name":        {DataType: "character varying"},
		"description": {DataType: "text"},
		"status":      {DataType: "character varying"},
		"created_at":  {DataType: "timestamp with time zone"},
		"updated_at":  {DataType: "timestamp with time zone"},
	},
	"sku": {
		"id":         {DataType: "bigint"},
		"sku_code":   {DataType: "character varying"},
		"name":       {DataType: "character varying"},
		"price":      {DataType: "numeric"},
		"created_at": {DataType: "timestamp with time zone"},
		"updated_at": {DataType: "timestamp with time zone"},
	},
	"inventory": {
		"id":        {DataType: "bigint"},
		"sku_id":    {DataType: "bigint"},
		"quantity":  {DataType: "integer"},
		"created_at": {DataType: "timestamp with time zone"},
		"updated_at": {DataType: "timestamp with time zone"},
	},
	"user": {
		"id":       {DataType: "bigint"},
		"username": {DataType: "character varying"},
		"email":    {DataType: "character varying"},
		"password": {DataType: "character varying"},
		"status":   {DataType: "character varying"},
		"created_at": {DataType: "timestamp with time zone"},
		"updated_at": {DataType: "timestamp with time zone"},
	},
	"sales_order": {
		"id":           {DataType: "bigint"},
		"order_no":     {DataType: "character varying"},
		"status":       {DataType: "character varying"},
		"total_amount": {DataType: "numeric"},
		"created_at":   {DataType: "timestamp with time zone"},
		"updated_at":   {DataType: "timestamp with time zone"},
	},
	"platform_integration_account": {
		"id":              {DataType: "bigint"},
		"platform_id":     {DataType: "bigint"},
		"account_name":    {DataType: "character varying"},
		"credentials":     {DataType: "text"},
		"status":          {DataType: "character varying"},
		"last_sync_at":    {DataType: "timestamp with time zone"},
		"created_at":      {DataType: "timestamp with time zone"},
		"updated_at":      {DataType: "timestamp with time zone"},
	},
}
