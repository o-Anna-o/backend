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
		// Домен услуги (контейнеровозы)
		apiGroup.GET("/ships", h.ShipAPIHandler.GetShipsAPI)
		apiGroup.GET("/ships/:id", h.ShipAPIHandler.GetShipAPI)
		apiGroup.POST("/ships", h.ShipAPIHandler.CreateShipAPI)    // модератор
		apiGroup.PUT("/ships/:id", h.ShipAPIHandler.UpdateShipAPI) // модератор
		apiGroup.DELETE("/ships/:id", h.ShipAPIHandler.DeleteShipAPI)
		apiGroup.POST("/ships/:id/add-to-ship-bucket", h.ShipAPIHandler.AddShipToRequestShipAPI)
		apiGroup.POST("/ships/:id/image", h.ShipAPIHandler.AddShipImageAPI)

		// Домен заявки
		apiGroup.GET("/request_ship/basket", h.RequestShipAPIHandler.GetRequestShipBasketAPI)
		apiGroup.GET("/request_ship", h.RequestShipAPIHandler.GetRequestShipsAPI)
		apiGroup.GET("/request_ship/:id", h.RequestShipAPIHandler.GetRequestShipAPI)
		apiGroup.PUT("/request_ship/:id", h.RequestShipAPIHandler.UpdateRequestShipAPI)
		apiGroup.PUT("/request_ship/:id/formation", h.RequestShipAPIHandler.FormRequestShipAPI)
		apiGroup.PUT("/request_ship/:id/completion", h.RequestShipAPIHandler.CompleteRequestShipAPI)
		apiGroup.DELETE("/request_ship/:id", h.RequestShipAPIHandler.DeleteRequestShipAPI)

		// Домен м-м
		apiGroup.PUT("/request_ship/:id/ships/:ship_id", h.RequestShipAPIHandler.UpdateShipInRequestAPI)
		apiGroup.DELETE("/request_ship/:id/ships/:ship_id", h.RequestShipAPIHandler.DeleteShipFromRequestShipAPI)

		// Домен пользователя
		apiGroup.POST("/users/register", h.UserAPIHandler.RegisterUserAPI)
		authGroup := apiGroup.Group("/", middleware.AuthMiddleware(h.Repository))
		{
			authGroup.GET("/users/profile", h.UserAPIHandler.GetUserProfileAPI)
			authGroup.PUT("/users/profile", h.UserAPIHandler.UpdateUserProfileAPI)
			authGroup.POST("/users/logout", h.UserAPIHandler.LogoutUserAPI)
			authGroup.POST("/users/login", h.UserAPIHandler.LoginUserAPI)
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
