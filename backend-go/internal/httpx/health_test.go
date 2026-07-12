package httpx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type healthPinger struct{ err error }

func (p healthPinger) PingContext(context.Context) error { return p.err }

func TestHealth_LivenessDoesNotDependOnReadiness(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	registerHealthHandlers(r, readinessDependencies{db: healthPinger{err: errors.New("down")}, eventBusRunning: func() bool { return false }, schedulerRunning: func() bool { return false }, version: "test"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if w.Code != http.StatusOK || w.Body.String() != `{"status":"alive","version":"test"}` {
		t.Fatalf("liveness response = %d %s", w.Code, w.Body.String())
	}
}

func TestHealth_ReadinessRequiresAllComponents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name                    string
		dbErr                   error
		bus, scheduler, traffic bool
		want                    int
	}{
		{"all ready", nil, true, true, true, http.StatusOK},
		{"database down", errors.New("down"), true, true, true, http.StatusServiceUnavailable},
		{"event bus down", nil, false, true, true, http.StatusServiceUnavailable},
		{"scheduler down", nil, true, false, true, http.StatusServiceUnavailable},
		{"traffic draining", nil, true, true, false, http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			registerHealthHandlers(r, readinessDependencies{db: healthPinger{err: tc.dbErr}, eventBusRunning: func() bool { return tc.bus }, schedulerRunning: func() bool { return tc.scheduler }, acceptingTraffic: func() bool { return tc.traffic }, version: "test"})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/ready", nil))
			if w.Code != tc.want {
				t.Fatalf("readiness status = %d, want %d; body=%s", w.Code, tc.want, w.Body.String())
			}
		})
	}
}
