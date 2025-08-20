package routes

import (
	"api/handlers"

	"github.com/gofiber/fiber/v2"
)

func UserRoutes(router fiber.Router) {
	router.Get("/@me", handlers.GetMe)
}
