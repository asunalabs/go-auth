package handlers

import (
	"api/database"
	"api/database/models"
	"api/utils"
	"context"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var db *gorm.DB

type RegisterProps struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginProps struct {
	Email    string
	Password string
}

func Register(c *fiber.Ctx) error {
	var body RegisterProps

	err := c.BodyParser(&body)

	if err != nil {
		return fiber.NewError(400, "Malformed request")
	}

	// Validate required fields
	if body.Username == "" {
		return fiber.NewError(400, "Username is required")
	}
	if body.Email == "" {
		return fiber.NewError(400, "Email is required")
	}
	if body.Password == "" {
		return fiber.NewError(400, "Password is required")
	}

	// Basic username validation
	if len(body.Username) < 3 {
		return fiber.NewError(400, "Username must be at least 3 characters long")
	}
	if len(body.Username) > 255 {
		return fiber.NewError(400, "Username must be less than 255 characters")
	}

	hash, err := utils.HashPassword(body.Password)

	if err != nil {
		// return the original error so the global error handler will log it
		// and return a generic 500 to the client
		return err
	}

	user := models.User{
		Username: body.Username,
		Email:    body.Email,
		Password: hash,
	}

	err = db.Create(&user).Error

	if err != nil {
		return fiber.NewError(400, "User with this email or username already exists")
	}

	jti, jwt, err := utils.GetSignedKey(user.ID)

	if err != nil {
		return err
	}

	refreshToken, hashedToken := utils.GenerateRefreshToken()

	session := models.Session{
		JTI:          jti,
		UserID:       user.ID,
		RefreshToken: hashedToken,
		Revoked:      false,
		ExpiresAt:    time.Now().Add(30 * 24 * time.Hour),
	}
	db.Create(&session)

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   os.Getenv("ENV") == "production",
	})

	// Send a welcome email asynchronously. Do not block registration on email delivery.
	go func(email string) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client := utils.NewSMTPClient()
		subject := "Welcome to Asuna Labs"
		body := "Welcome! Your account has been created successfully.\n\nThanks for joining."
		if err := client.Send(ctx, []string{email}, subject, body); err != nil {
			// Best-effort logging via standard error output; keep registration successful.
			// In an enterprise setup, replace with structured logger/metrics.
			_ = err
		}
	}(user.Email)

	return c.JSON(utils.Response{
		Success: true,
		Code:    200,
		Message: "Registered Successfully",
		Data: struct {
			Token string `json:"token"`
		}{
			Token: jwt,
		},
	})
}

func Login(c *fiber.Ctx) error {
	var body LoginProps
	err := c.BodyParser(&body)
	if err != nil {
		return fiber.NewError(400, "Malformed request")
	}

	var user models.User
	err = db.Where(&models.User{Email: body.Email}).First(&user).Error
	if err != nil {
		return fiber.NewError(404, "User not found")
	}

	// Verify the password against the stored hash
	if !utils.ComparePassword(body.Password, user.Password) {
		return fiber.NewError(401, "Invalid credentials")
	}

	jti, jwt, err := utils.GetSignedKey(user.ID)

	if err != nil {
		return err
	}

	refreshToken, hashedToken := utils.GenerateRefreshToken()

	session := models.Session{
		JTI:          jti,
		UserID:       user.ID,
		RefreshToken: hashedToken,
		Revoked:      false,
		ExpiresAt:    time.Now().Add(30 * 24 * time.Hour),
	}
	db.Create(&session)

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   os.Getenv("ENV") == "production",
	})

	return c.JSON(utils.Response{
		Success: true,
		Code:    200,
		Message: "Signed in successfully",
		Data: fiber.Map{
			"token": jwt,
		},
	})
}

func RefreshToken(c *fiber.Ctx) error {
	refreshToken := c.Cookies("refresh_token")

	if refreshToken == "" {
		return fiber.NewError(400, "Missing refresh_token")
	}

	hash := utils.HashTokenSHA256(refreshToken)

	var session models.Session

	err := db.Where(&models.Session{RefreshToken: hash}).First(&session).Error

	if err != nil {
		return fiber.NewError(401, "Unauthorized")
	}

	if session.Revoked {
		return fiber.NewError(401, "Unauthorized: Refresh token revoked")
	}

	if session.ExpiresAt.Before(time.Now()) {
		session.Revoked = true
		db.Save(&session)

		return fiber.NewError(401, "Unauthorized: Refresh token expired")
	}

	jti, jwt, err := utils.GetSignedKey(session.UserID)

	if err != nil {
		return err
	}

	session.JTI = jti
	db.Save(&session)

	return c.JSON(utils.Response{
		Success: true,
		Code:    200,
		Message: "Success",
		Data: fiber.Map{
			"token": jwt,
		},
	})
}

func RevokeToken(c *fiber.Ctx) error {
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		return fiber.NewError(400, "Missing refresh_token")
	}

	var session models.Session
	err := db.Where(&models.Session{RefreshToken: utils.HashTokenSHA256(refreshToken)}).First(&session).Error
	if err != nil {
		return fiber.NewError(404, "Invalid token")
	}

	session.Revoked = true
	db.Save(&session)

	c.ClearCookie("refresh_token")

	return c.Status(fiber.StatusOK).JSON(utils.Response{
		Success: true,
		Code:    200,
		Message: "Token revoked",
		Data:    nil,
	})
}

func SetupAuth() {
	db = database.GetInstance()
}
