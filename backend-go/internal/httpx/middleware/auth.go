package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lingmirror/backend-go/internal/config"
	"gorm.io/gorm"
)

// Auth returns a JWT authentication middleware.
// If db is non-nil, it also checks that the user exists and is not disabled.
func Auth(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
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
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.JWT.Secret), nil
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

		if userID, exists := claims["user_id"]; exists {
			c.Set("user_id", userID)

			// Check that the user still exists and is not disabled.
			if db != nil {
				var disabled bool
				db.Raw("SELECT status IS DISTINCT FROM 1 FROM \"user\" WHERE id = ?", int64(userID.(float64))).Scan(&disabled)
				if disabled {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
						"code":    403,
						"message": "账号已被禁用",
					})
					return
				}
			}
		}

		c.Next()
	}
}
