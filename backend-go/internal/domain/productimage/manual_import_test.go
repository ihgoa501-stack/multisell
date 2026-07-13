package productimage

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/dbtest"
	"gorm.io/gorm"
)

func newManualImportService(t *testing.T) (*Service, *fakeImageService) {
	t.Helper()
	db := dbtest.NewDB(t, &Asset{}, &ManualImport{})
	remote := newFakeImageService()
	return NewService(db, dbtest.NewLogger(t), remote), remote
}

func validManualImport(parent *Asset, key string) ManualImportInput {
	return ManualImportInput{ParentAssetID: parent.ID, ParentAssetSHA: parent.SHA256, ImportKind: ManualImportKind, Tool: "Photoshop", Operation: "removed dust and corrected exposure", FeeAmount: "2.50", FeeCurrency: "USD", Model: "unknown", ModelVersion: "unknown", ChannelRestriction: "*", SourceObservedAt: time.Now().UTC().Add(-time.Minute), IdempotencyKey: key}
}

func TestCreateManualImportPreservesLineageAndIdempotency(t *testing.T) {
	t.Parallel()
	svc, remote := newManualImportService(t)
	parent, err := svc.CreateAsset(context.Background(), 11, "source.png", "image/png", []byte("source bytes"))
	if err != nil {
		t.Fatal(err)
	}
	in := validManualImport(parent, "manual-1")
	created, err := svc.CreateManualImport(context.Background(), 11, in, "edited.png", "image/png", []byte("edited bytes"))
	if err != nil {
		t.Fatal(err)
	}
	replay, err := svc.CreateManualImport(context.Background(), 11, in, "edited.png", "image/png", []byte("edited bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if replay.ID != created.ID || remote.putCalls != 2 { // parent + one imported blob
		t.Fatalf("replay=%+v putCalls=%d", replay, remote.putCalls)
	}
	if created.ParentAssetID != parent.ID || created.ParentAssetSHA != parent.SHA256 || created.Truth != TruthUnknown || created.AssetSHA == parent.SHA256 {
		t.Fatalf("lineage not frozen: %+v", created)
	}
	var output Asset
	if err := svc.db.First(&output, created.AssetID).Error; err != nil {
		t.Fatal(err)
	}
	if output.Truth != TruthUnknown || output.SHA256 != created.AssetSHA {
		t.Fatalf("output=%+v", output)
	}

	changed := in
	changed.Operation = "different edit"
	if _, err := svc.CreateManualImport(context.Background(), 11, changed, "edited.png", "image/png", []byte("edited bytes")); err == nil {
		t.Fatal("same idempotency key accepted different metadata")
	}
}

func TestManualImportOwnerAndChannelNativeRestrictions(t *testing.T) {
	t.Parallel()
	svc, remote := newManualImportService(t)
	parent, err := svc.CreateAsset(context.Background(), 11, "source.png", "image/png", []byte("source"))
	if err != nil {
		t.Fatal(err)
	}
	foreign := validManualImport(parent, "foreign")
	if _, err := svc.CreateManualImport(context.Background(), 22, foreign, "edited.png", "image/png", []byte("edited")); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("foreign parent error=%v", err)
	}
	native := validManualImport(parent, "native")
	native.ImportKind, native.Tool, native.OriginalChannel, native.ChannelRestriction = ChannelNativeImportKind, "Shopify", "shopify", "ozon"
	if _, err := svc.CreateManualImport(context.Background(), 11, native, "edited.png", "image/png", []byte("edited")); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("cross-channel native import error=%v", err)
	}
	if remote.putCalls != 1 {
		t.Fatalf("invalid imports uploaded bytes: %d", remote.putCalls)
	}
}

func TestListManualImportsIsOwnerScopedAndPaginated(t *testing.T) {
	t.Parallel()
	svc, _ := newManualImportService(t)
	for owner := int64(11); owner <= 22; owner += 11 {
		parent, err := svc.CreateAsset(context.Background(), owner, "source.png", "image/png", []byte{byte(owner)})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := svc.CreateManualImport(context.Background(), owner, validManualImport(parent, "key"), "edited.png", "image/png", []byte{byte(owner), 1}); err != nil {
			t.Fatal(err)
		}
	}
	items, total, err := svc.ListManualImports(context.Background(), 11, 1, 1)
	if err != nil || total != 1 || len(items) != 1 || items[0].OwnerID != 11 {
		t.Fatalf("items=%+v total=%d err=%v", items, total, err)
	}
}

func TestManualImportHTTPCreateAndList(t *testing.T) {
	t.Parallel()
	svc, _ := newManualImportService(t)
	parent, err := svc.CreateAsset(context.Background(), 11, "source.png", "image/png", []byte("source"))
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int64(11)); c.Next() })
	r.POST("/manual-imports", h.CreateManualImport)
	r.GET("/manual-imports", h.ListManualImports)
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	file, _ := w.CreateFormFile("file", "edited.png")
	_, _ = file.Write([]byte("edited"))
	fields := map[string]string{"parent_asset_id": "1", "parent_asset_sha256": parent.SHA256, "import_kind": ManualImportKind, "tool": "Photoshop", "operation": "retouch", "fee_amount": "0", "fee_currency": "USD", "model": "unknown", "model_version": "unknown", "channel_restriction": "*", "source_observed_at": time.Now().UTC().Add(-time.Minute).Format(time.RFC3339), "idempotency_key": "http-1"}
	for key, value := range fields {
		_ = w.WriteField(key, value)
	}
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/manual-imports", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusCreated || !bytes.Contains(recorder.Body.Bytes(), []byte(`"truth":"unknown"`)) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	recorder = httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/manual-imports?page=1&size=20", nil))
	if recorder.Code != http.StatusOK || !bytes.Contains(recorder.Body.Bytes(), []byte(`"total":1`)) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
