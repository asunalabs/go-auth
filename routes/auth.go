package routes

import (
	"api/handlers"

	"github.com/gofiber/fiber/v2"
)

func AuthRoutes(router fiber.Router) {
	handlers.SetupAuth()
	handlers.SetupPasswordReset()
	router.Post("/register", handlers.Register)
	router.Post("/login", handlers.Login)
	router.Get("/refresh", handlers.RefreshToken)
	router.Get("/revoke", handlers.RevokeToken)
	router.Post("/request-password-reset", handlers.RequestPasswordReset)
	router.Post("/confirm-password-reset", handlers.ConfirmPasswordReset)
}
