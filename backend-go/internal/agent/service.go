package agent

import (
	"github.com/lingmirror/backend-go/internal/ai"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides agent business logic. It delegates to the AI registry
// for the canonical A1-A7 / G1-G3 roster and exposes legacy-compatible
// action endpoints that map onto the unified action center.
type Service struct {
	db       *gorm.DB
	logger   *zap.Logger
	registry *ai.AgentRegistry
}

// NewService creates a new agent service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{
		db:       db,
		logger:   logger,
		registry: ai.DefaultRegistry(),
	}
}

// ponytail: registry field kept for future external access; accessor removed until needed

// AgentSummary is the legacy-compatible agent view.
type AgentSummary struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Squad          string   `json:"squad"`
	Autonomy       string   `json:"autonomy"`
	DecisionPoints []string `json:"decision_points"`
	Description    string   `json:"description"`
	ModelHint      string   `json:"model_hint,omitempty"`
	Status         string   `json:"status"`
}

// List returns all registered agents.
func (s *Service) List() []AgentSummary {
	out := make([]AgentSummary, 0, len(s.registry.Agents))
	for _, a := range s.registry.Agents {
		out = append(out, AgentSummary{
			ID:             a.ID,
			Name:           a.Name,
			Squad:          a.Squad,
			Autonomy:       a.Autonomy,
			DecisionPoints: a.DecisionPoints,
			Description:    a.Description,
			ModelHint:      a.ModelHint,
			Status:         "active",
		})
	}
	return out
}

// Get returns a single agent by ID.
func (s *Service) Get(id string) (AgentSummary, bool) {
	a, ok := s.registry.Get(id)
	if !ok {
		return AgentSummary{}, false
	}
	return AgentSummary{
		ID:             a.ID,
		Name:           a.Name,
		Squad:          a.Squad,
		Autonomy:       a.Autonomy,
		DecisionPoints: a.DecisionPoints,
		Description:    a.Description,
		ModelHint:      a.ModelHint,
		Status:         "active",
	}, true
}
