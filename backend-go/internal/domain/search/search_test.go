package search

import (
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// createSearchTables creates the tables that Search() queries.
func createSearchTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS product (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE IF NOT EXISTS sku (id INTEGER PRIMARY KEY, code TEXT, spec_desc TEXT)`,
		`CREATE TABLE IF NOT EXISTS sales_order (id INTEGER PRIMARY KEY, order_no TEXT, recipient_name TEXT)`,
		`CREATE TABLE IF NOT EXISTS after_sales_order (id INTEGER PRIMARY KEY, reason TEXT)`,
		`CREATE TABLE IF NOT EXISTS exception_item (id INTEGER PRIMARY KEY, title TEXT)`,
		`CREATE TABLE IF NOT EXISTS settlement (id INTEGER PRIMARY KEY, settlement_no TEXT)`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create table: %v", err)
		}
	}
}

// seedSearchData populates the search tables with sample data.
func seedSearchData(db *gorm.DB) {
	for _, stmt := range []string{
		`INSERT INTO product (name) VALUES ('iPhone 15 Case')`,
		`INSERT INTO product (name) VALUES ('Samsung Galaxy Case')`,
		`INSERT INTO sku (code, spec_desc) VALUES ('IP15-BLK', 'Black iPhone 15 Case')`,
		`INSERT INTO sales_order (order_no, recipient_name) VALUES ('ORD-2026-001', 'Alice')`,
		`INSERT INTO after_sales_order (reason) VALUES ('Item damaged')`,
		`INSERT INTO exception_item (title) VALUES ('Low stock alert')`,
		`INSERT INTO settlement (settlement_no) VALUES ('STL-2026-001')`,
	} {
		db.Exec(stmt)
	}
}

// TestService_Search is a basic regression: ILIKE-compatible search across all tables.
func TestService_Search(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))
	createSearchTables(t, db)
	seedSearchData(db)

	results, err := svc.Search("iPhone", 20)
	require.NoError(t, err)
	require.NotEmpty(t, results, "expected at least one search result for 'iPhone'")

	// product hit by name
	var foundProduct bool
	for _, r := range results {
		if r.Type == "product" && r.Title == "iPhone 15 Case" {
			foundProduct = true
			break
		}
	}
	assert.True(t, foundProduct, "expected product 'iPhone 15 Case' in search results")
}

// TestService_Search_EmptyQuery verifies that an empty keyword returns empty results.
func TestService_Search_EmptyQuery(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	results, err := svc.Search("", 20)
	assert.NoError(t, err)
	assert.Empty(t, results)
}

// TestService_Recent verifies that Recent returns a non-nil empty slice for a user with no history.
func TestService_Recent(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`CREATE TABLE IF NOT EXISTS recent_search (
		id INTEGER PRIMARY KEY,
		user_id TEXT,
		query TEXT,
		searched_at TEXT
	)`)

	recent := svc.Recent("user1")
	assert.NotNil(t, recent)
	assert.Empty(t, recent)
}

// --- New tests below ---

// TestSearch_EmptyKeyword verifies empty string returns empty results.
func TestSearch_EmptyKeyword(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	results, err := svc.Search("", 20)
	assert.NoError(t, err)
	assert.Empty(t, results)
}

// TestSearch_ShortKeyword verifies that a keyword shorter than 2 characters returns empty results.
func TestSearch_ShortKeyword(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	results, err := svc.Search("A", 20)
	assert.NoError(t, err)
	assert.Empty(t, results)

	results, err = svc.Search("1", 10)
	assert.NoError(t, err)
	assert.Empty(t, results)
}

// TestSearch_ByProductTitle seeds products and verifies search returns the matching product.
func TestSearch_ByProductTitle(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))
	createSearchTables(t, db)
	seedSearchData(db)

	results, err := svc.Search("iPhone", 20)
	require.NoError(t, err)
	require.NotEmpty(t, results, "expected at least one result for 'iPhone'")

	var found bool
	for _, r := range results {
		if r.Type == "product" && r.Title == "iPhone 15 Case" {
			found = true
			assert.Equal(t, "/products/"+itoa(r.ID), r.URL, "product URL mismatch")
			break
		}
	}
	assert.True(t, found, "expected product 'iPhone 15 Case' in results")
}

// TestSearch_BySkuCode seeds SKUs and verifies search returns the matching SKU.
func TestSearch_BySkuCode(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))
	createSearchTables(t, db)
	seedSearchData(db)

	results, err := svc.Search("IP15", 20)
	require.NoError(t, err)
	require.NotEmpty(t, results, "expected at least one result for 'IP15'")

	var found bool
	for _, r := range results {
		if r.Type == "sku" && r.Title == "IP15-BLK" {
			found = true
			assert.Equal(t, "/skus/"+itoa(r.ID), r.URL, "sku URL mismatch")
			break
		}
	}
	assert.True(t, found, "expected sku 'IP15-BLK' in results")
}

// TestSearch_MaxResults verifies the number of results does not exceed the requested limit.
func TestSearch_MaxResults(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))
	createSearchTables(t, db)

	// Seed 10 rows per table, all matching the same keyword
	for i := 1; i <= 10; i++ {
		db.Exec(`INSERT INTO product (name) VALUES (?)`, "Searchable Product "+itoa(int64(i)))
		db.Exec(`INSERT INTO sku (code, spec_desc) VALUES (?, ?)`, "SKU-"+itoa(int64(i)), "Searchable sku "+itoa(int64(i)))
		db.Exec(`INSERT INTO sales_order (order_no, recipient_name) VALUES (?, ?)`, "ORD-"+itoa(int64(i)), "Searchable recipient")
		db.Exec(`INSERT INTO after_sales_order (reason) VALUES (?)`, "Searchable reason "+itoa(int64(i)))
		db.Exec(`INSERT INTO exception_item (title) VALUES (?)`, "Searchable exception "+itoa(int64(i)))
		db.Exec(`INSERT INTO settlement (settlement_no) VALUES (?)`, "STL-"+itoa(int64(i)))
	}

	limit := 6
	results, err := svc.Search("Searchable", limit)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results), limit, "results must not exceed limit %d", limit)
	assert.NotEmpty(t, results, "expected some results for 'Searchable'")
}

// TestSearch_MultiType verifies search returns multiple result types (product, order, settlement, etc.).
func TestSearch_MultiType(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))
	createSearchTables(t, db)
	seedSearchData(db)

	// "2026" matches order_no (ORD-2026-001) and settlement_no (STL-2026-001)
	results, err := svc.Search("2026", 20)
	require.NoError(t, err)
	require.NotEmpty(t, results, "expected results for '2026'")

	types := make(map[string]int)
	for _, r := range results {
		types[r.Type]++
	}

	assert.GreaterOrEqual(t, len(types), 2, "expected at least 2 different result types")
	assert.Contains(t, types, "order", "expected order results for keyword '2026'")
	assert.Contains(t, types, "settlement", "expected settlement results for keyword '2026'")
}

// TestRecent verifies that RecordRecentSearch persists and Recent retrieves search history.
func TestRecent(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`CREATE TABLE IF NOT EXISTS recent_search (
		id INTEGER PRIMARY KEY,
		user_id TEXT,
		query TEXT,
		searched_at TEXT
	)`)

	svc.RecordRecentSearch("user1", "iPhone")
	svc.RecordRecentSearch("user1", "Samsung")
	svc.RecordRecentSearch("user2", "Oculus") // different user, should not appear

	recent := svc.Recent("user1")
	require.Len(t, recent, 2, "user1 should have 2 recent searches")
	assert.Equal(t, "Samsung", recent[0].Query, "most recent should be first")
	assert.Equal(t, "iPhone", recent[1].Query)
	assert.NotEmpty(t, recent[0].Timestamp, "timestamp should be set")
}

// TestRecent_Order verifies recent searches are returned in reverse chronological order.
func TestRecent_Order(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t)
	svc := NewService(db, dbtest.NewLogger(t))

	db.Exec(`CREATE TABLE IF NOT EXISTS recent_search (
		id INTEGER PRIMARY KEY,
		user_id TEXT,
		query TEXT,
		searched_at TEXT
	)`)

	svc.RecordRecentSearch("user1", "first")
	time.Sleep(time.Millisecond)
	svc.RecordRecentSearch("user1", "second")
	time.Sleep(time.Millisecond)
	svc.RecordRecentSearch("user1", "third")

	recent := svc.Recent("user1")
	require.Len(t, recent, 3)
	assert.Equal(t, "third", recent[0].Query)
	assert.Equal(t, "second", recent[1].Query)
	assert.Equal(t, "first", recent[2].Query)
}
