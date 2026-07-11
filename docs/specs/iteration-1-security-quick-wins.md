# Spec: Iteration 1 — 安全速赢 + 数据地基

> 更新时间：2026-07-06
> 覆盖 Issue：[#280](https://github.com/lingmirror/multisell/issues/280)、[#281](https://github.com/lingmirror/multisell/issues/281)、[#130](https://github.com/lingmirror/multisell/issues/130)
> 系统层：Layer 1（Kernel）/ Layer 2（Domain）/ Layer 6（Docs）
> 优先级：P1（当前最高）——先止血再优化

## Objective

### 一句话

为 LingMirror 的所有环境（开发/测试/生产）建立三个最基本的安全和数据保护实践：消除硬编码凭证、防止数据丢失、消除关键业务计算的魔数。

### Issue 详情

| # | 标题 | 业务影响 | 当前状态 |
|---|------|----------|----------|
| 280 | JWT_SECRET 写死在 docker-compose.yml | 所有环境的 JWT 签名秘钥相同，泄漏一个即被全部破解 | 值 `${JWT_SECRET:-dev-secret-change-in-production}` — 生产仍用 dev-secret |
| 281 | 数据库无自动备份 | Docker volume 是唯一副本，容器删除或盘损 → 全量数据丢失 | 无备份脚本、无定时任务、无异地备份 |
| 130 | 利润计算硬编码 CNY→USD 汇率 7.2 | 偏差超 0.3 时利润偏差 4%+，ProfitWatch Agent 误判 | 代码已使用 exchangerate 模块（`getCNYRate()`），fallback 7.2 改为配置 |

### 验收标准

1. ✅ JWT_SECRET 在生产环境中通过环境变量设置，不依赖 docker-compose 默认值
2. ✅ 生产环境部署检查脚本验证 JWT_SECRET 不为默认值
3. ✅ pg_dump 定时备份脚本可用，本地备份 + 云存储同步
4. ✅ 备份恢复流程文档化（restore 步骤）
5. ✅ 利润汇率 fallback 值从 `config.yaml` 读取，可配置
6. ✅ `go test ./internal/domain/profit/...` 通过
7. ✅ `go vet ./...` 通过

## Tech Stack

- 后端：Go 1.25, GORM, PostgreSQL 15
- 配置：`backend-go/configs/config.yaml` + 环境变量覆盖（`viper`）
- 备份：`pg_dump` / cron / rclone 或类似云存储同步工具
- 部署：Docker Compose + environment 变量

## Commands

```bash
# 验证
cd backend-go && go test ./internal/domain/profit/...   # 利润模块测试
cd backend-go && go vet ./...                            # 静态分析

# 构建
cd backend-go && go build -o bin/server cmd/server/main.go

# 配置检查
grep -r 'dev-secret-change-in-production' .              # 确认无残留默认值

# 备份（部署后）
./scripts/backup_db.sh                                    # 手动备份
crontab -l                                                # 确认定时任务
```

## Project Structure

只改动少数现有文件，不新增目录：

| 文件 | 说明 | 涉及 Issue |
|------|------|------------|
| `docker-compose.yml` | JWT_SECRET 行加注释，说明必须在 .env 或环境变量中设置 | #280 |
| `docs/ops/OWNER_AND_AI_DEPLOYMENT_RUNBOOK.md` | 部署检查清单包含 JWT_SECRET 验证 | #280 |
| `scripts/backup_db.sh` | **新建** — 数据库备份脚本 | #281 |
| `docs/ops/DISASTER_RECOVERY.md` | 补充"数据库恢复"步骤 | #281 |
| `backend-go/configs/config.yaml` | 新增 `profit.default_cny_rate: 7.2` | #130 |
| `backend-go/internal/domain/profit/service.go` | `getCNYRate()` 的 fallback 从硬编码改为配置 | #130 |
| `backend-go/internal/domain/profit/service_test.go` | 新增 fallback + exchangerate 集成测试 | #130 |

## Code Style

遵循现有 Go 模块模式：

```go
// getCNYRate returns the current CNY->USD exchange rate.
// Falls back to config.DefaultCNYRate when exchangerate is unavailable.
func (s *Service) getCNYRate() float64 {
    if s.rateSvc == nil {
        return s.cfg.DefaultCNYRate  // config, not hardcoded 7.2
    }
    rate, err := s.rateSvc.GetLatest("CNY", "USD")
    if err != nil {
        s.logger.Warn("failed to get CNY/USD rate, using config default",
            zap.Float64("fallback", s.cfg.DefaultCNYRate), zap.Error(err))
        return s.cfg.DefaultCNYRate
    }
    return rate.Rate
}
```

命名规则：
- 配置键：`lower_snake_case`，与环境变量名对齐（如 `default_cny_rate`）
- Go：驼峰 `getCNYRate` / `DefaultCNYRate`
- Shell 脚本：`snake_case.sh`

## Testing Strategy

| 测试类型 | 覆盖 | 框架 |
|----------|------|------|
| 单元测试 | `getCNYRate()` fallback 逻辑——rateSvc=nil、GetLatest error、Rate<=0 | Go testing + testify |
| 集成测试 | `getCNYRate()` 通过 exchangerate 模块取真实汇率 | Go testing |
| 部署验证 | shell 检查 JWT_SECRET 非默认值 | `grep` / `test` |
| 破坏性测试 | 停止 exchangerate 服务后 profit 计算是否正常降级 | 手动 |

**必须做到的：**
- 新增测试覆盖 3 种 fallback 场景（nil svc / error / invalid rate）
- `go test ./internal/domain/profit/...` 全绿
- `go vet ./...` 无输出

**不需要做的：**
- 对备份脚本写单元测试（手工验证即可）
- 对 JWT_SECRET 检查写自动化测试

## Boundaries

### Always Do（必须遵守）

- 物理边界：JWT_SECRET **必须** 通过 `.env` 文件或环境变量注入，绝不在 docker-compose.yml 中写明文
- 数据安全：备份文件权限 `600`，禁止存入公开位置
- 可逆操作：利润汇率 fallback 值必须可配置，不改代码即可调整
- 提交前运行 `go test` 和 `go vet`
- 新代码附加对应单元测试

### Ask First（先问再做）

- 改动 `router.go` 初始化逻辑
- 引入新的外部依赖（如 rclone / aws-cli）
- 新增数据库迁移（本迭代不需要）
- 改动 docker-compose.yml 中其他服务的配置

### Never Do（禁止）

- 在版本控制中提交生产 JWT_SECRET 或其他凭证
- 在测试代码中硬 7.2 —— 用 `DefaultCNYRate` 配置值
- 删除或修改现有 profi `getCNYRate()` 的 exchangerate 集成逻辑（只用改 fallback）

## Success Criteria

```
□ JWT_SECRET 在 docker-compose.yml 中只有默认值提醒行，生产通过 .env 设置
□ 部署后运行 grep 'dev-secret-change-in-production' → 不应返回 0
□ backup_db.sh 单次运行可导出完整 pg_dump
□ DISASTER_RECOVERY.md 包含 "从备份恢复" 章节
□ profit.Service 构造时读取 config.DefaultCNYRate
□ getCNYRate 3 种 fallback 场景各有测试
□ go test ./internal/domain/profit/... 通过
□ go vet ./... 通过
□ #280 / #281 / #130 全部关闭
```

## Open Questions

1. 云存储备份用哪种方案？rclone（通用，支持 S3/GCS/OSS）还是直接用云厂商 CLI？
   建议：rclone，不绑定特定云厂商，Owner 确认配置后即可用。
2. profit 模块目前 `NewService` 需要 `*config.Config`，还是只加一个 `DefaultCNYRate float64` 字段？
   建议：加 `DefaultCNYRate` 字段，减少配置耦合。
