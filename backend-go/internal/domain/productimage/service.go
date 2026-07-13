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
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInvalidInput       = errors.New("invalid product image input")
	ErrTruthRequiresOwner = errors.New("actual truth requires separate Owner verification")
	ErrOutputHashMismatch = errors.New("image output hash mismatch")
)

// ProcessorCapability is the sanitized LingMirror-side view exposed to Owner
// APIs and MCP. It never contains provider credentials or raw provider JSON.
type ProcessorCapability struct {
	Code                string   `json:"code"`
	Name                string   `json:"name"`
	Configured          bool     `json:"configured"`
	Availability        string   `json:"availability"`
	Operations          []string `json:"operations"`
	Paid                bool     `json:"paid"`
	ReconcileSafe       bool     `json:"reconcile_safe"`
	Reason              string   `json:"reason,omitempty"`
	SafetyLevel         string   `json:"safety_level"`
	ProviderEnvironment string   `json:"provider_environment,omitempty"`
	Region              string   `json:"region,omitempty"`
	Watermarked         bool     `json:"watermarked"`
	NonPublishable      bool     `json:"non_publishable"`
	QuotaAvailable      bool     `json:"quota_available"`
	QuotaRemaining      int64    `json:"quota_remaining,omitempty"`
}

type EstimateResult struct {
	TaskID        int64  `json:"task_id"`
	Processor     string `json:"processor"`
	Operation     string `json:"operation"`
	Availability  string `json:"availability"`
	Amount        string `json:"amount,omitempty"`
	Currency      string `json:"currency,omitempty"`
	Exact         bool   `json:"exact"`
	CreatesIntent bool   `json:"creates_approval_or_paid_intent"`
	Reason        string `json:"reason,omitempty"`
}

type OutputReference struct {
	Ordinal        int    `json:"ordinal"`
	BlobSHA256     string `json:"blob_sha256"`
	MediaReference string `json:"media_reference"`
	Status         string `json:"status"`
	Sandbox        bool   `json:"sandbox"`
	Watermarked    bool   `json:"watermarked"`
	NonPublishable bool   `json:"non_publishable"`
}

const (
	photoroomProcessor   = "photoroom"
	photoroomRegion      = "us"
	photoroomEnvironment = "sandbox"
	openAIProcessor      = "openai"
	openAIOperation      = "OPENAI_IMAGE_EDIT"
	openAIRegion         = "us"
	openAIEnvironment    = "production"
)

var photoroomOperations = map[string]struct{}{
	"PHOTOROOM_REMOVE_BACKGROUND_SANDBOX": {},
	"PHOTOROOM_WHITE_BACKGROUND_SANDBOX":  {},
	"PHOTOROOM_AI_SHADOW_SANDBOX":         {},
}

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

func (s *Service) processorAvailable(code string) bool {
	if s == nil || s.image == nil {
		return false
	}
	if provider, ok := s.image.(interface{ ProcessorAvailable(string) bool }); ok {
		return provider.ProcessorAvailable(code)
	}
	return code == "" || code == "deterministic"
}

func (s *Service) processorCapability(ctx context.Context, code string) (imageservice.ProcessorCapability, bool) {
	code = cleanScope(code)
	if s == nil || s.image == nil {
		return imageservice.ProcessorCapability{}, false
	}
	if client, ok := s.image.(*imageservice.Client); ok && client == nil {
		return imageservice.ProcessorCapability{}, false
	}
	if reader, ok := s.image.(interface {
		ListProcessors(context.Context) ([]imageservice.ProcessorCapability, error)
	}); ok {
		items, err := reader.ListProcessors(ctx)
		if err != nil {
			return imageservice.ProcessorCapability{}, false
		}
		for _, item := range items {
			if cleanScope(item.Code) == code {
				return item, item.Available
			}
		}
		return imageservice.ProcessorCapability{}, false
	}
	return imageservice.ProcessorCapability{Code: code, Available: s.processorAvailable(code)}, s.processorAvailable(code)
}

// ListCapabilities is the sole capability source for HTTP and MCP handlers.
// External providers remain explicitly unavailable until a verified adapter is
// registered; the current private client only proves deterministic processing.
func (s *Service) ListCapabilities(page, size int) ([]ProcessorCapability, int) {
	return s.ListCapabilitiesContext(context.Background(), page, size)
}

func (s *Service) ListCapabilitiesContext(ctx context.Context, page, size int) ([]ProcessorCapability, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	deterministicAvailability := "unavailable"
	deterministicReason := "Image Service is not configured"
	if _, ok := s.processorCapability(ctx, "deterministic"); ok {
		deterministicAvailability = "available"
		deterministicReason = ""
	}
	all := []ProcessorCapability{
		{Code: "deterministic", Name: "凌镜标准处理", Configured: deterministicAvailability == "available", Availability: deterministicAvailability, Operations: []string{"DETERMINISTIC_RESIZE"}, ReconcileSafe: false, Reason: deterministicReason, SafetyLevel: "production_safe", QuotaAvailable: deterministicAvailability == "available"},
		{Code: "photoroom", Name: "Photoroom", Availability: "unavailable", Operations: []string{"PHOTOROOM_REMOVE_BACKGROUND_SANDBOX", "PHOTOROOM_WHITE_BACKGROUND_SANDBOX", "PHOTOROOM_AI_SHADOW_SANDBOX"}, Paid: true, Reason: "Photoroom sandbox is not explicitly available in Image Service", SafetyLevel: "sandbox_only", ProviderEnvironment: photoroomEnvironment, Watermarked: true, NonPublishable: true},
		{Code: "adobe", Name: "Adobe Firefly", Availability: "unavailable", Operations: []string{}, Paid: true, Reason: "Provider adapter is not configured"},
		{Code: openAIProcessor, Name: "OpenAI Images", Availability: "unavailable", Operations: []string{openAIOperation}, Paid: true, Reason: "Provider adapter is not configured", SafetyLevel: "production_paid", ProviderEnvironment: openAIEnvironment},
	}
	for i := range all {
		if remote, ok := s.processorCapability(ctx, all[i].Code); all[i].Code != "deterministic" && ok {
			all[i].Configured = true
			all[i].Availability = "available"
			all[i].Reason = ""
			all[i].SafetyLevel, all[i].ProviderEnvironment, all[i].Region, all[i].Watermarked, all[i].NonPublishable, all[i].QuotaAvailable, all[i].QuotaRemaining = remote.SafetyLevel, remote.ProviderEnvironment, remote.Region, remote.Watermarked, remote.NonPublishable, remote.QuotaAvailable, remote.QuotaRemaining
		}
	}
	total := len(all)
	start := (page - 1) * size
	if start >= total {
		return []ProcessorCapability{}, total
	}
	end := start + size
	if end > total {
		end = total
	}
	return all[start:end], total
}

// EstimateTask is read-only. In particular it never creates an approval,
// budget reservation, execution token, attempt, or provider request.
func (s *Service) EstimateTask(ctx context.Context, ownerID, taskID int64) (*EstimateResult, error) {
	task, err := s.GetTask(ctx, ownerID, taskID)
	if err != nil {
		return nil, err
	}
	if task.Processor == "" || task.Processor == "deterministic" {
		return &EstimateResult{TaskID: task.ID, Processor: "deterministic", Operation: task.Operation, Availability: "available", Amount: "0", Currency: "USD", Exact: true, CreatesIntent: false}, nil
	}
	if _, ok := s.processorCapability(ctx, task.Processor); ok {
		return &EstimateResult{TaskID: task.ID, Processor: task.Processor, Operation: task.Operation, Availability: "available", Exact: false, CreatesIntent: false, Reason: "A paid estimate is not yet available; no approval or paid intent was created"}, nil
	}
	return &EstimateResult{TaskID: task.ID, Processor: task.Processor, Operation: task.Operation, Availability: "unavailable", Exact: false, CreatesIntent: false, Reason: "Provider adapter is not configured; no approval or paid intent was created"}, nil
}

// ListOutputs returns only LingMirror-controlled references. It never returns
// provider URLs, arbitrary fetch URLs, filesystem paths, or image bytes.
func (s *Service) ListOutputs(ctx context.Context, ownerID, taskID int64, page, size int) ([]OutputReference, int, error) {
	task, err := s.GetTask(ctx, ownerID, taskID)
	if err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	if task.OutputBlobID == "" {
		return []OutputReference{}, 0, nil
	}
	if page > 1 {
		return []OutputReference{}, 1, nil
	}
	return []OutputReference{{Ordinal: 1, BlobSHA256: task.OutputBlobID, MediaReference: fmt.Sprintf("/api/v1/product-images/tasks/%d/output/content", task.ID), Status: task.Status, Sandbox: task.Sandbox, Watermarked: task.Watermarked, NonPublishable: task.NonPublishable}}, 1, nil
}

// ReconcileTask delegates no gate to MCP. The currently configured
// deterministic processor has no remote provider state to reconcile; external
// providers are not configured, so both cases fail closed without creating a
// paid intent or execution attempt.
func (s *Service) ReconcileTask(ctx context.Context, ownerID, taskID int64) (*Task, error) {
	task, err := s.GetTask(ctx, ownerID, taskID)
	if err != nil {
		return nil, err
	}
	if task.Processor == "" || task.Processor == "deterministic" {
		return nil, &ConflictError{Code: "RECONCILE_NOT_SUPPORTED"}
	}
	if _, ok := s.processorCapability(ctx, task.Processor); !ok {
		return nil, &ConflictError{Code: "PROVIDER_UNAVAILABLE"}
	}
	return nil, &ConflictError{Code: "RECONCILE_NOT_SUPPORTED"}
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
	asset := &Asset{OwnerID: ownerID, BlobID: remote.BlobID, Filename: filename, ContentType: contentType, SizeBytes: int64(len(body)), SHA256: remote.BlobID, Truth: TruthUnknown, SourceKind: "upload", ChannelRestriction: "*"}
	if err := s.db.WithContext(ctx).Create(asset).Error; err != nil {
		return nil, fmt.Errorf("persist image asset: %w", err)
	}
	return asset, nil
}

type canonicalSKU struct {
	ID int64 `gorm:"column:id;primaryKey"`
}

func (canonicalSKU) TableName() string { return "sku" }

func normalizeRecipe(ctx context.Context, db *gorm.DB, ownerID int64, source Asset, in *CreateTaskInput) ([]byte, string, error) {
	if in == nil || in.SKUID <= 0 || strings.TrimSpace(in.RecipeKey) == "" || len(in.RecipeKey) > 100 || in.RecipeVersion <= 0 || in.CandidateRound <= 0 {
		return nil, "", ErrInvalidInput
	}
	in.RecipeKey = strings.TrimSpace(in.RecipeKey)
	in.Recipe.SceneStructure = strings.TrimSpace(in.Recipe.SceneStructure)
	in.Recipe.Prompt = strings.TrimSpace(in.Recipe.Prompt)
	in.Recipe.NegativePrompt = strings.TrimSpace(in.Recipe.NegativePrompt)
	in.Recipe.Model = strings.TrimSpace(in.Recipe.Model)
	in.Recipe.ModelVersion = strings.TrimSpace(in.Recipe.ModelVersion)
	if in.Recipe.SceneStructure == "" || len(in.Recipe.SceneStructure) > 2000 || in.Recipe.Model == "" || len(in.Recipe.Model) > 100 || in.Recipe.ModelVersion == "" || len(in.Recipe.ModelVersion) > 100 || len(in.Recipe.Prompt) > 4000 || len(in.Recipe.NegativePrompt) > 2000 || len(in.Recipe.MustNotChange) == 0 || len(in.Recipe.MustNotChange) > 30 {
		return nil, "", ErrInvalidInput
	}
	for i := range in.Recipe.MustNotChange {
		in.Recipe.MustNotChange[i] = strings.TrimSpace(in.Recipe.MustNotChange[i])
		if in.Recipe.MustNotChange[i] == "" || len(in.Recipe.MustNotChange[i]) > 200 {
			return nil, "", ErrInvalidInput
		}
	}
	if len(in.Recipe.Parameters) == 0 {
		in.Recipe.Parameters = json.RawMessage(`{}`)
	}
	var parameters map[string]any
	if len(in.Recipe.Parameters) > 16<<10 || json.Unmarshal(in.Recipe.Parameters, &parameters) != nil || parameters == nil {
		return nil, "", ErrInvalidInput
	}
	var skuCount int64
	if err := db.WithContext(ctx).Model(&canonicalSKU{}).Where("id = ?", in.SKUID).Count(&skuCount).Error; err != nil || skuCount != 1 {
		return nil, "", ErrInvalidInput
	}
	assetIDs := append([]int64(nil), in.Recipe.ReferenceAssetIDs...)
	if len(assetIDs) > 10 {
		return nil, "", ErrInvalidInput
	}
	if in.Recipe.MaskAssetID != nil {
		assetIDs = append(assetIDs, *in.Recipe.MaskAssetID)
	}
	if len(assetIDs) > 0 {
		var count int64
		if err := db.WithContext(ctx).Model(&Asset{}).Where("owner_id = ? AND id IN ?", ownerID, assetIDs).Count(&count).Error; err != nil || count != int64(len(assetIDs)) {
			return nil, "", ErrInvalidInput
		}
	}
	if in.ParentTaskID == nil {
		if in.RecipeVersion != 1 || in.CandidateRound != 1 {
			return nil, "", ErrInvalidInput
		}
	} else {
		var parent Task
		if err := db.WithContext(ctx).Where("id = ? AND owner_id = ?", *in.ParentTaskID, ownerID).First(&parent).Error; err != nil || parent.SKUID != in.SKUID || parent.RecipeKey != in.RecipeKey || in.RecipeVersion != parent.RecipeVersion+1 || in.CandidateRound != parent.CandidateRound+1 {
			return nil, "", ErrInvalidInput
		}
	}
	if source.OwnerID != ownerID {
		return nil, "", ErrInvalidInput
	}
	b, err := json.Marshal(in.Recipe)
	if err != nil {
		return nil, "", ErrInvalidInput
	}
	digest := sha256.Sum256(b)
	return b, hex.EncodeToString(digest[:]), nil
}

func (s *Service) CreateTask(ctx context.Context, ownerID int64, in CreateTaskInput) (*Task, error) {
	in.IdempotencyKey, in.Operation, in.Format = strings.TrimSpace(in.IdempotencyKey), strings.ToUpper(strings.TrimSpace(in.Operation)), strings.ToLower(strings.TrimSpace(in.Format))
	in.Processor, in.Purpose, in.Channel, in.Region = cleanScope(in.Processor), cleanScope(in.Purpose), cleanScope(in.Channel), cleanScope(in.Region)
	isDeterministic := in.Processor == "deterministic" && in.Operation == "DETERMINISTIC_RESIZE" && in.Region == "local" && (in.Format == "png" || in.Format == "jpeg") && strings.TrimSpace(in.MaxCost) == "" && strings.TrimSpace(in.Currency) == ""
	_, isPhotoroomOperation := photoroomOperations[in.Operation]
	isPhotoroom := in.Processor == photoroomProcessor && isPhotoroomOperation && in.Region == photoroomRegion && in.Format == "png" && strings.TrimSpace(in.MaxCost) == "0" && strings.ToUpper(strings.TrimSpace(in.Currency)) == "USD"
	openAICost, openAICostOK := strictMoney(in.MaxCost)
	isOpenAI := in.Processor == openAIProcessor && in.Operation == openAIOperation && in.Region == openAIRegion && in.Format == "png" && strings.ToUpper(strings.TrimSpace(in.Currency)) == "USD" && openAICostOK && openAICost.GreaterThan(decimal.Zero) && strings.TrimSpace(in.Recipe.Model) == "gpt-image-2" && strings.TrimSpace(in.Recipe.ModelVersion) == "current" && strings.TrimSpace(in.Recipe.NegativePrompt) == "" && in.Recipe.MaskAssetID == nil && openAIReferencesExactSource(in.AssetID, in.Recipe.ReferenceAssetIDs) && emptyJSONObject(in.Recipe.Parameters) && ((in.Width == 1024 && in.Height == 1024) || (in.Width == 1024 && in.Height == 1536) || (in.Width == 1536 && in.Height == 1024))
	isExternal := isPhotoroom || isOpenAI
	if ownerID <= 0 || in.AssetID <= 0 || in.IdempotencyKey == "" || (!isDeterministic && !isExternal) || (isOpenAI && strings.TrimSpace(in.Recipe.Prompt) == "") || !validScope(in.Purpose) || !validScope(in.Channel) || in.Width <= 0 || in.Height <= 0 || s.image == nil {
		return nil, ErrInvalidInput
	}
	if capability, ok := s.processorCapability(ctx, photoroomProcessor); isPhotoroom && (!ok || capability.SafetyLevel != "sandbox_only" || capability.ProviderEnvironment != photoroomEnvironment || capability.Region != photoroomRegion || !capability.Watermarked || !capability.NonPublishable || !capability.QuotaAvailable || capability.QuotaRemaining <= 0) {
		return nil, &ConflictError{Code: "PROVIDER_UNAVAILABLE"}
	}
	if capability, ok := s.processorCapability(ctx, openAIProcessor); isOpenAI && (!ok || capability.SafetyLevel != "production_paid" || capability.ProviderEnvironment != openAIEnvironment || capability.Region != openAIRegion || capability.Watermarked || capability.NonPublishable) {
		return nil, &ConflictError{Code: "PROVIDER_UNAVAILABLE"}
	}
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.MaxCost = strings.TrimSpace(in.MaxCost)
	var asset Asset
	if err := s.db.WithContext(ctx).Where("id = ? AND owner_id = ?", in.AssetID, ownerID).First(&asset).Error; err != nil {
		return nil, err
	}
	if asset.ChannelRestriction == "" || (asset.ChannelRestriction != "*" && cleanScope(asset.ChannelRestriction) != in.Channel) {
		return nil, &ConflictError{Code: "ASSET_CHANNEL_RESTRICTED"}
	}
	recipeBytes, recipeHash, err := normalizeRecipe(ctx, s.db, ownerID, asset, &in)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var rights int64
	rightsQuery := s.db.WithContext(ctx).Model(&RightsGrant{}).Where("owner_id = ? AND asset_id = ? AND asset_sha = ? AND purpose = ? AND channel = ? AND provider = ? AND region = ? AND owner_verified = ? AND can_copy = ? AND can_modify = ? AND revoked_at IS NULL AND valid_from <= ? AND (valid_until IS NULL OR valid_until > ?)", ownerID, asset.ID, asset.SHA256, in.Purpose, in.Channel, in.Processor, in.Region, true, true, true, now, now)
	if in.Processor != "deterministic" {
		rightsQuery = rightsQuery.Where("can_third_party_ai = ? AND can_cross_border = ?", true, true)
	}
	if err := rightsQuery.Count(&rights).Error; err != nil {
		return nil, err
	}
	if rights == 0 {
		return nil, &ConflictError{Code: "INPUT_RIGHTS_REQUIRED"}
	}
	manifestBytes, _ := json.Marshal(struct {
		OwnerID                                                                             int64  `json:"owner_id"`
		AssetID                                                                             int64  `json:"asset_id"`
		SKUID                                                                               int64  `json:"sku_id"`
		BlobID                                                                              string `json:"blob_id"`
		RecipeKey                                                                           string `json:"recipe_key"`
		RecipeVersion                                                                       int    `json:"recipe_version"`
		RecipeHash                                                                          string `json:"recipe_hash"`
		ParentTaskID                                                                        *int64 `json:"parent_task_id,omitempty"`
		CandidateRound                                                                      int    `json:"candidate_round"`
		Operation                                                                           string `json:"operation"`
		Width                                                                               int    `json:"width"`
		Height                                                                              int    `json:"height"`
		Format, Processor, Purpose, Channel, Region, ProviderEnvironment, MaxCost, Currency string
	}{ownerID, asset.ID, in.SKUID, asset.BlobID, in.RecipeKey, in.RecipeVersion, recipeHash, in.ParentTaskID, in.CandidateRound, in.Operation, in.Width, in.Height, in.Format, in.Processor, in.Purpose, in.Channel, in.Region, map[bool]string{true: photoroomEnvironment, false: openAIEnvironment}[isPhotoroom], in.MaxCost, in.Currency})
	digest := sha256.Sum256(manifestBytes)
	manifest := hex.EncodeToString(digest[:])
	var existing Task
	var task *Task
	if err := s.db.WithContext(ctx).Where("owner_id = ? AND idempotency_key = ?", ownerID, in.IdempotencyKey).First(&existing).Error; err == nil {
		if existing.ManifestHash != manifest {
			return nil, &ConflictError{Code: "IDEMPOTENCY_CONFLICT"}
		}
		if existing.ImageServiceJobID != "" {
			return &existing, nil
		}
		if !isExternal || existing.Status != "CREATING" {
			return nil, &ConflictError{Code: "TASK_STATE_CONFLICT"}
		}
		task = &existing
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	providerEnvironment := ""
	if isExternal {
		if isPhotoroom {
			providerEnvironment = photoroomEnvironment
		} else {
			providerEnvironment = openAIEnvironment
		}
		if task == nil {
			task = &Task{OwnerID: ownerID, AssetID: asset.ID, SKUID: in.SKUID, RecipeKey: in.RecipeKey, RecipeVersion: in.RecipeVersion, RecipeManifest: recipeBytes, RecipeHash: recipeHash, ParentTaskID: in.ParentTaskID, CandidateRound: in.CandidateRound, IdempotencyKey: in.IdempotencyKey, ManifestHash: manifest, Operation: in.Operation, Processor: in.Processor, Purpose: in.Purpose, Channel: in.Channel, Region: in.Region, ProviderEnvironment: providerEnvironment, MaxCost: in.MaxCost, Currency: in.Currency, Sandbox: isPhotoroom, Watermarked: isPhotoroom, NonPublishable: isPhotoroom, Version: 1, Width: in.Width, Height: in.Height, Format: in.Format, Status: "CREATING"}
			if err := s.db.WithContext(ctx).Create(task).Error; err != nil {
				return nil, fmt.Errorf("persist image task intent: %w", err)
			}
		}
	}
	taskReference := ""
	if task != nil {
		taskReference = strconv.FormatInt(task.ID, 10)
	}
	remote, err := s.image.CreateJob(ctx, imageservice.CreateJobRequest{OwnerID: ownerID, LingMirrorTaskID: taskReference, LingMirrorTaskVersion: map[bool]int64{true: 1}[isExternal], IdempotencyKey: in.IdempotencyKey, ManifestHash: manifest, Operation: in.Operation, Processor: in.Processor, Prompt: in.Recipe.Prompt, InputBlobID: asset.BlobID, Width: in.Width, Height: in.Height, Format: in.Format, MaxCost: in.MaxCost, Currency: in.Currency, Region: in.Region, ProviderEnvironment: providerEnvironment, Sandbox: isPhotoroom, Watermarked: isPhotoroom, NonPublishable: isPhotoroom})
	if err != nil {
		return nil, fmt.Errorf("create image job: %w", err)
	}
	if remote.ID == "" || remote.OwnerID != ownerID || remote.ManifestHash != manifest || remote.Operation != in.Operation || remote.Processor != in.Processor {
		return nil, errors.New("create image job: Image Service returned mismatched ownership or manifest")
	}
	if isPhotoroom && (!remote.Sandbox || !remote.Watermarked || !remote.NonPublishable || remote.ProviderEnvironment != photoroomEnvironment || remote.Region != photoroomRegion || remote.MaxCost != "0" || remote.Currency != "USD") {
		return nil, errors.New("create image job: Image Service did not preserve sandbox restrictions")
	}
	if isOpenAI && (remote.Sandbox || remote.Watermarked || remote.NonPublishable || remote.ProviderEnvironment != openAIEnvironment || remote.Region != openAIRegion || remote.MaxCost != in.MaxCost || remote.Currency != "USD") {
		return nil, errors.New("create image job: Image Service did not preserve production paid restrictions")
	}
	if task == nil {
		task = &Task{OwnerID: ownerID, AssetID: asset.ID, SKUID: in.SKUID, RecipeKey: in.RecipeKey, RecipeVersion: in.RecipeVersion, RecipeManifest: recipeBytes, RecipeHash: recipeHash, ParentTaskID: in.ParentTaskID, CandidateRound: in.CandidateRound, IdempotencyKey: in.IdempotencyKey, ManifestHash: manifest, Operation: in.Operation, Processor: in.Processor, Purpose: in.Purpose, Channel: in.Channel, Region: in.Region, ProviderEnvironment: providerEnvironment, MaxCost: in.MaxCost, Currency: in.Currency, Version: 1, Width: in.Width, Height: in.Height, Format: in.Format}
	}
	task.ImageServiceJobID, task.Status, task.ErrorCode, task.OutputBlobID = remote.ID, remote.Status, remote.ErrorCode, remote.OutputBlobID
	task.Sandbox, task.Watermarked, task.NonPublishable = remote.Sandbox, remote.Watermarked, remote.NonPublishable
	if task.ID == 0 {
		if err := s.db.WithContext(ctx).Create(task).Error; err != nil {
			return nil, fmt.Errorf("persist image task: %w", err)
		}
	} else if err := s.db.WithContext(ctx).Model(&Task{}).Where("id=? AND owner_id=? AND status='CREATING'", task.ID, ownerID).Updates(map[string]any{"image_service_job_id": task.ImageServiceJobID, "status": task.Status, "error_code": task.ErrorCode, "output_blob_id": task.OutputBlobID, "sandbox": task.Sandbox, "watermarked": task.Watermarked, "non_publishable": task.NonPublishable}).Error; err != nil {
		return nil, fmt.Errorf("persist image task job: %w", err)
	}
	return task, nil
}

func emptyJSONObject(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}
	var value map[string]any
	return json.Unmarshal(raw, &value) == nil && len(value) == 0
}

func openAIReferencesExactSource(assetID int64, ids []int64) bool {
	return len(ids) == 0 || len(ids) == 1 && ids[0] == assetID
}

type ConflictError struct{ Code string }

func (e *ConflictError) Error() string { return e.Code }

// verifyRemoteTaskIdentity binds a LingMirror task to exactly one Image
// Service job. External processors must also echo the LingMirror task/version;
// deterministic jobs predate that field and remain bound by the other fields.
func verifyRemoteTaskIdentity(task *Task, remote *imageservice.Job, ownerID int64) bool {
	if task == nil || remote == nil || remote.ID == "" || remote.ID != task.ImageServiceJobID || remote.OwnerID != ownerID ||
		remote.ManifestHash != task.ManifestHash || remote.Operation != task.Operation || remote.Processor != task.Processor {
		return false
	}
	if task.Processor != "deterministic" {
		if remote.LingMirrorTaskID != strconv.FormatInt(task.ID, 10) || remote.LingMirrorTaskVersion != task.Version {
			return false
		}
	}
	if task.Processor == photoroomProcessor {
		return task.ProviderEnvironment == photoroomEnvironment && task.Region == photoroomRegion && task.MaxCost == "0" && task.Currency == "USD" && task.Sandbox && task.Watermarked && task.NonPublishable && remote.ProviderEnvironment == photoroomEnvironment && remote.Region == photoroomRegion && remote.MaxCost == "0" && remote.Currency == "USD" && remote.Sandbox && remote.Watermarked && remote.NonPublishable
	}
	if task.Processor == openAIProcessor {
		return task.ProviderEnvironment == openAIEnvironment && task.Region == openAIRegion && task.Currency == "USD" && !task.Sandbox && !task.Watermarked && !task.NonPublishable && remote.ProviderEnvironment == openAIEnvironment && remote.Region == openAIRegion && remote.MaxCost == task.MaxCost && remote.Currency == "USD" && !remote.Sandbox && !remote.Watermarked && !remote.NonPublishable
	}
	return true
}

func isNonPublishableOutput(task *Task, remote *imageservice.Job) bool {
	if task == nil {
		return true
	}
	if task.Processor == photoroomProcessor || task.Sandbox || task.Watermarked || task.NonPublishable {
		return true
	}
	return remote != nil && (remote.Sandbox || remote.Watermarked || remote.NonPublishable)
}

func verifyTaskChannelLineage(ctx context.Context, db *gorm.DB, ownerID int64, task *Task, channel, purpose string) error {
	if db == nil || task == nil || task.OwnerID != ownerID || cleanScope(task.Channel) != cleanScope(channel) || cleanScope(task.Purpose) != cleanScope(purpose) {
		return ErrGateBlocked
	}
	var asset Asset
	if err := db.WithContext(ctx).Where("id = ? AND owner_id = ?", task.AssetID, ownerID).First(&asset).Error; err != nil {
		return err
	}
	if asset.ChannelRestriction == "" || (cleanScope(asset.ChannelRestriction) != "*" && cleanScope(asset.ChannelRestriction) != cleanScope(channel)) {
		return ErrGateBlocked
	}
	if asset.ParentAssetID != nil && (*asset.ParentAssetID <= 0 || !isSHA256(asset.ParentAssetSHA)) {
		return ErrGateBlocked
	}
	return nil
}

// verifyCurrentInputRights is intentionally re-run immediately before Owner
// approval and execution. A grant revoked or expired after task creation must
// stop the sandbox call before approval consumption or any provider mutation.
func (s *Service) verifyCurrentInputRights(ctx context.Context, ownerID int64, task *Task) error {
	if s == nil {
		return &ConflictError{Code: "INPUT_RIGHTS_REQUIRED"}
	}
	return verifyCurrentInputRightsOnDB(ctx, s.db, ownerID, task)
}

func verifyCurrentInputRightsOnDB(ctx context.Context, db *gorm.DB, ownerID int64, task *Task) error {
	_, err := currentInputRightsGrantOnDB(ctx, db, ownerID, task, false)
	return err
}

func currentInputRightsGrantOnDB(ctx context.Context, db *gorm.DB, ownerID int64, task *Task, lock bool) (*RightsGrant, error) {
	if db == nil || task == nil || task.OwnerID != ownerID || task.AssetID <= 0 {
		return nil, &ConflictError{Code: "INPUT_RIGHTS_REQUIRED"}
	}
	var asset Asset
	if err := db.WithContext(ctx).Where("id=? AND owner_id=?", task.AssetID, ownerID).First(&asset).Error; err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	q := db.WithContext(ctx).Where("owner_id=? AND asset_id=? AND asset_sha=? AND purpose=? AND channel=? AND provider=? AND region=? AND owner_verified=? AND can_copy=? AND can_modify=? AND revoked_at IS NULL AND valid_from<=? AND (valid_until IS NULL OR valid_until>?)", ownerID, asset.ID, asset.SHA256, task.Purpose, task.Channel, task.Processor, task.Region, true, true, true, now, now)
	if task.Processor != "deterministic" {
		q = q.Where("can_third_party_ai=? AND can_cross_border=?", true, true)
	}
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var grant RightsGrant
	if err := q.Order("id DESC").First(&grant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &ConflictError{Code: "INPUT_RIGHTS_REQUIRED"}
		}
		return nil, err
	}
	return &grant, nil
}

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
			if remoteErr != nil || !verifyRemoteTaskIdentity(&tasks[i], remote, ownerID) {
				continue
			}
			mergeRemoteTaskState(&tasks[i], remote)
			_ = s.db.WithContext(ctx).Model(&Task{}).Where("id = ? AND owner_id = ?", tasks[i].ID, ownerID).Updates(map[string]any{"status": tasks[i].Status, "error_code": tasks[i].ErrorCode, "output_blob_id": tasks[i].OutputBlobID, "sandbox": tasks[i].Sandbox, "watermarked": tasks[i].Watermarked, "non_publishable": tasks[i].NonPublishable}).Error
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
			if !verifyRemoteTaskIdentity(&task, remote, ownerID) {
				return nil, errors.New("Image Service returned mismatched job identity")
			}
			mergeRemoteTaskState(&task, remote)
			_ = s.db.WithContext(ctx).Model(&Task{}).Where("id = ? AND owner_id = ?", task.ID, ownerID).Updates(map[string]any{"status": task.Status, "error_code": task.ErrorCode, "output_blob_id": task.OutputBlobID, "sandbox": task.Sandbox, "watermarked": task.Watermarked, "non_publishable": task.NonPublishable}).Error
		} else {
			s.logger.Debug("image service job refresh failed", zap.Error(err), zap.Int64("task_id", task.ID))
		}
	}
	if task.OutputBlobID != "" {
		task.OutputURL = fmt.Sprintf("/api/v1/product-images/tasks/%d/output/content", task.ID)
	}
	return &task, nil
}

func mergeRemoteTaskState(task *Task, remote *imageservice.Job) {
	if task.ErrorCode == "NO_CHARGE_CONFIRMED" || task.ErrorCode == "CHARGED_OUTPUT_UNRECOVERABLE" {
		return
	}
	status, code := remote.Status, remote.ErrorCode
	if task.Status == "RECONCILE_REQUIRED" && (remote.Status == "QUEUED" || (remote.Status == "FAILED" && remote.ErrorCode == "CANCELLED_NO_CHARGE_RECONCILIATION")) {
		status, code = "RECONCILE_REQUIRED", "INTERNAL_DISPATCH_OUTCOME_UNKNOWN"
	}
	task.Status, task.ErrorCode, task.OutputBlobID = status, code, remote.OutputBlobID
	task.Sandbox, task.Watermarked, task.NonPublishable = task.Sandbox || remote.Sandbox, task.Watermarked || remote.Watermarked, task.NonPublishable || remote.NonPublishable
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
	if task.Status == "RECONCILE_REQUIRED" || task.ErrorCode == "NO_CHARGE_CONFIRMED" {
		return nil, &ConflictError{Code: "RECONCILE_REQUIRED"}
	}
	if _, ok := s.processorCapability(ctx, task.Processor); !ok {
		return nil, &ConflictError{Code: "PROVIDER_UNAVAILABLE"}
	}
	if task.Processor == photoroomProcessor {
		if task.ProviderEnvironment != photoroomEnvironment || task.Region != photoroomRegion || task.MaxCost != "0" || task.Currency != "USD" || !task.Sandbox || !task.Watermarked || !task.NonPublishable {
			return nil, &ConflictError{Code: "SANDBOX_CONTRACT_REQUIRED"}
		}
	}
	if task.Processor == openAIProcessor && (task.ProviderEnvironment != openAIEnvironment || task.Region != openAIRegion || task.Currency != "USD" || task.Sandbox || task.Watermarked || task.NonPublishable) {
		return nil, &ConflictError{Code: "PRODUCTION_CONTRACT_REQUIRED"}
	}
	if len(s.executionTokenKey) < 32 {
		return nil, &ConflictError{Code: "APPROVAL_REQUIRED"}
	}
	authorized, ok := s.image.(interface {
		EnqueueAuthorizedExecution(context.Context, string, string, string) (*imageservice.Attempt, error)
	})
	if !ok {
		return nil, errors.New("Image Service client does not support authorized execution")
	}
	// Claim the approval and its budget reservation before the external mutation. A response lost after the
	// provider accepted the request must never allow a second paid submission.
	// The follow-up path is reconciliation, not restoring or reusing approval.
	now := time.Now().UTC()
	var approval ExecutionApproval
	var rightsSnapshot *ExecutionRightsSnapshot
	var token string
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Global lock order for execution approval paths: Owner/currency budget,
		// task, exact rights grant, approval, reservation. Revoke competes for the
		// exact same grant row, closing the check/use gap before any provider call.
		if err := budgetLock(tx, ownerID, task.Currency); err != nil {
			return err
		}
		var lockedTask Task
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND owner_id=?", task.ID, ownerID).First(&lockedTask).Error; err != nil {
			return err
		}
		if lockedTask.Version != task.Version || lockedTask.ManifestHash != task.ManifestHash || lockedTask.Processor != task.Processor || lockedTask.ImageServiceJobID != task.ImageServiceJobID {
			return &ConflictError{Code: "VERSION_CONFLICT"}
		}
		var lockedGrant *RightsGrant
		if lockedTask.Processor == photoroomProcessor || lockedTask.Processor == openAIProcessor {
			var err error
			lockedGrant, err = currentInputRightsGrantOnDB(ctx, tx, ownerID, &lockedTask, true)
			if err != nil {
				return err
			}
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("owner_id = ? AND task_id = ? AND task_version = ? AND manifest_hash = ? AND operation = ? AND processor = ? AND consumed_at IS NULL", ownerID, lockedTask.ID, lockedTask.Version, lockedTask.ManifestHash, lockedTask.Operation, lockedTask.Processor).Order("id DESC").First(&approval).Error; err != nil {
			return &ConflictError{Code: "APPROVAL_REQUIRED"}
		}
		now = time.Now().UTC()
		if !approval.ExpiresAt.After(now) {
			return &ConflictError{Code: "APPROVAL_REQUIRED"}
		}
		var costCount int64
		if err := tx.Model(&CostEntry{}).Where("owner_id = ? AND task_id = ? AND expected_task_version = ? AND provider = ? AND billing_status <> ?", ownerID, lockedTask.ID, lockedTask.Version, lockedTask.Processor, "unknown").Count(&costCount).Error; err != nil {
			return err
		}
		if costCount == 0 {
			return &ConflictError{Code: "BUDGET_COST_REQUIRED"}
		}
		var reservation BudgetReservation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("approval_id=? AND owner_id=? AND state='reserved'", approval.ID, ownerID).First(&reservation).Error; err != nil {
			return &ConflictError{Code: "BUDGET_RESERVATION_REQUIRED"}
		}
		if lockedGrant != nil {
			rightsSnapshot = &ExecutionRightsSnapshot{OwnerID: ownerID, ApprovalID: approval.ID, ApprovalExecutionID: approval.ExecutionID, TaskID: lockedTask.ID, TaskVersion: lockedTask.Version, ManifestHash: lockedTask.ManifestHash, Provider: lockedTask.Processor, GrantID: lockedGrant.ID, GrantVersion: lockedGrant.Version, AssetSHA: lockedGrant.AssetSHA, EvidenceSHA: lockedGrant.EvidenceSHA, GrantRequestHash: lockedGrant.RequestHash, CanCopy: lockedGrant.CanCopy, CanModify: lockedGrant.CanModify, CanThirdPartyAI: lockedGrant.CanThirdPartyAI, CanCrossBorder: lockedGrant.CanCrossBorder, ValidFrom: lockedGrant.ValidFrom, ValidUntil: lockedGrant.ValidUntil, ClaimedAt: now}
			if err := tx.Create(rightsSnapshot).Error; err != nil {
				return err
			}
		}
		token, err = s.signExecutionToken(&lockedTask, &approval, rightsSnapshot, now)
		if err != nil {
			return err
		}
		result := tx.Model(&ExecutionApproval{}).Where("id = ? AND owner_id = ? AND consumed_at IS NULL AND expires_at > ?", approval.ID, ownerID, now).Update("consumed_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return &ConflictError{Code: "APPROVAL_REQUIRED"}
		}
		claimed := tx.Model(&BudgetReservation{}).Where("id=? AND approval_id=? AND owner_id=? AND state='reserved'", reservation.ID, approval.ID, ownerID).Updates(map[string]any{"state": "claimed", "claimed_at": now, "updated_at": now})
		if claimed.Error != nil {
			return claimed.Error
		}
		if claimed.RowsAffected != 1 {
			return &ConflictError{Code: "BUDGET_RESERVATION_REQUIRED"}
		}
		updated := tx.Model(&Task{}).Where("id=? AND owner_id=? AND version=? AND manifest_hash=?", lockedTask.ID, ownerID, lockedTask.Version, lockedTask.ManifestHash).Updates(map[string]any{"status": "RECONCILE_REQUIRED", "error_code": "INTERNAL_DISPATCH_OUTCOME_UNKNOWN", "updated_at": now})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return &ConflictError{Code: "VERSION_CONFLICT"}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	attempt, err := authorized.EnqueueAuthorizedExecution(ctx, task.ImageServiceJobID, key, token)
	if err != nil {
		return nil, err
	}
	if attempt == nil || attempt.JobID != task.ImageServiceJobID || attempt.IdempotencyKey != key {
		return nil, &ConflictError{Code: "RECONCILE_REQUIRED"}
	}
	cleared := s.db.WithContext(context.WithoutCancel(ctx)).Model(&Task{}).Where("id=? AND owner_id=? AND version=? AND manifest_hash=? AND status='RECONCILE_REQUIRED' AND error_code='INTERNAL_DISPATCH_OUTCOME_UNKNOWN'", task.ID, ownerID, task.Version, task.ManifestHash).Updates(map[string]any{"status": "QUEUED", "error_code": "", "updated_at": time.Now().UTC()})
	if cleared.Error != nil || cleared.RowsAffected != 1 {
		return nil, &ConflictError{Code: "RECONCILE_REQUIRED"}
	}
	return attempt, nil
}

var moneyPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,9})(\.[0-9]{1,4})?$`)

func (s *Service) ApproveExecution(ctx context.Context, ownerID, taskID int64, in ApprovalInput) (*ExecutionApproval, error) {
	if ownerID <= 0 || len(s.executionTokenKey) < 32 {
		return nil, ErrInvalidInput
	}
	in.Processor, in.Currency = strings.TrimSpace(in.Processor), strings.ToUpper(strings.TrimSpace(in.Currency))
	in.MaxCost = strings.TrimSpace(in.MaxCost)
	if _, ok := strictMoney(in.MaxCost); !ok && !(in.Processor == photoroomProcessor && in.MaxCost == "0") {
		return nil, ErrInvalidInput
	}
	if in.Processor == "" || in.Processor == "deterministic" || !allowedExecutionCurrency(in.Currency) || in.ExpectedVersion <= 0 {
		return nil, ErrInvalidInput
	}
	var providerCapability imageservice.ProcessorCapability
	var providerAvailable bool
	if in.Processor == photoroomProcessor || in.Processor == openAIProcessor {
		providerCapability, providerAvailable = s.processorCapability(ctx, in.Processor)
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
	var approval *ExecutionApproval
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := budgetLock(tx, ownerID, in.Currency); err != nil {
			return err
		}
		if err := releaseExpiredReservations(tx, ownerID, in.Currency, now); err != nil {
			return err
		}
		var task Task
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND owner_id=?", taskID, ownerID).First(&task).Error; err != nil {
			return err
		}
		if task.Version != in.ExpectedVersion || task.Processor != in.Processor || task.Status != "QUEUED" {
			return &ConflictError{Code: "VERSION_CONFLICT"}
		}
		if task.Processor == photoroomProcessor {
			if !providerAvailable || providerCapability.SafetyLevel != "sandbox_only" || providerCapability.ProviderEnvironment != photoroomEnvironment || providerCapability.Region != photoroomRegion || !providerCapability.Watermarked || !providerCapability.NonPublishable || !providerCapability.QuotaAvailable || providerCapability.QuotaRemaining <= 0 {
				return &ConflictError{Code: "PROVIDER_UNAVAILABLE"}
			}
			if in.MaxCost != task.MaxCost || in.Currency != task.Currency || task.MaxCost != "0" || task.Currency != "USD" || task.ProviderEnvironment != photoroomEnvironment || task.Region != photoroomRegion || !task.Sandbox || !task.Watermarked || !task.NonPublishable {
				return &ConflictError{Code: "SANDBOX_CONTRACT_REQUIRED"}
			}
		}
		if task.Processor == openAIProcessor {
			if !providerAvailable || providerCapability.SafetyLevel != "production_paid" || providerCapability.ProviderEnvironment != openAIEnvironment || providerCapability.Region != openAIRegion || providerCapability.Watermarked || providerCapability.NonPublishable {
				return &ConflictError{Code: "PROVIDER_UNAVAILABLE"}
			}
			if in.MaxCost != task.MaxCost || in.Currency != task.Currency || task.Currency != "USD" || task.ProviderEnvironment != openAIEnvironment || task.Region != openAIRegion || task.Sandbox || task.Watermarked || task.NonPublishable {
				return &ConflictError{Code: "PRODUCTION_CONTRACT_REQUIRED"}
			}
		}
		if task.Processor == photoroomProcessor || task.Processor == openAIProcessor {
			if err := verifyCurrentInputRightsOnDB(ctx, tx, ownerID, &task); err != nil {
				return err
			}
		}
		var existing BudgetReservation
		err := tx.Where("owner_id=? AND task_id=? AND task_version=? AND manifest_hash=? AND provider=?", ownerID, task.ID, task.Version, task.ManifestHash, task.Processor).First(&existing).Error
		if err == nil {
			var prior ExecutionApproval
			if err := tx.First(&prior, existing.ApprovalID).Error; err != nil {
				return err
			}
			if prior.MaxCost != strings.TrimSpace(in.MaxCost) || prior.Currency != in.Currency {
				return &ConflictError{Code: "IDEMPOTENCY_CONFLICT"}
			}
			if existing.State == "released" {
				return &ConflictError{Code: "BUDGET_RESERVATION_RELEASED"}
			}
			prior.BudgetReservationID = existing.ID
			approval = &prior
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var policy BudgetPolicy
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("owner_id=? AND currency=? AND period_start <= ? AND period_end > ?", ownerID, in.Currency, now, now).First(&policy).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &ConflictError{Code: "BUDGET_POLICY_REQUIRED"}
			}
			return err
		}
		capAmount, _ := decimal.NewFromString(policy.TotalAmount)
		requested, _ := decimal.NewFromString(strings.TrimSpace(in.MaxCost))
		exposure, err := policyExposure(tx, policy.ID)
		if err != nil {
			return err
		}
		if exposure.Add(requested).GreaterThan(capAmount) {
			return &ConflictError{Code: "BUDGET_EXCEEDED"}
		}
		created := &ExecutionApproval{ExecutionID: executionID, OwnerID: ownerID, TaskID: task.ID, TaskVersion: task.Version, ManifestHash: task.ManifestHash, Operation: task.Operation, Processor: task.Processor, MaxCost: strings.TrimSpace(in.MaxCost), Currency: in.Currency, Nonce: nonce, ApprovedAt: now, ExpiresAt: now.Add(3 * time.Minute)}
		if err := tx.Create(created).Error; err != nil {
			return err
		}
		reservation := BudgetReservation{OwnerID: ownerID, PolicyID: policy.ID, ApprovalID: created.ID, TaskID: task.ID, TaskVersion: task.Version, ManifestHash: task.ManifestHash, Provider: task.Processor, Currency: in.Currency, ReservedAmount: created.MaxCost, State: "reserved"}
		if err := tx.Create(&reservation).Error; err != nil {
			return err
		}
		created.BudgetReservationID = reservation.ID
		costInput := CostEntryInput{Kind: "estimated", Category: "provider", Provider: task.Processor, Amount: created.MaxCost, Currency: created.Currency, ExchangeRate: "1", ExchangeRateSource: "same_currency_budget_cap", ObservedAt: now, BillingStatus: "estimated", IdempotencyKey: "execution-approval:" + executionID, ExpectedVersion: task.Version}
		cost := &CostEntry{OwnerID: ownerID, TaskID: task.ID, Kind: costInput.Kind, Category: costInput.Category, Provider: costInput.Provider, Amount: costInput.Amount, Currency: costInput.Currency, ExchangeRate: costInput.ExchangeRate, ExchangeRateSource: costInput.ExchangeRateSource, ObservedAt: now, BillingStatus: costInput.BillingStatus, IdempotencyKey: costInput.IdempotencyKey, RequestHash: requestHash(costInput), ExpectedTaskVersion: task.Version}
		if err := tx.Create(cost).Error; err != nil {
			return err
		}
		approval = created
		return nil
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

func (s *Service) signExecutionToken(task *Task, approval *ExecutionApproval, rights *ExecutionRightsSnapshot, now time.Time) (string, error) {
	if task == nil || approval == nil || len(s.executionTokenKey) < 32 {
		return "", ErrInvalidInput
	}
	nbf := now.Unix()
	exp := approval.ExpiresAt.Unix()
	if exp <= nbf || exp-nbf > 300 {
		return "", ErrInvalidInput
	}
	claims := map[string]any{"approval_execution_id": approval.ExecutionID, "task_id": strconv.FormatInt(task.ID, 10), "task_version": task.Version, "owner_id": task.OwnerID, "job_id": task.ImageServiceJobID, "manifest_hash": task.ManifestHash, "operation": task.Operation, "processor": task.Processor, "provider_environment": task.ProviderEnvironment, "region": task.Region, "sandbox": task.Sandbox, "watermarked": task.Watermarked, "non_publishable": task.NonPublishable, "max_cost": approval.MaxCost, "currency": approval.Currency, "nonce": approval.Nonce, "iat": now.Unix(), "nbf": nbf, "exp": exp, "aud": "lingmirror-image-service-execution"}
	if rights != nil {
		claims["execution_rights_snapshot_id"] = rights.ID
		claims["rights_grant_id"] = rights.GrantID
		claims["rights_grant_version"] = rights.GrantVersion
		claims["rights_evidence_sha256"] = rights.EvidenceSHA
	}
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
