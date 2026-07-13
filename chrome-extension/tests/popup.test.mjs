import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';
import { parseHTML } from 'linkedom';

const here = dirname(fileURLToPath(import.meta.url));
const extensionRoot = resolve(here, '..');

function generateValidToken() {
  const exp = Math.floor(Date.now() / 1000) + 3600;
  const payload = JSON.stringify({ type: 'extension_access', exp });
  const base64Payload = Buffer.from(payload).toString('base64url');
  return `header.${base64Payload}.signature`;
}

// Helper to set up mock environment and import popup.js
async function runPopupTest(status, storageValues = {}, sessionValues = {}) {
  const html = await readFile(resolve(extensionRoot, 'popup.html'), 'utf8');
  const { window } = parseHTML(html);

  // Set up globals that popup.js expects
  globalThis.window = window;
  globalThis.document = window.document;

  const promptCalls = [];
  globalThis.prompt = (...args) => {
    promptCalls.push(args);
    return null;
  };

  const sentMessages = [];
  const tabCreated = [];
  const localValues = { ...storageValues };
  const sValues = { ...sessionValues };

  globalThis.chrome = {
    runtime: {
      sendMessage: (msg, callback) => {
        sentMessages.push(msg);
        if (msg.type === 'get_status' && callback) {
          callback({ type: 'connection_status', status });
        }
      },
      onMessage: {
        addListener: () => {},
      },
    },
    storage: {
      local: {
        get: async (keys) => {
          const result = {};
          const keyArray = Array.isArray(keys) ? keys : [keys];
          for (const key of keyArray) {
            if (key in localValues) result[key] = localValues[key];
          }
          return result;
        },
        set: async (values) => {
          Object.assign(localValues, values);
        },
        remove: async (keys) => {
          const keyArray = Array.isArray(keys) ? keys : [keys];
          for (const key of keyArray) delete localValues[key];
        },
      },
      session: {
        get: async (keys) => {
          const result = {};
          const keyArray = Array.isArray(keys) ? keys : [keys];
          for (const key of keyArray) {
            if (key in sValues) result[key] = sValues[key];
          }
          return result;
        },
        set: async (values) => {
          Object.assign(sValues, values);
        },
        remove: async (keys) => {
          const keyArray = Array.isArray(keys) ? keys : [keys];
          for (const key of keyArray) delete sValues[key];
        },
      },
    },
    tabs: {
      create: (opts) => {
        tabCreated.push(opts);
      },
      query: async () => [],
    },
  };

  // Import the compiled popup.js dynamically with cache buster
  const uniqueId = Math.random().toString(36).slice(2);
  await import(`../build/popup.js?cache=${uniqueId}`);

  return { window, sentMessages, tabCreated, promptCalls };
}

test('popup displays boxBtn when status is connected and hides it otherwise', async () => {
  const { window } = await runPopupTest('connected', {}, {
    lingmirror_jwt: generateValidToken(),
  });

  // Wait a tiny bit for async init checks to complete
  await new Promise(resolve => setTimeout(resolve, 50));

  const boxBtn = window.document.getElementById('boxBtn');
  assert.equal(boxBtn.style.display, 'flex');

  const { window: window2 } = await runPopupTest('disconnected');

  // Wait a tiny bit for async init checks to complete
  await new Promise(resolve => setTimeout(resolve, 50));

  const boxBtn2 = window2.document.getElementById('boxBtn');
  assert.equal(boxBtn2.style.display, 'none');
});

test('popup boxBtn click opens the sourcing1688 tab with correct url', async () => {
  const { window, tabCreated } = await runPopupTest('connected', {
    lingmirror_server_url: 'ws://localhost:8080',
  }, {
    lingmirror_jwt: generateValidToken(),
  });

  // Wait a tiny bit for async init checks to complete
  await new Promise(resolve => setTimeout(resolve, 50));

  const boxBtn = window.document.getElementById('boxBtn');
  assert.ok(boxBtn);

  // Simulate click
  boxBtn.click();

  // Wait a tiny bit for async click handler to run
  await new Promise(resolve => setTimeout(resolve, 50));

  assert.equal(tabCreated.length, 1);
  assert.equal(tabCreated[0].url, 'http://localhost:3000/sourcing1688');
});

test('popup describes device pairing in Chinese and never asks for a server before pairing', async () => {
  const { window, tabCreated, promptCalls } = await runPopupTest('no_token', {
    lingmirror_server_url: 'wss://118.196.42.156',
  });
  await new Promise(resolve => setTimeout(resolve, 50));

  assert.equal(window.document.getElementById('statusLabel').textContent, '未连接');
  const button = window.document.getElementById('loginBtn');
  assert.equal(button.textContent, '连接凌镜');
  button.click();
  await new Promise(resolve => setTimeout(resolve, 50));

  assert.deepEqual(promptCalls, []);
  assert.equal(tabCreated[0].url, 'https://118.196.42.156/settings/plugin');
});
