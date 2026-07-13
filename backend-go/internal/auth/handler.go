package auth

import (
	"net/http"
	"strconv"

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

type extensionPairingCreateRequest struct {
	Environment string `json:"environment" binding:"required"`
}
type extensionPairingClaimRequest struct {
	Nonce        string `json:"nonce" binding:"required"`
	ClaimSecret  string `json:"claim_secret" binding:"required"`
	DeviceID     string `json:"device_id" binding:"required"`
	ExtensionID  string `json:"extension_id" binding:"required"`
	Environment  string `json:"environment" binding:"required"`
	BrowserLabel string `json:"browser_label" binding:"required"`
}
type extensionPairingExchangeRequest struct {
	Nonce       string `json:"nonce" binding:"required"`
	ClaimSecret string `json:"claim_secret" binding:"required"`
}
type extensionDeviceRefreshRequest struct {
	DeviceID     string `json:"device_id" binding:"required"`
	DeviceSecret string `json:"device_secret" binding:"required"`
	Environment  string `json:"environment" binding:"required"`
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
	if err := h.service.RevokeAllExtensionDevices(userID); err != nil {
		response.InternalError(c, err)
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
	if err := h.service.RevokeAllExtensionDevices(userID); err != nil {
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

func (h *Handler) CreateExtensionPairing(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return
	}
	var req extensionPairingCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "环境不能为空")
		return
	}
	result, err := h.service.CreateExtensionPairing(userID, h.cfg.Server.EffectiveDeploymentEnvironment())
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *Handler) ClaimExtensionPairing(c *gin.Context) {
	var req extensionPairingClaimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "浏览器身份不完整")
		return
	}
	if err := h.service.ClaimExtensionPairing(req.Nonce, req.ClaimSecret, req.DeviceID, req.ExtensionID, req.Environment, req.BrowserLabel); err != nil {
		response.Error(c, http.StatusConflict, err.Error())
		return
	}
	response.Success(c, gin.H{"claimed": true})
}

func (h *Handler) GetExtensionPairing(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return
	}
	pairingID, err := strconv.ParseInt(c.Param("pairingId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid pairing id")
		return
	}
	row, err := h.service.GetExtensionPairing(userID, pairingID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "pairing not found")
		return
	}
	response.Success(c, row)
}

func (h *Handler) ConfirmExtensionPairing(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return
	}
	pairingID, err := strconv.ParseInt(c.Param("pairingId"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid pairing id")
		return
	}
	if err := h.service.ConfirmExtensionPairing(userID, pairingID); err != nil {
		response.Error(c, http.StatusConflict, err.Error())
		return
	}
	response.Success(c, gin.H{"confirmed": true})
}

func (h *Handler) ExchangeExtensionPairing(c *gin.Context) {
	var req extensionPairingExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "配对凭据不完整")
		return
	}
	result, err := h.service.ExchangeExtensionPairing(req.Nonce, req.ClaimSecret)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *Handler) RefreshExtensionDevice(c *gin.Context) {
	var req extensionDeviceRefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "设备凭据不完整")
		return
	}
	result, err := h.service.RefreshExtensionDevice(req.DeviceID, req.DeviceSecret, req.Environment)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *Handler) ListExtensionDevices(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return
	}
	rows, err := h.service.ListExtensionDevices(userID)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, rows)
}

func (h *Handler) RevokeExtensionDevice(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "not authenticated")
		return
	}
	if err := h.service.RevokeExtensionDevice(userID, c.Param("deviceId")); err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}
	response.Success(c, gin.H{"revoked": true})
}
