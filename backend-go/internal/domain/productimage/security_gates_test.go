package productimage

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/imageservice"
	"github.com/lingmirror/backend-go/internal/rbac"
)

func TestRemoteIdentityRequiresProcessorAndExternalTaskVersion(t *testing.T) {
	task := &Task{ID: 7, OwnerID: 11, ImageServiceJobID: "job", ManifestHash: strings.Repeat("a", 64), Operation: openAIOperation, Processor: openAIProcessor, Region: openAIRegion, ProviderEnvironment: openAIEnvironment, MaxCost: "0.20", Currency: "USD", Version: 3}
	remote := &imageservice.Job{ID: "job", OwnerID: 11, ManifestHash: task.ManifestHash, Operation: task.Operation, Processor: task.Processor, LingMirrorTaskID: "7", LingMirrorTaskVersion: 3, Region: task.Region, ProviderEnvironment: task.ProviderEnvironment, MaxCost: task.MaxCost, Currency: task.Currency}
	if !verifyRemoteTaskIdentity(task, remote, 11) {
		t.Fatal("exact external identity rejected")
	}
	remote.LingMirrorTaskVersion = 2
	if verifyRemoteTaskIdentity(task, remote, 11) {
		t.Fatal("stale external task version accepted")
	}
	remote.LingMirrorTaskVersion, remote.Processor = 3, "deterministic"
	if verifyRemoteTaskIdentity(task, remote, 11) {
		t.Fatal("processor mismatch accepted")
	}
}

func TestCreateTaskRequiresExactCurrentInputRights(t *testing.T) {
	db := dbtest.NewDB(t, &canonicalSKU{}, &Asset{}, &Task{}, &RightsGrant{})
	if err := db.Create(&canonicalSKU{ID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	remote := newFakeImageService()
	svc := NewService(db, dbtest.NewLogger(t), remote)
	asset, err := svc.CreateAsset(context.Background(), 11, "input.png", "image/png", []byte("input"))
	if err != nil {
		t.Fatal(err)
	}
	in := validTaskInput(asset, "secured-task")
	if _, err := svc.CreateTask(context.Background(), 11, in); conflictCode(err) != "INPUT_RIGHTS_REQUIRED" {
		t.Fatalf("ungranted upload executed: %v", err)
	}
	assetID := asset.ID
	now := time.Now().UTC()
	wrong := RightsGrant{OwnerID: 11, AssetID: &assetID, AssetSHA: asset.SHA256, CanCopy: true, CanModify: true, Purpose: in.Purpose, Channel: "shopee", Provider: in.Processor, Region: in.Region, Jurisdiction: "ru-ru", Grantor: "owner", RightsChain: "evidence", EvidenceSHA: strings.Repeat("e", 64), OwnerVerified: true, ValidFrom: now.Add(-time.Minute), IdempotencyKey: "wrong-scope", RequestHash: strings.Repeat("a", 64), Version: 1}
	if err := db.Create(&wrong).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateTask(context.Background(), 11, in); conflictCode(err) != "INPUT_RIGHTS_REQUIRED" {
		t.Fatalf("wrong-scope grant executed: %v", err)
	}
	exact := wrong
	exact.ID, exact.Channel, exact.IdempotencyKey = 0, "ozon", "exact-scope"
	if err := db.Create(&exact).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateTask(context.Background(), 11, in); err != nil {
		t.Fatalf("exact rights did not authorize processing: %v", err)
	}
}

func TestChannelNativeImportCannotCrossChannelAfterNewTask(t *testing.T) {
	db := dbtest.NewDB(t, &canonicalSKU{}, &Asset{}, &Task{}, &RightsGrant{}, &ManualImport{})
	if err := db.Create(&canonicalSKU{ID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	remote := newFakeImageService()
	svc := NewService(db, dbtest.NewLogger(t), remote)
	parent, err := svc.CreateAsset(context.Background(), 11, "source.png", "image/png", []byte("source"))
	if err != nil {
		t.Fatal(err)
	}
	in := validManualImport(parent, "native-lineage")
	in.ImportKind, in.OriginalChannel, in.ChannelRestriction = ChannelNativeImportKind, "ozon", "ozon"
	created, err := svc.CreateManualImport(context.Background(), 11, in, "native.png", "image/png", []byte("native"))
	if err != nil {
		t.Fatal(err)
	}
	var asset Asset
	if err := db.First(&asset, created.AssetID).Error; err != nil {
		t.Fatal(err)
	}
	if asset.ParentAssetID == nil || *asset.ParentAssetID != parent.ID || asset.ParentAssetSHA != parent.SHA256 || asset.ChannelRestriction != "ozon" {
		t.Fatalf("import lineage/restriction not persisted: %+v", asset)
	}
	taskInput := validTaskInput(&asset, "cross-channel")
	taskInput.Channel = "shopee"
	if _, err := svc.CreateTask(context.Background(), 11, taskInput); conflictCode(err) != "ASSET_CHANNEL_RESTRICTED" {
		t.Fatalf("channel-native bytes crossed channel: %v", err)
	}
}

func TestRuleSnapshotRejectsEmptyOrNonEnforceableRules(t *testing.T) {
	db := dbtest.NewDB(t, &ImageRuleSnapshot{})
	svc := NewReleaseService(db, nil, strings.Repeat("k", 32), "key")
	base := CreateRuleSnapshotInput{Channel: "ozon", Site: "ru", Locale: "ru-RU", CategoryID: 1, EffectiveAt: time.Now().Add(-time.Minute), IdempotencyKey: "rule"}
	for _, raw := range [][]byte{[]byte(`{}`), []byte(`{"schema_version":"1"}`), []byte(`{"schema_version":"1","max_images":1,"allowed_roles":["main"],"allowed_formats":["gif"],"min_width_px":1,"min_height_px":1,"max_file_bytes":1}`)} {
		base.Rules = raw
		if _, err := svc.CreateRuleSnapshot(context.Background(), 1, base); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("non-enforceable rule accepted: %s err=%v", raw, err)
		}
	}
}

func TestProductImageRoutesRequireOwnerPermission(t *testing.T) {
	db := dbtest.NewDB(t, &rbac.Role{}, &rbac.Permission{}, &rbac.UserRole{}, &rbac.RolePermission{})
	role := rbac.Role{Name: "Administrator", Code: "admin", Status: 1}
	permission := rbac.Permission{Name: "Owner images", Code: "product_image.owner", Module: "product_image"}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&permission).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&rbac.UserRole{UserID: 1, RoleID: role.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&rbac.RolePermission{RoleID: role.ID, PermissionID: permission.ID}).Error; err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		if c.GetHeader("X-Test-Owner") == "yes" {
			c.Set("user_id", int64(1))
		} else {
			c.Set("user_id", int64(2))
		}
	})
	RegisterRoutes(api, db, dbtest.NewLogger(t), (*imageservice.Client)(nil))
	request := func(owner bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/product-images/capabilities", nil)
		if owner {
			req.Header.Set("X-Test-Owner", "yes")
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	if w := request(false); w.Code != http.StatusForbidden {
		t.Fatalf("ordinary authenticated user status=%d body=%s", w.Code, w.Body.String())
	}
	if w := request(true); w.Code != http.StatusOK {
		t.Fatalf("Owner role status=%d body=%s", w.Code, w.Body.String())
	}
}

func conflictCode(err error) string {
	var conflict *ConflictError
	if errors.As(err, &conflict) {
		return conflict.Code
	}
	return ""
}
