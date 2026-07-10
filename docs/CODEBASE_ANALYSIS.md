# 代码库解析说明

## 用途

本项目已经生成一份代码库结构地图，用于帮助 Agent 快速理解模块、调用关系、架构层和阅读顺序。它是分析辅助资料，不是业务事实源，也不替代当前源代码、治理文档或 `CodeGraph`。

## 当前解析快照

- 解析项目：LingMirror / MultiSell
- 对应 commit：`b18bd5a75c191c9c3d71d702c31693ab8c1ff3e0`
- 解析文件：1,283 个
- 图谱节点：3,412 个
- 图谱关系：7,390 条
- 架构层：10 个
- 导览步骤：12 步
- 当前有效技术栈：`backend-go/` + `frontend-next/`
- `backend/`、`frontend/` 等旧 Python/Vue 路径仅作为历史参考

图谱文件位于：

`.understand-anything/knowledge-graph.json`

该目录通常属于本地分析产物，可能不会进入 Git。若文件不存在，应以当前源代码和治理文档为准，并重新执行代码库解析。

## Agent 使用方式

1. 先阅读本文件、`AGENTS.md` 和必读治理文档。
2. 需要了解结构时，优先使用 CodeGraph 查询当前符号和调用链。
3. 若存在 `knowledge-graph.json`，可用它查看全局架构、层级和导览；涉及具体实现时必须回到当前源文件确认。
4. 不要把图谱中的历史路径、旧栈节点或历史测试结果当作当前实现事实。

## 重新解析

重新解析前确认工作区状态，并保留用户未提交修改。解析只读扫描业务源码，但会更新 `.understand-anything/` 下的分析产物。解析范围、排除规则和输出语言由：

`.understand-anything/.understandignore`

控制。项目结构发生较大变化后，应重新生成图谱并更新本快照中的 commit、统计数字和说明。
