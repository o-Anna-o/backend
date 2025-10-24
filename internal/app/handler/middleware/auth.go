package middleware

import (
	"net/http"
	"strings"

	"loading_time/internal/app/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware проверяет JWT и допустимые роли (пример как в Lab-4)
func AuthMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization header"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := utils.ParseJWT(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// сохраняем данные в контекст Gin
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)

		// Проверяем роль, если переданы allowedRoles
		if len(allowedRoles) > 0 {
			allowed := false
			for _, r := range allowedRoles {
				if r == claims.Role {
					allowed = true
					break
				}
			}
			if !allowed {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied for role: " + claims.Role})
				return
			}
		}

		c.Next()
	}
}

// ModeratorMiddleware — требует роль "moderator"
func ModeratorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleAny, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Moderator access required"})
			return
		}
		role, ok := roleAny.(string)
		if !ok || role != "moderator" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Moderator access required"})
			return
		}
		c.Next()
	}
}
