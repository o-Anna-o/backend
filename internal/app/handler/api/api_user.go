package api

import (
	"context"
	"loading_time/internal/app/ds"
	"loading_time/internal/app/repository"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

type UserHandler struct {
	Repository *repository.Repository
}

// RegisterUserAPI
func (h *UserHandler) RegisterUserAPI(c *gin.Context) {
	var user ds.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	created, err := h.Repository.RegisterUser(user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	created.Password = ""
	c.JSON(http.StatusCreated, gin.H{"data": created})
}

// LoginUserAPI
func (h *UserHandler) LoginUserAPI(c *gin.Context) {
	var cred struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&cred); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, sessionID, err := h.Repository.LoginUser(cred.Login, cred.Password)
	if err != nil {
		// возвращаем точную ошибку для отладки
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// ставим cookie на 24 часа
	expires := 24 * time.Hour
	c.SetCookie("jwt", token, int(expires.Seconds()), "/", "", false, true)
	c.SetCookie("session_id", sessionID, int(expires.Seconds()), "/", "", false, true)

	// отладки: распарсим claims
	claims := jwt.MapClaims{}
	_, _ = jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.Repository.JWTKey()), nil
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"data": gin.H{
			"token":      token,
			"session_id": sessionID,
			"claims":     claims, // временно для проверки
		},
	})
}

// LogoutUserAPI
func (h *UserHandler) LogoutUserAPI(c *gin.Context) {
	token, _ := c.Cookie("jwt")
	sessionID, _ := c.Cookie("session_id")

	if token != "" {
		claims := jwt.MapClaims{}
		if parsed, _ := jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(h.Repository.JWTKey()), nil
		}); parsed != nil {
			if uid, ok := claims["user_id"]; ok {
				switch v := uid.(type) {
				case float64:
					_ = h.Repository.Redis().Del(context.Background(), "jwt:"+strconv.Itoa(int(v))).Err()
				case int:
					_ = h.Repository.Redis().Del(context.Background(), "jwt:"+strconv.Itoa(v)).Err()
				}
			}
		}
	}

	if sessionID != "" {
		_ = h.Repository.Redis().Del(context.Background(), "sess:"+sessionID).Err()
	}

	// очистка cookie
	c.SetCookie("jwt", "", -1, "/", "", false, true)
	c.SetCookie("session_id", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}

// GetUserProfileAPI
func (h *UserHandler) GetUserProfileAPI(c *gin.Context) {
	uid, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var id int
	switch v := uid.(type) {
	case float64:
		id = int(v)
	case int:
		id = v
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user id type"})
		return
	}
	user, err := h.Repository.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	user.Password = ""
	c.JSON(http.StatusOK, user)
}

// UpdateUserProfileAPI
func (h *UserHandler) UpdateUserProfileAPI(c *gin.Context) {
	uid, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var user ds.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	switch v := uid.(type) {
	case float64:
		user.UserID = int(v)
	case int:
		user.UserID = v
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user id type"})
		return
	}
	if err := h.Repository.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Profile updated"})
}
