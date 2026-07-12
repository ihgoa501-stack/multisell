package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMaxRequestBodyRejectsKnownOversizeBeforeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	calls := 0
	r := gin.New()
	r.Use(MaxRequestBody(4))
	r.POST("/", func(c *gin.Context) { calls++; c.Status(http.StatusNoContent) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("12345")))
	if w.Code != http.StatusRequestEntityTooLarge || calls != 0 {
		t.Fatalf("status=%d calls=%d", w.Code, calls)
	}
}

func TestMaxRequestBodyBoundsChunkedReader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var readErr error
	r := gin.New()
	r.Use(MaxRequestBody(4))
	r.POST("/", func(c *gin.Context) {
		_, readErr = io.ReadAll(c.Request.Body)
		c.Status(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("12345"))
	req.ContentLength = -1
	r.ServeHTTP(httptest.NewRecorder(), req)
	if readErr == nil {
		t.Fatal("chunked body exceeded limit without read error")
	}
}
