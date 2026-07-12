package schemadrift

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MigrationParser reads migration SQL files and extracts CREATE TABLE definitions.
type MigrationParser struct {
	dir string
}

// NewMigrationParser creates a parser for the given migrations directory.
func NewMigrationParser(dir string) *MigrationParser {
	return &MigrationParser{dir: dir}
}

// ParseAll reads all .up.sql migration files and merges the table definitions.
func (p *MigrationParser) ParseAll() ([]TableDef, error) {
	entries, err := os.ReadDir(p.dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir %s: %w", p.dir, err)
	}

	re := regexp.MustCompile(`^\d+_.+\.up\.sql$`)
	var tableMap = make(map[string]*TableDef)

	for _, entry := range entries {
		if entry.IsDir() || !re.MatchString(entry.Name()) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(p.dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		tables, err := parseSQL(string(data))
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		for _, tbl := range tables {
			t := tbl
			if existing, ok := tableMap[tbl.Name]; ok {
				existing.Merge(&t)
			} else {
				tableMap[tbl.Name] = &t
			}
		}
	}

	result := make([]TableDef, 0, len(tableMap))
	for _, tbl := range tableMap {
		result = append(result, *tbl)
	}
	return result, nil
}

// knownMigrationLedgerTables are owned by migration tools rather than an
// application migration and must not be reported as extra application schema.
var knownMigrationLedgerTables = map[string]bool{
	"schema_migrations": true,
}

// createTableRE matches CREATE TABLE [IF NOT EXISTS] tablename (
var createTableRE = regexp.MustCompile(
	`(?i)^\s*CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?"?([a-zA-Z_][a-zA-Z0-9_]*)"?`,
)

// parseSQL extracts CREATE TABLE definitions from a SQL string.
// It handles multi-line, single-line, and IF NOT EXISTS forms.
// Non-CREATE statements (ALTER, INSERT, CREATE INDEX) are ignored.
func parseSQL(sql string) ([]TableDef, error) {
	// Remove block comments
	sql = removeBlockComments(sql)

	// Remove inline comments (-- to end of line), preserving newlines
	var cleanLines []string
	for _, line := range strings.Split(sql, "\n") {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		cleanLines = append(cleanLines, line)
	}
	sql = strings.Join(cleanLines, "\n")

	// Split into individual statements by semicolon
	statements := splitSQLStatements(sql)

	var tables []TableDef

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if !isCreateTableStatement(stmt) {
			continue
		}

		tbl, ok := extractCreateTableFromStmt(stmt)
		if ok {
			tables = append(tables, tbl)
		}
	}

	return tables, nil
}

// splitSQLStatements splits SQL by semicolons, respecting string literals.
func splitSQLStatements(sql string) []string {
	var statements []string
	var current strings.Builder
	inQuote := false
	var quoteChar byte

	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		if inQuote {
			current.WriteByte(ch)
			if ch == quoteChar {
				inQuote = false
			}
			continue
		}
		if ch == '\'' || ch == '"' {
			inQuote = true
			quoteChar = ch
			current.WriteByte(ch)
			continue
		}
		if ch == ';' {
			trimmed := strings.TrimSpace(current.String())
			if trimmed != "" {
				statements = append(statements, trimmed)
			}
			current.Reset()
			continue
		}
		current.WriteByte(ch)
	}
	// Last statement if no trailing semicolon
	trimmed := strings.TrimSpace(current.String())
	if trimmed != "" {
		statements = append(statements, trimmed)
	}

	return statements
}

// isCreateTableStatement checks if a statement starts with CREATE TABLE or CREATE TABLE IF NOT EXISTS.
func isCreateTableStatement(stmt string) bool {
	return createTableRE.MatchString(stmt)
}

// extractCreateTableFromStmt extracts table name and columns from a CREATE TABLE statement.
func extractCreateTableFromStmt(stmt string) (TableDef, bool) {
	matches := createTableRE.FindStringSubmatch(stmt)
	if matches == nil {
		return TableDef{}, false
	}
	tableName := strings.ToLower(matches[1])

	// Find the opening parenthesis — we need to locate it AFTER the table name
	afterTable := stmt[strings.Index(strings.ToLower(stmt), tableName)+len(tableName):]
	parenIdx := strings.Index(afterTable, "(")
	if parenIdx < 0 {
		return TableDef{Name: tableName}, true // table defined elsewhere, just name
	}

	content := afterTable[parenIdx+1:]

	// Find the matching closing parenthesis using depth counter
	depth := 0
	closeIdx := -1
	for i := 0; i < len(content); i++ {
		ch := content[i]
		if ch == '(' {
			depth++
		} else if ch == ')' {
			if depth == 0 {
				closeIdx = i
				break
			}
			depth--
		}
	}

	if closeIdx < 0 {
		return TableDef{Name: tableName}, true // unterminated, just name
	}

	columnsContent := content[:closeIdx]

	// Split into individual column definitions by comma at depth 0
	columnDefs := splitColumnDefs(columnsContent)

	tbl := TableDef{Name: tableName}
	for _, colDef := range columnDefs {
		colDef = strings.TrimSpace(colDef)
		// Skip table-level constraints (PRIMARY KEY (...), UNIQUE (...), etc.)
		if isTableConstraintLine(colDef) {
			continue
		}
		if col := parseColumnDef(colDef); col != nil {
			tbl.Columns = append(tbl.Columns, *col)
		}
	}

	return tbl, true
}

// splitColumnDefs splits the content between CREATE TABLE parentheses into individual
// column definitions, respecting nested parentheses and string literals.
func splitColumnDefs(content string) []string {
	var parts []string
	var current strings.Builder
	depth := 0
	inQuote := false
	var quoteChar byte

	for i := 0; i < len(content); i++ {
		ch := content[i]

		if inQuote {
			current.WriteByte(ch)
			if ch == quoteChar {
				inQuote = false
			}
			continue
		}

		switch {
		case ch == '\'' || ch == '"':
			inQuote = true
			quoteChar = ch
			current.WriteByte(ch)

		case ch == '(':
			depth++
			current.WriteByte(ch)

		case ch == ')':
			depth--
			current.WriteByte(ch)

		case ch == ',' && depth == 0:
			trimmed := strings.TrimSpace(current.String())
			if trimmed != "" {
				parts = append(parts, trimmed)
			}
			current.Reset()

		default:
			current.WriteByte(ch)
		}
	}

	trimmed := strings.TrimSpace(current.String())
	if trimmed != "" {
		parts = append(parts, trimmed)
	}

	return parts
}

// isTableConstraintLine checks if a column definition is actually a table-level constraint.
func isTableConstraintLine(s string) bool {
	upper := strings.ToUpper(strings.TrimSpace(s))
	for _, kw := range []string{
		"PRIMARY KEY", "UNIQUE", "INDEX", "KEY", "CONSTRAINT",
		"FOREIGN KEY", "CHECK", "EXCLUDE",
	} {
		if strings.HasPrefix(upper, kw) {
			return true
		}
	}
	return false
}

// removeBlockComments removes /* ... */ comments.
func removeBlockComments(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
			end := strings.Index(s[i+2:], "*/")
			if end < 0 {
				break
			}
			i += end + 4
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

// columnRE matches column definitions: name TYPE [constraints...]
var columnRE = regexp.MustCompile(
	`(?i)^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s+` + // column name
		`(` + // type group start
		`[a-zA-Z_][a-zA-Z0-9_]*` + // base type name
		`(?:\([^)]*\))?` + // optional precision e.g. (10,2)
		`(?:\[\])?` + // optional array suffix
		`)` + // type group end
		`(.*)`, // remaining constraints
)

// defaultNotAllowed is a set of lowercase constraint words that mark the end of a DEFAULT value.
var defaultNotAllowed = []string{"not null", "primary key", "unique", "references", "check"}

// parseColumnDef parses a single column definition string into a ColumnDef.
func parseColumnDef(s string) *ColumnDef {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	matches := columnRE.FindStringSubmatch(s)
	if matches == nil {
		return nil
	}

	name := strings.ToLower(matches[1])
	rawType := strings.TrimSpace(matches[2])
	constraints := strings.TrimSpace(matches[3])

	col := &ColumnDef{
		Name: name,
		Type: NormalizeType(rawType),
	}

	// Check NOT NULL
	col.Nullable = !isNotNull(constraints)

	// Check PRIMARY KEY
	if isPrimaryKey(constraints) {
		col.PrimaryKey = true
	}

	// Extract DEFAULT value
	if def := extractDefault(constraints); def != "" {
		col.Default = def
	}

	return col
}

// isNotNull checks if constraints contain NOT NULL.
func isNotNull(constraints string) bool {
	return strings.Contains(strings.ToUpper(constraints), "NOT NULL")
}

// isPrimaryKey checks if constraints contain PRIMARY KEY.
func isPrimaryKey(constraints string) bool {
	return strings.Contains(strings.ToUpper(constraints), "PRIMARY KEY")
}

// extractDefault finds the DEFAULT value in the constraints string.
func extractDefault(constraints string) string {
	upper := strings.ToUpper(constraints)
	idx := strings.Index(upper, "DEFAULT")
	if idx < 0 {
		return ""
	}

	rest := strings.TrimSpace(constraints[idx+7:])
	if rest == "" {
		return ""
	}

	end := findDefaultEnd(rest)
	return strings.TrimSpace(rest[:end])
}

// findDefaultEnd walks the string to find where a DEFAULT value ends.
// It handles strings, parentheses, and stops at constraint keywords or delimiters.
func findDefaultEnd(s string) int {
	inQuote := false
	var quoteChar byte
	parenDepth := 0
	start := 0

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if inQuote {
			if ch == quoteChar {
				inQuote = false
			}
			continue
		}

		switch {
		case ch == '\'' || ch == '"':
			inQuote = true
			quoteChar = ch

		case ch == '(':
			parenDepth++

		case ch == ')':
			if parenDepth > 0 {
				parenDepth--
				continue
			}
			return i

		case ch == ',' && parenDepth == 0:
			return i

		case ch == ' ' || ch == '\t':
			if start == 0 {
				start = i
			}
			after := strings.TrimSpace(s[i:])
			upper := strings.ToUpper(after)
			for _, kw := range defaultNotAllowed {
				if strings.HasPrefix(upper, kw) {
					return i
				}
			}

		default:
			start = 0
		}
	}

	return len(s)
}
