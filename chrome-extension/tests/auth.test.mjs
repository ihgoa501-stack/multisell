import assert from 'node:assert/strict';
import test from 'node:test';

import { getJWT, getLoginUrl } from '../build/shared/auth.js';

function unsignedToken(payload) {
  return `e30.${Buffer.from(JSON.stringify(payload)).toString('base64url')}.sig`;
}

test('local extension login targets the frontend auth bridge', () => {
  assert.equal(
    getLoginUrl('ws://localhost:8080'),
    'http://localhost:3000/settings/plugin',
  );
});

test('production extension login preserves the configured origin', () => {
  assert.equal(
    getLoginUrl('wss://owner.lingmirror.com'),
    'https://owner.lingmirror.com/settings/plugin',
  );
});

test('current owner IP login targets its paired settings page', () => {
  assert.equal(
    getLoginUrl('wss://118.196.42.156'),
    'https://118.196.42.156/settings/plugin',
  );
});

test('expired session access token is refreshed from the paired device instead of appearing connected', async () => {
  const expired = unsignedToken({ type: 'extension_access', exp: Math.floor(Date.now() / 1000) - 10 });
  const refreshed = unsignedToken({ type: 'extension_access', exp: Math.floor(Date.now() / 1000) + 900 });
  const sessionValues = { lingmirror_jwt: expired };
  const localValues = {
    lingmirror_server_url: 'wss://owner.lingmirror.com',
    lingmirror_extension_device: { deviceId: 'device-1', deviceSecret: 'secret', environment: 'production', apiOrigin: 'https://owner.lingmirror.com' },
  };
  globalThis.chrome = { storage: {
    session: {
      get: async () => ({ ...sessionValues }),
      set: async (values) => Object.assign(sessionValues, values),
      remove: async (key) => { delete sessionValues[key]; },
    },
    local: {
      get: async () => ({ ...localValues }),
      set: async (values) => Object.assign(localValues, values),
      remove: async (key) => { delete localValues[key]; },
    },
  } };
  let refreshCalls = 0;
  globalThis.fetch = async () => {
    refreshCalls += 1;
    return new Response(JSON.stringify({ data: { access_token: refreshed } }), { status: 200, headers: { 'Content-Type': 'application/json' } });
  };
  assert.equal(await getJWT(), refreshed);
  assert.equal(refreshCalls, 1);
  assert.equal(sessionValues.lingmirror_jwt, refreshed);
});
