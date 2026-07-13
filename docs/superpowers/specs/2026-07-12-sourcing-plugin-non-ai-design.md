# 1688 采集助手非 AI 功能增强设计规约

**日期：** 2026-07-12
**状态：** 待评审

---

## 1. 业务目标
为提高 **凌镜 (LingMirror)** 1688 选品插件在实际使用中的安全性和便利性，本期增强旨在解决以下非 AI 功能痛点：
1. **防止风控拦截：** 在列表页批量采集时引入随机延时（Jitter）与用户自定义选项，降低触发 1688 滑块验证码或 IP 封锁的概率。
2. **数据补全：** 补全列表页卡片提取中的“供应商名称”与“店铺 ID”，避免后台数据缺失。
3. **流程便利性：** 在插件 Popup 界面提供直达“我的采集箱”链接，并配套配置插件图标。

---

## 2. 详细技术方案

### A. 批量采集延时自定义与随机化 (Delay Jitter)
*   **修改文件：** [content-script-list.ts](file:///Users/lc/multisell/chrome-extension/content-script-list.ts)
*   **UI 变化：** 在注入的侧边栏 `installListCollectorUI` 面板中，新增一个数字输入框：
    *   **标签：** `采集间隔 (秒)`
    *   **默认值：** `2`
    *   **样式：** 适配原有简约紫色系主题。
*   **逻辑调整：**
    *   在 [collectOffers](file:///Users/lc/multisell/chrome-extension/content-script-list.ts#L283-L303) 函数中，每次发起提交前，读取该输入框的值（限制最小值为 `0.5` 秒）。
    *   应用变频抖动逻辑计算单次延迟：
        $$\text{Delay} = (\text{Input Seconds} \times 1000 \times 0.7) + \text{Math.random()} \times (\text{Input Seconds} \times 1000 \times 0.6)$$
        该公式既保证了平均时长符合用户设置，又使时间间隔变得无规律，有效防屏蔽。

### B. 列表页供应商字段提取 (Supplier Parser)
*   **修改文件：** [content-script-list.ts](file:///Users/lc/multisell/chrome-extension/content-script-list.ts)
*   **逻辑调整：**
    *   新增 `extractReliableSupplier(card: HTMLElement): { name: string; id: string }` 函数。
    *   **供应商名称获取：** 通过选择器匹配卡片中的 `.company-name`、`.company`、`.shop-name`、`.seller-name`，净化文字，过滤掉“联系商家”等非主体字符。
    *   **供应商 ID 获取：** 遍历卡片内的超链接，匹配形如 `https://*.1688.com` 的商铺二级域名，解析出子域名作为 `supplier_id_1688`。
    *   更新 [pageDataFromCard](file:///Users/lc/multisell/chrome-extension/content-script-list.ts#L141-L176)，填充 `supplier_name`、`supplier_id_1688` 与 `supplier_business_id`，并置 `field_statuses.supplier = "observed"`。

### C. Popup 新增直达采集箱入口 (Popup Shortcut)
*   **修改文件：** [popup.html](file:///Users/lc/multisell/chrome-extension/popup.html)、[popup.ts](file:///Users/lc/multisell/chrome-extension/popup.ts)
*   **HTML 调整：**
    在 `actions` 容器中追加：
    ```html
    <button id="boxBtn" class="btn-secondary" style="display: none;">
      <span class="btn-icon">📦</span> 我的采集箱
    </button>
    ```
*   **TS 逻辑调整：**
    *   在 `updateStatus` 时，如果连接状态为 `connected`，则 `boxBtn.style.display = "flex"`；否则隐藏。
    *   监听 `boxBtn` 的 `click` 事件，调用 `chrome.tabs.create` 新建标签页跳转至 `http://[main_domain]/sourcing1688`。

### D. 插件图标配置 (Icons Configuration)
*   **修改文件：** [manifest.json](file:///Users/lc/multisell/chrome-extension/manifest.json)
*   已生成图标并输出至 `chrome-extension/icons/` 下的 `icon16.png`、`icon48.png`、`icon128.png`。
*   在 [manifest.json](file:///Users/lc/multisell/chrome-extension/manifest.json) 中添加配置：
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
    }
    ```

---

## 3. 测试与回归方案
*   在修改完功能后，在 `chrome-extension` 目录运行 `npm run test`。
*   由于测试使用 `linkedom` 模拟 DOM 环境，对于新增的 DOM 选取和延迟逻辑，需确保测试框架环境中的 Mock 数据兼容。
