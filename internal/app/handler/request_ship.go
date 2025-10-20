package handler

import (
	"fmt"
	"loading_time/internal/app/ds"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GET /request_ship/:id - просмотр заявки
func (h *Handler) GetRequestShip(ctx *gin.Context) {
	idStr := ctx.Param("id")

	requestShipID, err := strconv.Atoi(idStr)
	if err != nil || requestShipID <= 0 {
		logrus.Errorf("Неверный ID заявки: %v", idStr)
		ctx.HTML(http.StatusBadRequest, "request_ship.html", gin.H{
			"request_ship": ds.RequestShip{},
			"error":        "Некорректный ID заявки",
		})
		return
	}

	requestShip, err := h.Repository.GetRequestShipExcludingDeleted(requestShipID)
	if err != nil {
		logrus.Errorf("Заявка не найдена или удалена: %v", err)
		ctx.HTML(http.StatusNotFound, "PageNotFound.html", gin.H{
			"id": idStr,
		})
		return
	}

	ctx.HTML(http.StatusOK, "request_ship.html", gin.H{
		"request_ship": requestShip,
	})
}

// GET /request_ship - редирект на черновик
func (h *Handler) CreateOrRedirectRequestShip(ctx *gin.Context) {
	requestShip, err := h.Repository.GetOrCreateUserDraft(1)
	if err != nil {
		logrus.Error(err)
		ctx.HTML(http.StatusInternalServerError, "request_ship.html", gin.H{
			"request_ship": ds.RequestShip{},
			"error":        "Не удалось создать черновик",
		})
		return
	}
	ctx.Redirect(http.StatusFound, fmt.Sprintf("/request_ship/%d", requestShip.RequestShipID))
}

// POST /request_ship/add/:ship_id - добавить корабль
func (h *Handler) AddShipToRequestShip(c *gin.Context) {
	shipIDStr := c.Param("ship_id")
	shipID, err := strconv.Atoi(shipIDStr)
	if err != nil {
		h.errorHandler(c, http.StatusBadRequest, err)
		return
	}

	requestShip, err := h.Repository.GetOrCreateUserDraft(1)
	if err != nil {
		h.errorHandler(c, http.StatusInternalServerError, err)
		return
	}

	err = h.Repository.AddShipToRequestShip(requestShip.RequestShipID, shipID)
	if err != nil {
		h.errorHandler(c, http.StatusInternalServerError, err)
		return
	}

	logrus.Infof("Корабль %d добавлен в заявку %d через ORM", shipID, requestShip.RequestShipID)
	c.Redirect(http.StatusFound, fmt.Sprintf("/request_ship/%d", requestShip.RequestShipID))
}

// POST /request_ship/:id/remove/:ship_id - удалить корабль из заявки (HTML-версия)
func (h *Handler) RemoveShipFromRequestShip(c *gin.Context) {
	requestShipIDStr := c.Param("id")
	shipIDStr := c.Param("ship_id")

	requestShipID, err := strconv.Atoi(requestShipIDStr)
	if err != nil {
		h.errorHandler(c, http.StatusBadRequest, err)
		return
	}

	shipID, err := strconv.Atoi(shipIDStr)
	if err != nil {
		h.errorHandler(c, http.StatusBadRequest, err)
		return
	}

	logrus.Infof("Удаление корабля %d из заявки %d", shipID, requestShipID)

	err = h.Repository.RemoveShipFromRequestShip(requestShipID, shipID)
	if err != nil {
		h.errorHandler(c, http.StatusInternalServerError, err)
		return
	}

	logrus.Infof("Корабль %d успешно удалён из заявки %d", shipID, requestShipID)
	c.Redirect(http.StatusFound, "/request_ship/"+requestShipIDStr)
}

// POST /request_ship/delete/:id - удалить заявку
func (h *Handler) DeleteRequestShip(c *gin.Context) {
	requestShipID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.errorHandler(c, http.StatusBadRequest, err)
		return
	}

	if err := h.Repository.DeleteRequestShipSQL(requestShipID); err != nil {
		h.errorHandler(c, http.StatusInternalServerError, err)
		return
	}

	logrus.Infof("Заявка %d помечена как удалённая", requestShipID)
	c.Redirect(http.StatusFound, "/ships")
}

// POST /request_ship/calculate_loading_time/:id - рассчитать время погрузки и сохранить поля
func (h *Handler) CalculateLoadingTime(c *gin.Context) {
	requestShipID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.errorHandler(c, http.StatusBadRequest, err)
		return
	}

	containers20ft, _ := strconv.Atoi(c.PostForm("containers_20ft"))
	containers40ft, _ := strconv.Atoi(c.PostForm("containers_40ft"))
	comment := c.PostForm("comment")

	err = h.Repository.UpdateRequestShipFields(requestShipID, containers20ft, containers40ft, comment)
	if err != nil {
		h.errorHandler(c, http.StatusInternalServerError, err)
		return
	}

	logrus.Infof("Заявка %d обновлена: 20ft=%d, 40ft=%d, comment=%s",
		requestShipID, containers20ft, containers40ft, comment)

	c.Redirect(http.StatusFound, fmt.Sprintf("/request_ship/%d", requestShipID))
}
