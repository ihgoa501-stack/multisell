> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# LingMirror AgentOS MVP 收尾 — Agent 详情页

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 添加 Agent 详情页，打通从 Squad → Agent 卡 → Agent 详情 → WorkItem/操作日志的完整浏览链路。

**Architecture:** 后端新增 `GET /api/agentos/agents/{agent_id}/detail` 聚合接口，按 agent_id 聚合其 WorkItem、操作日志、升级记录。前端新增详情页路由和视图，Squad 页的 Agent 卡片可点击跳转。

**Tech Stack:** FastAPI, async SQLAlchemy, Vue 3, Naive UI, TypeScript.

**Prerequisites:** Phase 1-4 已完成，37 个 agentos 测试通过。

---

## Scope Guard

- 不新增数据库表
- 不修改现有 Agent 模型
- 不引入 LLM 调用
- 不开发审批流引擎

## File Structure

### Create
- `frontend/src/views/agentos/AgentDetail.vue` — Agent 详情页面

### Modify
- `backend/app/agentos/router.py` — 新增 `/agentos/agents/{agent_id}/detail` endpoint
- `backend/app/agentos/service.py` — 新增 `get_agent_detail()` 聚合方法
- `backend/app/agentos/schemas.py` — 新增 `AgentDetailResponse` 响应模型
- `frontend/src/router/modules/agentos.ts` — 新增 agent-detail 路由
- `frontend/src/api/modules/agentos.ts` — 新增 `getAgentOSAgentDetail()` 方法
- `frontend/src/views/agentos/Squads.vue` — AgentStatusCard 添加 click 事件跳转到详情页
- `backend/tests/test_agentos_phase1.py` — 新增 Agent detail API 测试

---

## Task 1: Backend — Agent Detail API

**Files:**
- Modify: `backend/app/agentos/schemas.py`
- Modify: `backend/app/agentos/service.py`
- Modify: `backend/app/agentos/router.py`

- [ ] **Step 1: Add AgentDetailResponse schema**

在 `backend/app/agentos/schemas.py` 的 `AutonomyCandidateVO` 之后添加：

```python
class AgentDetailResponse(BaseModel):
    """Agent 详情响应"""
    agent: AgentOSAgent
    squad_name: str = ""
    current_work_items: list[AgentOSWorkItem] = Field(default_factory=list)
    recent_operations: list[AgentOSOperationLogVO] = Field(default_factory=list)
    decision_count_7d: int = 0
    adoption_rate_7d: float = 0.0
```

- [ ] **Step 2: Add get_agent_detail service method**

在 `backend/app/agentos/service.py` 中 `get_upgrade_candidates` 之后添加：

```python
    @staticmethod
    async def get_agent_detail(
        db: AsyncSession,
        user_id: int,
        agent_id: str,
    ) -> dict[str, Any]:
        """获取单个 Agent 的详情聚合数据"""
        meta = AGENT_META.get(agent_id, {})
        squad_id = AGENT_TO_SQUAD.get(agent_id, "governance")

        # 构建 AgentOSAgent
        agent = AgentOSAgent(
            id=agent_id,
            name=meta.get("name", agent_id),
            role=meta.get("role", ""),
            squad_id=squad_id,
            status="active",
            autonomy_level=AutonomyLevel.SUGGESTION,
        )

        # 查询该 Agent 的 WorkItem
        all_items = await AgentOSService._collect_all_work_items(db)
        agent_items = [i for i in all_items if i.agent_id == agent_id]

        # 查询操作日志
        operations = []
        try:
            ops_result = await AgentOSService.get_operations(
                db, limit=20, offset=0,
            )
            op_records = ops_result.get("records", [])
            from app.agentos.schemas import AgentOSOperationLogVO
            for op in op_records:
                if isinstance(op, AgentOSOperationLogVO) and agent_id in op.item_id:
                    operations.append(op)
        except Exception:
            pass

        # 决策统计
        decision_count = 0
        adoption_rate = 0.0
        try:
            from app.agent.models import AgentDecision
            seven_days_ago = datetime.now(timezone.utc) - timedelta(days=7)
            total = await db.scalar(
                select(sa_func.count()).select_from(AgentDecision)
                .where(
                    AgentDecision.agent_id == agent_id,
                    AgentDecision.created_at >= seven_days_ago,
                )
            ) or 0
            accepted = await db.scalar(
                select(sa_func.count()).select_from(AgentDecision)
                .where(
                    AgentDecision.agent_id == agent_id,
                    AgentDecision.user_action == "accepted",
                    AgentDecision.created_at >= seven_days_ago,
                )
            ) or 0
            decision_count = int(total)
            adoption_rate = round(accepted / total, 3) if total else 0
        except Exception:
            pass

        return {
            "agent": agent,
            "squad_name": SQUAD_TO_NAME.get(squad_id, squad_id),
            "current_work_items": agent_items[:20],
            "recent_operations": operations[:20],
            "decision_count_7d": decision_count,
            "adoption_rate_7d": adoption_rate,
        }
```

- [ ] **Step 3: Add router endpoint**

在 `backend/app/agentos/router.py` 的 Phase 3 endpoints 之后添加：

```python
# ── Phase 4 Finale: Agent Detail ──────────────────────────


@router.get("/agentos/agents/{agent_id}/detail", summary="Agent 详情")
async def get_agent_detail(
    agent_id: str,
    db=Depends(get_db),
    current_user: User = Depends(require_permission("agentos:view")),
):
    """返回单个 Agent 的详情、WorkItem 列表和操作记录"""
    data = await AgentOSService.get_agent_detail(db, current_user.id, agent_id)
    return Result.ok(data)
```

- [ ] **Step 4: Run tests**

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_phase1.py -q
```

Expected: 37 passed.

- [ ] **Step 5: Commit**

```bash
git add backend/app/agentos/schemas.py backend/app/agentos/service.py backend/app/agentos/router.py
git commit -m "feat(agentos): add agent detail API"
```

---

## Task 2: Backend Tests

**Files:**
- Modify: `backend/tests/test_agentos_phase1.py`

- [ ] **Step 1: Write tests**

在 `# ─── Phase 4: 自治等级真实写入测试` 段落后添加：

```python
# ─── Phase 4 Finale: Agent Detail 测试 ──────────────────


@pytest.mark.usefixtures("prepare_db")
class TestAgentDetailAPI:
    """Agent 详情接口测试"""

    async def test_agent_detail_returns_basic_info(self, async_client):
        """返回 Agent 基本信息"""
        resp = await async_client.get("/api/agentos/agents/A5/detail")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        detail = data["data"]
        assert detail["agent"]["id"] == "A5"
        assert detail["agent"]["name"] == "库存管家"
        assert detail["squad_name"] == "履约小队"
        assert "current_work_items" in detail
        assert "recent_operations" in detail

    async def test_agent_detail_unknown_agent(self, async_client):
        """不存在的 Agent 返回基础结构"""
        resp = await async_client.get("/api/agentos/agents/XX99/detail")
        assert resp.status_code == 200
        data = resp.json()
        assert data["code"] == 200
        detail = data["data"]
        assert detail["agent"]["id"] == "XX99"

    async def test_agent_detail_has_stats(self, async_client):
        """Agent 详情返回决策统计"""
        resp = await async_client.get("/api/agentos/agents/A5/detail")
        assert resp.status_code == 200
        detail = resp.json()["data"]
        assert "decision_count_7d" in detail
        assert "adoption_rate_7d" in detail
```

- [ ] **Step 2: Run tests**

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_phase1.py -q
```

Expected: 40 passed (37 + 3 new).

- [ ] **Step 3: Commit**

```bash
git add backend/tests/test_agentos_phase1.py
git commit -m "feat(agentos): add agent detail API tests"
```

---

## Task 3: Frontend — API + Route

**Files:**
- Modify: `frontend/src/api/modules/agentos.ts`
- Modify: `frontend/src/router/modules/agentos.ts`

- [ ] **Step 1: Add API method**

在 `frontend/src/api/modules/agentos.ts` 的 Phase 3 段落后添加：

```typescript
// ── Phase 4 Finale: Agent Detail ─────────────────────────

export interface AgentDetailResponse {
  agent: AgentOSAgent
  squad_name: string
  current_work_items: AgentOSWorkItem[]
  recent_operations: AgentOSOperationLog[]
  decision_count_7d: number
  adoption_rate_7d: number
}

export function getAgentOSAgentDetail(agentId: string) {
  return http.get(`/agentos/agents/${agentId}/detail`)
}
```

更新 `agentosApi` 对象追加 `getAgentOSAgentDetail`。

- [ ] **Step 2: Add route**

在 `frontend/src/router/modules/agentos.ts` 的 `children` 数组中添加：

```typescript
{
  path: 'agents/:agentId',
  name: 'AgentOSAgentDetail',
  component: () => import('@/views/agentos/AgentDetail.vue'),
  meta: { title: 'Agent 详情', menu: false, perm: 'agentos:view' },
},
```

- [ ] **Step 3: Build**

```bash
cd frontend && npm run build
```

Expected: build passes (the `AgentDetail.vue` doesn't exist yet so the lazy import will resolve at runtime; the build may warn about the missing chunk but won't fail). If it does fail, create a stub component first.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/api/modules/agentos.ts frontend/src/router/modules/agentos.ts
git commit -m "feat(agentos): add agent detail API and route"
```

---

## Task 4: Frontend — Agent Detail Page

**Files:**
- Create: `frontend/src/views/agentos/AgentDetail.vue`

- [ ] **Step 1: Create AgentDetail.vue**

创建 `frontend/src/views/agentos/AgentDetail.vue`：

```vue
<template>
  <div>
    <!-- 加载状态 -->
    <n-spin v-if="loading" :show="true" style="margin-top: 40px;">
      <div style="text-align: center; padding: 40px;">加载中...</div>
    </n-spin>

    <!-- 错误状态 -->
    <n-result v-else-if="error" status="error" title="加载失败" :description="error">
      <template #footer>
        <n-button @click="fetchDetail">重试</n-button>
      </template>
    </n-result>

    <template v-else-if="detail">
      <!-- 页面标题 + Agent 信息 -->
      <n-page-header :subtitle="detail.squad_name" @back="router.back()">
        <template #title>
          <n-space align="center" size="small">
            <n-avatar :size="32" round>{{ detail.agent.name.charAt(0) }}</n-avatar>
            <span>{{ detail.agent.name }}</span>
          </n-space>
        </template>
        <template #extra>
          <n-space>
            <AutonomyBadge :level="detail.agent.autonomy_level" />
            <n-tag v-if="detail.agent.status === 'active'" type="success" size="small" :bordered="false">工作中</n-tag>
            <n-tag v-else type="default" size="small" :bordered="false">{{ detail.agent.status }}</n-tag>
          </n-space>
        </template>
      </n-page-header>

      <!-- 统计行 -->
      <n-grid :cols="4" :x-gap="12" style="margin-top: 12px;">
        <n-grid-item>
          <n-card size="small">
            <div class="stat-value">{{ detail.agent.role || '-' }}</div>
            <div class="stat-label">角色</div>
          </n-card>
        </n-grid-item>
        <n-grid-item>
          <n-card size="small">
            <div class="stat-value">{{ detail.current_work_items.length }}</div>
            <div class="stat-label">当前任务</div>
          </n-card>
        </n-grid-item>
        <n-grid-item>
          <n-card size="small">
            <div class="stat-value">{{ detail.decision_count_7d }}</div>
            <div class="stat-label">7天决策</div>
          </n-card>
        </n-grid-item>
        <n-grid-item>
          <n-card size="small">
            <div class="stat-value">{{ (detail.adoption_rate_7d * 100).toFixed(0) }}%</div>
            <div class="stat-label">7天采纳率</div>
          </n-card>
        </n-grid-item>
      </n-grid>

      <!-- 两列布局 -->
      <n-grid :cols="2" :x-gap="12" style="margin-top: 12px;">
        <!-- 左：当前任务 -->
        <n-grid-item>
          <n-card title="当前任务" size="small">
            <template v-if="detail.current_work_items.length === 0">
              <n-empty description="暂无任务" style="padding: 20px 0;" />
            </template>
            <WorkItemCard
              v-for="item in detail.current_work_items"
              :key="item.id"
              :item="item"
              @status-updated="fetchDetail"
            />
          </n-card>
        </n-grid-item>

        <!-- 右：近期操作记录 -->
        <n-grid-item>
          <n-card title="近期操作记录" size="small">
            <template v-if="detail.recent_operations.length === 0">
              <n-empty description="暂无操作记录" style="padding: 20px 0;" />
            </template>
            <n-list v-else>
              <n-list-item v-for="op in detail.recent_operations" :key="op.id">
                <n-space align="center" justify="space-between">
                  <n-space align="center" size="small">
                    <n-tag :type="opActionType(op.action)" size="tiny" :bordered="false">
                      {{ opActionLabel(op.action) }}
                    </n-tag>
                    <span style="font-size: 13px;">{{ op.comment || op.new_status || '-' }}</span>
                  </n-space>
                  <span style="color: #999; font-size: 11px;">{{ formatTime(op.created_at) }}</span>
                </n-space>
              </n-list-item>
            </n-list>
          </n-card>
        </n-grid-item>
      </n-grid>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { getAgentOSAgentDetail } from '@/api/modules/agentos'
import type { AgentDetailResponse } from '@/api/modules/agentos'
import AutonomyBadge from '@/components/agentos/AutonomyBadge.vue'
import WorkItemCard from '@/components/agentos/WorkItemCard.vue'

const route = useRoute()
const router = useRouter()
const message = useMessage()

const loading = ref(false)
const error = ref<string | null>(null)
const detail = ref<AgentDetailResponse | null>(null)

async function fetchDetail() {
  const agentId = route.params.agentId as string
  if (!agentId) return

  loading.value = true
  error.value = null
  try {
    const res: any = await getAgentOSAgentDetail(agentId)
    detail.value = res?.data || res
  } catch (e: any) {
    error.value = e?.response?.data?.message || e?.message || '加载失败'
    message.error('加载 Agent 详情失败')
  } finally {
    loading.value = false
  }
}

function opActionType(action: string): 'success' | 'info' | 'warning' | 'error' {
  const map: Record<string, 'success' | 'info' | 'warning' | 'error'> = {
    approve: 'success',
    reject: 'warning',
    autonomy_upgrade: 'success',
    autonomy_downgrade: 'warning',
    status_update: 'info',
  }
  return map[action] || 'info'
}

function opActionLabel(action: string): string {
  const map: Record<string, string> = {
    approve: '审批通过',
    reject: '拒绝',
    autonomy_upgrade: '升级',
    autonomy_downgrade: '降级',
    status_update: '状态变更',
  }
  return map[action] || action
}

function formatTime(val: string | null): string {
  if (!val) return ''
  return new Date(val).toLocaleString('zh-CN')
}

onMounted(fetchDetail)
</script>

<style scoped>
.stat-value {
  font-size: 22px;
  font-weight: 700;
  line-height: 1.2;
}
.stat-label {
  font-size: 12px;
  color: #888;
  margin-top: 2px;
}
</style>
```

- [ ] **Step 2: Build**

```bash
cd frontend && npm run build
```

Expected: build passes.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/agentos/AgentDetail.vue
git commit -m "feat(agentos): add agent detail page"
```

---

## Task 5: Wire Agent Card Click on Squads Page

**Files:**
- Modify: `frontend/src/views/agentos/Squads.vue`

- [ ] **Step 1: Add click handler to AgentStatusCard**

在 `frontend/src/views/agentos/Squads.vue` 中，给 AgentStatusCard 添加 `click` 事件跳转到 `/agentos/agents/${agent.id}`。

找到 `v-for="agent in squad.agents"` 的 `AgentStatusCard` 调用，添加 `@click`：

```vue
<AgentStatusCard
  v-for="agent in squad.agents"
  :key="agent.id"
  :agent="agent"
  style="cursor: pointer;"
  @click="router.push(`/agentos/agents/${agent.id}`)"
/>
```

- [ ] **Step 2: Build**

```bash
cd frontend && npm run build
```

Expected: build passes.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/agentos/Squads.vue
git commit -m "feat(agentos): wire agent card click to detail page"
```

---

## Task 6: Full Verification

**Files:** none

- [ ] **Step 1: Backend tests**

```bash
cd backend && PYTHONPATH="$PWD" .venv/bin/python -m pytest tests/test_agentos_phase1.py -q
```

Expected: 40 passed (37 existing + 3 new agent detail tests).

- [ ] **Step 2: Frontend build**

```bash
cd frontend && npm run build
```

Expected: build passes.

- [ ] **Step 3: Git status**

```bash
git status --short
```

Expected: only AgentOS files changed.

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "feat(agentos): complete MVP finale with agent detail page"
```

---

## Self-Review

- **Spec coverage:** Agent detail API (Task 1), tests (Task 2), frontend API + route (Task 3), detail page (Task 4), squad card wiring (Task 5), verification (Task 6).
- **Placeholder scan:** No TBD/TODO/placeholder patterns.
- **Type consistency:**
  - Backend: `AgentDetailResponse`, `get_agent_detail()`
  - Frontend: `AgentDetailResponse`, `getAgentOSAgentDetail()`
  - Route: `/agentos/agents/:agentId`
  - API: `GET /api/agentos/agents/{agent_id}/detail`
