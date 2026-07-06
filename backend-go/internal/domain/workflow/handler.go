package workflow

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
)

type Handler struct {
	eng *Engine
}

func NewHandler(eng *Engine) *Handler {
	return &Handler{eng: eng}
}

// ── Def CRUD ─────────────────────────────────────────────────────────

func (h *Handler) ListDefs(c *gin.Context) {
	defs, err := h.eng.ListDefs(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, defs)
}

func (h *Handler) GetDef(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	def, err := h.eng.GetDef(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}
	response.Success(c, def)
}

func (h *Handler) CreateDef(c *gin.Context) {
	var def WorkflowDef
	if err := c.ShouldBindJSON(&def); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// Validate steps JSON.
	var steps []StepDef
	if def.Steps != "" {
		if err := json.Unmarshal([]byte(def.Steps), &steps); err != nil {
			response.Error(c, http.StatusBadRequest, "invalid steps JSON: "+err.Error())
			return
		}
	}
	if len(steps) == 0 {
		response.Error(c, http.StatusBadRequest, "at least one step required")
		return
	}

	if err := h.eng.CreateDef(c.Request.Context(), &def); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, def)
}

func (h *Handler) UpdateDef(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	var def WorkflowDef
	if err := c.ShouldBindJSON(&def); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	def.ID = id

	if err := h.eng.UpdateDef(c.Request.Context(), &def); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, def)
}

func (h *Handler) DeleteDef(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.eng.DeleteDef(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"deleted": id})
}

// ── Run lifecycle ────────────────────────────────────────────────────

func (h *Handler) StartRun(c *gin.Context) {
	defID, err := strconv.ParseInt(c.Param("defId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid def id")
		return
	}

	var req struct {
		Context map[string]interface{} `json:"context"`
	}
	c.ShouldBindJSON(&req)

	run, err := h.eng.StartRun(c.Request.Context(), defID, req.Context)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, run)
}

func (h *Handler) PauseRun(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid run id")
		return
	}
	if err := h.eng.PauseRun(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"status": "paused"})
}

func (h *Handler) ResumeRun(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid run id")
		return
	}
	if err := h.eng.ResumeRun(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"status": "resumed"})
}

func (h *Handler) ListRuns(c *gin.Context) {
	var workflowID *int64
	if widStr := c.Query("workflow_id"); widStr != "" {
		if wid, err := strconv.ParseInt(widStr, 10, 64); err == nil {
			workflowID = &wid
		}
	}

	p := common.ParsePagination(c)
	runs, total, err := h.eng.ListRunsFiltered(c.Request.Context(), workflowID, p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, runs, total, p.Page, p.Size)
}

func (h *Handler) GetRun(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid run id")
		return
	}
	run, err := h.eng.GetRun(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}
	response.Success(c, run)
}

// AdvanceStep allows external callers (e.g., event subscribers) to report step completion.
func (h *Handler) AdvanceStep(c *gin.Context) {
	runID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid run id")
		return
	}

	var req struct {
		StepName string                 `json:"step_name"`
		Output   map[string]interface{} `json:"output"`
		Error    string                 `json:"error,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	var execErr error
	if req.Error != "" {
		execErr = &stepError{req.Error}
	}

	if err := h.eng.AdvanceStep(c.Request.Context(), runID, req.StepName, req.Output, execErr); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"advanced": req.StepName})
}

func (h *Handler) GetMonitorStats(c *gin.Context) {
	stats, err := h.eng.GetMonitorStats(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, stats)
}

// GetMonitor returns per-status counts plus 24h completions.
func (h *Handler) GetMonitor(c *gin.Context) {
	stats, err := h.eng.GetMonitor(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, stats)
}

func (h *Handler) GetRunsStatusDistribution(c *gin.Context) {
	stats, err := h.eng.GetMonitorStats(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{
		"by_status":          stats.ByStatus,
		"average_duration_s": stats.AverageDurationS,
		"failure_by_step":    stats.FailureByStep,
	})
}

// RetryRun resets a failed run and attempts to re-execute.
func (h *Handler) RetryRun(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid run id")
		return
	}
	if err := h.eng.RetryRun(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"status": "retrying"})
}

type stepError struct{ msg string }

func (e *stepError) Error() string { return e.msg }

// ── Workflow (plural) endpoints — M5.1 ──────────────────────────────

// ListWorkflows returns a paginated list of workflow definitions.
func (h *Handler) ListWorkflows(c *gin.Context) {
	p := common.ParsePagination(c)
	defs, total, err := h.eng.ListDefsPaginated(c.Request.Context(), p.Page, p.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Paginated(c, defs, total, p.Page, p.Size)
}

// GetWorkflow returns a single workflow def with its nodes.
func (h *Handler) GetWorkflow(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	def, err := h.eng.GetDef(c.Request.Context(), int64(id))
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}

	// Fetch nodes associated with this workflow.
	nodes, _ := h.eng.ListNodes(c.Request.Context(), uint(id))

	response.Success(c, gin.H{
		"workflow": def,
		"nodes":    nodes,
	})
}

// CreateWorkflow creates a new workflow definition (with optional nodes).
func (h *Handler) CreateWorkflow(c *gin.Context) {
	var req struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Steps       string         `json:"steps"`
		Nodes       []WorkflowNode `json:"nodes,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Name == "" {
		response.Error(c, http.StatusBadRequest, "name is required")
		return
	}

	// Validate steps JSON if provided.
	var steps []StepDef
	if req.Steps != "" {
		if err := json.Unmarshal([]byte(req.Steps), &steps); err != nil {
			response.Error(c, http.StatusBadRequest, "invalid steps JSON: "+err.Error())
			return
		}
	}

	def := &WorkflowDef{
		Name:        req.Name,
		Description: req.Description,
		Steps:       req.Steps,
	}
	if len(steps) == 0 {
		def.Steps = "[]"
	}

	if err := h.eng.CreateDef(c.Request.Context(), def); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Create nodes if provided.
	for i := range req.Nodes {
		req.Nodes[i].WorkflowID = uint(def.ID)
		req.Nodes[i].OrderIndex = i
		if err := h.eng.CreateNode(c.Request.Context(), &req.Nodes[i]); err != nil {
			response.Error(c, http.StatusInternalServerError, "node create failed: "+err.Error())
			return
		}
	}

	response.Success(c, gin.H{
		"workflow": def,
		"nodes":    req.Nodes,
	})
}

// ── Approval endpoints — M5.2 ───────────────────────────────────────

// ApproveStep approves a pending approval step in a run.
func (h *Handler) ApproveStep(c *gin.Context) {
	runID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid run id")
		return
	}

	var req struct {
		StepName string `json:"step_name"`
		Reviewer string `json:"reviewer"`
		Comment  string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.StepName == "" {
		response.Error(c, http.StatusBadRequest, "step_name is required")
		return
	}

	if err := h.eng.ApproveStep(c.Request.Context(), runID, req.StepName, req.Reviewer, req.Comment); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"approved": req.StepName})
}

// RejectStep rejects a pending approval step in a run.
func (h *Handler) RejectStep(c *gin.Context) {
	runID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid run id")
		return
	}

	var req struct {
		StepName string `json:"step_name"`
		Reviewer string `json:"reviewer"`
		Comment  string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.StepName == "" {
		response.Error(c, http.StatusBadRequest, "step_name is required")
		return
	}

	if err := h.eng.RejectStep(c.Request.Context(), runID, req.StepName, req.Reviewer, req.Comment); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"rejected": req.StepName})
}
