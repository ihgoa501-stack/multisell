package content

import (
	"github.com/lingmirror/backend-go/internal/ai"
	"go.uber.org/zap"
)

// Service provides content generation and validation business logic.
type Service struct {
	generator *ContentGenerator
	validator *ContentValidator
	logger    *zap.Logger
}

// NewService creates a new content service.
func NewService(aiOrch *ai.Orchestrator, logger *zap.Logger) *Service {
	return &Service{
		generator: NewContentGenerator(aiOrch, logger.Named("content.generator")),
		validator: NewContentValidator(aiOrch, logger.Named("content.validator")),
		logger:    logger,
	}
}

// Generate creates product content using AI orchestration.
func (s *Service) Generate(in *GenerateRequest) (*GeneratedContent, error) {
	return s.generator.Generate(in)
}

// Validate performs LLM-as-judge validation on generated content.
func (s *Service) Validate(content *GeneratedContent, in *ValidateRequest) (*ContentReview, error) {
	return s.validator.Validate(content, in)
}
