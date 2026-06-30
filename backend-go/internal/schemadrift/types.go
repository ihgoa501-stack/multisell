package schemadrift

import "strings"

// ColumnDef represents a parsed column definition from either migration SQL or GORM model.
type ColumnDef struct {
	Name       string
	Type       string // normalized SQL type
	Nullable   bool
	Default    string // raw default expression
	PrimaryKey bool
}

// TableDef represents a table definition with its columns.
type TableDef struct {
	Name    string
	Columns []ColumnDef
}

// Merge adds columns from other that don't exist in this table def.
func (t *TableDef) Merge(other *TableDef) {
	existing := make(map[string]bool, len(t.Columns))
	for _, c := range t.Columns {
		existing[c.Name] = true
	}
	for _, c := range other.Columns {
		if !existing[c.Name] {
			t.Columns = append(t.Columns, c)
		}
	}
}

// DiffItem represents a single schema difference between migration SQL and GORM models.
type DiffItem struct {
	TableName      string
	ColumnName     string
	DiffType       string // "missing_table", "missing_column", "extra_column", "type_mismatch", "default_diff", "nullable_diff"
	MigrationValue string // value in migration SQL
	ModelValue     string // value in GORM model
}

// DiffReport is the complete drift detection result.
type DiffReport struct {
	Diffs          []DiffItem
	MigrationCount int // number of tables in migrations
	ModelCount     int // number of tables in models
}

// NormalizeType normalizes a PostgreSQL type string to a canonical form for comparison.
// It strips case and normalizes common synonyms.
func NormalizeType(raw string) string {
	t := strings.TrimSpace(raw)
	if t == "" {
		return ""
	}
	t = strings.ToLower(t)

	// Map synonyms to canonical form
	replacements := map[string]string{
		"bigint":                          "bigint",
		"int8":                            "bigint",
		"bigserial":                       "bigint",
		"serial8":                         "bigint",
		"integer":                         "integer",
		"int":                             "integer",
		"int4":                            "integer",
		"smallint":                        "smallint",
		"serial":                          "integer",
		"serial4":                         "integer",
		"int2":                            "smallint",
		"boolean":                         "boolean",
		"bool":                            "boolean",
		"double precision":                "double precision",
		"float8":                          "double precision",
		"double":                          "double precision",
		"real":                            "real",
		"float4":                          "real",
		"numeric":                         "numeric",
		"decimal":                         "numeric",
		"character varying":               "character varying",
		"varchar":                         "character varying",
		"character":                       "character",
		"char":                            "character",
		"text":                            "text",
		"timestamp with time zone":        "timestamp with time zone",
		"timestamptz":                     "timestamp with time zone",
		"timestamp without time zone":     "timestamp without time zone",
		"timestamp":                       "timestamp without time zone",
		"date":                            "date",
		"time with time zone":             "time with time zone",
		"timetz":                          "time with time zone",
		"time without time zone":          "time without time zone",
		"time":                            "time without time zone",
		"json":                            "json",
		"jsonb":                           "jsonb",
		"uuid":                            "uuid",
		"bytea":                           "bytea",
		"smallserial":                     "smallint",
	}

	// Handle parameterized types (e.g., varchar(255), numeric(10,2))
	paramIdx := strings.Index(t, "(")
	baseType := t
	params := ""
	if paramIdx >= 0 {
		baseType = strings.TrimSpace(t[:paramIdx])
		params = t[paramIdx:]
	}

	if canonical, ok := replacements[baseType]; ok {
		return canonical + params
	}

	// For unrecognized types, return as-is
	return t
}

// NormalizeDefault normalizes a default value for comparison.
// Strips surrounding quotes and whitespace so that 'active' and active compare equal.
func NormalizeDefault(raw string) string {
	v := strings.TrimSpace(raw)
	if len(v) >= 2 {
		if (v[0] == '\'' && v[len(v)-1] == '\'') || (v[0] == '"' && v[len(v)-1] == '"') {
			return v[1 : len(v)-1]
		}
	}
	return v
}
