package listingtask

import (
	"errors"

	"github.com/lingmirror/backend-go/internal/domain/operationlog"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const ImageReleaseAttestationRequiredMessage = "image release attestation required; use controlled publish attempt"

var ErrImageReleaseAttestationRequired = errors.New(ImageReleaseAttestationRequiredMessage)

// PublishHook is retained only as a compatibility seam for callers that still
// construct listing tasks. It must not be an external-write seam.
type PublishHook func(taskID int64, mode ExecutionMode) error

// NewPublishHook freezes the legacy direct-publish hook. Production and
// approval-required calls fail closed before any database lookup, platform
// adapter resolution, image URL handling, or external request. Dry-run and
// sandbox modes are no-op simulations; ExecuteTask records them as mock output.
//
// The unused dependencies remain in the signature so existing wiring cannot
// accidentally replace this guard with the historical adapter implementation.
func NewPublishHook(_ *gorm.DB, _ *operationlog.Service, _ *zap.Logger) PublishHook {
	return func(_ int64, mode ExecutionMode) error {
		if mode >= ExecutionModeApprovalRequired {
			return ErrImageReleaseAttestationRequired
		}
		return nil
	}
}
