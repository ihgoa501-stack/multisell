# Prism 运行集成（已退役）

> 状态：`superseded`
> 退役日期：2026-07-12

MultiSell 已移除旧 Prism HTTP 客户端、`trigger-prism` 触发器、Listing/loop 运行注入及 `PRISM_*` 配置。生产图片执行走 `services/image-service/`，经营审批和发布放行由 `internal/domain/productimage/` 负责。

保留边界：

- 历史图片表与 `imagegen` 历史读取不因本次退役而删除；
- 独立 `/Users/lc/prism` 仓库保留为历史实现参考，不是 MultiSell 运行依赖；
- 旧 `POST /api/v1/product-analysis/trigger-prism` 继续返回 404；
- Listing 生产发布在缺少有效图片放行凭证时失败关闭，不会回退到旧 Prism 或原图继续发布。

当前合同见 [Image Service 与 MCP 技术合同](features/image-service-mcp-contract.md)，迁移证据见 [Prism → Image Service 盘点](research/prism-to-image-service-inventory-2026-07-12.md)。
