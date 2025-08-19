package main

import (
	"api/database"
	"api/routes"
	"api/utils"
	"fmt"

	"os"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/joho/godotenv"
)


func main() {
	godotenv.Load()

	PORT := os.Getenv("PORT")

	database.Init()


	app := fiber.New(fiber.Config{
		Prefork: false,
	})
	

	app.Use("/metrics", monitor.New())

	api := app.Group("/api/v1")

	auth := api.Group("/auth")
	routes.AuthRoutes(auth)

	api.Use(jwtware.New(jwtware.Config{
		ContextKey: "user",
		Claims: &utils.JWTClaims{},
		SigningKey: jwtware.SigningKey{JWTAlg: "HS256", Key: []byte(os.Getenv("JWT_SECRET"))},
	}))

	

	app.Listen(fmt.Sprintf(":%s", PORT))
}