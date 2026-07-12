#!/bin/sh
set -eu

: "${OPS_ALERT_WEBHOOK_URL:?OPS_ALERT_WEBHOOK_URL is required}"
unit=${1:-unknown}
host=$(hostname)
json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}
payload=$(printf '{"text":"LingMirror operations failure: unit=%s host=%s"}' \
  "$(json_escape "$unit")" "$(json_escape "$host")")
curl --fail --silent --show-error --max-time 15 \
  -H 'Content-Type: application/json' \
  --data "$payload" "$OPS_ALERT_WEBHOOK_URL" >/dev/null
