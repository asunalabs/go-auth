package main

import (
	"api/database"
	"api/database/models"
	"api/routes"
	"api/utils"
	"errors"
	"fmt"
	"log"

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
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError

			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
			}

			err = ctx.Status(code).JSON(utils.Response{
				Success: false,
				Code: uint(code),
				Message: e.Error(),
				Data: nil,
			})

			if err != nil {
				return ctx.Status(fiber.StatusInternalServerError).JSON(utils.Response{
					Success: false,
					Code: 500,
					Message: "Internal server error",
					Data: nil,
				})
			}
			return nil
		},
	})

	app.Use("/metrics", monitor.New())

	api := app.Group("/api/v1")

	// Public auth routes (register/login/refresh) should not require JWTs.
	auth := api.Group("/auth")
	routes.AuthRoutes(auth)

	// Protected routes: apply JWT middleware only to this subgroup. Any routes
	// that require authentication should be registered under `protected`.
	protected := api.Group("/")
	protected.Use(jwtware.New(jwtware.Config{
		ContextKey: "user",
		Claims:     &utils.JWTClaims{},
		SigningKey: jwtware.SigningKey{JWTAlg: "HS256", Key: []byte(os.Getenv("JWT_SECRET"))},
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.JSON(utils.Response{
				Success: false,
				Code:    401,
				Message: "Unauthorized",
				Data:    err.Error(),
			})
		},
		SuccessHandler: func(c *fiber.Ctx) error {
			token := c.Locals("user").(*jwt.Token)
			claims := token.Claims.(*utils.JWTClaims)
			jti := claims.ID

			// Look up session by JTI. Use Where + First to query by the JTI column.
			var session models.Session
			err := db.Where(&models.Session{JTI: jti}).First(&session).Error

			if err != nil {
				return c.JSON(utils.Response{
					Success: false,
					Code:    401,
					Message: "Unauthorized",
					Data:    nil,
				})
			}

			if session.Revoked {
				return c.JSON(utils.Response{
					Success: false,
					Code:    401,
					Message: "Unauthorized",
					Data:    nil,
				})
			}

			return c.Next()
		},
	}))

	userGroup := protected.Group("/user")
	routes.UserRoutes(userGroup)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello world")
	})

	err := app.Listen(fmt.Sprintf(":%s", PORT))

	if err != nil {
		log.Fatal(err)
	}
}
