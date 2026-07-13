# OpenAI 商品图片 Owner 单次付费验收手册

> 状态：`ready_for_owner_inputs / external_call_not_authorized`
> 最后核验：2026-07-13
> 边界：只验证一个真实 Owner SKU 的一次 `gpt-image-2` 场景编辑；不证明可重复性、渠道效果或经营收益

## 1. 开始条件

以下条件必须全部满足：

1. Owner 选择一个真实 SKU 和一张不含人脸、地址、面单、二维码或秘密的商品原图；
2. Owner 确认拥有复制、修改、第三方 AI 处理和跨境传输该图片的权利；
3. 配方已冻结参考图、场景结构、Prompt、禁止改变项、模型和参数；
4. Owner 明确批准本次最高预算，只执行一次；
5. OpenAI 账号可用，并能在调用后核对请求和费用；
6. 凌镜与 Image Service 使用 PostgreSQL，迁移已完成；
7. 已准备记录原图、任务、输出、费用和 Owner 反馈的精确 ID 与哈希。

缺少任一项时停止，不用模拟结果代替。

## 2. 本地密钥配置

密钥只放在 Owner 本机未提交的环境文件或 secret storage，禁止发送到聊天、截图、日志或前端。

```text
IMAGE_SERVICE_OPENAI_ENABLED=true
IMAGE_SERVICE_OPENAI_API_KEY=<本地 secret>
IMAGE_SERVICE_SHARED_SECRET=<至少 32 字节的随机 secret A>
IMAGE_SERVICE_EXECUTION_TOKEN_SECRET=<不同的至少 32 字节随机 secret B>
IMAGE_SERVICE_DATABASE_URL=<本地 PostgreSQL URL>
```

上述数据库变量适用于 Docker Compose，仓库会将它传给容器内 `DATABASE_URL`；直接启动 Image Service 时改用 `IMAGE_SERVICE_JOB_STORE=postgres` 和 `DATABASE_URL=<本地 PostgreSQL URL>`。不要修改 Provider Host。

## 3. 启动前检查

1. 按 `OWNER_AND_AI_DEPLOYMENT_RUNBOOK.md` 启动数据库、迁移、后端、Image Service 和前端；
2. `/readyz` 必须 ready；
3. `/product-images` 中 OpenAI capability 必须同时显示：
   - `available`；
   - `production_paid`；
   - `provider_environment=production`；
   - `region=us`；
   - `watermarked=false`；
   - `non_publishable=false`；
4. 设置仅覆盖本次调用的 USD 预算策略，并确认剩余额度大于任务 `max_cost`。

任一字段不同就停止。

## 4. Owner 页面操作

1. 上传真实 SKU 原图并核对 SHA-256；
2. 为精确资产、Provider、Region、Purpose 和 Channel 登记未过期且未撤销的处理权利；
3. 创建 `OPENAI_IMAGE_EDIT` 任务，固定 `gpt-image-2`、1024×1024、PNG、单张参考图和版本化配方；
4. 在批准弹窗逐项核对 SKU、配方哈希、Prompt、禁止改变项、Provider 和最高预算；
5. 创建精确任务版本的执行批准；
6. Owner 再次明确确认费用后，只点击一次执行；
7. 页面没有明确终态前不得刷新后再次执行，也不得创建第二个相同任务规避门禁。

## 5. 结果裁决

### 通过

以下事实全部成立，才记为一次 `manually_verified`：

- 只有一次 Provider submit；
- Job、Attempt 和凌镜 Task 均进入 `READY`；
- Provider 返回 request ID 时，系统保存的是脱敏值；
- 输出字节可下载，SHA-256 与 `output_blob_id` 一致；
- Owner 全尺寸检查商品颜色、Logo、形状、数量和禁止改变项；
- Owner 对该候选保存选择、拒绝或返工反馈；
- 配方统计能按精确 SKU 聚合本次候选和返工；
- 根据 OpenAI 账单证据记录实际费用，预算预占进入 `spent`；
- 日志、页面、MCP 和错误响应均未出现 API Key；
- 没有自动创建图片集、发布证明或渠道发布。

这只证明一次真实调用和一次配方反馈闭环发生，不证明配方可重复有效。

### 不确定并停止

出现 timeout、断流、5xx、本地 Blob 保存失败、`RECONCILE_REQUIRED`、账单已扣费但无可恢复输出，或任务状态与 Provider 记录冲突时：

1. 立即停止，禁止重试；
2. 保存任务、attempt、脱敏 request ID、时间和账单证据；
3. 在 Owner 页面选择“确认未扣费”或“已扣费但无可恢复输出”之一进行对账；证据不足时保持待对账；
4. 不删除任务、不释放预算、不重置数据库来绕过门禁。

## 6. 验收后

1. 将 `IMAGE_SERVICE_OPENAI_ENABLED` 改回 `false`，从运行环境移除 API Key；
2. 保留原图权利、冻结配方、执行批准、预算、attempt、输出哈希和 Owner 反馈；
3. 生成带日期的事实记录，分别标记 `actual / manually_verified / unknown`；
4. 只有同类 SKU 重复验证后，才讨论自动推荐、批量或新增 Provider。

## 7. 当前未满足

- `unknown`：当前进程未配置 OpenAI API Key；
- `unknown`：尚未选择并登记真实 Owner SKU 原图及其精确权利；
- `unknown`：Owner 尚未批准一次真实付费调用和最高预算；
- `unknown`：尚无本次 Provider 结果和账单证据。

因此当前不得执行真实调用，也不得把工程测试写成真实 Provider 验收。
