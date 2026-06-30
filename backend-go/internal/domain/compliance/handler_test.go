package compliance

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestHandler_Check(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CheckResult{})
	svc := NewService(db, dbtest.NewLogger(t))
	h := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/check", h.Check)

	body := `{
		"product_id": 1,
		"product_name": "Baby Toy",
		"category": "toys",
		"country": "US",
		"platform": "shopee"
	}`
	req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST /check returned %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data CheckResult `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.ID == 0 {
		t.Error("expected result ID after save")
	}
}

func TestHandler_ListResults(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CheckResult{})
	svc := NewService(db, dbtest.NewLogger(t))
	h := NewHandler(svc)

	// Seed a result.
	input := &CheckInput{ProductName: "X", Category: "electronics", Country: "US", Platform: "shopee"}
	r, _ := svc.CheckProduct(input, nil)
	r.ProductID = 1
	_ = svc.SaveResult(r)

	gin.SetMode(gin.TestMode)
	rr := gin.New()
	rr.GET("/results", h.ListResults)

	req := httptest.NewRequest(http.MethodGet, "/results?page=1&size=20", nil)
	w := httptest.NewRecorder()
	rr.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /results returned %d", w.Code)
	}
}

func TestHandler_SuppressResult(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CheckResult{})
	svc := NewService(db, dbtest.NewLogger(t))
	h := NewHandler(svc)

	// Seed a result.
	risk := RiskHigh
	orig := &CheckResult{
		ProductID: 42,
		CheckType: "compliance",
		Status:    StatusFail,
		RiskLevel: &risk,
	}
	// Save via raw DB to get an ID (CheckProduct would call A7 agent)
	if err := db.Create(orig).Error; err != nil {
		t.Fatalf("seed result: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/results/:id/suppress", h.SuppressResult)

	body := `{"reason": "false positive"}`
	req := httptest.NewRequest(http.MethodPut, "/results/"+itoa(orig.ID)+"/suppress", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT /results/:id/suppress returned %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_GetResult(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CheckResult{})
	svc := NewService(db, dbtest.NewLogger(t))
	h := NewHandler(svc)

	// Seed a result.
	risk := RiskLow
	orig := &CheckResult{
		ProductID: 99,
		CheckType: "compliance",
		Status:    StatusPass,
		RiskLevel: &risk,
	}
	if err := db.Create(orig).Error; err != nil {
		t.Fatalf("seed result: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/results/:id", h.GetResult)

	req := httptest.NewRequest(http.MethodGet, "/results/"+itoa(orig.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /results/:id returned %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_GetResult_NotFound(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CheckResult{})
	svc := NewService(db, dbtest.NewLogger(t))
	h := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/results/:id", h.GetResult)

	req := httptest.NewRequest(http.MethodGet, "/results/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_PaginationDefaults(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &CheckResult{})
	svc := NewService(db, dbtest.NewLogger(t))

	// Seed results.
	risk := RiskLow
	for i := 0; i < 5; i++ {
		r := &CheckResult{
			ProductID: int64(200 + i),
			CheckType: "compliance",
			Status:    StatusPass,
			RiskLevel: &risk,
		}
		if err := db.Create(r).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	// Verify the service pagination works under the hood (matches what ListResults handler calls).
	p := &common.Pagination{Page: 1, Size: 2}
	items, total, err := svc.ListResults(p, "", "", 0)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total=5, got %d", total)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

// itoa is a helper for the test file (mirrors the one in middleware/audit.go).
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
