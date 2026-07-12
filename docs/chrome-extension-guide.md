# 凌镜1688采集助手 — Chrome扩展

> 相关目录：`chrome-extension/`
> 当前版本：0.2.0
> 使用者：Owner本人

## 用途

Owner浏览1688商品详情页时，点击一次“采集到凌镜”，商品经HTTPS保存到Owner私人采集箱。商品初始状态为`unverified_lead`，只表示Owner保存过该页面及页面当时的声明，不代表商品机会、可信供应商或可上架草稿。

页面不会在加载时自动上传，也不会自动发布、采购、改价或处理订单。

## 正常流程

```text
打开1688商品详情页
→ 点击“采集到凌镜”
→ 正在读取
→ 正在保存
→ 已保存（显示服务器记录编号）
→ 继续浏览或打开凌镜私人采集箱
```

决定继续研究时，在凌镜 `/sourcing1688` 页面关联选品任务。后端通过经营闸门后，记录才进入既有复核和受控草稿流程。

## 组成

| 部分 | 当前文件 | 用途 |
|---|---|---|
| 1688内容脚本 | `content-script.ts` → `build/content-script.js` | 商品解析、页面按钮和轻量结果面板 |
| 后台Service Worker | `background.ts` → `build/background.js` | 登录状态、HTTPS保存、旧WebSocket兼容 |
| 工具栏弹窗 | `popup.html` / `popup.ts` | 连接状态和备用采集入口 |
| 登录桥 | `auth-bridge.ts` | 从已登录凌镜页面完成插件连接 |
| 共享协议 | `shared/` | API地址、消息类型和私人收藏请求 |

## 安装

```bash
cd chrome-extension
npm install
npm test
```

然后：

1. Chrome打开 `chrome://extensions`；
2. 开启“开发者模式”；
3. 点击“加载已解压的扩展程序”；
4. 选择仓库中的 `chrome-extension/` 目录；
5. 固定“凌镜1688采集助手”图标；
6. 点击“连接凌镜”，在 `/settings/plugin` 核对浏览器身份并明确确认连接。

不需要复制JWT、Token或打开开发者工具。插件不接收网页登录JWT；短期访问凭证只保存在当前浏览器会话，设备恢复凭证可由Owner在设置页单独撤销。

## 页面与权限

P0只注入：

```text
https://detail.1688.com/offer/*
```

扩展不再申请淘宝、Ozon、1688列表页、浏览历史、Cookie、下载或全部网站权限。Manifest权限变化由 `tests/manifest.test.mjs` 检查。

## 私人收藏API

```http
POST /api/v1/extension/sourcing-1688/private-collections
GET /api/v1/extension/sourcing-1688/private-collections/requests/:requestId  # 超时后核对是否已保存
Authorization: Bearer <15分钟、设备绑定、仅sourcing1688.collect用途的插件凭证>
Content-Type: application/json
```

请求包含：

- `request_id`：本次点击的幂等身份；
- `source_url`、URL商品ID、页面声明商品ID和页面观察时间；
- `schema_version=sourcing1688.private.v1`、插件版本与解析器版本；
- 页面结构化数据及原始证据；
- 能可靠取得的标题、价格、MOQ、供应商、图片和SKU。

服务端分别保存原始证据、规范结构和请求信封三类SHA-256；同一`request_id`只有请求信封完全一致时才返回原结果，URL与页面商品ID冲突时拒绝保存。

只有服务端返回`status=saved`、`record_id`和`request_id`后，页面才能显示“已保存”。网络超时或未知响应不能冒充成功。

## 与旧WebSocket链的关系

旧的后端指定URL受控读取仍保留兼容能力，但不是新的一键私人收藏主路径。旧的页面加载自动上传和1688列表页自动落候选路径已经移除。WebSocket连接不能作为P0私人收藏保存成功的唯一依据。

## 常见提示

| 提示 | 含义与处理 |
|---|---|
| 当前不是商品页 | 打开 `detail.1688.com/offer/...` 后重试 |
| 需要登录或验证 | 完成1688登录/验证码，刷新后重试 |
| 请先登录凌镜 | 点击插件中的登录按钮完成连接 |
| 当前商品未确认保存 | 不要把它当成功；恢复网络或登录后重新采集 |
| 已保存，记录编号 #... | 服务端已持久化，可以继续浏览 |
| 商品已存在 | 打开已有记录或保存新的观察版本，不复制商品主体 |

## 验证命令

```bash
cd chrome-extension
npm test
```

自动测试证明编译、请求映射、成功回执规则、Manifest范围和无自动上传路径；它不能证明真实1688页面字段已经准确。真实可用仍需Owner用一个实际商品完成Chrome主闭环并人工对照页面。
