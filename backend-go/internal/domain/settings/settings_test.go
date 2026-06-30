package settings

import (
	"testing"
)

func TestService_GetLLMConfig_Default(t *testing.T) {
	svc := NewService(nil, nil)

	cfg := svc.GetLLMConfig()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Model != "gpt-4" {
		t.Errorf("expected default model 'gpt-4', got %s", cfg.Model)
	}
	if cfg.APIKeyPlaceholder != "sk-..." {
		t.Errorf("expected placeholder 'sk-...', got %s", cfg.APIKeyPlaceholder)
	}
	if cfg.Temperature != 0.7 {
		t.Errorf("expected default temperature 0.7, got %f", cfg.Temperature)
	}
	if cfg.MaxTokens != 4096 {
		t.Errorf("expected default max_tokens 4096, got %d", cfg.MaxTokens)
	}
}

func TestService_UpdateLLMConfig_FullUpdate(t *testing.T) {
	svc := NewService(nil, nil)

	model := "claude-3-opus"
	temp := 0.5
	maxTokens := 8192
	apiKey := "new-secret-key"

	cfg := svc.UpdateLLMConfig(&UpdateLLMInput{
		Model:       &model,
		APIKey:      &apiKey,
		Temperature: &temp,
		MaxTokens:   &maxTokens,
	})
	if cfg.Model != "claude-3-opus" {
		t.Errorf("expected model 'claude-3-opus', got %s", cfg.Model)
	}
	if cfg.Temperature != 0.5 {
		t.Errorf("expected temperature 0.5, got %f", cfg.Temperature)
	}
	if cfg.MaxTokens != 8192 {
		t.Errorf("expected max_tokens 8192, got %d", cfg.MaxTokens)
	}
}

func TestService_UpdateLLMConfig_PartialUpdate(t *testing.T) {
	svc := NewService(nil, nil)

	// Only update temperature.
	temp := 0.3
	cfg := svc.UpdateLLMConfig(&UpdateLLMInput{
		Temperature: &temp,
	})
	if cfg.Model != "gpt-4" {
		t.Errorf("expected unchanged model 'gpt-4', got %s", cfg.Model)
	}
	if cfg.Temperature != 0.3 {
		t.Errorf("expected temperature 0.3, got %f", cfg.Temperature)
	}
	if cfg.MaxTokens != 4096 {
		t.Errorf("expected unchanged max_tokens 4096, got %d", cfg.MaxTokens)
	}
}

func TestService_UpdateLLMConfig_APIKeySanitized(t *testing.T) {
	svc := NewService(nil, nil)

	apiKey := "sk-my-real-key-12345"
	svc.UpdateLLMConfig(&UpdateLLMInput{APIKey: &apiKey})

	cfg := svc.GetLLMConfig()
	// The API key should be replaced with the placeholder.
	if cfg.APIKeyPlaceholder != "sk-..." {
		t.Errorf("expected API key placeholder 'sk-...', got %s", cfg.APIKeyPlaceholder)
	}
}

func TestService_UpdateLLMConfig_Chain(t *testing.T) {
	svc := NewService(nil, nil)

	model := "gpt-4o"
	svc.UpdateLLMConfig(&UpdateLLMInput{Model: &model})

	model2 := "claude-sonnet-4"
	svc.UpdateLLMConfig(&UpdateLLMInput{Model: &model2})

	cfg := svc.GetLLMConfig()
	if cfg.Model != "claude-sonnet-4" {
		t.Errorf("expected updated model 'claude-sonnet-4', got %s", cfg.Model)
	}
}

func TestService_ConcurrentReadWrite(t *testing.T) {
	svc := NewService(nil, nil)

	// Concurrent access should be safe (via RWMutex).
	done := make(chan struct{})
	go func() {
		cfg := svc.GetLLMConfig()
		_ = cfg
		done <- struct{}{}
	}()

	temp := 0.1
	svc.UpdateLLMConfig(&UpdateLLMInput{Temperature: &temp})
	<-done

	cfg := svc.GetLLMConfig()
	if cfg.Temperature != 0.1 {
		t.Errorf("expected temperature 0.1, got %f", cfg.Temperature)
	}
}
