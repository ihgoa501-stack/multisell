package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

// LLM cache Prometheus counters.
// Use in PromQL: cache_hit_ratio = multisell_llm_cache_hit_tokens_total / (multisell_llm_cache_hit_tokens_total + multisell_llm_cache_miss_tokens_total)
var (
	llmCacheHitTokens = promauto.NewCounter(prometheus.CounterOpts{
		Name: "multisell_llm_cache_hit_tokens_total",
		Help: "Total input tokens served from prompt cache (cache hit).",
	})
	llmCacheMissTokens = promauto.NewCounter(prometheus.CounterOpts{
		Name: "multisell_llm_cache_miss_tokens_total",
		Help: "Total input tokens that were NOT served from prompt cache (cache miss).",
	})
	llmCacheCreationTokens = promauto.NewCounter(prometheus.CounterOpts{
		Name: "multisell_llm_cache_creation_tokens_total",
		Help: "Total tokens spent creating new cache entries (cache creation).",
	})
)

// LLMProvider is the abstract interface for language model backends.
type LLMProvider interface {
	// Chat sends a completion request and returns the model's text answer
	// plus token usage metadata.
	Chat(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
	// ChatStream sends a completion request and emits token chunks on the
	// returned channel. The channel closes when the stream ends.
	ChatStream(ctx context.Context, req *LLMRequest) (<-chan LLMChunk, error)
	// Name returns the provider identifier ("openai" | "anthropic" | "qwen" | "stub").
	Name() string
}

// LLMRequest is the provider-agnostic completion request.
type LLMRequest struct {
	Model       string                 `json:"model"`
	System      string                 `json:"system,omitempty"`
	Messages    []LLMMessage           `json:"messages"`
	Temperature float64                `json:"temperature,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// LLMMessage is a single chat message.
type LLMMessage struct {
	Role    string `json:"role"` // system | user | assistant | tool
	Content string `json:"content"`
}

// LLMResponse is the full completion response.
type LLMResponse struct {
	Answer      string  `json:"answer"`
	Model       string  `json:"model"`
	TokensIn    int     `json:"tokens_in"`
	TokensOut   int     `json:"tokens_out"`
	LatencyMs   int     `json:"latency_ms"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// LLMChunk is one streamed token.
type LLMChunk struct {
	Text  string
	Err   error
	Done  bool
}

// NewLLMProvider constructs the provider named by LLM_PROVIDER env var.
// Falls back to StubProvider when unset or "stub". Supports "openai" and
// "anthropic" via OpenAI-compatible endpoints; "qwen" uses DashScope's
// OpenAI-compatible /compatible-mode/v1 endpoint.
//
// In production (ENV=production) the stub provider is forbidden and a valid
// LLM_API_KEY is required — the process will terminate if these conditions
// are not met.
func NewLLMProvider(logger *zap.Logger) LLMProvider {
	name := strings.ToLower(strings.TrimSpace(os.Getenv("LLM_PROVIDER")))
	envVal := strings.ToLower(strings.TrimSpace(os.Getenv("ENV")))
	ginVal := strings.ToLower(strings.TrimSpace(os.Getenv("GIN_MODE")))
	isProd := envVal == "production" || ginVal == "release"

	switch name {
	case "openai", "qwen", "deepseek", "azure":
		if isProd && strings.TrimSpace(os.Getenv("LLM_API_KEY")) == "" {
			logger.Fatal("LLM_API_KEY is required in production",
				zap.String("provider", name))
		}
		return newOpenAICompatible(name, logger)
	case "anthropic":
		if isProd && strings.TrimSpace(os.Getenv("LLM_API_KEY")) == "" {
			logger.Fatal("LLM_API_KEY is required in production",
				zap.String("provider", name))
		}
		return newAnthropic(logger)
	case "", "stub":
		if isProd {
			logger.Fatal("LLM_PROVIDER must be set to a real provider in production (not empty/stub). " +
				"Supported: openai, anthropic, qwen, deepseek, azure")
		}
		logger.Warn("LLM_PROVIDER not set, using stub provider for development. " +
			"Set LLM_PROVIDER=openai, anthropic, or qwen for real AI calls.")
		return &StubProvider{logger: logger}
	default:
		if isProd {
			logger.Fatal("unsupported LLM_PROVIDER in production; use: openai, anthropic, qwen, deepseek, azure",
				zap.String("provider", name))
		}
		logger.Warn("unsupported LLM_PROVIDER, falling back to stub provider for development",
			zap.String("provider", name))
		return &StubProvider{logger: logger}
	}
}

// ---------- OpenAI-compatible provider ----------

type openAICompatible struct {
	name    string
	apiKey  string
	baseURL string
	model   string
	http    *http.Client
	logger  *zap.Logger
}

func newOpenAICompatible(name string, logger *zap.Logger) *openAICompatible {
	baseURL := os.Getenv("LLM_BASE_URL")
	if baseURL == "" {
		switch name {
		case "qwen":
			baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		case "deepseek":
			baseURL = "https://api.deepseek.com/v1"
		default:
			baseURL = "https://api.openai.com/v1"
		}
	}
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		switch name {
		case "qwen":
			model = "qwen-plus"
		case "deepseek":
			model = "deepseek-chat"
		default:
			model = "gpt-4o-mini"
		}
	}
	return &openAICompatible{
		name:    name,
		apiKey:  os.Getenv("LLM_API_KEY"),
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		http:    &http.Client{Timeout: 60 * time.Second},
		logger:  logger,
	}
}

func (p *openAICompatible) Name() string { return p.name }

func (p *openAICompatible) Chat(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if p.apiKey == "" {
		return nil, errors.New("LLM_API_KEY not set; cannot call real LLM")
	}
	if req.Model == "" {
		req.Model = p.model
	}
	start := time.Now()
	payload := p.buildPayload(req, false)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM %s returned %d: %s", p.name, resp.StatusCode, truncate(string(respBody), 300))
	}

	var parsed struct {
		Choices []struct {
			Message      LLMMessage `json:"message"`
			FinishReason string     `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			PromptTokensDetails *struct {
				CachedTokens int `json:"cached_tokens"`
			} `json:"prompt_tokens_details,omitempty"`
		} `json:"usage"`
		Model string `json:"model"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("LLM response parse failed: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return nil, errors.New("LLM returned no choices")
	}

	// Track cache token counters.
	pt := parsed.Usage.PromptTokens
	if d := parsed.Usage.PromptTokensDetails; d != nil && d.CachedTokens > 0 {
		llmCacheHitTokens.Add(float64(d.CachedTokens))
		llmCacheMissTokens.Add(float64(pt - d.CachedTokens))
	} else {
		llmCacheMissTokens.Add(float64(pt))
	}
	return &LLMResponse{
		Answer:       parsed.Choices[0].Message.Content,
		Model:        parsed.Model,
		TokensIn:     parsed.Usage.PromptTokens,
		TokensOut:    parsed.Usage.CompletionTokens,
		LatencyMs:    int(time.Since(start).Milliseconds()),
		FinishReason: parsed.Choices[0].FinishReason,
	}, nil
}

func (p *openAICompatible) ChatStream(ctx context.Context, req *LLMRequest) (<-chan LLMChunk, error) {
	if p.apiKey == "" {
		return nil, errors.New("LLM_API_KEY not set; cannot call real LLM")
	}
	if req.Model == "" {
		req.Model = p.model
	}
	payload := p.buildPayload(req, true)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM stream returned %d: %s", resp.StatusCode, truncate(string(errBody), 300))
	}

	ch := make(chan LLMChunk, 32)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		// Increase buffer for long lines (some SSE payloads span multiple KB).
		scanner.Buffer(make([]byte, 0, 16384), 65536)

		for scanner.Scan() {
			line := scanner.Text()

			// SSE uses empty line as event separator.
			if line == "" {
				continue
			}
			// Only process lines with the SSE data: prefix.
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			payload := strings.TrimPrefix(line, "data: ")

			// OpenAI sends "data: [DONE]" to signal stream end.
			if payload == "[DONE]" {
				ch <- LLMChunk{Done: true}
				return
			}

			var ev struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
					FinishReason string `json:"finish_reason"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(payload), &ev); err != nil {
				ch <- LLMChunk{Err: fmt.Errorf("SSE parse: %w", err)}
				ch <- LLMChunk{Done: true}
				return
			}

			if len(ev.Choices) > 0 {
				c := ev.Choices[0]
				if c.Delta.Content != "" {
					ch <- LLMChunk{Text: c.Delta.Content}
				}
				if c.FinishReason != "" {
					ch <- LLMChunk{Done: true}
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- LLMChunk{Err: fmt.Errorf("SSE read error: %w", err)}
		}
		ch <- LLMChunk{Done: true}
	}()
	return ch, nil
}

func (p *openAICompatible) buildPayload(req *LLMRequest, stream bool) map[string]interface{} {
	msgs := make([]map[string]string, 0, len(req.Messages)+1)
	if req.System != "" {
		msgs = append(msgs, map[string]string{"role": "system", "content": req.System})
	}
	for _, m := range req.Messages {
		msgs = append(msgs, map[string]string{"role": m.Role, "content": m.Content})
	}
	payload := map[string]interface{}{
		"model":    req.Model,
		"messages": msgs,
	}
	if req.Temperature > 0 {
		payload["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		payload["max_tokens"] = req.MaxTokens
	}
	if stream {
		payload["stream"] = true
	}
	return payload
}

// ---------- Anthropic provider (Messages API) ----------

type anthropicProvider struct {
	apiKey string
	model  string
	http   *http.Client
	logger *zap.Logger
}

func newAnthropic(logger *zap.Logger) *anthropicProvider {
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}
	return &anthropicProvider{
		apiKey: os.Getenv("LLM_API_KEY"),
		model:  model,
		http:   &http.Client{Timeout: 60 * time.Second},
		logger: logger,
	}
}

func (p *anthropicProvider) Name() string { return "anthropic" }

func (p *anthropicProvider) Chat(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if p.apiKey == "" {
		return nil, errors.New("LLM_API_KEY not set; cannot call Anthropic")
	}
	if req.Model == "" {
		req.Model = p.model
	}
	start := time.Now()
	msgs := make([]map[string]string, 0, len(req.Messages))
	for _, m := range req.Messages {
		msgs = append(msgs, map[string]string{"role": m.Role, "content": m.Content})
	}
	payload := map[string]interface{}{
		"model":      req.Model,
		"max_tokens": req.MaxTokens,
		"messages":   msgs,
	}
	if req.System != "" {
		payload["system"] = req.System
	}
	body, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Anthropic returned %d: %s", resp.StatusCode, truncate(string(respBody), 300))
	}
	var parsed struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
			CacheReadInputTokens    int `json:"cache_read_input_tokens,omitempty"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
		} `json:"usage"`
		Model string `json:"model"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	answer := ""
	if len(parsed.Content) > 0 {
		answer = parsed.Content[0].Text
	}

	// Track cache token counters.
	pt := parsed.Usage.InputTokens
	if parsed.Usage.CacheReadInputTokens > 0 {
		llmCacheHitTokens.Add(float64(parsed.Usage.CacheReadInputTokens))
		llmCacheMissTokens.Add(float64(pt - parsed.Usage.CacheReadInputTokens))
	} else {
		llmCacheMissTokens.Add(float64(pt))
	}
	if parsed.Usage.CacheCreationInputTokens > 0 {
		llmCacheCreationTokens.Add(float64(parsed.Usage.CacheCreationInputTokens))
	}
	return &LLMResponse{
		Answer:    answer,
		Model:     parsed.Model,
		TokensIn:  parsed.Usage.InputTokens,
		TokensOut: parsed.Usage.OutputTokens,
		LatencyMs: int(time.Since(start).Milliseconds()),
	}, nil
}

func (p *anthropicProvider) ChatStream(ctx context.Context, req *LLMRequest) (<-chan LLMChunk, error) {
	// Anthropic streaming uses a different SSE event schema; for simplicity we
	// fall back to non-streaming + single-chunk emission.
	resp, err := p.Chat(ctx, req)
	if err != nil {
		return nil, err
	}
	ch := make(chan LLMChunk, 2)
	go func() {
		defer close(ch)
		ch <- LLMChunk{Text: resp.Answer}
		ch <- LLMChunk{Done: true}
	}()
	return ch, nil
}

// ---------- Stub provider (default when LLM_PROVIDER unset) ----------

// StubProvider returns deterministic answers. It is the default so the AI
// runtime works without any external API key.
type StubProvider struct {
	logger *zap.Logger
}

func (s *StubProvider) Name() string { return "stub" }

func (s *StubProvider) Chat(_ context.Context, req *LLMRequest) (*LLMResponse, error) {
	answer := stubLLMAnswer(req)
	return &LLMResponse{
		Answer:    answer,
		Model:     "stub-v1",
		TokensIn:  countApproxTokens(req),
		TokensOut: 180,
		LatencyMs: 12,
	}, nil
}

func (s *StubProvider) ChatStream(_ context.Context, req *LLMRequest) (<-chan LLMChunk, error) {
	ch := make(chan LLMChunk, 16)
	answer := stubLLMAnswer(req)
	go func() {
		defer close(ch)
		runes := []rune(answer)
		step := 18
		for i := 0; i < len(runes); i += step {
			end := i + step
			if end > len(runes) {
				end = len(runes)
			}
			ch <- LLMChunk{Text: string(runes[i:end])}
			time.Sleep(15 * time.Millisecond)
		}
		ch <- LLMChunk{Done: true}
	}()
	return ch, nil
}

func stubLLMAnswer(req *LLMRequest) string {
	if len(req.Messages) == 0 {
		return "已生成建议，详见 trace。"
	}
	userMsg := req.Messages[len(req.Messages)-1].Content
	// Mirror the orchestrator's stub recommendation routing so /ai/chat
	// produces something coherent even without a real LLM.
	lower := strings.ToLower(userMsg)
	switch {
	case strings.Contains(lower, "库存"), strings.Contains(lower, "缺货"):
		return "建议立即补货 200 件，并切换至渠道 B（运费低 18%）。"
	case strings.Contains(lower, "利润"), strings.Contains(lower, "成本"):
		return "建议将该 SKU 售价上调 6%，否则利润率将持续低于目标。"
	case strings.Contains(lower, "listing"), strings.Contains(lower, "标题"):
		return "建议重写标题：前置核心关键词，描述补充规格卖点。"
	case strings.Contains(lower, "折扣"), strings.Contains(lower, "促销"):
		return "折扣 25% 触发价格底线，建议下调至 18%。"
	case strings.Contains(lower, "合规"), strings.Contains(lower, "认证"):
		return "建议移除标题中的敏感词，提交 CE 认证材料。"
	}
	return "已收到指令，已生成推理 trace。在未配置 LLM_PROVIDER 时使用 stub 应答；设置 LLM_PROVIDER=openai 和 LLM_API_KEY 后将接入真实模型。"
}

func countApproxTokens(req *LLMRequest) int {
	n := 0
	for _, m := range req.Messages {
		n += len([]rune(m.Content)) / 2
	}
	return n
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
