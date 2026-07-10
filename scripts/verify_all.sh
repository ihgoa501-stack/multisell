#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_E2E="${RUN_E2E:-1}"

section() {
  printf '\n==> %s\n' "$1"
}

section "Repository status"
cd "$ROOT_DIR"
git status --short --branch

section "Backend build"
cd "$ROOT_DIR/backend-go"
go build ./...

section "Backend vet"
go vet ./...

section "Backend tests"
go test ./...

section "Frontend unit tests"
cd "$ROOT_DIR/frontend-next"
npm test

section "Frontend lint"
npm run lint

section "Frontend build"
npm run build

if [[ "$RUN_E2E" == "1" ]]; then
  section "Frontend E2E"
  cd "$ROOT_DIR/frontend-next/e2e"
  npm run e2e
else
  section "Frontend E2E skipped"
  echo "RUN_E2E=$RUN_E2E"
fi

section "Verification complete"
