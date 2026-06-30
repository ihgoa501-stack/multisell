package schemadrift

import (
	"reflect"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// ModelReflector uses reflection to extract table definitions from GORM model structs.
type ModelReflector struct{}

// NewModelReflector creates a new ModelReflector.
func NewModelReflector() *ModelReflector {
	return &ModelReflector{}
}

// Reflect extracts table definitions from a slice of GORM model structs.
// Each model must implement TableName() string.
func (r *ModelReflector) Reflect(models []interface{}) []TableDef {
	var result []TableDef
	seen := make(map[string]bool)

	for _, m := range models {
		t := reflect.TypeOf(m)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		tableName := getTableName(m)
		if tableName == "" || seen[tableName] {
			continue
		}
		seen[tableName] = true

		td := TableDef{Name: tableName}
		td.Columns = r.reflectColumns(t)
		result = append(result, td)
	}
	return result
}

// getTableName calls the TableName() method on the model via reflection.
func getTableName(model interface{}) string {
	v := reflect.ValueOf(model)

	// Try pointer receiver first
	if v.Kind() == reflect.Ptr {
		if tn := callTableName(v); tn != "" {
			return tn
		}
	} else {
		// Try value receiver
		if tn := callTableName(v); tn != "" {
			return tn
		}
		// Also try pointer to value
		if v.CanAddr() {
			if tn := callTableName(v.Addr()); tn != "" {
				return tn
			}
		}
	}
	return ""
}

func callTableName(v reflect.Value) string {
	method := v.MethodByName("TableName")
	if !method.IsValid() {
		return ""
	}
	out := method.Call(nil)
	if len(out) == 0 {
		return ""
	}
	return strings.ToLower(out[0].String())
}

// reflectColumns extracts column definitions from struct fields.
func (r *ModelReflector) reflectColumns(t reflect.Type) []ColumnDef {
	var cols []ColumnDef

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		// Skip unexported fields
		if !f.IsExported() {
			continue
		}

		// Skip embedded structs (e.g., gorm.Model)
		if f.Anonymous {
			// Unroll embedded struct fields
			cols = append(cols, r.reflectColumns(f.Type)...)
			continue
		}

		gormTag := f.Tag.Get("gorm")
		if gormTag == "" {
			continue
		}

		col := ColumnDef{}
		col.Name = extractGormTagValue(gormTag, "column")
		if col.Name == "" {
			// Fall back to snake_case of field name
			col.Name = toSnakeCase(f.Name)
		}

		// Check primaryKey
		tagLower := strings.ToLower(gormTag)
		if strings.Contains(tagLower, "primarykey") {
			col.PrimaryKey = true
		}

		// Check autoIncrement → implies serial-like type
		if strings.Contains(tagLower, "autoincrement") {
			if strings.Contains(tagLower, ";type:") {
				// type is explicit, respect it
			} else if col.Name == "id" {
				// Typically bigserial/autoincrement
			}
		}

		// Extract explicit type from tag
		col.Type = extractGormTagValue(gormTag, "type")
		if col.Type == "" {
			col.Type = goTypeToSQL(f.Type)
		} else {
			col.Type = NormalizeType(col.Type)
		}

		// Extract DEFAULT
		col.Default = extractGormTagValue(gormTag, "default")

		// Nullability: not present in gorm tag means nullable (unless primary key)
		// GORM uses pointer types for nullable fields
		if strings.Contains(tagLower, "not null") {
			col.Nullable = false
		} else {
			col.Nullable = true
		}

		cols = append(cols, col)
	}
	return cols
}

// extractGormTagValue extracts a value from a gorm tag string.
// e.g., extractGormTagValue("column:id;primaryKey", "column") → "id"
func extractGormTagValue(tag, key string) string {
	key = key + ":"
	lower := strings.ToLower(tag)
	keyLower := strings.ToLower(key)

	idx := strings.Index(lower, keyLower)
	if idx < 0 {
		return ""
	}
	start := idx + len(key)
	if start >= len(tag) {
		return ""
	}
	end := strings.Index(tag[start:], ";")
	if end < 0 {
		return strings.TrimSpace(tag[start:])
	}
	return strings.TrimSpace(tag[start : start+end])
}

// goTypeToSQL maps Go types to PostgreSQL SQL types.
func goTypeToSQL(t reflect.Type) string {
	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		return goTypeToSQL(t.Elem())
	}

	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return "integer"
	case reflect.Int64:
		return "bigint"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "integer"
	case reflect.Uint64:
		return "bigint"
	case reflect.Float32:
		return "real"
	case reflect.Float64:
		return "double precision"
	case reflect.Bool:
		return "boolean"
	case reflect.String:
		return "text"
	case reflect.Slice:
		if t == reflect.TypeOf([]byte(nil)) {
			return "bytea"
		}
		return "jsonb"
	case reflect.Struct:
		switch t {
		case reflect.TypeOf(time.Time{}):
			return "timestamp with time zone"
		case reflect.TypeOf(decimal.Decimal{}):
			return "numeric"
		}
		return "jsonb"
	case reflect.Map:
		return "jsonb"
	default:
		return "text"
	}
}

// toSnakeCase converts PascalCase to snake_case.
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(r + 32) // to lowercase
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
