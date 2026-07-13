package sourcing1688

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm/clause"
)

const (
	PrivateFailureInvalidSourceURL = "invalid_source_url"
	PrivateFailureTitleParseFailed = "title_parse_failed"
	PrivateFailureSKUParseFailed   = "sku_parse_failed"
	PrivateFailureInvalidPayload   = "invalid_payload"
	PrivateFailureNetworkError     = "network_error"
)

var privateFailureMessages = map[string]string{
	PrivateFailureInvalidSourceURL: "商品链接无法识别，请确认当前页面是1688商品详情页",
	PrivateFailureTitleParseFailed: "未能读取商品标题，请刷新页面后重试",
	PrivateFailureSKUParseFailed:   "未能完整读取SKU信息，可稍后重试或仅保存已读取字段",
	PrivateFailureInvalidPayload:   "采集内容格式不完整，请刷新商品页面后重试",
	PrivateFailureNetworkError:     "采集请求未成功送达，请稍后查看对账结果",
}

// PrivateCaptureFailure is a safe, Owner-isolated operational record. It never
// stores page HTML, raw payloads, browser credentials or arbitrary error text.
type PrivateCaptureFailure struct {
	ID               int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	OwnerID          int64     `gorm:"column:owner_id;not null;uniqueIndex:ux_private_capture_failure,priority:1" json:"owner_id"`
	RequestID        string    `gorm:"column:request_id;size:80;not null;uniqueIndex:ux_private_capture_failure,priority:2" json:"request_id"`
	SourceURL        string    `gorm:"column:source_url;type:text;not null" json:"source_url"`
	ErrorCode        string    `gorm:"column:error_code;size:80;not null;uniqueIndex:ux_private_capture_failure,priority:3" json:"error_code"`
	SafeMessage      string    `gorm:"column:safe_message;type:text;not null" json:"safe_message"`
	SchemaVersion    string    `gorm:"column:schema_version;size:40;not null" json:"schema_version"`
	ExtensionVersion string    `gorm:"column:extension_version;size:40;not null" json:"extension_version"`
	ParserVersion    string    `gorm:"column:parser_version;size:40;not null" json:"parser_version"`
	OccurredAt       time.Time `gorm:"column:occurred_at;not null" json:"occurred_at"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (PrivateCaptureFailure) TableName() string { return "sourcing_1688_private_capture_failure" }

type PrivateCaptureFailureInput struct {
	OwnerID          int64     `json:"-"`
	RequestID        string    `json:"request_id" binding:"required"`
	SourceURL        string    `json:"source_url" binding:"required"`
	ErrorCode        string    `json:"error_code" binding:"required"`
	SchemaVersion    string    `json:"schema_version" binding:"required"`
	ExtensionVersion string    `json:"extension_version" binding:"required"`
	ParserVersion    string    `json:"parser_version" binding:"required"`
	OccurredAt       time.Time `json:"occurred_at" binding:"required"`
}

type PrivateCaptureFailureResult struct {
	Status           string                 `json:"status"`
	Failure          *PrivateCaptureFailure `json:"failure"`
	IdempotentReplay bool                   `json:"idempotent_replay"`
}

func safePrivateFailureURL(raw string) string {
	value, err := canonical1688URL(raw)
	if err != nil || offerIDFromURL(value) == "" {
		return "invalid://redacted"
	}
	runes := []rune(value)
	if len(runes) > 2048 {
		value = string(runes[:2048])
	}
	return value
}

func (s *Service) RequireActiveOwnerAccount(ownerID int64) error {
	if ownerID <= 0 {
		return ErrWorkflowGate
	}
	var count int64
	err := s.db.Table("user").Where("id = ? AND status = ? AND role IN ?", ownerID, 1, []string{"owner", "admin"}).Count(&count).Error
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("%w: active Owner account required", ErrWorkflowGate)
	}
	return nil
}

func (s *Service) RecordPrivateCaptureFailure(in *PrivateCaptureFailureInput) (*PrivateCaptureFailure, bool, error) {
	if in == nil || in.OwnerID <= 0 || !strings.HasPrefix(strings.TrimSpace(in.RequestID), "collect_") ||
		len(strings.TrimSpace(in.RequestID)) > 80 || strings.TrimSpace(in.SourceURL) == "" ||
		strings.TrimSpace(in.SchemaVersion) != "sourcing1688.private.v1" ||
		!strings.HasPrefix(strings.TrimSpace(in.ExtensionVersion), "0.2.") ||
		len(strings.TrimSpace(in.ExtensionVersion)) > 40 || strings.TrimSpace(in.ParserVersion) == "" ||
		len(strings.TrimSpace(in.ParserVersion)) > 40 || in.OccurredAt.IsZero() {
		return nil, false, ErrInvalidWorkflow
	}
	message, ok := privateFailureMessages[strings.TrimSpace(in.ErrorCode)]
	if !ok {
		return nil, false, fmt.Errorf("%w: unsupported private capture failure code", ErrInvalidWorkflow)
	}
	// SKU parse failure is non-terminal by product policy: the private bookmark
	// may still be saved with sku=parse_failed while the failure is retained for
	// parser diagnosis. Identity/title failures are terminal and receive a
	// durable not_saved receipt.
	if strings.TrimSpace(in.ErrorCode) != PrivateFailureSKUParseFailed {
		requestReceipt := PrivateCollectionRequest{
			OwnerID: in.OwnerID, RequestID: strings.TrimSpace(in.RequestID), Status: PrivateRequestNotSaved,
			FailureCode: strings.TrimSpace(in.ErrorCode), SafeMessage: message, CompletedAt: pointerTime(time.Now().UTC()),
		}
		receiptInsert := s.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "owner_id"}, {Name: "request_id"}}, DoNothing: true}).Create(&requestReceipt)
		if receiptInsert.Error != nil {
			return nil, false, receiptInsert.Error
		}
		if receiptInsert.RowsAffected == 0 {
			// A delayed browser failure report must never downgrade a request whose
			// snapshot was already committed as saved.
			if err := s.db.Model(&PrivateCollectionRequest{}).
				Where("owner_id = ? AND request_id = ? AND status IN ?", in.OwnerID, strings.TrimSpace(in.RequestID), []string{PrivateRequestReceiving, PrivateRequestReconcileRequired, PrivateRequestNotSaved}).
				Updates(map[string]any{"status": PrivateRequestNotSaved, "failure_code": requestReceipt.FailureCode, "safe_message": message,
					"completed_at": time.Now().UTC(), "updated_at": time.Now().UTC()}).Error; err != nil {
				return nil, false, err
			}
		}
	}
	record := PrivateCaptureFailure{
		OwnerID: in.OwnerID, RequestID: strings.TrimSpace(in.RequestID), SourceURL: safePrivateFailureURL(in.SourceURL),
		ErrorCode: strings.TrimSpace(in.ErrorCode), SafeMessage: message,
		SchemaVersion: strings.TrimSpace(in.SchemaVersion), ExtensionVersion: strings.TrimSpace(in.ExtensionVersion),
		ParserVersion: strings.TrimSpace(in.ParserVersion), OccurredAt: in.OccurredAt.UTC(),
	}
	result := s.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "owner_id"}, {Name: "request_id"}, {Name: "error_code"}}, DoNothing: true}).Create(&record)
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected == 1 {
		return &record, false, nil
	}
	var existing PrivateCaptureFailure
	err := s.db.Where("owner_id = ? AND request_id = ? AND error_code = ?", record.OwnerID, record.RequestID, record.ErrorCode).First(&existing).Error
	return &existing, true, err
}

func (s *Service) ListPrivateCaptureFailures(ownerID int64) ([]PrivateCaptureFailure, error) {
	if ownerID <= 0 {
		return nil, ErrInvalidWorkflow
	}
	var items []PrivateCaptureFailure
	err := s.db.Where("owner_id = ?", ownerID).Order("occurred_at DESC, id DESC").Limit(200).Find(&items).Error
	return items, err
}

func privateCollectFailureInput(in *PrivateCollectInput, validationErr error) *PrivateCaptureFailureInput {
	if in == nil || validationErr == nil {
		return nil
	}
	code := PrivateFailureInvalidPayload
	if strings.TrimSpace(in.SourceURL) == "" {
		code = PrivateFailureInvalidSourceURL
	} else if _, err := canonical1688URL(in.SourceURL); err != nil {
		code = PrivateFailureInvalidSourceURL
	} else if in.Title == nil || strings.TrimSpace(*in.Title) == "" {
		code = PrivateFailureTitleParseFailed
	} else {
		var statuses map[string]string
		if len(in.FieldStatuses) > 0 && jsonUnmarshalNoError(in.FieldStatuses, &statuses) && statuses["sku"] == "parse_failed" {
			code = PrivateFailureSKUParseFailed
		}
	}
	return &PrivateCaptureFailureInput{
		OwnerID: in.OwnerID, RequestID: in.RequestID, SourceURL: in.SourceURL, ErrorCode: code,
		SchemaVersion: in.SchemaVersion, ExtensionVersion: in.ExtensionVersion, ParserVersion: in.ParserVersion,
		OccurredAt: in.ObservedAt,
	}
}

func jsonUnmarshalNoError(raw []byte, target any) bool {
	return json.Unmarshal(raw, target) == nil
}
