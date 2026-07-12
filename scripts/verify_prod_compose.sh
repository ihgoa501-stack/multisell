#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
RENDERED=$(mktemp)
trap 'rm -f "$RENDERED"' EXIT HUP INT TERM

env \
  DB_USER=render_user \
  DB_PASSWORD=render_password \
  DB_NAME=render_db \
  JWT_SECRET=render_only_secret_abcdefghijklmnopqrstuvwxyz \
  AUDIT_CHECKPOINT_KEY=render_only_audit_checkpoint_key \
  DOMAIN=example.invalid \
  docker compose \
    --profile manual \
    -f "$ROOT/docker-compose.yml" \
    -f "$ROOT/docker-compose.prod.yml" \
    config --format json > "$RENDERED"

python3 - "$RENDERED" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as fh:
    config = json.load(fh)

services = config.get("services", {})
errors = []

for name in ("db", "backend", "frontend", "backup", "audit-checkpoint"):
    service = services.get(name, {})
    if service.get("ports"):
        errors.append(f"{name} must not publish host ports: {service['ports']}")

for name in ("backend", "frontend"):
    service = services.get(name, {})
    if service.get("volumes"):
        errors.append(f"{name} must not inherit development source mounts")
    if service.get("command"):
        errors.append(f"{name} must use its production image CMD, not a development command")

caddy_ports = {(str(p.get("published")), str(p.get("target"))) for p in services.get("caddy", {}).get("ports", [])}
if caddy_ports != {("80", "80"), ("443", "443")}:
    errors.append(f"caddy must be the only 80/443 entry point: {sorted(caddy_ports)}")

backend_env = services.get("backend", {}).get("environment", {})
if backend_env.get("SERVER_MODE") != "release":
    errors.append("backend SERVER_MODE must be release")
if backend_env.get("CORS_ALLOWED_ORIGINS") in (None, "", "*"):
    errors.append("production CORS_ALLOWED_ORIGINS must be explicit")
if str(backend_env.get("METRICS_ENABLED", "")).lower() != "true":
    errors.append("production metrics must be enabled for Prometheus alerting")
if str(backend_env.get("AUTH_REGISTRATION_ENABLED", "")).lower() != "false":
    errors.append("production public registration must be disabled")
if str(backend_env.get("SWAGGER_ENABLED", "")).lower() != "false":
    errors.append("production public Swagger must be disabled")

backup_env = services.get("backup", {}).get("environment", {})
if str(backup_env.get("BACKUP_REQUIRE_OFFSITE", "")).lower() != "true":
    errors.append("production backup must require offsite storage")
if str(backup_env.get("BACKUP_REQUIRE_IMMUTABLE_OFFSITE", "")).lower() != "true":
    errors.append("production backup must require immutable offsite storage")

checkpoint_env = services.get("audit-checkpoint", {}).get("environment", {})
if str(checkpoint_env.get("AUDIT_CHECKPOINT_REQUIRE_OFFSITE", "")).lower() != "true":
    errors.append("production audit checkpoint must require offsite storage")
if str(checkpoint_env.get("AUDIT_CHECKPOINT_REQUIRE_IMMUTABLE_OFFSITE", "")).lower() != "true":
    errors.append("production audit checkpoint must require immutable offsite storage")

db_env = services.get("db", {}).get("environment", {})
if db_env.get("POSTGRES_DB") != backend_env.get("DB_NAME"):
    errors.append("database POSTGRES_DB and backend DB_NAME must match")
if db_env.get("POSTGRES_USER") != backend_env.get("DB_USER"):
    errors.append("database POSTGRES_USER and backend DB_USER must match")

if errors:
    for error in errors:
        print(f"ERROR: {error}", file=sys.stderr)
    raise SystemExit(1)

print("production compose boundary verified: only Caddy publishes 80/443")
PY
