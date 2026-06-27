# 凌镜 AI 选品助手 — Chrome 扩展

> 相关目录：`chrome-extension/`

## 概述

Chrome 扩展用于从 1688 商品详情页实时采集结构化商品数据（标题、价格、图片、规格、供应商等），通过 WebSocket 回传给后端 A8 选品引擎。

## 架构

```
1688 详情页 ← content script 提取数据 → background service worker → WebSocket → ExtensionHandler → PluginDriver → Sourcing Agent
```

包含三个进程：

| 进程 | 文件 | 用途 |
|------|------|------|
| Content Script | `content.js` | 在 1688 详情页运行，提取商品信息 |
| Background Worker | `background.js` | WebSocket 连接管理，中转消息 |
| Popup | `popup.html` + `popup.js` | 用户界面，显示连接状态 |

## 安装

1. 打开 Chrome，访问 `chrome://extensions`
2. 开启"开发者模式"
3. 点击"加载已解压的扩展程序"，选择 `chrome-extension/` 目录
4. 扩展栏会出现凌镜图标

## WebSocket 协议

扩展与后端通过 WebSocket 通信（路径 `/ws`，需 JWT 认证）：

**后端 → 扩展：**
```json
{"type": "fetch_product", "id": "uuid", "payload": {"url": "https://detail.1688.com/xxx"}}
```

**扩展 → 后端：**
```json
{"type": "fetch_product_result", "id": "uuid", "payload": {"status": "ok", "data": {title, price_1688, images, ...}}}
```

**扩展 → 后端（错误）：**
```json
{"type": "fetch_product_error", "id": "uuid", "payload": {"code": "PARSE_FAILED", "message": "..."}}
```

心跳：扩展每 30 秒发送 `ping`，后端回复 `pong`。

## 内容脚本注入

Content script 自动注入到以下页面：

- `https://detail.1688.com/*` — 1688 商品详情
- `https://item.taobao.com/*` — 淘宝商品详情

注入后自动提取：标题、价格区间、最小起订量、多图、规格变体（颜色/尺寸/价格/库存）、供应商名/ID、描述文本、包裹重量/尺寸。

## 后端接线

```go
// ExtensionHandler 处理扩展的 WebSocket 连接
handler := realtime.NewExtensionHandler(hub, logger, jwtSecret)
handler.WithPluginDriver(pluginDriver)
// ⚠️ 目前 ExtensionHandler 尚未挂接到 WebSocket 路由
```
