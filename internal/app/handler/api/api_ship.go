package api

import (
	"context"
	"loading_time/internal/app/ds"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

type ShipHandler struct {
	Repository interface {
		GetShips() ([]ds.Ship, error)
		GetShipsByName(name string) ([]ds.Ship, error)
		GetShip(id int) (ds.Ship, error)
		CreateShip(ship *ds.Ship) error
		UpdateShip(id int, ship *ds.Ship) error
		DeleteShip(id int) error
		DB() *gorm.DB
	}
	MinioClient *minio.Client
}

// GetShipsAPI - GET /api/ships - список кораблей с фильтрацией
func (h *ShipHandler) GetShipsAPI(c *gin.Context) {
	nameFilter := c.Query("name")
	capacityFilter := c.Query("capacity")
	isActiveFilter := c.Query("is_active")

	db := h.Repository.DB()
	query := db.Model(&ds.Ship{})

	// Фильтр по имени
	if nameFilter != "" {
		query = query.Where("name ILIKE ?", "%"+nameFilter+"%")
	}

	// Фильтр по вместимости
	if capacityFilter != "" {
		capacity, err := strconv.ParseFloat(capacityFilter, 64)
		if err == nil {
			query = query.Where("capacity >= ?", capacity)
		}
	}

	// Фильтр по активности
	if isActiveFilter != "" {
		isActive := isActiveFilter == "true"
		query = query.Where("is_active = ?", isActive)
	}

	var ships []ds.Ship
	if err := query.Find(&ships).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  ships,
		"count": len(ships),
	})
}

// GetShipAPI - GET /api/ships/:id - один корабль
func (h *ShipHandler) GetShipAPI(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ship ID",
		})
		return
	}

	ship, err := h.Repository.GetShip(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Ship not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": ship,
	})
}

// CreateShipAPI - POST /api/ships - создание корабля
func (h *ShipHandler) CreateShipAPI(c *gin.Context) {
	var ship ds.Ship
	if err := c.BindJSON(&ship); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Устанавливаем значения по умолчанию
	ship.IsActive = true

	if err := h.Repository.CreateShip(&ship); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": ship,
	})
}

// UpdateShipAPI - PUT /api/ships/:id - обновление корабля
func (h *ShipHandler) UpdateShipAPI(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ship ID",
		})
		return
	}

	var shipUpdates ds.Ship
	if err := c.BindJSON(&shipUpdates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := h.Repository.UpdateShip(id, &shipUpdates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Получаем обновленный корабль для ответа
	updatedShip, err := h.Repository.GetShip(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": updatedShip,
	})
}

// DeleteShipAPI - DELETE /api/ships/:id - удаление корабля
func (h *ShipHandler) DeleteShipAPI(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid ship ID",
		})
		return
	}

	if err := h.Repository.DeleteShip(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Ship deleted successfully",
	})
}

// AddShipToRequestShipAPI - POST /api/ships/:id/add-to-ship-bucket - добавить корабль в заявку
func (h *ShipHandler) AddShipToRequestShipAPI(c *gin.Context) {
	shipID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid ship ID",
		})
		return
	}

	// Проверяем существование корабля
	_, err = h.Repository.GetShip(shipID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Ship not found",
		})
		return
	}

	const fixedUserID = 1
	db := h.Repository.DB()

	// Получаем черновик
	var requestShip ds.RequestShip
	err = db.Where("status = ? AND user_id = ?", "черновик", fixedUserID).First(&requestShip).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			requestShip = ds.RequestShip{
				Status: "черновик",
				UserID: fixedUserID,
			}
			if err := db.Create(&requestShip).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "description": err.Error()})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "description": err.Error()})
			return
		}
	}

	// Проверяем, JSON ли это запрос
	isJSON := strings.Contains(c.GetHeader("Content-Type"), "application/json") ||
		strings.Contains(c.GetHeader("Accept"), "application/json")

	// Проверяем, есть ли уже такой корабль в заявке
	var existingShip ds.ShipInRequest
	err = db.Where("request_ship_id = ? AND ship_id = ?", requestShip.RequestShipID, shipID).First(&existingShip).Error

	if err == nil {
		existingShip.ShipsCount++
		if err := db.Save(&existingShip).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "description": err.Error()})
			return
		}
	} else if err == gorm.ErrRecordNotFound {
		shipInRequest := ds.ShipInRequest{
			RequestShipID: requestShip.RequestShipID,
			ShipID:        shipID,
			ShipsCount:    1,
		}
		if err := db.Create(&shipInRequest).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "description": err.Error()})
			return
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "description": err.Error()})
		return
	}

	// Если JSON-запрос — возвращаем JSON, иначе редирект
	if isJSON {
		c.JSON(http.StatusOK, gin.H{
			"message": "Ship added to request",
			"data": gin.H{
				"request_ship_id": requestShip.RequestShipID,
				"ship_id":         shipID,
			},
		})
	} else {
		c.Redirect(http.StatusFound, "/request_ship/"+strconv.Itoa(requestShip.RequestShipID))
	}
}

// AddShipImageAPI - POST /api/ships/:id/image - добавление изображения
func (h *ShipHandler) AddShipImageAPI(c *gin.Context) {
	shipID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid ship ID",
		})
		return
	}
	if h.MinioClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "MinIO client not available",
		})
		return
	}

	// Парсинг multipart формы
	err = c.Request.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Failed to parse form data",
		})
		return
	}

	// Проверяем существование корабля
	ship, err := h.Repository.GetShip(shipID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "Ship not found",
		})
		return
	}

	// Получаем файл из формы
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		file, header, err = c.Request.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "No image file provided",
			})
			return
		}
	}
	defer file.Close()

	// Генерируем уникальное имя файла
	fileExt := filepath.Ext(header.Filename)
	newFileName := uuid.New().String() + fileExt

	// Загружаем в MinIO
	bucketName := "loading-time-img"
	objectName := "img/" + newFileName

	_, err = h.MinioClient.PutObject(
		context.Background(),
		bucketName,
		objectName,
		file,
		header.Size,
		minio.PutObjectOptions{
			ContentType: header.Header.Get("Content-Type"),
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to upload image",
		})
		return
	}

	// Удаляем старое изображение
	if ship.PhotoURL != "" {
		oldFileName := ship.PhotoURL
		if strings.Contains(oldFileName, "/") {
			parts := strings.Split(oldFileName, "/")
			oldFileName = parts[len(parts)-1]
		}
		oldObjectName := "img/" + oldFileName
		h.MinioClient.RemoveObject(context.Background(), bucketName, oldObjectName, minio.RemoveObjectOptions{})
	}

	// Сохраняем в БД только имя файла
	ship.PhotoURL = newFileName
	if err := h.Repository.UpdateShip(shipID, &ship); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to update ship",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"ship_id":   shipID,
			"photo_url": newFileName,
			"message":   "Image uploaded successfully",
		},
	})
}
