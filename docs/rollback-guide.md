# LingMirror 凌镜 — 回滚说明

更新时间：2026-07-01

## 通用回滚原则

1. **代码回滚**: `git revert <commit>` 而非 `git reset`，保留回滚历史。
2. **数据库回滚**: 先执行对应 `down.sql`，再部署旧代码。
3. **数据安全**: 回滚不删除业务数据，仅恢复 schema/代码。
4. **顺序**: 先回滚数据库 → 再回滚代码 → 验证服务健康。

## 按阶段回滚

### P8 文档收口

文件: `docs/PROJECT_STATUS.md`, `docs/FUNCTION_INVENTORY.md`, `docs/AGENT_CAPABILITIES.md`, `docs/known-risks.md`

- PR #211
- 回滚: `git revert e188bf66`
- 数据库: 无变更

### P7 治理强化

文件: `migrations/000047_forbidden_actions`, `internal/domain/actionpolicy/forbidden.go`, `internal/domain/actionpolicy/service.go`, `internal/ai/orchestrator.go`

- PR #212
- 代码回滚: `git revert b225ed44`
- 数据库回滚: `migrations/000047_forbidden_actions.down.sql`
- 影响: ForbiddenAction 检查停止，High-Risk 门禁移除

### P6 试运行准备

文件: `internal/ai/orchestrator.go`, `internal/agentos/service.go`, `internal/agentos/handler.go`, `docs/trial-run-guide.md`

- PR #198
- 代码回滚: `git revert 9f34f934`
- 数据库: 无变更
- 影响: Agent 失败不再记录 trace（回退到 silent fail），Failures API 移除

### P5 多业务场景扩大

文件: `internal/ai/orchestrator.go`, `internal/domain/approval/service.go`

- PR #174
- 代码回滚: `git revert cc61cc48`
- 数据库: 无变更
- 影响: UnifiedAction ↔ Approval 自动联动停止

### P4 AgentOS 驾驶舱

文件: `internal/agentos/service.go`, `internal/agentos/handler.go`, `internal/domain/owner/service.go`, `internal/domain/owner/handler.go`

- PR #158
- 代码回滚: `git revert c52f4cd0`
- 数据库: 无变更
- 影响: WorkItemDetail/AgentTimeline/AgentActivity/PipelineChain API 移除

## 全量回滚（极少数情况）

如果需要回滚到 P4 之前的稳定状态：

```bash
# 1. 回滚数据库（按时间倒序）
cd backend-go
psql -d multisell -f migrations/000047_forbidden_actions.down.sql

# 2. 回滚代码
git revert --no-commit c52f4cd0  # P4
git revert --no-commit cc61cc48  # P5
git revert --no-commit 9f34f934  # P6
git revert --no-commit b225ed44  # P7
git revert --no-commit e188bf66  # P8
git commit -m "revert: P4-P8 rollback"

# 3. 重启服务
systemctl restart lingmirror-backend  # 生产环境
# 或
docker compose restart backend  # 本地
```

## 验证回滚成功

```bash
# 服务健康
curl http://localhost:8080/api/health

# 核心功能
go test ./internal/domain/listingtask/...
go test ./internal/domain/approval/...
go test ./internal/domain/loop/...
```

## 注意事项

- P4-P6 有前端文件变更，回滚后需重新构建前端：`cd frontend-next && npm run build`
- P5 改动在 orchestrator.go，可能与其他 Agent 变更冲突，需手动解决合并冲突
- 回滚后推荐的恢复点验证：Owner 总控台可访问、AgentOS 驾驶舱可查看、审批流程可操作
