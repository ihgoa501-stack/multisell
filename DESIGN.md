# Design System — 凌镜 LingMirror

> **Current product boundary — 2026-07-11:** 本设计系统只服务 Owner 自用内部经营工作台。下文中 B2B SaaS、merchants、operations teams 和 SaaS dashboard 是历史设计语境，不得驱动当前开发。当前界面优先级依次为：实验现金风险、证据完整性、停止条件、订单/退货状态和最终净利润；与首轮 Ozon 实验无关的视觉重构冻结。

## Product Context
- **What this is:** Owner 本人用于运行真实跨境商品实验的内部经营工作台。
- **Who it's for:** 单一 Owner。
- **Space/industry:** Internal commerce operations and product experimentation.
- **Project type:** Internal decision and evidence workspace.

## Aesthetic Direction
- **Direction:** AI Agent Desktop — conversation-first interface where AI agents are the primary interaction mode. Traditional SaaS functions (CRUD, batch ops, reports) are tools that expand from the right panel.
- **Decoration level:** Minimal — typography and spacing carry the visual weight. Color is reserved for AI interaction areas only.
- **Mood:** Professional, AI-native, responsive. The user should feel like they're working WITH an intelligent agent, not clicking through a legacy ERP. Inspired by Linear, Claude Desktop, and Notion AI.
- **Reference sites:** Linear (linear.app), Claude Desktop, ChatGPT Desktop, Notion AI.

## Typography
- **Display/Hero:** Cabinet Grotesk (800/700/600 weight) — geometric, modern, strong presence. Used for product name, page headings, modal titles. Adds structural clarity to data-heavy screens.
- **Body:** DM Sans (400/500/600 weight) — clean, excellent readability at all sizes, shiping-friendly curves. Used for conversation text, labels, table cells, tooltips.
- **UI/Labels:** Same as body (DM Sans).
- **Data/Tables:** DM Sans with `font-variant-numeric: tabular-nums` — no extra font load, perfect column alignment.
- **Code:** JetBrains Mono — tool call logs, configuration fields, raw data display.
- **Loading:** Google Fonts via `<link>` tags (preconnect to fonts.googleapis.com + fonts.gstatic.com).
- **Scale:**
  - Hero: clamp(2rem, 5vw, 3.8rem), 800 weight, -0.03em letter-spacing
  - H1: clamp(1.5rem, 3.5vw, 2.5rem), 700 weight, -0.02em letter-spacing
  - H2: 1.2-1.8rem, 600 weight
  - Body: 0.82-1rem, 400 weight
  - Small: 0.72-0.78rem, 400 weight
  - Label: 0.68rem, 600 weight, 0.06em letter-spacing, uppercase

## Color
- **Approach:** Restrained — one accent (Indigo), one AI-highlight (Cyan), and a warm-black neutral palette. Color is meaningful, not decorative.
- **Primary:** `#6366F1` (Indigo 500) — primary actions, brand identity, navigation indicators.
- **Primary Light:** `#818CF8` (Indigo 400) — hover states, subtle indigo accents.
- **AI Highlight:** `#22D3EE` (Cyan 400) — AI activity indicators, streaming animations, agent presence, proactive suggestions.
- **AI Highlight Deep:** `#06B6D4` (Cyan 500) — hover/active states for AI elements.
- **Neutrals (Dark mode):**
  - Background: `#0B0B0E` (deepest level)
  - Surface 1: `#121216` (cards, panels)
  - Surface 2: `#1A1A20` (hover, inputs)
  - Surface 3: `#22222A` (scrollbars, subtle separators)
  - Border 1: `#282830` (panel borders)
  - Border 2: `#32323C` (input borders, stronger dividers)
  - Text primary: `#E8E8ED`
  - Text secondary: `#A0A0AB`
  - Text muted: `#6B6B78`
  - Text faint: `#4A4A55`
- **Neutrals (Light mode):**
  - Background: `#F0EFEA`
  - Surface 1: `#FAFAF7`
  - Surface 2: `#F0EFEA`
  - Surface 3: `#E6E5DF`
  - Border 1: `#D8D7D0`
  - Border 2: `#CAC9C2`
  - Text primary: `#1A1A18`
  - Text secondary: `#555450`
  - Text muted: `#8B8A84`
  - Text faint: `#B0AFAA`
- **Semantic:**
  - Success: `#34D399`
  - Warning: `#FBBF24`
  - Error: `#F87171`
  - Info: `#6366F1`
  - AI: `#22D3EE`
- **Dark mode strategy:** Dark-first design. Dark surfaces use warm blacks (not #000) to avoid eye strain. Light mode surfaces desaturate neutrals ~10-15% for a paper-like feel.

## Spacing
- **Base unit:** 8px (comfortable density — data pages need breathing room).
- **Scale:**
  - 2xs: 2px (icon internal padding)
  - xs: 4px (toolbar gaps, badge padding)
  - sm: 8px (message gaps, input padding)
  - md: 12px (panel padding, card padding)
  - lg: 16px (section padding, conversation gaps)
  - xl: 24px (page margins, large sections)
  - 2xl: 32px (hero spacing)
  - 3xl: 48px (major section breaks)

## Layout
- **Approach:** Hybrid — three-column resizable layout.
  - Left: Agent Hub (220px) — agent profile, running tasks, quick tools, session list.
  - Center: Conversation (flex, resizable) — AI chat at core, tool call cards for visibility.
  - Right: Tool Panel (flex, resizable) — full SaaS CRUD interface when expanded.
- **Panel ratios:** Three modes with CSS transition (0.4s cubic-bezier):
  - AI priority: 70% conversation / 30% tools
  - Balanced: 50% / 50%
  - Tool priority: 30% / 70%
- **Max content width:** Unconstrained (full-window app).
- **Border radius:**
  - xs: 2px (subtle dividers)
  - sm: 4px (inputs, small elements)
  - md: 6px (buttons, tool cards)
  - lg: 8px (agent avatar, cards)
  - xl: 10px (input bar, modal containers)
  - full: 9999px (badges, pills)

## Motion
- **Approach:** Intentional — every transition has purpose. AI interactions are animated to feel alive.
- **Panel resize:** 0.4s cubic-bezier(0.22, 1, 0.36, 1) — elastic feel for split-panel transitions.
- **Hover/active:** 80ms ease — instant micro-feedback on interactive elements.
- **State transitions:** 200ms ease — button states, panel expand/collapse.
- **AI streaming:** Custom stepping animation (1.4s cycle, staggered dots) — communicates "thinking" without visual noise.
- **Easing:**
  - Enter: ease-out (natural appearance)
  - Exit: ease-in (controlled disappearance)
  - Move/Resize: cubic-bezier(0.22, 1, 0.36, 1)

## Key Components

### Agent Hub (Left Sidebar)
- Agent avatar with online status indicator
- Active tasks list with progress
- Quick tool shortcuts (click → expand tool panel)
- Recent conversation sessions

### Conversation Panel (Center)
- Message bubbles with agent/user avatars
- Tool call cards — show which functions the agent invoked, with arguments and results
- Streaming indicator dots for in-progress AI responses
- Persistent input bar at bottom

### Tool Panel (Right)
- Expands from right with smooth slide animation
- Shows full SaaS interface: data table, filters, search, batch actions
- Context banner shows what the AI is currently filtering/doing
- "AI confirm" action button for AI-assisted batch operations

### Mode Indicator
- Three-way toggle at top: AI priority / Balanced / Tool priority
- Click panel header areas as shortcut to switch

## Decisions Log
| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-06-25 | AI Agent Desktop as primary interface paradigm | Category differentiation + user's "AI智能感" memorable-thing goal. Cross-border e-commerce SaaS all look the same — LingMirror should feel fundamentally different. |
| 2026-06-25 | Resizable split-panel (AI/Tool) | Users want to talk to AI AND directly manipulate data. Fixed-width panels serve neither well. Three ratio modes cover the full workflow spectrum. |
| 2026-06-25 | Dark-first design | AI tools universally default to dark (Linear, Claude, Cursor). Dark backgrounds make streaming AI content visually punchier. Warm blacks avoid eye strain. |
| 2026-06-25 | Cabinet Grotesk + DM Sans | Geometric sans for headings gives structural clarity to data-heavy pages. DM Sans has excellent readability at small sizes + built-in tabular numbers. |
| 2026-06-25 | Indigo + Cyan palette | Indigo avoids the traditional SaaS blue/green while staying professional. Cyan is fresh and reads as "AI" without falling into purple-gradient slop. |
