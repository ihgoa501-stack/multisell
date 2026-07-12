#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT HUP INT TERM
mkdir -p "$TMP/bin"

cat > "$TMP/bin/curl" <<'SH'
#!/bin/sh
: "${FAKE_CURL_BODY:?}"
expect_body=false
for arg in "$@"; do
  if [ "$expect_body" = "true" ]; then
    printf '%s' "$arg" > "$FAKE_CURL_BODY"
    expect_body=false
  elif [ "$arg" = "--data" ]; then
    expect_body=true
  fi
done
[ -s "$FAKE_CURL_BODY" ]
SH
chmod +x "$TMP/bin/curl"

if PATH="$TMP/bin:$PATH" OPS_ALERT_WEBHOOK_URL= \
  "$ROOT/scripts/notify_ops_failure.sh" test.service >/dev/null 2>&1; then
  echo "notification unexpectedly succeeded without webhook URL" >&2
  exit 1
fi

PATH="$TMP/bin:$PATH" FAKE_CURL_BODY="$TMP/body" \
  OPS_ALERT_WEBHOOK_URL=https://alerts.example.invalid/hook \
  "$ROOT/scripts/notify_ops_failure.sh" 'unit"quoted.service'

BODY_FILE="$TMP/body" python3 -c '
import json, os
payload = json.load(open(os.environ["BODY_FILE"]))
assert "unit=unit\"quoted.service" in payload["text"], payload
'

echo "operations failure webhook contract verified"
