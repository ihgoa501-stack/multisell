> ⚠️ 历史计划文档。引用已删除的旧栈，仅供参考。

# LingMirror Icon Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the LingMirror branding pass by integrating the generated LingMirror icon into favicon, login page, and main layout brand positions.

**Architecture:** This is a frontend-only static asset and UI integration task. Copy the generated icon into `frontend/public/brand/`, reference it through stable public URLs, and keep the original generated image untouched. Do not modify backend, database, routing, business logic, package names, or Docker service names.

**Tech Stack:** Vue 3, Vite public assets, Naive UI, existing Docker/Nginx frontend runtime.

---

## Scope

### In Scope

- Add LingMirror icon asset under `frontend/public/brand/`.
- Replace Vite favicon with LingMirror icon.
- Show LingMirror icon on login page.
- Show LingMirror icon in the main layout header.
- Keep text brand:

```text
凌镜 LingMirror
```

on login page, and:

```text
凌镜
```

in the compact header.

- Run frontend build.
- Confirm running frontend container is updated if using `localhost:3000`.

### Out Of Scope

Do not change:

- Backend files.
- API metadata.
- Database.
- Docker service names.
- Package names.
- Product business logic.
- Historical docs.
- Generated original image path.

---

## Source Icon

Generated image source:

```text
/Users/lc/.codex/generated_images/019ec007-548e-7061-8498-96b71cbabc7d/ig_0627180394b74a43016a2f67f0c5a4819a8e1747f950ba494e.png
```

Important:

- Copy this file into the project.
- Do not move or delete the original generated image.

Target project asset:

```text
frontend/public/brand/lingmirror-icon.png
```

Public URL after Vite build:

```text
/brand/lingmirror-icon.png
```

---

## Current State

Current frontend state:

- `frontend/public/` does not exist yet.
- `frontend/index.html` still uses:

```html
<link rel="icon" type="image/svg+xml" href="/vite.svg" />
```

- `frontend/src/views/Login.vue` uses text/emoji:

```vue
🪞 凌镜 LingMirror
```

- `frontend/src/components/Layout.vue` uses text/emoji:

```vue
🪞 凌镜
```

The goal is to use the generated icon image instead of the mirror emoji.

---

## Implementation Tasks

### Task 1: Add Brand Asset

**Files:**
- Create: `frontend/public/brand/lingmirror-icon.png`

- [ ] **Step 1: Create target directory**

Run:

```bash
mkdir -p frontend/public/brand
```

- [ ] **Step 2: Copy generated icon**

Run:

```bash
cp /Users/lc/.codex/generated_images/019ec007-548e-7061-8498-96b71cbabc7d/ig_0627180394b74a43016a2f67f0c5a4819a8e1747f950ba494e.png frontend/public/brand/lingmirror-icon.png
```

- [ ] **Step 3: Verify asset exists**

Run:

```bash
ls -lh frontend/public/brand/lingmirror-icon.png
```

Expected:

```text
frontend/public/brand/lingmirror-icon.png exists and is non-empty
```

---

### Task 2: Replace Favicon

**Files:**
- Modify: `frontend/index.html`

- [ ] **Step 1: Replace Vite favicon link**

Change:

```html
<link rel="icon" type="image/svg+xml" href="/vite.svg" />
```

to:

```html
<link rel="icon" type="image/png" href="/brand/lingmirror-icon.png" />
```

- [ ] **Step 2: Keep title unchanged**

Do not change:

```html
<title>凌镜 LingMirror - 跨境电商 AgentOS</title>
```

---

### Task 3: Login Page Icon

**Files:**
- Modify: `frontend/src/views/Login.vue`

- [ ] **Step 1: Replace emoji heading with icon + text block**

Current block:

```vue
<h2 style="margin: 0; font-size: 24px;">🪞 凌镜 LingMirror</h2>
<p style="margin: 4px 0 0; color: #888; font-size: 14px;">跨境电商 AgentOS</p>
```

Replace with:

```vue
<div style="display: flex; justify-content: center; align-items: center; gap: 10px;">
  <img
    src="/brand/lingmirror-icon.png"
    alt="凌镜 LingMirror"
    style="width: 36px; height: 36px; border-radius: 8px;"
  />
  <h2 style="margin: 0; font-size: 24px;">凌镜 LingMirror</h2>
</div>
<p style="margin: 8px 0 0; color: #888; font-size: 14px;">跨境电商 AgentOS</p>
```

Requirement:

- The icon must not stretch.
- The `alt` text must be present.
- Keep the login card layout stable.

---

### Task 4: Main Layout Header Icon

**Files:**
- Modify: `frontend/src/components/Layout.vue`

- [ ] **Step 1: Replace emoji brand heading**

Current heading:

```vue
<h2 style="margin: 0; color: #fff; font-size: 16px; letter-spacing: 1px;">🪞 凌镜</h2>
```

Replace with:

```vue
<div style="display: flex; align-items: center; gap: 8px;">
  <img
    src="/brand/lingmirror-icon.png"
    alt="凌镜"
    style="width: 24px; height: 24px; border-radius: 6px;"
  />
  <h2 style="margin: 0; color: #fff; font-size: 16px; letter-spacing: 1px;">凌镜</h2>
</div>
```

- [ ] **Step 2: Preserve search box layout**

Make sure the surrounding structure still has:

```vue
<div style="display: flex; align-items: center; gap: 12px;">
```

containing:

- the brand block
- `<n-auto-complete ... />`

Do not accidentally nest the search input inside the brand block.

---

### Task 5: Verification

**Files:**
- No additional files.

- [ ] **Step 1: Run frontend build**

Run:

```bash
cd frontend && npm run build
```

Expected:

```text
built
```

- [ ] **Step 2: Check built HTML favicon path**

Run:

```bash
rg -n "lingmirror-icon|vite.svg" frontend/dist frontend/index.html
```

Expected:

```text
frontend/index.html and frontend/dist/index.html reference /brand/lingmirror-icon.png
No frontend/index.html favicon reference to /vite.svg
```

Note:

- Other unrelated `vite.svg` files may not exist. If only no output for `vite.svg`, that is fine.

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

If frontend is served at `http://localhost:3000`:

1. Ensure latest `frontend/dist` is deployed to the running frontend container.
2. Open `/login`.
3. Confirm favicon is LingMirror icon.
4. Confirm login card shows LingMirror icon next to `凌镜 LingMirror`.
5. Login.
6. Confirm main layout header shows LingMirror icon next to `凌镜`.

If Docker serves stale assets, update container after build:

```bash
docker cp frontend/dist/. multisell-frontend-1:/usr/share/nginx/html/
```

Then hard refresh browser.

---

## Handoff Prompt For Another Agent

```text
请阅读并严格执行这个规划文档：
/Users/lc/multisell/docs/superpowers/plans/2026-06-15-lingmirror-icon-integration.md

目标是完成凌镜品牌收口：把已生成的凌镜图标接入 favicon、登录页和主布局顶部品牌。只做前端静态资源和 UI 展示，不要改后端、数据库、业务逻辑、Docker 服务名或包名。

图标源文件：
/Users/lc/.codex/generated_images/019ec007-548e-7061-8498-96b71cbabc7d/ig_0627180394b74a43016a2f67f0c5a4819a8e1747f950ba494e.png

目标文件：
frontend/public/brand/lingmirror-icon.png

完成后运行：
- cd frontend && npm run build
- rg -n "lingmirror-icon|vite.svg" frontend/dist frontend/index.html
- git diff --check

如果使用 localhost:3000 验收，还要确认最新 dist 已同步到 frontend 容器。

汇报改动文件、验证结果、是否更新了运行中的前端容器。
```

---

## Next Step After This

After icon integration is verified, run a separate review for non-brand changes that were mixed into the branding pass, especially:

- `backend/app/config.py`
- `backend/app/main.py`
- Docker/runtime behavior

Do not combine that review with icon integration.
