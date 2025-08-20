package middleware

import (
	"api/utils"
	"errors"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Logger is a lightweight logging interface used by the error handler. The
// standard library logger (*log.Logger) implements this.
type Logger interface {
	Printf(format string, v ...interface{})
}

// NewErrorHandler returns a Fiber error handler that:
// - avoids nil dereferences when the incoming error is not a *fiber.Error
// - logs the original error with a request id, method and path
// - returns a generic message for 5xx responses and exposes details for 4xx
// - attempts a minimal fallback if writing the response fails
func NewErrorHandler(logger Logger) func(*fiber.Ctx, error) error {
	if logger == nil {
		logger = log.Default()
	}

	return func(ctx *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		var fe *fiber.Error
		if errors.As(err, &fe) {
			// errors.As ensures fe won't be dereferenced when nil, but be
			// defensive and only use the code if it's non-zero.
			if fe != nil && fe.Code != 0 {
				code = fe.Code
			}
		}

		// pull request id if set by earlier middleware
		rid := ""
		if v := ctx.Locals("requestid"); v != nil {
			if s, ok := v.(string); ok {
				rid = s
			}
		}

		// structured-ish logging (replace with zap/zerolog when available)
		logger.Printf("time=%s request_id=%s method=%s path=%s status=%d error=%v",
			time.Now().Format(time.RFC3339), rid, ctx.Method(), ctx.Path(), code, err)

		// Only reveal error details for client errors (4xx). Server errors get
		// a generic message to avoid leaking internals.
		msg := "Internal server error"
		if code >= 400 && code < 500 {
			msg = err.Error()
		}

		resp := utils.Response{
			Success: false,
			Code:    uint(code),
			Message: msg,
			Data:    nil,
		}

		if writeErr := ctx.Status(code).JSON(resp); writeErr != nil {
			// If writing the JSON response failed, log and return a minimal
			// fallback body.
			logger.Printf("time=%s request_id=%s error_writing_response=%v",
				time.Now().Format(time.RFC3339), rid, writeErr)

			return ctx.Status(fiber.StatusInternalServerError).JSON(utils.Response{
				Success: false,
				Code:    500,
				Message: "Internal server error",
				Data:    nil,
			})
		}

		return nil
	}
}
