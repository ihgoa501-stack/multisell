package api

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lingmirror/image-service/internal/blobstore"
	"github.com/lingmirror/image-service/internal/core"
	"github.com/lingmirror/image-service/internal/executionauth"
)

const maxJSONBodyBytes = 64 << 10

type Server struct {
	secret       string
	executionKey []byte
	jobs         core.Repository
	blobs        *blobstore.Store
	mux          *http.ServeMux
}

func New(secret string, jobs core.Repository, blobs *blobstore.Store, executionKeys ...string) *Server {
	var key []byte
	if len(executionKeys) > 0 {
		key = []byte(executionKeys[0])
	}
	s := &Server{secret: secret, executionKey: key, jobs: jobs, blobs: blobs, mux: http.NewServeMux()}
	s.routes()
	return s
}
func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { write(w, 200, map[string]string{"status": "ok"}) })
	s.mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if err := s.jobs.Ping(ctx); err != nil {
			problem(w, 503, "NOT_READY", "job persistence unavailable")
			return
		}
		write(w, 200, map[string]string{"status": "ready"})
	})
	s.mux.HandleFunc("POST /internal/v1/blobs", s.auth(s.putBlob))
	s.mux.HandleFunc("GET /internal/v1/blobs/{id}/content", s.auth(s.getBlob))
	s.mux.HandleFunc("POST /internal/v1/jobs", s.auth(s.createJob))
	s.mux.HandleFunc("GET /internal/v1/jobs/{id}", s.auth(s.getJob))
	s.mux.HandleFunc("GET /internal/v1/jobs/{id}/attempts", s.auth(s.listAttempts))
	s.mux.HandleFunc("POST /internal/v1/jobs/{id}/executions", s.auth(s.executeJob))
}
func (s *Server) getBlob(w http.ResponseWriter, r *http.Request) {
	b, err := s.blobs.Get(r.PathValue("id"))
	if err != nil {
		problem(w, 404, "NOT_FOUND", "blob not found")
		return
	}
	meta, err := s.blobs.GetMetadata(r.PathValue("id"))
	if err != nil {
		problem(w, 500, "STORE_ERROR", "blob metadata unavailable")
		return
	}
	w.Header().Set("Content-Type", meta.MediaType)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(200)
	_, _ = w.Write(b)
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		want := "Bearer " + s.secret
		if s.secret == "" || subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
			problem(w, 401, "UNAUTHORIZED", "valid service credential required")
			return
		}
		next(w, r)
	}
}
func (s *Server) putBlob(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20))
	if err != nil {
		problem(w, 422, "VALIDATION_ERROR", "invalid blob")
		return
	}
	id, err := s.blobs.Put(b)
	if err != nil {
		if errors.Is(err, blobstore.ErrInvalidImage) || errors.Is(err, blobstore.ErrTooLarge) || errors.Is(err, blobstore.ErrTooManyPixels) {
			problem(w, 422, "VALIDATION_ERROR", err.Error())
			return
		}
		problem(w, 500, "STORE_ERROR", "blob persistence failed")
		return
	}
	write(w, 201, map[string]string{"blob_id": id})
}
func (s *Server) createJob(w http.ResponseWriter, r *http.Request) {
	var in core.CreateJob
	// Paid providers remain deliberately unavailable through this route until
	// action-bound Owner approval and budget tokens are enforced server-side.
	if decodeStrictJSON(w, r, &in) != nil || in.OwnerID <= 0 || len(strings.TrimSpace(in.IdempotencyKey)) == 0 || len(in.IdempotencyKey) > 256 || !isLowerHexSHA256(in.ManifestHash) || in.Operation != "DETERMINISTIC_RESIZE" || in.Prompt != "" || !isLowerHexSHA256(in.InputBlobID) || in.Width < 100 || in.Width > 4000 || in.Height < 100 || in.Height > 4000 || (in.Format != "png" && in.Format != "jpeg") {
		problem(w, 422, "VALIDATION_ERROR", "invalid job request")
		return
	}
	if _, err := s.blobs.Get(in.InputBlobID); err != nil {
		problem(w, 422, "INPUT_BLOB_INVALID", "input blob unavailable")
		return
	}
	j, replay, err := s.jobs.Create(in)
	if errors.Is(err, core.ErrIdempotencyConflict) {
		problem(w, 409, "IDEMPOTENCY_CONFLICT", err.Error())
		return
	}
	if err != nil {
		problem(w, 500, "STORE_ERROR", "job persistence failed")
		return
	}
	if replay {
		w.Header().Set("Idempotent-Replay", "true")
	}
	write(w, 201, j)
}
func (s *Server) getJob(w http.ResponseWriter, r *http.Request) {
	j, ok, err := s.jobs.GetJob(r.PathValue("id"))
	if err != nil {
		problem(w, 500, "STORE_ERROR", "job persistence unavailable")
		return
	}
	if !ok {
		problem(w, 404, "NOT_FOUND", "job not found")
		return
	}
	write(w, 200, j)
}
func (s *Server) executeJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_, ok, err := s.jobs.GetJob(id)
	if err != nil {
		problem(w, 500, "STORE_ERROR", "job persistence unavailable")
		return
	}
	if !ok {
		problem(w, 404, "NOT_FOUND", "job not found")
		return
	}
	var in struct {
		IdempotencyKey string `json:"idempotency_key"`
		ExecutionToken string `json:"execution_token,omitempty"`
	}
	if decodeStrictJSON(w, r, &in) != nil || strings.TrimSpace(in.IdempotencyKey) == "" || len(in.IdempotencyKey) > 256 {
		problem(w, 422, "VALIDATION_ERROR", "idempotency_key is required")
		return
	}
	job, _, err := s.jobs.GetJob(id)
	if err != nil {
		problem(w, 500, "STORE_ERROR", "job persistence unavailable")
		return
	}
	if job.Operation != "DETERMINISTIC_RESIZE" {
		claims, verifyErr := executionauth.Verify(strings.TrimSpace(in.ExecutionToken), s.executionKey, time.Now().UTC())
		if verifyErr != nil || claims.JobID != job.ID || claims.TaskID != job.LingMirrorTaskID || claims.TaskVersion != job.LingMirrorTaskVersion || claims.OwnerID != job.OwnerID || claims.ManifestHash != job.ManifestHash || claims.Operation != job.Operation || claims.Processor != job.Processor {
			problem(w, 403, "BUDGET_TOKEN_INVALID", "valid target-bound execution authorization required")
			return
		}
		attempt, err := s.jobs.EnqueueAuthorizedAttempt(id, strings.TrimSpace(in.IdempotencyKey), claims.Nonce)
		if errors.Is(err, core.ErrNonceConsumed) {
			problem(w, 409, "TOKEN_REPLAYED", "execution authorization already consumed")
			return
		}
		if err != nil {
			problem(w, 409, "VERSION_CONFLICT", err.Error())
			return
		}
		write(w, 202, attempt)
		return
	}
	attempt, replay, err := s.jobs.EnqueueAttempt(id, strings.TrimSpace(in.IdempotencyKey))
	if errors.Is(err, core.ErrAttemptIdempotencyConflict) {
		problem(w, 409, "IDEMPOTENCY_CONFLICT", err.Error())
		return
	}
	if err != nil {
		problem(w, 409, "VERSION_CONFLICT", err.Error())
		return
	}
	if replay {
		w.Header().Set("Idempotent-Replay", "true")
	}
	write(w, 202, attempt)
}

func decodeStrictJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxJSONBodyBytes))
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("request body must contain exactly one JSON value")
	}
	return nil
}

func isLowerHexSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}
func (s *Server) listAttempts(w http.ResponseWriter, r *http.Request) {
	if _, ok, err := s.jobs.GetJob(r.PathValue("id")); err != nil {
		problem(w, 500, "STORE_ERROR", "job persistence unavailable")
		return
	} else if !ok {
		problem(w, 404, "NOT_FOUND", "job not found")
		return
	}
	attempts, err := s.jobs.ListJobAttempts(r.PathValue("id"))
	if err != nil {
		problem(w, 500, "STORE_ERROR", "attempt persistence unavailable")
		return
	}
	write(w, 200, map[string]any{"items": attempts})
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, status int, code, msg string) {
	write(w, status, map[string]any{"error": map[string]string{"code": code, "message": strings.TrimSpace(msg)}})
}
