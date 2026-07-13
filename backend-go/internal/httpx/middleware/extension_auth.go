package middleware

import (
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lingmirror/backend-go/internal/config"
	"gorm.io/gorm"
)

// ExtensionAuth accepts only short-lived, device-bound extension tokens with
// the requested scope. Normal web access and refresh tokens are rejected.
func ExtensionAuth(cfg *config.Config, db *gorm.DB, requiredScope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "插件尚未配对或凭证已失效"})
			return
		}
		token, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrSignatureInvalid
			}
			kid, _ := t.Header["kid"].(string)
			if kid == "" || kid == cfg.JWT.EffectiveKeyID() {
				return []byte(cfg.JWT.Secret), nil
			}
			previous, parseErr := cfg.JWT.PreviousKeys()
			if parseErr != nil {
				return nil, jwt.ErrSignatureInvalid
			}
			secret, ok := previous[kid]
			if !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "插件凭证无效或已过期"})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || claims["type"] != "extension_access" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "该凭证不能用于插件采集"})
			return
		}
		uidFloat, ok := claims["user_id"].(float64)
		if !ok || uidFloat <= 0 || uidFloat != math.Trunc(uidFloat) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "插件Owner身份无效"})
			return
		}
		deviceID, _ := claims["device_id"].(string)
		environment, _ := claims["environment"].(string)
		issuer, issuerErr := claims.GetIssuer()
		audience, audienceErr := claims.GetAudience()
		audienceOK := false
		for _, value := range audience {
			if value == "lingmirror-sourcing1688" {
				audienceOK = true
				break
			}
		}
		if issuerErr != nil || audienceErr != nil || environment != cfg.Server.EffectiveDeploymentEnvironment() || issuer != "lingmirror-extension:"+environment || !audienceOK {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "插件凭证环境或用途无效"})
			return
		}
		scopes, _ := claims["scopes"].([]interface{})
		allowed := false
		for _, value := range scopes {
			if scope, ok := value.(string); ok && scope == requiredScope {
				allowed = true
				break
			}
		}
		if deviceID == "" || environment == "" || !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "插件没有此操作权限"})
			return
		}
		var active int64
		err = db.Raw(`SELECT COUNT(*) FROM extension_device ed JOIN "user" u ON u.id = ed.user_id
			WHERE ed.device_id = ? AND ed.user_id = ? AND ed.environment = ? AND ed.scope = ? AND ed.revoked_at IS NULL AND u.status = 1 AND u.role IN ('owner','admin')`,
			deviceID, int64(uidFloat), environment, requiredScope).Scan(&active).Error
		if err != nil || active != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "此浏览器连接已撤销或Owner账号不可用"})
			return
		}
		c.Set("user_id", int64(uidFloat))
		c.Set("extension_device_id", deviceID)
		c.Set("extension_environment", environment)
		c.Next()
	}
}
