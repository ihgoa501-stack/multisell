# ToolBridge 工具执行桥接

> 相关模块：`internal/platform/toolbridge/`

## 概述

ToolBridge 是 Agent 运行外部工具的桥接层，采用**插件驱动**模式。它管理一组 `ToolDriver`，按权重优先级尝试执行请求，当首选 driver 失败时自动降级。

```
Agent → ToolBridge → Driver(权重1) → fallthrough → Driver(权重2) → ... → error
```

## 添加新的 ToolDriver

1. 在 `drivers/` 下创建新的 driver 文件，实现 `ToolDriver` 接口：

```go
type ToolDriver interface {
    FetchPage(ctx context.Context, url string) (*PageData, error)
    Health() (available bool, latency time.Duration, err error)
}
```

2. 在 `bridge.go` 中用 `NewToolBridge` 注册带权重的 drivers：

```go
bridge := toolbridge.NewToolBridge([]toolbridge.DriverEntry{
    {Name: "plugin", Driver: pluginDriver, Weight: 10},
    {Name: "playwright", Driver: playwrightDriver, Weight: 20},
}, 30*time.Second, logger)
```

权重越低优先级越高。当权重 10 的 driver 不可用或超时，自动降级到权重 20 的 driver。

## 内置 Driver

目前实现了一个 **PluginDriver** (`drivers/plugin_driver.go`)：

- 通过 WebSocket 向 Chrome 扩展发送采集指令
- 扩展端从 1688 详情页提取商品信息
- 通过 WebSocket 回传结构化数据

## 架构

```
ToolBridge 生命周期：
1. NewToolBridge(drivers) — 按权重排序，创建桥接
2. Bridge.FetchPage(ctx, url) — 按权重顺序尝试 driver
3. Driver.FetchPage() — 如果成功，返回 PageData
4. Driver 全部失败 — 返回错误，无数据

上下文传递：
Agent → (context + URL) → ToolBridge → Driver → PageData
```

PageData 结构（同 sourcing.PageData）包含标题、价格、图片、规格、供应商等 20+ 个字段。
