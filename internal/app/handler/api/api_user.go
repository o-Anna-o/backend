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

// @Summary Register a new user
// @Description Register a new user with login and password
// @Tags users
// @Accept json
// @Produce json
// @Param user body ds.User true "User info"
// @Success 201 {object} object "data: registered user"
// @Failure 400 {object} object "error: message"
// @Failure 500 {object} object "error: message"
// @Router /api/users/register [post]
func (h *UserHandler) RegisterUserAPI(c *gin.Context) {
	var user ds.User
	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	registeredUser, err := h.Repository.RegisterUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": registeredUser,
	})
}

// LoginUserAPI — обновлён (ожидаем token, sessionID, err)

// @Summary Login user
// @Description Authenticate user, set session cookie and return JWT
// @Tags users
// @Accept json
// @Produce json
// @Param credentials body object{login=string,password=string} true "Credentials"
// @Success 200 {object} object "message: string, data: {token: string}"
// @Failure 400 {object} object "error: message"
// @Failure 401 {object} object "error: неверный логин или пароль"
// @Failure 500 {object} object "error: message"
// @Router /api/users/login [post]
func (h *UserHandler) LoginUserAPI(c *gin.Context) {
	var credentials struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := c.BindJSON(&credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, sessionID, err := h.Repository.LoginUser(credentials.Login, credentials.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	// Получаем exp из токена, чтобы установить ttl куки корректно
	claims := &jwt.MapClaims{}
	_, _ = jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.Repository.JWTKey()), nil
	})
	// безопасно получить exp
	var expiresAt time.Time
	if expV, ok := (*claims)["exp"]; ok {
		switch v := expV.(type) {
		case float64:
			expiresAt = time.Unix(int64(v), 0)
		case int64:
			expiresAt = time.Unix(v, 0)
		}
	} else {
		// fallback - 1 hour
		expiresAt = time.Now().Add(time.Hour)
	}

	// ставим cookie jwt
	c.SetCookie("jwt", token, int(time.Until(expiresAt).Seconds()), "/", "", false, true)

	// и cookie с session id (если sessionID не пустой)
	if sessionID != "" {
		// ставим ту же TTL
		c.SetCookie("session_id", sessionID, int(time.Until(expiresAt).Seconds()), "/", "", false, true)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"data":    gin.H{"token": token, "session_id": sessionID},
	})
}

// LogoutUserAPI — удаляем jwt (по user_id) и session (по session_id cookie)

// @Summary Logout user
// @Description Clear session cookie and remove JWT from Redis
// @Tags users
// @Produce json
// @Success 200 {object} object "message: string"
// @Failure 400 {object} object "error: message"
// @Router /api/users/logout [post]
func (h *UserHandler) LogoutUserAPI(c *gin.Context) {
	// пытаемся получить jwt из cookie
	token, _ := c.Cookie("jwt")
	// пытаемся получить session_id
	sessionID, _ := c.Cookie("session_id")

	// если есть jwt — удаляем его связку в Redis (репо использует user_id -> token)
	if token != "" {
		claims := jwt.MapClaims{}
		parsedToken, _ := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(h.Repository.JWTKey()), nil
		})
		if parsedToken != nil && parsedToken.Valid {
			if uidV, ok := claims["user_id"]; ok {
				switch v := uidV.(type) {
				case float64:
					userID := int(v)
					idStr := strconv.Itoa(userID)
					_ = h.Repository.Redis().Del(context.Background(), idStr).Err()
				case int:
					idStr := strconv.Itoa(v)
					_ = h.Repository.Redis().Del(context.Background(), idStr).Err()
				}
			}
		}
	}

	// если есть session_id — удаляем соответствующую запись (репо должна хранить sess:<sessionID>)
	if sessionID != "" {
		_ = h.Repository.Redis().Del(context.Background(), "sess:"+sessionID).Err()
	}

	// очистка cookie
	c.SetCookie("jwt", "", -1, "/", "", false, true)
	c.SetCookie("session_id", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}

// @Summary Get user profile
// @Description Get profile of the authenticated user
// @Tags users
// @Produce json
// @Success 200 {object} ds.User
// @Failure 401 {object} object "error: message"
// @Failure 500 {object} object "error: message"
// @Router /api/users/profile [get]
func (h *UserHandler) GetUserProfileAPI(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var id int
	if floatID, ok := userID.(float64); ok {
		id = int(floatID)
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}
	user, err := h.Repository.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

// @Summary Update user profile
// @Description Update profile of the authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Param user body ds.User true "User info"
// @Success 200 {object} object "message: string"
// @Failure 400 {object} object "error: message"
// @Failure 401 {object} object "error: message"
// @Failure 500 {object} object "error: message"
// @Router /api/users/profile [put]
func (h *UserHandler) UpdateUserProfileAPI(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var user ds.User
	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Безопасное преобразование userID из float64 в int
	if floatID, ok := userID.(float64); ok {
		user.UserID = int(floatID)
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}
	if err := h.Repository.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Profile updated"})
}
