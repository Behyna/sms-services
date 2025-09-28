package validator

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Behyna/sms-services/paymentgateway/internal/api/contract"
	"github.com/Behyna/sms-services/paymentgateway/internal/metrics"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

const (
	sep = " and "
)

type Error struct {
	Error       bool
	FailedField string
	Tag         string
	Value       interface{}
}

type IXValidator interface {
	Validator(data any, message string, c *fiber.Ctx) (responseErr contract.Response)
	Validate(data interface{}) []Error
}

type XValidator struct {
	validator *validator.Validate
	metrics   *metrics.Metrics
}

func NewXValidator(validator *validator.Validate, metrics *metrics.Metrics) IXValidator {
	for key, function := range valid {
		validator.RegisterValidation(key, function)
	}

	return &XValidator{
		validator: validator,
		metrics:   metrics,
	}
}

func (x XValidator) Validator(data any, message string, c *fiber.Ctx) (responseErr contract.Response) {
	start := time.Now()

	c.BodyParser(&data)
	if errs := x.Validate(data); len(errs) > 0 && errs[0].Error {
		errMsgs := make([]string, 0)
		for _, err := range errs {
			errMsgs = append(errMsgs, fmt.Sprintf(
				message,
				err.FailedField,
			))

			if x.metrics != nil {
				x.metrics.RecordValidationError(err.FailedField, err.Tag)
			}
		}
		errMess := strings.Join(errMsgs, sep)
		c.Status(http.StatusUnprocessableEntity)

		if x.metrics != nil {
			x.metrics.RecordValidationDuration("validation_error", time.Since(start))
		}

		return contract.Response{
			Code:    "1",
			Message: errMess,
		}
	}

	if x.metrics != nil {
		x.metrics.RecordValidationDuration("validation_success", time.Since(start))
	}

	return responseErr
}

func (x XValidator) Validate(data interface{}) []Error {
	var validationErrors []Error

	errs := x.validator.Struct(data)
	if errs != nil {
		for _, err := range errs.(validator.ValidationErrors) {
			var elem Error
			elem.FailedField = err.Field()
			elem.Tag = err.Tag()
			elem.Value = err.Value()
			elem.Error = true
			validationErrors = append(validationErrors, elem)
		}
	}
	return validationErrors
}
