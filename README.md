# 🌐 MultiSell — AI跨境电商商品中台

基于 **Python FastAPI + Vue 3 + PostgreSQL** 的跨境电商商品中台。

## 核心定位

**商品在这里创建 → AI加工 → 一键发布到多个平台。**

## 功能模块

| 模块 | 说明 |
|------|------|
| 商品管理 | 商品CRUD、批量操作、Excel导入导出、复制 |
| 分类管理 | 无限级分类树 |
| 品牌管理 | 品牌增删改查 |
| 规格与SKU | 规格定义、笛卡尔积自动生成SKU |
| 价格管理 | 多类型价格、批量调价、调价记录 |
| 库存管理 | 库存更新、安全库存预警、库存变动记录 |
| 供应商管理 | 供应商档案、商品-供应商绑定 |
| 平台管理 | 配置Ozon/Shopee等多平台API密钥 |
| 发布管理 | 一键发布商品到多平台、发布状态追踪 |
| AI增强 | AI生成商品标题/描述/SEO关键词 |
| 全局搜索 | 搜索商品/SKU/供应商（快捷键 `/`） |
| 仪表盘 | 数据总览、平台发布统计、近期动态 |
| 操作日志 | 系统操作审计记录 |

## 快速启动

### Docker一键启动

```bash
docker-compose up -d
```

访问 http://localhost:3001

### 本地开发

**后端：**
```bash
cd backend
pip install -r requirements.txt
uvicorn app.main:app --reload --port 8001
```

API文档：http://localhost:8001/docs

**前端：**
```bash
cd frontend
npm install
npm run dev
```

访问 http://localhost:3001

## 技术栈

- 后端：Python 3.11+ / FastAPI / SQLAlchemy 2.0 / PostgreSQL
- 前端：Vue 3 / TypeScript / Naive UI / Pinia
- 部署：Docker / Docker Compose / Nginx
