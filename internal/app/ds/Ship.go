package ds

// @Schema(description="Ship model representing a container ship")
type Ship struct {
	ShipID      int     `gorm:"primaryKey;column:ship_id"`
	Name        string  `gorm:"column:name"`
	Capacity    float64 `gorm:"column:capacity"`
	Length      float64 `gorm:"column:length"`
	Width       float64 `gorm:"column:width"`
	Draft       float64 `gorm:"column:draft"`
	Cranes      int     `gorm:"column:cranes"`
	Containers  int     `gorm:"column:containers"`
	Description string  `gorm:"column:description"`
	PhotoURL    string  `gorm:"column:photo_url"`
}

func (Ship) TableName() string {
	return "ships"
}
