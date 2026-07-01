package workflow

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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
	runs, err := h.eng.ListRuns(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, runs)
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

type stepError struct{ msg string }

func (e *stepError) Error() string { return e.msg }
