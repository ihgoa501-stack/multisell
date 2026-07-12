#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
prism_root="${PRISM_ROOT:-/Users/lc/prism}"
inventory="$repo_root/docs/research/prism-to-image-service-inventory-2026-07-12.md"

fail=0

if [[ ! -f "$inventory" ]]; then
  echo "ERROR: inventory document not found: $inventory" >&2
  exit 1
fi
if [[ ! -d "$prism_root" ]]; then
  echo "ERROR: Prism repository not found: $prism_root" >&2
  exit 1
fi
if ! command -v rg >/dev/null 2>&1; then
  echo "ERROR: rg is required" >&2
  exit 1
fi

check_reference() {
  local reference="$1"
  if ! rg -F -q -- "$reference" "$inventory"; then
    echo "UNINVENTORIED: $reference" >&2
    fail=1
  fi
}

# MultiSell: every source/config/frontend file that names Prism is in scope.
# The complete imagegen module is also in scope because its types/tables do not
# all contain the word Prism but overlap the successor product-image domain.
while IFS= read -r path; do
  [[ -n "$path" ]] && check_reference "$path"
done < <(
  cd "$repo_root"
  {
    rg -l -i 'prism|trigger-prism|prism_' backend-go frontend-next \
      --glob '!backend-go/docs/auto/**' \
      --glob '!**/node_modules/**' \
      --glob '!**/*.sum'
    rg --files backend-go/internal/domain/imagegen
  } | sort -u
)

# Standalone Prism: all maintained files except dependency checksums must have
# an explicit disposition. Absolute references make repository ownership clear.
while IFS= read -r relative; do
  [[ -n "$relative" ]] && check_reference "$prism_root/$relative"
done < <(
  cd "$prism_root"
  rg --files -g '!go.sum' | sort
)

# Known persistent objects must not disappear from the migration narrative.
for table in product_image_gen product_canvases prompt_template; do
  check_reference "$table"
done

if (( fail != 0 )); then
  echo "Prism migration inventory check failed." >&2
  exit 1
fi

echo "Prism migration inventory is complete for current files."
