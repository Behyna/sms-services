package errors

import (
	"errors"

	"github.com/Behyna/sms-services/paymentgateway/internal/constants"
	"github.com/Behyna/sms-services/paymentgateway/internal/service"
	"github.com/gofiber/fiber/v2"
)

func ErrorHandler() fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		var serviceErr service.Error
		if errors.As(err, &serviceErr) {
			return handleServiceError(c, serviceErr)
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Internal server error",
			"message": "Could not process the request",
		})
	}
}

func handleServiceError(c *fiber.Ctx, err service.Error) error {
	statusMap := map[string]int{
		constants.ErrCodeUserExisted:         fiber.StatusConflict,
		constants.ErrCodeUserNotFound:        fiber.StatusNotFound,
		constants.ErrCodeInsufficientBalance: fiber.StatusConflict,
		constants.ErrCodeOperationFailed:     fiber.StatusInternalServerError,
	}

	status := statusMap[err.Code]

	return c.Status(status).JSON(fiber.Map{
		"code":    err.Code,
		"message": constants.GetErrorMessage(err.Code),
	})
}
