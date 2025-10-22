package middleware

import (
	"context"
	"loading_time/internal/app/repository"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// AuthMiddleware - проверка JWT из куки или header, сохранение user_id/is_moderator в контексте
func AuthMiddleware(rep *repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Проверяем куки
		tokenStr, err := c.Cookie("jwt")
		if err != nil {
			// Проверяем header
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				// Guest: продолжаем без user
				c.Next()
				return
			}
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header"})
				c.Abort()
				return
			}
		}

		// Парсим JWT
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return []byte(rep.JWTKey()), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		userID := int(claims["user_id"].(float64))
		isModerator := claims["is_moderator"].(bool)

		// Проверяем токен в Redis
		idStr := strconv.Itoa(userID)
		storedToken, err := rep.Redis().Get(context.Background(), idStr).Result()
		if err != nil || storedToken != tokenStr {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expired or invalid"})
			c.Abort()
			return
		}

		// Сохраняем в контексте
		c.Set("user_id", userID)
		c.Set("is_moderator", isModerator)
		c.Next()
	}
}

// ModeratorMiddleware - проверка на модератора
func ModeratorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		isModerator, exists := c.Get("is_moderator")
		if !exists || !isModerator.(bool) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Moderator access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}
