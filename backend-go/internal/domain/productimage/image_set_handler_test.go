package productimage

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/imageservice"
	"gorm.io/gorm"
)

type imageSetTestListing struct{ ID, ProductID, PlatformID int64 }

func (imageSetTestListing) TableName() string { return "product_listing" }

type imageSetTestPlatform struct {
	ID   int64
	Code string
}

func (imageSetTestPlatform) TableName() string { return "platform" }

type imageSetTestMaster struct{ ID, OwnerID int64 }

func (imageSetTestMaster) TableName() string { return "product_master" }

type imageSetTestVariant struct{ ID, ProductMasterID, SKUProductID int64 }

func (imageSetTestVariant) TableName() string { return "product_variant" }

func imageSetHandlerRouter(t *testing.T, owner int64) (*gin.Engine, *gorm.DB, *fakeImageService) {
	t.Helper()
	db := dbtest.NewDB(t, &Asset{}, &Task{}, &RightsGrant{}, &Review{}, &CostEntry{}, &ImageSet{}, &ImageSetItem{}, &imageSetTestListing{}, &imageSetTestPlatform{}, &imageSetTestMaster{}, &imageSetTestVariant{})
	remote := newFakeImageService()
	db.Create(&Asset{ID: 1, OwnerID: owner, BlobID: strings.Repeat("9", 64), SHA256: strings.Repeat("9", 64), Filename: "input.png", ContentType: "image/png", SourceKind: "upload", ChannelRestriction: "*"})
	db.Create(&imageSetTestPlatform{ID: 5, Code: "ozon"})
	db.Create(&imageSetTestMaster{ID: 6, OwnerID: owner})
	db.Create(&imageSetTestVariant{ID: 7, ProductMasterID: 6, SKUProductID: 8})
	db.Create(&imageSetTestListing{ID: 91, ProductID: 8, PlatformID: 5})
	r := gin.New()
	g := r.Group("/product-images")
	g.Use(func(c *gin.Context) { c.Set("user_id", owner) })
	RegisterImageSetRoutes(g, db, remote)
	return r, db, remote
}

func seedImageSetGate(t *testing.T, db *gorm.DB, task Task, purpose string) {
	t.Helper()
	now := time.Now().UTC()
	grant := RightsGrant{OwnerID: task.OwnerID, AssetSHA: task.OutputBlobID, CanCopy: true, CanCrossBorder: true, CanCommercialPublish: true, CanPlatformSublicense: true, TrademarkCleared: true, LikenessCleared: true, Purpose: purpose, Jurisdiction: "ru-ru", Channel: "ozon", Provider: "deterministic", Region: "local", Grantor: "owner", RightsChain: "test evidence", EvidenceSHA: strings.Repeat("e", 64), OwnerVerified: true, ValidFrom: now.Add(-time.Hour), IdempotencyKey: "rights-" + task.IdempotencyKey, RequestHash: strings.Repeat("f", 64), Version: 1}
	review := Review{OwnerID: task.OwnerID, TaskID: task.ID, Decision: "five_axis_review", Truth: TruthUnknown, AssetSHA: task.OutputBlobID, Purpose: purpose, Channel: "ozon", ProductAuthenticity: ReviewPassed, RightsStatus: ReviewPassed, ChannelRules: ReviewPassed, ClaimsScene: ReviewPassed, TechnicalVisual: ReviewPassed, EvidenceSHA: strings.Repeat("e", 64), EvidenceTruth: TruthQuoted, IdempotencyKey: "review-" + task.IdempotencyKey, RequestHash: strings.Repeat("d", 64), ExpectedTaskVersion: 1, VerifiedAt: &now}
	if err := db.Create(&grant).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&review).Error; err != nil {
		t.Fatal(err)
	}
}

func normalizeImageSetTask(task *Task) {
	if task.Processor == "" {
		task.Processor = "deterministic"
	}
	if task.Purpose == "" {
		task.Purpose = "listing_main"
	}
	if task.Channel == "" {
		task.Channel = "ozon"
	}
	if task.Region == "" {
		task.Region = "local"
	}
}

func TestImageSetHandlerCreatesAndFreezesOwnerReadyCandidates(t *testing.T) {
	r, db, remote := imageSetHandlerRouter(t, 42)
	ready := Task{OwnerID: 42, AssetID: 1, ImageServiceJobID: "job-ready", IdempotencyKey: "ready", ManifestHash: strings.Repeat("a", 64), Operation: "DETERMINISTIC_RESIZE", Processor: "deterministic", Purpose: "listing_main", Channel: "ozon", Region: "local", Width: 100, Height: 100, Format: "png", Status: "READY", OutputBlobID: strings.Repeat("b", 64)}
	remote.jobs["job-ready"] = imageservice.Job{ID: "job-ready", OwnerID: 42, ManifestHash: ready.ManifestHash, Operation: ready.Operation, Processor: ready.Processor, Status: "READY", OutputBlobID: ready.OutputBlobID}
	if err := db.Create(&ready).Error; err != nil {
		t.Fatal(err)
	}
	seedImageSetGate(t, db, ready, "listing_main")
	body := fmt.Sprintf(`{"listing_id":91,"channel":"ozon","locale":"ru-RU","items":[{"task_id":%d,"role":"main","ordinal":1}]}`, ready.ID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/product-images/image-sets", strings.NewReader(body)))
	if w.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}
	var set ImageSet
	if err := db.Preload("Items").First(&set).Error; err != nil {
		t.Fatal(err)
	}
	if len(set.Items) != 1 || set.Items[0].AssetSHA != ready.OutputBlobID {
		t.Fatalf("set=%+v", set)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, fmt.Sprintf("/product-images/image-sets/%d/freeze", set.ID), nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"status":"frozen"`) || !strings.Contains(w.Body.String(), `"manifest_sha256"`) {
		t.Fatalf("freeze status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestImageSetHandlerRejectsNonReadyAndAnotherOwnersCandidates(t *testing.T) {
	r, db, remote := imageSetHandlerRouter(t, 42)
	for _, task := range []Task{
		{OwnerID: 42, AssetID: 1, ImageServiceJobID: "queued", IdempotencyKey: "queued", ManifestHash: strings.Repeat("a", 64), Operation: "DETERMINISTIC_RESIZE", Processor: "deterministic", Purpose: "listing_main", Channel: "ozon", Region: "local", Width: 100, Height: 100, Format: "png", Status: "QUEUED"},
		{OwnerID: 99, AssetID: 1, ImageServiceJobID: "other", IdempotencyKey: "other", ManifestHash: strings.Repeat("c", 64), Operation: "DETERMINISTIC_RESIZE", Processor: "deterministic", Purpose: "listing_main", Channel: "ozon", Region: "local", Width: 100, Height: 100, Format: "png", Status: "READY", OutputBlobID: strings.Repeat("d", 64)},
	} {
		if err := db.Create(&task).Error; err != nil {
			t.Fatal(err)
		}
	}
	remote.jobs["queued"] = imageservice.Job{ID: "queued", OwnerID: 42, ManifestHash: strings.Repeat("a", 64), Operation: "DETERMINISTIC_RESIZE", Processor: "deterministic", Status: "QUEUED"}
	for id, want := range map[int]string{1: "CANDIDATE_NOT_READY", 2: "NOT_FOUND"} {
		w := httptest.NewRecorder()
		body := fmt.Sprintf(`{"listing_id":91,"channel":"ozon","locale":"ru-RU","items":[{"task_id":%d,"role":"main","ordinal":1}]}`, id)
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/product-images/image-sets", strings.NewReader(body)))
		if !strings.Contains(w.Body.String(), want) {
			t.Fatalf("id=%d status=%d body=%s", id, w.Code, w.Body.String())
		}
	}
}

func TestImageSetHandlerBlocksReadyCandidateWithoutRightsAndFiveReviews(t *testing.T) {
	r, db, remote := imageSetHandlerRouter(t, 42)
	ready := Task{OwnerID: 42, AssetID: 1, ImageServiceJobID: "job-gated", IdempotencyKey: "gated", ManifestHash: strings.Repeat("a", 64), Operation: "DETERMINISTIC_RESIZE", Processor: "deterministic", Version: 1, Width: 100, Height: 100, Format: "png", Status: "READY", OutputBlobID: strings.Repeat("b", 64)}
	normalizeImageSetTask(&ready)
	if err := db.Create(&ready).Error; err != nil {
		t.Fatal(err)
	}
	remote.jobs[ready.ImageServiceJobID] = imageservice.Job{ID: ready.ImageServiceJobID, OwnerID: 42, ManifestHash: ready.ManifestHash, Operation: ready.Operation, Processor: ready.Processor, Status: "READY", OutputBlobID: ready.OutputBlobID}
	body := fmt.Sprintf(`{"listing_id":91,"channel":"ozon","locale":"ru-RU","items":[{"task_id":%d,"role":"main","ordinal":1}]}`, ready.ID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/product-images/image-sets", strings.NewReader(body)))
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "GATE_BLOCKED") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestImageSetFreezeRechecksRightsRevocation(t *testing.T) {
	r, db, remote := imageSetHandlerRouter(t, 42)
	ready := Task{OwnerID: 42, AssetID: 1, ImageServiceJobID: "job-revoke", IdempotencyKey: "revoke", ManifestHash: strings.Repeat("a", 64), Operation: "DETERMINISTIC_RESIZE", Processor: "deterministic", Version: 1, Width: 100, Height: 100, Format: "png", Status: "READY", OutputBlobID: strings.Repeat("b", 64)}
	normalizeImageSetTask(&ready)
	if err := db.Create(&ready).Error; err != nil {
		t.Fatal(err)
	}
	seedImageSetGate(t, db, ready, "listing_main")
	remote.jobs[ready.ImageServiceJobID] = imageservice.Job{ID: ready.ImageServiceJobID, OwnerID: 42, ManifestHash: ready.ManifestHash, Operation: ready.Operation, Processor: ready.Processor, Status: "READY", OutputBlobID: ready.OutputBlobID}
	body := fmt.Sprintf(`{"listing_id":91,"channel":"ozon","locale":"ru-RU","items":[{"task_id":%d,"role":"main","ordinal":1}]}`, ready.ID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/product-images/image-sets", strings.NewReader(body)))
	if w.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}
	var set ImageSet
	if err := db.First(&set).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := db.Model(&RightsGrant{}).Where("owner_id = ? AND asset_sha = ?", 42, ready.OutputBlobID).Updates(map[string]any{"revoked_at": now, "revocation_reason": "withdrawn"}).Error; err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, fmt.Sprintf("/product-images/image-sets/%d/freeze", set.ID), nil))
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "GATE_BLOCKED") {
		t.Fatalf("freeze status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestImageSetHandlerRejectsMissingOrUnownedListing(t *testing.T) {
	r, db, remote := imageSetHandlerRouter(t, 42)
	ready := Task{OwnerID: 42, AssetID: 1, ImageServiceJobID: "job-ready", IdempotencyKey: "ready-listing", ManifestHash: strings.Repeat("a", 64), Operation: "DETERMINISTIC_RESIZE", Width: 100, Height: 100, Format: "png", Status: "READY", OutputBlobID: strings.Repeat("b", 64)}
	normalizeImageSetTask(&ready)
	if err := db.Create(&ready).Error; err != nil {
		t.Fatal(err)
	}
	remote.jobs[ready.ImageServiceJobID] = imageservice.Job{ID: ready.ImageServiceJobID, OwnerID: 42, ManifestHash: ready.ManifestHash, Operation: ready.Operation, Processor: ready.Processor, Status: "READY", OutputBlobID: ready.OutputBlobID}
	seedImageSetGate(t, db, ready, "listing_main")
	for _, listingID := range []int{999, 91} {
		if listingID == 91 {
			db.Model(&imageSetTestMaster{}).Where("id = ?", 6).Update("owner_id", 99)
		}
		body := fmt.Sprintf(`{"listing_id":%d,"channel":"ozon","locale":"ru-RU","items":[{"task_id":%d,"role":"main","ordinal":1}]}`, listingID, ready.ID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/product-images/image-sets", strings.NewReader(body)))
		if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "LISTING_NOT_FOUND") {
			t.Fatalf("listing=%d status=%d body=%s", listingID, w.Code, w.Body.String())
		}
	}
}

func TestImageSetHandlerRejectsForgedReadyAndChangedLineageAtFreeze(t *testing.T) {
	r, db, remote := imageSetHandlerRouter(t, 42)
	ready := Task{OwnerID: 42, AssetID: 1, ImageServiceJobID: "job-ready", IdempotencyKey: "ready-lineage", ManifestHash: strings.Repeat("a", 64), Operation: "DETERMINISTIC_RESIZE", Width: 100, Height: 100, Format: "png", Status: "READY", OutputBlobID: strings.Repeat("b", 64)}
	normalizeImageSetTask(&ready)
	if err := db.Create(&ready).Error; err != nil {
		t.Fatal(err)
	}
	seedImageSetGate(t, db, ready, "listing_main")
	remote.jobs[ready.ImageServiceJobID] = imageservice.Job{ID: ready.ImageServiceJobID, OwnerID: 42, ManifestHash: ready.ManifestHash, Operation: ready.Operation, Processor: ready.Processor, Status: "RUNNING"}
	body := fmt.Sprintf(`{"listing_id":91,"channel":"ozon","locale":"ru-RU","items":[{"task_id":%d,"role":"main","ordinal":1}]}`, ready.ID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/product-images/image-sets", strings.NewReader(body)))
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "CANDIDATE_NOT_READY") {
		t.Fatalf("forged ready status=%d body=%s", w.Code, w.Body.String())
	}

	remote.jobs[ready.ImageServiceJobID] = imageservice.Job{ID: ready.ImageServiceJobID, OwnerID: 42, ManifestHash: ready.ManifestHash, Operation: ready.Operation, Processor: ready.Processor, Status: "READY", OutputBlobID: ready.OutputBlobID}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/product-images/image-sets", strings.NewReader(body)))
	if w.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}
	var set ImageSet
	if err := db.First(&set).Error; err != nil {
		t.Fatal(err)
	}
	changed := remote.jobs[ready.ImageServiceJobID]
	changed.OutputBlobID = strings.Repeat("c", 64)
	remote.jobs[ready.ImageServiceJobID] = changed
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, fmt.Sprintf("/product-images/image-sets/%d/freeze", set.ID), nil))
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "LINEAGE_INVALID") {
		t.Fatalf("freeze status=%d body=%s", w.Code, w.Body.String())
	}
	var persisted ImageSet
	if err := db.First(&persisted, set.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.Status != ImageSetDraft {
		t.Fatalf("changed lineage froze set: %+v", persisted)
	}
}
