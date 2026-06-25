package search

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var dbCounter atomic.Int64

func ilikeMatch(s, pattern string) bool {
	var b strings.Builder
	b.WriteString("(?i)^")
	for _, c := range pattern {
		switch c {
		case '%':
			b.WriteString(".*")
		case '_':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString("$")
	matched, _ := regexp.MatchString(b.String(), s)
	return matched
}

// ilikeConnPool wraps *sql.DB to rewrite PostgreSQL ILIKE → SQLite LIKE.
// SQLite LIKE is case-insensitive for ASCII by default.
type ilikeConnPool struct {
	*sql.DB
}

func (p *ilikeConnPool) rewrite(query string) string {
	return strings.ReplaceAll(query, " ILIKE ", " LIKE ")
}

func (p *ilikeConnPool) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return p.DB.QueryContext(ctx, p.rewrite(query), args...)
}

func (p *ilikeConnPool) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return p.DB.QueryRowContext(ctx, p.rewrite(query), args...)
}

func (p *ilikeConnPool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return p.DB.ExecContext(ctx, p.rewrite(query), args...)
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	n := dbCounter.Add(1)
	dsn := fmt.Sprintf("file:test_search_%d?mode=memory&cache=shared", n)
	sqlDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("dbtest: sql.Open failed: %v", err)
	}
	pool := &ilikeConnPool{DB: sqlDB}
	db, err := gorm.Open(sqlite.New(sqlite.Config{
		Conn: pool,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("dbtest: gorm.Open failed: %v", err)
	}

	db.Exec(`CREATE TABLE IF NOT EXISTS product (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS sku (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_id INTEGER NOT NULL,
		code TEXT,
		spec_desc TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS sales_order (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		order_no TEXT,
		recipient_name TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS after_sales_order (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		reason TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS exception_item (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS settlement (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		settlement_no TEXT
	)`)

	return db
}

func newService(t *testing.T) *Service {
	t.Helper()
	return NewService(newTestDB(t), dbtest.NewLogger(t))
}

func seedTestData(t *testing.T, db *gorm.DB) {
	t.Helper()

	db.Exec(`INSERT INTO product (name) VALUES ('Wireless Mouse')`)
	db.Exec(`INSERT INTO product (name) VALUES ('Bluetooth Keyboard')`)
	db.Exec(`INSERT INTO product (name) VALUES ('USB Hub')`)

	db.Exec(`INSERT INTO sku (product_id, code, spec_desc) VALUES (1, 'WM-001', 'Black 1200DPI')`)
	db.Exec(`INSERT INTO sku (product_id, code, spec_desc) VALUES (2, 'BK-002', 'White Mechanical')`)

	db.Exec(`INSERT INTO sales_order (order_no, recipient_name) VALUES ('ORD-20260101-001', 'John Smith')`)
	db.Exec(`INSERT INTO sales_order (order_no, recipient_name) VALUES ('ORD-20260102-002', 'Jane Doe')`)

	db.Exec(`INSERT INTO after_sales_order (reason) VALUES ('Wireless Mouse defective')`)
	db.Exec(`INSERT INTO exception_item (title) VALUES ('Payment timeout for order ORD-20260101')`)
	db.Exec(`INSERT INTO settlement (settlement_no) VALUES ('STL-202601001')`)
}

func TestSearch_ProductMatch(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`INSERT INTO product (name) VALUES ('Wireless Mouse')`)

	results, err := svc.Search("Wireless", 20)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Type != "product" {
		t.Fatalf("Type = %q, want %q", results[0].Type, "product")
	}
	if results[0].Title != "Wireless Mouse" {
		t.Fatalf("Title = %q, want %q", results[0].Title, "Wireless Mouse")
	}
}

func TestSearch_SKUMatch(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`INSERT INTO product (name) VALUES ('Widget')`)
	db.Exec(`INSERT INTO sku (product_id, code, spec_desc) VALUES (1, 'WG-999', 'Red Large')`)

	results, err := svc.Search("WG-999", 20)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Type != "sku" {
		t.Fatalf("Type = %q, want %q", results[0].Type, "sku")
	}
}

func TestSearch_OrderMatch(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`INSERT INTO sales_order (order_no, recipient_name) VALUES ('ORD-20260601-001', 'John Smith')`)

	results, err := svc.Search("ORD-20260601", 20)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Type != "order" {
		t.Fatalf("Type = %q, want %q", results[0].Type, "order")
	}
	if results[0].Title != "ORD-20260601-001" {
		t.Fatalf("Title = %q, want %q", results[0].Title, "ORD-20260601-001")
	}
}

func TestSearch_OrderMatchByRecipient(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`INSERT INTO sales_order (order_no, recipient_name) VALUES ('ORD-X', 'Alice Wonder')`)

	results, err := svc.Search("Alice", 20)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Subtitle != "Alice Wonder" {
		t.Fatalf("Subtitle = %q, want %q", results[0].Subtitle, "Alice Wonder")
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	svc := newService(t)

	results, err := svc.Search("", 20)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("got %d results, want 0 for empty query", len(results))
	}
}

func TestSearch_NoMatch(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`INSERT INTO product (name) VALUES ('Widget')`)

	results, err := svc.Search("zzzznonexistent", 20)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("got %d results, want 0", len(results))
	}
}

func TestSearch_MultiTableResults(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	seedTestData(t, db)

	results, err := svc.Search("Wireless", 20)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("expected results from product and after_sales_order, got %d", len(results))
	}

	types := make(map[string]bool)
	for _, r := range results {
		types[r.Type] = true
	}
	if !types["product"] {
		t.Fatal("expected product result")
	}
	if !types["aftersales"] {
		t.Fatal("expected aftersales result")
	}
}

func TestSearch_LimitRespected(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	for i := 0; i < 20; i++ {
		db.Exec(`INSERT INTO product (name) VALUES (?)`, dbtest.IToA(int64(i))+"-Widget")
	}

	results, err := svc.Search("Widget", 5)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) > 5 {
		t.Fatalf("got %d results, limit was 5", len(results))
	}
}

func TestSearch_URLFormat(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`INSERT INTO product (name) VALUES ('Gadget')`)
	db.Exec(`INSERT INTO sku (product_id, code, spec_desc) VALUES (1, 'GD-001', 'Blue')`)
	db.Exec(`INSERT INTO sales_order (order_no, recipient_name) VALUES ('ORD-1', 'Bob')`)

	results, err := svc.Search("Gadget", 20)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	for _, r := range results {
		if r.URL == "" {
			t.Fatalf("URL is empty for result type=%q id=%d", r.Type, r.ID)
		}
	}
}

func TestSearch_RecentPlaceholder(t *testing.T) {
	svc := newService(t)

	recent := svc.Recent("user-1")
	if len(recent) != 0 {
		t.Fatalf("Recent should return empty placeholder, got %d", len(recent))
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`INSERT INTO product (name) VALUES ('USB-C Cable')`)

	results, err := svc.Search("usb-c", 20)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1 (case-insensitive)", len(results))
	}
}
