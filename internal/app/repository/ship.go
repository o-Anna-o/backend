package repository

import (
	"fmt"
	"loading_time/internal/app/ds"
)

func (r *Repository) GetShips() ([]ds.Ship, error) {
	var ships []ds.Ship
	err := r.db.Find(&ships).Error
	if err != nil {
		return nil, err
	}
	if len(ships) == 0 {
		return nil, fmt.Errorf("массив пустой")
	}

	return ships, nil
}

func (r *Repository) GetShip(id int) (ds.Ship, error) {
	ship := ds.Ship{}
	err := r.db.Where("ship_id = ?", id).First(&ship).Error
	if err != nil {
		return ds.Ship{}, err
	}
	return ship, nil
}

func (r *Repository) GetShipsByName(name string) ([]ds.Ship, error) {
	var ships []ds.Ship
	err := r.db.Where("name ILIKE ?", "%"+name+"%").Find(&ships).Error
	if err != nil {
		return nil, err
	}
	return ships, nil
}

// CreateShip - создание корабля
func (r *Repository) CreateShip(ship *ds.Ship) error {
	return r.db.Create(ship).Error
}

// UpdateShip - обновление корабля
func (r *Repository) UpdateShip(id int, ship *ds.Ship) error {
	return r.db.Model(&ds.Ship{}).Where("ship_id = ?", id).Updates(ship).Error
}

// DeleteShip - удаление корабля (логическое)
func (r *Repository) DeleteShip(id int) error {
	return r.db.Model(&ds.Ship{}).Where("ship_id = ?", id).Update("is_active", false).Error
}
