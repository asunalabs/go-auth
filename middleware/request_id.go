package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// RequestID sets a request id on the context locals and the response header.
// It's lightweight and suitable for production; swap UUID generation if you
// prefer short ids.
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := uuid.NewString()
		c.Locals("requestid", id)
		c.Set("X-Request-Id", id)
		// also set a small timestamp header for quick tracing if needed
		c.Set("X-Request-Started-At", time.Now().UTC().Format(time.RFC3339))
		return c.Next()
	}
}
