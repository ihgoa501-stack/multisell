package order

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
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

func TestOrderRoutes_Unauthenticated(t *testing.T) {
	ts := integrationtest.NewTestServer(t, func(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
		RegisterRoutes(rg, db, logger, nil)
	}, &Order{}, &OrderItem{}, &OrderStatusLog{})
	defer ts.Close()

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"List", http.MethodGet, "/api/v1/order"},
		{"Create", http.MethodPost, "/api/v1/order"},
		{"Get", http.MethodGet, "/api/v1/order/1"},
		{"Update", http.MethodPut, "/api/v1/order/1"},
		{"Delete", http.MethodDelete, "/api/v1/order/1"},
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
			if result.Message == "" {
				t.Fatal("message should not be empty")
			}
		})
	}
}

func TestOrderRoutes_Create_Get_Update_Delete(t *testing.T) {
	ts := integrationtest.NewTestServer(t, func(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
		RegisterRoutes(rg, db, logger, nil)
	}, &Order{}, &OrderItem{}, &OrderStatusLog{})
	defer ts.Close()
	token := ts.Login(t)

	// 1. Create
	createBody := map[string]interface{}{
		"order_no":        "INTG-ORD-001",
		"status":          "pending",
		"total_amount":    199.99,
		"pay_amount":      199.99,
		"recipient_name":  "张三",
		"shipping_address": "北京市朝阳区",
		"items": []map[string]interface{}{
			{"sku_id": 1, "product_id": 1, "product_name": "商品A", "unit_price": 99.995, "quantity": 2},
		},
	}

	resp := ts.Post(t, "/api/v1/order", mustJSON(createBody), token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Create status = %d, want 200", resp.StatusCode)
	}
	var createResult struct {
		response.Result
		Data OrderResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&createResult); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if createResult.Code != 0 {
		t.Fatalf("code = %d, want 0", createResult.Code)
	}
	orderID := createResult.Data.ID
	if orderID == 0 {
		t.Fatal("order ID should not be 0")
	}
	if createResult.Data.OrderNo != "INTG-ORD-001" {
		t.Fatalf("order_no = %s", createResult.Data.OrderNo)
	}

	// 2. Get by ID
	resp = ts.Get(t, "/api/v1/order/"+itoa(orderID), token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Get status = %d, want 200", resp.StatusCode)
	}
	var getResult struct {
		response.Result
		Data OrderDetailResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&getResult); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if getResult.Data.Order.ID != orderID {
		t.Fatalf("got order ID %d, want %d", getResult.Data.Order.ID, orderID)
	}
	if len(getResult.Data.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(getResult.Data.Items))
	}

	// 3. List
	resp = ts.Get(t, "/api/v1/order", token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("List status = %d, want 200", resp.StatusCode)
	}
	var listResult struct {
		response.PageResult
		Data []OrderResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResult); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if listResult.Total < 1 {
		t.Fatalf("total = %d, want at least 1", listResult.Total)
	}

	// 4. Update to confirmed (state machine: pending → confirmed)
	resp = ts.Put(t, "/api/v1/order/"+itoa(orderID), mustJSON(map[string]interface{}{"status": "confirmed"}), token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Update to confirmed status = %d, want 200", resp.StatusCode)
	}

	// 5. Update to shipped (state machine: confirmed → shipped)
	newStatus := "shipped"
	updateBody := map[string]interface{}{
		"status": newStatus,
	}
	resp = ts.Put(t, "/api/v1/order/"+itoa(orderID), mustJSON(updateBody), token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Update to shipped status = %d, want 200", resp.StatusCode)
	}
	var updateResult struct {
		response.Result
		Data OrderResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&updateResult); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if updateResult.Data.Status != "shipped" {
		t.Fatalf("status = %s, want shipped", updateResult.Data.Status)
	}

	// 6. Delete
	resp = ts.Delete(t, "/api/v1/order/"+itoa(orderID), token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Delete status = %d, want 200", resp.StatusCode)
	}

	// 7. Verify deleted (GET returns 500 with gorm.ErrRecordNotFound)
	resp = ts.Get(t, "/api/v1/order/"+itoa(orderID), token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET after delete status = %d, want 500 or 404", resp.StatusCode)
	}
}

func TestOrderRoutes_Create_InvalidInput(t *testing.T) {
	ts := integrationtest.NewTestServer(t, func(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
		RegisterRoutes(rg, db, logger, nil)
	}, &Order{}, &OrderItem{}, &OrderStatusLog{})
	defer ts.Close()
	token := ts.Login(t)

	// Missing required order_no — binding validation fails
	body := map[string]interface{}{
		"status": "pending",
	}

	resp := ts.Post(t, "/api/v1/order", mustJSON(body), token)
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
	if result.Message == "" {
		t.Fatal("message should not be empty")
	}
}

func TestOrderRoutes_Get_NotFound(t *testing.T) {
	ts := integrationtest.NewTestServer(t, func(rg *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
		RegisterRoutes(rg, db, logger, nil)
	}, &Order{}, &OrderItem{}, &OrderStatusLog{})
	defer ts.Close()
	token := ts.Login(t)

	resp := ts.Get(t, "/api/v1/order/99999", token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 404 or 500", resp.StatusCode)
	}
}
