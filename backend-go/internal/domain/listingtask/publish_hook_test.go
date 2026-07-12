package listingtask

import (
	"errors"
	"strings"
	"testing"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"go.uber.org/zap"
)

func TestPublishHookRejectsProductionBeforeReadingLegacyState(t *testing.T) {
	db := dbtest.NewDB(t)
	hook := NewPublishHook(db, nil, zap.NewNop())
	err := hook(999, ExecutionModeProduction)
	if !errors.Is(err, ErrImageReleaseAttestationRequired) || !strings.Contains(err.Error(), ImageReleaseAttestationRequiredMessage) {
		t.Fatalf("error = %v", err)
	}
}

func TestPublishHookSandboxIsNoOpSimulation(t *testing.T) {
	db := dbtest.NewDB(t)
	hook := NewPublishHook(db, nil, zap.NewNop())
	if err := hook(999, ExecutionModeSandbox); err != nil {
		t.Fatalf("sandbox no-op error = %v", err)
	}
}
