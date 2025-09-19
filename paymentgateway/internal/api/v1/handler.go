package v1

import (
	"fmt"
	"time"

	"github.com/Behyna/sms-services/paymentgateway/internal/api/contract"
	"github.com/Behyna/sms-services/paymentgateway/internal/api/validator"
	"github.com/Behyna/sms-services/paymentgateway/internal/constants"
	"github.com/Behyna/sms-services/paymentgateway/internal/metrics"
	"github.com/Behyna/sms-services/paymentgateway/internal/service"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Handler struct {
	logger      *zap.Logger
	userService service.UserBalanceService
	XValidator  validator.IXValidator
	metrics     *metrics.Metrics
}

func NewHandler(logger *zap.Logger, userService service.UserBalanceService, XValidator validator.IXValidator, metrics *metrics.Metrics) *Handler {
	return &Handler{
		logger:      logger,
		userService: userService,
		XValidator:  XValidator,
		metrics:     metrics,
	}
}

func (h *Handler) Pong(c *fiber.Ctx) error {
	return c.SendString("pong")
}

func (h *Handler) CreateUsersBalance(c *fiber.Ctx) error {
	start := time.Now()
	var (
		paymentErr     contract.ResponseError
		res            contract.Response
		handlerRequest CreateUserBalanceRequest
	)

	// Record validation start time
	validationStart := time.Now()
	responseError := h.XValidator.Validator(&handlerRequest, constants.MessageErrorFormat, c)
	h.metrics.RecordValidationDuration("create_user_balance", time.Since(validationStart))

	if responseError != paymentErr {
		h.logger.Error("Error Validator", zap.Any("request", handlerRequest))
		h.metrics.RecordValidationError("user_balance", "validation_failed")
		responseError.Code = constants.ValidationFailed
		return c.JSON(responseError)
	}

	cmd := service.UserBalanceCommand{
		UserID: handlerRequest.UserID,
		Amount: handlerRequest.InitialBalance,
	}

	userBalance, err := h.userService.CreateUser(c.UserContext(), cmd)
	if err != nil {
		h.logger.Error("Error user service create balance", zap.Error(err))
		h.metrics.RecordUserBalanceCreationError()
		h.metrics.RecordTransactionError("topup", "creation_failed")
		paymentErr.Successful = true
		paymentErr.Code = "duplicate"
		paymentErr.Message = "user  balance already exists"
		paymentErr.Error = err.Error()
		return c.JSON(paymentErr)
	}

	// Record successful metrics
	h.metrics.RecordUserBalanceCreated()
	h.metrics.RecordTransactionCreated("topup")
	h.metrics.UpdateUserBalance(fmt.Sprintf("%s", cmd.UserID), cmd.Amount)

	h.logger.Info("User balance created successfully",
		zap.String("user_id", cmd.UserID),
		zap.Int64("initial_balance", cmd.Amount),
		zap.Duration("duration", time.Since(start)),
	)

	res.Successful = true
	res.Code = "success"
	res.Message = "user balance created successfully"
	res.Result = userBalance

	return c.JSON(res)
}

func (h *Handler) GetUserBalance(c *fiber.Ctx) error {
	start := time.Now()
	var (
		paymentErr     contract.ResponseError
		res            contract.Response
		handlerRequest GetUserBalanceRequest
	)

	// Record validation start time
	validationStart := time.Now()
	responseError := h.XValidator.Validator(&handlerRequest, constants.MessageErrorFormat, c)
	h.metrics.RecordValidationDuration("get_user_balance", time.Since(validationStart))

	if responseError != paymentErr {
		h.logger.Error("Error Validator", zap.Any("request", handlerRequest))
		h.metrics.RecordValidationError("user_balance", "validation_failed")
		responseError.Code = constants.ValidationFailed
		return c.JSON(responseError)
	}

	userBalance, err := h.userService.GetBalance(handlerRequest.UserID)
	if err != nil {
		h.logger.Error("Error getting user balance", zap.Error(err))
		h.metrics.RecordBalanceRetrieval("error")
		paymentErr.Successful = false
		paymentErr.Code = "not_found"
		paymentErr.Message = "user balance not found"
		paymentErr.Error = err.Error()
		return c.JSON(paymentErr)
	}

	// Record successful balance retrieval
	h.metrics.RecordBalanceRetrieval("success")
	h.metrics.UpdateUserBalance(fmt.Sprintf("%d", handlerRequest.UserID), userBalance.Balance)

	h.logger.Info("User balance retrieved successfully",
		zap.String("user_id", handlerRequest.UserID),
		zap.Int64("balance", userBalance.Balance),
		zap.Duration("duration", time.Since(start)),
	)

	res.Successful = true
	res.Code = "success"
	res.Message = "user balance retrieved successfully"
	res.Result = userBalance
	return c.JSON(res)
}

func (h *Handler) UpdateUserBalance(c *fiber.Ctx) error {
	var (
		paymentErr     contract.ResponseError
		res            contract.Response
		handlerRequest UpdateUserBalanceRequest
	)

	responseError := h.XValidator.Validator(&handlerRequest, constants.MessageErrorFormat, c)

	if responseError != paymentErr {
		h.logger.Error("Error Validator", zap.Any("request", handlerRequest))
		responseError.Code = constants.ValidationFailed
		return c.JSON(responseError)
	}

	cmd := service.UserBalanceCommand{
		UserID: handlerRequest.UserID,
		Amount: handlerRequest.Amount,
	}

	userBalance, err := h.userService.IncreaseBalance(c.UserContext(), cmd)
	if err != nil {
		h.logger.Error("Error updating user balance", zap.Error(err))
		paymentErr.Successful = false
		paymentErr.Code = "not_found"
		paymentErr.Message = "user balance not found"
		paymentErr.Error = err.Error()
		return c.JSON(paymentErr)
	}

	res.Successful = true
	res.Code = "success"
	res.Message = "user balance updated successfully"
	res.Result = userBalance
	return c.JSON(res)
}

func (h *Handler) DecreaseUserBalance(c *fiber.Ctx) error {
	var (
		paymentErr     contract.ResponseError
		res            contract.Response
		handlerRequest UpdateUserBalanceRequest
	)

	responseError := h.XValidator.Validator(&handlerRequest, constants.MessageErrorFormat, c)

	if responseError != paymentErr {
		h.logger.Error("Error Validator", zap.Any("request", handlerRequest))
		responseError.Code = constants.ValidationFailed
		return c.JSON(responseError)
	}

	cmd := service.UserBalanceCommand{
		UserID: handlerRequest.UserID,
		Amount: handlerRequest.Amount,
	}

	userBalance, err := h.userService.DecreaseBalance(c.UserContext(), cmd)
	if err != nil {
		h.logger.Error("Error updating user balance", zap.Error(err))
		paymentErr.Successful = false
		paymentErr.Code = "not_found"
		paymentErr.Message = "user balance not found"
		paymentErr.Error = err.Error()
		return c.JSON(paymentErr)
	}

	res.Successful = true
	res.Code = "success"
	res.Message = "user balance updated successfully"
	res.Result = userBalance
	return c.JSON(res)
}
