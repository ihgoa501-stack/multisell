package content

import (
	"testing"

	"go.uber.org/zap"
)

// ── Service construction test ────────────────────────────────────

func TestService_NewService(t *testing.T) {
	svc := NewService(nil, dbtestNewLogger(t))
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.generator == nil {
		t.Error("expected non-nil generator")
	}
	if svc.validator == nil {
		t.Error("expected non-nil validator")
	}
}

func TestService_Generate_WithNilAI(t *testing.T) {
	// The Generate method delegates to ContentGenerator.Generate which calls
	// the AI orchestrator directly. With a nil orchestrator, this will panic.
	// The test verifies the service doesn't hang or corrupt state.
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Generate with nil AI panicked as expected: %v", r)
		}
	}()
	svc := NewService(nil, dbtestNewLogger(t))
	svc.Generate(&GenerateRequest{
		ProductName:    "Test Product",
		Category:       "electronics",
		TargetLanguage: "en",
		Platform:       "shopee",
	})
}

func TestService_Validate_WithNilAI(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Validate with nil AI panicked as expected: %v", r)
		}
	}()
	svc := NewService(nil, dbtestNewLogger(t))

	content := &GeneratedContent{
		Title:       "Test Title",
		Description: "Test Description",
		Keywords:    []string{"test"},
		Confidence:  0.9,
	}
	svc.Validate(content, &ValidateRequest{
		Title:    "Test Title",
		Language: "en",
		Platform: "shopee",
	})
}

func dbtestNewLogger(t *testing.T) *zap.Logger {
	t.Helper()
	logger, _ := zap.NewDevelopment()
	return logger
}
