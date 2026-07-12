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
  DOMAIN=unused.invalid \
  PRIVATE_PORT=8088 \
  docker compose \
    --profile manual \
    -f "$ROOT/docker-compose.yml" \
    -f "$ROOT/docker-compose.prod.yml" \
    -f "$ROOT/docker-compose.private.yml" \
    config --format json > "$RENDERED"

python3 - "$RENDERED" <<'PY'
import json, sys

with open(sys.argv[1], encoding="utf-8") as fh:
    config = json.load(fh)

services = config.get("services", {})
errors = []
for name, service in services.items():
    ports = service.get("ports") or []
    if name != "caddy" and ports:
        errors.append(f"{name} must not publish host ports: {ports}")

ports = services.get("caddy", {}).get("ports") or []
if len(ports) != 1:
    errors.append(f"caddy must publish exactly one loopback port: {ports}")
else:
    port = ports[0]
    if str(port.get("host_ip")) != "127.0.0.1" or str(port.get("published")) != "8088" or str(port.get("target")) != "8088":
        errors.append(f"private entry point must be 127.0.0.1:8088->8088: {port}")

backend_env = services.get("backend", {}).get("environment", {})
origins = backend_env.get("CORS_ALLOWED_ORIGINS", "")
if "127.0.0.1:8088" not in origins or "localhost:8088" not in origins or "https://" in origins:
    errors.append(f"private CORS origins are invalid: {origins}")

for name in ("backup", "audit-checkpoint"):
    env = services.get(name, {}).get("environment", {})
    prefix = "BACKUP" if name == "backup" else "AUDIT_CHECKPOINT"
    if str(env.get(f"{prefix}_REQUIRE_OFFSITE", "")).lower() != "false":
        errors.append(f"{name} private mode must explicitly record offsite as unavailable")

mounts = services.get("caddy", {}).get("volumes") or []
if not any(str(m.get("source", "")).endswith("Caddyfile.private") for m in mounts if isinstance(m, dict)):
    errors.append("caddy must mount Caddyfile.private")

if errors:
    print("\n".join(f"ERROR: {e}" for e in errors), file=sys.stderr)
    raise SystemExit(1)
print("private compose boundary verified: only 127.0.0.1:8088 is published")
PY
