#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
RENDERED=$(mktemp)
WEBHOOK=$(mktemp)
trap 'rm -f "$RENDERED" "$WEBHOOK"' EXIT HUP INT TERM
printf 'https://example.invalid/render-only\n' > "$WEBHOOK"

env \
  DB_USER=render_user \
  DB_PASSWORD=render_password \
  DB_NAME=render_db \
  JWT_SECRET=render_only_secret_abcdefghijklmnopqrstuvwxyz \
  IMAGE_SERVICE_SHARED_SECRET=render_only_image_shared_secret_abcdefghijklmnopqrstuvwxyz \
  IMAGE_SERVICE_EXECUTION_TOKEN_SECRET=render_only_image_execution_secret_abcdefghijklmnopqrstuvwxyz \
  IMAGE_RELEASE_ATTESTATION_SECRET=render_only_image_attestation_secret_abcdefghijklmnopqrstuvwxyz \
  IMAGE_SERVICE_DATABASE_URL=postgresql://render_user:render_password@db:5432/render_db?sslmode=disable \
  AUDIT_CHECKPOINT_KEY=render_only_audit_checkpoint_key \
  GRAFANA_ADMIN_PASSWORD=render_only_grafana_password \
  ALERTMANAGER_SLACK_WEBHOOK_FILE="$WEBHOOK" \
  DOMAIN=example.invalid \
  docker compose --profile manual \
    -f "$ROOT/docker-compose.yml" \
    -f "$ROOT/docker-compose.prod.yml" \
    -f "$ROOT/docker-compose.monitoring.yml" \
    config --format json > "$RENDERED"

python3 - "$RENDERED" <<'PY'
import json, sys

with open(sys.argv[1], encoding="utf-8") as fh:
    services = json.load(fh).get("services", {})

errors = []
for name in ("prometheus", "grafana"):
    ports = services.get(name, {}).get("ports") or []
    if len(ports) != 1 or str(ports[0].get("host_ip")) != "127.0.0.1":
        errors.append(f"{name} must publish exactly one loopback-only port: {ports}")
if services.get("alertmanager", {}).get("ports"):
    errors.append("alertmanager must not publish a host port")

for name in ("prometheus", "grafana", "alertmanager"):
    if not services.get(name, {}).get("healthcheck"):
        errors.append(f"{name} must define a container healthcheck")

backend_env = services.get("backend", {}).get("environment", {})
if str(backend_env.get("METRICS_ENABLED", "")).lower() != "true":
    errors.append("backend metrics must be enabled when the monitoring stack is rendered")

prometheus_command = " ".join(services.get("prometheus", {}).get("command") or [])
if "prometheus.yml" not in prometheus_command:
    errors.append("prometheus must load the governed configuration")
alertmanager_command = " ".join(services.get("alertmanager", {}).get("command") or [])
if "alertmanager.yml" not in alertmanager_command:
    errors.append("alertmanager must load the governed configuration")

if errors:
    print("\n".join(f"ERROR: {error}" for error in errors), file=sys.stderr)
    raise SystemExit(1)
print("monitoring compose boundary verified: loopback dashboards, private alerts, healthchecks enabled")
PY
