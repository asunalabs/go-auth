package handlers

import (
	"api/database"
	"api/database/models"
	"api/utils"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var db *gorm.DB

type RegisterProps struct {
	Email string
	Password string
}

type LoginProps struct {
	Email string
	Password string
}

func Register(c *fiber.Ctx) error {
	var body RegisterProps

	err := c.BodyParser(&body)

	if err != nil {
		return c.JSON(utils.Response{
			Success: false,
			Code: 400,
			Message: "Malformed request",
			Data: nil,
		})
	}

	hash, err := utils.HashPassword(body.Password)

	if err != nil {
		return c.JSON(utils.Response{
			Success: false,
			Code: 500,
			Message: "Internal server error",
			Data: err.Error(),
		})
	}	

	user := models.User{
		Email: body.Email,
		Password: hash,
	}

	err = db.Create(&user).Error

	if err != nil {
		return c.JSON(utils.Response{
			Success: false,
			Code: 400,
			Message: "User with this email already exists",
			Data: nil,
		})
	}
;
	jti, jwt, err := utils.GetSignedKey(user.ID)

	if err != nil {
		return c.JSON(utils.Response{
			Success: false,
			Code: 500,
			Message: "Internal server error",
			Data: err.Error(),
		})
	}

	refreshToken, hashedToken := utils.GenerateRefreshToken()

	session := models.Session{
		JTI: jti,
		UserID: user.ID,
		RefreshToken: hashedToken,
		Revoked: false,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	db.Create(&session)

	c.Cookie(&fiber.Cookie{
		Name: "refresh_token",
		Value: refreshToken,
		HTTPOnly: true,
		SameSite: "Lax",
		Secure: os.Getenv("ENV") == "production",
	})

	return c.JSON(utils.Response{
		Success: true,
		Code: 200,
		Message: "Registered Successfully",
		Data: struct{
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
		return c.JSON(utils.Response{
			Success: false,
			Code: 400,
			Message: "Malformed request",
			Data: nil,
		})
	}

	var user models.User
	err = db.Where(&models.User{Email: body.Email}).First(&user).Error
	if err != nil {
		return c.JSON(utils.Response{
			Success: false,
			Code: 404,
			Message: "User not found",
			Data: nil,
		})
	}

	jti, jwt, err := utils.GetSignedKey(user.ID)

	if err != nil {
		return c.JSON(utils.Response{
			Success: false,
			Code: 500,
			Message: "Internal server error",
			Data: err.Error(),
		})
	}

	refreshToken, hashedToken := utils.GenerateRefreshToken()

	session := models.Session{
		JTI: jti,
		UserID: user.ID,
		RefreshToken: hashedToken,
		Revoked: false,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	db.Create(&session)

	c.Cookie(&fiber.Cookie{
		Name: "refresh_token",
		Value: refreshToken,
		HTTPOnly: true,
		SameSite: "Lax",
		Secure: os.Getenv("ENV") == "production",
	})

	return c.JSON(utils.Response{
		Success: true,
		Code: 200,
		Message: "Signed in successfully",
		Data: fiber.Map{
			"token":jwt,
		},
	})
}

func RefreshToken(c *fiber.Ctx) error {
	refreshToken := c.Cookies("refresh_token")

	if refreshToken == "" {
		return c.JSON(utils.Response{
			Success: false,
			Code: 400,
			Message: "Missing refresh_token",
			Data: nil,
		})
	}

	hash := utils.HashTokenSHA256(refreshToken)

	var session models.Session

	err := db.Where(&models.Session{RefreshToken: hash}).First(&session).Error

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.Response{
			Success: false,
			Code: 401,
			Message: "Unauthorized",
			Data: nil,
		})
	}

	if session.Revoked {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.Response{
			Success: false,
			Code: 401,
			Message: "Unauthorized: Refresh token revoked",
			Data: nil,
		})
	}

	if session.ExpiresAt.Before(time.Now()) {
		session.Revoked = true
		db.Save(&session)

		return c.Status(fiber.StatusUnauthorized).JSON(utils.Response{
			Success: false,
			Code: 401,
			Message: "Unauthorized: Refresh token expired",
			Data: nil,
		})
	}

	jti, jwt, err := utils.GetSignedKey(session.UserID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.Response{
			Success: false,
			Code: 500,
			Message: "Internal server error",
			Data: nil,
		})
	}

	session.JTI = jti
	db.Save(&session)


	return c.JSON(utils.Response{
		Success: true,
		Code: 200,
		Message: "Success",
		Data: fiber.Map{
			"token" : jwt,
		},
	})
}

func SetupAuth() {
	db = database.GetInstance()
}