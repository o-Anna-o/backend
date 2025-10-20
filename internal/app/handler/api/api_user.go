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

// @Summary Login user
// @Description Authenticate user, set session cookie and return JWT
// @Tags users
// @Accept json
// @Produce json
// @Param credentials body object{login=string,password=string} true "Credentials"
// @Success 200 {object} object "message: string, data: {token: string}"
// @Failure 400 {object} object "error: message"
// @Failure 500 {object} object "error: message"
// @Router /api/users/login [post]
func (h *UserHandler) LoginUserAPI(c *gin.Context) {
	var credentials struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := c.BindJSON(&credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	user := ds.User{Login: credentials.Login, Password: credentials.Password}
	token, err := h.Repository.LoginUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	claims := jwt.MapClaims{}
	_, _ = jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.Repository.JWTKey()), nil
	})
	expiresAt := time.Unix(int64(claims["exp"].(float64)), 0)
	c.SetCookie("jwt", token, int(time.Until(expiresAt).Seconds()), "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"data":    gin.H{"token": token},
	})
}

// @Summary Logout user
// @Description Clear session cookie and remove JWT from Redis
// @Tags users
// @Produce json
// @Success 200 {object} object "message: string"
// @Failure 400 {object} object "error: message"
// @Router /api/users/logout [post]
func (h *UserHandler) LogoutUserAPI(c *gin.Context) {
	token, err := c.Cookie("jwt")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No session cookie found",
		})
		return
	}
	claims := jwt.MapClaims{}
	parsedToken, _ := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.Repository.JWTKey()), nil
	})
	if !parsedToken.Valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid token",
		})
		return
	}
	userID := int(claims["user_id"].(float64))
	idStr := strconv.Itoa(userID)
	err = h.Repository.Redis().Del(context.Background(), idStr).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.SetCookie("jwt", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"message": "Logout successful",
	})
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
