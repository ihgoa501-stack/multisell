# Photoroom Sandbox 单次 Canary 运行手册

> 状态：`ready_for_owner_inputs / external_call_not_authorized`
> 最后核验：2026-07-13
> 边界：只验证一次带水印、不可发布的 Photoroom sandbox 请求；不验证生产可用、渠道接受或经营效果

## 1. 开始条件

只有以下条件全部满足才允许开始：

1. Owner 已创建 Photoroom sandbox 账号；
2. Owner 已在账号设置中确认训练 opt-out，并保留截图或导出证据；
3. Owner 选择一张非敏感商品测试图，并确认具有复制、修改、第三方 AI 处理和传输到美国的权利；
4. Owner 明确批准只消耗一次 sandbox canary 配额；
5. 图片不含人脸、面单、地址、二维码、秘密、身份证件或其他个人信息；
6. 当前环境为 `development` 或 `acceptance`，不能是 `production`；
7. 预期输出永久为 `sandbox + watermarked + non_publishable`，不得用于 Listing、广告或渠道发布。

缺少任一项时停止，不用模拟结果代替。

## 2. 密钥配置

API Key 只放在 Owner 本机未提交版本库的环境文件或 secret storage，禁止发送到聊天、截图、日志或前端。

需要设置：

```text
IMAGE_SERVICE_ENVIRONMENT=acceptance
IMAGE_SERVICE_PHOTOROOM_SANDBOX_ENABLED=true
IMAGE_SERVICE_PHOTOROOM_API_KEY=<本地 secret>
IMAGE_SERVICE_PHOTOROOM_SANDBOX_ACCOUNT_CONFIRMED=true
IMAGE_SERVICE_PHOTOROOM_TRAINING_OPT_OUT_CONFIRMED=true
```

同时配置两个互不相同、至少 32 字节的服务密钥：

```text
IMAGE_SERVICE_SHARED_SECRET=<随机 secret A>
IMAGE_SERVICE_EXECUTION_TOKEN_SECRET=<随机 secret B>
```

不要修改 Provider Host。运行时只允许 `https://image-api.photoroom.com`，并拒绝全部重定向。

## 3. 启动前检查

1. 执行 Image Service 与 LingMirror 的全部迁移；
2. 确认 `/readyz` 返回 ready；
3. 在凌镜 `/product-images` 查看 Photoroom capability；
4. capability 必须同时显示：
   - `available`；
   - `sandbox_only`；
   - `region=us`；
   - `provider_environment=sandbox`；
   - `watermarked=true`；
   - `non_publishable=true`；
   - `quota_remaining=1`。

任一字段不同就停止。

## 4. Owner 页面操作

1. 上传选定测试图；
2. 为精确资产 ID 与 SHA-256 登记权利：
   - `can_copy=true`；
   - `can_modify=true`；
   - `can_third_party_ai=true`；
   - `can_cross_border=true`；
   - `provider=photoroom`；
   - `region=us`；
   - purpose 与 channel 必须和任务完全一致；
3. 创建 Photoroom sandbox 任务，只选择三个允许操作之一；
4. 确认任务固定为 PNG、US、sandbox、`max_cost=0 USD`；
5. 创建精确版本的执行批准；
6. 再次检查权利未撤销、未过期；
7. Owner 明确确认“现在执行一次 canary”后才点击执行。

## 5. 结果裁决

### 通过

只有以下事实同时成立才记为该次 canary 通过：

- 只有一次 Provider submit；
- Job 与 Attempt 进入明确成功状态；
- 保存了脱敏 Provider request ID；
- 输出 SHA-256 可复算；
- 图片具有 Image Service 本地叠加并逐像素验证的明显 `SANDBOX` 横幅；
- Job、Task、Blob 下载头均显示 sandbox、水印、不可发布；
- 尝试创建 Image Set、release attestation 或 controlled publish 均被拒绝；
- Photoroom 配额变化与本次调用一致；
- 日志、MCP、前端和错误响应均未出现 API Key。

这只证明一次 sandbox 调用发生并被安全隔离，不证明可重复性、生产稳定性、图片质量或渠道可用性。

### 停止

出现以下任一情况立即停止：

- timeout、EOF、连接失败或 5xx；
- Job/Attempt 为 `RECONCILE_REQUIRED`；
- quota 已变成 0 但结果不明确；
- 输出不是 PNG、无法解码、哈希不一致或本地水印验证失败；
- 出现密钥、原图或个人数据泄漏迹象；
- 发现第二次 Provider submit。

停止后不得重试、不得重置数据库或配额来绕过一次性门禁。记录为 `unknown`，由 Owner决定是否联系 Provider人工核对。

## 6. Canary 后

1. 将 `IMAGE_SERVICE_PHOTOROOM_SANDBOX_ENABLED` 改回 `false`；
2. 从运行环境移除 API Key；
3. 保留任务、attempt、权利快照、预算预占、request ID、输出哈希和审计记录；
4. 不删除或改写不确定状态；
5. 不把 sandbox 输出加入任何可发布图片集合；
6. 生成带日期的 canary 事实记录，分别标记 `actual / unknown`。

## 7. 当前未满足

- `unknown`：尚无 Photoroom sandbox API Key；
- `unknown`：尚无 training opt-out 证据；
- `unknown`：尚未选择并登记一张具有精确处理权利的测试图；
- `unknown`：Owner 尚未授权一次真实外部调用。

因此当前不得执行 canary。
