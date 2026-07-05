# 凌镜 LingMirror — 项目评估报告

> 评估日期：2026-07-05
> 评估范围：`execution-gate-parallel` 分支全量代码审查
> 版本：v0.3.0.0

---

## 一、现有能力

### 架构完备

- **平台内核 4 原语**全部实现并集成：EventBus（15+ 订阅，glob 匹配）、Command Dispatcher（5+ 处理器）、Scheduler（5min-6hr 定时）、ToolBridge（插件化工具执行）
- **Agent 执行链路完整**：Orchestrator.Run() → 信任分查 → forbidden 阻断 → action_policy 自动审批 → approval 人工 → command 执行 → EventBus 发布 → 下游链路
- **Agent 注册表** 15 个 Agent，明确 Squad、Autonomy 分层、RiskFloor 基线
- **MOACoordinator** 多 Agent 并行决策 + 冲突检测 + 风险聚合 + 审批门
- **PipelineOrchestrator** 商品生命周期步进引擎（StartPipeline → step by step → complete）
- **审计 / RBAC / JWT** 三层安全基础设施就位
- **MutationGuard** + 幂等性 idempotency_key 保护 EventBus 写入
- **信任分 + 自主度升级** 闭环（trustscore.Recalculate → Upgrader.UpgradeEligible）
- **代谢系统 M1** 自我清理评分（impact/ref/freshness/semantic 四维）
- **熵系统 entropy** SPC 控制 + 健康评分
- **领域模块 63 个**，分层清晰（商品中台 / 供应链 / 订单履约 / 财务 / AgentOS / 集成 / 其他）
- **前端 88 个页面**，Ant Design 6 + Zustand + CRUD 通用组件
- **测试辅助** `dbtest.NewDB`（内存 SQLite，支持 t.Parallel）

### 设计清晰

- 模块统一模式（routes.go / handler.go / service.go / model.go）
- 标准响应信封（response.Success / Error / Paginated）
- 前端同类别名 `@` → `src/`，通用 CrudListPage 组件
- 治理文档完备（Constitution / Kernel Contracts / Owner-First Protocol / Agent Dev Protocol）
- `ponytail` 标记清晰标注了所有临时实现，易于识别技术债务

---

## 二、当前缺失与风险

### P0 — 构建与合并

| 问题 | 文件 | 影响 |
|------|------|------|
| 合并冲突（UU） | `routes.go`, `router.go` | 编译失败 |
| `UserIDFromCtx` 重复声明 | `internal/common/types.go:145` | 编译失败 |
| 全量测试基线不存在 | 全局 | 无法判断当前是否回退 |
| 16 个文件修改 + 2 个删除 + 1 个新增未最终提交 | 全分支 | 分支未收尾 |

### P1 — 技术债务

| 问题 | 位置 | 说明 |
|------|------|------|
| Agent 输出是 stub | `orchestrator.go:172` | `synthesizeOutput()` 产生确定性假数据，未接真实 LLM |
| MoA 聚合是字符串拼接 | `moa.go:296` | `synthesize()` 标记了 `naive concatenation` |
| 决策缓存 TTL 硬编码 | `orchestrator.go:70` | `newDecisionCache(5 * time.Minute)` |
| 信任分重算阻塞式 goroutine | `orchestrator.go:447` | 重算遍历全表，大表会饥饿 |
| 部分 handler 不强制 `decided_by` | `decision/handler.go`, `metabolism/service.go` | 审批人没有绑定 JWT（有的有，有的没有） |
| SQL 查询用 `INTERVAL '1 day'` 硬写 PostgreSQL | `metabolism/service.go:670` | 测试用 SQLite 会挂 |

### P2 — 能力缺口

| 缺口 | 影响 |
|------|------|
| 平台集成只有 ozon + shopee | 无 Lazada / Shopee / Amazon / Temu / TikTok Shop |
| 集成平台写回默认 dry-run | 无真正的生产级平台发布 |
| 88 个页面大量标记 `mock` | Owner 总控台、审批、数据概览、候选商品 4 个关键页面都是 mock |
| 履约中枢 marked `sandbox` | 物流/仓储/报关核心页面未完成 |
| `content_ai`, `scheduler`, A9, A11 标记 Phase 2 | 注册了但没有实现 |
| A8 的 `HandleSourcingTick` / `HandleSourcingRescan` 是空函数 | 选品定时扫描和补货重扫占位 |
| EventBus 订阅约 15 个，实际有多少真正跑通不确定 | 无压力测试 |
| 前端无 E2E 测试运行基线 | `npx playwright test` 从未记录结果 |

### P3 — 可观测性

| 缺口 | 影响 |
|------|------|
| Prometheus metrics 标记 opt-in | 默认关闭，无法监控生产 |
| Sentry 配置需要 DSN | 实际是否接入未知 |
| 没有运行时 Agent 决策延迟 / 成功率 / 错误率面板 | Owner 无法判断 AgentOS 是否正常工作 |
| 没有 LLM token 消耗面板 | 不知道成本 |

### P4 — 文档与测试

| 缺口 | 影响 |
|------|------|
| `go test ./...` 最新基线无记录 | 不能做 CI/CD |
| `go vet ./...` 最新基线无记录 | 可能有隐蔽问题 |
| `npm run build` 最新基线无记录 | 前端可能无法构建 |
| `npm test` 最新基线无记录 | 历史有 75/77 两种记录 |
| 测试覆盖率未知 | 63 个模块中很多 ⚠️ no covering tests found |

---

## 三、分支状态

```
execution-gate-parallel

  ↓ 合并 3 条并行工作流

9c2430e9 merge: P0-P1 AgentOS execution gate from 3 parallel workstreams
34ae8475 docs: clarify current verification facts
5fbeca88 P0: AgentOS execution gate — lifecycle + identity + catalog + audit

状态：
  14 staged，1 untracked（deliverables/），2 conflicts（UU）
  编译：❌ 失败（UserIDFromCtx 重复声明）
  测试：未知
```

---

## 四、建议优先级

1. **修合并冲突 + 编译** → 跑 `go test ./...` 建立基线
2. **`synthesizeOutput` 接真实 LLM** — Agent 产生假数据无法做 Owner 决策验证
3. **MoA `synthesize` 接 LLM 聚合** — 否则多 Agent 并行只是玩具
4. **补齐 P0 门禁检查** — for each action: RBAC → forbidden → policy → approval → audit → execute
5. **Owner 总控台从 mock 变为真实**

---

*本报告基于代码审查，不包含运行时行为验证（未启动 Docker / 数据库）。*
