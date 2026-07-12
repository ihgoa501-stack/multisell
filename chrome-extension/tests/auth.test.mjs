import assert from 'node:assert/strict';
import test from 'node:test';

import { getLoginUrl } from '../build/shared/auth.js';

test('local extension login targets the frontend auth bridge', () => {
  assert.equal(
    getLoginUrl('ws://localhost:8080'),
    'http://localhost:3000/login?extension_auth=1',
  );
});

test('production extension login preserves the configured origin', () => {
  assert.equal(
    getLoginUrl('wss://owner.lingmirror.com'),
    'https://owner.lingmirror.com/login?extension_auth=1',
  );
});
