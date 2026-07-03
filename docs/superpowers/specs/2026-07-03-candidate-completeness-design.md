# CandidateProduct 完整度引擎与采集工作台联动

Date: 2026-07-03
Status: Approved
Risk Level: Low-Medium

## 1. 业务目标

解决第一公里的核心矛盾：采集回来的数据能否稳定、完整、可判断地变成可继续经营决策的 CandidateProduct。

**当前问题：** completeness_status 只有 incomplete / ready_for_profit_check 两种状态，用户无法从列表判断候选商品处于哪个阶段。缺少 missing_fields 输出和筛选。

## 2. 架构变更

### 2.1 完整度引擎（candidate 模块内升级）

`computeCompleteness()` 从 2 种状态升级为 4 种：

| Status | 含义 | 用户操作 |
|--------|------|---------|
| `incomplete` | 缺少核心字段（标题/价格/主图） | 只能看到记录，无法决策 |
| `needs_review` | 有核心字段，缺供应商/包裹信息 | 可以人工录入或执行线索补充 |
| `research_ready` | 有供应商+部分包裹，可以进入调研 | 可启动利润计算和选品调研 |
| `listing_ready` | 全部11项字段完整 | 可进入上架草稿准备 |

检查的 11 个字段：title, purchase_price, main_image, supplier_id, package_weight_kg, package_length_cm, package_width_cm, package_height_cm, hs_code, target_sale_price, origin_country

### 2.2 API 变更

| 端点 | 变更 |
|------|------|
| GET /v1/candidates | 新增 query param `completeness_status` 筛选；返回每个 item 的 completeness_status + missing_fields |
| GET /v1/candidates/:id | 返回增加 missing_fields 数组 |
| PUT /v1/candidates/:id | 增强：更新后自动重算 completeness_status |
| POST /v1/candidates | 不变（已有自动计算） |

### 2.3 前端变更

- 列表新增「完整度」列（彩色 Tag）和「缺失字段」列
- 工具栏新增完整性筛选下拉框
- 详情展示缺失字段列表 + 下一步建议
- 重构使用 PageContainer/SectionCard（按 AGENTS.md 规范）

### 2.4 不变范围

- 不新增路由
- 不改 completeness 域模块
- 不改数据库 migration（已有字段和索引）
- 不改 order/inventory/listing/shipping 模块
- 不产生自动采购、自动上架、自动改价、自动改库存

## 3. 变更文件清单

| 文件 | 变更内容 |
|------|---------|
| backend-go/internal/domain/candidate/completeness.go | 升级 2→4 状态 |
| backend-go/internal/domain/candidate/model.go | 新增 CandidateDetail 含 missing_fields; 新增 ListCandidateFilter 入参 |
| backend-go/internal/domain/candidate/service.go | List 加 completeness_status 筛选、Update 后重算、Get 返回 missing_fields |
| backend-go/internal/domain/candidate/handler.go | missing_fields 返回、筛选参数 |
| backend-go/internal/domain/candidate/candidate_test.go | 4状态测试 + 筛选测试 |
| frontend-next/src/app/(main)/candidates/page.tsx | 完整度列+筛选+详情增强+PageContainer重构 |

## 4. 测试计划

- computeCompleteness 四种状态全覆盖
- 边界：price=0, dims=0, supplier=nil, empty strings
- List 按 completeness_status 筛选
- Update 后自动重算
- go test ./... pass, go vet ./... pass, npm run build pass

## 5. 风险边界

| 风险 | 级别 | 措施 |
|------|------|------|
| 误触自动采购/上架 | 高 | 不修改订单/库存/刊登模块 |
| 破坏已有 migration | 中 | 不新增 migration |
| 前端 build 失败 | 中 | 保证 build 通过 |
| 影响 completeness 模块 | 低 | 不修改该模块 |
