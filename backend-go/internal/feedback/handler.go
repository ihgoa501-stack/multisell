package feedback

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
)

// Handler handles feedback HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new feedback Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func parseID(c *gin.Context, field string) (int64, bool) {
	v := c.Param(field)
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return 0, false
	}
	return id, true
}

func getCurrentUserID(c *gin.Context) *int64 {
	uid, exists := c.Get("user_id")
	if !exists {
		return nil
	}
	switch v := uid.(type) {
	case int64:
		return &v
	case float64:
		id := int64(v)
		return &id
	default:
		return nil
	}
}

func getCurrentUserIDRequired(c *gin.Context) (int64, bool) {
	uid := getCurrentUserID(c)
	if uid == nil {
		response.Error(c, http.StatusUnauthorized, "请先登录")
		return 0, false
	}
	return *uid, true
}

// ===================== Projects =====================

func (h *Handler) ListProjects(c *gin.Context) {
	projects, err := h.service.ListProjects()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, projects)
}

func (h *Handler) GetProject(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	p, err := h.service.GetProject(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "项目不存在")
		return
	}
	response.Success(c, p)
}

func (h *Handler) CreateProject(c *gin.Context) {
	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体: "+err.Error())
		return
	}
	p, err := h.service.CreateProject(&req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

func (h *Handler) UpdateProject(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	p, err := h.service.UpdateProject(id, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, p)
}

func (h *Handler) DeleteProject(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteProject(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ===================== Categories =====================

func (h *Handler) ListCategories(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	cats, err := h.service.ListCategories(projectID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, cats)
}

func (h *Handler) CreateCategory(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	cat, err := h.service.CreateCategory(&req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, cat)
}

func (h *Handler) UpdateCategory(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	cat, err := h.service.UpdateCategory(id, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, cat)
}

func (h *Handler) DeleteCategory(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteCategory(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ===================== Tags =====================

func (h *Handler) ListTags(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	tags, err := h.service.ListTags(projectID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, tags)
}

func (h *Handler) CreateTag(c *gin.Context) {
	var req CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	tag, err := h.service.CreateTag(&req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, tag)
}

func (h *Handler) DeleteTag(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteTag(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ===================== Submissions =====================

func (h *Handler) ListSubmissions(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	status := c.Query("status")
	feedbackType := c.Query("feedback_type")
	severity := c.Query("severity")
	p := common.ParsePagination(c)

	subs, total, err := h.service.ListSubmissions(projectID, status, feedbackType, severity, p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, subs, total, p.Page, p.Size)
}

func (h *Handler) ListMySubmissions(c *gin.Context) {
	userID, ok := getCurrentUserIDRequired(c)
	if !ok {
		return
	}
	projectID, _ := strconv.ParseInt(c.Query("project_id"), 10, 64)
	p := common.ParsePagination(c)

	subs, total, err := h.service.ListSubmissionsByUser(userID, projectID, p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, subs, total, p.Page, p.Size)
}

func (h *Handler) GetSubmission(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	userID := getCurrentUserID(c)
	detail, err := h.service.GetSubmissionDetail(id, userID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "反馈不存在")
		return
	}
	response.Success(c, detail)
}

func (h *Handler) CreateSubmission(c *gin.Context) {
	var req CreateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体: "+err.Error())
		return
	}
	userID := getCurrentUserID(c)
	sub, err := h.service.CreateSubmission(&req, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, sub)
}

func (h *Handler) UpdateSubmissionStatus(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	userID, ok2 := getCurrentUserIDRequired(c)
	if !ok2 {
		return
	}
	var req UpdateSubmissionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体: "+err.Error())
		return
	}
	sub, err := h.service.UpdateSubmissionStatus(id, &req, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, sub)
}

func (h *Handler) UpdateSubmission(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req UpdateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体: "+err.Error())
		return
	}
	sub, err := h.service.UpdateSubmission(id, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, sub)
}

func (h *Handler) DeleteSubmission(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteSubmission(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ===================== Votes =====================

func (h *Handler) Vote(c *gin.Context) {
	submissionID, ok := parseID(c, "id")
	if !ok {
		return
	}
	userID, ok2 := getCurrentUserIDRequired(c)
	if !ok2 {
		return
	}
	var req CreateVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	count, err := h.service.Vote(submissionID, userID, req.VoteType)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"vote_count": count, "vote_type": req.VoteType})
}

// ===================== Comments =====================

func (h *Handler) AddComment(c *gin.Context) {
	submissionID, ok := parseID(c, "id")
	if !ok {
		return
	}
	userID, ok2 := getCurrentUserIDRequired(c)
	if !ok2 {
		return
	}
	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求体")
		return
	}
	comment, err := h.service.AddComment(submissionID, userID, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, comment)
}

func (h *Handler) ListComments(c *gin.Context) {
	submissionID, ok := parseID(c, "id")
	if !ok {
		return
	}
	includeInternal := c.Query("include_internal") == "true"
	comments, err := h.service.ListComments(submissionID, includeInternal)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, comments)
}

func (h *Handler) DeleteComment(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteComment(id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

// ===================== Tag Management =====================

func (h *Handler) AddTag(c *gin.Context) {
	submissionID, ok := parseID(c, "id")
	if !ok {
		return
	}
	tagID, ok := parseID(c, "tagId")
	if !ok {
		return
	}
	if err := h.service.AddTagToSubmission(submissionID, tagID); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"submission_id": submissionID, "tag_id": tagID})
}

func (h *Handler) RemoveTag(c *gin.Context) {
	submissionID, ok := parseID(c, "id")
	if !ok {
		return
	}
	tagID, ok := parseID(c, "tagId")
	if !ok {
		return
	}
	if err := h.service.RemoveTagFromSubmission(submissionID, tagID); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"submission_id": submissionID, "tag_id": tagID})
}

// ===================== Dashboard =====================

func (h *Handler) GetDashboardStats(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	stats, err := h.service.GetDashboardStats(projectID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, stats)
}

// ListSubmissionsForAgent returns pending feedback for AI agent consumption.
// GET /api/v1/feedback/pending-for-agent?status=pending&page=1&size=10
func (h *Handler) ListSubmissionsForAgent(c *gin.Context) {
	status := c.DefaultQuery("status", "pending")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	// Try to get the first project
	projects, err := h.service.ListProjects()
	if err != nil || len(projects) == 0 {
		response.Paginated(c, []SubmissionResponse{}, 0, page, size)
		return
	}

	subs, total, err := h.service.ListSubmissions(projects[0].ID, status, "", "", page, size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, subs, total, page, size)
}

func (h *Handler) Migrate(c *gin.Context) {
	if err := AutoMigrate(h.service.db); err != nil {
		response.Error(c, http.StatusInternalServerError, "迁移失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"message": "数据库迁移完成"})
}
