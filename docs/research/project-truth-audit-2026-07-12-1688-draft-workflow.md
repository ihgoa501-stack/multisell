# 1688 受控草稿链工程事实审计

> 审计日期：2026-07-12
> 范围：本工作树中的 1688 采集到内部待上架草稿变更
> 状态：工程变更快照；不替代同日产品方向审计

## 裁决

- `implemented`：新增受控 URL fetch、证据 capture、Owner review、convert/update-to-draft、快照读取和草稿读取 API；旧自由写入路由不再注册。
- `implemented`：来源快照包含 URL、观察时间、采集驱动、解析版本、原始 JSON 与 SHA-256，数据库触发器拒绝更新或删除。
- `implemented`：转草稿要求已批准市场、通过 opportunity gate 的 active 实验、同一 Owner、结构化供应商/合规证据、SKU 三段映射、图片权利和实际处理记录、10 类费用与独立汇率/贡献利润、渠道类目规则、配送模板与本地化内容。
- `implemented`：新增 offer ID 规范化、内容指纹疑似同款队列、价格/MOQ/供应商变化事件、采集失败记录和 Owner 重复项裁决。
- `implemented`：新增确定性规则验证，覆盖语言脚本、卖点、关键词、单位、禁用词、渠道必填属性、变体、图片规格、配送模板、成本换算、图片质量与 SKU 映射；unknown/mock/inferred 不能通过。
- `implemented`：图片真实执行解码、中心裁切、缩放和白底合成，保存处理前后 SHA-256 与输出字节；不声明能够自动判断或删除水印、中文和品牌。
- `implemented`：内部草稿具有 editing/pending_approval/approved_draft 生命周期，并复用 canonical approval 与审计；批准仍不发布。
- `implemented`：转草稿在单事务内创建内部产品、SKU、媒体、成本、追溯和 `product_listing(status=draft)`；没有平台适配器调用。
- `implemented`：另有独立发布申请、第二次 Owner 审批、显式执行与异常对账账本；请求和响应带 SHA-256，终态由 PostgreSQL 触发器保护。平台响应只裁决为 `submitted`，不得写成真实上线。
- `implemented`：受控 URL 采集通过 JWT Owner 定向浏览器扩展，保存实际结构化响应与脱敏 DOM 结构；采集失败、未配置、串单、过期观察或解析无效均留存 CaptureAttempt。
- `automated_verified`：2026-07-12 聚焦 Go 测试与 Next production build 通过。自动测试使用本地/模拟业务数据，不提高外部事实等级。
- `external_observed`：`unknown`。尚未采集并人工核验一个真实 1688 商品。
- `manually_verified`：`unknown`。尚未在真实登录、真实页面、真实图片权利、真实费用与已批准渠道条件下完成浏览器全链验收。
- `automated_verified`：在临时隔离的 PostgreSQL 16.14 空库中，全量迁移从 `000001` 升级到 `000091` 成功；确认快照、疑似同款、图片处理、采集失败和发布账本及 4 个 sourcing 触发器存在；`000084`–`000091` 回滚后重新升级成功。
- `automated_verified`：sourcing1688 竞态测试、ToolBridge/驱动测试、HTTP router 测试和聚焦 vet 通过；Chrome 扩展 TypeScript 构建及实际加载的 JavaScript 语法检查通过。
- `automated_verified`：全量 Go 回归通过（118 packages，3035 tests）；`go build ./...` 与 `go vet ./...` 通过。配套修正了 Owner 只读视图调用方字段兼容，以及测试服务器缺失 refresh session 表导致的登录 401。
- 生产 PostgreSQL 状态：`unknown`。临时数据库验证不代表生产已部署。

## 不得声称

当前不得声称 1688 生产采集可用、供应商可靠、图片有权使用、目标渠道接受草稿、平台已创建/上线商品、最终成本准确或商品值得经营。上述事实必须由一个真实 Owner 批准商品实验及相应外部凭证核验。
