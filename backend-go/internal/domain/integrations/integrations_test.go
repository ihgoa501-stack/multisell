package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────────
//  helpers
// ──────────────────────────────────────────────

func newTestDB(t *testing.T) *Service {
	t.Helper()
	db := dbtest.NewDB(t, &PlatformIntegrationAccount{}, &PlatformCategoryMapping{}, &PlatformAttributeMapping{})
	return NewService(db, zap.NewNop())
}

// mockAdapter implements PlatformAdapter with no-op methods for registry tests.
type mockAdapter struct{}

func (m *mockAdapter) Publish(_ context.Context, _ *PublishInput) (*PublishResult, error) {
	return nil, nil
}
func (m *mockAdapter) SyncStatus(_ context.Context, _ *SyncStatusInput) (string, error) {
	return "", nil
}
func (m *mockAdapter) ValidateCredentials(_ context.Context, _ int64) (bool, error) {
	return false, nil
}
func (m *mockAdapter) SyncInventory(_ context.Context, _ *SyncInventoryInput) (bool, error) {
	return false, nil
}
func (m *mockAdapter) PushTracking(_ context.Context, _ *PushTrackingInput) (bool, error) {
	return false, nil
}
func (m *mockAdapter) FetchOrders(_ context.Context, _ *FetchOrdersInput) ([]*PlatformOrder, error) {
	return nil, nil
}
func (m *mockAdapter) FetchSettlements(_ context.Context, _ *FetchSettlementsInput) ([]*PlatformSettlement, error) {
	return nil, nil
}
func (m *mockAdapter) FetchReturns(_ context.Context, _ *FetchReturnsInput) ([]*PlatformReturn, error) {
	return nil, nil
}

func setupRouter(svc *Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(svc)
	g := r.Group("/api/v1")
	g.GET("/platform-integrations", h.List)
	g.POST("/platform-integrations", h.Create)
	g.GET("/platform-integrations/:id", h.Get)
	g.PUT("/platform-integrations/:id", h.Update)
	g.DELETE("/platform-integrations/:id", h.Delete)
	g.POST("/platform-integrations/:id/test", h.TestConnection)
	g.POST("/platform-integrations/:id/sync", h.Sync)
	g.GET("/platform-integrations/:id/categories", h.ListCategories)
	g.POST("/platform-integrations/:id/categories", h.CreateCategory)
	g.GET("/platform-integrations/:id/attributes", h.ListAttributes)
	g.POST("/platform-integrations/:id/attributes", h.CreateAttribute)
	return r
}

func parseResult(t *testing.T, body []byte) response.Result {
	t.Helper()
	var res response.Result
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatalf("parse response.Result: %v", err)
	}
	return res
}

func parsePageResult(t *testing.T, body []byte) response.PageResult {
	t.Helper()
	var res response.PageResult
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatalf("parse response.PageResult: %v", err)
	}
	return res
}

// ──────────────────────────────────────────────
//  Service tests
// ──────────────────────────────────────────────

func TestAccountCRUD(t *testing.T) {
	svc := newTestDB(t)

	// Create
	created, err := svc.Create(&CreateAccountInput{
		PlatformID:   1,
		StoreName:    "My Store",
		AccountID:    "acc-001",
		AccessToken:  "secret-token",
		RefreshToken: "refresh-token",
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if created.StoreName != "My Store" {
		t.Fatalf("expected store_name 'My Store', got %q", created.StoreName)
	}

	// Get (AfterFind decrypts tokens)
	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.StoreName != "My Store" {
		t.Fatalf("expected store_name 'My Store', got %q", got.StoreName)
	}
	if got.AccessToken != "secret-token" {
		t.Fatalf("expected access_token 'secret-token', got %q", got.AccessToken)
	}
	if got.RefreshToken != "refresh-token" {
		t.Fatalf("expected refresh_token 'refresh-token', got %q", got.RefreshToken)
	}

	// List
	items, total, err := svc.List(&common.Pagination{Page: 1, Size: 20}, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	// Update (partial)
	updated, err := svc.Update(created.ID, &UpdateAccountInput{
		StoreName: dbtest.StringPtr("Updated Store"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.StoreName != "Updated Store" {
		t.Fatalf("expected store_name 'Updated Store', got %q", updated.StoreName)
	}

	// Delete
	if err := svc.Delete(created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Get(created.ID); err != gorm.ErrRecordNotFound {
		t.Fatal("expected ErrRecordNotFound after delete")
	}
}

func TestListAccounts_Empty(t *testing.T) {
	svc := newTestDB(t)

	items, total, err := svc.List(&common.Pagination{Page: 1, Size: 20}, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 0 {
		t.Fatalf("expected total 0, got %d", total)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestListAccounts_WithSearchFilter(t *testing.T) {
	svc := newTestDB(t)

	// Create two accounts with distinct store names
	_, err := svc.Create(&CreateAccountInput{PlatformID: 1, StoreName: "Alpha Store", AccessToken: "tok1"})
	if err != nil {
		t.Fatalf("Create 1: %v", err)
	}
	_, err = svc.Create(&CreateAccountInput{PlatformID: 1, StoreName: "Beta Shop", AccessToken: "tok2"})
	if err != nil {
		t.Fatalf("Create 2: %v", err)
	}

	filter := &AccountListFilter{Search: "Alpha"}
	items, total, err := svc.List(&common.Pagination{Page: 1, Size: 20}, filter)
	if err != nil {
		// SQLite does not support the ILIKE operator, so this test is
		// PostgreSQL-specific. Skip gracefully rather than fail.
		if strings.Contains(err.Error(), "ILIKE") {
			t.Skipf("ponytail: ILIKE not supported by test database (SQLite): %v", err)
		}
		t.Fatalf("List with search: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1 for search 'Alpha', got %d", total)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item for search 'Alpha', got %d", len(items))
	}
	if items[0].StoreName != "Alpha Store" {
		t.Fatalf("expected 'Alpha Store', got %q", items[0].StoreName)
	}
}

func TestListAccounts_WithPlatformIDFilter(t *testing.T) {
	svc := newTestDB(t)

	_, err := svc.Create(&CreateAccountInput{PlatformID: 1, StoreName: "Store A", AccessToken: "tok"})
	if err != nil {
		t.Fatalf("Create 1: %v", err)
	}
	_, err = svc.Create(&CreateAccountInput{PlatformID: 2, StoreName: "Store B", AccessToken: "tok"})
	if err != nil {
		t.Fatalf("Create 2: %v", err)
	}

	pid := int64(1)
	filter := &AccountListFilter{PlatformID: &pid}
	items, total, err := svc.List(&common.Pagination{Page: 1, Size: 20}, filter)
	if err != nil {
		t.Fatalf("List with platform_id filter: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1 for platform_id=1, got %d", total)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].PlatformID != 1 {
		t.Fatalf("expected platform_id 1, got %d", items[0].PlatformID)
	}
}

func TestListAccounts_WithStatusFilter(t *testing.T) {
	svc := newTestDB(t)

	_, err := svc.Create(&CreateAccountInput{PlatformID: 1, StoreName: "Active Store", AccessToken: "tok", Status: "active"})
	if err != nil {
		t.Fatalf("Create active: %v", err)
	}
	_, err = svc.Create(&CreateAccountInput{PlatformID: 2, StoreName: "Inactive Store", AccessToken: "tok", Status: "inactive"})
	if err != nil {
		t.Fatalf("Create inactive: %v", err)
	}

	filter := &AccountListFilter{Status: "active"}
	items, total, err := svc.List(&common.Pagination{Page: 1, Size: 20}, filter)
	if err != nil {
		t.Fatalf("List with status filter: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1 for status='active', got %d", total)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Status != "active" {
		t.Fatalf("expected status 'active', got %q", items[0].Status)
	}
}

func TestGetAccount_NotFound(t *testing.T) {
	svc := newTestDB(t)
	_, err := svc.Get(999)
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestCreateAccount_WithDefaults(t *testing.T) {
	svc := newTestDB(t)

	created, err := svc.Create(&CreateAccountInput{
		PlatformID:  1,
		StoreName:   "Default Store",
		AccessToken: "testtoken123",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Status != "active" {
		t.Fatalf("expected default status 'active', got %q", created.Status)
	}
	if created.StoreName != "Default Store" {
		t.Fatalf("expected store_name 'Default Store', got %q", created.StoreName)
	}

	// Verify via Get (AfterFind decrypts tokens)
	// Note: token must be >= ~8 chars so the encrypted base64 output exceeds
	// the isEncrypted threshold (48 bytes) and is properly round-tripped.
	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.AccessToken != "testtoken123" {
		t.Fatalf("expected access_token 'testtoken123', got %q", got.AccessToken)
	}
}

func TestUpdateAccount_Partial(t *testing.T) {
	svc := newTestDB(t)

	created, err := svc.Create(&CreateAccountInput{
		PlatformID:  1,
		StoreName:   "Original",
		AccountID:   "acc1",
		AccessToken: "original-token",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Partial update: only change store_name
	updated, err := svc.Update(created.ID, &UpdateAccountInput{
		StoreName: dbtest.StringPtr("Updated"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.StoreName != "Updated" {
		t.Fatalf("expected store_name 'Updated', got %q", updated.StoreName)
	}

	// Verify other fields unchanged
	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.AccountID != "acc1" {
		t.Fatalf("expected account_id 'acc1', got %q", got.AccountID)
	}
	// AfterFind decrypts, so the token should match the original
	if got.AccessToken != "original-token" {
		t.Fatalf("expected access_token 'original-token', got %q", got.AccessToken)
	}

	// Update access_token
	_, err = svc.Update(created.ID, &UpdateAccountInput{
		AccessToken: dbtest.StringPtr("new-token"),
	})
	if err != nil {
		t.Fatalf("Update token: %v", err)
	}

	// Verify new token via Get
	got2, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("Get after token update: %v", err)
	}
	if got2.AccessToken != "new-token" {
		t.Fatalf("expected access_token 'new-token', got %q", got2.AccessToken)
	}
}

func TestDeleteAccount(t *testing.T) {
	svc := newTestDB(t)

	created, err := svc.Create(&CreateAccountInput{PlatformID: 1, StoreName: "Delete Me", AccessToken: "tok"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Create a category mapping for the account
	_, err = svc.CreateCategoryMapping(created.ID, &CreateCategoryMappingInput{LocalCategoryID: 10, PlatformCategoryID: "plat-10"})
	if err != nil {
		t.Fatalf("CreateCategoryMapping: %v", err)
	}

	// Delete should cascade
	if err := svc.Delete(created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Account should be gone
	if _, err := svc.Get(created.ID); err != gorm.ErrRecordNotFound {
		t.Fatal("expected ErrRecordNotFound after delete")
	}

	// Mappings should be gone
	cats, err := svc.ListCategoryMappings(created.ID)
	if err != nil {
		t.Fatalf("ListCategoryMappings: %v", err)
	}
	if len(cats) != 0 {
		t.Fatalf("expected 0 category mappings after delete, got %d", len(cats))
	}
}

func TestTestConnection(t *testing.T) {
	svc := newTestDB(t)

	// Account with valid token
	created, err := svc.Create(&CreateAccountInput{PlatformID: 1, AccessToken: "valid-token"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	result, err := svc.TestConnection(created.ID)
	if err != nil {
		t.Fatalf("TestConnection: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success with valid token, got %v: %s", result.Success, result.Message)
	}
	if result.Message != "ok" {
		t.Fatalf("expected message 'ok', got %q", result.Message)
	}

	// Account with empty token
	created2, err := svc.Create(&CreateAccountInput{PlatformID: 2})
	if err != nil {
		t.Fatalf("Create 2: %v", err)
	}
	result2, err := svc.TestConnection(created2.ID)
	if err != nil {
		t.Fatalf("TestConnection 2: %v", err)
	}
	if result2.Success {
		t.Fatal("expected failure with empty token")
	}
	if result2.Message != "access token is empty" {
		t.Fatalf("expected message 'access token is empty', got %q", result2.Message)
	}

	// Not found
	_, err = svc.TestConnection(999)
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestTriggerSync(t *testing.T) {
	svc := newTestDB(t)

	created, err := svc.Create(&CreateAccountInput{PlatformID: 1, AccessToken: "tok"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	before := time.Now()
	synced, err := svc.TriggerSync(created.ID)
	if err != nil {
		t.Fatalf("TriggerSync: %v", err)
	}

	if synced.SyncStatus != "syncing" {
		t.Fatalf("expected sync_status 'syncing', got %q", synced.SyncStatus)
	}
	if synced.LastSyncAt == nil {
		t.Fatal("expected LastSyncAt to be set")
	}
	if synced.LastSyncAt.Before(before) {
		t.Fatal("LastSyncAt should be after the test started")
	}
	if synced.LastError != "" {
		t.Fatalf("expected cleared LastError, got %q", synced.LastError)
	}

	// Verify via Get
	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("Get after sync: %v", err)
	}
	if got.SyncStatus != "syncing" {
		t.Fatalf("expected sync_status 'syncing' via Get, got %q", got.SyncStatus)
	}

	// Not found
	_, err = svc.TriggerSync(999)
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestCategoryMappings(t *testing.T) {
	svc := newTestDB(t)

	created, err := svc.Create(&CreateAccountInput{PlatformID: 1, AccessToken: "tok"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Create
	m, err := svc.CreateCategoryMapping(created.ID, &CreateCategoryMappingInput{
		LocalCategoryID:      5,
		PlatformCategoryID:   "plat-5",
		PlatformCategoryName: "Electronics",
	})
	if err != nil {
		t.Fatalf("CreateCategoryMapping: %v", err)
	}
	if m.AccountID != created.ID {
		t.Fatalf("expected account_id %d, got %d", created.ID, m.AccountID)
	}
	if m.LocalCategoryID != 5 {
		t.Fatalf("expected local_category_id 5, got %d", m.LocalCategoryID)
	}
	if m.PlatformCategoryID != "plat-5" {
		t.Fatalf("expected platform_category_id 'plat-5', got %q", m.PlatformCategoryID)
	}
	if m.PlatformCategoryName != "Electronics" {
		t.Fatalf("expected platform_category_name 'Electronics', got %q", m.PlatformCategoryName)
	}

	// List
	items, err := svc.ListCategoryMappings(created.ID)
	if err != nil {
		t.Fatalf("ListCategoryMappings: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].LocalCategoryID != 5 {
		t.Fatalf("expected local_category_id 5, got %d", items[0].LocalCategoryID)
	}

	// List for another account returns empty
	items2, err := svc.ListCategoryMappings(999)
	if err != nil {
		t.Fatalf("ListCategoryMappings: %v", err)
	}
	if len(items2) != 0 {
		t.Fatalf("expected 0 items for different account, got %d", len(items2))
	}
}

func TestAttributeMappings(t *testing.T) {
	svc := newTestDB(t)

	created, err := svc.Create(&CreateAccountInput{PlatformID: 1, AccessToken: "tok"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Create
	m, err := svc.CreateAttributeMapping(created.ID, &CreateAttributeMappingInput{
		LocalAttrName:    "color",
		PlatformAttrID:   "attr-color",
		PlatformAttrName: "Color",
		Required:         true,
	})
	if err != nil {
		t.Fatalf("CreateAttributeMapping: %v", err)
	}
	if m.AccountID != created.ID {
		t.Fatalf("expected account_id %d, got %d", created.ID, m.AccountID)
	}
	if m.LocalAttrName != "color" {
		t.Fatalf("expected local_attr_name 'color', got %q", m.LocalAttrName)
	}
	if m.PlatformAttrID != "attr-color" {
		t.Fatalf("expected platform_attr_id 'attr-color', got %q", m.PlatformAttrID)
	}
	if m.PlatformAttrName != "Color" {
		t.Fatalf("expected platform_attr_name 'Color', got %q", m.PlatformAttrName)
	}
	if !m.Required {
		t.Fatal("expected Required=true")
	}

	// List
	items, err := svc.ListAttributeMappings(created.ID)
	if err != nil {
		t.Fatalf("ListAttributeMappings: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].LocalAttrName != "color" {
		t.Fatalf("expected local_attr_name 'color', got %q", items[0].LocalAttrName)
	}

	// List for another account returns empty
	items2, err := svc.ListAttributeMappings(999)
	if err != nil {
		t.Fatalf("ListAttributeMappings: %v", err)
	}
	if len(items2) != 0 {
		t.Fatalf("expected 0 items for different account, got %d", len(items2))
	}
}

func TestCreateCategoryMapping_InvalidAccountID(t *testing.T) {
	svc := newTestDB(t)

	_, err := svc.CreateCategoryMapping(0, &CreateCategoryMappingInput{
		LocalCategoryID: 1,
	})
	if err == nil {
		t.Fatal("expected error for accountID=0")
	}
	if err.Error() != "account_id is required" {
		t.Fatalf("expected 'account_id is required', got %q", err.Error())
	}

	_, err = svc.CreateCategoryMapping(-1, &CreateCategoryMappingInput{
		LocalCategoryID: 1,
	})
	if err == nil {
		t.Fatal("expected error for accountID=-1")
	}
	if err.Error() != "account_id is required" {
		t.Fatalf("expected 'account_id is required', got %q", err.Error())
	}
}

func TestCreateAttributeMapping_InvalidAccountID(t *testing.T) {
	svc := newTestDB(t)

	_, err := svc.CreateAttributeMapping(0, &CreateAttributeMappingInput{
		LocalAttrName: "size",
	})
	if err == nil {
		t.Fatal("expected error for accountID=0")
	}
	if err.Error() != "account_id is required" {
		t.Fatalf("expected 'account_id is required', got %q", err.Error())
	}

	_, err = svc.CreateAttributeMapping(-1, &CreateAttributeMappingInput{
		LocalAttrName: "size",
	})
	if err == nil {
		t.Fatal("expected error for accountID=-1")
	}
	if err.Error() != "account_id is required" {
		t.Fatalf("expected 'account_id is required', got %q", err.Error())
	}
}

// ──────────────────────────────────────────────
//  Registry tests
// ──────────────────────────────────────────────

func TestRegisterAndGetAdapter(t *testing.T) {
	// Use a unique code so it doesn't collide with init() registrations
	code := "test-mock"
	RegisterAdapter(code, &mockAdapter{})

	adapter, ok := GetAdapter(code)
	if !ok {
		t.Fatalf("GetAdapter(%q) returned false", code)
	}
	if adapter == nil {
		t.Fatal("GetAdapter returned nil adapter")
	}

	// Verify it behaves like our mock
	pub, err := adapter.Publish(context.Background(), nil)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if pub != nil {
		t.Fatal("expected nil PublishResult")
	}
}

func TestListAdapters(t *testing.T) {
	// init() registers "ozon" and "shopee"
	adapters := ListAdapters()
	if len(adapters) < 2 {
		t.Fatalf("expected at least 2 registered adapters (ozon, shopee), got %d", len(adapters))
	}

	if _, ok := adapters["ozon"]; !ok {
		t.Fatal("expected 'ozon' adapter to be registered")
	}
	if _, ok := adapters["shopee"]; !ok {
		t.Fatal("expected 'shopee' adapter to be registered")
	}

	// Verify the adapters are actual PlatformAdapter implementations
	for code, adapter := range adapters {
		if adapter == nil {
			t.Fatalf("adapter %q is nil", code)
		}
	}

	// GetAdapter should work for built-in codes
	if _, ok := GetAdapter("ozon"); !ok {
		t.Fatal("expected GetAdapter('ozon') to succeed")
	}
	if _, ok := GetAdapter("SHOPEE"); !ok {
		t.Fatal("expected GetAdapter('SHOPEE') to succeed (case-insensitive)")
	}
	if _, ok := GetAdapter("unknown"); ok {
		t.Fatal("expected GetAdapter('unknown') to fail")
	}
}

// ──────────────────────────────────────────────
//  Handler tests
// ──────────────────────────────────────────────

func TestHandler_List(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	// Create two accounts first
	_, err := svc.Create(&CreateAccountInput{PlatformID: 1, StoreName: "Alpha", AccessToken: "a"})
	if err != nil {
		t.Fatalf("Create 1: %v", err)
	}
	_, err = svc.Create(&CreateAccountInput{PlatformID: 2, StoreName: "Beta", AccessToken: "b"})
	if err != nil {
		t.Fatalf("Create 2: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/platform-integrations", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	res := parsePageResult(t, w.Body.Bytes())
	if res.Code != 0 {
		t.Fatalf("expected code 0, got %d: %s", res.Code, res.Message)
	}
	if res.Total != 2 {
		t.Fatalf("expected total 2, got %d", res.Total)
	}
}

func TestHandler_Get(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	created, err := svc.Create(&CreateAccountInput{PlatformID: 1, StoreName: "Found Store", AccessToken: "tok"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/platform-integrations/"+dbtest.IToA(created.ID), nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		res := parseResult(t, w.Body.Bytes())
		if res.Code != 0 {
			t.Fatalf("expected code 0, got %d", res.Code)
		}
		data, ok := res.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected Data to be object, got %T", res.Data)
		}
		if data["store_name"] != "Found Store" {
			t.Fatalf("expected store_name 'Found Store', got %v", data["store_name"])
		}
		// access_token should not appear (json:"-")
		if _, exists := data["access_token"]; exists {
			t.Fatal("access_token should be hidden via json:\"-\"")
		}
	})

	t.Run("not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/platform-integrations/999", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
		res := parseResult(t, w.Body.Bytes())
		if res.Message != "integration account not found" {
			t.Fatalf("expected 'integration account not found', got %q", res.Message)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/platform-integrations/abc", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
		res := parseResult(t, w.Body.Bytes())
		if res.Message != "invalid id" {
			t.Fatalf("expected 'invalid id', got %q", res.Message)
		}
	})
}

func TestHandler_Create(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	body := `{"platform_id":1,"store_name":"New Store","access_token":"tok-new","config":{"key":"val"}}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/platform-integrations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	res := parseResult(t, w.Body.Bytes())
	if res.Code != 0 {
		t.Fatalf("expected code 0, got %d: %s", res.Code, res.Message)
	}
	data, ok := res.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected Data to be object, got %T", res.Data)
	}
	if data["store_name"] != "New Store" {
		t.Fatalf("expected store_name 'New Store', got %v", data["store_name"])
	}
	// Verify config was stored
	if _, exists := data["config"]; !exists {
		t.Fatal("expected config in response")
	}
}

func TestHandler_Update(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	created, err := svc.Create(&CreateAccountInput{PlatformID: 1, StoreName: "Old Name", AccessToken: "old-token"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Update store_name and access_token via handler
	body := `{"store_name":"New Name","access_token":"new-token"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/platform-integrations/"+dbtest.IToA(created.ID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	res := parseResult(t, w.Body.Bytes())
	if res.Code != 0 {
		t.Fatalf("expected code 0, got %d: %s", res.Code, res.Message)
	}
	data, ok := res.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected Data to be object, got %T", res.Data)
	}
	if data["store_name"] != "New Name" {
		t.Fatalf("expected store_name 'New Name', got %v", data["store_name"])
	}

	// Verify via service that token was updated
	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.AccessToken != "new-token" {
		t.Fatalf("expected access_token 'new-token', got %q", got.AccessToken)
	}

	// Update non-existent
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("PUT", "/api/v1/platform-integrations/999", strings.NewReader(`{"store_name":"Nope"}`))
	req2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestHandler_Delete(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	created, err := svc.Create(&CreateAccountInput{PlatformID: 1, StoreName: "Delete Handler", AccessToken: "tok"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/api/v1/platform-integrations/"+dbtest.IToA(created.ID), nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	res := parseResult(t, w.Body.Bytes())
	if res.Code != 0 {
		t.Fatalf("expected code 0, got %d: %s", res.Code, res.Message)
	}
	data, ok := res.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected Data to be object, got %T", res.Data)
	}
	if data["id"] != float64(created.ID) {
		t.Fatalf("expected id %d, got %v", created.ID, data["id"])
	}

	// Verify deleted
	if _, err := svc.Get(created.ID); err != gorm.ErrRecordNotFound {
		t.Fatal("expected account to be deleted")
	}
}

func TestHandler_TestConnection(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	// Account with valid token
	created, err := svc.Create(&CreateAccountInput{PlatformID: 1, AccessToken: "valid-token"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/platform-integrations/"+dbtest.IToA(created.ID)+"/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	res := parseResult(t, w.Body.Bytes())
	if res.Code != 0 {
		t.Fatalf("expected code 0, got %d", res.Code)
	}
	data, ok := res.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected Data to be object, got %T", res.Data)
	}
	if data["success"] != true {
		t.Fatalf("expected success=true, got %v", data["success"])
	}

	// Non-existent
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/api/v1/platform-integrations/999/test", nil)
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestHandler_Sync(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	created, err := svc.Create(&CreateAccountInput{PlatformID: 1, AccessToken: "tok"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/platform-integrations/"+dbtest.IToA(created.ID)+"/sync", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	res := parseResult(t, w.Body.Bytes())
	if res.Code != 0 {
		t.Fatalf("expected code 0, got %d", res.Code)
	}
	data, ok := res.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected Data to be object, got %T", res.Data)
	}
	if data["sync_status"] != "syncing" {
		t.Fatalf("expected sync_status 'syncing', got %v", data["sync_status"])
	}

	// Non-existent
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/api/v1/platform-integrations/999/sync", nil)
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestHandler_ListCategories(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	created, err := svc.Create(&CreateAccountInput{PlatformID: 1, AccessToken: "tok"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Create a category mapping
	_, err = svc.CreateCategoryMapping(created.ID, &CreateCategoryMappingInput{
		LocalCategoryID:      10,
		PlatformCategoryID:   "plat-10",
		PlatformCategoryName: "Clothing",
	})
	if err != nil {
		t.Fatalf("CreateCategoryMapping: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/platform-integrations/"+dbtest.IToA(created.ID)+"/categories", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	res := parseResult(t, w.Body.Bytes())
	if res.Code != 0 {
		t.Fatalf("expected code 0, got %d", res.Code)
	}
	items, ok := res.Data.([]interface{})
	if !ok {
		t.Fatalf("expected Data to be array, got %T", res.Data)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	item, ok := items[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected item to be object, got %T", items[0])
	}
	if item["platform_category_name"] != "Clothing" {
		t.Fatalf("expected 'Clothing', got %v", item["platform_category_name"])
	}
}

func TestHandler_CreateCategory(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	created, err := svc.Create(&CreateAccountInput{PlatformID: 1, AccessToken: "tok"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	body := `{"local_category_id":20,"platform_category_id":"plat-20","platform_category_name":"Shoes"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/platform-integrations/"+dbtest.IToA(created.ID)+"/categories", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	res := parseResult(t, w.Body.Bytes())
	if res.Code != 0 {
		t.Fatalf("expected code 0, got %d: %s", res.Code, res.Message)
	}
	data, ok := res.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected Data to be object, got %T", res.Data)
	}
	if data["local_category_id"] != float64(20) {
		t.Fatalf("expected local_category_id 20, got %v", data["local_category_id"])
	}
	if data["platform_category_name"] != "Shoes" {
		t.Fatalf("expected 'Shoes', got %v", data["platform_category_name"])
	}
}

func TestHandler_ListAttributes(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	created, err := svc.Create(&CreateAccountInput{PlatformID: 1, AccessToken: "tok"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Create an attribute mapping
	_, err = svc.CreateAttributeMapping(created.ID, &CreateAttributeMappingInput{
		LocalAttrName:    "size",
		PlatformAttrID:   "attr-size",
		PlatformAttrName: "Size",
	})
	if err != nil {
		t.Fatalf("CreateAttributeMapping: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/platform-integrations/"+dbtest.IToA(created.ID)+"/attributes", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	res := parseResult(t, w.Body.Bytes())
	if res.Code != 0 {
		t.Fatalf("expected code 0, got %d", res.Code)
	}
	items, ok := res.Data.([]interface{})
	if !ok {
		t.Fatalf("expected Data to be array, got %T", res.Data)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	item, ok := items[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected item to be object, got %T", items[0])
	}
	if item["local_attr_name"] != "size" {
		t.Fatalf("expected 'size', got %v", item["local_attr_name"])
	}
}

func TestHandler_CreateAttribute(t *testing.T) {
	svc := newTestDB(t)
	router := setupRouter(svc)

	created, err := svc.Create(&CreateAccountInput{PlatformID: 1, AccessToken: "tok"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	body := `{"local_attr_name":"color","platform_attr_id":"attr-color","platform_attr_name":"Color","required":true}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/platform-integrations/"+dbtest.IToA(created.ID)+"/attributes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	res := parseResult(t, w.Body.Bytes())
	if res.Code != 0 {
		t.Fatalf("expected code 0, got %d: %s", res.Code, res.Message)
	}
	data, ok := res.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected Data to be object, got %T", res.Data)
	}
	if data["local_attr_name"] != "color" {
		t.Fatalf("expected 'color', got %v", data["local_attr_name"])
	}
	if data["required"] != true {
		t.Fatalf("expected required=true, got %v", data["required"])
	}
}
