#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> Seeding deterministic E2E data"
cd "$ROOT_DIR/backend-go"
go run scripts/demo_seed.go
echo "==> E2E seed complete"
