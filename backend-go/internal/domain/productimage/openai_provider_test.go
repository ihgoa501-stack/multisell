package productimage

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/imageservice"
)

type openAICapabilityImageService struct{ *fakeImageService }

func (f *openAICapabilityImageService) ListProcessors(context.Context) ([]imageservice.ProcessorCapability, error) {
	return []imageservice.ProcessorCapability{{Code: openAIProcessor, Available: true, Operations: []string{openAIOperation}, SafetyLevel: "production_paid", ProviderEnvironment: openAIEnvironment, Region: openAIRegion}}, nil
}

func TestOpenAITaskFreezesPromptProductionCostAndRemoteIdentity(t *testing.T) {
	db := dbtest.NewDB(t, &canonicalSKU{}, &Asset{}, &Task{}, &RightsGrant{})
	if err := db.Create(&canonicalSKU{ID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	remote := &openAICapabilityImageService{newFakeImageService()}
	svc := NewService(db, dbtest.NewLogger(t), remote)
	asset, err := svc.CreateAsset(t.Context(), 91, "product.png", "image/png", []byte("openai-product"))
	if err != nil {
		t.Fatal(err)
	}
	assetID := asset.ID
	_, err = svc.CreateRightsGrant(t.Context(), 91, RightsGrantInput{AssetID: &assetID, AssetSHA: asset.SHA256, CanCopy: true, CanModify: true, CanThirdPartyAI: true, CanCrossBorder: true, Purpose: "listing_main", Channel: "ozon", Jurisdiction: "us", Provider: openAIProcessor, Region: openAIRegion, Grantor: "owner", RightsChain: "test", EvidenceSHA: strings.Repeat("e", 64), OwnerVerified: true, ValidFrom: time.Now().Add(-time.Minute), IdempotencyKey: "openai-rights", ExpectedVersion: 1})
	if err != nil {
		t.Fatal(err)
	}
	in := CreateTaskInput{AssetID: asset.ID, SKUID: 1, RecipeKey: "shoe-scene", RecipeVersion: 1, CandidateRound: 1, Recipe: RecipeManifest{SceneStructure: "shoe on a clean shelf", Prompt: "Keep the exact shoe unchanged and place it on a clean shelf", Model: "gpt-image-2", ModelVersion: "current", Parameters: json.RawMessage(`{}`), MustNotChange: []string{"shape", "color", "logo"}}, IdempotencyKey: "openai-task", Operation: openAIOperation, Processor: openAIProcessor, Purpose: "listing_main", Channel: "ozon", Region: openAIRegion, Width: 1024, Height: 1024, Format: "png", MaxCost: "0.20", Currency: "USD"}
	task, err := svc.CreateTask(t.Context(), 91, in)
	if err != nil {
		t.Fatal(err)
	}
	job := remote.jobs[task.ImageServiceJobID]
	if task.ProviderEnvironment != openAIEnvironment || task.Sandbox || task.Watermarked || task.NonPublishable || task.MaxCost != "0.20" || job.Prompt != in.Recipe.Prompt || job.LingMirrorTaskID != strconv.FormatInt(task.ID, 10) || !verifyRemoteTaskIdentity(task, &job, 91) {
		t.Fatalf("production contract not frozen task=%+v job=%+v", task, job)
	}

	for name, mutate := range map[string]func(*CreateTaskInput){
		"empty prompt":  func(v *CreateTaskInput) { v.Recipe.Prompt = "" },
		"zero cost":     func(v *CreateTaskInput) { v.MaxCost = "0" },
		"wrong size":    func(v *CreateTaskInput) { v.Width = 1200 },
		"wrong region":  func(v *CreateTaskInput) { v.Region = "eu" },
		"model drift":   func(v *CreateTaskInput) { v.Recipe.Model = "another-model" },
		"unused params": func(v *CreateTaskInput) { v.Recipe.Parameters = json.RawMessage(`{"quality":"high"}`) },
	} {
		t.Run(name, func(t *testing.T) {
			bad := in
			bad.IdempotencyKey = "bad-" + name
			mutate(&bad)
			if _, err := svc.CreateTask(t.Context(), 91, bad); err == nil {
				t.Fatal("unsafe OpenAI task accepted")
			}
		})
	}
}
