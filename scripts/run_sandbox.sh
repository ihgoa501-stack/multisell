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

echo "==> Preparing cache directories..."
mkdir -p "$HOME/.go-pkg-cache" "$HOME/.npm-cache" "/tmp/reports/pr-${SANDBOX_ID}"

echo "==> Allocating git worktree in $WORKTREE_DIR..."
# Remove any residual directory or worktree registration
rm -rf "$WORKTREE_DIR"
git worktree prune || true
git worktree add "$WORKTREE_DIR" HEAD

cd "$WORKTREE_DIR"

# Ensure the external network for Traefik routing exists on the host
echo "==> Checking external docker network 'traefik_public'..."
docker network inspect traefik_public >/dev/null 2>&1 || docker network create traefik_public

echo "==> Spinning up isolated Docker Compose sandbox..."
# Make sure we clean up any previous compose leftovers with the same project name
docker compose -p "sandbox-pr-${SANDBOX_ID}" -f docker-compose.sandbox.yml down -v --remove-orphans || true

# Run compose up and wait for the E2E test container to exit
set +e
docker compose -p "sandbox-pr-${SANDBOX_ID}" -f docker-compose.sandbox.yml up --abort-on-container-exit --exit-code-from e2e e2e
EXIT_CODE=$?
set -e

echo "==> Copying Playwright test reports..."
CONTAINER_ID=$(docker compose -p "sandbox-pr-${SANDBOX_ID}" -f docker-compose.sandbox.yml ps -q e2e)
if [ -n "$CONTAINER_ID" ]; then
  docker cp "${CONTAINER_ID}:/work/frontend-next/e2e/playwright-report" "/tmp/reports/pr-${SANDBOX_ID}" || true
else
  echo "Warning: e2e container not found, skipping report copy"
fi

echo "==> Tearing down sandbox..."
docker compose -p "sandbox-pr-${SANDBOX_ID}" -f docker-compose.sandbox.yml down -v

echo "==> Cleaning up worktree..."
cd /Users/lc/multisell
git worktree remove "$WORKTREE_DIR" --force

echo "==> Sandbox execution finished with exit code: $EXIT_CODE"
exit $EXIT_CODE
