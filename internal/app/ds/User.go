package ds

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// @Schema(description="User model representing a registered user")
type User struct {
	UserID              int     `gorm:"primaryKey;column:user_id"`
	FIO                 string  `gorm:"column:fio"`
	Login               string  `gorm:"column:login;unique"`
	Password            string  `gorm:"column:password"`
	Contacts            string  `gorm:"column:contacts"`
	CargoWeight         float64 `gorm:"column:cargo_weight"`
	Containers20ftCount int     `gorm:"column:containers_20ft_count"`
	Containers40ftCount int     `gorm:"column:containers_40ft_count"`
	Role                string  `gorm:"column:role"` // "guest" | "creator" | "moderator"
}

// Хук для хеширования пароля перед сохранением
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

func (User) TableName() string {
	return "users"
}
