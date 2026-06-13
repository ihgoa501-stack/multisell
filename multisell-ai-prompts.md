# 🤖 MultiSell — 多AI协作开发 Prompt 模板

## 通用说明（发给每个AI前加上这段话）

```
你是一个全栈开发者，负责 MultiSell 跨境电商商品中台的一个功能模块。
MultiSell 技术栈：Python FastAPI + Vue 3 + Naive UI + PostgreSQL

协作规则：
1. 你只能在 feat/xxx 分支上工作，不要改 main 分支
2. 你只能修改指定目录下的文件，不要改别人的模块
3. 公共文件（main.py、router/index.ts、Layout.vue）如果必须改，只追加内容不要删别人的
4. 数据库模型在 backend/app/models.py，新增表就在文件末尾加
5. 所有API响应格式统一使用 Result.ok(data) / PageResult.ok(records, total)
6. 前端API调用使用 src/api/index.ts 里的封装，使用 src/api/http.ts 的 axios 实例

项目地址：https://github.com/ihgoa501-stack/multisell.git
```

---

## AI-1：订单管理 — feat/order-management

### 任务
为 MultiSell 新增订单管理模块。支持销售订单的创建、管理、状态流转。

### 改哪些文件

| 文件 | 说明 |
|------|------|
| `backend/app/models.py` | 末尾加 Order、OrderItem 表 |
| `backend/app/order/` | 新建整个模块（schema/service/router 各一个） |
| `backend/app/main.py` | 注册 order_router（只加一行 import + 一行 include_router） |
| `frontend/src/views/order/` | 新建订单列表页、订单详情页 |
| `frontend/src/router/index.ts` | 加 order 路由（只追加一行） |
| `frontend/src/components/Layout.vue` | 菜单加"订单管理"（只追加一个菜单项） |
| `frontend/src/api/index.ts` | 加 orderApi（只追加一段） |

### 数据库表设计

```python
class Order(Base):
    __tablename__ = "order"
    id, order_no(订单号), product_id, sku_id, quantity, total_price, 
    status(draft/pending/paid/shipped/delivered/cancelled), 
    buyer_name, buyer_phone, buyer_address, remark, operator, 
    created_at, updated_at

class OrderItem(Base):
    __tablename__ = "order_item"
    id, order_id(FK), sku_id(FK), product_name, sku_desc, 
    price, quantity, subtotal
```

### API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/orders | 创建订单 |
| GET | /api/orders | 订单列表（分页、按状态筛选） |
| GET | /api/orders/{id} | 订单详情 |
| PUT | /api/orders/{id}/status | 更新订单状态（发货/取消等） |

### 前端页面

- **订单列表页**：表格展示（订单号、商品、数量、金额、状态、时间），支持按状态筛选
- **订单详情页**：订单基本信息 + 商品明细列表 + 状态流转记录，支持"发货"操作按钮

---

## AI-2：数据分析报表 — feat/analytics-report

### 任务
为 MultiSell 新增数据分析报表页面，展示商品销量、库存健康度、平台发布统计等图表。

### 改哪些文件

| 文件 | 说明 |
|------|------|
| `backend/app/dashboard/service.py` | 扩展 stats 接口加上销量数据 |
| `backend/app/dashboard/router.py` | 新增报表相关接口（只追加） |
| `frontend/src/views/dashboard/Dashboard.vue` | 首页加图表展示区域 |
| `frontend/src/views/report/` | 新建报表页面 |
| `frontend/src/router/index.ts` | 加 report 路由 |
| `frontend/src/components/Layout.vue` | 菜单加"数据报表" |
| `frontend/src/api/index.ts` | 加 reportApi |

### 后端新增接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/reports/product-stats | 商品统计（总数/上架/草稿/下架/各分类分布） |
| GET | /api/reports/platform-stats | 平台发布统计（各平台已发布/待发布/失败数量） |

### 前端页面

- **数据报表页**（/reports）：
  - 商品分布饼图（用简单CSS实现或echarts，不要引入太重库）
  - 平台发布状态柱状图
  - 库存健康度环形图
  - 带时间筛选器
  
### 注意
- 不要引入新的图表库，用 CSS/SVG 实现简单图表，或者用 Naive UI 的进度条/卡片展示

---

## AI-3：接入真实AI（LLM） — feat/ai-real-llm

### 任务
将当前 mock 的 AI 增强接口替换为调用真实 LLM API（OpenAI 兼容接口）。

### 改哪些文件

| 文件 | 说明 |
|------|------|
| `backend/app/core/router.py` | 修改 ai-enhance 接口，调用真实的LLM |
| `backend/app/config.py` | 加 LLM_API_KEY、LLM_API_URL、LLM_MODEL 配置 |
| `backend/.env` | 加环境变量（不提交） |

### 实现方案

当前 mock 代码在 `backend/app/core/router.py` 的 `ai_enhance_product` 函数里（约300行左右）：

```python
@router.post("/products/{product_id}/ai-enhance", summary="AI优化商品信息")
async def ai_enhance_product(product_id: int, db: AsyncSession = Depends(get_db)):
```

改成调用 OpenAI 兼容 API：

```python
import httpx

async def call_llm(prompt: str) -> str:
    async with httpx.AsyncClient() as client:
        resp = await client.post(
            settings.LLM_API_URL or "https://api.openai.com/v1/chat/completions",
            headers={"Authorization": f"Bearer {settings.LLM_API_KEY}"},
            json={
                "model": settings.LLM_MODEL or "gpt-4o-mini",
                "messages": [{"role": "user", "content": prompt}],
                "temperature": 0.7,
            },
            timeout=30,
        )
        return resp.json()["choices"][0]["message"]["content"]
```

- 支持通过环境变量配置 API 地址、Key、模型名
- 增加失败重试（3次）
- 没有配置 Key 时回退到 mock 数据
- 生成的标题/描述/关键词改用一个 prompt 一次性返回，然后解析 JSON

### Prompt 设计

```python
PROMPT_TEMPLATE = """
你是一个电商商品标题和描述优化专家。
请根据以下商品信息，生成优化的标题、描述和SEO关键词。

商品名称：{name}
商品副标题：{subtitle}
商品描述：{description}

请以JSON格式返回：
{{
  "title": "优化后的商品标题（不超过200字）",
  "description": "优化后的商品描述（包含产品特色、卖点、适合场景，200-500字）",
  "keywords": ["关键词1", "关键词2", "关键词3", "关键词4", "关键词5"]
}}
只返回JSON，不要额外说明。
"""
```

---

## AI-4：响应式布局 — feat/mobile-responsive

### 任务
让 MultiSell 的前端在手机/平板上也能正常使用。

### 改哪些文件

| 文件 | 说明 |
|------|------|
| `frontend/src/components/Layout.vue` | 侧边栏改为折叠式汉堡菜单 |
| `frontend/src/views/**/*.vue` | 所有页面的表格/表单适配小屏 |
| `frontend/index.html` | 加 viewport meta |

### 实现要点

1. **Layout.vue**：
   - 屏幕 < 768px 时：侧边菜单自动折叠成顶部汉堡图标
   - 点击汉堡图标展开/收起菜单（浮层覆盖）
   - 搜索框在小屏时缩短宽度或隐藏

2. **表格页**（ProductList, SupplierList, 等）：
   - 小屏时表格自动水平滚动
   - 操作按钮列保持可点

3. **表单页**（ProductForm）：
   - 小屏时标签在上、输入框在下（不用左右布局）
   - 表单宽度100%

4. **naive-ui 自适应**：利用 n-grid 的 responsive 属性控制列数

### 注意
- 不要改功能逻辑，只改样式和布局
- 用 Naive UI 内置的响应式属性，不要手写 media query 除非必要
- 测试断点：手机 375px、平板 768px、桌面 1200px+
