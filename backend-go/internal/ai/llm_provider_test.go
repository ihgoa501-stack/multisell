package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStubProvider_Name(t *testing.T) {
	p := &StubProvider{logger: testLogger()}
	if got := p.Name(); got != "stub" {
		t.Errorf("Name() = %q, want %q", got, "stub")
	}
}

func TestOpenAICompatibleChatRoundTripsStructuredToolCalls(t *testing.T) {
	var requestBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"test-model","choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"call-1","type":"function","function":{"name":"demand_case_read","arguments":"{\"demand_case_id\":7}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":12,"completion_tokens":3}}`))
	}))
	defer server.Close()

	provider := &openAICompatible{name: "openai", apiKey: "test-key", baseURL: server.URL, model: "test-model", http: &http.Client{Timeout: time.Second}, logger: testLogger()}
	resp, err := provider.Chat(context.Background(), &LLMRequest{
		Messages: []LLMMessage{
			{Role: "user", Content: "查看案件"},
			{Role: "assistant", ToolCalls: []LLMToolCall{{ID: "previous", Name: "demand_case_read", Arguments: json.RawMessage(`{"demand_case_id":7}`)}}},
			{Role: "tool", ToolCallID: "previous", Name: "demand_case_read", Content: `{"ok":true}`},
		},
		Tools: []LLMTool{{Name: "demand_case_read", Description: "read", Strict: true, Parameters: map[string]interface{}{"type": "object", "additionalProperties": false}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "demand_case_read" || string(resp.ToolCalls[0].Arguments) != `{"demand_case_id":7}` {
		t.Fatalf("tool response not parsed: %#v", resp)
	}
	tools, ok := requestBody["tools"].([]interface{})
	if !ok || len(tools) != 1 || requestBody["tool_choice"] != "auto" {
		t.Fatalf("tool definition missing: %#v", requestBody)
	}
	messages, ok := requestBody["messages"].([]interface{})
	if !ok || len(messages) != 3 || messages[2].(map[string]interface{})["tool_call_id"] != "previous" {
		t.Fatalf("tool continuation missing: %#v", requestBody["messages"])
	}
}

func TestAnthropicChatRoundTripsStructuredToolCalls(t *testing.T) {
	var requestBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"claude-test","stop_reason":"tool_use","content":[{"type":"text","text":"先读取。"},{"type":"tool_use","id":"toolu-1","name":"demand_case_read","input":{"demand_case_id":7}}],"usage":{"input_tokens":10,"output_tokens":4}}`))
	}))
	defer server.Close()

	provider := &anthropicProvider{apiKey: "test-key", baseURL: server.URL, model: "claude-test", http: &http.Client{Timeout: time.Second}, logger: testLogger()}
	resp, err := provider.Chat(context.Background(), &LLMRequest{
		Messages: []LLMMessage{{Role: "user", Content: "查看"}, {Role: "assistant", ToolCalls: []LLMToolCall{{ID: "old", Name: "demand_case_read", Arguments: json.RawMessage(`{"demand_case_id":7}`)}}}, {Role: "tool", ToolCallID: "old", Content: `{"ok":true}`}},
		Tools:    []LLMTool{{Name: "demand_case_read", Parameters: map[string]interface{}{"type": "object"}}}, MaxTokens: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Answer != "先读取。" || len(resp.ToolCalls) != 1 || resp.ToolCalls[0].ID != "toolu-1" || string(resp.ToolCalls[0].Arguments) != `{"demand_case_id":7}` {
		t.Fatalf("anthropic tool response not parsed: %#v", resp)
	}
	if tools, ok := requestBody["tools"].([]interface{}); !ok || len(tools) != 1 {
		t.Fatalf("anthropic tools missing: %#v", requestBody)
	}
	if messages, ok := requestBody["messages"].([]interface{}); !ok || len(messages) != 3 || messages[2].(map[string]interface{})["role"] != "user" {
		t.Fatalf("anthropic tool result continuation missing: %#v", requestBody["messages"])
	}
}

func TestStubProvider_Chat_Inventory(t *testing.T) {
	p := &StubProvider{logger: testLogger()}
	resp, err := p.Chat(context.Background(), &LLMRequest{
		Messages: []LLMMessage{{Role: "user", Content: "缺货"}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if !strings.Contains(resp.Answer, "补货") {
		t.Errorf("answer = %q, want 补货", resp.Answer)
	}
	if resp.Model != "stub-v1" {
		t.Errorf("model = %s", resp.Model)
	}
	if resp.TokensIn <= 0 {
		t.Errorf("TokensIn = %d, want > 0", resp.TokensIn)
	}
}

func TestStubProvider_Chat_Profit(t *testing.T) {
	p := &StubProvider{logger: testLogger()}
	resp, err := p.Chat(context.Background(), &LLMRequest{
		Messages: []LLMMessage{{Role: "user", Content: "利润太低"}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if !strings.Contains(resp.Answer, "利润率") && !strings.Contains(resp.Answer, "成本") {
		t.Errorf("answer = %q, want 利润率 or 成本", resp.Answer)
	}
}

func TestStubProvider_Chat_EmptyMessages(t *testing.T) {
	p := &StubProvider{logger: testLogger()}
	resp, err := p.Chat(context.Background(), &LLMRequest{})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Answer == "" {
		t.Error("expected non-empty answer for empty messages")
	}
}

func TestStubProvider_Stream_NoErrors(t *testing.T) {
	p := &StubProvider{logger: testLogger()}
	ch, err := p.ChatStream(context.Background(), &LLMRequest{
		Messages: []LLMMessage{{Role: "user", Content: "库存"}},
	})
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	gotDone := false
	gotErr := false
	for chunk := range ch {
		if chunk.Err != nil {
			gotErr = true
			t.Errorf("unexpected error in stream: %v", chunk.Err)
		}
		if chunk.Done {
			gotDone = true
		}
	}
	if gotErr {
		t.Fatal("stream should not produce errors")
	}
	if !gotDone {
		t.Fatal("stream did not close with Done=true")
	}
}

func TestStubProvider_Stream_MatchesNonStreaming(t *testing.T) {
	p := &StubProvider{logger: testLogger()}
	req := &LLMRequest{
		Messages: []LLMMessage{{Role: "user", Content: "合规认证检查"}},
	}

	// Non-streaming answer.
	resp, err := p.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	// Streaming answer should be identical.
	ch, err := p.ChatStream(context.Background(), req)
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	var sb strings.Builder
	for chunk := range ch {
		sb.WriteString(chunk.Text)
	}
	if sb.String() != resp.Answer {
		t.Errorf("stream output = %q, want %q", sb.String(), resp.Answer)
	}
}

func TestStubLLMAnswer_AllRoutes(t *testing.T) {
	tests := []struct {
		name string
		req  *LLMRequest
		want string
	}{
		{"stock_short", &LLMRequest{Messages: []LLMMessage{{Content: "库存不够了"}}}, "补货"},
		{"out_of_stock", &LLMRequest{Messages: []LLMMessage{{Content: "缺货严重"}}}, "补货"},
		{"profit", &LLMRequest{Messages: []LLMMessage{{Content: "利润下降"}}}, "利润率"},
		{"cost", &LLMRequest{Messages: []LLMMessage{{Content: "成本太高"}}}, "利润率"},
		{"listing", &LLMRequest{Messages: []LLMMessage{{Content: "优化 listing"}}}, "标题"},
		{"title", &LLMRequest{Messages: []LLMMessage{{Content: "标题重写"}}}, "标题"},
		{"discount", &LLMRequest{Messages: []LLMMessage{{Content: "折扣活动"}}}, "折扣"},
		{"promotion", &LLMRequest{Messages: []LLMMessage{{Content: "促销方案"}}}, "折扣"},
		{"compliance", &LLMRequest{Messages: []LLMMessage{{Content: "合规检查"}}}, "认证"},
		{"certification", &LLMRequest{Messages: []LLMMessage{{Content: "CE 认证"}}}, "认证"},
		{"default_fallback", &LLMRequest{Messages: []LLMMessage{{Content: "你好世界"}}}, "已收到指令"},
		{"no_messages", &LLMRequest{}, "已生成建议"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stubLLMAnswer(tt.req)
			if !strings.Contains(got, tt.want) {
				t.Errorf("stubLLMAnswer = %q, want substring %q", got, tt.want)
			}
		})
	}
}

func TestCountApproxTokens(t *testing.T) {
	req := &LLMRequest{
		Messages: []LLMMessage{
			{Role: "user", Content: "检查库存"},
		},
	}
	n := countApproxTokens(req)
	if n <= 0 {
		t.Errorf("countApproxTokens = %d, want > 0", n)
	}
}

func TestCountApproxTokens_Empty(t *testing.T) {
	req := &LLMRequest{}
	n := countApproxTokens(req)
	if n != 0 {
		t.Errorf("countApproxTokens = %d, want 0", n)
	}
}

func TestReadLLMResponseRejectsOversizeBody(t *testing.T) {
	_, err := readLLMResponse(strings.NewReader(strings.Repeat("x", maxLLMResponseBytes+1)))
	if err == nil || !strings.Contains(err.Error(), "size limit") {
		t.Fatalf("oversize response error = %v", err)
	}
}

func TestNewLLMProvider_Default(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "")
	t.Setenv("ENV", "")
	t.Setenv("GIN_MODE", "")
	p := NewLLMProvider(testLogger())
	if p == nil {
		t.Fatal("NewLLMProvider returned nil")
	}
	if p.Name() != "stub" {
		t.Errorf("Name() = %q, want %q", p.Name(), "stub")
	}
}

func TestNewLLMProvider_ProductionWithoutKeyIsDisabled(t *testing.T) {
	t.Setenv("GIN_MODE", "release")
	t.Setenv("ENV", "production")
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("LLM_API_KEY", "")

	p := NewLLMProvider(testLogger())
	if p.Name() != "disabled" {
		t.Fatalf("expected disabled provider, got %q", p.Name())
	}
	if _, err := p.Chat(context.Background(), &LLMRequest{}); err == nil {
		t.Fatal("disabled provider must fail closed")
	}
}
