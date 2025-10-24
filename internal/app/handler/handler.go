package handler

import (
	"loading_time/internal/app/handler/api"
	"loading_time/internal/app/handler/middleware"
	"loading_time/internal/app/repository"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	Repository            *repository.Repository
	ShipAPIHandler        *api.ShipHandler
	RequestShipAPIHandler *api.RequestShipHandler
	UserAPIHandler        *api.UserHandler
}

func NewHandler(rep *repository.Repository) *Handler {
	return &Handler{
		Repository:            rep,
		ShipAPIHandler:        &api.ShipHandler{Repository: rep},
		RequestShipAPIHandler: &api.RequestShipHandler{Repository: rep},
		UserAPIHandler:        &api.UserHandler{Repository: rep},
	}
}

func (h *Handler) SetupRoutes(router *gin.Engine) {
	router.GET("/ship/:id", h.GetShip)
	router.GET("/request_ship", h.CreateOrRedirectRequestShip)
	router.GET("/request_ship/:id", h.GetRequestShip)
	router.POST("/request_ship/calculate_loading_time/:id", h.CalculateLoadingTime)
	router.GET("/ships", h.GetShips)

	// API маршруты
	apiGroup := router.Group("/api")
	{
		//  1. ГОСТЬ: Чтение + регистрация/вход
		apiGroup.GET("/ships", h.ShipAPIHandler.GetShipsAPI)
		apiGroup.GET("/ships/:id", h.ShipAPIHandler.GetShipAPI)
		apiGroup.GET("/request_ship/basket", h.RequestShipAPIHandler.GetRequestShipBasketAPI)

		// Регистрация и вход — ГОСТЬ
		apiGroup.POST("/users/register", h.UserAPIHandler.RegisterUserAPI)
		apiGroup.POST("/users/login", h.UserAPIHandler.LoginUserAPI)

		//  2. АВТОРИЗОВАННЫЕ (creator + moderator)
		authGroup := apiGroup.Group("", middleware.AuthMiddleware(h.Repository))
		{
			// УСЛУГИ
			authGroup.POST("/ships", h.ShipAPIHandler.CreateShipAPI)
			authGroup.PUT("/ships/:id", h.ShipAPIHandler.UpdateShipAPI)
			authGroup.DELETE("/ships/:id", h.ShipAPIHandler.DeleteShipAPI)
			authGroup.POST("/ships/:id/add-to-ship-bucket", h.ShipAPIHandler.AddShipToRequestShipAPI)
			authGroup.POST("/ships/:id/image", h.ShipAPIHandler.AddShipImageAPI)

			// ЗАЯВКИ
			authGroup.GET("/request_ship", h.RequestShipAPIHandler.GetRequestShipsAPI)
			authGroup.GET("/request_ship/:id", h.RequestShipAPIHandler.GetRequestShipAPI)
			authGroup.PUT("/request_ship/:id", h.RequestShipAPIHandler.UpdateRequestShipAPI)
			authGroup.PUT("/request_ship/:id/formation", h.RequestShipAPIHandler.FormRequestShipAPI)
			authGroup.DELETE("/request_ship/:id", h.RequestShipAPIHandler.DeleteRequestShipAPI)

			// М-М
			authGroup.PUT("/request_ship/:id/ships/:ship_id", h.RequestShipAPIHandler.UpdateShipInRequestAPI)
			authGroup.DELETE("/request_ship/:id/ships/:ship_id", h.RequestShipAPIHandler.DeleteShipFromRequestShipAPI)

			// ПРОФИЛЬ
			authGroup.POST("/users/logout", h.UserAPIHandler.LogoutUserAPI)
			authGroup.GET("/users/profile", h.UserAPIHandler.GetUserProfileAPI)
			authGroup.PUT("/users/profile", h.UserAPIHandler.UpdateUserProfileAPI)
		}

		//  3. ТОЛЬКО МОДЕРАТОР
		modGroup := apiGroup.Group("", middleware.ModeratorMiddleware())
		{
			modGroup.PUT("/request_ship/:id/completion", h.RequestShipAPIHandler.CompleteRequestShipAPI)
		}
	}
}

func (h *Handler) RegisterStatic(router *gin.Engine) {
	router.Static("/styles", "./resources/styles")
	router.Static("/img", "./resources/img")
}

func (h *Handler) errorHandler(c *gin.Context, code int, err error) {
	logrus.Error(err.Error())
	c.JSON(code, gin.H{
		"description": err.Error(),
	})
}
