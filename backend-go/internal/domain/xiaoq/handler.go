package xiaoq

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/common"
	"github.com/lingmirror/backend-go/internal/response"
	"gorm.io/gorm"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func ownerID(c *gin.Context) (int64, bool) {
	id := common.UserIDFromCtx(c)
	if id == nil || *id <= 0 {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return 0, false
	}
	return *id, true
}

func (h *Handler) Identity(c *gin.Context) {
	if _, ok := ownerID(c); !ok {
		return
	}
	response.Success(c, h.service.Identity())
}

func (h *Handler) Capabilities(c *gin.Context) {
	if _, ok := ownerID(c); !ok {
		return
	}
	response.Success(c, h.service.Capabilities())
}

func (h *Handler) Message(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	var in MessageInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Error(c, http.StatusBadRequest, "message and a supported target are required")
		return
	}
	result, err := h.service.SendMessage(c.Request.Context(), owner, in)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			response.Error(c, http.StatusBadRequest, "invalid message or target")
		case errors.Is(err, gorm.ErrRecordNotFound):
			response.Error(c, http.StatusNotFound, "target not found")
		default:
			var runErr *RunError
			if errors.As(err, &runErr) {
				response.Error(c, http.StatusBadGateway, "AI provider failed; trace_id="+runErr.TraceID)
				return
			}
			response.InternalError(c, err)
		}
		return
	}
	response.Success(c, result)
}

func (h *Handler) Trace(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	traceID := strings.TrimSpace(c.Param("trace_id"))
	detail, err := h.service.GetTrace(c.Request.Context(), owner, traceID)
	if err != nil {
		if errors.Is(err, ErrTraceNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "trace not found")
			return
		}
		response.InternalError(c, err)
		return
	}
	response.Success(c, detail)
}
