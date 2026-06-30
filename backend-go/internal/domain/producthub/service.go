package producthub

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/lingmirror/backend-go/internal/domain/sku"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service provides product hub business logic.
type Service struct {
	db      *gorm.DB
	logger  *zap.Logger
	version *VersionService
}

// NewService creates a new product hub service.
func NewService(db *gorm.DB, logger *zap.Logger) *Service {
	return &Service{
		db:      db,
		logger:  logger,
		version: NewVersionService(db, logger),
	}
}

// RecordDecision records an agent decision trace and automatically creates
// a ProductVersion snapshot of the product's key fields.
func (s *Service) RecordDecision(productID int64, in *DecisionRecordInput) (*DecisionRecord, error) {
	// Capture current product state as snapshot.
	var p sku.Product
	if err := s.db.First(&p, productID).Error; err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	snapshot, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal product snapshot: %w", err)
	}

	// VersionData captures the decision-specific metadata.
	versionData, _ := json.Marshal(map[string]interface{}{
		"action":     in.Action,
		"reasoning":  in.Reasoning,
		"confidence": in.Confidence,
	})

	reason := fmt.Sprintf("agent:%s action:%s", in.AgentID, in.Action)

	// Create the ProductVersion entry.
	if _, err := s.version.CreateVersion(productID, in.AgentID, reason, snapshot, versionData); err != nil {
		s.logger.Warn("failed to create product version snapshot",
			zap.Int64("product_id", productID),
			zap.String("agent_id", in.AgentID),
			zap.Error(err),
		)
		// Don't fail the whole request just because versioning failed.
	}

	record := &DecisionRecord{
		AgentID:    in.AgentID,
		Action:     in.Action,
		Reasoning:  in.Reasoning,
		Confidence: in.Confidence,
		CreatedAt:  time.Now().Format(time.RFC3339),
	}
	return record, nil
}

// GetDecision returns the service's version service for external use.
func (s *Service) GetDecision() *VersionService {
	return s.version
}
