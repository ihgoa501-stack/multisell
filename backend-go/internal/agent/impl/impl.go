package impl

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Agent is the interface that all agent implementations must satisfy.
type Agent interface {
	ID() string
	Name() string
	Execute(ctx interface{}) (interface{}, error)
	Decide(decisionPoint string, ctx map[string]interface{}) (map[string]interface{}, float64, string, error)
}

// All returns all registered agent implementations.
func All(db *gorm.DB, logger *zap.Logger) map[string]Agent {
	return map[string]Agent{}
}
