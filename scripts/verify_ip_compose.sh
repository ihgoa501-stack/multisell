#!/bin/sh
set -eu
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
OUT=$(mktemp)
trap 'rm -f "$OUT"' EXIT HUP INT TERM
env DB_USER=x DB_PASSWORD=render_password_123456789 DB_NAME=x \
  JWT_SECRET=render_jwt_secret_abcdefghijklmnopqrstuvwxyz \
  AUDIT_CHECKPOINT_KEY=render_audit_key_abcdefghijklmnopqrstuvwxyz \
  DOMAIN=unused.invalid PUBLIC_IP=118.196.42.156 \
  docker compose --profile manual \
    -f "$ROOT/docker-compose.yml" -f "$ROOT/docker-compose.prod.yml" -f "$ROOT/docker-compose.ip.yml" \
    config --format json > "$OUT"
python3 - "$OUT" <<'PY'
import json, sys
c=json.load(open(sys.argv[1], encoding="utf-8")); s=c.get("services",{}); errors=[]
for name, svc in s.items():
    ports=svc.get("ports") or []
    if name != "caddy" and ports: errors.append(f"{name} publishes ports: {ports}")
ports=s.get("caddy",{}).get("ports") or []
actual={(str(p.get("published")),str(p.get("target"))) for p in ports}
if actual != {("80","80"),("443","443")}: errors.append(f"unexpected Caddy ports: {actual}")
origins=s.get("backend",{}).get("environment",{}).get("CORS_ALLOWED_ORIGINS")
if origins != "https://118.196.42.156": errors.append(f"unexpected CORS origin: {origins}")
mounts=s.get("caddy",{}).get("volumes") or []
if not any(str(m.get("source","")).endswith("Caddyfile.ip") for m in mounts if isinstance(m,dict)): errors.append("Caddyfile.ip not mounted")
if errors:
    print("\n".join("ERROR: "+e for e in errors), file=sys.stderr); raise SystemExit(1)
print("IP compose boundary verified: only Caddy 80/443 is public")
PY
