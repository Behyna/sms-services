package middleware

import (
	"errors"

	"github.com/Behyna/sms-services/smsgateway/internal/constants"
	"github.com/Behyna/sms-services/smsgateway/internal/service"
	"github.com/gofiber/fiber/v2"
)

func ErrorHandler() fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		var serviceErr service.Error
		if errors.As(err, &serviceErr) {
			return handleServiceError(c, serviceErr)
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    constants.ErrCodeInternalError,
			"message": constants.GetErrorMessage(constants.ErrCodeInternalError),
		})
	}
}

func handleServiceError(c *fiber.Ctx, err service.Error) error {
	errorCode := err.Code

	status := constants.GetHTTPStatus(errorCode)
	if status == 500 && err.Code != constants.ErrCodeInternalError {
		errorCode = constants.ErrCodeInternalError
	}

	return c.Status(status).JSON(fiber.Map{
		"code":    errorCode,
		"message": constants.GetErrorMessage(errorCode),
	})
}
