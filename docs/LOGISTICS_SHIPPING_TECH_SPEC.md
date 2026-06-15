# MultiSell 物流与运费技术规格

> 技术设计文档 · 2026-06-15
> 状态：第五阶段已实现（物流字段、运费计算、订单运费快照与利润一期、商品列表物流数据工作台、运费计算器手动试算）

---

## 1. 当前代码上下文

### 1.1 现有模型

| 模型 | 现有相关字段 | 备注 |
|---|---|---|
| `Product` | 已实现商品尺寸、包装尺寸/重量、`cargo_type` | 第一阶段已落地基础字段 |
| `Sku` | `weight` + 已实现 SKU 包装覆盖字段 | `weight` 保留为历史重量字段，不表示包装重量 |
| `Supplier` | `name`, `contact`, `address` | 需扩展物流渠道子模型 |
| `Order` | `shipping_fee`、利润输入/输出字段 | `shipping_fee` 继续作为当前订单选用快照总运费的兼容冗余字段 |
| `OrderItem` | 无物流字段 | 可选记录单品包装数据 |

> **重要**：`Product` / `Sku` 基础物流字段、`ShippingProvider` / `ShippingChannel` / 报价规则、运费计算和订单运费快照均已落地。真实承运商 API、装箱优化、面单/追踪和运费对账仍未实现。

### 1.2 项目模块约定

后端模块结构（未实现物流模块）：

```
backend/app/<module>/
  __init__.py    # 暴露 router
  router.py     # HTTP 路由 + 依赖注入
  service.py    # 业务逻辑
  schemas.py    # Pydantic 请求/响应模型
```

新物流模块将遵循此结构，文件夹名为 `shipping`。

---

## 2. 推荐数据模型

### 2.1 Product 表新增字段

```sql
-- 商品尺寸（用于描述商品，不是运费计算的必填项）
product_length_cm    DECIMAL(10,2)  DEFAULT NULL,
product_width_cm     DECIMAL(10,2)  DEFAULT NULL,
product_height_cm    DECIMAL(10,2)  DEFAULT NULL,
product_weight_kg    DECIMAL(10,2)  DEFAULT NULL,

-- 包装尺寸（运费计算必填项）
package_length_cm    DECIMAL(10,2)  DEFAULT NULL,
package_width_cm     DECIMAL(10,2)  DEFAULT NULL,
package_height_cm    DECIMAL(10,2)  DEFAULT NULL,
package_weight_kg    DECIMAL(10,2)  DEFAULT NULL,

-- 货品类型
cargo_type           VARCHAR(50)    DEFAULT 'normal',
```

### 2.2 Sku 表新增字段

```sql
sku_length_cm        DECIMAL(10,2)  DEFAULT NULL,
sku_width_cm         DECIMAL(10,2)  DEFAULT NULL,
sku_height_cm        DECIMAL(10,2)  DEFAULT NULL,
sku_weight_kg        DECIMAL(10,2)  DEFAULT NULL,
```

### 2.3 ShippingProvider（物流供应商）

物流供应商是"发货方"维度，通常与现有 `Supplier` 重合或独立。

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | BIGINT PK | |
| `name` | VARCHAR(200) NOT NULL | 供应商名称，如"云途物流" |
| `code` | VARCHAR(50) UNIQUE | 编码，如 `yuntufast` |
| `contact` | VARCHAR(100) | 联系人 |
| `phone` | VARCHAR(50) | 联系电话 |
| `remark` | TEXT | 备注 |
| `status` | SMALLINT DEFAULT 1 | 0-禁用, 1-启用 |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |

### 2.4 ShippingChannel（物流渠道）

每个供应商有多个物流产品（渠道）。

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | BIGINT PK | |
| `provider_id` | BIGINT FK → `shipping_provider.id` | 所属物流供应商 |
| `name` | VARCHAR(200) NOT NULL | 渠道名称，"美国普货专线" |
| `code` | VARCHAR(50) | 渠道编码 |
| `volumetric_divisor` | INT NOT NULL DEFAULT 6000 | 抛重系数：6000/5000/8000 |
| `cargo_types` | JSON | 支持货品类型，如 `["normal","battery"]` |
| `estimated_delivery_min` | INT | 最短时效（天） |
| `estimated_delivery_max` | INT | 最长时效（天） |
| `currency` | VARCHAR(10) DEFAULT 'CNY' | 报价币种 |
| `sort_order` | INT DEFAULT 0 | 排序 |
| `status` | SMALLINT DEFAULT 1 | 0-禁用, 1-启用 |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |

### 2.5 ShippingZone（物流区域 / 目的地覆盖）

渠道 × 国家（或邮编范围）。

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | BIGINT PK | |
| `channel_id` | BIGINT FK → `shipping_channel.id` | 物流渠道 |
| `zone_id` | BIGINT FK → `shipping_zone.id`, NULL | 物流区域；NULL 表示渠道全局规则 |
| `country_code` | VARCHAR(10) NOT NULL | 国家代码 ISO 3166-1 alpha-2，如 `US` |
| `postal_code_from` | VARCHAR(20) | 邮编范围起始（可选） |
| `postal_code_to` | VARCHAR(20) | 邮编范围截止（可选） |
| `status` | SMALLINT DEFAULT 1 | |

### 2.6 ShippingQuoteRule（报价规则）

每个渠道可有多段规则。

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | BIGINT PK | |
| `channel_id` | BIGINT FK → `shipping_channel.id` | 物流渠道 |
| `rule_type` | VARCHAR(50) NOT NULL | 规则类型（见 §6） |
| `priority` | INT DEFAULT 0 | 优先级，值小优先 |
| `min_weight_kg` | DECIMAL(10,3) DEFAULT 0 | 适用最小重量 |
| `max_weight_kg` | DECIMAL(10,3) DEFAULT NULL | 适用最大重量（NULL 无上限） |
| `first_kg` | DECIMAL(10,3) DEFAULT 0 | 首重（kg） |
| `first_price` | DECIMAL(10,2) DEFAULT 0 | 首重价格 |
| `additional_kg` | DECIMAL(10,3) DEFAULT 0 | 续重单位（kg） |
| `additional_price` | DECIMAL(10,2) DEFAULT 0 | 续重单价 |
| `fixed_fee` | DECIMAL(10,2) DEFAULT 0 | 固定费 |
| `per_kg_price` | DECIMAL(10,2) DEFAULT 0 | 每公斤价格 |
| `minimum_charge` | DECIMAL(10,2) DEFAULT NULL | 最低收费 |
| `tier_config` | JSON | 阶梯配置（见 §6.3 示例） |
| `surcharge_fixed` | DECIMAL(10,2) DEFAULT 0 | 附加费（固定） |
| `fuel_surcharge_pct` | DECIMAL(5,2) DEFAULT 0 | 燃油附加费百分比 |
| `rounding_increment` | DECIMAL(10,3) DEFAULT 0.1 | 计费重向上取整增量（kg） |
| `remark` | TEXT | |
| `status` | SMALLINT DEFAULT 1 | |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |

### 2.7 ShippingQuoteSnapshot（运费快照）

存储每笔订单选用的运费计算结果。

实际表名：`sales_order_shipping_snapshot`。

实际绑定接口：

- `POST /api/orders/{id}/shipping-quote`
- `PUT /api/orders/{id}/profit-inputs`

`sales_order.shipping_fee` 保留为兼容字段，并始终等于当前订单选用快照的 `total_shipping_fee`。

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | BIGINT PK | |
| `order_id` | BIGINT FK → `sales_order.id` | 订单 ID |
| `provider_id` | BIGINT FK → `shipping_provider.id` | 选用的物流供应商 |
| `channel_id` | BIGINT FK → `shipping_channel.id` | 选用的物流渠道 |
| `destination_country` | VARCHAR(10) | 目的地国家 |
| `destination_postal_code` | VARCHAR(20) | 目的地邮编 |
| `actual_weight_kg` | DECIMAL(10,3) | 实际重量 |
| `volumetric_weight_kg` | DECIMAL(10,3) | 体积重 |
| `chargeable_weight_kg` | DECIMAL(10,3) | 计费重 |
| `volumetric_divisor` | INT | 使用的抛重系数 |
| `base_shipping_fee` | DECIMAL(10,2) | 基础运费 |
| `surcharge_fee` | DECIMAL(10,2) | 附加费 |
| `fuel_surcharge_fee` | DECIMAL(10,2) | 燃油附加费 |
| `total_shipping_fee` | DECIMAL(10,2) | 总运费 |
| `currency` | VARCHAR(10) | 币种 |
| `calculation_detail` | JSON | 计算过程 JSON，用于前端展示 |
| `estimated_delivery_days` | INT | 预计时效（天） |
| `created_at` | TIMESTAMP | |

---

## 3. 字段验证规则

| 验证项 | 规则 |
|---|---|
| 包装尺寸 | 所有 > 0；`Decimal(10,2)` 精度 2 位小数 |
| 包装重量 | > 0；`Decimal(10,2)` |
| 抛重系数 | 整数，常见值 5000 / 6000 / 8000 |
| 币种 | ISO 4217 大写，如 `CNY`、`USD` |
| 国家代码 | ISO 3166-1 alpha-2 大写 |
| 计费重取整 | 按 `rounding_increment` 向上取整，默认 0.1 kg |

> **内部规范单位**：长度一律用 `cm`（厘米），重量一律用 `kg`（公斤）。前端或其他系统传入其他单位时，由接入层转换。

---

## 4. 回退规则

```
IF SKU.sku_weight_kg IS NOT NULL
    AND SKU.sku_length_cm IS NOT NULL
    AND SKU.sku_width_cm IS NOT NULL
    AND SKU.sku_height_cm IS NOT NULL
THEN
    package_weight_kg = SKU.sku_weight_kg
    package_length_cm = SKU.sku_length_cm
    package_width_cm  = SKU.sku_width_cm
    package_height_cm = SKU.sku_height_cm
ELSE
    package_weight_kg = Product.package_weight_kg
    package_length_cm = Product.package_length_cm
    package_width_cm  = Product.package_width_cm
    package_height_cm = Product.package_height_cm
END IF

IF 回退后任一字段 IS NULL OR = 0 THEN
    包装数据不完整 → 运费计算被阻塞
END IF
```

> 商品尺寸（`product_*` 字段）和包装尺寸（`package_*` 字段）是两组独立字段，商品尺寸**不参与**运费计算。

---

## 5. 计费重公式

### 5.1 体积重

```
volumetric_weight_kg = (package_length_cm × package_width_cm × package_height_cm) / volumetric_divisor
```

### 5.2 计费重

```
chargeable_weight_kg = max(actual_weight_kg, volumetric_weight_kg)
```

### 5.3 取整规则

```
# 按渠道指定增量向上取整
rounded_chargeable_kg = ceil(chargeable_weight_kg / rounding_increment) × rounding_increment
```

- 默认 `rounding_increment = 0.1 kg`（按 0.1 kg 向上取整）
- 实例：0.35 kg → 0.4 kg；1.01 kg → 1.1 kg

---

## 6. 报价规则类型

### 6.1 `fixed_plus_per_kg`（固定费 + 每公斤）

```
fee = fixed_fee + (rounded_chargeable_kg × per_kg_price)
```

### 6.2 `first_weight_plus_increment`（首重 + 续重）

```
if rounded_chargeable_kg <= first_kg:
    fee = first_price
else:
    additional_units = ceil((rounded_chargeable_kg - first_kg) / additional_kg)
    fee = first_price + (additional_units × additional_price)
```

### 6.3 `tiered_weight`（阶梯重量）

`tier_config` 示例：

```json
[
  {"min_kg": 0,    "max_kg": 0.5,  "price": 35},
  {"min_kg": 0.5,  "max_kg": 1,    "price": 48},
  {"min_kg": 1,    "max_kg": 2,    "price": 70},
  {"min_kg": 2,    "max_kg": 5,    "price": 110}
]
```

```
在 tier_config 中查找 rounded_chargeable_kg 所在的区间；取对应价格。
```

### 6.4 `manual_table`（人工定价表）

保留给后续需要按目的地精确匹配行情的场景。第一期暂不详细设计。

### 6.5 最低收费（所有规则类型通用）

```
final_fee = max(fee, minimum_charge)   -- 若 minimum_charge 有值
```

### 6.6 燃油附加费

```
fuel_fee = final_fee × (fuel_surcharge_pct / 100)
```

### 6.7 总运费

```
total_shipping_fee = final_fee + surcharge_fixed + fuel_fee
```

---

## 7. 计算算法（伪代码）

```
function calculate_shipping(product_id, sku_id, quantity, destination_country, cargo_type):
    # Step 1: 解析包装数据
    pkg = resolve_package_data(product_id, sku_id)
    if pkg is incomplete:
        return error("物流数据不完整")

    # Step 2: 按数量和目的地查找可用渠道
    channels = find_active_channels(destination_country, cargo_type)
    if channels is empty:
        return error("无可用物流渠道")

    # Step 3: 批量计算
    results = []
    for channel in channels:
        # 实际重量
        actual_weight = pkg.weight × quantity

        # 体积重（单件 × 数量——第一期简化：单件打包，不优化装箱）
        vw = (pkg.length × pkg.width × pkg.height × quantity) / channel.volumetric_divisor

        # 计费重
        cw = max(actual_weight, vw)
        cw_rounded = ceil_to_increment(cw, channel.rounding_increment)

        # 运费计算
        rules = get_active_rules(channel.id, cw_rounded)
        fee = apply_rule(rules, cw_rounded)

        # 最低收费
        fee = max(fee, rules.minimum_charge) if rules.minimum_charge else fee

        # 附加费
        fee += rules.surcharge_fixed

        # 燃油附加费
        fuel = fee × (rules.fuel_surcharge_pct / 100)
        total = fee + fuel

        results.append({
            provider: channel.provider_name,
            channel: channel.name,
            actual_weight, volumetric_weight: vw,
            chargeable_weight: cw_rounded,
            base_fee: fee,
            fuel_fee: fuel,
            total_fee: total,
            currency, delivery_days, calc_detail
        })

    # Step 4: 按运费升序排序
    return sorted(results, by=total_fee)
```

---

## 8. API 草案

所有接口挂 `/api/shipping` 前缀。

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/api/shipping/providers` | 物流供应商列表 |
| `POST` | `/api/shipping/providers` | 创建物流供应商 |
| `PUT` | `/api/shipping/providers/{id}` | 更新物流供应商 |
| `DELETE` | `/api/shipping/providers/{id}` | 删除物流供应商 |
| `GET` | `/api/shipping/channels` | 物流渠道列表（可筛选 provider_id） |
| `POST` | `/api/shipping/channels` | 创建物流渠道 |
| `PUT` | `/api/shipping/channels/{id}` | 更新物流渠道 |
| `DELETE` | `/api/shipping/channels/{id}` | 删除物流渠道 |
| `POST` | `/api/shipping/import-rules` | 导入 `.xlsx` / `.csv` 物流报价表 |
| `POST` | `/api/shipping/calculate` | 运费计算（单次查询，返回多渠道对比） |
| `POST` | `/api/orders/{id}/shipping-quote` | 订单绑定运费快照 |
| `PUT` | `/api/orders/{id}/profit-inputs` | 更新订单利润输入 |

### 报价表导入

`POST /api/shipping/import-rules` 使用 `multipart/form-data` 上传字段 `file`。

必填列：

- `provider_name`
- `channel_name`
- `country_code`
- `rule_type`

可选列：

- `provider_code`
- `channel_code`
- `volumetric_divisor`
- `cargo_types`
- `currency`
- `estimated_delivery_min`
- `estimated_delivery_max`
- `priority`
- `fixed_fee`
- `per_kg_price`
- `first_kg`
- `first_price`
- `additional_kg`
- `additional_price`
- `minimum_charge`
- `surcharge_fixed`
- `fuel_surcharge_pct`
- `rounding_increment`

导入行为：

- 按供应商编码/名称复用或创建 `ShippingProvider`。
- 按供应商 + 渠道编码/名称复用或创建 `ShippingChannel`。
- 按渠道 + `country_code` 复用或创建 `ShippingZone`。
- 每行创建一条绑定 `zone_id` 的 `ShippingQuoteRule`。
- 计算运费时优先使用匹配目的地区域的规则；没有区域级规则时回退 `zone_id = NULL` 的渠道全局规则。

`POST /api/shipping/calculate` 支持两种模式：

- `mode=sku`：根据 `sku_id` 解析 SKU 包装字段，缺失时回退商品包装字段。
- `mode=manual`：直接使用请求中的 `package.length_cm`、`package.width_cm`、`package.height_cm`、`package.weight_kg` 计算运费，不写入商品或订单数据。

### 运费计算请求示例

SKU 模式：

```json
POST /api/shipping/calculate
{
  "mode": "sku",
  "sku_id": 42,
  "quantity": 2,
  "destination_country": "US",
  "postal_code": "10001",
  "cargo_type": "normal"
}
```

手动模式：

```json
POST /api/shipping/calculate
{
  "mode": "manual",
  "quantity": 2,
  "destination_country": "US",
  "postal_code": "10001",
  "cargo_type": "normal",
  "package": {
    "length_cm": 30,
    "width_cm": 20,
    "height_cm": 10,
    "weight_kg": 0.8
  }
}
```

### 运费计算响应示例

```json
{
  "mode": "sku",
  "sku_id": 42,
  "quantity": 2,
  "destination_country": "US",
  "package": {
    "source": "sku",
    "weight_kg": 0.25,
    "length_cm": 15,
    "width_cm": 10,
    "height_cm": 8
  },
  "results": [
    {
      "provider": "云途物流",
      "channel": "美国普货",
      "actual_weight_kg": 0.5,
      "volumetric_weight_kg": 0.4,
      "chargeable_weight_kg": 0.5,
      "base_fee": 29.0,
      "surcharge_fee": 0,
      "fuel_surcharge_fee": 0,
      "total_fee": 29.0,
      "currency": "CNY",
      "estimated_delivery_days": "7-15",
      "calculation_detail": "固定费8元 + 计费重0.5kg × 42元/kg = 29元"
    },
    {
      "provider": "燕文物流",
      "channel": "美国小包",
      "actual_weight_kg": 0.5,
      "volumetric_weight_kg": 0.3,
      "chargeable_weight_kg": 0.5,
      "base_fee": 32.0,
      ...
    }
  ]
}
```

---

## 9. 前端草案

### 9.1 商品编辑页 — 物流信息区块

在现有 `ProductForm.vue` 中新增"物流信息"区块，包含：

- 商品尺寸（非必填）：长度、宽度、高度、重量
- 包装尺寸（必填）：长度、宽度、高度、重量
- 货品类型选择：普通 / 带电池 / 液体 / 敏感品

### 9.2 SKU 管理页 — 物流覆盖区块

在 `SkuManage.vue` 中新增"物流覆盖"区块：

- SKU 包装长度 / 宽度 / 高度 / 重量（可选）
- 显示"未覆盖，使用商品默认"状态

### 9.3 物流供应商与渠道管理页

新页面 `frontend/src/views/shipping/`，包含：

- 供应商列表：名称、编码、渠道数、状态
- 渠道列表：名称、抛重系数、支持货品、时效、币种
- 报价规则编辑器：支持添加/编辑各类型规则
- 区域覆盖编辑器：选择国家 + 可选邮编范围

### 9.4 运费计算面板

新组件或页面：

- 选择 SKU + 数量 + 目的地国家（可选邮编）
- 触发计算，展示多渠道对比列表
- 每行展示：供应商、渠道、计费重、基础费、附加费、总费、时效
- 支持按总价/时效排序
- 可"选用此方案"→ 写入订单快照

### 9.5 订单详情运费展示

在 `OrderDetail.vue` 中新增运费快照区块：

- 选用的物流供应商 / 渠道
- 计费重、体积重、实际重量
- 费用明细和总运费

---

## 10. 测试计划

### 10.1 后端测试

| 测试场景 | 说明 |
|---|---|
| SKU 覆盖优先级 | SKU 字段非空时使用 SKU 值，否则回退商品值 |
| 缺失字段阻塞 | 包装数据缺失时计算返回明确错误 |
| 体积重大于实际重 | 抛货场景：体积重 > 实际重，计费重=体积重 |
| 实际重大于体积重 | 重货场景：实际重 > 体积重，计费重=实际重 |
| 最低收费 | 按规则算出的运费低于最低收费时使用最低收费 |
| 阶梯价格 | 各重量区间返回正确价格 |
| 首重+续重 | 多种重量验证首重续重公式 |
| 燃油附加费 | 验证百分比附加 |
| 不活跃渠道 | 禁用状态的渠道不参与计算 |
| 货品类型限制 | 电池品不匹配普货渠道时排除 |
| 多数量计算 | quantity > 1 时重量和体积正确放大 |
| 取整规则 | 验证 rounding_increment 向上取整 |
| 排序结果 | 多渠道按总价升序返回 |

### 10.2 测试文件结构

```text
backend/tests/
  test_shipping_calculation.py    # 运费计算核心逻辑
  test_shipping_provider.py       # 供应商/渠道 CRUD
  test_shipping_fallback.py       # SKU→商品回退测试
```
