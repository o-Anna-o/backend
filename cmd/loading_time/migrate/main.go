package main

import (
	"loading_time/internal/app/config"
	"loading_time/internal/app/ds"
	"loading_time/internal/app/dsn"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_, err := config.NewConfig()
	if err != nil {
		logrus.Fatalf("error loading config: %v", err)
	}

	postgresString := dsn.FromEnv()
	db, err := gorm.Open(postgres.Open(postgresString), &gorm.Config{})
	if err != nil {
		logrus.Fatalf("error connecting to database: %v", err)
	}

	// Порядок миграций: сначала users, потом request_ship, ships, ships_in_request
	err = db.AutoMigrate(&ds.User{})
	if err != nil {
		logrus.Fatalf("error migrating users: %v", err)
	}
	err = db.AutoMigrate(&ds.RequestShip{})
	if err != nil {
		logrus.Fatalf("error migrating request_ship: %v", err)
	}
	err = db.AutoMigrate(&ds.Ship{})
	if err != nil {
		logrus.Fatalf("error migrating ships: %v", err)
	}
	err = db.AutoMigrate(&ds.ShipInRequest{})
	if err != nil {
		logrus.Fatalf("error migrating ships_in_request: %v", err)
	}

	logrus.Info("Database migration completed")
}
