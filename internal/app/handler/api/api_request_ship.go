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

// GetRequestShipBasketAPI - GET /api/requests/basket - иконка корзины

// @Summary Get request basket
// @Description Retrieve the count of ships in the user's draft request
// @Tags request_ships
// @Produce json
// @Success 200 {object} object "data: {request_ship_id: int, ships_count: int}"
// @Failure 500 {object} object "error: string"
// @Router /api/requests/basket [get]
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

// GetRequestShipsAPI - GET /api/request_ship - список заявок

// @Summary Get list of shipping requests
// @Description Retrieve a list of requests with optional filters
// @Tags request_ships
// @Produce json
// @Param start_date query string false "Start date filter"
// @Param end_date query string false "End date filter"
// @Param status query string false "Status filter"
// @Success 200 {object} []ds.RequestShip
// @Failure 500 {object} object "error: string"
// @Router /api/request_ship [get]
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

// @Summary Get a single request
// @Description Retrieve details of a specific request with its ships
// @Tags request_ships
// @Produce json
// @Param id path int true "Request ID"
// @Success 200 {object} object "request_ship_id: int, status: string, creation_date: string, containers_20ft_count: int, containers_40ft_count: int, comment: string, loading_time: int, ships: []object"
// @Failure 400 {object} object "error: string"
// @Failure 404 {object} object "error: string"
// @Router /api/request_ship/{id} [get]
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
// @Summary Update request fields
// @Description Update fields of an existing request
// @Tags request_ships
// @Accept json
// @Produce json
// @Param id path int true "Request ID"
// @Param request body object{containers_20ft_count=int,containers_40ft_count=int,comment=string} true "Request updates"
// @Success 200 {object} object "status: string, message: string"
// @Failure 400 {object} object "error: string"
// @Failure 500 {object} object "error: string"
// @Router /api/request-ships/{id} [put]
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

// @Summary Form a request
// @Description Finalize a draft request by the creator
// @Tags request_ships
// @Produce json
// @Param id path int true "Request ID"
// @Success 200 {object} object "status: string, message: string"
// @Failure 400 {object} object "description: string"
// @Failure 404 {object} object "description: string"
// @Failure 500 {object} object "error: string"
// @Router /api/request_ship/{id}/formation [put]
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

// @Summary Complete or reject a request
// @Description Allow moderator to complete or reject a formed request
// @Tags request_ships
// @Produce json
// @Param id path int true "Request ID"
// @Param action formData string true "Action (complete or reject)"
// @Success 200 {object} object "status: string, message: string, loading_time: int (if completed)"
// @Failure 400 {object} object "description: string"
// @Failure 404 {object} object "description: string"
// @Failure 500 {object} object "error: string"
// @Router /api/request-ships/{id}/completion [post]
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

// @Summary Delete ship from request
// @Description Remove a ship from a specific request
// @Tags request_ships
// @Produce json
// @Param id path int true "Request ID"
// @Param ship_id path int true "Ship ID"
// @Success 200 {object} object "description: string"
// @Failure 400 {object} object "status: string, description: string"
// @Failure 500 {object} object "status: string, description: string"
// @Router /api/request_ship/{id}/ships/{ship_id} [delete]
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

// @Summary Update ship count in request
// @Description Update the number of ships in a specific request
// @Tags request_ships
// @Accept json
// @Produce json
// @Param id path int true "Request ID"
// @Param ship_id path int true "Ship ID"
// @Param request body object{ships_count=int} true "Updated ship count"
// @Success 200 {object} object "status: string, message: string"
// @Failure 400 {object} object "description: string"
// @Failure 500 {object} object "error: string"
// @Router /api/request_ship/{id}/ships/{ship_id} [put]
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

// @Summary Delete a request
// @Description Remove an entire request from the system
// @Tags request_ships
// @Produce json
// @Param id path int true "Request ID"
// @Success 200 {object} object "status: string, message: string"
// @Failure 400 {object} object "message: string"
// @Failure 500 {object} object "error: string"
// @Router /api/request_ship/{id} [delete]
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
