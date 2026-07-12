# LingMirror Image Service

Internal image execution service for LingMirror. It is not a public API and does not decide image rights, product truth, channel compliance, listing approval, or publication.

## Run

PostgreSQL is the required job/attempt/nonce authority for acceptance and
production. Apply the embedded, advisory-lock-protected migration explicitly
before starting the server:

```bash
DATABASE_URL='postgresql://image_service:secret@127.0.0.1:5432/image_service?sslmode=require' \
go run ./cmd/migrate

IMAGE_SERVICE_ENVIRONMENT=production \
IMAGE_SERVICE_JOB_STORE=postgres \
DATABASE_URL='postgresql://image_service:secret@127.0.0.1:5432/image_service?sslmode=require' \
IMAGE_SERVICE_SHARED_SECRET=dev-secret \
IMAGE_SERVICE_EXECUTION_TOKEN_SECRET=replace-with-independent-random-secret-32-bytes \
go run ./cmd/server
```

`IMAGE_SERVICE_JOB_STORE=file` remains available only for development and unit
tests. It persists to `IMAGE_SERVICE_DATA_DIR/jobs.json`; the server refuses to
start with that store when `IMAGE_SERVICE_ENVIRONMENT` is `acceptance` or
`production`. PostgreSQL enforces Owner-scoped job idempotency, globally unique
attempt idempotency, one queued/running attempt per job, lease ownership and
single-use execution nonces. Workers claim work with `FOR UPDATE SKIP LOCKED`.

PostgreSQL integration tests use `IMAGE_SERVICE_TEST_DATABASE_URL`. They skip
with an explicit reason when it is unset; a skip is not evidence of a real
database verification.

The first implementation provides authenticated blob ingestion, durable idempotent job records, content-addressed storage, and deterministic resize/white-canvas processing.

An OpenAI Images edit adapter and operation registry exist behind unit-tested internal contracts. The adapter defaults to `gpt-image-2` and fails closed without a key. It is intentionally **not registered by the server**. Non-deterministic execution requires a short-lived, target-bound, single-use authorization signed by LingMirror with the independent execution-token secret; valid authorization only permits a durable local attempt and does not make any Provider available. External providers and production MCP must not be described as available before registration and a real sandbox contract check are complete.
