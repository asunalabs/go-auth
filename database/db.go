package database

import (
	"api/database/models"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Init() (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(os.Getenv("DB_URI")), &gorm.Config{})

	e := db.AutoMigrate(&models.User{}, &models.Session{})
	if e != nil {
		log.Println("Migration failed")
	}


	return db, err
}