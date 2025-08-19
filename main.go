package main

import (
	"api/database"
	"api/database/models"
	"api/routes"
	"api/utils"
	"fmt"

	"os"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)


func main() {
	godotenv.Load()

	PORT := os.Getenv("PORT")

	database.Init()

	db := database.GetInstance()

	app := fiber.New(fiber.Config{
		Prefork: false,
	})
	

	app.Use("/metrics", monitor.New())

	api := app.Group("/api/v1")

	auth := api.Group("/auth")
	routes.AuthRoutes(auth)

	app.Use(jwtware.New(jwtware.Config{
		ContextKey: "user",
		Claims: &utils.JWTClaims{},
		SigningKey: jwtware.SigningKey{JWTAlg: "HS256", Key: []byte(os.Getenv("JWT_SECRET"))},
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.JSON(utils.Response{
				Success: false,
				Code: 401,
				Message: "Unauthorized",
				Data: err.Error(),
			})
		},
		SuccessHandler: func(c *fiber.Ctx) error {
			token := c.Locals("user").(*jwt.Token)
			claims := token.Claims.(*utils.JWTClaims)
			jti := claims.ID

			session := models.Session{
				JTI: jti,
			}

			err := db.Find(&session).Error

			if err != nil {
				return c.JSON(utils.Response{
					Success: false,
					Code: 401,
					Message: "Unauthorized",
					Data: nil,
				})
			}

			if session.Revoked {
				return c.JSON(utils.Response{
					Success: false,
					Code: 401,
					Message: "Unauthorized",
					Data: nil,
				})
			}

			return c.Next()
		},
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello world")
	})

	app.Listen(fmt.Sprintf(":%s", PORT))
}