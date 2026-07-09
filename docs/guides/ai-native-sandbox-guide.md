# AI-Native Automated Staging Sandbox and E2E Testing Guide

> **Purpose**: Guide AI coding agents on how to configure staging docker sandboxes, isolate concurrent builds, optimize compilations, and verify commits via Playwright.

---

## 1. Sandbox Architecture Overview

Since multiple AI agents can develop on the platform concurrently, we run isolated staging sandboxes on every Pull Request (PR). The sandbox ensures that:
- **No Host Port Exposure**: Database, backend, and E2E services run on an internal docker network.
- **Port Conflict Prevention**: Port collisions are resolved by namespacing the compose stack at runtime.
- **Data Isolation**: Fresh Postgres instances are seeded with deterministic datasets for each pipeline run.

---

## 2. Docker Staging Stack Configuration (`docker-compose.sandbox.yml`)

The template uses static service names. The executor isolates the stack using compose projects:
`docker compose -p sandbox-pr-${SANDBOX_ID} -f docker-compose.sandbox.yml up`

```yaml
version: '3.8'

networks:
  sandbox_internal:
    internal: true # Strict firewall cutting off internet access for database & backend
  traefik_public:
    external: true # Shared external network for frontend routing

services:
  db:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: multisell_sandbox
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    networks:
      - sandbox_internal
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d multisell_sandbox"]
      interval: 3s
      timeout: 3s
      retries: 5

  migrate:
    image: migrate/migrate:v4.18.1
    depends_on:
      db:
        condition: service_healthy
    volumes:
      - ./backend-go/migrations:/migrations:ro
    networks:
      - sandbox_internal
    command:
      - -path
      - /migrations
      - -database
      - postgresql://postgres:${DB_PASSWORD}@db:5432/multisell_sandbox?sslmode=disable
      - up

  backend:
    image: golang:1.25
    working_dir: /app
    volumes:
      - ./backend-go:/app
      - ~/.go-pkg-cache:/go/pkg/mod:ro # Read-only Go module cache sharing
    environment:
      DB_HOST: db
      DB_PORT: "5432"
      DB_USER: postgres
      DB_PASSWORD: ${DB_PASSWORD}
      DB_NAME: multisell_sandbox
      JWT_SECRET: ${JWT_SECRET}
      SERVER_PORT: "8080"
    networks:
      - sandbox_internal
    depends_on:
      db:
        condition: service_healthy
      migrate:
        condition: service_completed_successfully
    command: >
      sh -c "go mod download && go run cmd/server/main.go"

  seed:
    image: golang:1.25
    working_dir: /work
    volumes:
      - .:/work
      - ~/.go-pkg-cache:/go/pkg/mod:ro
    environment:
      DB_HOST: db
      DB_PORT: "5432"
      DB_USER: postgres
      DB_PASSWORD: ${DB_PASSWORD}
      DB_NAME: multisell_sandbox
    networks:
      - sandbox_internal
    depends_on:
      migrate:
        condition: service_completed_successfully
    command: ["bash", "scripts/e2e_seed.sh"]

  frontend:
    image: node:22
    working_dir: /app
    volumes:
      - ./frontend-next:/app
      - ~/.npm-cache:/root/.npm:ro # Read-only npm cache sharing
    environment:
      NEXT_PUBLIC_API_URL: http://backend:8080/api
    networks:
      - sandbox_internal
      - traefik_public
    depends_on:
      - backend
    command: >
      sh -c "npm ci && npm run dev -- --hostname 0.0.0.0 --port 3000"
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=traefik_public"
      - "traefik.http.routers.frontend-pr-${SANDBOX_ID}.rule=Host(`pr-${SANDBOX_ID}.staging.lingmirror.com`)"
      - "traefik.http.routers.frontend-pr-${SANDBOX_ID}.service=frontend-pr-${SANDBOX_ID}"
      - "traefik.http.services.frontend-pr-${SANDBOX_ID}.loadbalancer.server.port=3000"

  e2e:
    image: mcr.microsoft.com/playwright:v1.55.0-jammy
    working_dir: /work/frontend-next/e2e
    volumes:
      - .:/work
    environment:
      E2E_BASE_URL: http://frontend:3000
      E2E_API_BASE: http://backend:8080
      E2E_SKIP_WEB_SERVER: "1"
    networks:
      - sandbox_internal
    depends_on:
      frontend:
        condition: service_started
      backend:
        condition: service_started
      seed:
        condition: service_completed_successfully
    command: >
      bash -lc "npm ci &&
                for i in $$(seq 1 90); do
                  if curl -fsS http://frontend:3000/login >/dev/null && curl -fsS http://backend:8080/api/health >/dev/null; then
                    break
                  fi
                  sleep 1
                done &&
                npm run e2e"
```

---

## 3. Concurrency Safety Rules
When multiple sandboxes run concurrently on the same host, agents must apply the following guidelines:
1. **Isolated Workspace Worktrees**: Do not run compose volumes against the same checked-out project folder. Create separate git worktrees at `/tmp/sandboxes/pr-${SANDBOX_ID}`.
2. **Internal Network Bridging**: The `backend` container is strictly set on `sandbox_internal` (with `internal: true`). It cannot access the internet, preventing leaks. The `frontend` joins both networks to receive routing from Traefik.
3. **Traefik Network Label**: Always specify `"traefik.docker.network=traefik_public"` to prevent Traefik from routing to the wrong internal network IP (which would cause a 502 Bad Gateway).

---

## 4. Run Script Workflow (`scripts/run_sandbox.sh`)
This script automates workspace allocation, container provisioning, verification, and log collection:

```bash
#!/usr/bin/env bash
set -euo pipefail

SANDBOX_ID="${1:-}"
if [ -z "$SANDBOX_ID" ]; then
  echo "Error: Please specify a sandbox ID"
  echo "Usage: $0 <sandbox-id>"
  exit 1
fi

export SANDBOX_ID
export DB_PASSWORD=$(openssl rand -hex 16)
export JWT_SECRET=$(openssl rand -hex 32)
WORKTREE_DIR="/tmp/sandboxes/pr-${SANDBOX_ID}"

echo "Allocating worktree in $WORKTREE_DIR..."
rm -rf "$WORKTREE_DIR"
git worktree add "$WORKTREE_DIR" HEAD

cd "$WORKTREE_DIR"

echo "Spinning up isolated Docker Compose namespace sandbox..."
docker compose -p "sandbox-pr-${SANDBOX_ID}" -f docker-compose.sandbox.yml down -v --remove-orphans || true
set +e
docker compose -p "sandbox-pr-${SANDBOX_ID}" -f docker-compose.sandbox.yml up --abort-on-container-exit --exit-code-from e2e e2e
EXIT_CODE=$?
set -e

echo "Copying Playwright test reports..."
docker cp "$(docker compose -p "sandbox-pr-${SANDBOX_ID}" -f docker-compose.sandbox.yml ps -q e2e)":/work/frontend-next/e2e/playwright-report /tmp/reports/pr-${SANDBOX_ID} || true

echo "Tearing down sandbox..."
docker compose -p "sandbox-pr-${SANDBOX_ID}" -f docker-compose.sandbox.yml down -v

echo "Cleaning up worktree..."
cd /Users/lc/multisell
git worktree remove "$WORKTREE_DIR" --force

echo "Sandbox execution finished with exit code: $EXIT_CODE"
exit $EXIT_CODE
```

---

## 5. AI Self-Healing Protocol
If the sandbox execution returns a non-zero exit code:
1. **Locate Logs**: Read `/tmp/reports/pr-${SANDBOX_ID}/playwright-report` to check failures, screenshots, and videos.
2. **Analyze**: Identify if the failure is a backend code bug, frontend type issue, or mock database schema mismatch.
3. **Correct**: Apply minimal code edits, commit, and re-trigger `run_sandbox.sh`.
