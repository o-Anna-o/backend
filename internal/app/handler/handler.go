package handler

import (
	"loading_time/internal/app/handler/api"
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
	router.GET("/ships", h.GetShips)
	router.GET("/ship/:id", h.GetShip)
	router.GET("/request_ship", h.CreateOrRedirectRequestShip)
	router.GET("/request_ship/:id", h.GetRequestShip)
	router.POST("/request_ship/calculate_loading_time/:id", h.CalculateLoadingTime)

	// API маршруты
	apiGroup := router.Group("/api")
	{
		// Домен услуги (контейнеровозы)
		apiGroup.GET("/ships", h.ShipAPIHandler.GetShipsAPI)
		apiGroup.GET("/ships/:id", h.ShipAPIHandler.GetShipAPI)
		apiGroup.POST("/ships", h.ShipAPIHandler.CreateShipAPI)
		apiGroup.PUT("/ships/:id", h.ShipAPIHandler.UpdateShipAPI)
		apiGroup.POST("/ships/:id/image", h.ShipAPIHandler.AddShipImageAPI)
		apiGroup.DELETE("/ships/:id", h.ShipAPIHandler.DeleteShipAPI)
		apiGroup.POST("/ships/:id/add-to-ship-bucket", h.ShipAPIHandler.AddShipToRequestShipAPI)

		apiGroup.PUT("/request_ship/:id/ships/:ship_id", h.RequestShipAPIHandler.UpdateShipInRequestAPI)
		apiGroup.POST("/request_ship/:id/ships/:ship_id", h.RequestShipAPIHandler.DeleteShipFromRequestShipAPI)
		apiGroup.DELETE("/request_ship/:id/ships/:ship_id", h.RequestShipAPIHandler.DeleteShipFromRequestShipAPI)
		apiGroup.PUT("/request_ship/:id/formation", h.RequestShipAPIHandler.FormRequestShipAPI)
		apiGroup.POST("/request_ship/:id/completion", h.RequestShipAPIHandler.CompleteRequestShipAPI)

		apiGroup.GET("/request_ship/:id", h.RequestShipAPIHandler.GetRequestShipAPI)
		apiGroup.PUT("/request_ship/:id", h.RequestShipAPIHandler.UpdateRequestShipAPI)
		apiGroup.POST("/request_ship/:id", h.RequestShipAPIHandler.DeleteRequestShipAPI)
		apiGroup.DELETE("/request_ship/:id", h.RequestShipAPIHandler.DeleteRequestShipAPI)
		apiGroup.GET("/request_ship/basket", h.RequestShipAPIHandler.GetRequestShipBasketAPI)
		apiGroup.GET("/request_ship", h.RequestShipAPIHandler.GetRequestShipsAPI)

		apiGroup.GET("/users/profile", h.UserAPIHandler.GetUserProfileAPI)
		apiGroup.PUT("/users/profile", h.UserAPIHandler.UpdateUserProfileAPI)

		apiGroup.POST("/users/register", h.UserAPIHandler.RegisterUserAPI)
		apiGroup.POST("/users/login", h.UserAPIHandler.LoginUserAPI)
		apiGroup.POST("/users/logout", h.UserAPIHandler.LogoutUserAPI)
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
