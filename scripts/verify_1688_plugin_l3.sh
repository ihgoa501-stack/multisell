#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

node - "$ROOT" <<'NODE'
const fs = require('node:fs');
const path = require('node:path');

const root = process.argv[2];
const read = (name) => fs.readFileSync(path.join(root, name), 'utf8');
const manifest = JSON.parse(read('chrome-extension/manifest.json'));
const pkg = JSON.parse(read('chrome-extension/package.json'));
const lock = JSON.parse(read('chrome-extension/package-lock.json'));
const detail = read('chrome-extension/content-script.ts');
const list = read('chrome-extension/content-script-list.ts');
const backend = read('backend-go/internal/domain/sourcing1688/private_collection.go');

const failures = [];
const requireContract = (condition, message) => { if (!condition) failures.push(message); };
requireContract(manifest.version === pkg.version, `manifest ${manifest.version} != package ${pkg.version}`);
requireContract(lock.version === pkg.version && lock.packages?.['']?.version === pkg.version, 'package-lock version differs from package.json');
requireContract(detail.includes('sourcing1688.private.v1') && list.includes('sourcing1688.private.v1'), 'detail/list schema version differs');
requireContract(detail.includes(`lingmirror-extension@${pkg.version}`), 'detail parser version differs from extension version');
requireContract(backend.includes('sourcing1688.private.v1'), 'backend does not accept the private collection schema');
requireContract(backend.includes(`HasPrefix(in.ExtensionVersion, "${pkg.version.split('.').slice(0, 2).join('.')}.")`), 'backend extension compatibility differs from manifest version');
requireContract(manifest.content_scripts?.some((entry) => entry.js?.includes('build/auth-bridge.js')), 'pairing bridge is not registered');
requireContract(manifest.content_scripts?.some((entry) => entry.js?.includes('build/content-script.js')), 'detail collector is not registered');
requireContract(manifest.content_scripts?.some((entry) => entry.js?.includes('build/content-script-list.js')), 'list collector is not registered');
requireContract(!manifest.host_permissions?.includes('<all_urls>'), 'extension requests <all_urls>');

if (failures.length) {
  for (const failure of failures) console.error(`ERROR: ${failure}`);
  process.exit(1);
}
console.log(`Version contract verified: extension ${pkg.version}, schema sourcing1688.private.v1`);
NODE

(cd "$ROOT/chrome-extension" && npm test)
(cd "$ROOT/frontend-next" && npm test -- \
  'src/app/(main)/settings/plugin/page.test.tsx' \
  'src/app/(main)/sourcing1688/page.test.ts')
(cd "$ROOT/backend-go" && go test ./internal/auth ./internal/domain/sourcing1688)

echo "L3 automated gate passed. Real Chrome gate is still required; follow docs/features/1688-plugin-l3-acceptance.md."
