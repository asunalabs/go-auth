package handlers

import (
	"api/database"
	"api/database/models"
	"api/utils"
	"os"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var db *gorm.DB

type RegisterProps struct {
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

func SetupAuth() {
	db = database.GetInstance()
}