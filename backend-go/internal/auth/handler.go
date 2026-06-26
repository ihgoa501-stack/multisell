package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/config"
	"github.com/lingmirror/backend-go/internal/response"
	"go.uber.org/zap"
)

// Handler handles auth HTTP requests.
type Handler struct {
	service *Service
	cfg     *config.Config
	logger  *zap.Logger
}

// NewHandler creates a new auth handler.
func NewHandler(service *Service, cfg *config.Config, logger *zap.Logger) *Handler {
	return &Handler{service: service, cfg: cfg, logger: logger}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type registerRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=100"`
	Password    string `json:"password" binding:"required,min=6"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Role        string `json:"role"` // optional, defaults to "user"
}

// Register handles new user registration.
// On success, returns access+refresh tokens and the user object
// so the frontend can immediately proceed to an authenticated page.
func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数无效: "+err.Error())
		return
	}
	if req.Role != "" && req.Role != "user" {
		// Only "user" role is allowed via registration; elevated roles
		// must be assigned by an admin after creation.
		req.Role = "user"
	}
	user, err := h.service.Register(req.Username, req.Password, req.DisplayName, req.Email, "user")
	if err != nil {
		response.Error(c, http.StatusConflict, err.Error())
		return
	}

	accessToken, err := h.service.GenerateAccessToken(user)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "令牌生成失败")
		return
	}
	refreshToken, err := h.service.GenerateRefreshToken(user)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "令牌生成失败")
		return
	}

	response.Success(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "bearer",
		"expires_in":    h.cfg.JWT.ExpiryHours * 3600,
		"user":          user.ToVO(),
	})
}

// Login handles user login.
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "用户名和密码不能为空")
		return
	}

	accessToken, refreshToken, userVO, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	response.Success(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "bearer",
		"expires_in":    h.cfg.JWT.ExpiryHours * 3600,
		"user":          userVO,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh handles token refresh.
func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "refresh_token 不能为空")
		return
	}

	accessToken, refreshToken, userVO, err := h.service.Refresh(req.RefreshToken)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	response.Success(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "bearer",
		"expires_in":    h.cfg.JWT.ExpiryHours * 3600,
		"user":          userVO,
	})
}

// CurrentUser returns the authenticated user info from the JWT context.
func (h *Handler) CurrentUser(c *gin.Context) {
	uid, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return
	}

	var userID int64
	switch v := uid.(type) {
	case float64:
		userID = int64(v)
	case int64:
		userID = v
	case int:
		userID = int64(v)
	default:
		response.Error(c, http.StatusUnauthorized, "invalid user identity")
		return
	}

	user, err := h.service.GetUserByID(userID)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "用户不存在")
		return
	}

	response.Success(c, user.ToVO())
}
