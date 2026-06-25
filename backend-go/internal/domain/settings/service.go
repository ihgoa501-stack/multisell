package settings

import (
	"sync"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides settings business logic.
type Service struct {
	db     *gorm.DB
	logger *zap.Logger

	mu   sync.RWMutex
	llm  LLMConfig
}

// NewService creates a new settings service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{
		db:     db,
		logger: logger,
		llm: LLMConfig{
			Model:             "gpt-4",
			APIKeyPlaceholder: "sk-...",
			Temperature:       0.7,
			MaxTokens:         4096,
		},
	}
}

// GetLLMConfig returns the current LLM configuration.
func (s *Service) GetLLMConfig() *LLMConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg := s.llm
	return &cfg
}

// UpdateLLMConfig updates LLM configuration and returns the new state.
func (s *Service) UpdateLLMConfig(in *UpdateLLMInput) *LLMConfig {
	s.mu.Lock()
	defer s.mu.Unlock()

	if in.Model != nil {
		s.llm.Model = *in.Model
	}
	if in.APIKey != nil {
		s.llm.APIKeyPlaceholder = "sk-..."
	}
	if in.Temperature != nil {
		s.llm.Temperature = *in.Temperature
	}
	if in.MaxTokens != nil {
		s.llm.MaxTokens = *in.MaxTokens
	}
	cfg := s.llm
	return &cfg
}
