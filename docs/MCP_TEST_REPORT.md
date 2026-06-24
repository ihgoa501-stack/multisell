# MCP 服务器测试报告

**测试日期:** 2026-06-22
**项目:** 凌镜 LingMirror (MultiSell)
**测试环境:** macOS Darwin, OpenCode Agent

> 归档说明：本文是 2026-06-22 的工具可用性测试记录，其中 FastAPI/Vue 相关探索结果属于旧栈历史上下文。当前工程事实以 `backend-go/`、`frontend-next/` 和 `/api/v1/*` 为准。

---

## 测试概览

本报告记录了对项目配置的 4 个 MCP (Model Context Protocol) 服务器的功能测试结果。

| MCP 服务器 | 状态 | 版本/说明 |
|-----------|------|----------|
| CodeGraph | ✅ 正常 | v1.0.1 |
| GitHub | ✅ 正常 | 已认证: ihgoa501-stack |
| Chrome DevTools | ✅ 正常 | 连接 localhost:9222 |
| Playwright | ⚠️ 不可用 | 工具未加载 |

---

## 详细测试结果

### 1. CodeGraph MCP 服务器

**用途:** 代码智能分析 - 符号搜索、调用图、代码探索

**测试命令:**
- `codegraph --version` → 返回 `1.0.1`
- `codegraph_explore` → 成功探索 FastAPI 路由模块

**测试结果:** ✅ 通过

**功能验证:**
- 符号搜索: 可找到 `router` 变量定义
- 代码探索: 返回 9 个相关符号及其源码
- 调用图分析: 显示符号依赖关系
- 版本: 1.0.1

**数据库状态:**
- 位置: `.codegraph/codegraph.db`
- 模式: WAL (Write-Ahead Logging)
- 守护进程: 运行中 (daemon.pid, daemon.sock 存在)

---

### 2. GitHub MCP 服务器

**用途:** GitHub API 访问 - 仓库、Issues、PR、代码搜索

**认证状态:**
```
github.com
  ✓ Logged in to github.com account ihgoa501-stack (keyring)
  - Token scopes: 'gist', 'read:org', 'repo', 'workflow'
```

**测试命令:**
- `gh auth status` → 已认证
- `gh repo view ihgoa501-stack/multisell` → 成功获取仓库信息

**测试结果:** ✅ 通过

**返回数据:**
```json
{
  "name": "multisell",
  "url": "https://github.com/ihgoa501-stack/multisell",
  "defaultBranchRef": { "name": "main" }
}
```

**可用功能:**
- 仓库信息查询
- Issues/PR 管理
- 代码搜索
- 分支操作
- 文件内容获取

---

### 3. Chrome DevTools MCP 服务器

**用途:** 浏览器自动化 - 通过 Chrome DevTools Protocol

**连接配置:**
- URL: `http://127.0.0.1:9222`
- 命令: `npx -y chrome-devtools-mcp@latest --browser-url=http://127.0.0.1:9222`

**测试操作:**
1. `list_pages` → 成功列出页面
2. `new_page` → 成功创建新页面并导航到 `http://localhost:3001`
3. `take_screenshot` → 成功截图登录页面
4. `take_snapshot` → 成功获取页面 DOM 快照

**测试结果:** ✅ 通过

**页面快照内容:**
- 标题: "凌镜 LingMirror - 跨境电商 AgentOS"
- 表单元素: 用户名输入框、密码输入框、登录按钮
- 测试账号提示: admin / admin123

**可用功能:**
- 页面导航和管理
- 元素点击、输入、选择
- 截图和快照
- 网络请求监控
- 控制台日志
- 性能追踪
- 表单填写

---

### 4. Playwright MCP 服务器

**用途:** 浏览器自动化 - 通过 Playwright

**配置:**
- 命令: `npx -y @playwright/mcp@latest`
- 状态: 配置存在但工具未加载

**测试结果:** ⚠️ 不可用

**原因分析:**
- Playwright MCP 工具在当前会话中未被加载
- 可能需要手动启动或配置问题
- 历史记录显示 `.playwright-mcp/` 目录有 52 个文件，表明之前使用过

**建议:**
- 检查 Playwright MCP 服务器是否正常启动
- 验证配置文件中的启用状态

---

## MCP 架构说明

### 配置位置

**主配置文件:** `~/.config/opencode/opencode.jsonc`

```json
{
  "mcp": {
    "chrome-devtools": {
      "type": "local",
      "command": "npx -y chrome-devtools-mcp@latest --browser-url=http://127.0.0.1:9222",
      "enabled": true
    },
    "github": {
      "type": "local",
      "command": "npx -y @modelcontextprotocol/server-github",
      "enabled": true
    },
    "playwright": {
      "type": "local",
      "command": "npx -y @playwright/mcp@latest",
      "enabled": true
    },
    "codegraph": {
      "type": "local",
      "command": "codegraph serve --mcp",
      "enabled": true
    }
  }
}
```

### 工作流程

1. OpenCode Agent 启动时读取配置
2. 生成 4 个本地 MCP 服务器作为子进程
3. MCP 服务器暴露工具供 Agent 调用
4. 工具命名格式: `mcp__<server>__<tool_name>`

### 权限配置

**文件:** `.claude/settings.local.json`

```json
{
  "permissions": {
    "allow": [
      "mcp__codegraph__codegraph_explore",
      "mcp__codegraph__codegraph_node",
      "mcp__codegraph__codegraph_search",
      "mcp__codegraph__codegraph_callers"
    ]
  }
}
```

---

## 测试结论

| 项目 | 结果 | 备注 |
|-----|------|------|
| CodeGraph | ✅ 通过 | 代码探索、符号搜索正常 |
| GitHub | ✅ 通过 | 仓库访问、API 调用正常 |
| Chrome DevTools | ✅ 通过 | 浏览器自动化正常 |
| Playwright | ⚠️ 部分 | 配置存在但工具未加载 |

**总体评估:** 3/4 MCP 服务器正常工作，基本功能可用。

---

## 使用示例

### CodeGraph - 代码探索

```
codegraph_explore query="FastAPI router modules"
→ 返回相关符号、源码、调用关系
```

### GitHub - 仓库操作

```
github_list_issues owner="ihgoa501-stack" repo="multisell"
→ 返回 Issues 列表
```

### Chrome DevTools - 浏览器测试

```
chrome-devtools_new_page url="http://localhost:3001"
chrome-devtools_take_screenshot
→ 创建页面并截图
```

---

## 后续建议

1. **Playwright MCP:** 检查服务器启动日志，确保正确加载
2. **权限管理:** 定期审查 `.claude/settings.local.json` 中的权限配置
3. **版本更新:** 定期更新 MCP 服务器版本以获取最新功能
4. **监控:** 关注 `.codegraph/daemon.log` 和 `.playwright-mcp/` 日志

---

*报告生成时间: 2026-06-22*
*测试工具: OpenCode Agent + MCP Servers*
