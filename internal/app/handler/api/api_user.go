package api

import (
	"context" // для Redis
	"loading_time/internal/app/ds"
	"loading_time/internal/app/repository"
	"net/http"
	"strconv" // для Logout
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4" // для JWT parsing в Logout
)

type UserHandler struct {
	Repository *repository.Repository
}

// @Summary Get user profile
// @Description Get profile fields for logged-in user
// @Tags users
// @Produce json
// @Success 200 {object} gin.H
// @Failure 404 {object} gin.H
// @Router /api/users/profile [get]
func (h *UserHandler) GetUserProfileAPI(c *gin.Context) {
	const fixedUserID = 1

	var user ds.User

	db := h.Repository.DB()

	err := db.First(&user, fixedUserID).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"description": "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   user,
	})
}

// @Summary Update user profile
// @Description Update profile fields
// @Tags users
// @Accept json
// @Produce json
// @Param updates body object {fio=string,contacts=string,cargo_weight=number,containers_20ft_count=integer,containers_40ft_count=integer} true "Updates"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/users/profile [put]
func (h *UserHandler) UpdateUserProfileAPI(c *gin.Context) {
	const fixedUserID = 1

	var updates struct {
		FIO                 string  `json:"fio"`
		Contacts            string  `json:"contacts"`
		CargoWeight         float64 `json:"cargo_weight"`
		Containers20ftCount int     `json:"containers_20ft_count"`
		Containers40ftCount int     `json:"containers_40ft_count"`
	}

	if err := c.BindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	db := h.Repository.DB()
	err := db.Model(&ds.User{}).Where("user_id = ?", fixedUserID).Updates(updates).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
	})
}

// @Summary Register a new user
// @Description Register a new user with login and password (password hashed automatically)
// @Tags users
// @Accept json
// @Produce json
// @Param user body ds.User true "User info"
// @Success 201 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/users/register [post]
func (h *UserHandler) RegisterUserAPI(c *gin.Context) {
	var user ds.User
	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	registeredUser, err := h.Repository.RegisterUser(user) // должно работать, если repository/user.go правильный
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
// @Param credentials body object {login=string,password=string} true "Credentials"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
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

	token, err := h.Repository.LoginUser(user) // должно работать
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.SetCookie("jwt", token, int((time.Hour).Seconds()), "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"data": gin.H{
			"token": token,
		},
	})
}

// @Summary Logout user
// @Description Logout, delete session and cookie
// @Tags users
// @Produce json
// @Success 200 {object} gin.H
// @Router /api/users/logout [post]
func (h *UserHandler) LogoutUserAPI(c *gin.Context) {
	token, err := c.Cookie("jwt")
	if err == nil && token != "" {
		parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			return []byte("SuperSecretKey"), nil
		})
		if err == nil && parsedToken.Valid {
			claims := parsedToken.Claims.(jwt.MapClaims)
			userID := int(claims["user_id"].(float64))
			idStr := strconv.Itoa(userID)
			h.Repository.Redis().Del(context.Background(), idStr)
		}
		c.SetCookie("jwt", "", -1, "/", "", false, true)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logout successful",
	})
}
