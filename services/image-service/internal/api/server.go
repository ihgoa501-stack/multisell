package api

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/lingmirror/image-service/internal/blobstore"
	"github.com/lingmirror/image-service/internal/core"
	"github.com/lingmirror/image-service/internal/executionauth"
)

const maxJSONBodyBytes = 64 << 10

type Server struct {
	secret         string
	executionKey   []byte
	jobs           core.Repository
	blobs          *blobstore.Store
	mux            *http.ServeMux
	paidOperations map[string]string
}

type Config struct{ PaidOperations map[string]string }

func New(secret string, jobs core.Repository, blobs *blobstore.Store, executionKeys ...string) *Server {
	var key []byte
	if len(executionKeys) > 0 {
		key = []byte(executionKeys[0])
	}
	s := &Server{secret: secret, executionKey: key, jobs: jobs, blobs: blobs, mux: http.NewServeMux(), paidOperations: map[string]string{}}
	s.routes()
	return s
}
func NewConfigured(secret string, jobs core.Repository, blobs *blobstore.Store, executionKey string, cfg Config) *Server {
	s := New(secret, jobs, blobs, executionKey)
	for operation, processor := range cfg.PaidOperations {
		s.paidOperations[operation] = processor
	}
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
		if err := s.blobs.Ping(ctx); err != nil {
			problem(w, 503, "NOT_READY", "blob persistence unavailable")
			return
		}
		write(w, 200, map[string]string{"status": "ready"})
	})
	s.mux.HandleFunc("POST /internal/v1/blobs", s.auth(s.putBlob))
	s.mux.HandleFunc("GET /internal/v1/blobs/{id}/content", s.auth(s.getBlob))
	s.mux.HandleFunc("POST /internal/v1/jobs", s.auth(s.createJob))
	s.mux.HandleFunc("GET /internal/v1/processors", s.auth(s.listProcessors))
	s.mux.HandleFunc("GET /internal/v1/jobs/{id}", s.auth(s.getJob))
	s.mux.HandleFunc("GET /internal/v1/jobs/{id}/attempts", s.auth(s.listAttempts))
	s.mux.HandleFunc("POST /internal/v1/jobs/{id}/executions", s.auth(s.executeJob))
	s.mux.HandleFunc("POST /internal/v1/jobs/{id}/quiesce", s.auth(s.quiesceJob))
}
func (s *Server) quiesceJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in struct {
		OwnerID               int64  `json:"owner_id"`
		LingMirrorTaskID      string `json:"lingmirror_task_id"`
		LingMirrorTaskVersion int64  `json:"lingmirror_task_version"`
		ManifestHash          string `json:"manifest_hash"`
	}
	if decodeStrictJSON(w, r, &in) != nil || in.OwnerID <= 0 || in.LingMirrorTaskID == "" || in.LingMirrorTaskVersion <= 0 || !isLowerHexSHA256(in.ManifestHash) {
		problem(w, 422, "VALIDATION_ERROR", "exact job identity is required")
		return
	}
	job, ok, err := s.jobs.GetJob(id)
	if err != nil {
		problem(w, 500, "STORE_ERROR", "job persistence unavailable")
		return
	}
	if !ok {
		problem(w, 404, "NOT_FOUND", "job not found")
		return
	}
	if job.OwnerID != in.OwnerID || job.LingMirrorTaskID != in.LingMirrorTaskID || job.LingMirrorTaskVersion != in.LingMirrorTaskVersion || job.ManifestHash != in.ManifestHash {
		problem(w, 409, "VERSION_CONFLICT", "job identity changed")
		return
	}
	job, err = s.jobs.QuiesceJob(id)
	if errors.Is(err, core.ErrJobActive) {
		problem(w, 409, "JOB_ACTIVE", "job is running or already has output")
		return
	}
	if err != nil {
		problem(w, 500, "STORE_ERROR", "job quiescence could not be proven")
		return
	}
	write(w, 200, job)
}
func (s *Server) listProcessors(w http.ResponseWriter, _ *http.Request) {
	remaining, err := s.jobs.CanaryRemaining("photoroom")
	if err != nil {
		problem(w, 503, "STORE_ERROR", "capability persistence unavailable")
		return
	}
	_, photoroomConfigured := s.paidOperations["PHOTOROOM_REMOVE_BACKGROUND_SANDBOX"]
	_, openAIConfigured := s.paidOperations["OPENAI_IMAGE_EDIT"]
	write(w, 200, map[string]any{"items": []any{
		map[string]any{"code": "deterministic", "available": true, "operations": []string{"DETERMINISTIC_RESIZE"}, "safety_level": "local"},
		map[string]any{"code": "photoroom", "available": photoroomConfigured && remaining > 0, "operations": []string{"PHOTOROOM_REMOVE_BACKGROUND_SANDBOX", "PHOTOROOM_WHITE_BACKGROUND_SANDBOX", "PHOTOROOM_AI_SHADOW_SANDBOX"}, "safety_level": "sandbox_only", "provider_environment": "sandbox", "region": "us", "watermarked": true, "non_publishable": true, "quota_available": remaining > 0, "quota_remaining": remaining},
		map[string]any{"code": "openai", "available": openAIConfigured, "operations": []string{"OPENAI_IMAGE_EDIT"}, "safety_level": "production_paid", "provider_environment": "production", "region": "us", "watermarked": false, "non_publishable": false},
	}})
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
	if meta.NonPublishable {
		w.Header().Set("X-Image-Sandbox", "true")
		w.Header().Set("X-Image-Watermarked", "true")
		w.Header().Set("X-Image-Publishable", "false")
		w.Header().Set("X-Image-Restriction", meta.RestrictionReason)
	}
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
	if decodeStrictJSON(w, r, &in) != nil || in.OwnerID <= 0 || len(strings.TrimSpace(in.IdempotencyKey)) == 0 || len(in.IdempotencyKey) > 256 || !isLowerHexSHA256(in.ManifestHash) || !isLowerHexSHA256(in.InputBlobID) || in.Width < 100 || in.Width > 4000 || in.Height < 100 || in.Height > 4000 || (in.Format != "png" && in.Format != "jpeg") {
		problem(w, 422, "VALIDATION_ERROR", "invalid job request")
		return
	}
	if in.Operation == "DETERMINISTIC_RESIZE" {
		if strings.TrimSpace(in.Prompt) != "" || in.Processor != "" && in.Processor != "deterministic" || in.MaxCost != "" || in.Currency != "" || in.Region != "" && in.Region != "local" || in.ProviderEnvironment != "" || in.Sandbox || in.Watermarked || in.NonPublishable {
			problem(w, 422, "VALIDATION_ERROR", "invalid deterministic job request")
			return
		}
	} else {
		processor, available := s.paidOperations[in.Operation]
		if !available {
			problem(w, 422, "PROVIDER_UNAVAILABLE", "requested provider operation is unavailable")
			return
		}
		switch in.Operation {
		case "PHOTOROOM_REMOVE_BACKGROUND_SANDBOX", "PHOTOROOM_WHITE_BACKGROUND_SANDBOX", "PHOTOROOM_AI_SHADOW_SANDBOX":
			if processor != "photoroom" || strings.TrimSpace(in.Prompt) != "" || in.Processor != processor || in.LingMirrorTaskID == "" || in.LingMirrorTaskVersion <= 0 || in.ProviderEnvironment != "sandbox" || in.Region != "us" || !in.Sandbox || !in.Watermarked || !in.NonPublishable || in.Format != "png" || in.MaxCost != "0" || in.Currency != "USD" {
				problem(w, 422, "VALIDATION_ERROR", "invalid sandbox provider job request")
				return
			}
		case "OPENAI_IMAGE_EDIT":
			if processor != "openai" || strings.TrimSpace(in.Prompt) == "" || in.Processor != processor || in.LingMirrorTaskID == "" || in.LingMirrorTaskVersion <= 0 || in.ProviderEnvironment != "production" || in.Region != "us" || in.Sandbox || in.Watermarked || in.NonPublishable || in.Format != "png" || !positiveMoney(in.MaxCost) || in.Currency != "USD" || !supportedOpenAIImageSize(in.Width, in.Height) {
				problem(w, 422, "VALIDATION_ERROR", "invalid production OpenAI job request")
				return
			}
		default:
			problem(w, 422, "PROVIDER_UNAVAILABLE", "requested provider operation is unavailable")
			return
		}
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

func positiveMoney(value string) bool {
	cost, ok := new(big.Rat).SetString(value)
	return ok && cost.Sign() > 0
}

func supportedOpenAIImageSize(width, height int) bool {
	return width == 1024 && (height == 1024 || height == 1536) || width == 1536 && height == 1024
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
		if verifyErr != nil || claims.JobID != job.ID || claims.TaskID != job.LingMirrorTaskID || claims.TaskVersion != job.LingMirrorTaskVersion || claims.OwnerID != job.OwnerID || claims.ManifestHash != job.ManifestHash || claims.Operation != job.Operation || claims.Processor != job.Processor || claims.MaxCost != job.MaxCost || claims.Currency != job.Currency || claims.Region != job.Region || claims.ProviderEnvironment != job.ProviderEnvironment || claims.Sandbox != job.Sandbox || claims.Watermarked != job.Watermarked || claims.NonPublishable != job.NonPublishable {
			problem(w, 403, "BUDGET_TOKEN_INVALID", "valid target-bound execution authorization required")
			return
		}
		attempt, err := s.jobs.EnqueueAuthorizedAttempt(id, strings.TrimSpace(in.IdempotencyKey), claims.Nonce, claims.Processor)
		if errors.Is(err, core.ErrNonceConsumed) {
			problem(w, 409, "TOKEN_REPLAYED", "execution authorization already consumed")
			return
		}
		if errors.Is(err, core.ErrCanaryQuotaExceeded) {
			problem(w, 409, "CANARY_QUOTA_EXHAUSTED", "provider sandbox canary quota is exhausted")
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

func allowedCurrency(value string) bool {
	switch value {
	case "USD", "EUR", "CNY", "GBP", "JPY":
		return true
	}
	return false
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
