package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lingmirror/backend-go/internal/domain/approval"
	"go.uber.org/zap"
)

// WriteBack executes a platform write-back with approval gating and retry support.
func (s *Service) WriteBack(ctx context.Context, req *WriteBackRequest) (*WriteBackResult, error) {
	var acct PlatformIntegrationAccount
	if err := s.db.WithContext(ctx).First(&acct, req.AccountID).Error; err != nil {
		return nil, fmt.Errorf("write-back: account %d not found: %w", req.AccountID, err)
	}
	var plat struct{ Code string }
	if err := s.db.Table("platform").Select("code").Where("id = ?", acct.PlatformID).Scan(&plat).Error; err != nil {
		return nil, fmt.Errorf("write-back: platform lookup: %w", err)
	}
	adapter, ok := GetAdapter(plat.Code)
	if !ok {
		return nil, fmt.Errorf("write-back: no adapter for platform %s", plat.Code)
	}

	mode := ExecutionMode(acct.ExecutionMode)

	// Production mode requires approval
	if mode >= ExecutionModeApprovalRequired && s.approvalSvc != nil {
		apprReq, err := s.approvalSvc.RequireApproval(&approval.CreateApprovalInput{
			RequestType: "write_back_" + req.Action,
			Requester:   "system",
			NewValue:    fmt.Sprintf("write-back %s on account %d (%s)", req.Action, acct.ID, acct.StoreName),
			Reason:      fmt.Sprintf("production write-back of action=%s requires approval", req.Action),
			TargetType:  "integration_account", TargetID: acct.ID,
			RiskLevel: "high", EntityType: "write_back", EntityID: acct.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("write-back approval error: %w", err)
		}
		if apprReq != nil {
			return &WriteBackResult{
				ReferenceID: "", Action: req.Action, Success: false,
				Message: fmt.Sprintf("write-back requires approval (approval_id=%d)", apprReq.ID), Retryable: false,
			}, nil
		}
	}

	refID := req.ReferenceID
	if refID == "" {
		refID = uuid.New().String()
	}

	// Idempotent check
	var existing WriteBackRecord
	if req.ReferenceID != "" {
		err := s.db.WithContext(ctx).Where("reference_id = ?", req.ReferenceID).First(&existing).Error
		if err == nil && existing.Status == "success" {
			return &WriteBackResult{
				ReferenceID: refID, Action: req.Action, Success: true,
				Message: "already completed (idempotent)", Retryable: false,
			}, nil
		}
	}

	payloadStr := "{}"
	if req.Payload != nil {
		payloadStr = string(req.Payload)
	}

	// Dry-run: mock success
	if mode == ExecutionModeDryRun {
		s.logger.Info("write-back dry-run", zap.String("action", req.Action), zap.Int64("account", req.AccountID))
		s.db.WithContext(ctx).Create(&WriteBackRecord{
			ReferenceID: refID, AccountID: req.AccountID, Action: req.Action,
			Payload: payloadStr, Status: "success",
			Result: `{"dry_run":true,"message":"dry-run: no platform call was made"}`,
		})
		return &WriteBackResult{
			ReferenceID: refID, Action: req.Action, Success: true,
			Message: "dry-run: no platform call was made", Retryable: false,
			Result: map[string]interface{}{"dry_run": true},
		}, nil
	}

	// Execute via adapter
	result, err := s.executeWriteBack(ctx, adapter, req.Action, acct, req.Payload)

	// Record result
	resultStr := ""
	resultErr := ""
	if err != nil {
		resultErr = err.Error()
	} else if result != nil {
		b, _ := json.Marshal(result)
		resultStr = string(b)
	}

	status := "success"
	if err != nil {
		status = "failed"
	}

	var record WriteBackRecord
	if req.ReferenceID != "" {
		s.db.WithContext(ctx).Where("reference_id = ?", req.ReferenceID).First(&record)
	}
	record.ReferenceID = refID
	record.AccountID = req.AccountID
	record.Action = req.Action
	record.Payload = payloadStr
	record.Status = status
	record.Result = resultStr
	record.Error = resultErr
	record.RetryCount++
	record.UpdatedAt = time.Now()
	if record.ID == 0 {
		record.CreatedAt = time.Now()
		s.db.WithContext(ctx).Create(&record)
	} else {
		s.db.WithContext(ctx).Save(&record)
	}

	if err != nil {
		return nil, fmt.Errorf("write-back failed (ref=%s): %w", refID, err)
	}
	return &WriteBackResult{
		ReferenceID: refID, Action: req.Action, Success: true,
		Message: "write-back completed", Retryable: false, Result: result,
	}, nil
}

// executeWriteBack dispatches to the correct adapter method based on action.
func (s *Service) executeWriteBack(ctx context.Context, adapter PlatformAdapter, action string, acct PlatformIntegrationAccount, payload json.RawMessage) (interface{}, error) {
	switch action {
	case "sync_inventory":
		var in struct {
			SkuCode     string `json:"sku_code"`
			PlatformSKU string `json:"platform_sku"`
			Quantity    int    `json:"quantity"`
		}
		if err := json.Unmarshal(payload, &in); err != nil {
			return nil, fmt.Errorf("write-back sync_inventory: invalid payload: %w", err)
		}
		ok, err := adapter.SyncInventory(ctx, &SyncInventoryInput{
			PlatformID: acct.PlatformID, SkuCode: in.SkuCode, PlatformSKU: in.PlatformSKU, Quantity: in.Quantity,
		})
		if err != nil {
			return nil, err
		}
		return map[string]bool{"synced": ok}, nil

	case "push_tracking":
		var in struct {
			OrderSN        string `json:"order_sn"`
			TrackingNumber string `json:"tracking_number"`
			CarrierCode    string `json:"carrier_code"`
		}
		if err := json.Unmarshal(payload, &in); err != nil {
			return nil, fmt.Errorf("write-back push_tracking: invalid payload: %w", err)
		}
		ok, err := adapter.PushTracking(ctx, &PushTrackingInput{
			PlatformID: acct.PlatformID, OrderSN: in.OrderSN, TrackingNumber: in.TrackingNumber, CarrierCode: in.CarrierCode,
		})
		if err != nil {
			return nil, err
		}
		return map[string]bool{"pushed": ok}, nil

	case "sync_status":
		var in SyncStatusInput
		if err := json.Unmarshal(payload, &in); err != nil {
			return nil, fmt.Errorf("write-back sync_status: invalid payload: %w", err)
		}
		in.PlatformID = acct.PlatformID
		status, err := adapter.SyncStatus(ctx, &in)
		if err != nil {
			return nil, err
		}
		return map[string]string{"status": status}, nil

	case "validate_credentials":
		ok, err := adapter.ValidateCredentials(ctx, acct.ID)
		if err != nil {
			return nil, err
		}
		return map[string]bool{"valid": ok}, nil

	default:
		return nil, fmt.Errorf("write-back: unknown action %q", action)
	}
}

// RetryWriteBack retries a failed write-back by its reference ID.
func (s *Service) RetryWriteBack(ctx context.Context, refID string) (*WriteBackResult, error) {
	var record WriteBackRecord
	if err := s.db.WithContext(ctx).Where("reference_id = ?", refID).First(&record).Error; err != nil {
		return nil, fmt.Errorf("retry: record %s not found: %w", refID, err)
	}
	if record.Status == "success" {
		return &WriteBackResult{
			ReferenceID: refID, Action: record.Action, Success: true,
			Message: "already completed", Retryable: false,
		}, nil
	}
	req := &WriteBackRequest{
		Action: record.Action, AccountID: record.AccountID, ReferenceID: refID,
	}
	if record.Payload != "" && record.Payload != "{}" {
		req.Payload = json.RawMessage(record.Payload)
	}
	return s.WriteBack(ctx, req)
}
