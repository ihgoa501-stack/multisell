# 知名 AI Agent / Coding Agent 技术栈调研

> 调研日期：2026-07-11
> 目的：为 LingMirror 桌面端与本地 Agent 内核选型提供事实依据。
> 证据原则：只采用厂商官方文档、官方博客、官方代码仓库以及官方发行物中的包证据；不把社区猜测当作事实。

## 结论先行

市场上已经形成三条主要路线：

1. **Electron / VS Code 系**：VS Code + Copilot、Cursor，以及高度疑似沿用同类架构的 Windsurf。优势是编辑器、终端、扩展、跨平台和更新生态成熟；代价是内存、多进程与 IPC 治理复杂。
2. **薄桌面壳 + 独立本地 Agent 运行时**：Codex Desktop 最有代表性。桌面客户端不是 Agent 内核；它通过 Codex CLI 的 `app-server` API 驱动 Rust 本地运行时。这个结构最接近 LingMirror 的需求。
3. **Rust 原生全栈桌面**：Zed 使用 Rust + 自研 GPUI，追求性能和原生体验，但工程门槛、组件生态和产品迭代成本显著更高。

Claude Desktop、ChatGPT Desktop 与 Windsurf 的桌面壳缺乏足够的一手架构说明，因此不纳入技术栈样本，避免把产品能力或界面相似性误当成实现证据。

对 LingMirror 最重要的行业共识不是“大家都用 Electron”，而是：

> UI 壳可以替换，但文件、Shell、PTY、审批、沙箱、任务状态与 Agent 协议应当形成独立、可测试、可被多个客户端调用的本地运行时。

## 证据等级

- **已确认**：官方源码、官方技术文档或厂商明确表述。
- **强推断**：官方发行物依赖、官方仓库问题中的可复现包结构等直接技术证据，但厂商未正式声明架构。
- **未知**：公开的一手资料不足，不作猜测。

## 横向对比

| 产品 | 桌面壳 / UI | 本地 Agent 层 | 终端 / 进程 | 更新与分发 | 置信度 |
|---|---|---|---|---|---|
| OpenAI Codex Desktop | Electron（发行物包证据）；具体 UI 框架未公开 | Rust Codex CLI `app-server`，JSON-RPC，默认 stdio | CLI 负责文件、命令、审批、系统沙箱；发行物含 PTY 相关原生模块的证据 | macOS/Windows 官方客户端；具体更新实现未公开 | 壳：强推断；运行时：已确认 |
| Claude Code | 终端 UI；官方变更日志确认使用 React Compiler，但完整实现闭源 | 独立本地 CLI，支持工具、子 Agent、hooks、MCP、权限规则 | 直接执行 Bash/Shell；有后台任务、权限和 allow/deny 规则 | 原生安装器、Homebrew、WinGet；npm 已弃用 | 已确认（具体打包器未知） |
| Cursor | VS Code fork + Electron，多进程；主体为 TypeScript/Web UI | Agent 功能分布在 renderer、utility/extension processes 与云端服务 | 继承 VS Code 终端/扩展体系；Agent 可调用终端；本地/云 Agent 可切换 | 自有下载/CDN与更新域名；跨平台桌面 | 已确认 |
| VS Code + Copilot | Electron + TypeScript，renderer / main / utility / extension host 多进程 | Copilot 作为编辑器内 Agent 能力，依托 VS Code 扩展与工作台 API | 集成终端使用 xterm.js；Node/PTY 进程层与扩展宿主隔离 | 官方跨平台安装器与内置更新体系 | 已确认 |
| Zed | Rust + 自研 GPUI，GPU 加速原生 UI；非 Electron | Agent Panel / assistant 与编辑器核心同属 Rust 应用，可接模型与外部 Agent 协议 | Rust 原生终端与任务体系 | macOS/Linux/Windows 官方包及包管理器 | 已确认 |

## 逐项研究

### 1. OpenAI Codex Desktop

**已确认**

- OpenAI 将其定义为多 Agent 的桌面“command center”，支持并行和长任务。[Introducing the Codex app](https://openai.com/index/introducing-the-codex-app/)
- 官方 Codex 仓库说明 `codex app-server` 是富客户端的接口，使用双向 JSON-RPC；默认传输是 stdio JSONL，另有实验性 WebSocket 和 Unix Socket。[codex app-server README](https://github.com/openai/codex/blob/main/codex-rs/app-server/README.md)
- OpenAI 仓库维护者明确说明 Desktop 和 IDE extension 都建立在 Codex CLI 的 app-server API 之上，Desktop 本身不开源。[官方 GitHub 答复](https://github.com/openai/codex/discussions/16538)
- 本地执行采用与 Codex CLI 相同的系统级沙箱；默认限制可写目录，并对网络等高权限操作请求许可。[OpenAI 产品公告](https://openai.com/index/introducing-the-codex-app/)
- Codex CLI 主体是 Rust；其开源仓库提供执行、审批、sandbox 与 app-server 协议实现。[openai/codex](https://github.com/openai/codex)

**强推断**

- 官方发行物的包结构和官方 issue 中的可复现崩溃信息显示存在 Electron main process、V8、Node 原生模块、`better-sqlite3` 与 `node-pty`。因此 Desktop 壳为 Electron 的判断很强，但 OpenAI 没有在产品架构文档中正式声明。[官方仓库 issue 中的发行物证据](https://github.com/openai/codex/issues/22004)
- 由此推断其大致形态是：Electron UI → 子进程启动 Rust Codex runtime → stdio JSON-RPC/JSONL → UI 展示任务、diff、终端和审批。

**未知**

- Renderer 使用 React、原生 DOM 或其他框架，官方未说明。
- macOS updater 的具体库、签名流水线和本地数据库 schema 未公开。

**启示**：这是最有价值的参照。OpenAI 没把高权限执行写进 renderer，而是复用一个跨 CLI、IDE、Desktop 的 Agent runtime 与稳定协议。

### 2. Claude Code

**已确认**

- Claude Code 是“运行在终端中的 agentic coding tool”，可读代码库、执行任务与 Git 工作流。[官方仓库](https://github.com/anthropics/claude-code)
- 官方支持原生安装脚本、Homebrew Cask 与 WinGet，npm 安装已经弃用。[官方安装说明](https://github.com/anthropics/claude-code)
- 官方变更日志暴露了后台 Bash、MCP stdio 子进程、并行 subagent、permissions、plugins 与 worktree 等运行时能力；也明确提到用 React Compiler 改进 UI 渲染。[官方 CHANGELOG](https://github.com/anthropics/claude-code/blob/main/CHANGELOG.md)

**未知**

- 当前原生可执行文件内部究竟是 Bun、打包 JavaScript、Rust/Go wrapper 还是混合实现，官方没有公开完整产品源码，不能从仓库语言占比推断。
- TUI 的具体组件库与自动更新内部机制未公开。

**启示**：CLI 本身可以是完整 Agent 产品，而 GUI 只是另一种客户端。权限规则、后台进程、MCP 和恢复能力应在运行时层，不应依赖 Electron。

### 3. Cursor

**已确认**

- Cursor 官方明确说产品最初 fork VS Code，以便突破 extension UI 能力边界。[Cursor 3 官方博客](https://cursor.com/blog/cursor-3)
- 官方稳定性文章明确写明桌面端基于 VS Code 和 Electron，并描述 renderer、utility processes、extensions、storage 与 agent functionality 的多进程结构。[Keeping the Cursor app stable](https://cursor.com/blog/app-stability)
- Cursor 使用进程隔离降低 extension crash 影响；进程间存在 IPC 和持久化消息通道。
- 自有安全页列出更新/扩展下载域名，并说明定期合并 VS Code 上游。[Cursor Security](https://www.cursor.com/security)
- 早期 shadow workspace 使用隐藏 Electron window，为 Agent 提供独立 LSP 上下文；官方也讨论了复制工作区、磁盘隔离与并行 Agent 的权衡。[Shadow Workspaces](https://www.cursor.com/blog/shadow-workspace)

**启示**：VS Code fork 能最快获得 IDE、LSP、terminal 和 extension 生态，但长期要承担上游合并、V8 OOM、巨型 IPC payload 和进程生命周期治理成本。

### 4. VS Code + GitHub Copilot

**已确认**

- VS Code 官方仓库以 TypeScript 为主，并明确使用 Electron。[microsoft/vscode](https://github.com/microsoft/vscode)
- 源码组织明确区分 browser、node、electron-browser、electron-utility 和 electron-main；extension host 与 workbench API 也有明确边界。[VS Code Source Code Organization](https://github.com/microsoft/vscode/wiki/source-code-organization)
- 官方路线图明确持续维护 xterm.js integrated terminal，并讨论 terminal session persistence。[VS Code Roadmap](https://github.com/microsoft/vscode/wiki/Roadmap)

**启示**：Electron 并不等于“renderer 直接拿 Node 权限”。成熟结构依靠 renderer sandbox、main/utility/extension host 隔离和窄 IPC 接口。

### 5. Zed

**已确认**

- Zed 源码约 98% Rust，支持 macOS、Linux、Windows，采用自研 GPUI。[zed-industries/zed](https://github.com/zed-industries/zed)
- GPUI 是 Rust 的 GPU 加速混合 immediate/retained UI 框架；macOS 使用 Metal，框架集成了窗口、平台服务与 async executor。[GPUI README](https://github.com/zed-industries/zed/blob/main/crates/gpui/README.md)

**启示**：Rust 原生 UI 能获得优秀性能和单语言内核，但 GPUI 仍处于 pre-1.0，组件、招聘、无障碍和跨平台打磨成本都高于 Electron/React。它适合“性能是产品壁垒”的编辑器，不是 LingMirror 首版最经济的路线。

## 对 LingMirror 的技术判断

### 建议保留的选择

**Electron + React/Vite 作为驾驶舱，独立本地 runtime 作为发动机**仍然是合理首选，但 runtime 应当从第一天就做成明确协议边界，而不是 Electron main 中的一组零散 IPC handler。

推荐结构：

```text
React Renderer（无 Node 权限）
        │ typed IPC：窗口、通知、更新等少量桌面能力
Electron Main
        │ 启动/监督
Local Agent Runtime
        │ JSON-RPC over stdio（首选）或本机 socket
        ├─ workspace / filesystem / patch
        ├─ PTY / process / Git
        ├─ agent loop / tool registry
        ├─ approval / policy / audit
        └─ SQLite / OS secure storage
```

### 对此前“Go daemon”方案的修正

- **方向正确**：独立 daemon/runtime 与 Codex 的成熟架构一致。
- **协议应先于语言**：先定义 command、event、approval、task recovery 和 streaming schema，再决定 Go/Rust。建议兼容 JSON-RPC 2.0 思路，默认 stdio，后续再增加 local socket/WebSocket。
- **Go 可继续采用**：LingMirror 已有 Go 后端和治理契约，团队复用价值高；没有必要为了追随 Codex/Zed 改成 Rust。
- **PTY 要优先验证**：Go runtime 需要验证 Windows ConPTY、macOS/Linux PTY、resize、UTF-8、shell profile、进程树终止和会话恢复。这里是桌面 Agent 最容易低估的工程区。
- **不要让 renderer 直接调用 daemon 的开放端口**：由 Electron main 生成短期能力令牌并监督进程，或者直接走 stdio / domain socket，可减少本机端口劫持和权限混乱。

### 不建议现在做的事

- 不建议 fork VS Code：LingMirror 的核心是电商 AgentOS，不是代码编辑器；承担 VS Code 上游合并成本没有商业回报。
- 不建议首版采用 Rust + GPUI：会把主要风险从 Agent 闭环转移到 UI 基础设施。
- 不建议把 Agent loop、Shell 和文件权限塞入 Electron main：这会让 CLI、云端执行器、未来移动端控制和自动化入口无法复用。
- 不建议把 Windsurf/Cursor 的“编辑器内 Agent”结构原样复制：它们围绕 LSP 和源码工作区优化；LingMirror 更需要业务对象、审批流、审计和跨平台适配器。

## 最终选型建议

LingMirror 首版可定为：

| 层 | 建议 |
|---|---|
| Desktop shell | Electron |
| UI | React + TypeScript + Vite + Ant Design |
| Local runtime | Go 独立进程 |
| Client-runtime protocol | typed JSON-RPC；首选 stdio，预留 local socket |
| Terminal | xterm.js + Go PTY/ConPTY adapter |
| Local state | SQLite，schema migrations |
| Secrets | Keychain / Credential Manager |
| Policy | risk level + allow/deny + Owner approval + audit |
| Cloud | 现有 Go/Gin AgentOS，通过 HTTPS/WebSocket |
| Distribution | electron-builder 起步；macOS notarization、Windows signing/installer/update 单独验证 |

一句话总结：

> 采用 Codex 的“薄客户端 + 独立 Agent runtime”分层，借用 VS Code/Cursor 已验证的 Electron 安全与进程隔离经验，但不 fork 编辑器；保留 Go 以复用 LingMirror 现有内核和治理资产。
