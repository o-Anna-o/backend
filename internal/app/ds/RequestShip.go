package ds

import (
	"time"
)

// @Schema(description="RequestShip model representing a shipping request")
type RequestShip struct {
	RequestShipID       int             `gorm:"primaryKey;column:request_ship_id"`
	Status              string          `gorm:"column:status"`
	CreationDate        time.Time       `gorm:"column:creation_date"`
	UserID              int             `gorm:"column:user_id"`
	User                User            `gorm:"foreignKey:UserID"` // автозаполнение пользователя в заявках
	CompletionDate      *time.Time      `gorm:"column:completion_date"`
	Containers20ftCount int             `gorm:"column:containers_20ft_count"`
	Containers40ftCount int             `gorm:"column:containers_40ft_count"`
	Comment             string          `gorm:"column:comment"`
	LoadingTime         float64         `gorm:"column:loading_time"`
	Ships               []ShipInRequest `gorm:"foreignKey:RequestShipID"`
}

func (RequestShip) TableName() string {
	return "request_ship"
}

func (r *RequestShip) CalculateLoadingTime() float64 {
	// общее время = (20ft * 2 + 40ft * 3) / количество кранов
	// 2 - это часы на погрузку одного 20ft контейнера и 3 -соответственно одного 40ft контейнера
	totalCranes := 0
	for _, shipInRequest := range r.Ships {
		totalCranes += shipInRequest.Ship.Cranes * shipInRequest.ShipsCount
	}

	if totalCranes == 0 {
		return 0
	}

	totalContainerTime := float64(r.Containers20ftCount)*2 + float64(r.Containers40ftCount)*3
	return totalContainerTime / float64(totalCranes)
}
