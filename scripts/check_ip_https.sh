#!/bin/sh
set -eu

PUBLIC_IP=${PUBLIC_IP:?PUBLIC_IP is required}
WARN_SECONDS=${CERT_WARN_SECONDS:-172800}

cert=$(printf '\n' | openssl s_client -connect "${PUBLIC_IP}:443" 2>/dev/null | openssl x509 -outform PEM)
[ -n "$cert" ] || { echo "certificate unavailable" >&2; exit 1; }
printf '%s\n' "$cert" | openssl x509 -checkend "$WARN_SECONDS" -noout >/dev/null || {
  echo "IP certificate expires within ${WARN_SECONDS} seconds" >&2
  exit 1
}
printf '%s\n' "$cert" | openssl x509 -text -noout | grep -q "IP Address:${PUBLIC_IP}" || {
  echo "certificate SAN does not match ${PUBLIC_IP}" >&2
  exit 1
}

curl --fail --silent --show-error --max-time 10 "https://${PUBLIC_IP}/api/health" >/dev/null
curl --fail --silent --show-error --max-time 10 "https://${PUBLIC_IP}/api/ready" >/dev/null
echo "IP HTTPS certificate and readiness verified"
