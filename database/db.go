package database

import (
	"api/database/models"
	"log"
	"net/url"
	"os"

	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var Database *gorm.DB

func Init() {
	serviceURI := os.Getenv("DB_URI")

	conn, _ := url.Parse(serviceURI)

	db, err := gorm.Open(postgres.Open(conn.String()))
	if err != nil {
		log.Fatal("Database Failed to load")
	}

	sqlDB, e := db.DB()
	if e != nil {
		log.Fatal("Failed")
	}
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	db.AutoMigrate(&models.User{}, &models.Session{})

	Database = db
}

func GetInstance() *gorm.DB {
	if Database == nil {
		log.Fatal("Database not loaded yet")
	}

	return Database
}