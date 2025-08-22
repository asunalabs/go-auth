package routes

import (
	"api/handlers"

	"github.com/gofiber/fiber/v2"
)

func UserRoutes(router fiber.Router) {
	// Profile management
	router.Get("/@me", handlers.GetMe)
	router.Patch("/profile", handlers.UpdateProfile)
	router.Get("/profile/options", handlers.GetProfileOptions)

	// OAuth account management
	oauth := router.Group("/oauth")
	oauth.Get("/accounts", handlers.GetOAuthAccounts)
	oauth.Delete("/accounts/:provider", handlers.UnlinkOAuthAccount)
}
