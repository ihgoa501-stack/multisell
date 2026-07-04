#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# LingMirror / MultiSell -- Smoke Test Setup
#
# Builds the server, ensures DB and migrations, seeds data, starts the
# server in background, then runs the smoke test.
#
# Usage:
#   bash backend-go/scripts/smoke_test_setup.sh
#   bash backend-go/scripts/smoke_test_setup.sh --help
#
# Environment:
#   DB_HOST       database host   (default: localhost)
#   DB_PORT       database port   (default: 5432)
#   DB_USER       database user   (default: postgres)
#   DB_PASSWORD   database pass   (default: postgres)
#   DB_NAME       database name   (default: multisell)
#   SERVER_PORT   server port     (default: 8080)
#   SMOKE_USER    test user       (default: smoke_test_user)
#   SMOKE_PASS    test password   (default: smoke_test_pass123)
#   SKIP_BUILD    set to "1" to skip go build
#   SKIP_DB       set to "1" to skip docker/db checks
#   SKIP_MIGRATE  set to "1" to skip migration
# ---------------------------------------------------------------------------
set -euo pipefail

# Resolve project root
SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$SCRIPT_DIR"

# Defaults
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"
DB_NAME="${DB_NAME:-multisell}"
SERVER_PORT="${SERVER_PORT:-8080}"

DATABASE_URL="postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

# Colors
RED='\033[0;31m'
GRN='\033[0;32m'
CYN='\033[0;36m'
NC='\033[0m'

# ---------------------------------------------------------------------------
# Help
# ---------------------------------------------------------------------------
if [ "${1:-}" = "--help" ] || [ "${1:-}" = "-h" ]; then
  cat <<'HELP'
LingMirror / MultiSell -- Smoke Test Setup

  Builds the binary, ensures DB is up, runs migrations, seeds seed data,
  starts the server, then invokes the smoke test.

  Usage: bash backend-go/scripts/smoke_test_setup.sh [--skip-build] [--skip-db]

  Flags:
    --skip-build  don't rebuild the binary (use existing)
    --skip-db     don't check docker / database connectivity
    --skip-migrate  skip migration step

  Env vars:
    DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME
    SERVER_PORT
    SMOKE_USER, SMOKE_PASS
    SKIP_BUILD=1, SKIP_DB=1, SKIP_MIGRATE=1
HELP
  exit 0
fi

# Parse flags
for arg in "$@"; do
  case "$arg" in
    --skip-build) SKIP_BUILD=1 ;;
    --skip-db) SKIP_DB=1 ;;
    --skip-migrate) SKIP_MIGRATE=1 ;;
  esac
done

# ---------------------------------------------------------------------------
# 0. Check dependencies
# ---------------------------------------------------------------------------
echo ""
echo "╔══════════════════════════════════════════════╗"
echo "║   LingMirror Smoke Test Setup                ║"
echo "╚══════════════════════════════════════════════╝"

MISSING=""
command -v go >/dev/null 2>&1 || MISSING="$MISSING go"
command -v curl >/dev/null 2>&1 || MISSING="$MISSING curl"
command -v psql >/dev/null 2>&1 || { echo "  (psql not found — install postgresql client for migration checks)"; }
if [ -n "$MISSING" ]; then
  echo -e "  ${RED}Missing dependencies:$MISSING${NC}"
  exit 1
fi
echo "  Go:    $(go version 2>/dev/null | head -c 35 || echo '?')"
echo "  Dir:   $SCRIPT_DIR"
echo ""

# ---------------------------------------------------------------------------
# 1. Ensure PostgreSQL is running (via docker if possible)
# ---------------------------------------------------------------------------
ensure_db() {
  # Try direct psql connection
  if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1" >/dev/null 2>&1; then
    echo "  Database already accessible"
    return 0
  fi

  # Try docker
  if command -v docker >/dev/null 2>&1; then
    echo "  Starting PostgreSQL via Docker..."
    docker compose up -d db 2>/dev/null || docker-compose up -d db 2>/dev/null || {
      echo -e "  ${RED}Docker Compose not found; start PostgreSQL manually${NC}"
      return 1
    }

    # Wait for healthy
    for i in $(seq 1 30); do
      if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1" >/dev/null 2>&1; then
        echo "  Database ready after ${i}s"
        return 0
      fi
      sleep 1
    done

    echo -e "  ${RED}Database not ready after 30s${NC}"
    return 1
  fi

  echo -e "  ${RED}Cannot connect to PostgreSQL at $DB_HOST:$DB_PORT${NC}"
  return 1
}

if [ "${SKIP_DB:-0}" != "1" ]; then
  echo "--- 1. Database ---"
  ensure_db
  echo ""
else
  echo "--- 1. Database (skipped) ---"
  echo ""
fi

# ---------------------------------------------------------------------------
# 2. Run migrations
# ---------------------------------------------------------------------------
if [ "${SKIP_MIGRATE:-0}" != "1" ]; then
  echo "--- 2. Migrations ---"
  if command -v migrate >/dev/null 2>&1; then
    echo "  Running migrations via migrate CLI..."
    migrate -path "$SCRIPT_DIR/backend-go/migrations" -database "$DATABASE_URL" up 2>&1 || {
      echo "  (migrate may have returned errors for already-applied migrations)"
    }
  else
    # Fallback: run .up.sql files via psql
    echo "  migrate CLI not found; applying SQL via psql..."
    for f in "$SCRIPT_DIR"/backend-go/migrations/*.up.sql; do
      [ -f "$f" ] || continue
      name=$(basename "$f")
      echo "    $name"
      PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$f" >/dev/null 2>&1 || true
    done
  fi
  echo ""
else
  echo "--- 2. Migrations (skipped) ---"
  echo ""
fi

# ---------------------------------------------------------------------------
# 3. Build server binary
# ---------------------------------------------------------------------------
if [ "${SKIP_BUILD:-0}" != "1" ]; then
  echo "--- 3. Build ---"
  echo "  Building bin/server..."
  (cd "$SCRIPT_DIR/backend-go" && go build -o bin/server cmd/server/main.go) || {
    echo -e "  ${RED}Build failed${NC}"
    exit 1
  }
  echo -e "  ${GRN}Build complete${NC}"
  echo ""
else
  echo "--- 3. Build (skipped) ---"
  echo ""
fi

# ---------------------------------------------------------------------------
# 4. Start server in background
# ---------------------------------------------------------------------------
echo "--- 4. Start server ---"

# Kill any existing process on the target port
if lsof -ti ":$SERVER_PORT" >/dev/null 2>&1; then
  echo "  Port $SERVER_PORT in use — stopping existing process"
  lsof -ti ":$SERVER_PORT" | xargs kill 2>/dev/null || true
  sleep 2
fi

# Export env vars for the server
export DB_HOST DB_PORT DB_USER DB_PASSWORD DB_NAME
export JWT_SECRET="${JWT_SECRET:-dev-secret-change-in-production}"
export SERVER_PORT
export GIN_MODE="${GIN_MODE:-debug}"

echo "  Starting server on port $SERVER_PORT..."
(cd "$SCRIPT_DIR/backend-go" && nohup ./bin/server > "$SCRIPT_DIR/backend-go/scripts/smoke_server.log" 2>&1) &
SERVER_PID=$!

# Write PID to a file for cleanup
echo "$SERVER_PID" > "$SCRIPT_DIR/backend-go/scripts/smoke_server.pid"
echo "  PID: $SERVER_PID"
echo "  Logs: backend-go/scripts/smoke_server.log"
echo ""

# ---------------------------------------------------------------------------
# 5. Wait for server readiness then run smoke test
# ---------------------------------------------------------------------------
echo "--- 5. Smoke test ---"

SMOKE_SCRIPT="$SCRIPT_DIR/backend-go/scripts/smoke_test.sh"
if [ ! -f "$SMOKE_SCRIPT" ]; then
  echo -e "  ${RED}smoke_test.sh not found at $SMOKE_SCRIPT${NC}"
  exit 1
fi

# Export smoke test config
export BASE_URL="http://localhost:$SERVER_PORT"
export SMOKE_USER SMOKE_PASS

# Wait for server
echo "  Waiting for server..."
for i in $(seq 1 30); do
  if curl -sf "$BASE_URL/api/health" >/dev/null 2>&1; then
    echo "  Server ready after ${i}s"
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo -e "  ${RED}Server didn't start within 30s${NC}"
    echo "  Last log lines:"
    tail -5 "$SCRIPT_DIR/backend-go/scripts/smoke_server.log" 2>/dev/null || true
    exit 1
  fi
  sleep 1
done

# Run the smoke test
bash "$SMOKE_SCRIPT"
SMOKE_EXIT=$?

if [ "$SMOKE_EXIT" -eq 0 ]; then
  echo -e "${GRN}Smoke test exited with code $SMOKE_EXIT${NC}"
else
  echo -e "${RED}Smoke test exited with code $SMOKE_EXIT${NC}"
fi

# Stop the server
echo ""
echo "--- Cleanup ---"
echo "  Stopping server (PID $SERVER_PID)..."
kill "$SERVER_PID" 2>/dev/null || true
rm -f "$SCRIPT_DIR/backend-go/scripts/smoke_server.pid"
echo "  Logs preserved at: backend-go/scripts/smoke_server.log"

exit "$SMOKE_EXIT"
