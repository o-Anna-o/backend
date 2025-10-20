package handler

import (
	"loading_time/internal/app/ds"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *Handler) GetShips(ctx *gin.Context) {
	var ships []ds.Ship
	var err error

	searchQuery := ctx.Query("search")
	if searchQuery == "" {
		ships, err = h.Repository.GetShips()
	} else {
		ships, err = h.Repository.GetShipsByName(searchQuery)
	}
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	// Получение черновика заявки
	const fixedUserID = 1
	requestShipCount := 0
	requestShipID := 0

	requestShip, err := h.Repository.GetOrCreateUserDraft(fixedUserID)
	if err == nil {
		logrus.Infof("Найдена заявка ID=%d, количество кораблей в заявке: %d", requestShip.RequestShipID, len(requestShip.Ships))
		for i, shipInRequest := range requestShip.Ships {
			logrus.Infof("Корабль %d: ID=%d, количество: %d", i, shipInRequest.ShipID, shipInRequest.ShipsCount)
			requestShipCount += shipInRequest.ShipsCount
		}
		requestShipID = requestShip.RequestShipID
	} else {
		logrus.Errorf("Ошибка получения заявки: %v", err)
	}

	logrus.Infof("Итоговый счетчик для отображения: %d", requestShipCount)

	ctx.HTML(http.StatusOK, "index.html", gin.H{
		"ships":              ships,
		"search":             searchQuery,
		"request_ship_count": requestShipCount,
		"request_ship_id":    requestShipID,
	})
}

func (h *Handler) GetShip(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	ship, err := h.Repository.GetShip(id)
	if err != nil {
		h.errorHandler(ctx, http.StatusNotFound, err)
		return
	}

	ctx.HTML(http.StatusOK, "ship.html", gin.H{
		"ship": ship,
	})
}
