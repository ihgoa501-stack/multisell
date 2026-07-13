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
IMAGE_SERVICE_SHARED_SECRET=replace-with-random-shared-secret-at-least-32-bytes \
IMAGE_SERVICE_EXECUTION_TOKEN_SECRET=replace-with-independent-random-secret-32-bytes \
go run ./cmd/server
```

`IMAGE_SERVICE_JOB_STORE=file` remains available only for development and unit
tests. It persists to `IMAGE_SERVICE_DATA_DIR/jobs.json`; the server refuses to
start with that store when `IMAGE_SERVICE_ENVIRONMENT` is `acceptance` or
`production`. PostgreSQL enforces Owner-scoped job idempotency, globally unique
attempt idempotency, one queued/running attempt per job, lease ownership and
single-use execution nonces. Workers claim work with `FOR UPDATE SKIP LOCKED`.

`IMAGE_SERVICE_ENVIRONMENT` is a closed enum: `development`, `acceptance`, or
`production`; every other value refuses startup. Acceptance and production also
require the shared secret to be at least 32 bytes and different from the
execution-token secret.

PostgreSQL integration tests use `IMAGE_SERVICE_TEST_DATABASE_URL`. They skip
with an explicit reason when it is unset; a skip is not evidence of a real
database verification.

The first implementation provides authenticated blob ingestion, durable idempotent job records, content-addressed storage, and deterministic resize/white-canvas processing.

The OpenAI Images edit adapter is disabled by default and fixed to `gpt-image-2`.
It registers only with `IMAGE_SERVICE_OPENAI_ENABLED=true` and the dedicated
`IMAGE_SERVICE_OPENAI_API_KEY`. Each production request also requires exact
input-rights evidence, a short-lived target-bound Owner authorization, and a
matching budget reservation. `max_cost` is LingMirror's reservation ceiling,
not a Provider-enforced price cap. Without a real credential and Owner SKU run,
the path is `automated_verified`, not externally verified.

The OpenAI Images Edits contract does not document a verifiable idempotency or
result-query API. LingMirror therefore treats each paid intent as one-shot:
timeouts, interrupted responses, 5xx, or a local blob-write failure become
`RECONCILE_REQUIRED` and are never retried automatically. The sanitized
Provider request ID is persisted on both successful and failed attempts when
the response supplied one. Job/output/error and attempt/request ID reach their
terminal states in one repository transaction, so a crash cannot leave a
terminal job with an untraceable running attempt. A queued job can be quiesced only through the
private exact owner/task/version/manifest endpoint before no-charge or
charged-without-output reconciliation is allowed.

A Photoroom Image Editing adapter exists at `internal/providers/photoroom` with
the fixed safety level `sandbox_only`. It is disabled by default. A single
non-production, non-publishable canary can be enabled only when all four gates
are explicit:

```bash
IMAGE_SERVICE_PHOTOROOM_SANDBOX_ENABLED=true
IMAGE_SERVICE_PHOTOROOM_API_KEY=...
IMAGE_SERVICE_PHOTOROOM_SANDBOX_ACCOUNT_CONFIRMED=true
IMAGE_SERVICE_PHOTOROOM_TRAINING_OPT_OUT_CONFIRMED=true
```

Only `development` and `acceptance` may enable this gate; production refuses to
start when it is enabled. Compose passes the four settings with safe defaults
(`false`/empty), and its production overlay forcibly disables them. There is no production-mode Photoroom switch or BaseURL override;
runtime traffic is fixed to `https://image-api.photoroom.com` (userinfo, query,
fragment and foreign hosts are rejected), and the HTTP client refuses every
redirect so the API key and image bytes cannot cross hosts. It accepts blob bytes only and only
supports background removal, a white background, or AI soft shadow.

Authenticated `GET /internal/v1/processors` reports the deterministic and
Photoroom capabilities. Photoroom always reports `safety_level=sandbox_only`,
`provider_environment=sandbox`, `region=us`, `watermarked=true`, and
`non_publishable=true`. `available=true` additionally requires the four gates
and the one-shot canary quota to remain. The quota is exactly one accepted
execution for the lifetime of the persistence database; zero-cost sandbox work
still consumes it, and there is deliberately no timer or automatic reset.

Before sending any network bytes, it consumes a PostgreSQL-unique submit claim
bound to the exact job. The LingMirror execution token is separately bound to
job/task/version/manifest/operation/processor, `max_cost=0`, `currency=USD`,
`region=us`, exact sandbox/watermarked/non-publishable restrictions, and a single-use nonce. The
one-shot canary quota, nonce and durable attempt are claimed atomically. Timeout,
EOF, connection failure, and 5xx persist the job and attempt as
`RECONCILE_REQUIRED`, with the sanitized Provider request ID when available.
Because the sandbox endpoint has no reliable query operation, such work cannot
be retried. A 429 is also terminal for that intent. Successful Provider bytes
are not assumed to contain a Provider watermark. Image Service decodes them,
deterministically overlays a conspicuous opaque `SANDBOX` pixel banner,
re-encodes PNG, then decodes and verifies the exact marker pixels before
persistence. Only those locally verified bytes are content-hashed and
permanently marked `sandbox=true`, `watermarked=true`, and
`non_publishable=true`; blob downloads repeat those restrictions in
`X-Image-*` headers. Tests use loopback HTTP fixtures only; no real Photoroom
call is part of the test suite.
