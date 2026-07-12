package category

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestHandlerUpdatePreservesFieldsMissingFromRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := dbtest.NewDB(t, &Category{})
	svc := NewService(db, dbtest.NewLogger(t))
	original := &Category{Name: "Original", ParentID: 9, Level: 2, SortOrder: 7, Status: 1}
	if err := svc.Create(t.Context(), original); err != nil {
		t.Fatal(err)
	}
	originalCreatedAt := original.CreatedAt

	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodPut, "/categories/1", strings.NewReader(`{"name":"Updated"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	h.Update(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	got, err := svc.GetByID(t.Context(), original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Updated" {
		t.Fatalf("name = %q, want Updated", got.Name)
	}
	if got.ParentID != 9 || got.Level != 2 || got.SortOrder != 7 || got.Status != 1 {
		t.Fatalf("fields were overwritten: %+v", got)
	}
	if got.CreatedAt.IsZero() || !got.CreatedAt.Equal(originalCreatedAt) {
		t.Fatalf("created_at changed: got %v, want %v", got.CreatedAt, originalCreatedAt)
	}
}

func TestHandlerUpdateAllowsExplicitZeroWithoutChangingCreatedAt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := dbtest.NewDB(t, &Category{})
	svc := NewService(db, dbtest.NewLogger(t))
	original := &Category{Name: "Original", ParentID: 9, Level: 2, SortOrder: 7, Status: 1}
	if err := svc.Create(t.Context(), original); err != nil {
		t.Fatal(err)
	}
	originalCreatedAt := original.CreatedAt

	h := NewHandler(svc)
	body := `{"parent_id":0,"level":0,"sort_order":0,"status":0,"created_at":"2000-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPut, "/categories/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	h.Update(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	got, err := svc.GetByID(t.Context(), original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ParentID != 0 || got.Level != 0 || got.SortOrder != 0 || got.Status != 0 {
		t.Fatalf("explicit zero values were not saved: %+v", got)
	}
	if !got.CreatedAt.Equal(originalCreatedAt) {
		t.Fatalf("created_at changed: got %v, want %v", got.CreatedAt, originalCreatedAt)
	}
}
