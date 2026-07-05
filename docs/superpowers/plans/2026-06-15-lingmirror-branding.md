> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# LingMirror Branding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebrand the user-facing product from `MultiSell` to `凌镜 LingMirror`, while keeping technical package/repository identifiers stable for this phase.

**Architecture:** This is a low-risk branding pass. Update visible UI labels, browser title, backend app metadata, README, and product vision docs. Do not rename the repository, Docker services, package names, database names, or Python module paths in this phase.

**Tech Stack:** Vue 3, Vite, FastAPI, Markdown documentation, existing Docker Compose runtime.

---

## Brand Decision

Final product brand:

```text
凌镜 LingMirror
```

Product positioning:

```text
跨境电商 AgentOS
```

Full one-line description:

```text
凌镜 LingMirror 是面向中小跨境电商团队的 AI Agent 协作运营平台，从商品上架前利润决策切入，逐步扩展到商品资料治理、物流测算、平台费用核算、上架执行、订单履约和经营分析。
```

Naming rule:

- User-facing product name: `凌镜 LingMirror`
- Short UI brand name: `凌镜`
- Product category: `跨境电商 AgentOS`
- Legacy technical project name: `MultiSell`

Use this phrase when both names are needed:

```text
凌镜 LingMirror（原 MultiSell）
```

---

## Scope

### In Scope

Update visible branding in:

- Browser title.
- Login page.
- Main layout sidebar/header brand.
- Backend API metadata.
- Health test if it asserts old app name.
- README.
- Product vision document.
- Project status document.
- Governance document.

### Out Of Scope

Do not rename these in this phase:

- Repository folder `/Users/lc/multisell`.
- Docker Compose service names.
- Docker image names.
- `frontend/package.json` package name.
- `package-lock.json` package name.
- Database names.
- Admin email `admin@multisell.com`.
- Historical plan filenames.
- Historical implementation docs unless they are current overview docs.
- Python module names.
- API paths.

Reason:

Changing technical identifiers creates deployment, test, and import risk. This phase is a brand presentation layer change only.

---

## Current Brand Occurrences

Important occurrences found:

Frontend:
- `frontend/index.html`
- `frontend/src/components/Layout.vue`
- `frontend/src/views/Login.vue`

Backend:
- `backend/app/config.py`
- `backend/app/main.py`
- `backend/tests/test_health.py`

Docs:
- `README.md`
- `docs/PRODUCT_VISION_AND_MVP.md`
- `docs/PROJECT_STATUS.md`
- `docs/PROJECT_GOVERNANCE_AND_AGENT_WORKFLOW.md`
- `docs/ROADMAP.md`
- `docs/DEVELOPMENT_GUIDE.md`
- `docs/PERMISSIONS_AND_AUDIT.md`

Do not mass-replace every historical occurrence. Historical plan docs can keep `MultiSell` unless they are current entry points.

---

## Target Text

Use these exact replacements:

```text
MultiSell - AI跨境电商商品中台
```

becomes:

```text
凌镜 LingMirror - 跨境电商 AgentOS
```

```text
MultiSell
```

in visible UI brand positions becomes:

```text
凌镜
```

```text
AI跨境电商商品中台
```

in login subtitle becomes:

```text
跨境电商 AgentOS
```

Backend app description should become:

```text
凌镜 LingMirror - 面向中小跨境电商团队的 AI Agent 协作运营平台
```

---

## Implementation Tasks

### Task 1: Frontend Visible Branding

**Files:**
- Modify: `frontend/index.html`
- Modify: `frontend/src/components/Layout.vue`
- Modify: `frontend/src/views/Login.vue`

- [ ] **Step 1: Update browser title**

In `frontend/index.html`, change:

```html
<title>MultiSell - AI跨境电商商品中台</title>
```

to:

```html
<title>凌镜 LingMirror - 跨境电商 AgentOS</title>
```

- [ ] **Step 2: Update layout brand**

In `frontend/src/components/Layout.vue`, change visible sidebar/header brand:

```vue
🌐 MultiSell
```

to:

```vue
🪞 凌镜
```

If the project wants ASCII-only source later, use:

```vue
凌镜
```

But the current codebase already uses Chinese UI strings and emoji, so the mirror emoji is acceptable.

- [ ] **Step 3: Update login page brand**

In `frontend/src/views/Login.vue`, change:

```vue
🌐 MultiSell
```

to:

```vue
🪞 凌镜 LingMirror
```

Change subtitle:

```vue
AI跨境电商商品中台
```

to:

```vue
跨境电商 AgentOS
```

- [ ] **Step 4: Build frontend**

Run:

```bash
cd frontend && npm run build
```

Expected:

```text
built
```

---

### Task 2: Backend App Metadata

**Files:**
- Modify: `backend/app/config.py`
- Modify: `backend/app/main.py`
- Modify: `backend/tests/test_health.py`

- [ ] **Step 1: Update app metadata**

In `backend/app/config.py`, change:

```python
APP_NAME: str = "MultiSell - AI跨境电商商品中台"
APP_DESCRIPTION: str = "MultiSell - AI原生跨境电商商品中台"
```

to:

```python
APP_NAME: str = "凌镜 LingMirror - 跨境电商 AgentOS"
APP_DESCRIPTION: str = "凌镜 LingMirror - 面向中小跨境电商团队的 AI Agent 协作运营平台"
```

- [ ] **Step 2: Update module docstring only if desired**

In `backend/app/main.py`, current docstring is:

```python
"""MultiSell — FastAPI 入口"""
```

Change to:

```python
"""凌镜 LingMirror — FastAPI 入口"""
```

- [ ] **Step 3: Update health test**

Open `backend/tests/test_health.py`.

If it asserts:

```python
assert "MultiSell" in data["service"]
```

replace with:

```python
assert "凌镜 LingMirror" in data["service"]
```

If the health response returns config `APP_NAME`, this should pass.

- [ ] **Step 4: Run backend health test**

Run:

```bash
cd backend && python3 -m pytest tests/test_health.py -q
```

Expected:

```text
passed
```

If this fails because the sandbox cannot connect to local PostgreSQL, rerun with local permission or report the environment blocker.

---

### Task 3: Current Documentation Branding

**Files:**
- Modify: `README.md`
- Modify: `docs/PRODUCT_VISION_AND_MVP.md`
- Modify: `docs/PROJECT_STATUS.md`
- Modify: `docs/PROJECT_GOVERNANCE_AND_AGENT_WORKFLOW.md`
- Optional modify: `docs/ROADMAP.md`
- Optional modify: `docs/DEVELOPMENT_GUIDE.md`

- [ ] **Step 1: Update README title**

Change:

```markdown
# 🌐 MultiSell — AI跨境电商商品中台
```

to:

```markdown
# 🪞 凌镜 LingMirror — 跨境电商 AgentOS
```

Change the first description from:

```markdown
基于 **Python FastAPI + Vue 3 + PostgreSQL** 的跨境电商商品中台。
```

to:

```markdown
基于 **Python FastAPI + Vue 3 + PostgreSQL** 的 AI Agent 协作跨境电商运营平台。
```

- [ ] **Step 2: Add legacy name note**

Near the README top, add:

```markdown
> 技术项目名暂保留 `MultiSell`；对外产品品牌为 `凌镜 LingMirror`。
```

- [ ] **Step 3: Update product vision doc**

In `docs/PRODUCT_VISION_AND_MVP.md`, change the title:

```markdown
# MultiSell 最终产品形态与第一可用版本定义
```

to:

```markdown
# 凌镜 LingMirror 最终产品形态与第一可用版本定义
```

At the top, add:

```markdown
产品品牌：凌镜 LingMirror
技术项目名：MultiSell
```

Replace current final-positioning sentence with:

```text
凌镜 LingMirror 是一个面向中小跨境电商团队的 AI Agent 协作运营平台。
```

- [ ] **Step 4: Update current status and governance docs**

In `docs/PROJECT_STATUS.md`, title becomes:

```markdown
# 凌镜 LingMirror Project Status
```

In `docs/PROJECT_GOVERNANCE_AND_AGENT_WORKFLOW.md`, title becomes:

```markdown
# 凌镜 LingMirror 项目收口与 Agent 协作规范
```

Add one note in both:

```markdown
说明：`MultiSell` 是历史技术项目名；当前产品品牌为 `凌镜 LingMirror`。
```

- [ ] **Step 5: Do not rewrite historical plans**

Do not edit files under `docs/superpowers/plans/` unless a current plan explicitly needs brand text. Those are historical execution records.

- [ ] **Step 6: Check docs**

Run:

```bash
rg -n "AI跨境电商商品中台|AI原生跨境电商商品中台" README.md docs frontend backend
```

Expected:

```text
No user-facing current docs or UI still use old product subtitle.
Historical plan docs may still contain old text and do not need changes.
```

---

### Task 4: Verification

**Files:**
- No new files required.

- [ ] **Step 1: Run frontend build**

Run:

```bash
cd frontend && npm run build
```

Expected:

```text
built
```

- [ ] **Step 2: Run backend health test**

Run:

```bash
cd backend && python3 -m pytest tests/test_health.py -q
```

Expected:

```text
passed
```

- [ ] **Step 3: Run whitespace check**

Run:

```bash
git diff --check
```

Expected:

```text
no output
```

- [ ] **Step 4: Manual browser check**

If frontend runs at `http://localhost:3000`:

1. Open `/login`.
2. Confirm login page shows `凌镜 LingMirror`.
3. Login.
4. Confirm sidebar/header shows `凌镜`.
5. Confirm browser title is `凌镜 LingMirror - 跨境电商 AgentOS`.

If Docker serves stale frontend assets, rebuild or sync `frontend/dist` into the running frontend container, then refresh browser with hard reload.

---

## Handoff Prompt For Another Agent

```text
请阅读并严格执行这个规划文档：
/Users/lc/multisell/docs/superpowers/plans/2026-06-15-lingmirror-branding.md

目标是把用户可见品牌从 MultiSell 调整为“凌镜 LingMirror”，定位为“跨境电商 AgentOS”。这次只做品牌展示层改名，不要重命名仓库、Docker 服务、包名、数据库名、API 路径或历史计划文档。

执行要求：
1. 只改规划允许的前端、后端 metadata、当前入口文档。
2. 不要大范围全局替换 historical docs。
3. 不要回滚其他人的改动。
4. 完成后运行：
   - cd frontend && npm run build
   - cd backend && python3 -m pytest tests/test_health.py -q
   - git diff --check
5. 汇报改动文件、验证结果、是否更新了运行中的前端容器。
```

---

## Future Phase: Technical Rename

Technical rename can be considered later, but not now.

Future technical rename would include:

- Repository name.
- Docker service/image names.
- Package names.
- Database names.
- Default admin email domain.
- GitHub URLs.
- CI/CD identifiers.

Do that only after product direction is stable and the current MVP is working.
