package handlers

import (
	"api/database"
	"api/database/models"
	"api/utils"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

type RequestPasswordResetProps struct {
	Email string `json:"email" validate:"required,email"`
}

type ConfirmPasswordResetProps struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

// RequestPasswordReset initiates a password reset flow for the given email.
// It implements rate limiting and sends a secure token via email.
func RequestPasswordReset(c *fiber.Ctx) error {
	var body RequestPasswordResetProps
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(400, "Malformed request")
	}

	// Basic email validation
	if body.Email == "" {
		return fiber.NewError(400, "Email is required")
	}

	db := database.GetInstance()

	// Check if user exists - but don't reveal if they don't (security)
	var user models.User
	userExists := db.Where(&models.User{Email: body.Email}).First(&user).Error == nil

	// Rate limiting: check if there's already a recent reset request
	var recentReset models.PasswordReset
	recentResetExists := db.Where("email = ? AND created_at > ? AND used = false",
		body.Email, time.Now().Add(-15*time.Minute)).First(&recentReset).Error == nil

	if recentResetExists {
		return fiber.NewError(429, "Password reset already requested recently. Please check your email or wait 15 minutes.")
	}

	// Always generate a token and simulate sending email for security
	// (don't reveal whether user exists)
	token, hashedToken := utils.GenerateSecureToken()

	if userExists {
		// Mark any existing unused tokens as used
		db.Model(&models.PasswordReset{}).Where("email = ? AND used = false", body.Email).Update("used", true)

		// Create new password reset record
		passwordReset := models.PasswordReset{
			Email:     body.Email,
			Token:     hashedToken,
			Used:      false,
			ExpiresAt: time.Now().Add(1 * time.Hour), // 1 hour expiry
		}

		if err := db.Create(&passwordReset).Error; err != nil {
			return fmt.Errorf("failed to create password reset: %w", err)
		}

		// Send reset email asynchronously
		go func(email, resetToken string) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			client := utils.NewSMTPClient()
			subject := "Password Reset Request"

			clientUrl := os.Getenv("CLIENT_URL")
			resetURL := fmt.Sprintf("%s/reset-password?token=%s", clientUrl, resetToken)
			body := fmt.Sprintf(`You requested a password reset for your account.

Click the link below to reset your password:
%s

This link will expire in 1 hour.

If you didn't request this reset, please ignore this email.

Thanks,
Asuna Labs Team`, resetURL)

			if err := client.Send(ctx, []string{email}, subject, body); err != nil {
				// Log error but don't fail the request
				// In production, consider using a proper logger
				_ = err
			}
		}(user.Email, token)
	}

	// Always return success to prevent user enumeration
	return c.JSON(utils.Response{
		Success: true,
		Code:    200,
		Message: "If an account with that email exists, a password reset link has been sent.",
		Data:    nil,
	})
}

// ConfirmPasswordReset validates the reset token and updates the user's password.
func ConfirmPasswordReset(c *fiber.Ctx) error {
	var body ConfirmPasswordResetProps
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(400, "Malformed request")
	}

	if body.Token == "" {
		return fiber.NewError(400, "Reset token is required")
	}

	if body.Password == "" {
		return fiber.NewError(400, "New password is required")
	}

	if len(body.Password) < 8 {
		return fiber.NewError(400, "Password must be at least 8 characters long")
	}

	db := database.GetInstance()

	// Hash the provided token to compare with stored hash
	hashedToken := utils.HashTokenSHA256(body.Token)

	// Find the password reset record
	var passwordReset models.PasswordReset
	err := db.Where("token = ? AND used = false AND expires_at > ?",
		hashedToken, time.Now()).First(&passwordReset).Error

	if err != nil {
		return fiber.NewError(400, "Invalid or expired reset token")
	}

	// Find the user
	var user models.User
	err = db.Where(&models.User{Email: passwordReset.Email}).First(&user).Error
	if err != nil {
		return fiber.NewError(404, "User not found")
	}

	// Hash the new password
	hashedPassword, err := utils.HashPassword(body.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update user password
	err = db.Model(&user).Update("password", hashedPassword).Error
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Mark the reset token as used
	err = db.Model(&passwordReset).Update("used", true).Error
	if err != nil {
		return fmt.Errorf("failed to mark reset token as used: %w", err)
	}

	// Revoke all existing sessions for security
	err = db.Model(&models.Session{}).Where("user_id = ?", user.ID).Update("revoked", true).Error
	if err != nil {
		return fmt.Errorf("failed to revoke sessions: %w", err)
	}

	return c.JSON(utils.Response{
		Success: true,
		Code:    200,
		Message: "Password reset successfully. Please log in with your new password.",
		Data:    nil,
	})
}

func SetupPasswordReset() {
	db = database.GetInstance()
}
