package middleware

import (
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lingmirror/backend-go/internal/config"
)

// Auth returns a JWT authentication middleware.
func Auth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "missing authorization header",
			})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid authorization format",
			})
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrSignatureInvalid
			}
			keyID, _ := t.Header["kid"].(string)
			if keyID == "" {
				keys := []jwt.VerificationKey{[]byte(cfg.JWT.Secret)}
				if previous, parseErr := cfg.JWT.PreviousKeys(); parseErr == nil {
					for _, secret := range previous {
						keys = append(keys, []byte(secret))
					}
				}
				return jwt.VerificationKeySet{Keys: keys}, nil
			}
			if keyID == cfg.JWT.EffectiveKeyID() {
				return []byte(cfg.JWT.Secret), nil
			}
			previous, parseErr := cfg.JWT.PreviousKeys()
			if parseErr != nil {
				return nil, jwt.ErrSignatureInvalid
			}
			secret, ok := previous[keyID]
			if !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid or expired token",
			})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid token claims",
			})
			return
		}

		// Validate token type — reject refresh tokens used as access tokens.
		if tokenType, exists := claims["type"]; !exists || tokenType != "access" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid or expired token",
			})
			return
		}

		userID, exists := claims["user_id"]
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid token claims"})
			return
		}
		var normalizedUserID int64
		switch v := userID.(type) {
		case float64:
			if v <= 0 || v != math.Trunc(v) || v > math.MaxInt64 {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid token claims"})
				return
			}
			normalizedUserID = int64(v)
		case int64:
			normalizedUserID = v
		default:
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid token claims"})
			return
		}
		if normalizedUserID <= 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid token claims"})
			return
		}
		c.Set("user_id", normalizedUserID)

		if username, exists := claims["username"]; exists {
			if s, ok := username.(string); ok {
				c.Set("username", s)
			}
		}

		c.Next()
	}
}
