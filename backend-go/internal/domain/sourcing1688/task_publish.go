package sourcing1688

import (
	"context"
	"fmt"
)

func (s *Service) RequestTaskPublish(sourceID, taskLinkID int64, in *PublishRequestInput) (*PublishAttempt, error) {
	if in == nil || taskLinkID <= 0 {
		return nil, ErrInvalidWorkflow
	}
	in.TaskLinkID = taskLinkID
	return s.RequestPublish(sourceID, in)
}

func (s *Service) DecideTaskPublish(sourceID, taskLinkID, attemptID int64, in *PublishDecisionInput) (*PublishAttempt, error) {
	if in == nil || taskLinkID <= 0 {
		return nil, ErrInvalidWorkflow
	}
	in.TaskLinkID = taskLinkID
	return s.DecidePublish(sourceID, attemptID, in)
}

func (s *Service) ExecuteTaskPublish(ctx context.Context, sourceID, taskLinkID, attemptID, ownerID int64) (*PublishAttempt, error) {
	var attempt PublishAttempt
	if err := s.db.Where("id = ? AND sourcing_product_id = ? AND task_link_id = ?", attemptID, sourceID, taskLinkID).First(&attempt).Error; err != nil {
		return nil, fmt.Errorf("%w: publish attempt belongs to another task", ErrWorkflowGate)
	}
	return s.executePublishForTask(ctx, sourceID, taskLinkID, attemptID, ownerID)
}

func (s *Service) ReconcileTaskPublish(ctx context.Context, sourceID, taskLinkID, attemptID int64, in *PublishReconcileInput) (*PublishAttempt, error) {
	if in == nil || taskLinkID <= 0 {
		return nil, ErrInvalidWorkflow
	}
	in.TaskLinkID = taskLinkID
	return s.reconcilePublishForTask(ctx, sourceID, taskLinkID, attemptID, in)
}
