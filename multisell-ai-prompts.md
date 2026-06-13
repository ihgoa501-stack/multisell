# 🤖 MultiSell — 多AI协作开发 Prompt 模板

## 通用说明（发给每个AI前加上这段话）

```
你是一个全栈开发者，负责 MultiSell 跨境电商商品中台的一个功能模块。
MultiSell 技术栈：Python FastAPI + Vue 3 + Naive UI + PostgreSQL

协作规则（重要）：
1. 你只能在你的 feat/xxx 分支上工作，不要改 main 分支
2. 你只能修改指定目录下的文件，不要改别人的模块

⚠️ 核心规则：以下文件你绝对不要改！
  - backend/app/main.py           → 自动注册所有路由，你只需建模块即可
  - frontend/src/router/index.ts  → 自动加载 router/modules/*.ts
  - frontend/src/components/Layout.vue → 自动扫描路由 meta 生成菜单
  - frontend/src/api/index.ts     → 自动合并 api/modules/*.ts

3. 要加新模块，用以下方式（不需要改任何公共文件）：
   a) 后端：新建 backend/app/<模块名>/ 目录，包含 router.py + __init__.py + schemas.py + service.py
      确保 __init__.py 里 export router 变量（from app.xxx.router import router）
      → 路由自动注册到 /api 前缀下

   b) 前端路由：在 frontend/src/router/modules/ 下新建 <模块名>.ts
      导出 routes: RouteRecordRaw[] 数组
      设置 meta: { menu: true, icon: 'xxx' } 让菜单自动出现
      → 不用改 router/index.ts

   c) 前端API：在 frontend/src/api/modules/ 下新建 <模块名>.ts
      export const xxxApi = { ... } (用 http.get/post/put/delete)
      → 自动合并到 apiModules 对象

4. 数据库模型在 backend/app/models.py，新增表就在文件末尾加
5. 所有API响应格式统一使用 Result.ok(data) / PageResult.ok(records, total)
6. 前端API调用使用 import http from '@/api/http' 的 axios 实例
7. 可用图标 key：home / list / layers / palette / cash / archive / people / tag
   doc-text / warning / globe / cart / chart / cube / settings / trend / shield / download
   如需新图标，在 Layout.vue 的 iconMap 里加一行

项目地址：https://github.com/ihgoa501-stack/multisell.git
```

---

## AI-1：订单管理后端 — feat/order-management

### 任务
为 MultiSell 新增订单管理后端模块。支持销售订单的创建、管理、状态流转。

### 改哪些文件

| 文件 | 说明 |
|------|------|
| `backend/app/models.py` | 末尾加 Order、OrderItem 表 |
| `backend/app/order/` | **新建**整个模块（__init__.py / router.py / schemas.py / service.py） |
| （不改 main.py，自动注册） | |

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

---

## AI-2：订单管理前端 — feat/order-frontend

### 任务
为订单管理模块创建前端页面。

### 改哪些文件

| 文件 | 说明 |
|------|------|
| `frontend/src/views/order/` | **新建**订单列表页、订单详情页 |
| `frontend/src/router/modules/order.ts` | **新建**路由配置（包含菜单meta） |
| `frontend/src/api/modules/order.ts` | **新建**API 定义 |
| （不改 router/index.ts、不改 Layout.vue） | |

### 页面要求

- **订单列表页（/orders）**：表格展示（订单号、商品、数量、金额、状态、时间），支持按状态筛选
- **订单详情页（/orders/:id）**：订单基本信息 + 商品明细列表 + 状态流转记录，支持"发货"操作按钮

---

## AI-3：数据分析报表后端 — feat/analytics-backend

### 任务
为 MultiSell 扩展仪表盘统计接口，增加报表数据 API。

### 改哪些文件

| 文件 | 说明 |
|------|------|
| `backend/app/dashboard/service.py` | 扩展 stats 接口（只追加） |
| `backend/app/dashboard/router.py` | 新增报表相关接口（只追加） |
| （不改 main.py） | |

### 新增接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/reports/product-stats | 商品统计（总数/上架/草稿/下架/各分类分布） |
| GET | /api/reports/platform-stats | 平台发布统计（各平台已发布/待发布/失败数量） |

---

## AI-4：数据分析报表前端 — feat/analytics-frontend

### 任务
为报表模块创建前端页面。

### 改哪些文件

| 文件 | 说明 |
|------|------|
| `frontend/src/views/report/` | **新建**报表页面 |
| `frontend/src/router/modules/report.ts` | **新建**路由配置 |
| `frontend/src/api/modules/report.ts` | **新建** API 定义 |
| （不改 router/index.ts、不改 Layout.vue） | |

### 页面

- **数据报表页（/reports）**：
  - 商品分布饼图（用简单CSS实现或echarts，不要引入太重库）
  - 平台发布状态柱状图
  - 库存健康度环形图
  - 带时间筛选器

---

## AI-5：接入真实AI（LLM） — feat/ai-real-llm

### 任务
将当前 mock 的 AI 增强接口替换为调用真实 LLM API（OpenAI 兼容接口）。

### 改哪些文件

| 文件 | 说明 |
|------|------|
| `backend/app/core/router.py` | 修改 `ai_enhance_product` 函数，调用真实 LLM |
| `backend/app/config.py` | 加 LLM_API_KEY、LLM_API_URL、LLM_MODEL 配置 |

### 实现方案

```python
# 改为调用 OpenAI 兼容 API
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

## AI-6：响应式布局 — feat/mobile-responsive

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

---

## AI-7：Excel导入/导出 — feat/excel-export

### 任务
新增商品批量操作：Excel 导入商品、Excel 导出商品列表/SKU数据。

### 改哪些文件

| 文件 | 说明 |
|------|------|
| `backend/app/core/router.py` | 加 export/import 接口（追加在文件末尾） |
| `backend/requirements.txt` | 加 openpyxl 依赖 |
| `frontend/src/views/product/ProductList.vue` | 加"导出"按钮 |
| `frontend/src/router/modules/excel.ts` | 如需独立页面则新建 |
| `frontend/src/api/modules/excel.ts` | **新建**API 定义 |

### 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/products/export | 导出商品列表为 Excel |
| POST | /api/products/import | 从 Excel 导入商品 |
| GET | /api/products/export-template | 下载导入模板 |

---

## AI-8：权限管理（RBAC） — feat/rbac

### 任务
新增用户角色权限管理：角色定义、权限分配、API 权限校验。

### 改哪些文件

| 文件 | 说明 |
|------|------|
| `backend/app/models.py` | 加 Role、Permission 表 |
| `backend/app/auth/` | 扩展已有 auth 模块（加 role 管理） |
| `backend/app/rbac/` | **新建**RBAC 模块 |
| `backend/app/common/schemas.py` | 加权限相关 Schema |
| `frontend/src/views/rbac/` | **新建**用户管理/角色管理页面 |
| `frontend/src/router/modules/rbac.ts` | **新建**路由 |
| `frontend/src/api/modules/rbac.ts` | **新建**API |

### 注意
- 加配置开关 `AUTH_ENABLED=true/false` 控制是否开启权限校验
- 默认 AUTH_ENABLED=false，所有人可访问（兼容其他AI开发的模块）
- 最后再开启，避免影响其他人调试

---

## AI-9：数据Seeder/维护工具 — feat/seeder

### 任务
编写数据初始化脚本和运维工具：创建初始 admin 用户、种子数据、数据库迁移帮助。

### 改哪些文件

| 文件 | 说明 |
|------|------|
| `backend/seed.py` | **新建**种子数据脚本（运行时独立，不介入 main.py） |
| `backend/scripts/` | **新建**运维工具目录 |
| `backend/README.md` | 加数据初始化说明 |

### Seed 脚本内容

- 创建默认管理员账号（admin / admin123）
- 创建演示分类数据（服装/电子/家居等）
- 创建示例平台（Ozon、Shopee、Wildberries）
- 创建示例品牌
