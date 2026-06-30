# 凌镜 7 天一人 Agent 公司 MVP 计划

更新时间：2026-06-29
版本：v0.1 (Day 0)
Lead Agent：当前 Claude Code 主会话

---

## 总览

在 7 天内完成最小可运行经营闭环。已有 80% 基础设施，本计划覆盖剩余 20% 的接线、增强和种子数据。

### 演示目标

```
20 个候选商品
-> 资料完整度检查
-> 成本 / 物流 / 平台费 / 利润计算
-> Agent 给出上架建议
-> 生成任务 / 待审批
-> Owner 总控台看到风险、建议、审批、失败
-> 平台只读或模拟数据进入
-> 所有关键动作有记录
```

### 5 条并行线

| 线 | 名称 | 负责人 | 产出 |
|----|------|--------|------|
| A | 经营闭环线 | Lead + Subagent | 完整度检查、利润测算、任务生成、审批接线 |
| B | Owner 总控台线 | Subagent | 风险面板、建议列表、审批操作 |
| C | Agent 建议线 | Subagent | 商品分析 Agent、利润评估 Agent、上架建议 Agent |
| D | 平台数据线 | Subagent | Mock 订单/结算数据、模拟同步 |
| E | QA / Review / 文档线 | Subagent | 每日验证、风险审查、文档更新、验收日志 |

---

## Day 1 — 产品完整度检查 + 种子数据

**重点: A 线启动，E 线并行**

### A 线 — 经营闭环线
1. 新建 `candidate` domain 模块：candidate_product 模型 + CRUD API
   - 字段：title, description, main_image, images(list), category_id, brand_id
   - 规格: spec_json (JSONB: 颜色/尺寸/重量/材质)
   - 采购成本: purchase_price, purchase_currency
   - 包装信息: package_weight_kg, package_length/width/height_cm
   - 供应商: supplier_id
   - 目标售价: target_sale_price, target_currency
   - 状态: draft / in_review / approved / rejected
2. 新建 `completeness` domain 模块
   - CompletenessCheckService.Check(productID) → 评分 + 缺失清单
   - 检查项: 标题/描述/图片/类目/品牌/规格/包装/运输属性/HS编码/采购成本/目标售价
   - API: POST /api/v1/completeness/check/:productId
   - API: GET /api/v1/completeness/checks?status=complete|incomplete
3. 创建 20 个种子候选商品（通过 seed 脚本）
   - 10 个相对完整（评分 >80%）
   - 5 个中等完整（评分 50-80%）
   - 5 个不完整（评分 <50%）
4. 接线到 router.go (v1 group)

### E 线 — QA / Review / 文档
1. 运行 `go test ./...` 确认基线
2. 运行 `go vet ./...`
3. 创建 Day 1 acceptance entry
4. 更新 battle board

**验收标准:**
- `GET /api/v1/completeness/checks` 返回 20 条记录，含评分和缺失项
- `go test ./...` 通过
- 种子数据可用

---

## Day 2 — 成本/物流/平台费汇总 + 利润测算

**重点: A 线串联，C 线启动**

### A 线 — 经营闭环线
1. 新建 profit_summary API
   - 输入: product_id → 汇总采购成本 + 物流费(A10) + 平台费(platformfee) + 关税(tariff)
   - 输出: total_cost, shipping_cost, platform_fee, tariff, target_profit, profit_margin
   - API: GET /api/v1/profit/summary/:productId
   - API: GET /api/v1/profit/summaries?status=profitable|marginal|unprofitable
2. 接线 logistics A10 → profit 计算
   - 调用 logistics 费率引擎计算运费
3. 接线 platformfee → profit 计算
   - 调用 platform_fee 规则计算费用
4. 接线 tariff → profit 计算（关税引擎）
5. 接线 supplier cost → profit 计算

### C 线 — Agent 建议线
1. 实现利润评估 Agent 规则
   - 输入: completeness 评分 + profit 汇总 + market average
   - 输出: listing_recommendation (建议上架/谨慎/不建议)
   - 存储到 decision 记录
2. 实现商品完整度 Agent
   - 分析缺失项 → 给出具体补充建议（业务语言）

### E 线 — QA / Review / 文档
1. `go test ./...` + `go vet ./...`
2. 检查新增 API 是否有测试
3. 更新 battle board + acceptance log

**验收标准:**
- 20 个商品都能算出利润和利润率
- Agent 给出建议（每个商品至少一条）
- `go test ./...` 通过

---

## Day 3 — Agent 上架建议 + 任务生成

**重点: A 线 + C 线交汇**

### A 线 — 经营闭环线
1. 接线: completeness_check → profit_summary → listing_recommendation
   - 一条 API 触发全链路: POST /api/v1/loop/evaluate/:productId
2. 接线: listing_recommendation → listingtask
   - 推荐上架时自动创建 listingtask
   - listingtask.status = blocked（要求审批）
   - 记录 decision_snapshot

### C 线 — Agent 建议线
1. 实现上架建议 Agent
   - 综合完整度评分 + 利润率 + 市场判断
   - 输出: 建议 + 理由 + 风险提示 + 补充资料建议
   - 输出: 置信度评分

### B 线 — Owner 总控台线 (启动)
1. 后端 API: GET /api/v1/agentos/risk/summary
   - 返回: 低利润商品数、缺失数据商品数、待审批任务数、同步异常数
2. 后端 API: GET /api/v1/agentos/suggestions
   - 返回: Agent 建议列表（按时间/风险排序）

### E 线 — QA / Review / 文档
1. 全链路冒烟测试（手动 + API）
2. 检查审批接线

**验收标准:**
- 一条 API 触发完整度→利润→建议全链路
- "建议上架"的商品有对应 listingtask（blocked 状态）
- Agent 风险汇总 API 有数据

---

## Day 4 — Owner 总控台 + 审批接线

**重点: B 线完成，A/C 线收尾**

### B 线 — Owner 总控台线
1. 前端: `/agentos/risk` 页面 — 风险面板
   - 低利润商品清单
   - 不完整商品清单
   - 待审批任务数量徽章
2. 前端: `/agentos/suggestions` 页面 — Agent 建议
   - 每个建议展示: 商品名、Agent 名、建议内容、置信度、时间
   - 操作: 采纳 / 拒绝 / 稍后处理
3. 前端: `/agentos/approvals` 页面 — 待审批
   - listingtask 待审批列表
   - 操作: 批准 / 拒绝（写入 approval_request）

### A 线 — 经营闭环线
1. 接线: Owner 审批操作 → listingtask status 变更
   - 批准: blocked → pending_approval → approved
   - 拒绝: blocked → rejected
2. 接线: 每个操作写入 operationlog

### D 线 — 平台数据线 (启动)
1. Ozon mock adapter 增强
   - 生成示例订单
   - 生成示例费用
   - 生成示例结算

### E 线 — QA / Review / 文档
1. 前端 build + test 检查
2. 检查审批+审计是否符合 KERNEL_CONTRACTS

**验收标准:**
- 在总控台能看到风险、建议、待审批
- 批准/拒绝操作能变更 listingtask 状态
- 操作记录在 operationlog

---

## Day 5 — 平台数据展示 + 端到端闭环

**重点: D 线完成，全链路打通**

### D 线 — 平台数据线
1. Mock 数据接入 API
   - GET /api/v1/mock/orders — 模拟 Ozon 订单
   - GET /api/v1/mock/settlements — 模拟结算
   - GET /api/v1/mock/fees — 模拟费用
2. 前端: 平台同步状态视图
   - 同步时间、同步状态、失败原因
3. 接线: mock 数据 → dashboard/settlement 显示

### A 线 — 经营闭环线
1. 全链路验收接线
   - seed → completeness → profit → agent → recommendation → listingtask → approval
2. 边缘情况处理
   - 商品数据不完整时 Agent 建议明确说明原因
   - 利润为负时 Agent 判断"不建议上架"

### B 线 — Owner 总控台线
1. 前端: 平台数据状态卡片
2. 前端: 操作日志查看

### E 线 — QA / Review / 文档
1. 全链路 E2E 验证
2. 检查所有 prohibitions 是否遵守

**验收标准:**
- 全链路可走通
- Mock 数据在总控台可浏览
- Operationlog 可查看

---

## Day 6 — 打磨 + 种子数据完整化

**重点: 所有线收尾，演示准备**

### A 线 — 经营闭环线
1. 完善 20 个种子商品的数据
   - 让演示更有说服力（不同利润、不同完整度）
2. 错误处理和回退
3. API 返回中文化

### B 线 — Owner 总控台线
1. 页面打磨
2. 中文业务语言标签
3. 空状态处理
4. 加载状态

### C 线 — Agent 建议线
1. Agent 建议文案优化
2. 建议理由更业务化（不是 technical 描述）

### D 线 — 平台数据线
1. 模拟数据丰富度提升
2. 同步日志更多细节

### E 线 — QA / Review / 文档
1. 完整回归测试
2. 风险审查
3. 文档最终化

**验收标准:**
- 所有页面和 API 可用
- Owner 能在总控台轻松看懂所有信息

---

## Day 7 — 演示交付

**重点: 演示 + 文档 + 最终检查**

### E 线 — QA / Review / 文档
1. 撰写演示脚本
2. 最终 review 检查
3. 更新所有文档
4. 演示录屏准备

### Owner 演示流程
1. 打开总控台 → 看到 20 个商品概况
2. 查看完整度检查 → 看到哪些商品缺什么
3. 查看利润分析 → 哪些赚钱哪些亏
4. 查看 Agent 建议 → 哪些推荐上架
5. 审批一个商品 → 状态变更
6. 查看平台数据 → 看到 mock 订单
7. 查看操作日志 → 看到记录

**验收标准:**
- 演示流程走通
- Owner 不需要懂代码就能理解
- `go test ./...` + `go vet ./...` 通过
- `npm run build` 通过
- `npm test` 通过

---

## 风险

| 风险 | 影响 | 缓解 |
|------|------|------|
| logistics 费率引擎需要真实配置 | 计算可能需要 mock | 准备预计算值 fallback |
| 现有 domain 接线复杂 | 开发时间超预期 | 优先 wire, 后优化 |
| 审批流程复杂 | 演示超范围 | 先做简单 approve/reject |
| 前端组件缺失 | 需要自建 | 用 CrudListPage 模板 |
| Agent 建议不够智能 | 演示效果差 | 先写规则 + prompt，不依赖 LLM streaming |

## 不做的事

- 不接第二个平台
- 不做真实自动发布
- 不做自动改价
- 不做自动改库存
- 不做自动退款 / 赔付
- 不新增无关页面
