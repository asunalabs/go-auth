package routes

import (
	"api/handlers"

	"github.com/gofiber/fiber/v2"
)



func AuthRoutes(router fiber.Router) {
	handlers.SetupAuth()
	router.Post("/register", handlers.Register)
}
