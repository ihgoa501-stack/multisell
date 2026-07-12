package productimage

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

func TestGetTaskUsesJWTContextAndHidesAnotherOwnersTask(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Asset{}, &Task{}, &Review{})
	task := Task{OwnerID: 41, AssetID: 1, IdempotencyKey: "owner-41-task", ManifestHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Operation: "DETERMINISTIC_RESIZE", Width: 100, Height: 100, Format: "png", Status: "queued"}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	h := NewHandler(NewService(db, dbtest.NewLogger(t), newFakeImageService()))
	r := gin.New()
	r.GET("/tasks/:id", func(c *gin.Context) { c.Set("user_id", int64(42)); h.GetTask(c) })
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tasks/1", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got == "" || !strings.Contains(got, `"error_code":"NOT_FOUND"`) {
		t.Fatalf("body=%s", got)
	}
}

func TestGetTaskRequiresOwnerIdentity(t *testing.T) {
	t.Parallel()
	db := dbtest.NewDB(t, &Asset{}, &Task{}, &Review{})
	h := NewHandler(NewService(db, dbtest.NewLogger(t), newFakeImageService()))
	r := gin.New()
	r.GET("/tasks/:id", h.GetTask)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/tasks/1", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
