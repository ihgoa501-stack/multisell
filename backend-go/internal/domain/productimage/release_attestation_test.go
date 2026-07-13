package productimage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"sync"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/imageservice"
	"gorm.io/gorm"
)

type releaseMaster struct{ ID, OwnerID int64 }

func (releaseMaster) TableName() string { return "product_master" }

type releaseVariant struct{ ID, ProductMasterID, SkuProductID int64 }

func (releaseVariant) TableName() string { return "product_variant" }

func releaseFixture(t *testing.T) (*ReleaseService, *Service, *fakeImageService, *ImageSet, *RightsGrant, *ImageRuleSnapshot) {
	t.Helper()
	db := dbtest.NewDB(t, &Asset{}, &Task{}, &RightsGrant{}, &Review{}, &CostEntry{}, &ImageSet{}, &ImageSetItem{}, &ImageRuleSnapshot{}, &ImageSetDecision{}, &ImageReleaseAttestation{}, &ImageReleaseAttestationItem{}, &ImagePublishAttempt{}, &releaseListingRow{}, &releaseProductRow{}, &releasePlatformRow{}, &releaseAccountRow{}, &releaseMaster{}, &releaseVariant{})
	remote := newFakeImageService()
	img := image.NewRGBA(image.Rect(0, 0, 2, 3))
	img.Set(0, 0, color.White)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	blob := shaHex(buf.Bytes())
	remote.blobs[blob] = buf.Bytes()
	if err := db.Create(&Asset{ID: 1, OwnerID: 42, BlobID: fmt.Sprintf("%064x", 6), SHA256: fmt.Sprintf("%064x", 6), Filename: "input.png", ContentType: "image/png", SourceKind: "upload", ChannelRestriction: "*"}).Error; err != nil {
		t.Fatal(err)
	}
	task := Task{OwnerID: 42, AssetID: 1, ImageServiceJobID: "job-release", OutputBlobID: blob, IdempotencyKey: "task-release", ManifestHash: fmt.Sprintf("%064x", 7), Operation: "DETERMINISTIC_RESIZE", Processor: "deterministic", Purpose: "listing_main", Channel: "ozon", Region: "local", Version: 1, Width: 2, Height: 3, Format: "png", Status: "READY"}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	remote.jobs[task.ImageServiceJobID] = imageservice.Job{ID: task.ImageServiceJobID, OwnerID: 42, Status: "READY", ManifestHash: task.ManifestHash, Operation: task.Operation, Processor: "deterministic", OutputBlobID: blob}
	if err := db.Create(&releaseProductRow{ID: 77, CategoryID: 9}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&releasePlatformRow{ID: 5, Code: "ozon"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&releaseAccountRow{ID: 6, PlatformID: 5, Status: "active"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&releaseListingRow{ID: 10, ProductID: 77, PlatformID: 5, Status: "draft", PublishedData: []byte(`{"title":"approved"}`)}).Error; err != nil {
		t.Fatal(err)
	}
	db.Create(&releaseMaster{ID: 2, OwnerID: 42})
	db.Create(&releaseVariant{ID: 3, ProductMasterID: 2, SkuProductID: 77})
	setSvc := NewImageSetService(db)
	set, err := setSvc.CreateDraft(context.Background(), CreateImageSetInput{OwnerID: 42, ListingID: 10, Channel: "ozon", Locale: "ru-RU", Items: []ImageSetItemInput{{Role: "main", Ordinal: 1, Locale: "ru-RU", Channel: "ozon", AssetSHA: blob, TaskID: task.ID, OutputBlobID: blob, TaskManifestHash: task.ManifestHash, Operation: task.Operation, Processor: "deterministic", ImageServiceJobID: task.ImageServiceJobID}}})
	if err != nil {
		t.Fatal(err)
	}
	set, err = setSvc.SelectAndFreeze(context.Background(), 42, set.ID)
	if err != nil {
		t.Fatal(err)
	}
	gov := NewService(db, dbtest.NewLogger(t), remote)
	ri := validRightsInput()
	ri.AssetSHA = blob
	ri.IdempotencyKey = "release-right"
	grant, err := gov.CreateRightsGrant(context.Background(), 42, ri)
	if err != nil {
		t.Fatal(err)
	}
	_, err = gov.CreateFiveAxisReview(context.Background(), 42, task.ID, FiveAxisReviewInput{AssetSHA: blob, Purpose: "listing_main", Channel: "ozon", ProductAuthenticity: "passed", RightsStatus: "passed", ChannelRules: "passed", ClaimsScene: "passed", TechnicalVisual: "passed", EvidenceSHA: fmt.Sprintf("%064x", 8), EvidenceTruth: "quoted", IdempotencyKey: "release-review", ExpectedVersion: 1})
	if err != nil {
		t.Fatal(err)
	}
	release := NewReleaseService(db, remote, "01234567890123456789012345678901", "release-v1")
	_, err = release.DecideImageSet(context.Background(), 42, set.ID, DecideImageSetInput{Decision: "approved", Reason: "Owner checked exact full-size bytes", IdempotencyKey: "decision", ExpectedVersion: set.Version})
	if err != nil {
		t.Fatal(err)
	}
	rule, err := release.CreateRuleSnapshot(context.Background(), 42, CreateRuleSnapshotInput{Channel: "ozon", Site: "ru", Locale: "ru-RU", CategoryID: 9, Rules: []byte(`{"schema_version":"1","max_images":10,"allowed_roles":["main","gallery","detail","ad_cover"],"allowed_formats":["png","jpeg"],"min_width_px":1,"min_height_px":1,"max_file_bytes":10485760}`), EffectiveAt: time.Now().Add(-time.Hour), IdempotencyKey: "rule"})
	if err != nil {
		t.Fatal(err)
	}
	return release, gov, remote, set, grant, rule
}

func issueRelease(t *testing.T, svc *ReleaseService, set *ImageSet, rule *ImageRuleSnapshot, key string, ttl time.Duration) *ImageReleaseAttestation {
	t.Helper()
	a, err := svc.Issue(context.Background(), 42, IssueAttestationInput{ImageSetID: set.ID, RuleSnapshotID: rule.ID, PlatformAccountID: 6, Site: "ru", IdempotencyKey: key, TTL: ttl})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestReleaseAttestationBindsCanonicalClaimsAndOwner(t *testing.T) {
	svc, _, _, set, _, rule := releaseFixture(t)
	a := issueRelease(t, svc, set, rule, "issue", time.Hour)
	if len(a.ClaimsSHA256) != 64 || len(a.Signature) != 64 || a.KeyID != "release-v1" || len(a.Items) != 1 || a.Items[0].Width != 2 || a.Items[0].Height != 3 {
		t.Fatalf("attestation=%+v", a)
	}
	if _, err := svc.Get(context.Background(), 99, a.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("cross owner err=%v", err)
	}
	again := issueRelease(t, svc, set, rule, "issue", time.Hour)
	if again.ID != a.ID {
		t.Fatal("idempotent issue created another attestation")
	}
	if _, err := svc.Issue(context.Background(), 42, IssueAttestationInput{ImageSetID: set.ID, RuleSnapshotID: rule.ID, PlatformAccountID: 6, Site: "ru", IdempotencyKey: "issue", TTL: 2 * time.Hour}); err == nil {
		t.Fatal("same idempotency key accepted changed intent")
	}
}

func TestReleaseAttestationPermanentlyRejectsSandboxOutput(t *testing.T) {
	svc, _, remote, set, _, rule := releaseFixture(t)
	item := set.Items[0]
	if err := svc.db.Model(&Task{}).Where("id=?", item.TaskID).Updates(map[string]any{"sandbox": true, "watermarked": true, "non_publishable": true}).Error; err != nil {
		t.Fatal(err)
	}
	job := remote.jobs[item.ImageServiceJobID]
	job.Sandbox, job.Watermarked, job.NonPublishable = true, true, true
	remote.jobs[item.ImageServiceJobID] = job
	if _, err := svc.Issue(t.Context(), 42, IssueAttestationInput{ImageSetID: set.ID, RuleSnapshotID: rule.ID, PlatformAccountID: 6, Site: "ru", IdempotencyKey: "sandbox-rejected", TTL: time.Hour}); !errors.Is(err, ErrReleaseGateBlocked) {
		t.Fatalf("sandbox output received release attestation: %v", err)
	}
}

func TestReleaseAttestationDetectsBlobTamperAndRightsRevocation(t *testing.T) {
	t.Run("blob", func(t *testing.T) {
		svc, _, remote, set, _, rule := releaseFixture(t)
		a := issueRelease(t, svc, set, rule, "tamper", time.Hour)
		remote.blobs[a.Items[0].BlobID] = []byte("tampered")
		if err := svc.Consume(context.Background(), 42, a.ID, "test", 1); !errors.Is(err, ErrReleaseGateBlocked) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("rights", func(t *testing.T) {
		svc, gov, _, set, grant, rule := releaseFixture(t)
		a := issueRelease(t, svc, set, rule, "revoke", time.Hour)
		if _, err := gov.RevokeRightsGrant(context.Background(), 42, grant.ID, RevokeRightsInput{ExpectedVersion: grant.Version, IdempotencyKey: "revoke", Reason: "withdrawn"}); err != nil {
			t.Fatal(err)
		}
		if err := svc.Consume(context.Background(), 42, a.ID, "test", 1); !errors.Is(err, ErrReleaseGateBlocked) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestReleaseAttestationRuleChangeAndExpiryFailClosed(t *testing.T) {
	t.Run("new rule", func(t *testing.T) {
		svc, _, _, set, _, rule := releaseFixture(t)
		a := issueRelease(t, svc, set, rule, "old-rule", time.Hour)
		if _, err := svc.CreateRuleSnapshot(context.Background(), 42, CreateRuleSnapshotInput{Channel: "ozon", Site: "ru", Locale: "ru-RU", CategoryID: 9, Rules: []byte(`{"schema_version":"1","max_images":9,"allowed_roles":["main"],"allowed_formats":["png"],"min_width_px":1,"min_height_px":1,"max_file_bytes":10485760}`), EffectiveAt: time.Now().Add(-time.Minute), IdempotencyKey: "rule-2"}); err != nil {
			t.Fatal(err)
		}
		if err := svc.Consume(context.Background(), 42, a.ID, "test", 1); !errors.Is(err, ErrReleaseGateBlocked) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("expired", func(t *testing.T) {
		svc, _, _, set, _, rule := releaseFixture(t)
		a := issueRelease(t, svc, set, rule, "expire", time.Hour)
		svc.db.Model(a).Update("expires_at", time.Now().Add(-time.Second))
		if err := svc.Consume(context.Background(), 42, a.ID, "test", 1); !errors.Is(err, ErrAttestationExpired) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestReleaseAttestationConcurrentSingleConsumption(t *testing.T) {
	svc, _, _, set, _, rule := releaseFixture(t)
	a := issueRelease(t, svc, set, rule, "consume", time.Hour)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := int64(1); i <= 2; i++ {
		wg.Add(1)
		go func(id int64) { defer wg.Done(); errs <- svc.Consume(context.Background(), 42, a.ID, "publish", id) }(i)
	}
	wg.Wait()
	close(errs)
	success, blocked := 0, 0
	for err := range errs {
		if err == nil {
			success++
		} else if errors.Is(err, ErrAttestationConsumed) {
			blocked++
		} else {
			t.Fatalf("unexpected err=%v", err)
		}
	}
	if success != 1 || blocked != 1 {
		t.Fatalf("success=%d blocked=%d", success, blocked)
	}
}
