# CI / E2E 验证记录

> 验证日期：2026-07-12
> 范围：GitHub Actions CI 触发、现有质量门禁、E2E 发布依赖

## 结论

- `external_observed`：GitHub Actions 的 `CI` 工作流已由真实 Pull Request 触发。检查到的运行包括 `29175219545` 和 `29175188618`。
- `external_observed`：上述运行中的后端测试、前端构建/测试/lint 和治理检查存在成功记录。
- `external_observed`：E2E 当前不稳定。一条运行在测试数据初始化时因 `listing_task_product_id_fkey` 失败；另一条运行启动全栈后有 14 条通过、9 条失败。
- `implemented`：镜像发布现在强制依赖治理检查、Go lint、Go 测试、前端检查和 E2E 全部成功。
- `implemented`：E2E 失败时上传 Playwright 报告、测试产物和后端日志，并设置 20 分钟超时。
- `implemented`：CI 支持手动触发，同一分支的新运行会取消旧运行。
- `unknown`：本次配置修改尚未在 GitHub runner 上完成一次新的全绿运行；提交并触发后才能提升为 `external_observed`。

## 证据限制

CI 和 E2E 通过只能证明测试环境中的工程行为，不证明真实市场、真实成交、生产部署或最终净利润成立。现有部分浏览器测试覆盖历史 mock 页面和已冻结流程，不能替代当前“候选市场 → Owner 批准 → 经营实验 → 最终利润”的主线验收。

## 下一验证动作

1. 将本次修改提交到 Pull Request，观察 GitHub runner 的完整结果。
2. 从失败产物定位并修复剩余 E2E，而不是用重试或跳过隐藏失败。
3. 为当前经营主线补充少量真实全栈 E2E，再将其作为生产发布的明确验收项。
