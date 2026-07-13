import assert from 'node:assert/strict';
import test from 'node:test';

function storageArea(initial = {}) {
  const values = { ...initial };
  return {
    values,
    async get(keys) {
      const result = {};
      for (const key of keys) if (key in values) result[key] = values[key];
      return result;
    },
    async set(next) { Object.assign(values, next); },
    async remove(keys) {
      for (const key of Array.isArray(keys) ? keys : [keys]) delete values[key];
    },
  };
}

test('changing API origin clears device and access credentials', async () => {
  const local = storageArea({
    lingmirror_server_url: 'wss://owner.lingmirror.com',
    lingmirror_extension_device: {
      deviceId: 'device-1', deviceSecret: 'secret', environment: 'production',
      apiOrigin: 'https://owner.lingmirror.com',
    },
  });
  const session = storageArea({ lingmirror_jwt: 'short-lived-token' });
  globalThis.chrome = { storage: { local, session } };
  const { setServerUrl } = await import(`../build/shared/auth.js?case=${Date.now()}`);

  await setServerUrl('wss://new-owner.lingmirror.com');

  assert.equal(local.values.lingmirror_server_url, 'wss://new-owner.lingmirror.com');
  assert.equal(local.values.lingmirror_extension_device, undefined);
  assert.equal(session.values.lingmirror_jwt, undefined);
});

test('equivalent server URL keeps the paired device credential', async () => {
  const credential = {
    deviceId: 'device-1', deviceSecret: 'secret', environment: 'production',
    apiOrigin: 'https://owner.lingmirror.com',
  };
  const local = storageArea({
    lingmirror_server_url: 'wss://owner.lingmirror.com',
    lingmirror_extension_device: credential,
  });
  const session = storageArea({ lingmirror_jwt: 'short-lived-token' });
  globalThis.chrome = { storage: { local, session } };
  const { setServerUrl } = await import(`../build/shared/auth.js?case=${Date.now()}-same`);

  await setServerUrl('wss://owner.lingmirror.com/ws/extension');

  assert.deepEqual(local.values.lingmirror_extension_device, credential);
  assert.equal(session.values.lingmirror_jwt, 'short-lived-token');
});
