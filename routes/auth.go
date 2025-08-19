package routes

import (
	"api/handlers"

	"github.com/gofiber/fiber/v2"
)



func AuthRoutes(router fiber.Router) {
	handlers.SetupAuth()
	router.Post("/register", handlers.Register)
	router.Post("/login", handlers.Login)
	router.Get("/refresh", handlers.RefreshToken)
}
