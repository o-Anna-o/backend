package ds

type ShipInRequest struct {
	RequestShipID int  `gorm:"primaryKey;column:request_ship_id"`
	ShipID        int  `gorm:"primaryKey;column:ship_id"`
	ShipsCount    int  `gorm:"column:ships_count"`
	Ship          Ship `gorm:"foreignKey:ShipID"`
}

func (ShipInRequest) TableName() string {
	return "ships_in_request"
}
