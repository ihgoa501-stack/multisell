package productimage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lingmirror/backend-go/internal/imageservice"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrInvalidInput       = errors.New("invalid product image input")
	ErrTruthRequiresOwner = errors.New("actual truth requires separate Owner verification")
	ErrOutputHashMismatch = errors.New("image output hash mismatch")
)

type ImageService interface {
	PutBlob(context.Context, string, io.Reader) (*imageservice.PutBlobResponse, error)
	CreateJob(context.Context, imageservice.CreateJobRequest) (*imageservice.Job, error)
	GetJob(context.Context, string) (*imageservice.Job, error)
	EnqueueExecution(context.Context, string, string) (*imageservice.Attempt, error)
	ListAttempts(context.Context, string) ([]imageservice.Attempt, error)
	GetBlob(context.Context, string) ([]byte, string, error)
}

type Service struct {
	db                *gorm.DB
	logger            *zap.Logger
	image             ImageService
	executionTokenKey []byte
}

func NewService(db *gorm.DB, logger *zap.Logger, image ImageService, executionTokenKeys ...string) *Service {
	var key []byte
	if len(executionTokenKeys) > 0 {
		key = []byte(executionTokenKeys[0])
	}
	return &Service{db: db, logger: logger, image: image, executionTokenKey: key}
}

func (s *Service) CreateAsset(ctx context.Context, ownerID int64, filename, contentType string, body []byte) (*Asset, error) {
	filename, contentType = strings.TrimSpace(filename), strings.TrimSpace(contentType)
	if ownerID <= 0 || filename == "" || len(body) == 0 || s.image == nil {
		return nil, ErrInvalidInput
	}
	remote, err := s.image.PutBlob(ctx, contentType, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("upload image bytes: %w", err)
	}
	// Image Service sanitizes and re-encodes untrusted uploads before hashing;
	// its blob ID is therefore the authoritative hash of the stored bytes.
	if len(remote.BlobID) != 64 {
		return nil, errors.New("upload image bytes: Image Service returned an invalid content address")
	}
	asset := &Asset{OwnerID: ownerID, BlobID: remote.BlobID, Filename: filename, ContentType: contentType, SizeBytes: int64(len(body)), SHA256: remote.BlobID, Truth: TruthUnknown}
	if err := s.db.WithContext(ctx).Create(asset).Error; err != nil {
		return nil, fmt.Errorf("persist image asset: %w", err)
	}
	return asset, nil
}

func (s *Service) CreateTask(ctx context.Context, ownerID int64, in CreateTaskInput) (*Task, error) {
	in.IdempotencyKey, in.Operation, in.Format = strings.TrimSpace(in.IdempotencyKey), strings.TrimSpace(in.Operation), strings.ToLower(strings.TrimSpace(in.Format))
	if ownerID <= 0 || in.AssetID <= 0 || in.IdempotencyKey == "" || in.Operation != "DETERMINISTIC_RESIZE" || in.Width <= 0 || in.Height <= 0 || (in.Format != "png" && in.Format != "jpeg") || s.image == nil {
		return nil, ErrInvalidInput
	}
	var asset Asset
	if err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", in.AssetID, ownerID).First(&asset).Error; err != nil {
		return nil, err
	}
	manifestBytes, _ := json.Marshal(struct {
		OwnerID   int64  `json:"owner_id"`
		AssetID   int64  `json:"asset_id"`
		BlobID    string `json:"blob_id"`
		Operation string `json:"operation"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		Format    string `json:"format"`
	}{ownerID, asset.ID, asset.BlobID, in.Operation, in.Width, in.Height, in.Format})
	digest := sha256.Sum256(manifestBytes)
	manifest := hex.EncodeToString(digest[:])
	var existing Task
	if err := s.db.WithContext(ctx).Where("owner_id = ? AND idempotency_key = ?", ownerID, in.IdempotencyKey).First(&existing).Error; err == nil {
		if existing.ManifestHash != manifest {
			return nil, &ConflictError{Code: "IDEMPOTENCY_CONFLICT"}
		}
		return &existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	remote, err := s.image.CreateJob(ctx, imageservice.CreateJobRequest{OwnerID: ownerID, IdempotencyKey: in.IdempotencyKey, ManifestHash: manifest, Operation: in.Operation, InputBlobID: asset.BlobID, Width: in.Width, Height: in.Height, Format: in.Format})
	if err != nil {
		return nil, fmt.Errorf("create image job: %w", err)
	}
	if remote.OwnerID != ownerID || remote.ManifestHash != manifest {
		return nil, errors.New("create image job: Image Service returned mismatched ownership or manifest")
	}
	task := &Task{OwnerID: ownerID, AssetID: asset.ID, ImageServiceJobID: remote.ID, IdempotencyKey: in.IdempotencyKey, ManifestHash: manifest, Operation: in.Operation, Processor: "deterministic", Version: 1, Width: in.Width, Height: in.Height, Format: in.Format, Status: remote.Status, ErrorCode: remote.ErrorCode}
	if err := s.db.WithContext(ctx).Create(task).Error; err != nil {
		return nil, fmt.Errorf("persist image task: %w", err)
	}
	return task, nil
}

type ConflictError struct{ Code string }

func (e *ConflictError) Error() string { return e.Code }

func (s *Service) ListTasks(ctx context.Context, ownerID int64, page, size int) ([]Task, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	q := s.db.WithContext(ctx).Model(&Task{}).Where("owner_id = ?", ownerID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var tasks []Task
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&tasks).Error
	if err == nil && s.image != nil {
		for i := range tasks {
			remote, remoteErr := s.image.GetJob(ctx, tasks[i].ImageServiceJobID)
			if remoteErr != nil || remote.OwnerID != ownerID || remote.ManifestHash != tasks[i].ManifestHash {
				continue
			}
			tasks[i].Status, tasks[i].ErrorCode, tasks[i].OutputBlobID = remote.Status, remote.ErrorCode, remote.OutputBlobID
			_ = s.db.WithContext(ctx).Model(&Task{}).Where("id = ? AND owner_id = ?", tasks[i].ID, ownerID).Updates(map[string]any{"status": tasks[i].Status, "error_code": tasks[i].ErrorCode, "output_blob_id": tasks[i].OutputBlobID}).Error
		}
	}
	if tasks == nil {
		tasks = []Task{}
	}
	for i := range tasks {
		if tasks[i].OutputBlobID != "" {
			tasks[i].OutputURL = fmt.Sprintf("/api/v1/product-images/tasks/%d/output/content", tasks[i].ID)
		}
	}
	return tasks, total, err
}

func (s *Service) GetTask(ctx context.Context, ownerID, id int64) (*Task, error) {
	var task Task
	if err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", id, ownerID).First(&task).Error; err != nil {
		return nil, err
	}
	if task.ImageServiceJobID != "" && s.image != nil {
		if remote, err := s.image.GetJob(ctx, task.ImageServiceJobID); err == nil {
			if remote.ID != task.ImageServiceJobID || remote.OwnerID != ownerID || remote.ManifestHash != task.ManifestHash || remote.Operation != task.Operation || (remote.Processor != "" && remote.Processor != task.Processor) {
				return nil, errors.New("Image Service returned mismatched job identity")
			}
			if task.Processor != "" && task.Processor != "deterministic" && (remote.LingMirrorTaskID != strconv.FormatInt(task.ID, 10) || remote.LingMirrorTaskVersion != task.Version) {
				return nil, errors.New("Image Service returned mismatched LingMirror task identity")
			}
			if task.Processor != "" && task.Processor != "deterministic" && (remote.LingMirrorTaskID != strconv.FormatInt(task.ID, 10) || remote.LingMirrorTaskVersion != task.Version || remote.Operation != task.Operation || remote.Processor != task.Processor) {
				return nil, errors.New("Image Service returned mismatched paid execution target")
			}
			task.Status, task.ErrorCode, task.OutputBlobID = remote.Status, remote.ErrorCode, remote.OutputBlobID
			_ = s.db.WithContext(ctx).Model(&Task{}).Where("id = ? AND owner_id = ?", task.ID, ownerID).Updates(map[string]any{"status": task.Status, "error_code": task.ErrorCode, "output_blob_id": task.OutputBlobID}).Error
		} else {
			s.logger.Debug("image service job refresh failed", zap.Error(err), zap.Int64("task_id", task.ID))
		}
	}
	if task.OutputBlobID != "" {
		task.OutputURL = fmt.Sprintf("/api/v1/product-images/tasks/%d/output/content", task.ID)
	}
	return &task, nil
}

func (s *Service) Execute(ctx context.Context, ownerID, id int64, key string) (*imageservice.Attempt, error) {
	task, err := s.GetTask(ctx, ownerID, id)
	if err != nil {
		return nil, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, ErrInvalidInput
	}
	if task.Processor == "" || task.Processor == "deterministic" {
		return s.image.EnqueueExecution(ctx, task.ImageServiceJobID, key)
	}
	if len(s.executionTokenKey) < 32 {
		return nil, &ConflictError{Code: "APPROVAL_REQUIRED"}
	}
	var approval ExecutionApproval
	if err := s.db.WithContext(ctx).Where("owner_id = ? AND task_id = ? AND task_version = ? AND manifest_hash = ? AND operation = ? AND processor = ? AND consumed_at IS NULL AND expires_at > ?", ownerID, task.ID, task.Version, task.ManifestHash, task.Operation, task.Processor, time.Now().UTC()).Order("id DESC").First(&approval).Error; err != nil {
		return nil, &ConflictError{Code: "APPROVAL_REQUIRED"}
	}
	var costCount int64
	if err := s.db.WithContext(ctx).Model(&CostEntry{}).Where("owner_id = ? AND task_id = ? AND expected_task_version = ? AND provider = ? AND billing_status <> ?", ownerID, task.ID, task.Version, task.Processor, "unknown").Count(&costCount).Error; err != nil {
		return nil, err
	}
	if costCount == 0 {
		return nil, &ConflictError{Code: "BUDGET_COST_REQUIRED"}
	}
	token, err := s.signExecutionToken(task, &approval, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	authorized, ok := s.image.(interface {
		EnqueueAuthorizedExecution(context.Context, string, string, string) (*imageservice.Attempt, error)
	})
	if !ok {
		return nil, errors.New("Image Service client does not support authorized execution")
	}
	attempt, err := authorized.EnqueueAuthorizedExecution(ctx, task.ImageServiceJobID, key, token)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	result := s.db.WithContext(ctx).Model(&ExecutionApproval{}).Where("id = ? AND owner_id = ? AND consumed_at IS NULL", approval.ID, ownerID).Update("consumed_at", now)
	if result.Error != nil || result.RowsAffected != 1 {
		return nil, errors.New("persist approval consumption")
	}
	return attempt, nil
}

var moneyPattern = regexp.MustCompile(`^[1-9][0-9]{0,9}(\.[0-9]{1,4})?$|^0\.(?:[0-9]{0,3}[1-9])$`)

func (s *Service) ApproveExecution(ctx context.Context, ownerID, taskID int64, in ApprovalInput) (*ExecutionApproval, error) {
	if ownerID <= 0 || len(s.executionTokenKey) < 32 || !moneyPattern.MatchString(strings.TrimSpace(in.MaxCost)) {
		return nil, ErrInvalidInput
	}
	in.Processor, in.Currency = strings.TrimSpace(in.Processor), strings.ToUpper(strings.TrimSpace(in.Currency))
	if in.Processor == "" || in.Processor == "deterministic" || !allowedExecutionCurrency(in.Currency) || in.ExpectedVersion <= 0 {
		return nil, ErrInvalidInput
	}
	task, err := s.GetTask(ctx, ownerID, taskID)
	if err != nil {
		return nil, err
	}
	if task.Version != in.ExpectedVersion || task.Processor != in.Processor || task.Status != "QUEUED" {
		return nil, &ConflictError{Code: "VERSION_CONFLICT"}
	}
	executionID, err := randomHex(16)
	if err != nil {
		return nil, err
	}
	nonce, err := randomHex(32)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	approval := &ExecutionApproval{ExecutionID: executionID, OwnerID: ownerID, TaskID: task.ID, TaskVersion: task.Version, ManifestHash: task.ManifestHash, Operation: task.Operation, Processor: task.Processor, MaxCost: strings.TrimSpace(in.MaxCost), Currency: in.Currency, Nonce: nonce, ApprovedAt: now, ExpiresAt: now.Add(3 * time.Minute)}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(approval).Error; err != nil {
			return err
		}
		costInput := CostEntryInput{Kind: "estimated", Category: "provider", Provider: task.Processor, Amount: approval.MaxCost, Currency: approval.Currency, ExchangeRate: "1", ExchangeRateSource: "same_currency_budget_cap", ObservedAt: now, BillingStatus: "estimated", IdempotencyKey: "execution-approval:" + executionID, ExpectedVersion: task.Version}
		cost := &CostEntry{OwnerID: ownerID, TaskID: task.ID, Kind: costInput.Kind, Category: costInput.Category, Provider: costInput.Provider, Amount: costInput.Amount, Currency: costInput.Currency, ExchangeRate: costInput.ExchangeRate, ExchangeRateSource: costInput.ExchangeRateSource, ObservedAt: now, BillingStatus: costInput.BillingStatus, IdempotencyKey: costInput.IdempotencyKey, RequestHash: requestHash(costInput), ExpectedTaskVersion: task.Version}
		return tx.Create(cost).Error
	}); err != nil {
		return nil, err
	}
	return approval, nil
}

func allowedExecutionCurrency(v string) bool {
	switch v {
	case "USD", "EUR", "CNY", "GBP", "JPY":
		return true
	}
	return false
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Service) signExecutionToken(task *Task, approval *ExecutionApproval, now time.Time) (string, error) {
	if task == nil || approval == nil || len(s.executionTokenKey) < 32 {
		return "", ErrInvalidInput
	}
	nbf := now.Unix()
	exp := approval.ExpiresAt.Unix()
	if exp <= nbf || exp-nbf > 300 {
		return "", ErrInvalidInput
	}
	claims := map[string]any{"approval_execution_id": approval.ExecutionID, "task_id": strconv.FormatInt(task.ID, 10), "task_version": task.Version, "owner_id": task.OwnerID, "job_id": task.ImageServiceJobID, "manifest_hash": task.ManifestHash, "operation": task.Operation, "processor": task.Processor, "max_cost": approval.MaxCost, "currency": approval.Currency, "nonce": approval.Nonce, "iat": now.Unix(), "nbf": nbf, "exp": exp, "aud": "lingmirror-image-service-execution"}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(body)
	input := header + "." + payload
	mac := hmac.New(sha256.New, s.executionTokenKey)
	_, _ = mac.Write([]byte(input))
	return input + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (s *Service) Attempts(ctx context.Context, ownerID, id int64) ([]imageservice.Attempt, error) {
	task, err := s.GetTask(ctx, ownerID, id)
	if err != nil {
		return nil, err
	}
	return s.image.ListAttempts(ctx, task.ImageServiceJobID)
}

func (s *Service) OutputContent(ctx context.Context, ownerID, id int64) ([]byte, string, error) {
	task, err := s.GetTask(ctx, ownerID, id)
	if err != nil {
		return nil, "", err
	}
	if task.Status != "READY" || task.OutputBlobID == "" {
		return nil, "", ErrInvalidInput
	}
	body, mediaType, err := s.image.GetBlob(ctx, task.OutputBlobID)
	if err != nil {
		return nil, "", err
	}
	digest := sha256.Sum256(body)
	if hex.EncodeToString(digest[:]) != task.OutputBlobID {
		s.logger.Error("image service output failed content-address verification", zap.Int64("task_id", task.ID), zap.String("expected_blob_id", task.OutputBlobID))
		return nil, "", ErrOutputHashMismatch
	}
	return body, mediaType, nil
}

func (s *Service) CreateReview(ctx context.Context, review *Review) error {
	if review == nil || review.OwnerID <= 0 || review.TaskID <= 0 || strings.TrimSpace(review.Decision) == "" {
		return ErrInvalidInput
	}
	if review.Truth == TruthActual {
		return ErrTruthRequiresOwner
	}
	if review.Truth == "" {
		review.Truth = TruthUnknown
	}
	return s.db.WithContext(ctx).Create(review).Error
}
