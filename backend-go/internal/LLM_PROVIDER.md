# LLM Provider 配置

LingMirror AI Runtime 支持多个 LLM provider，通过环境变量切换。默认使用 stub（无需任何配置即可演示 UI 流程）。

## 配置项

| 环境变量 | 必填 | 默认 | 说明 |
|---|---|---|---|
| `LLM_PROVIDER` | 否 | `stub` | `stub` / `openai` / `anthropic` / `qwen` / `deepseek` / `azure` |
| `LLM_API_KEY` | provider≠stub 时必填 | — | 对应平台的 API key |
| `LLM_MODEL` | 否 | 按平台默认 | `gpt-4o-mini` / `claude-3-5-sonnet-20241022` / `qwen-plus` / `deepseek-chat` |
| `LLM_BASE_URL` | 否 | 按平台默认 | 自定义 OpenAI 兼容端点（如 Azure OpenAI、私有部署 vLLM） |

## 各平台默认值

| Provider | Base URL | 默认 Model |
|---|---|---|
| `openai` | `https://api.openai.com/v1` | `gpt-4o-mini` |
| `qwen` | `https://dashscope.aliyuncs.com/compatible-mode/v1` | `qwen-plus` |
| `deepseek` | `https://api.deepseek.com/v1` | `deepseek-chat` |
| `anthropic` | `https://api.anthropic.com/v1` | `claude-3-5-sonnet-20241022` |
| `stub` | — | `stub-v1`（本地确定性应答） |

## 启用真实 LLM 示例

```bash
# OpenAI
export LLM_PROVIDER=openai
export LLM_API_KEY=sk-xxxx
export LLM_MODEL=gpt-4o-mini

# 通义千问
export LLM_PROVIDER=qwen
export LLM_API_KEY=sk-xxxx
export LLM_MODEL=qwen-plus

# DeepSeek
export LLM_PROVIDER=deepseek
export LLM_API_KEY=sk-xxxx

# Anthropic
export LLM_PROVIDER=anthropic
export LLM_API_KEY=sk-ant-xxxx
```

## 行为说明

- **stub provider**：返回与 `ai.Orchestrator.synthesizeOutput` 兼容的确定性文本，不调任何外部 API。用于开发/演示/测试，无网络依赖。
- **真实 provider**：Orchestrator 在每次 `Run` / `Chat` 时构造 prompt（含 agent spec + decision_point + context），调用 `provider.Chat`，把响应写入 `ai_trace.final_output` + 自动创建 `unified_action`。调用失败时自动降级到 stub 并记日志。
- **流式**：`POST /ai/chat` 带 `stream: true` 时，OpenAI 兼容 provider 用 SSE 透传 token；Anthropic 当前降级为单次返回（其流式协议不同）。
- **超时**：单次调用 30s 超时，HTTP client 60s 超时。
- **Token 计数**：响应中的 `token_count` 来自 provider 返回的 usage（stub 用近似估算）。
