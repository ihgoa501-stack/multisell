# 1688 选品插件非 AI 功能增强实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 1688 采集助手插件的多选变频延迟设置、列表页商铺解析、主站采集箱快捷跳转以及图标资源配置，增强采集流程的安全和便利。

**Architecture:**
- 在列表页注入 UI 中添加秒级数值输入框，并在并发批量采集时读取该数值结合随机 Jitter 作为延迟间隔。
- 使用 DOM 卡片类名选择器和商铺二级域名正则对列表卡片提取供应商名称与 ID。
- 在 Popup 动作区域加入“我的采集箱”按钮，点击后利用配对域名跳转直达选品看板页面。

**Tech Stack:** TypeScript, Chrome Extension API (v3), Node.js test runner, linkedom.

## Global Constraints
- 所有修改均应仅涉及 Chrome Extension 插件相关逻辑，禁止修改后端 Go 服务或主站页面原有逻辑（Popup 跳转逻辑应保持动态配置）。
- 每次代码修改必须通过 `npm run build` 重新编译 TS 代码并使用 `npm run test` 执行插件所有测试验证，确保 100% 通过。

---

### Task 1: 插件清单配置文件配置图标路径 (manifest.json Icons)

**Files:**
- Modify: `chrome-extension/manifest.json:6-25`

**Interfaces:**
- Consumes: 已生成并存放在 `chrome-extension/icons/` 目录下的 `icon16.png`、`icon48.png`、`icon128.png`。

- [ ] **Step 1: 修改 manifest.json 增加图标配置**
  在 `manifest.json` 的顶层添加 `"icons"`，并在 `"action"` 下面添加 `"default_icon"`。

  ```json
  "icons": {
    "16": "icons/icon16.png",
    "48": "icons/icon48.png",
    "128": "icons/icon128.png"
  },
  "action": {
    "default_popup": "popup.html",
    "default_icon": {
      "16": "icons/icon16.png",
      "48": "icons/icon48.png",
      "128": "icons/icon128.png"
    }
  },
  ```

- [ ] **Step 2: 验证 manifest.json 的格式**
  使用 JSON 格式校验工具或运行插件的测试套件，确保没有语法错误。
  运行：`npm run test`
  期望：`manifest is limited to the Owner 1688 private collection scope` 等测试顺利通过且没有 JSON 解析异常。

- [ ] **Step 3: 提交修改**
  ```bash
  git add chrome-extension/manifest.json
  git commit -m "chore(extension): configure manifest icons and action default_icon"
  ```

---

### Task 2: Popup 面板“我的采集箱”快速跳转按钮实现 (Popup Direct Box Link)

**Files:**
- Modify: `chrome-extension/popup.html:20-27`
- Modify: `chrome-extension/popup.ts`

**Interfaces:**
- Consumes: `getLoginUrl`、`getServerUrl` (来自 `shared/auth.js`)

- [ ] **Step 1: 修改 popup.html 追加“我的采集箱”按钮**
  在 `actions` 容器末尾添加 `boxBtn` 按钮：
  ```html
  <div class="actions">
    <button id="fetchBtn" class="btn-primary" disabled>
      <span class="btn-icon">＋</span> 采集到凌镜
    </button>
    <button id="loginBtn" class="btn-secondary">登录凌镜</button>
    <button id="boxBtn" class="btn-secondary" style="display: none;">
      <span class="btn-icon">📦</span> 我的采集箱
    </button>
  </div>
  ```

- [ ] **Step 2: 修改 popup.ts 增加元素定义、状态绑定与跳转逻辑**
  修改 `popup.ts`，添加 `boxBtn` 的绑定，在 `updateStatus` 中根据状态控制显示隐藏，并绑定点击跳转至 `/sourcing1688` 路径：

  ```typescript
  // 1. 获取 DOM 引用
  const boxBtn = document.getElementById("boxBtn") as HTMLButtonElement;

  // 2. 在 updateStatus 结尾中绑定显示逻辑
  boxBtn.style.display = status === "connected" ? "flex" : "none";

  // 3. 实现跳转逻辑并绑定事件
  async function handleOpenBox(): Promise<void> {
    const serverUrl = await getServerUrl();
    const loginUrl = getLoginUrl(serverUrl);
    const httpUrl = loginUrl.replace("/settings/plugin", "/sourcing1688");
    chrome.tabs.create({ url: httpUrl });
  }
  boxBtn.addEventListener("click", handleOpenBox);
  ```

- [ ] **Step 3: 运行构建并确认无编译错误**
  运行：`npm run build`
  期望：`tsc` 顺利完成且没有 TypeScript 编译错误。

- [ ] **Step 4: 提交修改**
  ```bash
  git add chrome-extension/popup.html chrome-extension/popup.ts
  git commit -m "feat(extension): add 'My Sourcing Box' shortcut button in popup"
  ```

---

### Task 3: 列表页卡片供应商名称及 ID 识别提取 (List Card Supplier Parsing)

**Files:**
- Modify: `chrome-extension/content-script-list.ts`
- Modify: `chrome-extension/tests/content-script-list.test.mjs`

**Interfaces:**
- Produces: `supplier_name`, `supplier_id_1688`, `supplier_business_id` (写入 `ListPageData`)，置 `field_statuses.supplier = "observed"`。

- [ ] **Step 1: 在 content-script-list.ts 中实现供应商和店铺 ID 解析函数**
  在 `content-script-list.ts` 中添加 `extractReliableSupplier` 帮助函数：
  ```typescript
  function extractReliableSupplier(card: HTMLElement): { name: string; id: string } {
    const selectors = [
      ".company-name", ".company", ".shop-name", ".shopname", ".seller-name",
      "[class*='company']", "[class*='shop']", "[data-company]",
    ];
    let name = "";
    for (const sel of selectors) {
      const el = card.querySelector<HTMLElement>(sel);
      if (el) {
        name = safeText(el.textContent);
        if (name && name.length <= 160 && !/^(进店|进入店铺|联系商家|收藏店铺|查看全部|实力商家|金牌制造)$/.test(name)) {
          break;
        }
      }
    }
    let id = "";
    const anchors = card.querySelectorAll<HTMLAnchorElement>("a[href]");
    for (const anchor of Array.from(anchors)) {
      const href = anchor.href;
      if (!href || href.includes("/offer/")) continue;
      const match = href.match(/^https?:\/\/([a-zA-Z0-9_-]+)\.1688\.com(?:\/|$)/);
      if (match) {
        id = match[1].trim();
        if (!name) {
          name = safeText(anchor.getAttribute("title") || anchor.textContent);
        }
        break;
      }
    }
    return { name, id };
  }
  ```

- [ ] **Step 2: 更新 pageDataFromCard 函数填充供应商信息**
  在 `pageDataFromCard` 中引入该解析器，并将 `field_statuses.supplier` 置为 `"observed"`（如果获取到名称）：
  ```typescript
    const supplier = extractReliableSupplier(card);
    const pageData: ListPageData = {
      schema_version: "sourcing1688.private.v1",
      offer_id_url: identity.offerId,
      offer_id_page: identity.offerId,
      source_url: identity.sourceURL,
      collected_at: new Date().toISOString(),
      driver: "chrome_extension_list_visible",
      parser_version: "1688-list-visible-v1",
      title,
      price_1688: price.price,
      price_model: price.model,
      currency: "CNY",
      min_order_qty: moq,
      images,
      supplier_name: supplier.name,
      supplier_id_1688: supplier.id,
      supplier_business_id: supplier.id,
      field_statuses: {
        title: "observed",
        price: price.price > 0 ? "observed" : "unknown",
        moq: moq > 0 ? "observed" : "unknown",
        supplier: supplier.name ? "observed" : "unknown",
        images: images.length > 0 ? "observed" : "unknown",
        sku: "unknown",
      },
    };
  ```

- [ ] **Step 3: 修改测试文件，增加卡片供应商数据并验证提取**
  在 `chrome-extension/tests/content-script-list.test.mjs` 中更新 `productCard` 以支持 mock 店铺属性，并添加专用单元测试断言：
  ```javascript
  // 1. 修改 productCard 支持传入 company 和 shopId
  function productCard(id, options = {}) {
    const title = options.title || `商品${id}`;
    const price = options.price === false ? '' : `<span class="price">¥${options.price || '8.80'}</span>`;
    const moq = options.moq ? `<span>${options.moq}件起批</span>` : '';
    const image = options.image === false ? '' : '<img src="https://cbu01.alicdn.com/img/card.jpg">';
    const company = options.company ? `<div class="company-name">${options.company}</div>` : '';
    const shopLink = options.shopId ? `<a href="https://${options.shopId}.1688.com">进入店铺</a>` : '';
    return `<li class="offer-item" ${options.hidden ? 'style="display:none"' : ''}>
      <a href="https://detail.1688.com/offer/${id}.html" title="${title}">${image}<span class="title">${title}</span></a>${price}${moq}${company}${shopLink}
    </li>`;
  }

  // 2. 添加单元测试用例
  test('visible list extraction parses supplier name and id if present in card', () => {
    const loaded = loadList(productCard('7001', { company: '杭州智造有限公司', shopId: 'hzsourcing' }));
    const offers = JSON.parse(vm.runInContext('JSON.stringify(extractVisibleOffers().map(({offerId,pageData}) => ({offerId,pageData})))', loaded.context));
    assert.equal(offers.length, 1);
    assert.equal(offers[0].pageData.supplier_name, '杭州智造有限公司');
    assert.equal(offers[0].pageData.supplier_id_1688, 'hzsourcing');
    assert.equal(offers[0].pageData.field_statuses.supplier, 'observed');
  });
  ```

- [ ] **Step 4: 编译并运行测试**
  运行：`npm run rebuild && npm run test`
  期望：所有 43 项测试（包括新增的供应商提取测试）全部 PASS。

- [ ] **Step 5: 提交修改**
  ```bash
  git add chrome-extension/content-script-list.ts chrome-extension/tests/content-script-list.test.mjs
  git commit -m "feat(extension): parse supplier name and subdomain ID in list page cards"
  ```

---

### Task 4: 列表页批量采集延时自定义与随机化抖动 (Random Jitter & Delay Input)

**Files:**
- Modify: `chrome-extension/content-script-list.ts`

**Interfaces:**
- Consumes: `#lingmirror-batch-delay-input` (用户在 UI 中输入的延时秒数值)

- [ ] **Step 1: 在 installListCollectorUI 中增加延时输入组件**
  修改 `installListCollectorUI` 函数，在 UI 控制按钮的上方，插入“采集间隔”输入框：

  ```typescript
  // 在 title, panelStatus 下方插入 delayContainer
  const delayContainer = document.createElement("div");
  Object.assign(delayContainer.style, { marginTop: "8px", marginBottom: "8px", display: "flex", gap: "6px", alignItems: "center" });
  const delayLabel = document.createElement("label");
  delayLabel.textContent = "采集间隔 (秒):";
  const delayInput = document.createElement("input");
  delayInput.type = "number";
  delayInput.id = "lingmirror-batch-delay-input";
  delayInput.min = "0.5";
  delayInput.step = "0.5";
  delayInput.value = "2.0";
  Object.assign(delayInput.style, { width: "55px", border: "1px solid #c7d2fe", borderRadius: "5px", padding: "3px 5px", font: "12px system-ui" });
  delayContainer.append(delayLabel, delayInput);

  // 更新 panel.append 顺序，将 delayContainer 追加进去
  panel.append(title, panelStatus, delayContainer, selectAll, collectSelectedButton, collectPage, cancel, panelResults);
  ```

- [ ] **Step 2: 修改 collectOffers 支持读取延时及随机 Jitter 变频**
  更新 `collectOffers` 函数逻辑：
  ```typescript
  // 1. 获取延时输入值
  const host = document.getElementById(LIST_UI_HOST_ID);
  const delayInput = host?.shadowRoot?.getElementById("lingmirror-batch-delay-input") as HTMLInputElement | null;
  const inputSeconds = parseFloat(delayInput?.value || "2.0");
  const targetDelayMs = Math.max(0.5, inputSeconds) * 1000;

  // 2. 循环中将原 250ms 延时替换为随机抖动时延
  if (index + 1 < offers.length) {
    const jitter = (targetDelayMs * 0.7) + Math.random() * (targetDelayMs * 0.6);
    await new Promise((resolve) => setTimeout(resolve, jitter));
  }
  ```

- [ ] **Step 3: 运行完整构建并回归测试**
  运行：`npm run rebuild && npm run test`
  期望：全部测试通过。由于测试中包含对 `SPA and virtual-list mutations` 异步批量停止的测试，需确保延时设定后，测试内的 mock 等待未受负面影响。

- [ ] **Step 4: 提交修改**
  ```bash
  git add chrome-extension/content-script-list.ts
  git commit -m "feat(extension): add user-adjustable random jitter delay in list collection"
  ```
