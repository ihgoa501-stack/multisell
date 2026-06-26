package importbatch

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Handler handles importbatch HTTP requests.
type Handler struct {
	service   *Service
	processor *ProcessBatchDispatcher
	logger    *zap.Logger
}

// NewHandler creates a new importbatch handler.
func NewHandler(service *Service, db *gorm.DB, logger *zap.Logger) *Handler {
	return &Handler{
		service:   service,
		processor: NewProcessBatchDispatcher(db, logger),
		logger:    logger,
	}
}

func parseIBID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return 0, false
	}
	return id, true
}

// ListBatches GET /api/v1/importbatch
func (h *Handler) ListBatches(c *gin.Context) {
	p := common.ParsePagination(c)
	items, total, err := h.service.ListBatches(c.Query("source_type"), c.Query("status"), p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}

// GetBatch GET /api/v1/importbatch/:id
func (h *Handler) GetBatch(c *gin.Context) {
	id, ok := parseIBID(c)
	if !ok {
		return
	}
	b, err := h.service.GetBatch(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "批次不存在")
		return
	}
	response.Success(c, b)
}

// CreateBatch POST /api/v1/importbatch
func (h *Handler) CreateBatch(c *gin.Context) {
	var b ImportBatch
	if err := c.ShouldBindJSON(&b); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	if b.SourceType == "" {
		response.Error(c, http.StatusBadRequest, "source_type 不能为空")
		return
	}
	if err := h.service.CreateBatch(&b); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, b)
}

// UpdateBatch PUT /api/v1/importbatch/:id
func (h *Handler) UpdateBatch(c *gin.Context) {
	id, ok := parseIBID(c)
	if !ok {
		return
	}
	b, err := h.service.GetBatch(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "批次不存在")
		return
	}
	if err := c.ShouldBindJSON(b); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	b.ID = id
	if err := h.service.UpdateBatch(b); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, b)
}

// DeleteBatch DELETE /api/v1/importbatch/:id
func (h *Handler) DeleteBatch(c *gin.Context) {
	id, ok := parseIBID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteBatch(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// Upload POST /api/v1/importbatch/upload
func (h *Handler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "请上传文件")
		return
	}

	batchType := c.PostForm("source_type")
	if batchType == "" {
		response.Error(c, http.StatusBadRequest, "source_type 不能为空")
		return
	}

	// Validate file format.
	if _, err := DetectFormat(file.Filename); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// Save file to uploads/ directory.
	ext := filepath.Ext(file.Filename)
	savedName := fmt.Sprintf("%s_%s%s", batchType, uuid.New().String()[:8], ext)
	savePath := filepath.Join("uploads", savedName)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		response.Error(c, http.StatusInternalServerError, "文件保存失败")
		return
	}

	// Create batch record.
	batch := &ImportBatch{
		SourceType: batchType,
		FileName:  savedName,
		Status:    "pending",
		CreatedBy: c.GetString("current_user"),
		CreatedAt: time.Now(),
	}
	if err := h.service.CreateBatch(batch); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Process asynchronously.
	batchID := batch.ID
	go func() {
		if err := h.processor.ProcessBatch(batchID); err != nil {
			h.logger.Error("process import batch", zap.Int64("batch_id", batchID), zap.Error(err))
		}
	}()

	response.Success(c, batch)
}

// ListRows GET /api/v1/importbatch/:id/rows
func (h *Handler) ListRows(c *gin.Context) {
	id, ok := parseIBID(c)
	if !ok {
		return
	}
	p := common.ParsePagination(c)
	items, total, err := h.service.ListRows(id, c.Query("status"), p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, items, total, p.Page, p.Size)
}
