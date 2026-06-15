# 平台费用规则方案文档 (Platform Fee Rules Plan)

## 1. 目标 (Goals)

标准化计算不同平台、国家、类目的销售费用，为“上架前经营决策”提供准确的成本依据。

- **标准化**：避免人工重复输入平台费率，减少误差。
- **动态匹配**：根据选择的平台、国家（站点）和商品类目自动匹配最优费率规则。
- **可维护性**：提供后台管理页面，方便在平台政策调整时快速更新规则。
- **透明化**：在利润测算结果中清晰展示各项费用的来源和计算过程。

## 2. 非目标 (Non-goals)

- **实时 API 接入**：第一阶段不通过平台 API 实时获取费率，采用本地维护规则库。
- **复杂税务处理**：不处理复杂的增值税（VAT）抵扣逻辑，仅在费率中预留相关比例。
- **动态调价建议**：本方案仅负责费用的准确计算，不负责自动调整平台售价。

## 3. 数据模型建议 (Data Model)

建议新增 `platform_fee_rule` 表。

| 字段名 | 类型 | 说明 |
| :--- | :--- | :--- |
| id | Integer | 主键 |
| platform_id | Integer | 外键，关联 `platform` 表 |
| site_code | String | 国家/站点代码 (如 RU, MY, BR) |
| category_id | Integer | 可选，外键，关联 `category` 表 (类目级佣金) |
| commission_pct | Decimal | 平台佣金比例 (%) |
| payment_fee_pct | Decimal | 支付/手续费比例 (%) |
| fixed_fee | Decimal | 固定交易费/订单处理费 (每单) |
| advertising_pct | Decimal | 广告/营销预留比例 (%) |
| other_reserve_fee | Decimal | 其他预留成本 (固定金额) |
| status | Integer | 状态 (1: 生效, 0: 失效) |
| remark | Text | 备注 (如规则版本或政策来源) |
| created_at | DateTime | 创建时间 |
| updated_at | DateTime | 更新时间 |

## 4. API 建议 (API Design)

### 4.1 规则管理 (CRUD)
- `GET /api/platform-fee-rules`：列表查询（支持按平台、站点、类目筛选）。
- `POST /api/platform-fee-rules`：创建规则。
- `GET /api/platform-fee-rules/{id}`：详情查询。
- `PUT /api/platform-fee-rules/{id}`：更新规则。
- `DELETE /api/platform-fee-rules/{id}`：删除规则。

### 4.2 规则匹配 (Internal/Public)
- `POST /api/platform-fee-rules/match`：根据平台、站点、类目匹配最佳规则。
    - **匹配优先级**：(平台+站点+类目) > (平台+站点) > (平台全局)。

## 5. 与 Pre-listing Decision 集成方式

目前 `PreListingDecisionRequest` 接收手动的费率参数。改进方案如下：

1. **请求参数扩展**：
   ```python
   class PreListingDecisionRequest(BaseModel):
       sku_id: int
       destination_country: str
       target_sale_price: float
       platform_id: Optional[int]  # 新增：指定平台
       category_id: Optional[int]  # 新增：指定类目（或从SKU自动获取）
       # 以下参数保留，若指定了 platform_id 则优先从规则库匹配，未匹配到时使用传入值
       platform_fee_pct: Optional[float] 
       payment_fee_pct: Optional[float]
       # ...
   ```

2. **逻辑流程**：
   - 步骤 1：根据 `platform_id`, `destination_country`, `category_id` 调用 `PlatformFeeRuleService.match()`。
   - 步骤 2：如果匹配成功，使用规则中的 `commission_pct`, `payment_fee_pct`, `fixed_fee`, `advertising_pct`, `other_reserve_fee` 进行计算。
   - 步骤 3：如果匹配失败，退而求其次使用请求中的默认参数，并在响应的 `warnings` 中提示“未匹配到预设规则，使用默认/手动输入费率”。
   - 步骤 4：在 `PreListingDecisionResponse` 中增加 `rule_id` 或 `applied_rule_summary` 字段，方便前端展示。

## 6. 前端交互 (Frontend)

1. **规则管理页**：
   - 物流管理类似的表格界面。
   - 支持批量导入/导出（后续扩展）。
   - 快速复制规则（例如从 RU 站点复制规则到 BY 站点）。

2. **利润测算面板（集成）**：
   - 用户选择“平台”后，系统自动带入该平台在该国家的预设费率。
   - 允许用户手动覆盖（Override）自动匹配的费率。
   - 鼠标悬停在费用项上时，显示“来自预设规则：[规则名称/备注]”。

## 7. 测试计划 (Test Plan)

- **单元测试 (Unit Tests)**：
    - 测试匹配算法：确保优先级逻辑正确（类目级规则覆盖全局规则）。
    - 测试计算准确性：各种比例和固定费叠加后的总成本计算。
- **集成测试 (Integration Tests)**：
    - 模拟从 `PreListingDecisionService` 调用规则匹配。
    - 验证当规则失效 (status=0) 时不会被匹配到。
- **边界测试**：
    - 售价为 0 或负数的情况。
    - 费率超过 100% 的极端情况。

## 8. 迁移计划 (Migration Plan)

1. **Schema 迁移**：使用 Alembic 创建 `platform_fee_rule` 表。
2. **数据初始化**：预置主流平台（如 Ozon, Shopee）的通用默认规则。
3. **代码重构**：更新 `PreListingDecisionService`，引入规则查找逻辑，保持对旧 API 的向后兼容性（如果未传 platform_id 则继续使用旧逻辑）。

## 9. 风险和分阶段交付 (Risks & Phased Delivery)

### 风险：
- **规则维护成本**：平台政策变动频繁，需要专人或 Agent 负责更新。
- **类目匹配复杂性**：不同平台的类目体系不同，建立统一的 `category_id` 匹配可能存在难度。

### 分阶段交付：
- **Phase 1**：实现 `platform_fee_rule` 基础表结构和 CRUD API，支持“平台+站点”匹配。
- **Phase 2**：支持“类目”级精细费率匹配。
- **Phase 3**：在利润测算 UI 中完整集成自动匹配与手动覆盖功能。
