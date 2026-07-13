package schemadrift

import (
	"testing"
)

func TestParseSQL_SimpleCreateTable(t *testing.T) {
	sql := `CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW()
);`

	tables, err := parseSQL(sql)
	if err != nil {
		t.Fatalf("parseSQL failed: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}

	tbl := tables[0]
	if tbl.Name != "users" {
		t.Errorf("expected table name 'users', got %q", tbl.Name)
	}

	cols := tbl.Columns
	if len(cols) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(cols))
	}

	// Check id column
	if cols[0].Name != "id" {
		t.Errorf("expected column 'id', got %q", cols[0].Name)
	}
	if cols[0].Type != "bigint" {
		t.Errorf("expected type 'bigint', got %q", cols[0].Type)
	}
	if !cols[0].PrimaryKey {
		t.Error("expected id to be primary key")
	}

	// Check name column
	if cols[1].Name != "name" {
		t.Errorf("expected column 'name', got %q", cols[1].Name)
	}
	if cols[1].Type != "character varying(100)" {
		t.Errorf("expected type 'character varying(100)', got %q", cols[1].Type)
	}
	if cols[1].Nullable {
		t.Errorf("expected name to be NOT NULL, but Nullable=true")
	}

	// Check email column
	if cols[2].Name != "email" {
		t.Errorf("expected column 'email', got %q", cols[2].Name)
	}
	if !cols[2].Nullable {
		t.Errorf("expected email to be nullable")
	}
	if cols[2].Default != "''" {
		t.Errorf("expected email default '', got %q", cols[2].Default)
	}

	// Check created_at column
	if cols[3].Name != "created_at" {
		t.Errorf("expected column 'created_at', got %q", cols[3].Name)
	}
	if cols[3].Type != "timestamp with time zone" {
		t.Errorf("expected type 'timestamp with time zone', got %q", cols[3].Type)
	}
}

func TestParseSQL_CreateTableIfNotExists(t *testing.T) {
	sql := `CREATE TABLE IF NOT EXISTS tariff_rule (
    id BIGSERIAL PRIMARY KEY,
    country_code VARCHAR(10) NOT NULL,
    duty_rate_pct NUMERIC(10,4) DEFAULT 0
);`

	tables, err := parseSQL(sql)
	if err != nil {
		t.Fatalf("parseSQL failed: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	if tables[0].Name != "tariff_rule" {
		t.Errorf("expected 'tariff_rule', got %q", tables[0].Name)
	}
}

func TestParseSQL_MultipleTables(t *testing.T) {
	sql := `CREATE TABLE table_a (id INT PRIMARY KEY, name TEXT);
CREATE TABLE table_b (id INT PRIMARY KEY, value BIGINT);`

	tables, err := parseSQL(sql)
	if err != nil {
		t.Fatalf("parseSQL failed: %v", err)
	}
	if len(tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(tables))
	}
	if tables[0].Name != "table_a" || tables[1].Name != "table_b" {
		t.Errorf("unexpected table ordering: %s %s", tables[0].Name, tables[1].Name)
	}
}

func TestParseSQL_WithComments(t *testing.T) {
	sql := `-- This is a comment
/* block comment */
CREATE TABLE products (
    id BIGSERIAL PRIMARY KEY, -- inline comment
    name VARCHAR(200) NOT NULL /* another comment */
);
-- trailing comment`

	tables, err := parseSQL(sql)
	if err != nil {
		t.Fatalf("parseSQL failed: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	if tables[0].Name != "products" {
		t.Errorf("expected 'products', got %q", tables[0].Name)
	}
	if len(tables[0].Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(tables[0].Columns))
	}
}

func TestParseSQL_SkipsAlterAndInsert(t *testing.T) {
	sql := `ALTER TABLE users ADD COLUMN age INT;
INSERT INTO logs VALUES (1);
CREATE TABLE actual_table (id INT);`

	tables, err := parseSQL(sql)
	if err != nil {
		t.Fatalf("parseSQL failed: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	if tables[0].Name != "actual_table" {
		t.Errorf("expected 'actual_table', got %q", tables[0].Name)
	}
}

func TestParseSQL_SkipsTableConstraints(t *testing.T) {
	sql := `CREATE TABLE orders (
    id BIGSERIAL,
    user_id BIGINT NOT NULL,
    total NUMERIC(10,2),
    PRIMARY KEY (id),
    UNIQUE (user_id, id)
);`

	tables, err := parseSQL(sql)
	if err != nil {
		t.Fatalf("parseSQL failed: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	if len(tables[0].Columns) != 3 {
		t.Errorf("expected 3 columns (skipping constraints), got %d", len(tables[0].Columns))
	}
}

func TestParseSQL_NoCreateTable(t *testing.T) {
	sql := `-- just a comment
INSERT INTO logs VALUES (1);
UPDATE config SET value = 'x';`

	tables, err := parseSQL(sql)
	if err != nil {
		t.Fatalf("parseSQL failed: %v", err)
	}
	if len(tables) != 0 {
		t.Errorf("expected 0 tables, got %d", len(tables))
	}
}

func TestParseSQL_Empty(t *testing.T) {
	tables, err := parseSQL("")
	if err != nil {
		t.Fatalf("parseSQL failed: %v", err)
	}
	if len(tables) != 0 {
		t.Errorf("expected 0 tables, got %d", len(tables))
	}
}

func TestParseSQL_ColumnTypes(t *testing.T) {
	sql := `CREATE TABLE type_test (
    c1 BIGINT,
    c2 INT,
    c3 SMALLINT,
    c4 TEXT,
    c5 BOOLEAN,
    c6 JSONB,
    c7 UUID,
    c8 TIMESTAMPTZ,
    c9 TIMESTAMP,
    c10 DATE,
    c11 REAL,
    c12 DOUBLE PRECISION,
    c13 NUMERIC(14,2),
    c14 VARCHAR(255),
    c15 BYTEA
);`

	tables, err := parseSQL(sql)
	if err != nil {
		t.Fatalf("parseSQL failed: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}

	typeChecks := []struct {
		idx  int
		name string
		want string
	}{
		{0, "c1", "bigint"},
		{1, "c2", "integer"},
		{2, "c3", "smallint"},
		{3, "c4", "text"},
		{4, "c5", "boolean"},
		{5, "c6", "jsonb"},
		{6, "c7", "uuid"},
		{7, "c8", "timestamp with time zone"},
		{8, "c9", "timestamp without time zone"},
		{9, "c10", "date"},
		{10, "c11", "real"},
		{11, "c12", "double precision"},
		{13, "c14", "character varying(255)"},
	}

	for _, tc := range typeChecks {
		col := tables[0].Columns[tc.idx]
		if col.Type != tc.want {
			t.Errorf("column %s: expected type %q, got %q", tc.name, tc.want, col.Type)
		}
	}

	// Check that NUMERIC(14,2) preserves its precision
	c13 := tables[0].Columns[12]
	if c13.Type != "numeric(14,2)" {
		t.Errorf("expected 'numeric(14,2)', got %q", c13.Type)
	}
}

func TestParseSQL_OpenParenOnSameLine(t *testing.T) {
	sql := `CREATE TABLE items
(
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL
);`

	tables, err := parseSQL(sql)
	if err != nil {
		t.Fatalf("parseSQL failed: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	if tables[0].Name != "items" {
		t.Errorf("expected 'items', got %q", tables[0].Name)
	}
	if len(tables[0].Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(tables[0].Columns))
	}
}

func TestParseSQL_SchemaMigrations(t *testing.T) {
	sql := `SELECT version FROM schema_migrations`
	tables, err := parseSQL(sql)
	if err != nil {
		t.Fatalf("parseSQL failed: %v", err)
	}
	if len(tables) != 0 {
		t.Errorf("expected 0 tables, got %d", len(tables))
	}
}

func TestNormalizeType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"BIGINT", "bigint"},
		{"bigserial", "bigint"},
		{"int8", "bigint"},
		{"INTEGER", "integer"},
		{"int", "integer"},
		{"int4", "integer"},
		{"SERIAL", "integer"},
		{"smallint", "smallint"},
		{"int2", "smallint"},
		{"bool", "boolean"},
		{"BOOLEAN", "boolean"},
		{"real", "real"},
		{"float4", "real"},
		{"double precision", "double precision"},
		{"float8", "double precision"},
		{"numeric", "numeric"},
		{"decimal", "numeric"},
		{"VARCHAR", "character varying"},
		{"character varying", "character varying"},
		{"text", "text"},
		{"timestamptz", "timestamp with time zone"},
		{"TIMESTAMPTZ", "timestamp with time zone"},
		{"jsonb", "jsonb"},
		{"JSON", "json"},
		{"uuid", "uuid"},
		{"bytea", "bytea"},
		{"varchar(255)", "character varying(255)"},
		{"NUMERIC(10,2)", "numeric(10,2)"},
		{"character(5)", "character(5)"},
	}
	for _, tc := range tests {
		got := NormalizeType(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeType(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestParseSQL_KnownMigrationPattern(t *testing.T) {
	sql := `CREATE TABLE IF NOT EXISTS lifecycle_step (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL,
    step VARCHAR(50) NOT NULL,
    agent_id VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    result TEXT,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`

	tables, err := parseSQL(sql)
	if err != nil {
		t.Fatalf("parseSQL failed: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	if tables[0].Name != "lifecycle_step" {
		t.Errorf("expected 'lifecycle_step', got %q", tables[0].Name)
	}
	if len(tables[0].Columns) != 12 {
		t.Errorf("expected 12 columns, got %d", len(tables[0].Columns))
	}

	// Check status column default
	statusCol := tables[0].Columns[4]
	if statusCol.Name != "status" {
		t.Errorf("expected status column, got %q", statusCol.Name)
	}
	if statusCol.Default != "'pending'" {
		t.Errorf("expected default 'pending', got %q", statusCol.Default)
	}
}

func TestParseSQL_AlterWithAddColumn(t *testing.T) {
	sql := `ALTER TABLE operation_log
  ADD COLUMN IF NOT EXISTS user_id BIGINT;
ALTER TABLE operation_log
  ADD COLUMN IF NOT EXISTS result VARCHAR(20) DEFAULT '';`

	tables, err := parseSQL(sql)
	if err != nil {
		t.Fatalf("parseSQL failed: %v", err)
	}
	if len(tables) != 0 {
		t.Errorf("expected 0 tables (ALTER TABLE should be skipped), got %d", len(tables))
	}
}

func TestParseSQL_ComplexCreateTableWithDefaultInParens(t *testing.T) {
	sql := `CREATE TABLE config (
    id BIGINT PRIMARY KEY,
    value TEXT DEFAULT 'default_value',
    created_at TIMESTAMPTZ DEFAULT NOW()
);`

	tables, err := parseSQL(sql)
	if err != nil {
		t.Fatalf("parseSQL failed: %v", err)
	}
	if len(tables) != 1 || len(tables[0].Columns) != 3 {
		t.Fatalf("expected 1 table with 3 columns, got %d tables, %d cols",
			len(tables), len(tables[0].Columns))
	}
}

func TestParseSQL_QuotedTableName(t *testing.T) {
	tables, err := parseSQL(`CREATE TABLE IF NOT EXISTS "user" (id BIGINT PRIMARY KEY);`)
	if err != nil || len(tables) != 1 || tables[0].Name != "user" {
		t.Fatalf("tables=%+v err=%v", tables, err)
	}
}

func TestParseSQL_DoesNotTreatCreateInsideProcedureAsTopLevelMigration(t *testing.T) {
	sql := `DO $$ BEGIN
IF NOT EXISTS (SELECT 1) THEN
  CREATE TABLE should_not_be_parsed (id BIGINT);
END IF;
END $$;`
	tables, err := parseSQL(sql)
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 0 {
		t.Fatalf("procedure body produced fake tables: %+v", tables)
	}
}

func TestParseColumnRenames(t *testing.T) {
	renames := parseColumnRenames(`
		ALTER TABLE spc_control_limit RENAME COLUMN metric_name TO metric;
		ALTER TABLE IF EXISTS "sample" RENAME COLUMN "old_name" TO "new_name";
	`)
	if len(renames) != 2 {
		t.Fatalf("renames=%+v", renames)
	}
	if renames[0] != (columnRename{table: "spc_control_limit", from: "metric_name", to: "metric"}) {
		t.Fatalf("first rename=%+v", renames[0])
	}
	if renames[1] != (columnRename{table: "sample", from: "old_name", to: "new_name"}) {
		t.Fatalf("second rename=%+v", renames[1])
	}
}

func TestParseColumnTypeChangesAndMultiwordCreateTypes(t *testing.T) {
	tables, err := parseSQL(`CREATE TABLE sample (created_at TIMESTAMP WITH TIME ZONE NOT NULL, score DOUBLE PRECISION);`)
	if err != nil || len(tables) != 1 || tables[0].Columns[0].Type != "timestamp with time zone" || tables[0].Columns[1].Type != "double precision" {
		t.Fatalf("tables=%+v err=%v", tables, err)
	}
	changes := parseColumnTypeChanges(`
		ALTER TABLE unrelated ADD COLUMN note TEXT;
		-- Preserve exact source bytes.
		ALTER TABLE sample ALTER COLUMN payload TYPE BYTEA USING payload::text::bytea;
	`)
	if len(changes) != 1 || changes[0] != (columnTypeChange{table: "sample", column: "payload", sqlType: "bytea"}) {
		t.Fatalf("changes=%+v", changes)
	}
}
