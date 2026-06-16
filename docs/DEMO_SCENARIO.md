# 凌镜 LingMirror Demo 场景

本文档说明如何使用 Stage 13 Demo Seed / Sandbox Scenario 的模拟数据，
在无真实业务数据的情况下演示完整的核心业务闭环。

---

## 1. 环境准备

### 1.1 确保 PostgreSQL 运行

```bash
docker ps | grep postgres
```

如果未运行，从项目根目录启动：

```bash
cd /Users/lc/multisell
docker compose up -d
```

### 1.2 确保测试数据库存在

```bash
psql -h localhost -U postgres -c "CREATE DATABASE product_management_test;" 2>/dev/null || true
```

### 1.3 确保依赖已安装

```bash
cd /Users/lc/multisell/backend
source .venv/bin/activate
pip install -r requirements.txt
```

---

## 2. 加载 Demo Seed 数据

```bash
cd /Users/lc/multisell/backend
python scripts/load_demo_data.py
```

预期输出：

```
=======================================================
  凌镜 LingMirror Demo 数据加载
=======================================================

📦 数据库: postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test
🔄 连接中...
✔ 数据表已就绪

📋 加载 demo 数据...
  ├─ 确保 admin 用户...
  ├─ 确保 demo 用户...
   │   ✔ demo / demo123 已创建
  ├─ 确保权限码...
  ...

=======================================================
  ✅ Demo 数据加载完成！
=======================================================

products created/updated: 7/0
skus created/updated: 14/0
inventory seeded: 14
shipping rules seeded: 4
platform fee rules seeded: 6
demo csv paths: .../docs/demo-data/order_import_demo.csv, ...
```

> 脚本是幂等的，可重复执行。第二次执行不会创建重复数据。

### 加载了什么数据

| 实体 | 数量 | 说明 |
|---|---|---|
| 用户 | 2 | admin / demo（密码同用户名 + "123"） |
| 商品 | 7 个 | 蓝牙耳机、智能手表、夹克、保温杯、精华液、坚果礼盒、瑜伽垫 |
| SKU | 14 个 | 每商品 2 个 SKU |
| 库存 | 14 条 | 每 SKU 对应一个库存记录 |
| 物流供应商 | 3 家 | CDEK, Russian Post, Shopee Xpress |
| 物流渠道 | 4 个 | CDEK Economy, CDEK Standard, EMS, Standard |
| 报价规则 | 4 条 | 每渠道 1 条 fixed_plus_per_kg 规则 |
| 平台费用规则 | 6 条 | Ozon/Shopee/Wildberries 各两条 |
| 供应商 | 3 家 | 电子/服装/家居 |
| 权限码 | 30+ | 覆盖所有业务模块 |

---

## 3. 启动后端

```bash
cd /Users/lc/multisell/backend
source .venv/bin/activate
uvicorn app.main:app --reload --port 8000
```

后端运行在 `http://localhost:8000`。

---

## 4. 启动前端

另开终端：

```bash
cd /Users/lc/multisell/frontend
npm install
npm run dev
```

前端运行在 `http://localhost:5173`（或终端提示的地址）。

---

## 5. Demo 操作流程

### 5.1 CSV 订单导入

打开：**订单导入 → 上传 CSV**

文件路径：`docs/demo-data/order_import_demo.csv`

上传后查看批次详情，应看到：
- 7 行记录，创建 6 个订单
- OZ-10001 为 2 个 SKU 合并的同一订单
- WB-30002 有运费但无运费快照（因为瑜伽垫 seed 未创建运费快照）

### 5.2 处理经营链路

在订单导入批次详情页，点击 **处理链路（Process Chain）**。

预期结果：
- 账本重建成功
- 异常生成成功
- 部分负利润订单（WB-30001 坚果）触发异常

### 5.3 运费账单导入

打开：**物流 → 运费账单 → 导入**

文件路径：`docs/demo-data/shipping_bill_demo.csv`

导入后点击 **对账**，预期结果：
- RUS-TRK-001: matched（金额差异 ≤ 阈值）
- RUS-TRK-002: amount_mismatch（账单含附加费）
- SP-20002: matched（按 order_no 匹配）
- RUS-TRK-003: amount_mismatch（金额差异）
- UNMATCHED-BILL-001: unmatched_bill

### 5.4 平台结算导入

打开：**平台结算 → 导入**

文件路径：`docs/demo-data/platform_settlement_demo.csv`

导入后应看到 18 行记录，其中：
- 大多数行 matched（有对应平台订单号）
- UNMATCHED-SETTLEMENT-001: unmatched

### 5.5 查看利润看板

打开：**财务 → 利润看板**

预期可看到：
- 利润摘要卡片（总收入、成本、费用、利润）
- 订单利润列表（6 个订单）
- 运费差异报表（matched / amount_mismatch）
- 负利润订单列表（WB-30001 坚果应为负利润）
- 成本层分布统计

### 5.6 查看异常工作台

打开：**异常工作台**

点击 **扫描生成异常**，可看到：
- 订单负利润异常（WB-30001）
- 运费账单 unmatched（UNMATCHED-BILL-001）
- 平台结算 unmatched（UNMATCHED-SETTLEMENT-001）

### 5.7 运费重算与快照

打开：**订单 → 订单详情 → 选择任一订单 → 运费报价**

可重新试算运费并保存运费快照。

### 5.8 上架前利润测算

打开：**决策 → 上架前决策**

选择任一 Demo SKU（如 DEMO-BT-BLACK），填目标售价和数量，可进行利润测算。

---

## 6. 重跑 Demo

### 清空数据重跑

```bash
cd /Users/lc/multisell/backend
python scripts/db_reset.py    # 如果有重置脚本
# 或直接清表：
psql -h localhost -U postgres -d product_management_test -c "
DO \$\$ DECLARE r RECORD; BEGIN FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname='public') LOOP EXECUTE 'TRUNCATE TABLE ' || quote_ident(r.tablename) || ' CASCADE'; END LOOP; END \$\$;"
python scripts/load_demo_data.py
```

### 只重跑 demo 数据（不清空已有配置）

```bash
cd /Users/lc/multisell/backend
python scripts/load_demo_data.py
```

脚本幂等，不会创建重复 SKU、品牌、分类等。

---

## 7. 验证

### 后端测试

```bash
cd /Users/lc/multisell/backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test \
  .venv/bin/python -m pytest tests/test_demo_seed.py -q
```

### 前端构建

```bash
cd /Users/lc/multisell/frontend
npm run build
```
