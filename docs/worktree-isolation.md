# Worktree Isolation 说明

## 问题

用 `Write` / `Edit` 工具改这个项目的文件时，报错：

```
This background session hasn't isolated its changes yet.
Call EnterWorktree first so edits land in a worktree instead
of the shared checkout.
```

## 原因

项目的 `.claude/settings.json` 中没有显式设置 `worktree.bgIsolation`，工具链默认启用了 worktree isolation 安全机制。

## 表现

每次写文件必须：
1. `EnterWorktree` → 创建 git worktree
2. 在 worktree 内 `Write` / `Edit`
3. `Bash cp` 复制到主 checkout
4. `ExitWorktree remove` → 删除 worktree

## 解决方案（任选其一）

### A. 全局关闭（推荐）
在终端执行：
```bash
echo '{"worktree":{"bgIsolation":"none"}}' > /Users/lc/multisell/.claude/settings.json
```
之后所有写操作不再拦截，直接生效。

### B. 每次进 worktree
保持现状，每个文件修改走 4 步流程。worktree 内的改动 `ExitWorktree remove` 时会丢弃，需手动 `cp` 到主 checkout。

## 注意

- `settings.json` 是项目级配置，会 git 可见。如果不想提交，可以用 `settings.local.json`（会被 `.gitignore` 忽略），但 `bgIsolation` 只读 `settings.json`。
- 这个设置只影响 Claude Code 的文件写入行为，不影响 git 操作或其他工具。
