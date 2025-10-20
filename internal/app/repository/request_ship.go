package repository

import (
	"loading_time/internal/app/ds"
	"time"

	"gorm.io/gorm"
)

func (r *Repository) GetRequestShip(id int) (ds.RequestShip, error) {
	request_ship := ds.RequestShip{}
	// обязательно проверяем ошибки, и если они появились - передаем выше, то есть хендлеру
	err := r.db.Preload("Ships.Ship").Preload("User").Where("id = ?", id).First(&request_ship).Error
	if err != nil {
		return ds.RequestShip{}, err
	}
	return request_ship, nil
}

// GetOrCreateUserDraft - перейти или создать черновик
func (r *Repository) GetOrCreateUserDraft(userID int) (ds.RequestShip, error) {
	var requestShip ds.RequestShip

	// Ищем существующий черновик для данного пользователя
	err := r.db.Preload("Ships.Ship").Preload("User").Where("status = ? AND user_id = ?", "черновик", userID).First(&requestShip).Error
	if err == nil {
		return requestShip, nil // черновик найден
	}
	if err != gorm.ErrRecordNotFound {
		return ds.RequestShip{}, err
	}

	// Создаем новый черновик
	requestShip = ds.RequestShip{
		Status:       "черновик",
		UserID:       userID,
		CreationDate: time.Now(),
	}

	err = r.db.Create(&requestShip).Error
	if err != nil {
		return ds.RequestShip{}, err
	}

	return requestShip, nil
}

// AddShipToRequestShip - добавить корабль в заявку через ORM
func (r *Repository) AddShipToRequestShip(requestShipID, shipID int) error {
	// Сначала проверяем, есть ли уже такой корабль в заявке
	var existingShip ds.ShipInRequest
	err := r.db.Where("request_ship_id = ? AND ship_id = ?", requestShipID, shipID).First(&existingShip).Error

	if err == nil {
		// Корабль уже есть в заявке - увеличиваем количество
		existingShip.ShipsCount++
		return r.db.Save(&existingShip).Error
	}

	shipInRequest := ds.ShipInRequest{
		RequestShipID: requestShipID,
		ShipID:        shipID,
		ShipsCount:    1,
	}
	return r.db.Create(&shipInRequest).Error
}

// RemoveShipFromRequestShip — удалить корабль из заявки
func (r *Repository) RemoveShipFromRequestShip(requestShipID, shipID int) error {
	return r.db.
		Where("request_ship_id = ? AND ship_id = ?", requestShipID, shipID).
		Delete(&ds.ShipInRequest{}).
		Error
}

// логическое удаление заявки через SQL
func (r *Repository) DeleteRequestShipSQL(requestShipID int) error {
	return r.db.Model(&ds.RequestShip{}).
		Where("request_ship_id = ?", requestShipID).
		Update("status", "удалён").Error
}

// GetRequestShipExcludingDeleted - получить заявку исключая удаленные (через ORM)
func (r *Repository) GetRequestShipExcludingDeleted(id int) (ds.RequestShip, error) {
	var requestShip ds.RequestShip
	err := r.db.Preload("Ships.Ship").Preload("User").Where("request_ship_id = ? AND status != ?", id, "удалён").First(&requestShip).Error // добавили Preload("User")
	if err != nil {
		return ds.RequestShip{}, err
	}
	return requestShip, nil
}

// CalculateLoadingTime - рассчитывает время погрузки по формуле
func (r *Repository) CalculateLoadingTime(requestShipID, containers20ft, containers40ft int) (float64, error) {
	// Получаем заявку с кораблями
	var requestShip ds.RequestShip
	err := r.db.Preload("Ships.Ship").Where("request_ship_id = ?", requestShipID).First(&requestShip).Error
	if err != nil {
		return 0, err
	}

	// Рассчитываем общее количество кранов
	totalCranes := 0
	for _, shipInRequest := range requestShip.Ships {
		totalCranes += shipInRequest.Ship.Cranes * shipInRequest.ShipsCount
	}

	if totalCranes == 0 {
		return 0, nil
	}

	// общее время = (20ft * 2 + 40ft * 3) / количество кранов
	totalContainerTime := float64(containers20ft)*2 + float64(containers40ft)*3
	return totalContainerTime / float64(totalCranes), nil
}

// UpdateRequestShipFields - обновляет поля заявки и рассчитывает время погрузки
func (r *Repository) UpdateRequestShipFields(requestShipID, containers20ft, containers40ft int, comment string) error {
	// Рассчитываем время погрузки
	loadingTime, err := r.CalculateLoadingTime(requestShipID, containers20ft, containers40ft)
	if err != nil {
		return err
	}

	// Обновляем заявку
	return r.db.Model(&ds.RequestShip{}).Where("request_ship_id = ?", requestShipID).Updates(map[string]interface{}{
		"containers_20ft_count": containers20ft,
		"containers_40ft_count": containers40ft,
		"comment":               comment,
		"loading_time":          loadingTime,
	}).Error
}

func (r *Repository) GetRequestShipsFiltered(startDate, endDate, status string) ([]ds.RequestShip, error) {
	var requestShips []ds.RequestShip
	query := r.db.Model(&ds.RequestShip{}).Where("status != ?", "deleted") // исключаем удалённые

	// Фильтры по дате
	if startDate != "" {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("created_at <= ?", endDate)
	}

	// Фильтр по статусу
	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Preload("Ships").Preload("User").Find(&requestShips).Error // добавили Preload("User")
	return requestShips, err
}

//_______________________________________________________________________________________________________

// для REST API

// UpdateRequestShipStatus - обновляет статус заявки
func (r *Repository) UpdateRequestShipStatus(requestShipID int, status string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if status == "сформирован" {
		updates["formation_date"] = time.Now()
	}

	return r.db.Model(&ds.RequestShip{}).
		Where("request_ship_id = ?", requestShipID).
		Updates(updates).Error
}

// CompleteRequestShip - завершает заявку (устанавливает модератора, статус и время)
func (r *Repository) CompleteRequestShip(requestShipID, moderatorID int, status string, loadingTime float64) error {
	return r.db.Model(&ds.RequestShip{}).
		Where("request_ship_id = ?", requestShipID).
		Updates(map[string]interface{}{
			"status":          status,
			"moderator_id":    moderatorID,
			"completion_date": time.Now(),
			"loading_time":    loadingTime,
		}).Error
}

// UpdateShipCountInRequest - обновляет количество кораблей в заявке
func (r *Repository) UpdateShipCountInRequest(requestShipID, shipID, count int) error {
	return r.db.Model(&ds.ShipInRequest{}).
		Where("request_ship_id = ? AND ship_id = ?", requestShipID, shipID).
		Update("ships_count", count).Error
}

// DeleteRequestShip - полностью удалить заявку
func (r *Repository) DeleteRequestShip(requestShipID int) error {
	return r.db.Delete(&ds.RequestShip{}, requestShipID).Error
}

// UpdateRequestShipLoadingTime - сохраняет рассчитанное время погрузки
func (r *Repository) UpdateRequestShipLoadingTime(requestShipID int, loadingTime float64) error {
	return r.db.Model(&ds.RequestShip{}).
		Where("request_ship_id = ?", requestShipID).
		Update("loading_time", loadingTime).Error
}
