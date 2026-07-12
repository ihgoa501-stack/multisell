package ai

import (
	"context"
	"strings"
	"testing"
)

func TestStubProvider_Name(t *testing.T) {
	p := &StubProvider{logger: testLogger()}
	if got := p.Name(); got != "stub" {
		t.Errorf("Name() = %q, want %q", got, "stub")
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
