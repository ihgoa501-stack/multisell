package postgres

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/lib/pq"
	"github.com/lingmirror/image-service/internal/core"
)

//go:embed schema.sql
var schema string

type Store struct{ db *sql.DB }

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL is required for PostgreSQL image job storage")
	}
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect image database: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }
func (s *Store) Close() error                   { return s.db.Close() }

func (s *Store) Migrate(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('lingmirror_image_service_schema_v1'))`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("migrate image schema: %w", err)
	}
	return tx.Commit()
}

const jobColumns = `id, owner_id, lingmirror_task_id, lingmirror_task_version, idempotency_key, manifest_hash,
 operation, processor, prompt, input_blob_id, output_blob_id, width, height, format, status, error_code,
 version, created_at, updated_at`

func scanJob(row interface{ Scan(...any) error }) (*core.Job, error) {
	var j core.Job
	err := row.Scan(&j.ID, &j.OwnerID, &j.LingMirrorTaskID, &j.LingMirrorTaskVersion, &j.IdempotencyKey,
		&j.ManifestHash, &j.Operation, &j.Processor, &j.Prompt, &j.InputBlobID, &j.OutputBlobID,
		&j.Width, &j.Height, &j.Format, &j.Status, &j.ErrorCode, &j.Version, &j.CreatedAt, &j.UpdatedAt)
	return &j, err
}

func (s *Store) Create(in core.CreateJob) (*core.Job, bool, error) {
	if in.Operation != "DETERMINISTIC_RESIZE" && (in.LingMirrorTaskID == "" || in.LingMirrorTaskVersion <= 0 || in.Processor == "") {
		return nil, false, errors.New("paid job requires LingMirror task, version, and processor binding")
	}
	id, err := randomID()
	if err != nil {
		return nil, false, err
	}
	if in.Processor == "" && in.Operation == "DETERMINISTIC_RESIZE" {
		in.Processor = "deterministic"
	}
	now := time.Now().UTC()
	row := s.db.QueryRow(`INSERT INTO image_jobs (`+jobColumns+`) VALUES
	 ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'',$11,$12,$13,'QUEUED','',1,$14,$14)
	 ON CONFLICT (owner_id,idempotency_key) DO UPDATE SET idempotency_key=EXCLUDED.idempotency_key
	 RETURNING `+jobColumns, id, in.OwnerID, in.LingMirrorTaskID, in.LingMirrorTaskVersion, in.IdempotencyKey,
		in.ManifestHash, in.Operation, in.Processor, in.Prompt, in.InputBlobID, in.Width, in.Height, in.Format, now)
	j, err := scanJob(row)
	if err != nil {
		return nil, false, err
	}
	if j.ManifestHash != in.ManifestHash {
		return nil, false, core.ErrIdempotencyConflict
	}
	return j, j.ID != id, nil
}

func (s *Store) GetJob(id string) (*core.Job, bool, error) {
	j, err := scanJob(s.db.QueryRow(`SELECT `+jobColumns+` FROM image_jobs WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	return j, err == nil, err
}

func (s *Store) Transition(id string, from, to core.JobStatus, output, code string) (*core.Job, error) {
	j, err := scanJob(s.db.QueryRow(`UPDATE image_jobs SET status=$3, output_blob_id=$4, error_code=$5,
	 version=version+1, updated_at=now() WHERE id=$1 AND status=$2 RETURNING `+jobColumns, id, from, to, output, code))
	if errors.Is(err, sql.ErrNoRows) {
		if _, ok, getErr := s.GetJob(id); getErr != nil {
			return nil, getErr
		} else if !ok {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("status conflict: expected %s", from)
	}
	return j, err
}

const attemptColumns = `id, job_id, idempotency_key, number, status, lease_owner, lease_until,
 error_code, created_at, started_at, completed_at`

func scanAttempt(row interface{ Scan(...any) error }) (*core.Attempt, error) {
	var a core.Attempt
	err := row.Scan(&a.ID, &a.JobID, &a.IdempotencyKey, &a.Number, &a.Status, &a.LeaseOwner,
		&a.LeaseUntil, &a.ErrorCode, &a.CreatedAt, &a.StartedAt, &a.CompletedAt)
	return &a, err
}

func (s *Store) EnqueueAttempt(jobID, key string) (*core.Attempt, bool, error) {
	return s.enqueue(jobID, key, "", false)
}

func (s *Store) EnqueueAuthorizedAttempt(jobID, key, nonce string) (*core.Attempt, error) {
	a, _, err := s.enqueue(jobID, key, nonce, true)
	return a, err
}

func (s *Store) enqueue(jobID, key, nonce string, authorized bool) (*core.Attempt, bool, error) {
	if key == "" {
		return nil, false, errors.New("attempt idempotency key is required")
	}
	if authorized && nonce == "" {
		return nil, false, errors.New("execution authorization nonce is required")
	}
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	var status core.JobStatus
	if err = tx.QueryRow(`SELECT status FROM image_jobs WHERE id=$1 FOR UPDATE`, jobID).Scan(&status); errors.Is(err, sql.ErrNoRows) {
		return nil, false, os.ErrNotExist
	} else if err != nil {
		return nil, false, err
	}
	if authorized {
		var consumed bool
		if err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM image_consumed_nonces WHERE nonce=$1)`, nonce).Scan(&consumed); err != nil {
			return nil, false, err
		}
		if consumed {
			return nil, false, core.ErrNonceConsumed
		}
	}
	var existingJob string
	err = tx.QueryRow(`SELECT job_id FROM image_attempts WHERE idempotency_key=$1`, key).Scan(&existingJob)
	if err == nil {
		if authorized || existingJob != jobID {
			return nil, false, core.ErrAttemptIdempotencyConflict
		}
		a, scanErr := scanAttempt(tx.QueryRow(`SELECT `+attemptColumns+` FROM image_attempts WHERE idempotency_key=$1`, key))
		return a, true, scanErr
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}
	if status != core.JobQueued {
		return nil, false, core.ErrJobNotExecutable
	}
	var number int
	if err = tx.QueryRow(`SELECT COALESCE(MAX(number),0)+1 FROM image_attempts WHERE job_id=$1`, jobID).Scan(&number); err != nil {
		return nil, false, err
	}
	id, err := randomID()
	if err != nil {
		return nil, false, err
	}
	now := time.Now().UTC()
	a, err := scanAttempt(tx.QueryRow(`INSERT INTO image_attempts
	 (id,job_id,idempotency_key,number,status,lease_owner,lease_until,error_code,created_at)
	 VALUES ($1,$2,$3,$4,'QUEUED','',NULL,'',$5) RETURNING `+attemptColumns, id, jobID, key, number, now))
	if err != nil {
		return nil, false, mapAttemptConstraint(err)
	}
	if authorized {
		if _, err = tx.Exec(`INSERT INTO image_consumed_nonces(nonce,attempt_id,consumed_at) VALUES($1,$2,$3)`, nonce, id, now); err != nil {
			if isUnique(err) {
				return nil, false, core.ErrNonceConsumed
			}
			return nil, false, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, false, err
	}
	return a, false, nil
}

func (s *Store) ClaimAttempt(owner string, lease time.Duration) (*core.Attempt, bool, error) {
	if owner == "" || lease <= 0 {
		return nil, false, errors.New("valid lease owner and duration are required")
	}
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	a, err := scanAttempt(tx.QueryRow(`SELECT ` + attemptColumns + ` FROM image_attempts a
	 WHERE (a.status='QUEUED' OR (a.status='RUNNING' AND a.lease_until <= now()))
	 AND NOT EXISTS (SELECT 1 FROM image_attempts live WHERE live.job_id=a.job_id AND live.status='RUNNING' AND live.lease_until > now())
	 ORDER BY a.created_at,a.id FOR UPDATE SKIP LOCKED LIMIT 1`))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	until := time.Now().UTC().Add(lease)
	a, err = scanAttempt(tx.QueryRow(`UPDATE image_attempts SET status='RUNNING', lease_owner=$2, lease_until=$3,
	 started_at=COALESCE(started_at,now()) WHERE id=$1 RETURNING `+attemptColumns, a.ID, owner, until))
	if err != nil {
		return nil, false, err
	}
	if err = tx.Commit(); err != nil {
		return nil, false, err
	}
	return a, true, nil
}

func (s *Store) CompleteAttempt(id, owner string, status core.AttemptStatus, code string) (*core.Attempt, error) {
	if status != core.AttemptSucceeded && status != core.AttemptFailed {
		return nil, errors.New("attempt completion requires terminal status")
	}
	a, err := scanAttempt(s.db.QueryRow(`UPDATE image_attempts SET status=$3,error_code=$4,completed_at=now(),lease_owner='',lease_until=NULL
	 WHERE id=$1 AND status='RUNNING' AND lease_owner=$2 AND lease_until>now() RETURNING `+attemptColumns, id, owner, status, code))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, core.ErrAttemptLeaseLost
	}
	return a, err
}

func (s *Store) RenewAttemptLease(id, owner string, lease time.Duration) error {
	if lease <= 0 {
		return errors.New("valid lease duration is required")
	}
	result, err := s.db.Exec(`UPDATE image_attempts SET lease_until=now()+$3::interval WHERE id=$1 AND status='RUNNING' AND lease_owner=$2 AND lease_until>now()`, id, owner, lease.String())
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return core.ErrAttemptLeaseLost
	}
	return nil
}

func (s *Store) ListJobAttempts(jobID string) ([]core.Attempt, error) {
	rows, err := s.db.Query(`SELECT `+attemptColumns+` FROM image_attempts WHERE job_id=$1 ORDER BY number`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]core.Attempt, 0)
	for rows.Next() {
		a, err := scanAttempt(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *a)
	}
	return items, rows.Err()
}

func mapAttemptConstraint(err error) error {
	var pg *pq.Error
	if errors.As(err, &pg) && string(pg.Code) == "23505" {
		if pg.Constraint == "image_attempts_one_active_per_job" {
			return core.ErrJobAlreadyActive
		}
		return core.ErrAttemptIdempotencyConflict
	}
	return err
}

func isUnique(err error) bool {
	var pg *pq.Error
	return errors.As(err, &pg) && string(pg.Code) == "23505"
}
