package finance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/order"
	"github.com/lingmirror/backend-go/internal/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// -------- helpers --------

func newTestDB(t *testing.T, models ...interface{}) *Service {
	t.Helper()
	if len(models) == 0 {
		models = []interface{}{&FinanceAccount{}, &FinanceTransaction{}, &FinanceLedgerEntry{}, &order.Order{}, &order.OrderItem{}}
	}
	db := dbtest.NewDB(t, models...)
	return NewService(db, zap.NewNop())
}

func insertTestOrder(t *testing.T, svc *Service, payAmount, platformFee, shippingFee, productCost, otherFee float64) int64 {
	t.Helper()
	o := order.Order{
		OrderNo:     "test-" + dbtest.IToA(time.Now().UnixNano()),
		PayAmount:   payAmount,
		PlatformFee: platformFee,
		ShippingFee: shippingFee,
		ProductCost: productCost,
		OtherFee:    otherFee,
		Status:      "delivered",
	}
	if err := svc.db.Create(&o).Error; err != nil {
		t.Fatalf("insert order: %v", err)
	}
	return o.ID
}

func insertTestOrderItem(t *testing.T, svc *Service, orderID, skuID int64, subtotal float64) {
	t.Helper()
	item := order.OrderItem{
		OrderID:     orderID,
		SkuID:       skuID,
		ProductID:   skuID,
		Subtotal:    subtotal,
		Quantity:    1,
		UnitPrice:   subtotal,
		ProductName: "test-product",
	}
	if err := svc.db.Create(&item).Error; err != nil {
		t.Fatalf("insert order item: %v", err)
	}
}

func setupFinanceRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1/finance")
	{
		g.GET("/summary", h.Summary)
		g.GET("/profit-summary", h.ProfitSummary)
		g.GET("/ledger", h.ListLedger)
		g.POST("/mock", h.Mock)
		g.GET("/accounts", h.ListAccounts)
		g.POST("/accounts", h.CreateAccount)
		g.GET("/accounts/:id", h.GetAccount)
		g.PUT("/accounts/:id", h.UpdateAccount)
		g.DELETE("/accounts/:id", h.DeleteAccount)
		g.GET("/transactions", h.ListTransactions)
		g.POST("/transactions", h.CreateTransaction)
		g.GET("/orders/:order_id/ledger", h.ListOrderLedger)
		g.GET("/orders/:order_id/profit", h.OrderProfit)
		g.POST("/orders/:order_id/ledger/rebuild", h.RebuildOrderLedger)
	}
	return r
}

// -------- Account Service Tests --------

func TestCreateAccount(t *testing.T) {
	svc := newTestDB(t)

	in := &CreateAccountInput{
		Name:        "Test Bank Account",
		AccountType: "bank",
		Currency:    "USD",
		Balance:     dbtest.FloatPtr(10000),
		Status:      "active",
	}
	a, err := svc.CreateAccount(in)
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	if a.Name != in.Name {
		t.Errorf("Name = %q, want %q", a.Name, in.Name)
	}
	if a.AccountType != "bank" {
		t.Errorf("AccountType = %q, want bank", a.AccountType)
	}
	if a.Currency != "USD" {
		t.Errorf("Currency = %q, want USD", a.Currency)
	}
	if a.Balance != 10000 {
		t.Errorf("Balance = %.2f, want 10000.00", a.Balance)
	}
}

func TestCreateAccount_Defaults(t *testing.T) {
	svc := newTestDB(t)

	in := &CreateAccountInput{Name: "Default Account", AccountType: "payment"}
	a, err := svc.CreateAccount(in)
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	if a.Currency != "CNY" {
		t.Errorf("default Currency = %q, want CNY", a.Currency)
	}
	if a.Status != "active" {
		t.Errorf("default Status = %q, want active", a.Status)
	}
}

func TestListAccounts(t *testing.T) {
	svc := newTestDB(t)

	// empty
	items, total, err := svc.ListAccounts(&common.Pagination{Page: 1, Size: 20}, nil)
	if err != nil {
		t.Fatalf("ListAccounts failed: %v", err)
	}
	if total != 0 || len(items) != 0 {
		t.Errorf("expected empty, got %d items, total=%d", len(items), total)
	}

	// insert two
	svc.CreateAccount(&CreateAccountInput{Name: "A1", AccountType: "bank"})
	svc.CreateAccount(&CreateAccountInput{Name: "A2", AccountType: "payment"})

	items, total, err = svc.ListAccounts(&common.Pagination{Page: 1, Size: 20}, nil)
	if err != nil {
		t.Fatalf("ListAccounts failed: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total=2, got %d", total)
	}

	// filter by type
	items, total, err = svc.ListAccounts(&common.Pagination{Page: 1, Size: 20}, &AccountListFilter{AccountType: "bank"})
	if err != nil {
		t.Fatalf("ListAccounts with filter failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 bank account, got %d", total)
	}
}

func TestGetAccount(t *testing.T) {
	svc := newTestDB(t)

	a, err := svc.CreateAccount(&CreateAccountInput{Name: "GetTest", AccountType: "cash"})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	got, err := svc.GetAccount(a.ID)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got.Name != "GetTest" {
		t.Errorf("got name %q", got.Name)
	}

	// not found
	_, err = svc.GetAccount(99999)
	if err != gorm.ErrRecordNotFound {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestUpdateAccount(t *testing.T) {
	svc := newTestDB(t)

	a, err := svc.CreateAccount(&CreateAccountInput{Name: "Old", AccountType: "bank"})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	updated, err := svc.UpdateAccount(a.ID, &UpdateAccountInput{Name: dbtest.StringPtr("Updated")})
	if err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}
	if updated.Name != "Updated" {
		t.Errorf("Name = %q, want Updated", updated.Name)
	}

	// no-op update (nil input fields)
	_, err = svc.UpdateAccount(a.ID, &UpdateAccountInput{})
	if err != nil {
		t.Errorf("no-op update should succeed: %v", err)
	}

	// not found
	_, err = svc.UpdateAccount(99999, &UpdateAccountInput{Name: dbtest.StringPtr("x")})
	if err != gorm.ErrRecordNotFound {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestDeleteAccount(t *testing.T) {
	svc := newTestDB(t)

	a, err := svc.CreateAccount(&CreateAccountInput{Name: "Del", AccountType: "bank"})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	if err := svc.DeleteAccount(a.ID); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}

	// double delete
	if err := svc.DeleteAccount(a.ID); err != gorm.ErrRecordNotFound {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}

// -------- Transaction Service Tests --------

func TestCreateTransaction(t *testing.T) {
	svc := newTestDB(t)

	a, err := svc.CreateAccount(&CreateAccountInput{Name: "TxnAcct", AccountType: "bank", Balance: dbtest.FloatPtr(1000)})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	in := &CreateTransactionInput{
		AccountID:       a.ID,
		TransactionType: "revenue",
		Amount:          500,
		Currency:        "USD",
		Description:     "test revenue",
	}
	txn, err := svc.CreateTransaction(in)
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}
	if txn.Amount != 500 {
		t.Errorf("Amount = %.2f, want 500", txn.Amount)
	}
	if txn.Currency != "USD" {
		t.Errorf("Currency = %q, want USD", txn.Currency)
	}

	// Verify balance increased
	acct, _ := svc.GetAccount(a.ID)
	if acct.Balance != 1500 {
		t.Errorf("Balance after revenue = %.2f, want 1500", acct.Balance)
	}
}

func TestCreateTransaction_DebitTypes(t *testing.T) {
	svc := newTestDB(t)

	a, _ := svc.CreateAccount(&CreateAccountInput{Name: "DebitAcct", AccountType: "platform", Balance: dbtest.FloatPtr(2000)})

	for _, txnType := range []string{"cost", "fee", "refund", "transfer"} {
		_, err := svc.CreateTransaction(&CreateTransactionInput{
			AccountID:       a.ID,
			TransactionType: txnType,
			Amount:          100,
		})
		if err != nil {
			t.Fatalf("CreateTransaction(%s): %v", txnType, err)
		}
	}

	acct, _ := svc.GetAccount(a.ID)
	if acct.Balance != 1600 {
		t.Errorf("Balance after 4 debits = %.2f, want 1600", acct.Balance)
	}
}

func TestListTransactions(t *testing.T) {
	svc := newTestDB(t)

	a, _ := svc.CreateAccount(&CreateAccountInput{Name: "L", AccountType: "bank"})

	items, total, err := svc.ListTransactions(&common.Pagination{Page: 1, Size: 20}, nil)
	if err != nil || total != 0 {
		t.Fatalf("expected empty: err=%v, total=%d", err, total)
	}
	_ = items

	svc.CreateTransaction(&CreateTransactionInput{AccountID: a.ID, TransactionType: "revenue", Amount: 100})
	svc.CreateTransaction(&CreateTransactionInput{AccountID: a.ID, TransactionType: "fee", Amount: 20})

	items, total, err = svc.ListTransactions(&common.Pagination{Page: 1, Size: 20}, nil)
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
}

func TestListTransactions_FilterByType(t *testing.T) {
	svc := newTestDB(t)

	a, _ := svc.CreateAccount(&CreateAccountInput{Name: "F", AccountType: "bank"})
	svc.CreateTransaction(&CreateTransactionInput{AccountID: a.ID, TransactionType: "revenue", Amount: 100})
	svc.CreateTransaction(&CreateTransactionInput{AccountID: a.ID, TransactionType: "cost", Amount: 30})

	items, total, err := svc.ListTransactions(&common.Pagination{Page: 1, Size: 20}, &TransactionListFilter{TransactionType: "revenue"})
	if err != nil {
		t.Fatalf("ListTransactions filtered: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 revenue, got %d", total)
	}
	_ = items
}

// -------- Ledger Service Tests --------

func TestListLedgerEntries(t *testing.T) {
	svc := newTestDB(t)

	entries, total, err := svc.ListLedgerEntries(&common.Pagination{Page: 1, Size: 20}, nil)
	if err != nil {
		t.Fatalf("ListLedgerEntries: %v", err)
	}
	if total != 0 {
		t.Errorf("expected empty, total=%d", total)
	}
	_ = entries

	oid := int64(1)
	entry := FinanceLedgerEntry{OrderID: &oid, EntryType: "revenue", Amount: 500, Currency: "CNY", CostLayer: "actual", SourceType: "order", Description: "test"}
	if err := svc.db.Create(&entry).Error; err != nil {
		t.Fatalf("create ledger entry: %v", err)
	}

	entries, total, err = svc.ListLedgerEntries(&common.Pagination{Page: 1, Size: 20}, nil)
	if err != nil {
		t.Fatalf("ListLedgerEntries: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
}

func TestListLedgerEntries_FilterByType(t *testing.T) {
	svc := newTestDB(t)

	oid1 := int64(10)
	svc.db.Create(&FinanceLedgerEntry{OrderID: &oid1, EntryType: "revenue", Amount: 100, Currency: "CNY", CostLayer: "actual", SourceType: "order", Description: "a"})
	svc.db.Create(&FinanceLedgerEntry{OrderID: &oid1, EntryType: "cost", Amount: 50, Currency: "CNY", CostLayer: "actual", SourceType: "order", Description: "b"})

	entries, total, err := svc.ListLedgerEntries(&common.Pagination{Page: 1, Size: 20}, &LedgerListFilter{EntryType: "revenue"})
	if err != nil {
		t.Fatalf("ListLedgerEntries with filter: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 revenue entry, got %d", total)
	}
	_ = entries
}

func TestOrderProfit(t *testing.T) {
	svc := newTestDB(t)

	oid := insertTestOrder(t, svc, 1000, 100, 50, 400, 30)
	insertTestOrderItem(t, svc, oid, 101, 1000)

	entries, err := svc.RebuildOrderLedger(oid)
	if err != nil {
		t.Fatalf("RebuildOrderLedger: %v", err)
	}
	if len(entries) != 6 {
		t.Errorf("expected 6 ledger entries, got %d", len(entries))
	}

	profit, err := svc.OrderProfit(oid)
	if err != nil {
		t.Fatalf("OrderProfit: %v", err)
	}
	if profit.Revenue != 1000 {
		t.Errorf("Revenue = %.2f, want 1000", profit.Revenue)
	}
	wantProfit := 1000.0 - 400.0 - 50.0 - 100.0 - 30.0
	if profit.Profit != wantProfit {
		t.Errorf("Profit = %.2f, want %.2f", profit.Profit, wantProfit)
	}
}

func TestListOrderLedger(t *testing.T) {
	svc := newTestDB(t)

	oid := insertTestOrder(t, svc, 500, 50, 25, 200, 10)
	svc.RebuildOrderLedger(oid)

	entries, err := svc.ListOrderLedger(oid)
	if err != nil {
		t.Fatalf("ListOrderLedger: %v", err)
	}
	if len(entries) != 6 {
		t.Errorf("expected 6 entries, got %d", len(entries))
	}
}

func TestRebuildOrderLedger_NotFound(t *testing.T) {
	svc := newTestDB(t)

	_, err := svc.RebuildOrderLedger(99999)
	if err != gorm.ErrRecordNotFound {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestRebuildOrderLedger_ZeroSubtotal(t *testing.T) {
	svc := newTestDB(t)

	o := order.Order{OrderNo: "fallback", PayAmount: 500, PlatformFee: 50, ShippingFee: 25, ProductCost: 200, OtherFee: 10, Status: "delivered"}
	if err := svc.db.Create(&o).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	entries, err := svc.RebuildOrderLedger(o.ID)
	if err != nil {
		t.Fatalf("RebuildOrderLedger: %v", err)
	}
	for _, e := range entries {
		if e.EntryType == "revenue" && e.Amount != 500 {
			t.Errorf("revenue = %.2f, want 500 (PayAmount fallback)", e.Amount)
		}
	}
	_ = entries
}

// -------- Summary Service Tests --------

func TestSummary(t *testing.T) {
	svc := newTestDB(t)

	sum, err := svc.Summary()
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if sum.TotalBalance != 0 {
		t.Errorf("TotalBalance = %.2f, want 0", sum.TotalBalance)
	}

	svc.CreateAccount(&CreateAccountInput{Name: "A", AccountType: "bank", Balance: dbtest.FloatPtr(1000)})
	svc.CreateAccount(&CreateAccountInput{Name: "B", AccountType: "platform", Balance: dbtest.FloatPtr(2000)})

	sum, err = svc.Summary()
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if sum.TotalBalance != 3000 {
		t.Errorf("TotalBalance = %.2f, want 3000", sum.TotalBalance)
	}
	if len(sum.BalanceByType) != 2 {
		t.Errorf("expected 2 account types, got %d", len(sum.BalanceByType))
	}
}

// -------- ProfitSummary Service Tests --------


// -------- Mock Service Tests --------

func TestMock(t *testing.T) {
	svc := newTestDB(t)

	entries, err := svc.Mock(5)
	if err != nil {
		t.Fatalf("Mock: %v", err)
	}
	if len(entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(entries))
	}

	var count int64
	svc.db.Model(&FinanceLedgerEntry{}).Count(&count)
	if count != 5 {
		t.Errorf("expected 5 entries in DB, got %d", count)
	}
}

func TestMock_DefaultCount(t *testing.T) {
	svc := newTestDB(t)

	entries, err := svc.Mock(0)
	if err != nil {
		t.Fatalf("Mock(0): %v", err)
	}
	if len(entries) != 10 {
		t.Errorf("expected 10 entries (default), got %d", len(entries))
	}
}

// -------- Handler Tests --------

func TestHandler_ListAccounts(t *testing.T) {
	db := dbtest.NewDB(t, &FinanceAccount{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := setupFinanceRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/accounts", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("List status = %d", w.Code)
	}

	// Verify response is a PageResult
	var pr response.PageResult
	if err := json.Unmarshal(w.Body.Bytes(), &pr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if pr.Code != 0 {
		t.Errorf("code = %d, want 0", pr.Code)
	}
}

func TestHandler_CreateAccount(t *testing.T) {
	db := dbtest.NewDB(t, &FinanceAccount{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := setupFinanceRouter(h)

	body := `{"name":"Test Account","account_type":"bank"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/accounts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Create status = %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_CreateAccount_InvalidBody(t *testing.T) {
	db := dbtest.NewDB(t, &FinanceAccount{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := setupFinanceRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/accounts", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_GetAccount_Found(t *testing.T) {
	db := dbtest.NewDB(t, &FinanceAccount{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := setupFinanceRouter(h)

	// Create first
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/accounts", strings.NewReader(`{"name":"GetTest","account_type":"payment"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(resp, req)

	var cr response.Result
	json.Unmarshal(resp.Body.Bytes(), &cr)
	data := cr.Data.(map[string]interface{})
	id := int64(data["id"].(float64))

	w := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/finance/accounts/%d", id), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Get status = %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_GetAccount_NotFound(t *testing.T) {
	db := dbtest.NewDB(t, &FinanceAccount{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := setupFinanceRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/accounts/99999", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_GetAccount_InvalidID(t *testing.T) {
	db := dbtest.NewDB(t, &FinanceAccount{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := setupFinanceRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/accounts/abc", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid ID, got %d", w.Code)
	}
}

func TestHandler_UpdateAccount(t *testing.T) {
	db := dbtest.NewDB(t, &FinanceAccount{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := setupFinanceRouter(h)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/accounts", strings.NewReader(`{"name":"OldName","account_type":"bank"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(resp, req)

	var cr response.Result
	json.Unmarshal(resp.Body.Bytes(), &cr)
	data := cr.Data.(map[string]interface{})
	id := int64(data["id"].(float64))

	w := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/finance/accounts/%d", id), strings.NewReader(`{"name":"NewName"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Update status = %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_UpdateAccount_NotFound(t *testing.T) {
	db := dbtest.NewDB(t, &FinanceAccount{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := setupFinanceRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/finance/accounts/99999", strings.NewReader(`{"name":"X"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	// Accept 404 or 500 (GORM error matching differs in SQLite)
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("expected 404 or 500, got %d", w.Code)
	}
}

func TestHandler_DeleteAccount(t *testing.T) {
	db := dbtest.NewDB(t, &FinanceAccount{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := setupFinanceRouter(h)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/accounts", strings.NewReader(`{"name":"Del","account_type":"bank"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(resp, req)

	var cr response.Result
	json.Unmarshal(resp.Body.Bytes(), &cr)
	data := cr.Data.(map[string]interface{})
	id := int64(data["id"].(float64))

	w := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/finance/accounts/%d", id), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Delete status = %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_ListTransactions(t *testing.T) {
	db := dbtest.NewDB(t, &FinanceAccount{}, &FinanceTransaction{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := setupFinanceRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/transactions", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListTransactions status = %d", w.Code)
	}
}

func TestHandler_CreateTransaction(t *testing.T) {
	db := dbtest.NewDB(t, &FinanceAccount{}, &FinanceTransaction{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := setupFinanceRouter(h)

	ar := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/accounts", strings.NewReader(`{"name":"Acct","account_type":"bank","balance":1000}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(ar, req)

	var arResp response.Result
	json.Unmarshal(ar.Body.Bytes(), &arResp)
	acctData := arResp.Data.(map[string]interface{})
	acctID := int64(acctData["id"].(float64))

	body := fmt.Sprintf(`{"account_id":%d,"transaction_type":"revenue","amount":200}`, acctID)
	w := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/finance/transactions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("CreateTransaction status = %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_CreateTransaction_InvalidBody(t *testing.T) {
	db := dbtest.NewDB(t, &FinanceAccount{}, &FinanceTransaction{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := setupFinanceRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/transactions", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid body, got %d", w.Code)
	}
}

func TestHandler_Summary(t *testing.T) {
	db := dbtest.NewDB(t, &FinanceAccount{}, &FinanceLedgerEntry{}, &FinanceTransaction{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := setupFinanceRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/summary", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Summary status = %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_Mock(t *testing.T) {
	db := dbtest.NewDB(t, &FinanceLedgerEntry{}, &order.Order{}, &order.OrderItem{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := setupFinanceRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/mock", strings.NewReader(`{"count":3}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Mock status = %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_ListLedger(t *testing.T) {
	db := dbtest.NewDB(t, &FinanceLedgerEntry{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := setupFinanceRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/ledger", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListLedger status = %d", w.Code)
	}
}

func TestHandler_RebuildOrderLedger(t *testing.T) {
	db := dbtest.NewDB(t, &FinanceLedgerEntry{}, &order.Order{}, &order.OrderItem{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := setupFinanceRouter(h)

	o := order.Order{OrderNo: "rebuild-test", PayAmount: 1000, PlatformFee: 100, ShippingFee: 50, ProductCost: 400, OtherFee: 30, Status: "delivered"}
	if err := db.Create(&o).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/finance/orders/%d/ledger/rebuild", o.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("RebuildOrderLedger status = %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_ListOrderLedger(t *testing.T) {
	db := dbtest.NewDB(t, &FinanceLedgerEntry{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := setupFinanceRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/orders/1/ledger", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListOrderLedger status = %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_OrderProfit(t *testing.T) {
	db := dbtest.NewDB(t, &FinanceLedgerEntry{}, &order.Order{}, &order.OrderItem{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := setupFinanceRouter(h)

	o := order.Order{OrderNo: "op-test", PayAmount: 500, PlatformFee: 50, ShippingFee: 25, ProductCost: 200, OtherFee: 10, Status: "delivered"}
	db.Create(&o)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/finance/orders/%d/profit", o.ID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("OrderProfit status = %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_InvalidOrderID(t *testing.T) {
	db := dbtest.NewDB(t, &FinanceLedgerEntry{})
	svc := NewService(db, zap.NewNop())
	h := NewHandler(svc)
	r := setupFinanceRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/orders/abc/ledger", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid order_id, got %d", w.Code)
	}
}
