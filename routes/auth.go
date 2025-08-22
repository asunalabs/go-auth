package routes

import (
	"api/handlers"

	"github.com/gofiber/fiber/v2"
)

func AuthRoutes(router fiber.Router) {
	handlers.SetupAuth()
	handlers.SetupPasswordReset()

	// Traditional auth routes
	router.Post("/register", handlers.Register)
	router.Post("/login", handlers.Login)
	router.Get("/refresh", handlers.RefreshToken)
	router.Get("/revoke", handlers.RevokeToken)
	router.Post("/request-password-reset", handlers.RequestPasswordReset)
	router.Post("/confirm-password-reset", handlers.ConfirmPasswordReset)

	// OAuth routes
	oauth := router.Group("/oauth")
	oauth.Post("/initiate", handlers.OAuthInitiate)
	oauth.Get("/:provider/callback", handlers.OAuthCallback)
}
