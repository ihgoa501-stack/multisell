# Prism

> **AI Agent 的产品图像基础设施** —— 专为 Agent 而生，不做给人用的界面。

Prism **不是又一个图片编辑工具，也不是 PhotoRoom 的替代品**。它是第一个也只服务于 AI Agent 的产品图像引擎——把电商商品图生成做成一套代码化、可编程、纯 API 调用的设计系统。没有 GUI，没有"上传-点按钮-下载"的流程。只有 Agent 到 Agent 的调用。

---

## 一句话

**Prism 让任何 AI Agent 能在 1 秒内生成符合跨境平台规则的专业商品图，成本是 PhotoRoom 的 1/30。**

---

## 为什么是 Prism

### 问题

| 现有方案 | 问题 |
|---------|------|
| PhotoRoom / Pixelcut | 给人用的 APP，Agent 调不了；不支持 Ozon/WB |
| Remove.bg / Slazzer | 只去背景，不做生成 |
| Fal.ai / Replicate | 通用推理，无电商模板、无合规规则 |
| 自己接模型 | 每个卖家从头搭，重复造轮子 |

行业缺的是一个 **面向 Agent 的商品图基础设施层**。

### 解决方案

```
Agent 只需说一句话：
"生成这件羽绒服在 Ozon 的上架图，暖色调场景"

Prism 理解：
├── 去背景 → 商品抠出
├── 模板 → "winter_outdoor" 场景模板
├── 合规 → Ozon 主图 1000×1000, 无文字, 白底≥85%
├── 品牌 → 该卖家的 LoRA 风格
└── 渲染 → Flux Pro → 合规检查 → 输出
```

---

## 核心能力

### 1. 商品图管线引擎

```
原图 → 去背景 → 场景生成 → 尺寸裁切 → 合规检查 → 水印/Logo → 输出
```

每个环节可插拔、可配置、可独立替换。

### 2. Design System（设计系统）

不是模板库，是**代码化的设计系统**：

```yaml
# template/winter_outdoor.yaml
name: winter_outdoor
description: 冬季户外场景
prompt_template: "A {product} placed in a snowy outdoor winter scene,
                  soft natural lighting, high-end product photography,
                  shallow depth of field"
platforms:
  - ozon
  - wb
  - amazon
aspect_ratios:
  - 1:1    # 主图
  - 4:5    # 附图
  - 9:16   # 视频封面
```

### 3. 跨境合规引擎

| 平台 | 支持规格 |
|------|---------|
| Ozon | 主图 1000×1000、白底≥85%、无水印 |
| Wildberries | 正方形主图、无文字 overlay、RGB 色彩空间 |
| Amazon | 纯白底、主体≥85%、文件 ≤10MB |
| 自建站 | 可自定义规则 |

### 4. Agent-Native API

不是给人点的按钮，是给 Agent 调用的接口：

```
gRPC（主）+ HTTP REST（辅）
```

Agent 调用流程：

```go
// MultiSell Agent 内部
images, _ := prism.Generate(ctx, &GenerateRequest{
    ProductID: "p-xxx",
    ProductName: "冬季羽绒服",
    RawImages:  []string{"https://cdn.xxx/raw.jpg"},
    Template:   "winter_outdoor",
    Platform:   "ozon",
    BrandID:    "seller-123",  // → 自动应用该卖家 LoRA
})
```

---

## 市场定位

### 目标客户

| 客户 | 场景 | 价值 |
|------|------|------|
| **MultiSell（内部）** | ProductScout + ListingOptimizer 自动出图 | 第一个集成客户 |
| **跨境卖家 SaaS** | 对接其他 ERP/店铺管理工具 | 嵌入即用 |
| **电商 Agent 平台** | 需要商品图生成能力的 Agent | API 调用 |
| **独立开发者** | 二次开发、自建 pipeline | SDK + 文档 |

### 竞争差异化

```
              PhotoRoom                Prism
               ▲
               │
   给人用       │   Pixelcut
               │
               │
   给Agent用 ──┼─────────────────────► Prism
               │
               │
               │
               ▼
              Fal.ai / Replicate

              境内为主        跨境为主
```

**Prism 占据的独特位置**：**Agent-Native × 跨境优先**。

---

## 技术架构

```
Client (Agent/SDK)
    │
    │ gRPC / HTTP
    ▼
┌──────────────────────────────────────────────┐
│              Prism API Layer                  │
│  Auth → Rate Limit → Billing → Route         │
└────────────────────┬─────────────────────────┘
                     │
┌────────────────────▼─────────────────────────┐
│           Pipeline Engine                     │
│  ┌──────────┐ ┌──────────┐ ┌──────────────┐ │
│  │ RemoveBG │ │ Scene    │ │ Compliance   │ │
│  │ Provider │ │ Provider │ │ Validator    │ │
│  └──────────┘ └──────────┘ └──────────────┘ │
└────────────────────┬─────────────────────────┘
                     │
┌────────────────────▼─────────────────────────┐
│           Design System                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────────┐ │
│  │ Template │ │ Brand    │ │ Style LoRA   │ │
│  │ Registry │ │ Guideline│ │ Engine       │ │
│  └──────────┘ └──────────┘ └──────────────┘ │
└──────────────────────────────────────────────┘
                     │
┌────────────────────▼─────────────────────────┐
│           Inference Providers                 │
│  Fal.ai │ Replicate │ Stability │ ComfyUI    │
└──────────────────────────────────────────────┘
                     │
┌────────────────────▼─────────────────────────┐
│           Storage (S3/MinIO)                  │
│           输出图片、模板、LoRA                │
└──────────────────────────────────────────────┘
```

### 技术选型

| 层 | 选择 | 理由 |
|----|------|------|
| 语言 | **Go** | 高性能、低资源、跟主栈一致 |
| API | **gRPC** + HTTP REST | gRPC 传图高效，REST 方便外部 |
| 推理 | **Fal.ai**（主）→ **Replicate**（备）→ **ComfyUI**（长期） | 成本最低、可切换 |
| 去背景 | **Bria RMBG 2.0**（开源） | 免费，去背景成本 \$0 |
| 存储 | **MinIO / S3** | 图片产物必须对象存储 |
| 计费 | **基于信用点** | 行业标准（Pixelcut 已验证） |

---

## 路线图

```
Phase 1（0-2月）— MVP
├── 去背景 + Flux 场景生成 + 3 个内置模板
├── gRPC + HTTP API
├── Ozon + WB 合规检查
├── 简单按量计费
└── 集成 MultiSell 内部

Phase 2（2-4月）— 产品化
├── 模板注册中心（开放自定义模板）
├── API 公开文档 + 开发者门户
├── SDK: Go / Python / JS
├── 管理 Dashboard
└── 对外 Beta 邀请

Phase 3（4-6月）— 商业化
├── LoRA 微调服务（卖家专属风格，核心壁垒）
├── 模板市场
├── Amazon / Shopee 合规支持
├── 正式定价发布
└── 独立运营
```

---

## 定价哲学

不是 "API 按量收费"，是 **"Design System as a Service"**：

| 资源 | 定价参考 | 成本 | 毛利空间 |
|------|---------|------|---------|
| 基础去背景 | 免费 / 极低价 | $0 | 高 |
| 场景生成 | $0.01-0.02/张 | $0.003 | 3-5x |
| LoRA 训练 | $10-20/训练 | $1-2 | 5-10x |
| 模板市场 | 订阅制 | 一次性开发 | SaaS 利润 |

对比：Pixelcut API 场景生成 **$0.10/张**，Prism **$0.01-0.02/张**，便宜 5-10 倍。

---

## 为什么 Prism 能赢

1. **PhotoRoom 不做 Agent 市场** —— 它卖给人用，你卖给 Agent 用
2. **Pixelcut 不做跨境** —— 你从第一天就做 Ozon/WB
3. **Fal.ai 不做设计系统** —— 你在这个层面封装价值
4. **没有人做 Agent-Native 图像引擎** —— 你是第一个

---

## 开始

```bash
# Prism 当前处于 Phase 0 搭建阶段
# 开发者: 请参考 docs/ 目录
# 集成: 请参考 api/proto/v1/
```

> **Prism** — 给 AI Agent 一双眼睛。
