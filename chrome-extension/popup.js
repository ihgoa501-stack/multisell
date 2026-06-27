/**
 * 凌镜 AI 选品助手 — Popup Script
 *
 * Displays WebSocket connection status, provides login/logout controls,
 * and allows manual triggering of product extraction on the current tab.
 */

// ─── DOM References ───────────────────────────────────────────────────

var statusDot = document.getElementById('statusDot');
var statusLabel = document.getElementById('statusLabel');
var fetchBtn = document.getElementById('fetchBtn');
var loginBtn = document.getElementById('loginBtn');
var resultCard = document.getElementById('resultCard');
var resultContent = document.getElementById('resultContent');
var settingsLink = document.getElementById('settingsLink');

// ─── State ───────────────────────────────────────────────────────────

var currentStatus = 'disconnected';
var isFetching = false;

// ─── Status UI ───────────────────────────────────────────────────────

/**
 * Update the connection status UI elements.
 * @param {'connected'|'disconnected'|'no_token'|'error'} status
 */
function updateStatus(status) {
  currentStatus = status;
  statusDot.className = 'status-dot ' + status;

  switch (status) {
    case 'connected':
      statusLabel.textContent = '已连接';
      fetchBtn.disabled = false;
      loginBtn.textContent = '已连接';
      loginBtn.title = '打开凌镜工作台';
      break;

    case 'no_token':
      statusLabel.textContent = '未登录';
      fetchBtn.disabled = true;
      loginBtn.textContent = '登录凌镜';
      loginBtn.title = '';
      break;

    case 'disconnected':
      statusLabel.textContent = '未连接';
      fetchBtn.disabled = true;
      loginBtn.textContent = '登录凌镜';
      loginBtn.title = '';
      break;

    case 'error':
      statusLabel.textContent = '连接错误';
      fetchBtn.disabled = true;
      loginBtn.textContent = '重新连接';
      loginBtn.title = '';
      break;
  }
}

/**
 * Display the result of a product extraction.
 * @param {Object} payload - The extraction result payload
 */
function showResult(payload) {
  resultCard.classList.add('visible');

  if (payload && payload.status === 'ok' && payload.data) {
    var data = payload.data;
    resultContent.className = '';
    resultContent.textContent =
      '标题:    ' + (data.title || 'N/A') + '\n' +
      '价格:    ¥' + (data.price_1688 != null ? data.price_1688 : 'N/A') + '\n' +
      '图片:    ' + (data.images ? data.images.length : 0) + ' 张\n' +
      '规格:    ' + (data.spec_variants ? data.spec_variants.length : 0) + ' 个\n' +
      '供应商:  ' + (data.supplier_name || 'N/A');
  } else if (payload && payload.code) {
    resultContent.className = 'error-text';
    resultContent.textContent = '错误: ' + payload.code + '\n' + (payload.message || '');
  } else {
    resultContent.className = 'error-text';
    resultContent.textContent = '异常响应: ' + JSON.stringify(payload);
  }
}

/**
 * Show the "extracting" loading state.
 */
function showFetching() {
  resultCard.classList.add('visible');
  resultContent.className = '';
  resultContent.innerHTML = '<span class="spinner"></span> 正在采集商品数据...';
}

// ─── Actions ─────────────────────────────────────────────────────────

/**
 * Fetch product data from the current active tab.
 */
function handleFetch() {
  if (isFetching) return;
  isFetching = true;
  fetchBtn.disabled = true;
  showFetching();

  chrome.tabs.query({ active: true, currentWindow: true }, function (tabs) {
    var tab = tabs && tabs[0];

    if (!tab || !tab.id || !tab.url) {
      showResult({ code: 'NO_TAB', message: '无法获取当前标签页' });
      isFetching = false;
      if (currentStatus === 'connected') fetchBtn.disabled = false;
      return;
    }

    // Verify we're on a valid product detail page
    var isValid = tab.url.indexOf('detail.1688.com') !== -1 ||
                  tab.url.indexOf('item.taobao.com') !== -1;

    if (!isValid) {
      showResult({
        code: 'WRONG_PAGE',
        message: '请打开 1688 或淘宝商品详情页。\n当前页面: ' + tab.url
      });
      isFetching = false;
      if (currentStatus === 'connected') fetchBtn.disabled = false;
      return;
    }

    // Send fetch request to content script
    chrome.tabs.sendMessage(tab.id, {
      type: 'fetch_product_from_page',
      requestId: 'popup_' + Date.now()
    }, function (response) {
      if (response && response.type === 'fetch_product_from_page_result') {
        showResult(response.payload);
      } else {
        showResult({ code: 'UNEXPECTED', message: '内容脚本无响应' });
      }
      isFetching = false;
      if (currentStatus === 'connected') fetchBtn.disabled = false;
    });
  });
}

/**
 * Handle login/logout.
 */
function handleLogin() {
  getStore(JWT_KEY).then(function (token) {
    if (token && currentStatus !== 'no_token') {
      // Already logged in and connected — open dashboard
      getStore('lingmirror_ws_url').then(function (wsUrl) {
        var httpUrl = (wsUrl || 'http://localhost:8080')
          .replace(/^ws:\/\//, 'http://')
          .replace(/^wss:\/\//, 'https://');
        chrome.tabs.create({ url: httpUrl });
      });
      return;
    }

    // Not logged in — ask for server URL first
    getStore('lingmirror_ws_url').then(function (wsUrl) {
      var serverUrl = wsUrl || 'ws://localhost:8080';
      var httpUrl = serverUrl
        .replace(/^ws:\/\//, 'http://')
        .replace(/^wss:\/\//, 'https://');

      var input = prompt('凌镜服务器地址:', httpUrl.replace(/\/ws\/extension$/, ''));
      if (input && input.trim()) {
        var inputWsUrl = input.trim()
          .replace(/^http:\/\//, 'ws://')
          .replace(/^https:\/\//, 'wss://');
        setStore('lingmirror_ws_url', inputWsUrl).then(function () {
          // Open login page — user will copy JWT from there
          var loginUrl = inputWsUrl
            .replace(/^ws:\/\//, 'http://')
            .replace(/^wss:\/\//, 'https://');
          chrome.tabs.create({ url: loginUrl + '/login' }, function () {
            // After opening login, the user must paste their JWT.
            // We'll prompt for it.
            setTimeout(function () {
              var jwt = prompt('请从凌镜工作台复制你的 JWT Token 并粘贴到此处:');
              if (jwt && jwt.trim()) {
                chrome.runtime.sendMessage({ type: 'set_token', token: jwt.trim() });
                updateStatus('disconnected');
              }
            }, 500);
          });
        });
      }
    });
  });
}

/**
 * Show server URL settings prompt.
 */
function handleSettings() {
  getStore('lingmirror_ws_url').then(function (wsUrl) {
    var httpUrl = (wsUrl || 'http://localhost:8080')
      .replace(/^ws:\/\//, 'http://')
      .replace(/^wss:\/\//, 'https://');

    var input = prompt('凌镜服务器地址 (WebSocket):', httpUrl.replace(/\/ws\/extension$/, ''));
    if (input && input.trim()) {
      var newUrl = input.trim()
        .replace(/^http:\/\//, 'ws://')
        .replace(/^https:\/\//, 'wss://');
      chrome.runtime.sendMessage({ type: 'set_server_url', url: newUrl });
    }
  });
}

// ─── Listen for Status Broadcasts ────────────────────────────────────

chrome.runtime.onMessage.addListener(function (message) {
  if (message.type === 'connection_status') {
    updateStatus(message.status);
  }
});

// ─── Initialization ──────────────────────────────────────────────────

function init() {
  // Query background for current status
  chrome.runtime.sendMessage({ type: 'get_status' }, function (response) {
    if (response && response.type === 'connection_status') {
      updateStatus(response.status);
    }
  });

  // If no JWT stored, update immediately
  getStore(JWT_KEY).then(function (token) {
    if (!token) {
      updateStatus('no_token');
    }
  });
}

// ─── Wire Up Event Listeners ─────────────────────────────────────────

fetchBtn.addEventListener('click', handleFetch);
loginBtn.addEventListener('click', handleLogin);
settingsLink.addEventListener('click', handleSettings);

// Run initialization
init();
