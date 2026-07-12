import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

test('manifest is limited to the Owner 1688 private collection scope', async () => {
  const manifest = JSON.parse(await readFile(new URL('../manifest.json', import.meta.url), 'utf8'));
  assert.equal(manifest.name, '凌镜 1688 采集助手');
  assert.equal(manifest.version, '0.2.0');
  assert.deepEqual(manifest.permissions, ['storage', 'activeTab', 'alarms']);
  assert.ok(manifest.host_permissions.includes('https://detail.1688.com/*'));
  assert.ok(manifest.host_permissions.includes('https://*.1688.com/*'));
  assert.equal(manifest.host_permissions.some((value) => (value.includes('*.') && value !== 'https://*.1688.com/*') || value === '<all_urls>'), false);
  assert.equal(manifest.host_permissions.some((value) => value.includes('ozon') || value.includes('taobao')), false);
  const listScriptEntry = manifest.content_scripts.find((entry) => entry.js.includes('build/content-script-list.js'));
  assert.ok(listScriptEntry);
  assert.deepEqual(listScriptEntry.matches, ['https://www.1688.com/*', 'https://s.1688.com/*', 'https://*.1688.com/*']);
  assert.deepEqual(
    manifest.content_scripts.flatMap((entry) => entry.matches).filter((value) => value.includes('lingmirror')),
    ['https://lingmirror.com/settings/plugin*', 'https://owner.lingmirror.com/settings/plugin*'],
  );
});

test('compiled content script has no page-load auto upload path', async () => {
  const source = await readFile(new URL('../build/content-script.js', import.meta.url), 'utf8');
  assert.equal(source.includes('autoExtract'), false);
  assert.equal(source.includes('requestId: "auto_"'), false);
  assert.equal(source.includes('submitPrivateCollection('), false);
  assert.equal(source.includes('采集到凌镜'), true);
});

test('manifest contains correct icon paths and action default_icon configurations', async () => {
  const manifest = JSON.parse(await readFile(new URL('../manifest.json', import.meta.url), 'utf8'));
  assert.deepEqual(manifest.icons, {
    "16": "icons/icon16.png",
    "48": "icons/icon48.png",
    "128": "icons/icon128.png"
  });
  assert.deepEqual(manifest.action?.default_icon, {
    "16": "icons/icon16.png",
    "48": "icons/icon48.png",
    "128": "icons/icon128.png"
  });
});
