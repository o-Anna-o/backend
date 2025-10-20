package api

import (
	"loading_time/internal/app/ds"
	"loading_time/internal/app/repository"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type RequestShipHandler struct {
	Repository *repository.Repository
}

// GetRequestsBasketAPI - GET /api/requests/basket - иконка корзины
func (h *RequestShipHandler) GetRequestShipBasketAPI(c *gin.Context) {
	const fixedUserID = 1

	requestShip, err := h.Repository.GetOrCreateUserDraft(fixedUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	totalShipsCount := 0
	for _, ship := range requestShip.Ships {
		totalShipsCount += ship.ShipsCount
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"request_ship_id": requestShip.RequestShipID,
			"ships_count":     totalShipsCount,
		},
	})
}

func (h *RequestShipHandler) GetRequestShipsAPI(c *gin.Context) {
	// Получаем фильтры из query-параметров
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	status := c.Query("status")

	// Вызываем репозиторий для получения списка заявок
	requestShips, err := h.Repository.GetRequestShipsFiltered(startDate, endDate, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Возвращаем JSON
	c.JSON(http.StatusOK, requestShips)
}

// GetRequestShipAPI - GET /api/request_ship/:id - одна заявка с услугами
func (h *RequestShipHandler) GetRequestShipAPI(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request ID",
		})
		return
	}

	requestShip, err := h.Repository.GetRequestShipExcludingDeleted(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Request not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"request_ship_id":       requestShip.RequestShipID,
		"status":                requestShip.Status,
		"creation_date":         requestShip.CreationDate,
		"containers_20ft_count": requestShip.Containers20ftCount,
		"containers_40ft_count": requestShip.Containers40ftCount,
		"comment":               requestShip.Comment,
		"loading_time":          requestShip.LoadingTime,
		"ships": func() []gin.H {
			ships := []gin.H{}
			for _, shipInRequest := range requestShip.Ships {
				ships = append(ships, gin.H{
					"ship_id":     shipInRequest.Ship.ShipID,
					"name":        shipInRequest.Ship.Name,
					"photo_url":   shipInRequest.Ship.PhotoURL,
					"capacity":    shipInRequest.Ship.Capacity,
					"cranes":      shipInRequest.Ship.Cranes,
					"ships_count": shipInRequest.ShipsCount,
				})
			}
			return ships
		}(),
	})
}

// UpdateRequestShipAPI - PUT /api/request-ships/:id - изменения полей заявки
func (h *RequestShipHandler) UpdateRequestShipAPI(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{

			"message": "Invalid request ID",
		})
		return
	}

	var updates struct {
		Containers20ftCount int    `json:"containers_20ft_count"`
		Containers40ftCount int    `json:"containers_40ft_count"`
		Comment             string `json:"comment"`
	}

	if err := c.BindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{

			"error": err.Error(),
		})
		return
	}

	// Обновляем поля без расчета времени (расчет будет при завершении)
	err = h.Repository.UpdateRequestShipFields(id, updates.Containers20ftCount, updates.Containers40ftCount, updates.Comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{

			"error": err.Error(),
		})
		return
	}

	// Проверка на запрос от формы
	if c.PostForm("_method") == "PUT" {
		c.Redirect(http.StatusFound, "/request_ship/"+strconv.Itoa(id))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Request updated successfully",
	})
}

// FormRequestShipAPI - PUT /api/request_ship/:id/formation - сформировать создателем + расчёт времени
func (h *RequestShipHandler) FormRequestShipAPI(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid request ID",
		})
		return
	}

	// Получаем заявку
	requestShip, err := h.Repository.GetRequestShipExcludingDeleted(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{

			"description": "Request not found",
		})
		return
	}

	// Проверка: хотя бы один корабль
	if len(requestShip.Ships) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{

			"description": "Cannot form request: at least one ship must be added",
		})
		return
	}

	// Проверка: указаны контейнеры
	if requestShip.Containers20ftCount <= 0 && requestShip.Containers40ftCount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{

			"description": "Cannot form request: container counts must be specified",
		})
		return
	}

	// УБРАЛИ расчет времени погрузки - только меняем статус

	// меняем статус на "сформирован"
	err = h.Repository.UpdateRequestShipStatus(id, "сформирован")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{

			"error": err.Error(),
		})
		return
	}

	// Если запрос пришёл от HTML-формы — делаем редирект на страницу заявки
	if c.PostForm("_method") == "PUT" {
		c.Redirect(http.StatusFound, "/request_ship/"+strconv.Itoa(id))
		return
	}

	// Если запрос API — отдаём JSON
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Request formed successfully",
	})
}

// CompleteRequestShipAPI - POST /api/request-ships/:id/completion - завершить/отклонить модератором
func (h *RequestShipHandler) CompleteRequestShipAPI(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logrus.Errorf("CompleteRequestShipAPI: Invalid request ID: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{

			"message": "Invalid request ID",
		})
		return
	}

	action := c.PostForm("action")
	if action == "" {
		logrus.Errorf("CompleteRequestShipAPI: Action must be specified for request_ship_id=%d", id)
		c.JSON(http.StatusBadRequest, gin.H{

			"description": "Action must be specified",
		})
		return
	}

	requestShip, err := h.Repository.GetRequestShipExcludingDeleted(id)
	if err != nil {
		logrus.Errorf("CompleteRequestShipAPI: Request not found for request_ship_id=%d: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{

			"description": "Request not found",
		})
		return
	}

	// Проверяем что заявка в статусе "сформирован"
	if requestShip.Status != "сформирован" {
		logrus.Errorf("CompleteRequestShipAPI: Invalid status for request_ship_id=%d: %s", id, requestShip.Status)
		c.JSON(http.StatusBadRequest, gin.H{

			"description": "Only formed requests can be completed or rejected",
		})
		return
	}

	const moderatorID = 1 // Фиксированный модератор

	if action == "complete" {
		// Рассчитываем время погрузки (бизнес-логика из задания)
		loadingTime, err := h.Repository.CalculateLoadingTime(
			id,
			requestShip.Containers20ftCount,
			requestShip.Containers40ftCount,
		)
		if err != nil {
			logrus.Errorf("CompleteRequestShipAPI: Failed to calculate loading time for request_ship_id=%d: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{

				"error": err.Error(),
			})
			return
		}

		// Завершаем заявку с расчетом времени
		err = h.Repository.CompleteRequestShip(id, moderatorID, "завершен", loadingTime)
		if err != nil {
			logrus.Errorf("CompleteRequestShipAPI: Failed to complete request_ship_id=%d: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{

				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":       "success",
			"message":      "Request completed successfully",
			"loading_time": loadingTime,
		})

	} else if action == "reject" {
		// Отклоняем заявку
		err = h.Repository.CompleteRequestShip(id, moderatorID, "отклонен", 0)
		if err != nil {
			logrus.Errorf("CompleteRequestShipAPI: Failed to reject request_ship_id=%d: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{

				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Request rejected successfully",
		})
	} else {
		logrus.Errorf("CompleteRequestShipAPI: Invalid action for request_ship_id=%d: %s", id, action)
		c.JSON(http.StatusBadRequest, gin.H{

			"description": "Action must be 'complete' or 'reject'",
		})
	}
}

// DeleteShipFromRequestShipAPI - DELETE /api/request_ship/:id/ships/:ship_id - удаление корабля из заявки
func (h *RequestShipHandler) DeleteShipFromRequestShipAPI(c *gin.Context) {
	requestShipIDStr := c.Param("id")
	shipIDStr := c.Param("ship_id")

	requestShipID, err := strconv.Atoi(requestShipIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "description": "Invalid request ship ID"})
		return
	}

	shipID, err := strconv.Atoi(shipIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "description": "Invalid ship ID"})
		return
	}

	// Удаляем корабль из заявки
	if err := h.Repository.RemoveShipFromRequestShip(requestShipID, shipID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "description": err.Error()})
		return
	}

	// Проверка на запрос от формы
	if c.PostForm("_method") == "DELETE" {
		// Получаем обновленную заявку
		updatedRequestShip, err := h.Repository.GetRequestShipExcludingDeleted(requestShipID)
		if err != nil {
			c.Redirect(http.StatusFound, "/ships")
			return
		}

		if len(updatedRequestShip.Ships) == 0 {
			c.Redirect(http.StatusFound, "/ships")
		} else {
			c.Redirect(http.StatusFound, "/request_ship/"+requestShipIDStr)
		}
		return
	}

	// Для чистых API-запросов возвращаем JSON
	c.JSON(http.StatusOK, gin.H{"description": "Ship removed from request ship"})
}

// UpdateShipInRequestAPI - PUT /api/request_ship/:id/ships/:ship_id - обновление количества кораблей в заявке
func (h *RequestShipHandler) UpdateShipInRequestAPI(c *gin.Context) {
	requestShipIDStr := c.Param("id")
	shipIDStr := c.Param("ship_id")

	requestShipID, err := strconv.Atoi(requestShipIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{

			"description": "Invalid request ship ID",
		})
		return
	}

	shipID, err := strconv.Atoi(shipIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{

			"description": "Invalid ship ID",
		})
		return
	}

	var input struct {
		ShipsCount int `json:"ships_count"`
	}

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{

			"description": "Invalid input data",
		})
		return
	}

	if input.ShipsCount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{

			"description": "Ships count must be greater than zero",
		})
		return
	}

	// Обновляем количество кораблей в заявке
	err = h.Repository.UpdateShipCountInRequest(requestShipID, shipID, input.ShipsCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{

			"error": err.Error(),
		})
		return
	}

	// Проверка на запрос от формы
	if c.PostForm("_method") == "PUT" {
		c.Redirect(http.StatusFound, "/request_ship/"+requestShipIDStr)
		return
	}

	// Для чистых API-запросов — возвращаем JSON
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Ship count in request updated successfully",
	})
}

// DeleteRequestShipAPI - DELETE /api/request_ship/:id - удаление всей заявки
func (h *RequestShipHandler) DeleteRequestShipAPI(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logrus.Errorf("DeleteRequestShipAPI: Invalid request ID: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{

			"message": "Invalid request ID",
		})
		return
	}

	logrus.Infof("DeleteRequestShipAPI: Attempting to delete request_ship_id=%d", id)

	// Удаляем зависимые записи
	err = h.Repository.DB().Delete(&ds.ShipInRequest{}, "request_ship_id = ?", id).Error
	if err != nil {
		logrus.Errorf("DeleteRequestShipAPI: Failed to delete ShipInRequest for request_ship_id=%d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{

			"error": err.Error(),
		})
		return
	}

	// Удаляем заявку
	err = h.Repository.DB().Delete(&ds.RequestShip{}, id).Error
	if err != nil {
		logrus.Errorf("DeleteRequestShipAPI: Failed to delete RequestShip for request_ship_id=%d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{

			"error": err.Error(),
		})
		return
	}

	// Проверка на запрос от формы
	if c.PostForm("_method") == "DELETE" {
		logrus.Infof("DeleteRequestShipAPI: Redirecting to /ships for request_ship_id=%d", id)
		c.Redirect(http.StatusFound, "/ships")
		return
	}

	logrus.Infof("DeleteRequestShipAPI: Returning JSON for request_ship_id=%d", id)
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Request ship deleted successfully",
	})
}
