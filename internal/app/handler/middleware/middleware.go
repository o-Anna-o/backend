package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"loading_time/internal/app/ds"
	"loading_time/internal/app/repository"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// AuthMiddleware — проверка JWT из куки или Authorization header, кладет в context "user_id" (int) и "role" (string).
func AuthMiddleware(rep *repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. JWT из куки
		tokenStr, err := c.Cookie("jwt")
		if err != nil || tokenStr == "" {
			// 2. JWT из заголовка Authorization
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				// Гость – только GET
				if c.Request.Method != "GET" {
					c.AbortWithStatusJSON(http.StatusUnauthorized,
						gin.H{"error": "не авторизован"})
					return
				}
				c.Next()
				return
			}
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// 3. Парсим JWT
		claims := &ds.JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims,
			func(*jwt.Token) (interface{}, error) { return []byte(rep.JWTKey()), nil })
		if err != nil || !token.Valid {
			if c.Request.Method != "GET" {
				c.AbortWithStatusJSON(http.StatusUnauthorized,
					gin.H{"error": "Invalid token"})
				return
			}
			c.Next()
			return
		}

		// 4. Проверяем, что токен всё ещё в Redis
		idStr := strconv.Itoa(claims.UserID)
		stored, err := rep.Redis().Get(context.Background(), idStr).Result()
		if err != nil || stored != tokenStr {
			if c.Request.Method != "GET" {
				c.AbortWithStatusJSON(http.StatusUnauthorized,
					gin.H{"error": "Token expired"})
				return
			}
			c.Next()
			return
		}

		// 5. Сохраняем в контекст
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// ModeratorMiddleware — требует роль "moderator"
func ModeratorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleAny, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden,
				gin.H{"error": "Moderator access required"})
			return
		}
		if role, ok := roleAny.(string); !ok || role != "moderator" {
			c.AbortWithStatusJSON(http.StatusForbidden,
				gin.H{"error": "Moderator access required"})
			return
		}
		c.Next()
	}
}
