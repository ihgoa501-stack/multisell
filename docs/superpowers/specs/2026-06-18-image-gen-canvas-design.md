# 商品图片生成器 — 无限画布设计文档

> 日期：2026-06-18
> 状态：待实施

## 一句话说明

在现有 AI 生图能力基础上，新增一个 Fabric.js 无限画布页面，支持商品多角度生图、图片编辑（inpaint/outpaint/去背景）、视频生成（slideshow/AI video），所有生成结果在画布里自由排列编辑。

## 整体架构

```
frontend/src/views/image_gen_canvas/
├── CanvasEditor.vue            # 主视图
├── components/
│   ├── FabricCanvas.vue        # 无限画布核心
│   ├── SideToolbar.vue         # 左侧工具栏
│   ├── ImagePanel.vue          # 右侧生成结果面板
│   └── VideoPanel.vue          # 视频时间线控制
└── composables/
    ├── useFabricCanvas.ts      # 画布状态
    └── useGeneration.ts        # 生成调用逻辑

backend/app/image_gen/
├── canvas_schemas.py           # 新增：画布数据模型
├── canvas_service.py           # 新增：画布 CRUD
├── canvas_router.py            # 新增：画布 API
├── schemas.py                  # 扩展：inpaint/outpaint/video 模型
├── service.py                  # 扩展：inpaint/outpaint/video 方法
└── router.py                   # 扩展：新增端点
```

## 前端路由

- 新页面：`/image-gen-canvas`，路由入口 `frontend/src/router/modules/imageGenCanvas.ts`
- 保留现有 `/image-gen` 页面

## 无限画布核心

### Fabric.js 配置

- 库：`fabric@6` + `@types/fabric`
- 初始尺寸：撑满容器，4000×4000 虚拟空间（视口区域展示，拖拽平移浏览）
- 缩放范围：0.1x - 5x（鼠标滚轮缩放）
- 平移：鼠标拖拽空白区域
- 网格：20px 浅灰背景网格，对齐吸附
- 选中：蓝色边框 + 8 个控制手柄

### 交互操作

| 操作 | 实现 |
|------|------|
| 从右侧拖入图片 | `fabric.Image.fromURL` -> `canvas.add()` |
| 拖拽移动 | Fabric 默认 |
| 旋转/缩放 | Fabric 控制手柄 |
| 删除 | Delete 键 / 右键菜单 |
| 复制 | Ctrl+D / 右键菜单 |
| 置顶/置底 | `bringToFront` / `sendToBack` |
| 锁定 | `lockMovementX/Y = true` |
| 导出画布 | `canvas.toDataURL({multiplier: 2})` |

### 右键菜单

- 置顶 / 置底
- 复制图层
- 删除
- 锁定位置
- 设为主图
- 局部重绘（进入 mask 模式）
- 导出此图层

### 编辑流程

**局部重绘（Inpaint）：**
1. 选中图片 → 右键 → 局部重绘
2. 画红色半透明区域标记 mask
3. 输入 prompt → 确认 → API 调用 → 新图替换原图

**扩图（Outpaint）：**
1. 选中图片 → 右键 → 扩图
2. 选择方向（上/下/左/右）→ 输入 prompt
3. 确认 → API 调用 → 新图替换原图

**去背景：**
1. 选中图片 → 右键 → 去背景
2. API 调用 → 透明背景图替换原图

## 后端 API

### 画布管理

| 方法 | 路径 | 功能 |
|------|------|------|
| POST | `/api/image-gen/canvas` | 创建/更新画布 |
| GET | `/api/image-gen/canvas/:id` | 加载画布 |
| GET | `/api/image-gen/canvas` | 画布列表（按 product_id 筛选）|
| DELETE | `/api/image-gen/canvas/:id` | 删除画布 |

### 图片编辑

| 方法 | 路径 | 功能 |
|------|------|------|
| POST | `/api/image-gen/inpaint` | 局部重绘（image_url + mask_base64 + prompt）|
| POST | `/api/image-gen/outpaint` | 扩图（image_url + direction + prompt）|

### 视频生成

| 方法 | 路径 | 功能 |
|------|------|------|
| POST | `/api/image-gen/video` | AI 视频生成（prompt + image_url 可选）|
| POST | `/api/image-gen/video/slideshow` | 图片序列合成视频 |
| GET | `/api/image-gen/video/status/:id` | 轮询视频进度 |

### 数据库模型

```python
class ProductCanvas(Base):
    __tablename__ = "product_canvases"
    id: Mapped[int] = mapped_column(primary_key=True)
    product_id: Mapped[int] = mapped_column(ForeignKey("products.id"))
    name: Mapped[str] = mapped_column(String(200), default="未命名画布")
    layers: Mapped[dict] = mapped_column(JSON)  # Fabric JSON
    thumbnail: Mapped[Optional[str]]
    created_by: Mapped[int] = mapped_column(ForeignKey("users.id"))
    created_at: Mapped[datetime] = mapped_column(server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(server_default=func.now(), onupdate=func.now())
```

## Provider 策略

| 功能 | 实现方式 |
|------|----------|
| 生图 | Replicate FLUX.2 Pro / OpenAI DALL-E 3（现有）|
| 去背景 | remove.bg API（现有） |
| 局部重绘 | Replicate FLUX.2 Pro `image` + `mask` |
| 扩图 | Replicate FLUX outpaint |
| AI 视频 | Replicate `stability-ai/stable-video-diffusion` |
| Slideshow | FFmpeg 后端合成 |

## 视频生成细节

### Slideshow 模式
1. 用户在画布选中帧图片（或传 image_urls 列表）
2. 配置：每帧时长（默认 2s）、转场效果
3. 后端 FFmpeg 合成：`-framerate 1/2 -pattern_type glob` 交叉淡入淡出
4. 返回预览 URL 和下载

### AI Video 模式
1. 可选起始图片作为第一帧
2. 输入 prompt（如 "product rotating 360 degrees on white background"）
3. 调用 Replicate API，轮询 prediction 直至完成
4. 返回 MP4 URL

## 实施顺序

1. FabricCanvas 无限画布（缩放/平移/添加/删除图片）
2. SideToolbar + 商品搜索 + 连接现有生图 API
3. ImagePanel（生成结果展示，拖入画布）
4. 画布保存/加载 API + 模型
5. Inpaint/Outpaint API + 前端 mask 交互
6. 去背景 API 调用集成
7. VideoPanel + Slideshow 生成
8. AI 视频生成
