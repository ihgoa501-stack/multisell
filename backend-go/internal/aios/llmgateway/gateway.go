// Package llmgateway provides an intelligent LLM Gateway that handles model
// routing, semantic caching, and fallback chains for AIOS agent requests.
//
// The gateway abstracts away the complexity of choosing the right LLM model
// for each request, managing cost, latency, and reliability through:
//   - Content-aware model routing (Haiku / Sonnet / Opus)
//   - Semantic caching with TTL-based invalidation
//   - Graceful fallback chains when primary models fail or timeout
//   - Cost and latency tracking per request
//
// Providers are injected from the outside — the gateway itself never calls
// an LLM directly, making it fully testable without real API calls.
package llmgateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Core types
// ---------------------------------------------------------------------------

// Message represents a single message in an LLM conversation.
type Message struct {
	Role    string `json:"role"`    // "user" | "assistant" | "system"
	Content string `json:"content"` // message body
}

// ToolDef describes a function-calling tool available to the LLM.
type ToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Schema      any    `json:"schema,omitempty"`
}

// ToolCallResult represents a tool call the LLM decided to make.
type ToolCallResult struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// Request is the unified input to the LLM Gateway.
type Request struct {
	AgentID  string    `json:"agent_id"`
	UserID   int64     `json:"user_id"`
	System   string    `json:"system"`
	Messages []Message `json:"messages"`
	Tools    []ToolDef `json:"tools,omitempty"`

	MinModel    string        `json:"min_model,omitempty"`
	MaxLatency  time.Duration `json:"max_latency,omitempty"`
	CacheKey    string        `json:"cache_key,omitempty"`
	BypassCache bool          `json:"bypass_cache"`
	Sensitive   bool          `json:"sensitive"`
}

// Response is the unified output from the LLM Gateway.
type Response struct {
	Content   string           `json:"content"`
	ModelUsed string           `json:"model_used"`
	TokensIn  int              `json:"tokens_in"`
	TokensOut int              `json:"tokens_out"`
	Latency   time.Duration    `json:"latency"`
	Cached    bool             `json:"cached"`
	Cost      float64          `json:"estimated_cost_usd"`
	ToolCalls []ToolCallResult `json:"tool_calls,omitempty"`
}

// ---------------------------------------------------------------------------
// Provider interface
// ---------------------------------------------------------------------------

// Provider is the interface that wraps an actual LLM invocation.
// Implementations are injected into the Gateway and never call real LLMs
// during unit tests.
type Provider interface {
	Name() string
	Chat(ctx context.Context, req *Request) (*Response, error)
}

// ---------------------------------------------------------------------------
// RoutingEvent
// ---------------------------------------------------------------------------

// RoutingEvent captures every routing decision for observability and cost analysis.
type RoutingEvent struct {
	AgentID      string  `json:"agent_id"`
	UserID       int64   `json:"user_id"`
	RequestHash  string  `json:"request_hash"`
	RequestSize  int     `json:"request_size"`
	Selected     string  `json:"selected"`
	Reason       string  `json:"reason"`
	LatencyMs    int64   `json:"latency_ms"`
	CostUsd      float64 `json:"cost_usd"`
	CacheHit     bool    `json:"cache_hit"`
	FallbackUsed bool    `json:"fallback_used"`
}

// ---------------------------------------------------------------------------
// Gateway
// ---------------------------------------------------------------------------

// Gateway is the central LLM Gateway that orchestrates model routing, caching,
// LLM invocation, and observability for every agent request.
type Gateway struct {
	router   Router
	cache    Cache
	fallback FallbackChain
	provider Provider
	logger   *zap.Logger
}

// GatewayOption allows functional configuration of a Gateway.
type GatewayOption func(*Gateway)

// WithRouter sets a custom router for the gateway.
func WithRouter(r Router) GatewayOption {
	return func(g *Gateway) { g.router = r }
}

// WithCache sets a custom cache implementation.
func WithCache(c Cache) GatewayOption {
	return func(g *Gateway) { g.cache = c }
}

// WithFallback sets a custom fallback chain.
func WithFallback(f FallbackChain) GatewayOption {
	return func(g *Gateway) { g.fallback = f }
}

// NewGateway creates a new Gateway with the given provider, logger, and options.
// Defaults:
//   - router: DefaultRouter
//   - cache:  MemoryCache with 5-minute TTL
//   - fallback: DefaultFallbackChain
func NewGateway(provider Provider, logger *zap.Logger, opts ...GatewayOption) *Gateway {
	g := &Gateway{
		router:   NewDefaultRouter(),
		cache:    NewMemoryCache(5 * time.Minute),
		fallback: DefaultFallbackChain{},
		provider: provider,
		logger:   logger,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Chat executes the full gateway pipeline:
//  1. Route selection
//  2. Cache lookup
//  3. Provider invocation with fallback
//  4. Cache storage (on success)
//  5. Routing event emission
func (g *Gateway) Chat(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()

	target := g.router.Select(ctx, req)
	g.logger.Debug("model selected",
		zap.String("agent_id", req.AgentID),
		zap.String("model", target.Model),
		zap.String("reason", target.Reason),
	)

	var cacheKey string
	cacheHit := false
	if !req.Sensitive && !req.BypassCache {
		cacheKey = req.CacheKey
		if cacheKey == "" {
			cacheKey = deriveCacheKey(req.System, lastMessage(req.Messages), req.Tools)
		}
		if cached, ok := g.cache.Get(ctx, cacheKey); ok {
			g.logger.Debug("cache hit",
				zap.String("agent_id", req.AgentID),
				zap.String("cache_key", cacheKey),
			)
			cached.Cached = true
			cached.Latency = time.Since(start)
			emitRoutingEvent(g.logger, req, cacheKey, target, true, false, time.Since(start), cached.Cost)
			return cached, nil
		}
	}

	resp, fallbackUsed, err := g.fallback.Execute(ctx, g.provider, target, req)
	elapsed := time.Since(start)

	if err != nil {
		g.logger.Error("LLM gateway call failed",
			zap.String("agent_id", req.AgentID),
			zap.String("model", target.Model),
			zap.Duration("latency", elapsed),
			zap.Error(err),
		)
		return nil, fmt.Errorf("llm gateway: %w", err)
	}

	resp.Latency = elapsed
	resp.Cached = false

	if !req.Sensitive && !req.BypassCache && cacheKey != "" && !fallbackUsed {
		g.cache.Set(ctx, cacheKey, resp, g.cache.DefaultTTL())
	}

	emitRoutingEvent(g.logger, req, cacheKey, target, cacheHit, fallbackUsed, elapsed, resp.Cost)

	g.logger.Info("LLM call completed",
		zap.String("agent_id", req.AgentID),
		zap.String("model", resp.ModelUsed),
		zap.Int("tokens_in", resp.TokensIn),
		zap.Int("tokens_out", resp.TokensOut),
		zap.Duration("latency", elapsed),
		zap.Float64("cost", resp.Cost),
		zap.Bool("cached", cacheHit),
		zap.Bool("fallback_used", fallbackUsed),
	)

	return resp, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var modelHierarchy = map[string]int{
	"haiku":  1,
	"sonnet": 2,
	"opus":   3,
}

func lastMessage(msgs []Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			return msgs[i].Content
		}
	}
	return ""
}

func deriveCacheKey(system string, lastMsg string, tools []ToolDef) string {
	toolSig := ""
	for _, t := range tools {
		toolSig += t.Name + ","
	}
	input := system + "||" + lastMsg + "||" + toolSig
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}

func emitRoutingEvent(logger *zap.Logger, req *Request, hash string, target ModelTarget, cacheHit, fallbackUsed bool, latency time.Duration, cost float64) {
	event := RoutingEvent{
		AgentID:      req.AgentID,
		UserID:       req.UserID,
		RequestHash:  hash,
		RequestSize:  estimateTokens(req),
		Selected:     target.Model,
		Reason:       target.Reason,
		LatencyMs:    latency.Milliseconds(),
		CostUsd:      cost,
		CacheHit:     cacheHit,
		FallbackUsed: fallbackUsed,
	}
	if cacheHit {
		logger.Debug("routing event", zap.Any("event", event))
	} else {
		logger.Info("routing event", zap.Any("event", event))
	}
}

func estimateTokens(req *Request) int {
	total := len(req.System) + len(lastMessage(req.Messages))
	for _, msg := range req.Messages {
		total += len(msg.Content)
	}
	return total / 4
}

func modelTier(model string) int {
	base := strings.ToLower(model)
	for _, prefix := range []string{"claude-", "gpt-", "gemini-"} {
		base = strings.TrimPrefix(base, prefix)
	}
	for tier, value := range modelHierarchy {
		if strings.Contains(base, tier) {
			return value
		}
	}
	return 0
}

func maxModel(a, b string) string {
	if modelTier(a) >= modelTier(b) {
		return a
	}
	return b
}

// ---------------------------------------------------------------------------
// Testing helpers
// ---------------------------------------------------------------------------

// Ensure Provider interface is satisfied at compile time.
var _ Provider = (*mockProvider)(nil)

// mockProvider is a test double that returns a fixed response or error.
type mockProvider struct {
	name     string
	response *Response
	err      error
}

func (m *mockProvider) Name() string {
	if m.name != "" {
		return m.name
	}
	return "mock"
}

func (m *mockProvider) Chat(_ context.Context, _ *Request) (*Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}
