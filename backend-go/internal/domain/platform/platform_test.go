package platform

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/lingmirror/backend-go/internal/common"
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
	dsn := fmt.Sprintf("file:test_%d?mode=memory&cache=shared", n)
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
	if err := db.AutoMigrate(&Platform{}, &Store{}); err != nil {
		t.Fatalf("dbtest: AutoMigrate failed: %v", err)
	}
	return db
}

func newService(t *testing.T) *Service {
	t.Helper()
	return NewService(newTestDB(t), dbtest.NewLogger(t))
}

func TestPlatform_Create(t *testing.T) {
	svc := newService(t)

	in := &CreatePlatformInput{Name: "Shopee", Code: "shopee"}
	p, err := svc.CreatePlatform(in)
	if err != nil {
		t.Fatalf("CreatePlatform failed: %v", err)
	}
	if p.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}
	if p.Name != "Shopee" {
		t.Fatalf("Name = %q, want %q", p.Name, "Shopee")
	}
	if p.Status != 1 {
		t.Fatalf("Status = %d, want 1 (default)", p.Status)
	}
}

func TestPlatform_GetPlatform(t *testing.T) {
	svc := newService(t)

	p, _ := svc.CreatePlatform(&CreatePlatformInput{Name: "Lazada", Code: "lazada"})

	got, err := svc.GetPlatform(p.ID)
	if err != nil {
		t.Fatalf("GetPlatform failed: %v", err)
	}
	if got.Code != "lazada" {
		t.Fatalf("Code = %q, want %q", got.Code, "lazada")
	}
}

func TestPlatform_GetPlatform_NotFound(t *testing.T) {
	svc := newService(t)

	if _, err := svc.GetPlatform(999); err == nil {
		t.Fatal("expected error for non-existent platform")
	}
}

func TestPlatform_UpdatePlatform(t *testing.T) {
	svc := newService(t)

	p, _ := svc.CreatePlatform(&CreatePlatformInput{Name: "Old", Code: "old"})

	newName := "Updated"
	updated, err := svc.UpdatePlatform(p.ID, &UpdatePlatformInput{Name: &newName})
	if err != nil {
		t.Fatalf("UpdatePlatform failed: %v", err)
	}
	if updated.Name != "Updated" {
		t.Fatalf("Name = %q, want %q", updated.Name, "Updated")
	}
	if updated.Code != "old" {
		t.Fatalf("Code should not change, got %q", updated.Code)
	}
}

func TestPlatform_DeletePlatform(t *testing.T) {
	svc := newService(t)

	p, _ := svc.CreatePlatform(&CreatePlatformInput{Name: "Delete Me", Code: "del"})

	if err := svc.DeletePlatform(p.ID); err != nil {
		t.Fatalf("DeletePlatform failed: %v", err)
	}
	if _, err := svc.GetPlatform(p.ID); err == nil {
		t.Fatal("expected error after DeletePlatform")
	}
}

func TestPlatform_DeletePlatform_NotFound(t *testing.T) {
	svc := newService(t)

	if err := svc.DeletePlatform(999); err != gorm.ErrRecordNotFound {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestPlatform_ListPlatforms_Pagination(t *testing.T) {
	svc := newService(t)

	for i := 0; i < 25; i++ {
		_, _ = svc.CreatePlatform(&CreatePlatformInput{Name: dbtest.IToA(int64(i)), Code: dbtest.IToA(int64(i))})
	}

	p := &common.Pagination{Page: 1, Size: 10}
	items, total, err := svc.ListPlatforms(p, "")
	if err != nil {
		t.Fatalf("ListPlatforms failed: %v", err)
	}
	if total != 25 {
		t.Fatalf("total = %d, want 25", total)
	}
	if len(items) != 10 {
		t.Fatalf("returned %d items, want 10", len(items))
	}
}

func TestPlatform_ListPlatforms_Search(t *testing.T) {
	svc := newService(t)

	_, _ = svc.CreatePlatform(&CreatePlatformInput{Name: "Shopee", Code: "shopee"})
	_, _ = svc.CreatePlatform(&CreatePlatformInput{Name: "Lazada", Code: "lazada"})
	_, _ = svc.CreatePlatform(&CreatePlatformInput{Name: "Amazon", Code: "amazon"})

	p := &common.Pagination{Page: 1, Size: 20}
	items, total, err := svc.ListPlatforms(p, "shop")
	if err != nil {
		t.Fatalf("ListPlatforms search failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if items[0].Name != "Shopee" {
		t.Fatalf("Name = %q, want %q", items[0].Name, "Shopee")
	}
}

func TestStore_Create(t *testing.T) {
	svc := newService(t)

	p, _ := svc.CreatePlatform(&CreatePlatformInput{Name: "Platform1", Code: "p1"})

	in := &CreateStoreInput{UserID: 1, Name: "My Store", PlatformID: &p.ID}
	st, err := svc.CreateStore(in)
	if err != nil {
		t.Fatalf("CreateStore failed: %v", err)
	}
	if st.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}
	if st.Name != "My Store" {
		t.Fatalf("Name = %q, want %q", st.Name, "My Store")
	}
}

func TestStore_GetStore(t *testing.T) {
	svc := newService(t)

	p, _ := svc.CreatePlatform(&CreatePlatformInput{Name: "P", Code: "p"})
	st, _ := svc.CreateStore(&CreateStoreInput{UserID: 1, Name: "Shop1", PlatformID: &p.ID})

	got, err := svc.GetStore(st.ID)
	if err != nil {
		t.Fatalf("GetStore failed: %v", err)
	}
	if got.Name != "Shop1" {
		t.Fatalf("Name = %q, want %q", got.Name, "Shop1")
	}
}

func TestStore_Update(t *testing.T) {
	svc := newService(t)

	st, _ := svc.CreateStore(&CreateStoreInput{UserID: 1, Name: "Old"})

	newName := "Updated Store"
	updated, err := svc.UpdateStore(st.ID, &UpdateStoreInput{Name: &newName})
	if err != nil {
		t.Fatalf("UpdateStore failed: %v", err)
	}
	if updated.Name != "Updated Store" {
		t.Fatalf("Name = %q, want %q", updated.Name, "Updated Store")
	}
}

func TestStore_Delete(t *testing.T) {
	svc := newService(t)

	st, _ := svc.CreateStore(&CreateStoreInput{UserID: 1, Name: "DeleteMe"})

	if err := svc.DeleteStore(st.ID); err != nil {
		t.Fatalf("DeleteStore failed: %v", err)
	}
	if _, err := svc.GetStore(st.ID); err == nil {
		t.Fatal("expected error after DeleteStore")
	}
}

func TestStore_Delete_NotFound(t *testing.T) {
	svc := newService(t)

	if err := svc.DeleteStore(999); err != gorm.ErrRecordNotFound {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestStore_ListStores_FilterByPlatform(t *testing.T) {
	svc := newService(t)

	p1, _ := svc.CreatePlatform(&CreatePlatformInput{Name: "P1", Code: "p1"})
	p2, _ := svc.CreatePlatform(&CreatePlatformInput{Name: "P2", Code: "p2"})

	_, _ = svc.CreateStore(&CreateStoreInput{UserID: 1, Name: "S1", PlatformID: &p1.ID})
	_, _ = svc.CreateStore(&CreateStoreInput{UserID: 1, Name: "S2", PlatformID: &p1.ID})
	_, _ = svc.CreateStore(&CreateStoreInput{UserID: 1, Name: "S3", PlatformID: &p2.ID})

	pg := &common.Pagination{Page: 1, Size: 20}
	items, total, err := svc.ListStores(pg, &p1.ID, "")
	if err != nil {
		t.Fatalf("ListStores failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("returned %d items, want 2", len(items))
	}
}

func TestPlatform_CreatePlatform_WithSortOrder(t *testing.T) {
	svc := newService(t)

	so := 5
	p, err := svc.CreatePlatform(&CreatePlatformInput{Name: "Sorted", Code: "sorted", SortOrder: &so})
	if err != nil {
		t.Fatalf("CreatePlatform failed: %v", err)
	}
	if p.SortOrder != 5 {
		t.Fatalf("SortOrder = %d, want 5", p.SortOrder)
	}
}
