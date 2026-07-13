package xiaoq

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/lingmirror/backend-go/internal/ai"
	"github.com/lingmirror/backend-go/internal/dbtest"
)

const realProviderAcceptanceConfirmation = "I_ACCEPT_ONE_PAID_XIAOQ_TEST"

type acceptanceProvider struct {
	provider ai.LLMProvider
	calls    int
	tokens   int
}

func (p *acceptanceProvider) Name() string { return p.provider.Name() }

func (p *acceptanceProvider) Chat(ctx context.Context, req *ai.LLMRequest) (*ai.LLMResponse, error) {
	if p.calls >= 2 {
		return nil, errors.New("acceptance call limit reached")
	}
	p.calls++
	if req.MaxTokens == 0 || req.MaxTokens > 400 {
		req.MaxTokens = 400
	}
	resp, err := p.provider.Chat(ctx, req)
	if resp != nil {
		p.tokens += resp.TokensIn + resp.TokensOut
	}
	return resp, err
}

func (p *acceptanceProvider) ChatStream(context.Context, *ai.LLMRequest) (<-chan ai.LLMChunk, error) {
	return nil, errors.New("streaming is disabled in acceptance")
}

func TestRealProviderAcceptance(t *testing.T) {
	if os.Getenv("XIAOQ_REAL_PROVIDER_ACCEPTANCE") != realProviderAcceptanceConfirmation {
		t.Skip("paid acceptance requires explicit confirmation")
	}
	provider := ai.NewRequiredLLMProvider(dbtest.NewLogger(t))
	if provider.Name() == "disabled" {
		t.Fatal("real provider and API key are required")
	}
	limited := &acceptanceProvider{provider: provider}
	svc := newTestService(t, limited)
	result, err := svc.SendMessage(context.Background(), 42, MessageInput{
		Message:    "请读取候选市场案件事实，仅调用案件读取能力一次，然后根据工具结果说明已知、未知和下一步。",
		TargetType: TargetDemandCase, DemandCaseID: 7,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Mode != "agent_runtime_v1" || result.Provider == "" || result.Provider == "stub" || len(result.Evidence) == 0 {
		t.Fatalf("unverified runtime result: mode=%s provider=%s evidence=%d", result.Mode, result.Provider, len(result.Evidence))
	}
	if limited.calls > 2 {
		t.Fatalf("provider calls=%d, want <=2", limited.calls)
	}
	t.Logf("provider=%s model=%s calls=%d tokens=%d trace_id=%s", result.Provider, result.Model, limited.calls, limited.tokens, result.TraceID)
}
