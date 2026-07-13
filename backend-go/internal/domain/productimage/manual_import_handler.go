package productimage

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
)

func (h *Handler) CreateManualImport(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		problem(c, 422, "VALIDATION_ERROR", "file is required")
		return
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, (10<<20)+1))
	if err != nil || len(body) == 0 || len(body) > 10<<20 {
		problem(c, 422, "VALIDATION_ERROR", "file must be between 1 byte and 10 MiB")
		return
	}
	parentID, err := strconv.ParseInt(c.PostForm("parent_asset_id"), 10, 64)
	observedAt, timeErr := time.Parse(time.RFC3339, c.PostForm("source_observed_at"))
	if err != nil || timeErr != nil {
		problem(c, 422, "VALIDATION_ERROR", "invalid manual import metadata")
		return
	}
	in := ManualImportInput{ParentAssetID: parentID, ParentAssetSHA: c.PostForm("parent_asset_sha256"), ImportKind: c.PostForm("import_kind"), Tool: c.PostForm("tool"), Operation: c.PostForm("operation"), FeeAmount: c.PostForm("fee_amount"), FeeCurrency: c.PostForm("fee_currency"), Model: c.PostForm("model"), ModelVersion: c.PostForm("model_version"), OriginalChannel: c.PostForm("original_channel"), ChannelRestriction: c.PostForm("channel_restriction"), SourceObservedAt: observedAt, IdempotencyKey: c.PostForm("idempotency_key")}
	item, err := h.service.CreateManualImport(c.Request.Context(), owner, in, header.Filename, header.Header.Get("Content-Type"), body)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.Result{Code: 0, Message: "ok", Data: item})
}

func (h *Handler) ListManualImports(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	p := common.ParsePagination(c)
	items, total, err := h.service.ListManualImports(c.Request.Context(), owner, p.Page, p.Size)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}
