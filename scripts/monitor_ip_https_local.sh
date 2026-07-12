#!/bin/sh
set -u

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
PUBLIC_IP=${PUBLIC_IP:-118.196.42.156}

if PUBLIC_IP="$PUBLIC_IP" "$ROOT/scripts/check_ip_https.sh"; then
  exit 0
fi

message="LingMirror HTTPS、证书或 readiness 检查失败：${PUBLIC_IP}"
/usr/bin/osascript -e "display notification \"$message\" with title \"LingMirror 运维告警\" sound name \"Basso\"" >/dev/null 2>&1 || true
echo "$message" >&2
exit 1
