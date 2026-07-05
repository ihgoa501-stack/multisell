> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# Logistics Attributes Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 MultiSell 增加第一阶段物流基础字段，使商品和 SKU 能保存包装尺寸、包装重量、商品尺寸和货品类型，并在商品维护、SKU 管理、商品列表和发布前检查中体现物流数据完整性。

**Architecture:** 第一阶段只扩展现有 `Product` / `Sku` 数据模型和既有商品、SKU、发布链路，不新增完整物流模块。后续物流供应商、物流渠道、目的地覆盖、报价规则、运费计算器和订单运费快照另行进入 ShippingProvider / ShippingChannel 阶段。

**Tech Stack:** FastAPI, SQLAlchemy async, Alembic, PostgreSQL, Pydantic, pytest, Vue 3, Naive UI, Vite.

---

## 阶段边界

### 本阶段必须做

- `Product` 增加商品尺寸字段：`product_length_cm`、`product_width_cm`、`product_height_cm`、`product_weight_kg`。
- `Product` 增加默认包装字段：`package_length_cm`、`package_width_cm`、`package_height_cm`、`package_weight_kg`。
- `Product` 增加货品类型字段：`cargo_type`，默认 `normal`。
- `Sku` 增加 SKU 级包装覆盖字段：`sku_length_cm`、`sku_width_cm`、`sku_height_cm`、`sku_weight_kg`。
- 保留现有 `Sku.weight`，继续视为当前历史 SKU 重量字段，不把它重命名，也不把它作为包装重量使用。
- 新增服务层物流完整性判断：商品默认包装四字段均大于 0 时为完整；SKU 仅在四个 SKU 包装字段均大于 0 时覆盖商品包装，否则回退商品包装。
- 商品列表展示物流状态：`物流完整` / `物流不完整`。
- 发布前检查增加物流基础字段校验，但只校验数据是否完整，不计算运费。

### 本阶段明确不做

- 不创建 `ShippingProvider` 表。
- 不创建 `ShippingChannel`、`ShippingZone`、`ShippingQuoteRule`、`ShippingQuoteSnapshot` 表。
- 不做供应商物流渠道管理页面。
- 不做报价规则配置。
- 不做运费计算器、体积重/计费重报价比较接口。
- 不接入真实承运商 API。
- 不改订单运费快照模型。

### 设计决策

- 后续物流供应商使用独立 `ShippingProvider`，不复用现有商品供货商 `Supplier`。原因：采购供应商和物流承运商生命周期、字段、权限和运营角色不同。
- 现有 `Sku.weight` 保留为历史 SKU 重量字段，不作为包装总重。原因：PRD 明确包装重量包含商品和包装材料，且现有字段语义不足以承载包装总重。
- 新增 `Sku.sku_weight_kg` 作为 SKU 级包装重量覆盖字段。只有 `sku_length_cm`、`sku_width_cm`、`sku_height_cm`、`sku_weight_kg` 四项全部有效时，才使用 SKU 覆盖值。
- 商品尺寸 `product_*` 只用于描述商品本身，不参与物流完整性判断，也不参与发布前物流检查。

---

## 文件改动地图

### 后端测试

- Create: `backend/tests/test_logistics_attributes.py`
  - 覆盖商品物流字段创建、更新、详情、列表返回。
  - 覆盖 SKU 包装覆盖字段更新和返回。
  - 覆盖物流完整性状态。
  - 覆盖 `Sku.weight` 与 `Sku.sku_weight_kg` 语义分离。
- Modify: `backend/tests/test_listing.py`
  - 发布前检查增加物流基础字段不完整时阻止发布的断言。

### 数据库迁移和模型

- Create: `backend/alembic/versions/20260614_01_add_logistics_attributes.py`
  - 给 `product` 表新增 9 个物流基础字段。
  - 给 `sku` 表新增 4 个 SKU 包装覆盖字段。
  - downgrade 删除对应字段。
- Modify: `backend/app/models.py`
  - `Product` 增加商品尺寸、包装尺寸、包装重量、货品类型列。
  - `Sku` 增加 SKU 包装覆盖列。
  - 不删除、不重命名 `Sku.weight`。

### 后端 schema / service / router

- Modify: `backend/app/core/schemas.py`
  - `ProductCreate`、`ProductUpdate`、`ProductVO` 增加物流字段。
  - `ProductVO` 增加 `logistics_status`、`logistics_status_name`、可选的包装展示辅助字段。
- Modify: `backend/app/core/service.py`
  - `product_to_vo()` 返回物流字段。
  - 增加商品默认包装完整性判断函数。
  - 保持 `ProductService.create/update/list_products/get_by_id` 的现有边界。
- Modify: `backend/app/core/router.py`
  - 继续复用现有 `/products` 创建、更新、详情、列表接口。
  - 不新增物流专用 router。
- Modify: `backend/app/sku/schemas.py`
  - `SkuUpdate`、`SkuVO` 增加 `sku_length_cm`、`sku_width_cm`、`sku_height_cm`、`sku_weight_kg`。
- Modify: `backend/app/sku/router.py`
  - `sku_to_vo()` 返回 SKU 包装覆盖字段。
- Modify: `backend/app/sku/service.py`
  - `update_sku()` 允许更新新增 SKU 包装覆盖字段。
  - `generate_skus()` 不自动从 `Sku.weight` 填充 `sku_weight_kg`。
- Modify: `backend/app/listing/service.py`
  - 发布前检查增加商品包装数据完整性要求。
  - 只阻止缺少物流基础字段的发布，不做运费计算。

### 前端

- Modify: `frontend/src/views/product/ProductForm.vue`
  - 在商品表单增加“商品尺寸”和“包装信息”输入区。
  - 录入 `cargo_type`。
  - 编辑商品时回填物流字段。
  - 保存时随现有 `productApi.create/update` 提交。
- Modify: `frontend/src/views/sku/SkuManage.vue`
  - SKU 表格展示包装覆盖摘要。
  - 编辑 SKU 弹窗增加 SKU 包装长宽高和 SKU 包装重量输入。
  - 保留现有 `weight` 字段语义，不把它显示为包装重量。
- Modify: `frontend/src/views/product/ProductList.vue`
  - 商品列表增加物流状态列。
  - 物流完整显示成功标签，物流不完整显示警告标签。
- Optional Modify: `frontend/src/api/index.ts`
  - 当前 API 封装使用 `any` 透传，不一定需要改；只有在实现时抽出类型或前端 payload helper 时才修改。

### 文档

- Modify: `docs/PROJECT_STATUS.md`
  - 实现完成后补充“物流基础字段第一阶段”状态和验证结果。
- Modify: `docs/LOGISTICS_SHIPPING_TECH_SPEC.md`
  - 实现完成后标记 Product/SKU 基础字段已进入第一阶段实现，ShippingProvider/ShippingChannel 仍未实现。
- Modify: `docs/LOGISTICS_AND_SHIPPING_PRD.md`
  - 如验收口径有变化，只更新第一阶段边界说明。
- Modify: `docs/DEVELOPMENT_GUIDE.md`
  - 如新增测试命令或迁移注意事项，只补充开发说明。

---

## TDD 任务拆分

### Task 1: 后端测试先行

**Files:**

- Create: `backend/tests/test_logistics_attributes.py`
- Modify: `backend/tests/test_listing.py`

- [ ] **Step 1: 写商品物流字段创建测试**

  测试目标：

  - `POST /api/products` 接收 `product_length_cm`、`product_width_cm`、`product_height_cm`、`product_weight_kg`。
  - 接收 `package_length_cm`、`package_width_cm`、`package_height_cm`、`package_weight_kg`。
  - 接收 `cargo_type=battery`。
  - 响应中返回上述字段。
  - 包装四字段均大于 0 时，响应返回 `logistics_status=complete` 和 `logistics_status_name=物流完整`。

- [ ] **Step 2: 写商品物流字段更新测试**

  测试目标：

  - 创建无包装字段商品后，详情返回 `物流不完整`。
  - `PUT /api/products/{product_id}` 补齐包装四字段后，详情返回 `物流完整`。
  - 商品尺寸字段为空不影响物流完整性。

- [ ] **Step 3: 写商品列表物流状态测试**

  测试目标：

  - 商品 A 缺少 `package_weight_kg`，列表返回 `logistics_status=incomplete`。
  - 商品 B 包装四字段完整，列表返回 `logistics_status=complete`。
  - 列表不需要查询 ShippingProvider、ShippingChannel 或报价规则。

- [ ] **Step 4: 写 SKU 包装覆盖字段测试**

  测试目标：

  - `PUT /api/skus/{sku_id}` 可以更新 `sku_length_cm`、`sku_width_cm`、`sku_height_cm`、`sku_weight_kg`。
  - `GET /api/skus/{sku_id}` 和 `GET /api/products/{product_id}/skus` 返回这些字段。
  - 更新 `weight` 不会自动改变 `sku_weight_kg`。
  - 更新 `sku_weight_kg` 不会自动改变 `weight`。

- [ ] **Step 5: 写发布前检查物流测试**

  修改 `backend/tests/test_listing.py`：

  - 复用现有发布测试夹具。
  - 商品主图、SKU、价格、库存、平台都完整，但缺少包装四字段时，发布返回 400。
  - 错误信息包含“物流数据不完整”或等价明确原因。
  - 补齐包装字段后，原 mock 发布成功测试仍通过。

- [ ] **Step 6: 运行失败测试**

  Run:

  ```bash
  cd backend && python3 -m pytest tests/test_logistics_attributes.py tests/test_listing.py -q
  ```

  Expected:

  - `test_logistics_attributes.py` 因模型字段、schema 字段或响应字段不存在而失败。
  - `test_listing.py` 中新发布前检查用例因尚未校验物流字段而失败。

### Task 2: Alembic migration

**Files:**

- Create: `backend/alembic/versions/20260614_01_add_logistics_attributes.py`

- [ ] **Step 1: 创建迁移文件**

  迁移必须新增以下列：

  `product` 表：

  - `product_length_cm NUMERIC(10, 2) NULL`
  - `product_width_cm NUMERIC(10, 2) NULL`
  - `product_height_cm NUMERIC(10, 2) NULL`
  - `product_weight_kg NUMERIC(10, 2) NULL`
  - `package_length_cm NUMERIC(10, 2) NULL`
  - `package_width_cm NUMERIC(10, 2) NULL`
  - `package_height_cm NUMERIC(10, 2) NULL`
  - `package_weight_kg NUMERIC(10, 2) NULL`
  - `cargo_type VARCHAR(50) NULL DEFAULT 'normal'`

  `sku` 表：

  - `sku_length_cm NUMERIC(10, 2) NULL`
  - `sku_width_cm NUMERIC(10, 2) NULL`
  - `sku_height_cm NUMERIC(10, 2) NULL`
  - `sku_weight_kg NUMERIC(10, 2) NULL`

- [ ] **Step 2: downgrade 对称删除**

  downgrade 按新增顺序的反向删除列，保证本地测试库可回滚。

- [ ] **Step 3: 运行迁移验证**

  Run:

  ```bash
  cd backend && python3 -m alembic upgrade head
  ```

  Expected:

  - 迁移执行成功。
  - PostgreSQL `product` 和 `sku` 表出现新增列。

### Task 3: 模型 / schema / service / router

**Files:**

- Modify: `backend/app/models.py`
- Modify: `backend/app/core/schemas.py`
- Modify: `backend/app/core/service.py`
- Modify: `backend/app/core/router.py`
- Modify: `backend/app/sku/schemas.py`
- Modify: `backend/app/sku/router.py`
- Modify: `backend/app/sku/service.py`

- [ ] **Step 1: 更新 SQLAlchemy 模型**

  在 `Product` 增加商品尺寸、包装尺寸、包装重量、货品类型字段。

  在 `Sku` 增加 SKU 包装覆盖字段。

  不改动 `Sku.weight` 字段定义，注释可进一步说明它不是包装重量。

- [ ] **Step 2: 更新商品 schema**

  `ProductCreate` 和 `ProductUpdate` 接收全部新增商品物流字段。

  `ProductVO` 返回：

  - 全部新增商品物流字段。
  - `logistics_status`，取值 `complete` / `incomplete`。
  - `logistics_status_name`，取值 `物流完整` / `物流不完整`。

  字段校验：

  - 尺寸和重量允许为空。
  - 非空时必须大于 0。
  - `cargo_type` 默认 `normal`，允许 `normal`、`battery`、`liquid`、`sensitive`。

- [ ] **Step 3: 更新商品 service**

  在 `backend/app/core/service.py` 增加共享判断：

  - `package_length_cm`、`package_width_cm`、`package_height_cm`、`package_weight_kg` 全部非空且大于 0，商品物流完整。
  - 任一为空或小于等于 0，商品物流不完整。

  `product_to_vo()` 必须使用该判断填充状态字段。

- [ ] **Step 4: 检查商品 router**

  保持现有 `/products`、`/products/{product_id}` 路由。

  不新增 `/shipping` 路由。

  确认 `ProductService.create/update` 使用 schema 的 `model_dump()` 后可以透传新增字段。

- [ ] **Step 5: 更新 SKU schema 和 VO**

  `SkuUpdate` 接收：

  - `sku_length_cm`
  - `sku_width_cm`
  - `sku_height_cm`
  - `sku_weight_kg`

  `SkuVO` 返回以上字段，并继续返回历史 `weight`。

- [ ] **Step 6: 更新 SKU router/service**

  `sku_to_vo()` 返回 SKU 包装覆盖字段。

  `SpecService.update_sku()` 保持只更新传入字段，允许新增字段通过。

  `SpecService.generate_skus()` 生成新 SKU 时不从商品包装字段复制 SKU 覆盖字段，也不从 `weight` 推导 `sku_weight_kg`。

- [ ] **Step 7: 运行后端局部测试**

  Run:

  ```bash
  cd backend && python3 -m pytest tests/test_logistics_attributes.py -q
  ```

  Expected:

  - 物流字段创建、更新、列表、SKU 覆盖测试通过。

### Task 4: 前端商品表单

**Files:**

- Modify: `frontend/src/views/product/ProductForm.vue`

- [ ] **Step 1: 增加表单字段**

  在基础商品信息之后增加两个区块：

  - 商品尺寸：长、宽、高、商品重量。
  - 包装信息：包装长、包装宽、包装高、包装重量、货品类型。

  使用 `n-input-number`，单位显示为 `cm` / `kg`，精度 2 位。

- [ ] **Step 2: 初始化 form**

  `form` 初始值包含全部新增字段：

  - 商品尺寸字段默认为 `null`。
  - 包装字段默认为 `null`。
  - `cargo_type` 默认为 `normal`。

- [ ] **Step 3: 编辑模式回填**

  加载商品详情时回填物流字段。

  对历史商品，后端返回空值时保持 `null`，不要自动填 `0`。

- [ ] **Step 4: 前端基础校验**

  对包装字段使用非负限制，但不要强制必填。

  原因：草稿商品允许物流不完整；发布前检查再阻止发布。

- [ ] **Step 5: 手工验证商品表单**

  Run:

  ```bash
  cd frontend && npm run build
  ```

  Expected:

  - TypeScript/Vite 构建通过。
  - 商品创建和编辑页面无模板编译错误。

### Task 5: SKU 管理页

**Files:**

- Modify: `frontend/src/views/sku/SkuManage.vue`

- [ ] **Step 1: SKU 表格增加包装摘要**

  SKU 列表增加“包装覆盖”列：

  - 四个 SKU 包装字段都有值时显示 `L × W × H cm / W kg`。
  - 任一字段缺失时显示 `使用商品默认包装`。

- [ ] **Step 2: 编辑弹窗增加 SKU 包装字段**

  在 SKU 编辑弹窗中增加：

  - SKU 包装长 `sku_length_cm`
  - SKU 包装宽 `sku_width_cm`
  - SKU 包装高 `sku_height_cm`
  - SKU 包装重量 `sku_weight_kg`

  保留现有价格、库存、条形码、状态编辑能力。

- [ ] **Step 3: 保留 `weight` 语义**

  如果页面继续展示或编辑 `weight`，标签必须避免写成“包装重量”。

  推荐标签：`历史重量` 或 `重量(旧字段)`。

  保存 payload 必须同时支持 `weight` 和 `sku_weight_kg`，二者互不覆盖。

- [ ] **Step 4: 保存并刷新**

  `saveSku()` payload 增加四个 SKU 包装覆盖字段。

  保存成功后继续调用 `fetchSkus()` 刷新列表。

- [ ] **Step 5: 前端构建验证**

  Run:

  ```bash
  cd frontend && npm run build
  ```

  Expected:

  - 构建通过。
  - SKU 管理页无模板编译错误。

### Task 6: 商品列表物流状态

**Files:**

- Modify: `frontend/src/views/product/ProductList.vue`

- [ ] **Step 1: 增加物流状态列**

  在商品列表状态列或平台列附近增加“物流”列：

  - `logistics_status=complete` 显示绿色 `物流完整`。
  - 其他情况显示黄色 `物流不完整`。

- [ ] **Step 2: 缺省兼容**

  如果历史接口或异常数据没有 `logistics_status`，按不完整展示。

- [ ] **Step 3: 列宽和操作区检查**

  调整列宽，避免新增列后挤压“操作”按钮。

- [ ] **Step 4: 前端构建验证**

  Run:

  ```bash
  cd frontend && npm run build
  ```

  Expected:

  - 构建通过。
  - 商品列表页面无模板编译错误。

### Task 7: 发布前检查

**Files:**

- Modify: `backend/app/listing/service.py`
- Modify: `backend/tests/test_listing.py`

- [ ] **Step 1: 扩展 preflight 规则**

  在现有发布前检查中增加物流基础字段要求：

  - `package_length_cm > 0`
  - `package_width_cm > 0`
  - `package_height_cm > 0`
  - `package_weight_kg > 0`

  只检查商品级默认包装字段。

  不在发布前检查中解析 SKU 覆盖、不计算体积重、不查询物流渠道。

- [ ] **Step 2: 错误信息**

  缺失物流字段时，缺失项列表应出现可读提示，例如：

  - `物流数据不完整：请填写包装长宽高和包装重量`

- [ ] **Step 3: 运行发布测试**

  Run:

  ```bash
  cd backend && python3 -m pytest tests/test_listing.py -q
  ```

  Expected:

  - 缺少物流字段的发布用例返回 400。
  - 补齐物流字段的 mock 发布用例继续通过。

### Task 8: 文档更新

**Files:**

- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/LOGISTICS_SHIPPING_TECH_SPEC.md`
- Modify: `docs/LOGISTICS_AND_SHIPPING_PRD.md`
- Modify: `docs/DEVELOPMENT_GUIDE.md`

- [ ] **Step 1: 更新项目状态**

  在 `docs/PROJECT_STATUS.md` 增加“物流基础字段第一阶段”：

  - 已实现 Product/SKU 物流基础字段。
  - 已实现商品列表物流状态。
  - 已实现发布前物流完整性检查。
  - ShippingProvider/ShippingChannel/报价规则/运费计算器仍未实现。

- [ ] **Step 2: 更新技术规格实现状态**

  在 `docs/LOGISTICS_SHIPPING_TECH_SPEC.md` 标记：

  - Product/SKU 基础字段已进入第一阶段实现。
  - `ShippingProvider` 等后续表仍是推荐模型，未建表。

- [ ] **Step 3: 检查 PRD 阶段边界**

  如 PRD 仍把完整运费计算描述为同一阶段，补一句：

  - 第一阶段只落地 Product/SKU 物流基础字段，运费计算链路另行排期。

- [ ] **Step 4: 更新开发指南**

  如实现新增了专门测试文件，在 `docs/DEVELOPMENT_GUIDE.md` 的测试示例中补充：

  ```bash
  cd backend && python3 -m pytest tests/test_logistics_attributes.py -q
  ```

### Task 9: 验证

**Files:**

- No source changes unless verification exposes required fixes.

- [ ] **Step 1: 运行物流属性测试**

  Run:

  ```bash
  cd backend && python3 -m pytest tests/test_logistics_attributes.py -q
  ```

  Expected:

  - 全部通过。

- [ ] **Step 2: 运行发布测试**

  Run:

  ```bash
  cd backend && python3 -m pytest tests/test_listing.py -q
  ```

  Expected:

  - 全部通过。

- [ ] **Step 3: 运行后端全量测试**

  Run:

  ```bash
  cd backend && python3 -m pytest -q
  ```

  Expected:

  - 全部通过。

- [ ] **Step 4: 运行前端构建**

  Run:

  ```bash
  cd frontend && npm run build
  ```

  Expected:

  - 构建通过。

- [ ] **Step 5: 运行 diff 检查**

  Run:

  ```bash
  git diff --check
  ```

  Expected:

  - 无空白错误。

---

## 验收清单

- [ ] 数据库迁移只新增 Product/SKU 物流基础字段，没有创建 ShippingProvider/ShippingChannel/报价规则相关表。
- [ ] `Sku.weight` 仍存在，未被删除、重命名或改成包装重量。
- [ ] SKU 新增 `sku_weight_kg`，作为包装重量覆盖字段。
- [ ] 商品创建、更新、详情、列表接口返回物流字段和物流状态。
- [ ] SKU 详情和列表接口返回 SKU 包装覆盖字段。
- [ ] 商品表单能录入和回填商品尺寸、包装信息、货品类型。
- [ ] SKU 管理页能录入和展示 SKU 包装覆盖字段。
- [ ] 商品列表显示物流状态。
- [ ] 发布前检查能阻止物流数据不完整的商品发布。
- [ ] 所有后端测试和前端构建通过。
- [ ] 文档明确第一阶段不包含完整物流供应商、渠道、报价和运费计算能力。
