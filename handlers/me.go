package handlers

import (
	"api/database"
	"api/database/models"
	"api/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// GetMe returns the authenticated user's public profile. It expects the JWT
// middleware to have populated c.Locals("user") with a *jwt.Token whose
// claims are *utils.JWTClaims.
func GetMe(c *fiber.Ctx) error {
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(*utils.JWTClaims)

	db := database.GetInstance()

	var user models.User
	if err := db.Preload("Sessions").First(&user, claims.Subject).Error; err != nil {
		return fiber.NewError(404, "User not found")
	}

	// Sanitize sensitive fields
	user.Password = ""

	return c.JSON(utils.Response{
		Success: true,
		Code:    200,
		Message: "Success",
		Data:    user,
	})
}
