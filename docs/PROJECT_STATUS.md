# 凌镜 LingMirror Project Status

说明：`MultiSell` 是历史技术项目名；当前产品品牌为 `凌镜 LingMirror`。

更新时间：2026-06-15

## 一句话说明

MultiSell 是一个 AI 原生跨境电商商品中台，目标流程是：

商品创建 -> SKU / 价格 / 库存维护 -> AI 优化 -> 多平台发布 -> 订单和报表运营。

当前项目已经从原型整理成可测试、可继续扩展的 FastAPI + Vue 3 + PostgreSQL 应用。

项目协作和验收规则见：

- [产品愿景与第一可用版本](PRODUCT_VISION_AND_MVP.md)
- [项目收口与 Agent 协作规范](PROJECT_GOVERNANCE_AND_AGENT_WORKFLOW.md)

## 当前已完成

### 运行与数据库

- Docker Compose 已对齐 PostgreSQL 本地默认配置。
- 后端支持 Alembic migration 启动路径。
- 测试使用独立数据库 `product_management_test`。
- `.env.example` 已补充本地开发关键环境变量。

相关文件：

- `docker-compose.yml`
- `.env.example`
- `backend/alembic.ini`
- `backend/alembic/`
- `backend/tests/conftest.py`

### 商品核心链路

已覆盖商品从创建到 SKU / 价格 / 库存 / 供应商绑定的基础生命周期。

已修复或明确：

- SKU 生成保持幂等，同规格矩阵重复生成不会破坏已有库存。
- SKU 列表库存读取 `Inventory.quantity`，不再依赖废弃字段 `Sku.stock`。
- 商品删除会阻止删除存在 SKU、供应商绑定或平台发布记录的商品。

相关测试：

- `backend/tests/test_product_lifecycle.py`

### 商品列表物流数据工作台

状态：已完成。

已实现：
- 商品列表展示商品尺寸、商品重量、包装尺寸、包装重量、计费体积重、货品类型。
- 物流状态列显示 `可计算运费` / `缺物流数据`，缺数据时列明具体缺失字段。
- 支持按货品类型（普通/带电/液体/敏感）筛选。
- 支持按物流完整状态（complete/incomplete）筛选。
- 一键 `只看缺物流数据` 快速筛选按钮。
- 导出/导入包含完整的商品长宽高重量、包装长宽高重量、货品类型列。
- 导入按表头名称解析，不再依赖列索引。
- 导入模板内置货品类型下拉验证（normal/battery/liquid/sensitive）。
- 商品详情页新增 `物流信息` 卡片，展示货品类型、尺寸、重量、体积重预览、物流状态及缺失字段。
- 物流不完整时提供 `补齐物流数据` 快捷按钮。

相关测试：
- `backend/tests/test_product_list_logistics.py` — 12 个测试覆盖物流字段、筛选、Excel 导入导出。B7: 直到运费计算前置条件完善。

### 订单模块

已新增销售订单后端模块：

- 订单表：`sales_order`
- 订单明细表：`sales_order_item`
- 订单状态日志表：`sales_order_status_log`
- 支持订单创建、列表、详情、状态流转。

当前订单状态流：

`pending -> paid -> shipped -> delivered -> completed`

取消路径：

- `pending -> cancelled`
- `paid -> cancelled`

### 订单库存闭环

状态：已完成。

已实现：
- 创建订单锁定库存（`Inventory.locked_quantity += order_quantity`）。
- 支付订单（`pending -> paid`）扣减实物库存并释放锁定。
- 待支付订单取消（`pending -> cancelled`）释放锁定库存。
- 库存可用量 = 实物库存 - 锁定库存。
- 库存不足时阻止创建订单。
- 所有库存变动使用行锁（`SELECT ... FOR UPDATE`）防止并发超卖。
- 库存变动写入 `InventoryLog`，含订单号信息。
- `InventoryVO` 新增 `locked_quantity` 和 `available_quantity` 字段。

相关测试：

- `backend/tests/test_order_inventory_closure.py` — 5 个闭环行为测试。
- `backend/tests/test_order.py` — 新增库存设置，新增状态流转回归测试。

暂未实现：
- `paid -> cancelled` 自动退库存（需售后工作流）。
- 售后退货入库。
- 多仓库分配。
- 并发压力测试。

相关测试：

已从路由内 mock 发布，升级为 adapter 结构：

- `backend/app/listing/service.py`
- `backend/app/listing/adapters/base.py`
- `backend/app/listing/adapters/mock.py`

当前能力：

- 发布前检查商品主图、SKU、销售价、库存和平台状态。
- mock adapter 可创建 `ProductListing.status=synced`。
- adapter 报错时记录 `status=failed` 和 `sync_message`。
- 平台 API key 加密存储，并且不会从平台列表 / 详情接口返回。
- 发布前新增物流完整性检查，缺少商品级包装长宽高或包装重量时阻止发布。

相关测试：

- `backend/tests/test_listing.py`

### 物流基础字段第一阶段

已实现：

- `Product` 商品尺寸、默认包装尺寸/重量、`cargo_type`。
- `Sku` 包装覆盖尺寸/重量字段，保留 `Sku.weight` 作为历史重量字段。
- 商品创建、更新、详情、列表返回物流字段和 `logistics_status` / `logistics_status_name`。
- SKU 更新、详情、列表返回 SKU 包装覆盖字段。
- 商品列表显示物流完整状态。
- 发布前物流完整性检查。

相关测试：

- `backend/tests/test_logistics_attributes.py`
- `backend/tests/test_listing.py`

### 物流和运费第二阶段 — 运费计算基础系统

状态：已实现。

已实现：

- `ShippingProvider` / `ShippingChannel` / `ShippingZone` / `ShippingQuoteRule` 数据模型和迁移。
- 供应商/渠道/区域/规则完整 CRUD 管理 API。
- `POST /api/shipping/calculate` 运费计算 API，支持多渠道对比。
- 三种报价规则：固定费+每公斤、首重续重、阶梯重量。
- 体积重、计费重、取整、最低收费、附加费和燃油附加费。
- 货品类型过滤、目的地国家过滤、不活跃渠道排除。
- 按总运费升序排列。
- 运费计算器前端页面（`ShippingCalculator.vue`）。
- 物流管理前端页面（`ShippingManage.vue`）。
- 权限码：`shipping:view`、`shipping:manage`、`shipping:calculate`。
- 管理写操作审计日志。

相关测试：

- `backend/tests/test_shipping_calculation.py` — 16 个计算逻辑测试。

### 运费计算器手动试算

状态：已完成。

已支持：
- 手动输入包装长、宽、高、重量、数量、国家、货品类型计算运费。
- 保留 SKU 计算模式，SKU 模式继续从 SKU 或商品包装字段读取物流数据。
- 手动模式和 SKU 模式共用同一套渠道匹配、体积重、计费重、报价规则计算逻辑。

相关测试：
- `backend/tests/test_shipping_calculation.py` — 新增 `TestManualCalculation` 4 个测试（手动计算、维度校验、必填校验、向后兼容）。
- `backend/tests/test_shipping_management.py` — 22 个管理 CRUD 和权限测试。

相关文件：

- `backend/app/shipping/` — 运费计算后端模块。
- `docs/LOGISTICS_AND_SHIPPING_PRD.md` — 产品需求文档。
- `docs/LOGISTICS_SHIPPING_TECH_SPEC.md` — 技术规格文档。
- `docs/LOGISTICS_QUOTE_RULE_EXAMPLES.md` — 报价规则示例。
- `docs/superpowers/plans/2026-06-14-shipping-phase-2-calculation.md` — 实现计划。

第一阶段仍未实现：

- 真实物流承运商 API 集成。
- 装箱优化。

### 运费计算器手动试算

状态：已完成。

已支持：
- 手动输入包装长、宽、高、重量、数量、国家、货品类型计算运费。
- 保留 SKU 计算模式，SKU 模式继续从 SKU 或商品包装字段读取物流数据。
- 手动模式和 SKU 模式共用同一套渠道匹配、体积重、计费重、报价规则计算逻辑。

相关测试：
- `backend/tests/test_shipping_calculation.py` — 新增 `TestManualCalculation` 4 个测试（手动计算、维度校验、必填校验、向后兼容）。
- `backend/tests/test_shipping_management.py` — 22 个管理 CRUD 和权限测试。

相关文件：

- `backend/app/shipping/` — 运费计算后端模块。
- `docs/LOGISTICS_AND_SHIPPING_PRD.md` — 产品需求文档。
- `docs/LOGISTICS_SHIPPING_TECH_SPEC.md` — 技术规格文档。
- `docs/LOGISTICS_QUOTE_RULE_EXAMPLES.md` — 报价规则示例。
- `docs/superpowers/plans/2026-06-14-shipping-phase-2-calculation.md` — 实现计划。

第一阶段仍未实现：

- 真实物流承运商 API 集成。
- 装箱优化。

### 商品到运费计算器预填

已支持：
- 商品列表完整物流数据商品可一键进入运费计算器。
- 商品详情物流卡片可一键试算运费。
- 跳转时自动带入包装长、宽、高、重量、货品类型和商品来源。
- 物流数据不完整时引导回商品编辑页补齐数据。

相关文件：

- `frontend/src/views/product/ProductList.vue` — 运费试算/补物流按钮。
- `frontend/src/views/product/ProductDetail.vue` — 物流卡片试算运费。
- `frontend/src/views/shipping/ShippingCalculator.vue` — 来源商品提示。

暂未实现：
- SKU 级包装覆盖预填。
- 运输报价对比历史记录。

### 物流和运费第三阶段 — 订单运费快照与利润一期

状态：已完成。

已实现：

- `sales_order_shipping_snapshot` 保存订单选用的运费快照。
- `POST /api/orders/{id}/shipping-quote` 调用现有运费计算并保存最低价或指定渠道报价。
- `PUT /api/orders/{id}/profit-inputs` 支持维护利润输入。
- 订单详情返回 `shipping_snapshot` 和 `profit`。
- 历史订单运费不受后续报价规则变化影响。
- 利润一期字段：销售额、商品成本、运费、平台费、支付费、其他费用、利润、利润率。
- 前端订单详情页展示运费快照和利润测算。
- 新接口接入 `order:update` 权限和操作日志。

未实现：

- 多包裹装箱优化。
- 真实物流商 API 报价。
- 面单、追踪号、物流轨迹。
- 运费对账。

### 物流和运费第四阶段 — 报价表导入

状态：已完成。

已实现：

- `shipping_quote_rule.zone_id` 支持同一物流渠道按目的国家/区域使用不同报价规则。
- 旧规则 `zone_id = null` 继续作为渠道全局规则兼容。
- `POST /api/shipping/import-rules` 支持导入 `.xlsx` / `.csv` 报价表。
- 导入时自动创建物流供应商、物流渠道、目的地区域和区域级报价规则。
- 导入结果返回成功行数、错误行数、新增供应商/渠道/区域/规则数量和行级错误。
- 前端物流管理页增加“导入报价表”入口，并展示导入结果。
- 新增 `backend/tests/test_shipping_rate_import.py` 覆盖区域级报价和导入错误。

未实现：

- 报价版本管理。
- Excel 模板下载。
- 运费报价批量预览和回滚。

### 权限和审计主业务模块覆盖

已通过 TDD 分模块完成主业务模块权限和审计扩展：

统一权限依赖：

- `backend/app/auth/dependencies.py`

| 模块 | 权限码 | 审计日志 |
| --- | --- | --- |
| 商品 | `product:view/create/update/delete/import/export/ai` | 已覆盖 |
| 分类 | `category:view/create/update/delete` | 已覆盖 |
| 品牌 | `brand:view/create/update/delete` | 已覆盖 |
| 订单 | `order:view/create/update/update_status/cancel` | 已覆盖 |
| 库存 | `inventory:view/update` | 已覆盖 |
| 价格 | `price:view/update/batch_update` | 已覆盖 |
| SKU | `sku:view/create/update` | 已覆盖 |
| 供应商 | `supplier:view/create/update/delete` | 已覆盖 |
| 平台 | `platform:view/create/update/delete` | 已覆盖 |
| 发布 | `listing:view/publish` | 已覆盖 |
| 报表 | `dashboard:view`, `report:view` | — |
| RBAC | `rbac:view/manage` | — |
| 操作日志 | `operation_log:view` | — |
| 搜索 | `search:view` | — |

权限规则：

- `AUTH_ENABLED=False`：本地和测试默认跳过鉴权，返回系统用户。
- `AUTH_ENABLED=True`：必须携带 Bearer token。
- `User.role == "admin"`：管理员绕过显式权限码。
- 普通用户：必须通过角色拥有对应 `Permission.code`。

相关测试文件：

- `backend/tests/test_auth_rbac_audit_integration.py`
- `backend/tests/test_order_auth_audit.py`
- `backend/tests/test_inventory_price_auth_audit.py`
- `backend/tests/test_sku_supplier_auth_audit.py`
- `backend/tests/test_platform_listing_auth_audit.py`
- `backend/tests/test_admin_surface_auth.py`

测试助手：

- `backend/tests/auth_helpers.py`

### 前端权限感知

- 路由守卫保存 redirect 参数，登录后跳回原页面。
- 侧边菜单根据用户权限自动隐藏无权限项。
- `/auth/me` 接口返回用户权限列表。
- 各菜单路由通过 `meta.perm` 标记所需权限码。

## 当前验证结果

最近一次验证（2026-06-15 Phase 1 收口验证）：

```bash
cd backend && TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test .venv/bin/python -m pytest -q
cd frontend && npm run build
```

结果：196 passed, 0 failed, 227 warnings (均为 DeprecationWarning), 前端 build 通过（2.37s）。

## 已知限制

### 平台发布仍是 mock

adapter 边界已经建好，但真实平台 API 尚未接入。后续应按平台逐个实现：

- Ozon
- Shopee
- Wildberries
- AliExpress
- Temu

### Excel 导入需要继续修

Excel 导出和模板已有，但导入字段与模板还需要重新校准，并补充 SKU、价格、库存的批量导入能力。

### AI 能力还偏基础

当前 AI 优化能 fallback，但还缺：

- JSON schema 校验
- 人工确认流程
- 多语言标题 / 描述
- 平台差异化文案
- 敏感词和平台规则检查

## 后续优先级

推荐顺序：

1. ~~扩展权限和审计到全系统。~~ ✅
2. ~~完成订单库存扣减闭环。~~ ✅
3. 修复 Excel 导入并支持 SKU / 价格 / 库存批量导入。
4. 接入第一个真实平台 adapter。
5. 增强 AI、报表、定时同步和失败重试。

---

### 上架前经营决策

状态：已完成第一版。

已实现：
- 根据 SKU、目的国、目标售价、平台费、支付费、其他费用测算利润。
- 复用现有运费计算结果选择最低可用报价。
- 利润率达到阈值时建议上架。
- 物流数据缺失或无可用渠道时返回需补数据。

仍未实现：
- 平台类目佣金规则库。
- 多平台批量比较。
- AI 自动生成上架建议说明。
- 人工审批流。

### Excel 批量上架决策

状态：已完成第一版。

已实现：
- 下载批量上架决策 Excel 模板。
- 上传 `.xlsx` 后进行行级解析和校验预览。
- 有效行可自动填入批量决策页面。
- 错误行在页面展示行号和错误原因。
- 批量测算结果可导出为 Excel。

暂未实现：
- 保存导入历史。
- 保存测算历史。
- 从 Excel 中按商家 SKU 编码自动查找系统 SKU。
- 从 approve 结果直接创建平台发布任务。

### 决策到上架任务

状态：已完成第一版。

已实现：
- 批量上架决策 approve 结果可生成上架任务。
- 上架任务按商品和平台去重，避免重复任务。
- 上架任务复用发布前检查，缺商品图、SKU、价格、库存、物流数据时进入 blocked。
- ready 任务可调用现有发布 adapter 发布。
- 任务创建、重检、取消、发布接入权限和审计日志。

暂未实现：
- 真实平台 API。
- 平台类目映射。
- 平台属性映射。
- 发布失败自动重试队列。

### 运费账单对账

状态：已完成第一版。

已实现：
- 支持导入物流商 CSV 运费账单（支持中英文双格式列名）。
- 运单号、订单号自动匹配订单运费快照。
- 对账结果按 matched / unmatched_bill / missing_snapshot / amount_mismatch / currency_mismatch 分类。
- 手动解决差异功能。
- 对账操作接入权限（shipping:bill:import / shipping:bill:view / shipping:reconcile）和审计日志。
- 前端导入、对账、明细查询、异常行过滤页面。
- 对账汇总 API。

暂未实现：
- 实际物流商 API 直连。
- 自动对账调度。
- 按时间范围批量对账。
- 从对账结果自动调整订单利润。

### 平台费用规则

状态：已完成第一版。

已实现：
- 本地维护平台费用规则。
- 支持平台全局、平台+站点、平台+站点+类目三级匹配。
- 上架前经营决策可自动套用匹配到的平台费用规则。
- 无匹配规则时继续使用手动费率并返回 warning。
- 写操作接入权限和审计日志。

仍未实现：
- 真实平台费率 API 同步。
- 费用规则批量导入导出。
- 完整前端规则管理页。
- 跨平台类目映射。
