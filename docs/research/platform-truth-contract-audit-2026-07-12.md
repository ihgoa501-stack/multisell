# 平台真相与领域合同交付审计

> 日期：2026-07-12
> 对应路线：ADR-001 第 1 单元“平台真相与领域合同”
> 范围：代码与文档交付，不包含外部经营验证

## 本次建立的合同

- 后端 `internal/domain/platformtruth/` 成为平台方向、六类真实性等级、事实系统/决策系统边界和核心模块处置的只读合同。
- `GET /api/v1/platform-truth` 位于 JWT 保护路由内，不写数据库、不触发外部平台、不改变价格、库存、订单、资金、权限或审批。
- Owner 可从 `/platform-truth` 读取合同；原菜单“经营实验”更正为“事实核验案卷”，技术路径 `/experiments` 暂不迁移。
- 核心模块明确标记为 `reuse / adapt / migrate / rebuild / freeze`，并声明 `implemented / planned / superseded` 等证据等级与小Q支持状态。
- 当前 `internal/domain/` 全部 70 个领域目录已逐项登记；测试将目录清单与合同反向比对，禁止新领域静默绕过分类。第 70 个 `productimage` 在新增后由该测试拦截，审计后归入第 3 单元共享支撑。
- 合同同时定义经营证据真实性等级、工程声明等级、五类活动系统加冻结范围、对象身份规则和来源规则。

## 证据等级

| 声明 | 等级 |
|---|---|
| 合同、只读 API、Owner 页面和菜单入口存在 | `implemented` |
| 合同结构、真实性等级顺序、70 个领域目录覆盖、闭合枚举和唯一性 | `automated_verified`（以后端聚焦测试结果为准） |
| 页面在生产构建中可编译 | `automated_verified`（以前端构建结果为准） |
| 全部现有领域目录已完成处置分类 | `implemented / automated_verified`；分类是开发治理合同，不代表对应改造已经完成 |
| `adapt / migrate / rebuild / delete` 条目已经全部执行 | `planned`；应在后续对应纵向单元实施，不能用本合同冒充完成 |
| 真实市场、发布、订单、利润或现金已发生 | `unknown`；本次未连接任何外部经营来源 |

## 边界

该合同是治理与解释入口，不是经营结果仪表盘，也不自动升级数据库记录的真实性。任何模块存在、测试通过、页面可见或对象关联，都不能证明真实交易、生产可用、最终利润或因果反馈已经成立。

`delete` 处置仅表示建议退出目标架构；它不授权删除代码、数据库记录或历史审计。实际删除仍需单独影响审计、回滚方案和 Owner 明确批准。

## 完成要求核验

| ADR-001 第 1 单元要求 | 权威证据 | 裁决 |
|---|---|---|
| 统一经营证据真实性等级 | `CurrentContract().TruthLevels` 固定六类并由测试校验顺序 | `automated_verified` |
| 统一工程声明等级 | `ClaimLevels` 固定 policy 至 superseded 八类 | `implemented / automated_verified` |
| 明确对象身份与 Owner 边界 | `ObjectIdentityRules`、全部领域 `owner_scope=single_owner` | `implemented / automated_verified` |
| 明确来源规则 | `SourceRules` 规定来源、观察时间、不可变快照、actual 升级和财务对账 | `implemented / automated_verified` |
| 分开事实系统与决策系统 | `SystemBoundaries` 分开 fact / decision，并另列 collaboration / kernel / support / frozen | `implemented / automated_verified` |
| 现有模块全部分类 | 70 个领域目录与 70 条登记反向比对；处置枚举闭合 | `automated_verified` |
| `experiment` 纠偏 | 分类为 fact + migrate；Owner 菜单显示“事实核验案卷” | `implemented`；技术包名和路径暂保留 |
| Owner 可理解体验 | `/platform-truth` 展示方向、统计、边界、等级、规则、筛选、处置和 unknown | `implemented / automated_verified` |
| 安全边界 | JWT 集成测试；API 仅 GET；无数据库或外部副作用；delete 不授权执行 | `automated_verified` |
| 文档同步 | AGENTS、CLAUDE、INDEX、模块目录和本审计已同步；文档漂移检查通过 | `automated_verified` |

## 本次验证结果

- Go build、vet 通过；后端全量 119 个包、3145 项测试通过。
- 前端 26 个测试文件、135 项测试通过；Next.js 生产构建成功并生成 `/platform-truth`；本目标四个前端文件定向 ESLint 通过。
- 全仓 ESLint 未全绿：`src/app/(main)/settings/plugin/page.tsx` 存在一项并行工作错误，另有 10 项既有警告。本目标未修改该页面，也不以局部通过掩盖全仓失败。
- `scripts/check_doc_drift.sh` 检查 231 个引用，缺失 0。
