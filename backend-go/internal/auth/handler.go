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
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Role        string `json:"role"` // ignored on purpose — hardcoded in Register for security
}

// Register handles new user registration.
// @Summary      Register a new user
// @Description  Create a new operator account. Role is hardcoded to "operator" for security.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  registerRequest  true  "Registration info"
// @Success      200   {object}  response.Result{data=UserVO}
// @Router       /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数无效: "+err.Error())
		return
	}
	// Security: hardcode role to "operator" instead of accepting it from the client.
	// This prevents self-registration as admin or other privileged roles.
	// Role changes must go through the admin user management endpoint (not yet implemented).
	user, err := h.service.Register(req.Username, req.Password, req.DisplayName, req.Email, "operator")
	if err != nil {
		response.Error(c, http.StatusConflict, err.Error())
		return
	}
	response.Success(c, user.ToVO())
}

// Login handles user login.
// @Summary      User login
// @Description  Authenticate with username and password, returns JWT tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  loginRequest  true  "Login credentials"
// @Success      200   {object}  response.Result
// @Router       /auth/login [post]
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

func authenticatedUserID(c *gin.Context) (int64, bool) {
	uid, ok := c.Get("user_id")
	if !ok {
		return 0, false
	}
	userID, ok := uid.(int64)
	return userID, ok && userID > 0
}

// Refresh handles token refresh.
// @Summary      Refresh JWT token
// @Description  Exchange a refresh token for a new access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  refreshRequest  true  "Refresh token"
// @Success      200   {object}  response.Result
// @Router       /auth/refresh [post]
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

// Logout revokes the refresh-token family for the current device/session.
func (h *Handler) Logout(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return
	}
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "refresh_token 不能为空")
		return
	}
	if err := h.service.RevokeRefreshFamily(req.RefreshToken, userID); err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}
	response.Success(c, gin.H{"revoked": true})
}

// LogoutAll revokes every refresh session owned by the authenticated user.
func (h *Handler) LogoutAll(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return
	}
	if err := h.service.RevokeAllRefreshSessions(userID); err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, gin.H{"revoked_all": true})
}

// CurrentUser returns the authenticated user info from the JWT context.
// @Summary      Get current user
// @Description  Return the authenticated user's profile info
// @Tags         auth
// @Produce      json
// @Success      200  {object}  response.Result{data=UserVO}
// @Security     BearerAuth
// @Router       /auth/me [get]
func (h *Handler) CurrentUser(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return
	}

	user, err := h.service.GetUserByID(userID)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "用户不存在")
		return
	}

	response.Success(c, user.ToVO())
}
