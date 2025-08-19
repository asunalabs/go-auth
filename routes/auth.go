package routes

import (
	"api/utils"

	"github.com/gofiber/fiber/v2"
)

type RegisterProps struct {
	Email string
	Password string
}

func Register(c *fiber.Ctx) error {
	var body RegisterProps

	err := c.BodyParser(&body)

	if err != nil {
		return c.JSON(utils.Response{
			
		})
	}
}