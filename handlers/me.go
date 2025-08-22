package handlers

import (
	"api/database"
	"api/database/models"
	"api/utils"
	"fmt"

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
	if err := db.Preload("Sessions").Preload("OAuthLinks").First(&user, claims.Subject).Error; err != nil {
		return fiber.NewError(404, "User not found")
	}

	// Sanitize sensitive fields
	user.Password = ""
	for i := range user.OAuthLinks {
		user.OAuthLinks[i].AccessToken = ""
		user.OAuthLinks[i].RefreshToken = ""
	}

	return c.JSON(utils.Response{
		Success: true,
		Code:    200,
		Message: "Success",
		Data:    user,
	})
}

// GetOAuthAccounts returns the user's linked OAuth accounts
func GetOAuthAccounts(c *fiber.Ctx) error {
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(*utils.JWTClaims)

	db := database.GetInstance()

	var oauthAccounts []models.OAuthAccount
	if err := db.Where("user_id = ?", claims.Subject).Find(&oauthAccounts).Error; err != nil {
		return fiber.NewError(500, "Failed to fetch OAuth accounts")
	}

	// Sanitize sensitive fields
	for i := range oauthAccounts {
		oauthAccounts[i].AccessToken = ""
		oauthAccounts[i].RefreshToken = ""
	}

	return c.JSON(utils.Response{
		Success: true,
		Code:    200,
		Message: "Success",
		Data:    oauthAccounts,
	})
}

// UnlinkOAuthAccount removes an OAuth provider link from the user's account
func UnlinkOAuthAccount(c *fiber.Ctx) error {
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(*utils.JWTClaims)
	provider := c.Params("provider")

	if provider == "" {
		return fiber.NewError(400, "Provider parameter required")
	}

	oauthProvider := models.OAuthProvider(provider)
	if oauthProvider != models.OAuthProviderGoogle && oauthProvider != models.OAuthProviderGithub {
		return fiber.NewError(400, "Invalid OAuth provider")
	}

	db := database.GetInstance()

	// Start transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if user exists and get their account type
	var user models.User
	if err := tx.First(&user, claims.Subject).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(404, "User not found")
	}

	// Find the OAuth account to unlink
	var oauthAccount models.OAuthAccount
	err := tx.Where("user_id = ? AND provider = ?", claims.Subject, oauthProvider).First(&oauthAccount).Error
	if err != nil {
		tx.Rollback()
		return fiber.NewError(404, "OAuth account not linked")
	}

	// Check business rules for unlinking
	if user.AccountType == models.AccountTypeOAuth {
		// Count remaining OAuth links
		var oauthCount int64
		tx.Model(&models.OAuthAccount{}).Where("user_id = ?", claims.Subject).Count(&oauthCount)

		if oauthCount <= 1 {
			tx.Rollback()
			return fiber.NewError(400, "Cannot unlink the only authentication method. Please set a password first or link another OAuth account.")
		}
	}

	// Delete the OAuth account
	if err := tx.Delete(&oauthAccount).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(500, "Failed to unlink OAuth account")
	}

	// Update user account type if necessary
	if user.AccountType == models.AccountTypeHybrid {
		var remainingOAuthCount int64
		tx.Model(&models.OAuthAccount{}).Where("user_id = ?", claims.Subject).Count(&remainingOAuthCount)

		if remainingOAuthCount == 0 && user.Password != "" {
			// No more OAuth accounts but has password - revert to email type
			user.AccountType = models.AccountTypeEmail
			tx.Save(&user)
		}
	}

	tx.Commit()

	return c.JSON(utils.Response{
		Success: true,
		Code:    200,
		Message: fmt.Sprintf("%s account unlinked successfully", provider),
		Data:    nil,
	})
}

// UpdateProfileRequest represents the request body for updating user profile
type UpdateProfileRequest struct {
	Username string          `json:"username,omitempty"`
	Currency models.Currency `json:"currency,omitempty"`
	Timezone models.Timezone `json:"timezone,omitempty"`
}

// UpdateProfile updates the authenticated user's profile information
func UpdateProfile(c *fiber.Ctx) error {
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(*utils.JWTClaims)

	var req UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(400, "Invalid request body")
	}

	db := database.GetInstance()

	// Start transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get current user
	var user models.User
	if err := tx.First(&user, claims.Subject).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(404, "User not found")
	}

	// Validate and update fields
	updates := make(map[string]interface{})

	// Username validation and update
	if req.Username != "" {
		if len(req.Username) < 3 {
			tx.Rollback()
			return fiber.NewError(400, "Username must be at least 3 characters long")
		}
		if len(req.Username) > 255 {
			tx.Rollback()
			return fiber.NewError(400, "Username must be less than 255 characters")
		}

		// Check if username is already taken by another user
		var count int64
		if err := tx.Model(&models.User{}).Where("username = ? AND id != ?", req.Username, user.ID).Count(&count).Error; err != nil {
			tx.Rollback()
			return fiber.NewError(500, "Database error checking username")
		}
		if count > 0 {
			tx.Rollback()
			return fiber.NewError(409, "Username already taken")
		}

		updates["username"] = req.Username
	}

	// Currency validation and update
	if req.Currency != "" {
		if !isValidCurrency(req.Currency) {
			tx.Rollback()
			return fiber.NewError(400, "Invalid currency. Supported currencies: ron, eur, gbp, usd")
		}
		updates["currency"] = req.Currency
	}

	// Timezone validation and update
	if req.Timezone != "" {
		if !isValidTimezone(req.Timezone) {
			tx.Rollback()
			return fiber.NewError(400, "Invalid timezone")
		}
		updates["timezone"] = req.Timezone
	}

	// Check if there are any updates to apply
	if len(updates) == 0 {
		tx.Rollback()
		return fiber.NewError(400, "No valid fields to update")
	}

	// Apply updates
	if err := tx.Model(&user).Updates(updates).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(500, "Failed to update profile")
	}

	// Reload user with updated data
	if err := tx.Preload("OAuthLinks").First(&user, claims.Subject).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(500, "Failed to reload user data")
	}

	tx.Commit()

	// Sanitize sensitive fields
	user.Password = ""
	for i := range user.OAuthLinks {
		user.OAuthLinks[i].AccessToken = ""
		user.OAuthLinks[i].RefreshToken = ""
	}

	return c.JSON(utils.Response{
		Success: true,
		Code:    200,
		Message: "Profile updated successfully",
		Data:    user,
	})
}

// GetProfileOptions returns available currencies and timezones
func GetProfileOptions(c *fiber.Ctx) error {
	currencies := []models.Currency{
		models.CurrencyRON,
		models.CurrencyEUR,
		models.CurrencyGBP,
		models.CurrencyUSD,
	}

	timezones := []models.Timezone{
		models.TimezoneUTC,
		models.TimezoneEuropeAmsterdam,
		models.TimezoneEuropeBerlin,
		models.TimezoneEuropeBucharest,
		models.TimezoneEuropeLondon,
		models.TimezoneEuropeParis,
		models.TimezoneEuropeRome,
		models.TimezoneAmericaNewYork,
		models.TimezoneAmericaChicago,
		models.TimezoneAmericaDenver,
		models.TimezoneAmericaLosAngeles,
		models.TimezoneAsiaShanghai,
		models.TimezoneAsiaTokyo,
		models.TimezoneAsiaKolkata,
		models.TimezoneAustraliaSydney,
	}

	return c.JSON(utils.Response{
		Success: true,
		Code:    200,
		Message: "Profile options retrieved successfully",
		Data: fiber.Map{
			"currencies": currencies,
			"timezones":  timezones,
		},
	})
}

// Helper functions for validation

func isValidCurrency(currency models.Currency) bool {
	validCurrencies := []models.Currency{
		models.CurrencyRON,
		models.CurrencyEUR,
		models.CurrencyGBP,
		models.CurrencyUSD,
	}

	for _, valid := range validCurrencies {
		if currency == valid {
			return true
		}
	}
	return false
}

func isValidTimezone(timezone models.Timezone) bool {
	validTimezones := []models.Timezone{
		models.TimezoneUTC,
		models.TimezoneEuropeAmsterdam,
		models.TimezoneEuropeBerlin,
		models.TimezoneEuropeBucharest,
		models.TimezoneEuropeLondon,
		models.TimezoneEuropeParis,
		models.TimezoneEuropeRome,
		models.TimezoneAmericaNewYork,
		models.TimezoneAmericaChicago,
		models.TimezoneAmericaDenver,
		models.TimezoneAmericaLosAngeles,
		models.TimezoneAsiaShanghai,
		models.TimezoneAsiaTokyo,
		models.TimezoneAsiaKolkata,
		models.TimezoneAustraliaSydney,
	}

	for _, valid := range validTimezones {
		if timezone == valid {
			return true
		}
	}
	return false
}
