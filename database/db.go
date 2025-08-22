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

	db.AutoMigrate(&models.User{}, &models.Session{}, &models.PasswordReset{}, &models.OAuthAccount{}, &models.OAuthState{})

	// Add composite unique index for OAuth accounts (user_id + provider)
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_oauth_accounts_user_provider ON oauth_accounts(user_id, provider) WHERE deleted_at IS NULL")

	Database = db
}

func GetInstance() *gorm.DB {
	if Database == nil {
		log.Fatal("Database not loaded yet")
	}

	return Database
}
