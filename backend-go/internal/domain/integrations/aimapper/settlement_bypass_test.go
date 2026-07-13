package aimapper

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestPipelineRejectsDirectSettlementFactPersistence(t *testing.T) {
	p := &Pipeline{logger: zap.NewNop()}
	err := p.persist(context.Background(), "settlement_item", map[string]interface{}{"transaction_id": "forged"})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("legacy AI mapper settlement bypass was not frozen: %v", err)
	}
}
