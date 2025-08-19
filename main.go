package main

import (
	"api/database"
	"api/utils"
	"fmt"
	"log"
	"os"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var Database *gorm.DB

func main() {
	godotenv.Load()

	PORT := os.Getenv("PORT")

	db, err := database.Init()

	Database = db

	if err != nil {
		log.Fatal(err)
	}
	
	app := fiber.New(fiber.Config{
		Prefork: false,
	})
	

	app.Use("/metrics", monitor.New())

	app.Use(jwtware.New(jwtware.Config{
		ContextKey: "user",
		Claims: &utils.JWTClaims{},
		SigningKey: jwtware.SigningKey{JWTAlg: "HS256", Key: []byte(os.Getenv("JWT_SECRET"))},
	}))


	app.Listen(fmt.Sprintf(":%s", PORT))
}