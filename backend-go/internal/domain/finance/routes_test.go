package finance

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/domain/order"
	"github.com/lingmirror/backend-go/internal/integrationtest"
	"github.com/lingmirror/backend-go/internal/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// mustJSON marshals v to a JSON string, panicking on error.
func mustJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestFinanceRoutesExcludeMockMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), dbtest.NewDB(t), zap.NewNop())

	for _, route := range router.Routes() {
		if route.Method == http.MethodPost && route.Path == "/api/v1/finance/mock" {
			t.Fatal("formal finance routes must not expose the mock ledger mutation")
		}
	}
}

func TestFinanceRoutes_Unauthenticated(t *testing.T) {
	ts := integrationtest.NewTestServer(t,
		func(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
			RegisterRoutes(rg, db, logger)
		},
		&FinanceAccount{}, &FinanceTransaction{}, &FinanceLedgerEntry{}, &order.Order{}, &order.OrderItem{},
	)
	defer ts.Close()

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"ListAccounts", http.MethodGet, "/api/v1/finance/accounts"},
		{"CreateAccount", http.MethodPost, "/api/v1/finance/accounts"},
		{"GetAccount", http.MethodGet, "/api/v1/finance/accounts/1"},
		{"UpdateAccount", http.MethodPut, "/api/v1/finance/accounts/1"},
		{"DeleteAccount", http.MethodDelete, "/api/v1/finance/accounts/1"},
		{"Summary", http.MethodGet, "/api/v1/finance/summary"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *http.Response
			switch tt.method {
			case http.MethodGet:
				resp = ts.Get(t, tt.path, "")
			case http.MethodPost:
				resp = ts.Post(t, tt.path, "", "")
			case http.MethodPut:
				resp = ts.Put(t, tt.path, "", "")
			case http.MethodDelete:
				resp = ts.Delete(t, tt.path, "")
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", resp.StatusCode)
			}

			var result response.Result
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if result.Code != 401 {
				t.Fatalf("code = %d, want 401", result.Code)
			}
		})
	}
}

func TestFinanceRoutes_AccountCRUD(t *testing.T) {
	ts := integrationtest.NewTestServer(t,
		func(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
			RegisterRoutes(rg, db, logger)
		},
		&FinanceAccount{}, &FinanceTransaction{}, &FinanceLedgerEntry{}, &order.Order{}, &order.OrderItem{},
	)
	defer ts.Close()
	token := ts.Login(t)

	// 1. Create account
	createBody := map[string]interface{}{
		"name":         "测试银行账户",
		"account_type": "bank",
		"currency":     "CNY",
		"balance":      10000.00,
	}

	resp := ts.Post(t, "/api/v1/finance/accounts", mustJSON(createBody), token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Create status = %d, want 200", resp.StatusCode)
	}
	var createResult struct {
		response.Result
		Data FinanceAccount `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&createResult); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if createResult.Code != 0 {
		t.Fatalf("code = %d, want 0", createResult.Code)
	}
	accountID := createResult.Data.ID
	if accountID == 0 {
		t.Fatal("account ID should not be 0")
	}
	if createResult.Data.Name != "测试银行账户" {
		t.Fatalf("name = %s", createResult.Data.Name)
	}

	// 2. Get account by ID
	resp = ts.Get(t, "/api/v1/finance/accounts/"+dbtest.IToA(accountID), token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Get status = %d, want 200", resp.StatusCode)
	}
	var getResult struct {
		response.Result
		Data FinanceAccount `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&getResult); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if getResult.Data.ID != accountID {
		t.Fatalf("id = %d, want %d", getResult.Data.ID, accountID)
	}

	// 3. List accounts
	resp = ts.Get(t, "/api/v1/finance/accounts", token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("List status = %d, want 200", resp.StatusCode)
	}
	var listResult struct {
		response.PageResult
		Data []FinanceAccount `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResult); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if listResult.Total < 1 {
		t.Fatalf("total = %d, want at least 1", listResult.Total)
	}

	// 4. Update account
	newName := "更新银行账户"
	updateBody := map[string]interface{}{
		"name": newName,
	}
	resp = ts.Put(t, "/api/v1/finance/accounts/"+dbtest.IToA(accountID), mustJSON(updateBody), token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Update status = %d, want 200", resp.StatusCode)
	}
	var updateResult struct {
		response.Result
		Data FinanceAccount `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&updateResult); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if updateResult.Data.Name != "更新银行账户" {
		t.Fatalf("name = %s, want 更新银行账户", updateResult.Data.Name)
	}

	// 5. Delete account
	resp = ts.Delete(t, "/api/v1/finance/accounts/"+dbtest.IToA(accountID), token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Delete status = %d, want 200", resp.StatusCode)
	}

	// 6. Verify deleted
	resp = ts.Get(t, "/api/v1/finance/accounts/"+dbtest.IToA(accountID), token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("GET after delete status = %d, want 404 or 500", resp.StatusCode)
	}
}

func TestFinanceRoutes_CreateAccount_InvalidInput(t *testing.T) {
	ts := integrationtest.NewTestServer(t,
		func(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
			RegisterRoutes(rg, db, logger)
		},
		&FinanceAccount{}, &FinanceTransaction{}, &FinanceLedgerEntry{}, &order.Order{}, &order.OrderItem{},
	)
	defer ts.Close()
	token := ts.Login(t)

	// Missing required name and account_type
	body := map[string]interface{}{}

	resp := ts.Post(t, "/api/v1/finance/accounts", mustJSON(body), token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}

	var result response.Result
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Code != 400 {
		t.Fatalf("code = %d, want 400", result.Code)
	}
}

func TestFinanceRoutes_GetAccount_NotFound(t *testing.T) {
	ts := integrationtest.NewTestServer(t,
		func(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
			RegisterRoutes(rg, db, logger)
		},
		&FinanceAccount{}, &FinanceTransaction{}, &FinanceLedgerEntry{}, &order.Order{}, &order.OrderItem{},
	)
	defer ts.Close()
	token := ts.Login(t)

	resp := ts.Get(t, "/api/v1/finance/accounts/99999", token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}

	var result response.Result
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.Contains(result.Message, "not found") {
		t.Fatalf("message = %s, want 'not found'", result.Message)
	}
}
