#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)

grep -Eq '^[[:space:]]+send_resolved:[[:space:]]+true[[:space:]]*$' \
  "$ROOT/deploy/alertmanager/alertmanager.yml"

for unit in lingmirror-backup.service lingmirror-audit-checkpoint.service; do
  grep -qx 'WorkingDirectory=/opt/multisell' "$ROOT/ops/systemd/$unit"
done

echo "operations delivery config contracts verified"
