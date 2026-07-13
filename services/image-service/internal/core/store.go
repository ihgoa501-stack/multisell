package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

var ErrIdempotencyConflict = errors.New("idempotency key reused with different manifest")
var ErrAttemptIdempotencyConflict = errors.New("attempt idempotency key reused for another job")
var ErrAttemptLeaseLost = errors.New("attempt lease lost")
var ErrJobAlreadyActive = errors.New("job already has an active execution attempt")
var ErrJobNotExecutable = errors.New("job is not executable in its current state")
var ErrJobActive = errors.New("job execution is active")

type Store struct {
	mu              sync.Mutex
	path            string
	jobs            map[string]*Job
	attempts        map[string]*Attempt
	consumedNonces  map[string]time.Time
	providerSubmits map[string]time.Time
	canaryClaims    map[string]time.Time
}

type snapshot struct {
	Jobs            map[string]*Job      `json:"jobs"`
	Attempts        map[string]*Attempt  `json:"attempts"`
	ConsumedNonces  map[string]time.Time `json:"consumed_nonces"`
	ProviderSubmits map[string]time.Time `json:"provider_submits"`
	CanaryClaims    map[string]time.Time `json:"canary_claims"`
}

func OpenStore(path string) (*Store, error) {
	s := &Store{path: path, jobs: map[string]*Job{}, attempts: map[string]*Attempt{}, consumedNonces: map[string]time.Time{}, providerSubmits: map[string]time.Time{}, canaryClaims: map[string]time.Time{}}
	b, err := os.ReadFile(path)
	if err == nil && len(b) > 0 {
		var snap snapshot
		if err := json.Unmarshal(b, &snap); err != nil {
			return nil, fmt.Errorf("decode job store: %w", err)
		}
		if snap.Jobs != nil {
			s.jobs = snap.Jobs
		}
		if snap.Attempts != nil {
			s.attempts = snap.Attempts
		}
		if snap.ConsumedNonces != nil {
			s.consumedNonces = snap.ConsumedNonces
		}
		if snap.ProviderSubmits != nil {
			s.providerSubmits = snap.ProviderSubmits
		}
		if snap.CanaryClaims != nil {
			s.canaryClaims = snap.CanaryClaims
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return s, nil
}

func (s *Store) Create(in CreateJob) (*Job, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if in.Operation != "DETERMINISTIC_RESIZE" && (in.LingMirrorTaskID == "" || in.LingMirrorTaskVersion <= 0 || in.Processor == "") {
		return nil, false, errors.New("paid job requires LingMirror task, version, and processor binding")
	}
	for _, existing := range s.jobs {
		if existing.OwnerID == in.OwnerID && existing.IdempotencyKey == in.IdempotencyKey {
			if existing.ManifestHash != in.ManifestHash {
				return nil, false, ErrIdempotencyConflict
			}
			copy := *existing
			return &copy, true, nil
		}
	}
	now := time.Now().UTC()
	processor := in.Processor
	if processor == "" && in.Operation == "DETERMINISTIC_RESIZE" {
		processor = "deterministic"
	}
	id, err := newID()
	if err != nil {
		return nil, false, err
	}
	j := &Job{ID: id, OwnerID: in.OwnerID, LingMirrorTaskID: in.LingMirrorTaskID, LingMirrorTaskVersion: in.LingMirrorTaskVersion, IdempotencyKey: in.IdempotencyKey, ManifestHash: in.ManifestHash, Operation: in.Operation, Processor: processor, Prompt: in.Prompt, InputBlobID: in.InputBlobID, Width: in.Width, Height: in.Height, Format: in.Format, MaxCost: in.MaxCost, Currency: in.Currency, Region: in.Region, ProviderEnvironment: in.ProviderEnvironment, Sandbox: in.Sandbox, Watermarked: in.Watermarked, NonPublishable: in.NonPublishable, Status: JobQueued, Version: 1, CreatedAt: now, UpdatedAt: now}
	s.jobs[j.ID] = j
	if err := s.persist(); err != nil {
		delete(s.jobs, j.ID)
		return nil, false, err
	}
	copy := *j
	return &copy, false, nil
}

func (s *Store) Get(id string) (*Job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil, false
	}
	copy := *j
	return &copy, true
}

func (s *Store) GetJob(id string) (*Job, bool, error) {
	j, ok := s.Get(id)
	return j, ok, nil
}

func (s *Store) QuiesceJob(id string) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	if job.Status == JobRunning || job.Status == JobReady {
		return nil, ErrJobActive
	}
	if job.Status != JobQueued {
		copy := *job
		return &copy, nil
	}
	for _, attempt := range s.attempts {
		if attempt.JobID == id && attempt.Status == AttemptRunning {
			return nil, ErrJobActive
		}
	}
	now := time.Now().UTC()
	previousJob := *job
	previousAttempts := make(map[string]Attempt)
	for _, attempt := range s.attempts {
		if attempt.JobID == id && attempt.Status == AttemptQueued {
			previousAttempts[attempt.ID] = *attempt
			attempt.Status, attempt.ErrorCode, attempt.CompletedAt = AttemptFailed, "CANCELLED_NO_CHARGE_RECONCILIATION", &now
		}
	}
	job.Status, job.ErrorCode, job.Version, job.UpdatedAt = JobFailed, "CANCELLED_NO_CHARGE_RECONCILIATION", job.Version+1, now
	if err := s.persist(); err != nil {
		*job = previousJob
		for id, attempt := range previousAttempts {
			*s.attempts[id] = attempt
		}
		return nil, err
	}
	copy := *job
	return &copy, nil
}

func (s *Store) Transition(id string, from, to JobStatus, output, code string) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	if j.Status != from {
		return nil, fmt.Errorf("status conflict: have %s want %s", j.Status, from)
	}
	previous := *j
	j.Status, j.OutputBlobID, j.ErrorCode, j.Version, j.UpdatedAt = to, output, code, j.Version+1, time.Now().UTC()
	if err := s.persist(); err != nil {
		*j = previous
		return nil, err
	}
	copy := *j
	return &copy, nil
}

// EnqueueAttempt creates one durable execution attempt. Repeating the same
// idempotency key for the same job returns the original attempt; reusing it for
// another job is rejected.
func (s *Store) EnqueueAttempt(jobID, idempotencyKey string) (*Attempt, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return nil, false, os.ErrNotExist
	}
	if idempotencyKey == "" {
		return nil, false, errors.New("attempt idempotency key is required")
	}
	for _, existing := range s.attempts {
		if existing.IdempotencyKey != idempotencyKey {
			continue
		}
		if existing.JobID != jobID {
			return nil, false, ErrAttemptIdempotencyConflict
		}
		return cloneAttempt(existing), true, nil
	}
	if job.Status != JobQueued {
		return nil, false, ErrJobNotExecutable
	}
	for _, existing := range s.attempts {
		if existing.JobID == jobID && (existing.Status == AttemptQueued || existing.Status == AttemptRunning) {
			return nil, false, ErrJobAlreadyActive
		}
	}
	number := 1
	for _, existing := range s.attempts {
		if existing.JobID == jobID && existing.Number >= number {
			number = existing.Number + 1
		}
	}
	id, err := newID()
	if err != nil {
		return nil, false, err
	}
	now := time.Now().UTC()
	attempt := &Attempt{ID: id, JobID: jobID, IdempotencyKey: idempotencyKey, Number: number, Status: AttemptQueued, CreatedAt: now}
	s.attempts[attempt.ID] = attempt
	if err := s.persist(); err != nil {
		delete(s.attempts, attempt.ID)
		return nil, false, err
	}
	return cloneAttempt(attempt), false, nil
}

var ErrNonceConsumed = errors.New("execution authorization nonce already consumed")

// EnqueueAuthorizedAttempt atomically persists nonce consumption and the attempt.
// A failed enqueue never consumes the nonce.
func (s *Store) EnqueueAuthorizedAttempt(jobID, idempotencyKey, nonce string, providers ...string) (*Attempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if nonce == "" {
		return nil, errors.New("execution authorization nonce is required")
	}
	if _, exists := s.consumedNonces[nonce]; exists {
		return nil, ErrNonceConsumed
	}
	job, ok := s.jobs[jobID]
	if !ok {
		return nil, os.ErrNotExist
	}
	provider := ""
	if len(providers) > 0 {
		provider = providers[0]
	}
	if provider == "photoroom" {
		if _, used := s.canaryClaims[provider]; used {
			return nil, ErrCanaryQuotaExceeded
		}
	}
	if idempotencyKey == "" {
		return nil, errors.New("attempt idempotency key is required")
	}
	if job.Status != JobQueued {
		return nil, ErrJobNotExecutable
	}
	for _, existing := range s.attempts {
		if existing.IdempotencyKey == idempotencyKey {
			return nil, ErrAttemptIdempotencyConflict
		}
		if existing.JobID == jobID && (existing.Status == AttemptQueued || existing.Status == AttemptRunning) {
			return nil, ErrJobAlreadyActive
		}
	}
	number := 1
	for _, existing := range s.attempts {
		if existing.JobID == jobID && existing.Number >= number {
			number = existing.Number + 1
		}
	}
	id, err := newID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	attempt := &Attempt{ID: id, JobID: jobID, IdempotencyKey: idempotencyKey, Number: number, Status: AttemptQueued, CreatedAt: now}
	s.attempts[attempt.ID] = attempt
	s.consumedNonces[nonce] = now
	if provider == "photoroom" {
		s.canaryClaims[provider] = now
	}
	if err := s.persist(); err != nil {
		delete(s.attempts, attempt.ID)
		delete(s.consumedNonces, nonce)
		if provider == "photoroom" {
			delete(s.canaryClaims, provider)
		}
		return nil, err
	}
	return cloneAttempt(attempt), nil
}

var ErrCanaryQuotaExceeded = errors.New("provider canary quota exhausted")

func (s *Store) ClaimProviderSubmit(jobID, provider string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := provider + ":" + jobID
	if _, exists := s.providerSubmits[key]; exists {
		return false, nil
	}
	s.providerSubmits[key] = time.Now().UTC()
	if err := s.persist(); err != nil {
		delete(s.providerSubmits, key)
		return false, err
	}
	return true, nil
}
func (s *Store) CanaryRemaining(provider string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, used := s.canaryClaims[provider]; used {
		return 0, nil
	}
	return 1, nil
}

// ClaimAttempt leases the oldest queued attempt, or takes over a running
// attempt whose lease expired. A lease owner must be stable for one worker
// process and is required when completing the attempt.
func (s *Store) ClaimAttempt(leaseOwner string, lease time.Duration) (*Attempt, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if leaseOwner == "" || lease <= 0 {
		return nil, false, errors.New("valid lease owner and duration are required")
	}
	now := time.Now().UTC()
	eligible := make([]*Attempt, 0)
	runningJobs := make(map[string]bool)
	for _, attempt := range s.attempts {
		if attempt.Status == AttemptRunning && attempt.LeaseUntil != nil && attempt.LeaseUntil.After(now) {
			runningJobs[attempt.JobID] = true
		}
	}
	for _, attempt := range s.attempts {
		if (attempt.Status == AttemptQueued && !runningJobs[attempt.JobID]) || (attempt.Status == AttemptRunning && attempt.LeaseUntil != nil && !attempt.LeaseUntil.After(now)) {
			eligible = append(eligible, attempt)
		}
	}
	if len(eligible) == 0 {
		return nil, false, nil
	}
	sort.Slice(eligible, func(i, j int) bool {
		if eligible[i].CreatedAt.Equal(eligible[j].CreatedAt) {
			return eligible[i].ID < eligible[j].ID
		}
		return eligible[i].CreatedAt.Before(eligible[j].CreatedAt)
	})
	attempt := eligible[0]
	previous := *attempt
	until := now.Add(lease)
	attempt.Status = AttemptRunning
	attempt.LeaseOwner = leaseOwner
	attempt.LeaseUntil = &until
	if attempt.StartedAt == nil {
		attempt.StartedAt = &now
	}
	if err := s.persist(); err != nil {
		*attempt = previous
		return nil, false, err
	}
	return cloneAttempt(attempt), true, nil
}

// CompleteAttempt applies a terminal result only for the worker that still
// owns an unexpired lease, preventing a stale worker from overwriting a retry.
func (s *Store) CompleteAttempt(id, leaseOwner string, status AttemptStatus, errorCode string, providerRequestID ...string) (*Attempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if status != AttemptSucceeded && status != AttemptFailed && status != AttemptReconcileRequired {
		return nil, errors.New("attempt completion requires terminal status")
	}
	attempt, ok := s.attempts[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	now := time.Now().UTC()
	if attempt.Status != AttemptRunning || attempt.LeaseOwner != leaseOwner || attempt.LeaseUntil == nil || !attempt.LeaseUntil.After(now) {
		return nil, ErrAttemptLeaseLost
	}
	previous := *attempt
	attempt.Status = status
	attempt.ErrorCode = errorCode
	if len(providerRequestID) > 0 {
		attempt.ProviderRequestID = providerRequestID[0]
	}
	attempt.CompletedAt = &now
	attempt.LeaseOwner = ""
	attempt.LeaseUntil = nil
	if err := s.persist(); err != nil {
		*attempt = previous
		return nil, err
	}
	return cloneAttempt(attempt), nil
}

func (s *Store) FinalizeAttempt(in AttemptFinalization) (*Job, *Attempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if in.AttemptStatus != AttemptSucceeded && in.AttemptStatus != AttemptFailed && in.AttemptStatus != AttemptReconcileRequired {
		return nil, nil, errors.New("attempt finalization requires terminal status")
	}
	attempt, ok := s.attempts[in.AttemptID]
	if !ok {
		return nil, nil, os.ErrNotExist
	}
	now := time.Now().UTC()
	if attempt.JobID != in.JobID || attempt.Status != AttemptRunning || attempt.LeaseOwner != in.LeaseOwner || attempt.LeaseUntil == nil || !attempt.LeaseUntil.After(now) {
		return nil, nil, ErrAttemptLeaseLost
	}
	job, ok := s.jobs[in.JobID]
	if !ok {
		return nil, nil, os.ErrNotExist
	}
	if job.Status != in.FromJobStatus {
		return nil, nil, fmt.Errorf("status conflict: have %s want %s", job.Status, in.FromJobStatus)
	}
	previousJob, previousAttempt := *job, *attempt
	job.Status, job.OutputBlobID, job.ErrorCode, job.Version, job.UpdatedAt = in.ToJobStatus, in.OutputBlobID, in.ErrorCode, job.Version+1, now
	attempt.Status, attempt.ErrorCode, attempt.ProviderRequestID = in.AttemptStatus, in.ErrorCode, in.ProviderRequestID
	attempt.CompletedAt, attempt.LeaseOwner, attempt.LeaseUntil = &now, "", nil
	if err := s.persist(); err != nil {
		*job, *attempt = previousJob, previousAttempt
		return nil, nil, err
	}
	jobCopy := *job
	return &jobCopy, cloneAttempt(attempt), nil
}

func (s *Store) RenewAttemptLease(id, leaseOwner string, lease time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if lease <= 0 {
		return errors.New("valid lease duration is required")
	}
	attempt, ok := s.attempts[id]
	if !ok {
		return os.ErrNotExist
	}
	now := time.Now().UTC()
	if attempt.Status != AttemptRunning || attempt.LeaseOwner != leaseOwner || attempt.LeaseUntil == nil || !attempt.LeaseUntil.After(now) {
		return ErrAttemptLeaseLost
	}
	previous := *attempt
	until := now.Add(lease)
	attempt.LeaseUntil = &until
	if err := s.persist(); err != nil {
		*attempt = previous
		return err
	}
	return nil
}

func (s *Store) ListAttempts(jobID string) []Attempt {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Attempt, 0)
	for _, attempt := range s.attempts {
		if attempt.JobID == jobID {
			result = append(result, *cloneAttempt(attempt))
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Number < result[j].Number })
	return result
}

func (s *Store) ListJobAttempts(jobID string) ([]Attempt, error) {
	return s.ListAttempts(jobID), nil
}

func (s *Store) Ping(_ context.Context) error { return nil }
func (s *Store) Close() error                 { return nil }

func cloneAttempt(in *Attempt) *Attempt {
	copy := *in
	if in.LeaseUntil != nil {
		v := *in.LeaseUntil
		copy.LeaseUntil = &v
	}
	if in.StartedAt != nil {
		v := *in.StartedAt
		copy.StartedAt = &v
	}
	if in.CompletedAt != nil {
		v := *in.CompletedAt
		copy.CompletedAt = &v
	}
	return &copy
}

func (s *Store) persist() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(snapshot{Jobs: s.jobs, Attempts: s.attempts, ConsumedNonces: s.consumedNonces, ProviderSubmits: s.providerSubmits, CanaryClaims: s.canaryClaims}, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	tmp := s.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	err = d.Sync()
	closeErr = d.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
