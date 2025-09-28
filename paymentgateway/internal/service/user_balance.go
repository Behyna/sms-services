package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Behyna/sms-services/paymentgateway/internal/constants"
	"github.com/Behyna/sms-services/paymentgateway/internal/metrics"
	"github.com/Behyna/sms-services/paymentgateway/internal/model"
	"github.com/Behyna/sms-services/paymentgateway/internal/repository"
	"go.uber.org/zap"
)

var ErrInsufficientBalance = errors.New("INSUFFICIENT_BALANCE")

type UserBalanceService interface {
	CreateUser(ctx context.Context, cmd UserBalanceCommand) (CreateUserResult, error)
	GetBalance(userID string) (model.UserBalance, error)
	IncreaseBalance(ctx context.Context, cmd UserBalanceCommand) (UpdateBalanceResult, error)
	DecreaseBalance(ctx context.Context, cmd UserBalanceCommand) (UpdateBalanceResult, error)
}

type userBalanceService struct {
	txManager       repository.TxManager
	userRepo        repository.UserBalanceRepository
	transactionRepo repository.TransactionRepository
	log             *zap.Logger
	metrics         *metrics.Metrics
}

func NewUserBalanceService(txManager repository.TxManager, userRepo repository.UserBalanceRepository, log *zap.Logger, transactionRepo repository.TransactionRepository, metrics *metrics.Metrics) UserBalanceService {
	return &userBalanceService{txManager: txManager, userRepo: userRepo, log: log, transactionRepo: transactionRepo, metrics: metrics}
}

func (s *userBalanceService) CreateUser(ctx context.Context, cmd UserBalanceCommand) (CreateUserResult, error) {
	start := time.Now()

	ub := model.UserBalance{
		UserID:    cmd.UserID,
		Balance:   cmd.Amount,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	transaction := model.Transaction{
		UserID:         cmd.UserID,
		IdempotencyKey: cmd.IdempotencyKey,
		TxType:         model.TxTypeIncrease,
		Amount:         cmd.Amount,
		CreatedAt:      time.Now(),
	}

	err := s.txManager.WithTx(ctx, func(ctx context.Context) error {
		userBalanceStart := time.Now()
		if err := s.userRepo.Create(ctx, &ub); err != nil {
			s.log.Error("error create user balance", zap.Error(err))
			s.metrics.RecordDBQuery("insert", "user_balances", "error", time.Since(userBalanceStart))

			if errors.Is(err, repository.ErrUserBalanceExists) {
				return NewServiceError(constants.ErrCodeUserExisted, err)
			}

			return NewServiceError(constants.ErrCodeOperationFailed, err)
		}

		s.metrics.RecordDBQuery("insert", "user_balances", "success", time.Since(userBalanceStart))

		transactionStart := time.Now()
		if err := s.transactionRepo.Create(ctx, &transaction); err != nil {
			s.log.Error("error create user transaction", zap.Error(err))
			s.metrics.RecordDBQuery("insert", "transactions", "error", time.Since(transactionStart))
			return NewServiceError(constants.ErrCodeOperationFailed, err)
		}
		s.metrics.RecordDBQuery("insert", "transactions", "success", time.Since(transactionStart))

		return nil
	})

	if err != nil {
		s.log.Error("Failed to create user balance",
			zap.String("user_id", cmd.UserID),
			zap.Int64("initial_balance", cmd.Amount),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)
		return CreateUserResult{}, err
	}

	s.log.Info("User balance created successfully",
		zap.String("user_id", cmd.UserID),
		zap.Int64("initial_balance", cmd.Amount),
		zap.Duration("total_duration", time.Since(start)),
	)

	result := CreateUserResult{
		UserBalance:     ub,
		TransactionID:   transaction.TransactionID,
		TransactionTime: transaction.CreatedAt,
	}

	return result, nil
}

func (s *userBalanceService) GetBalance(userID string) (model.UserBalance, error) {
	start := time.Now()

	userBalance, err := s.userRepo.FindByUserID(userID)
	duration := time.Since(start)

	if err != nil {
		s.log.Error("Failed to get user balance",
			zap.String("user_id", userID),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		s.metrics.RecordDBQuery("select", "user_balances", "error", duration)
		return model.UserBalance{}, err
	}

	s.metrics.RecordDBQuery("select", "user_balances", "success", duration)
	s.metrics.UpdateUserBalance(fmt.Sprintf("%s", userID), userBalance.Balance)

	s.log.Debug("User balance retrieved successfully",
		zap.String("user_id", userID),
		zap.Int64("balance", userBalance.Balance),
		zap.Duration("duration", duration),
	)

	return userBalance, nil
}

func (s *userBalanceService) IncreaseBalance(ctx context.Context, cmd UserBalanceCommand) (UpdateBalanceResult, error) {
	ub, err := s.userRepo.FindByUserID(cmd.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserBalanceNotFound) {
			return UpdateBalanceResult{}, NewServiceError(constants.ErrCodeUserNotFound, err)
		}
		return UpdateBalanceResult{}, NewServiceError(constants.ErrCodeOperationFailed, err)
	}

	transaction := model.Transaction{
		UserID:         cmd.UserID,
		IdempotencyKey: cmd.IdempotencyKey,
		TxType:         model.TxTypeIncrease,
		Amount:         cmd.Amount,
		CreatedAt:      time.Now(),
	}

	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		err := s.transactionRepo.Create(ctx, &transaction)
		if err != nil {
			s.log.Error("error create user transaction", zap.Error(err))
			return NewServiceError(constants.ErrCodeOperationFailed, err)
		}

		ub.Balance += cmd.Amount
		ub.UpdatedAt = time.Now()

		if err := s.userRepo.UpdateBalance(ctx, ub.UserID, ub.Balance); err != nil {
			s.log.Error("error create user balance", zap.Error(err))
			return NewServiceError(constants.ErrCodeOperationFailed, err)
		}

		return nil
	})

	if err == nil {
		result := UpdateBalanceResult{
			UserBalance:     ub,
			TransactionID:   transaction.TransactionID,
			TransactionTime: transaction.CreatedAt,
		}

		return result, nil
	}

	if !errors.Is(err, repository.ErrTransactionExisted) {
		return UpdateBalanceResult{}, err
	}

	existedTransaction, err := s.transactionRepo.GetByIdempotencyKey(model.TxTypeIncrease, cmd.IdempotencyKey)
	if err != nil {
		s.log.Error("error get transaction by idempotency key", zap.Error(err))
		return UpdateBalanceResult{}, NewServiceError(constants.ErrCodeOperationFailed, err)
	}

	s.log.Info("Idempotent request transaction already exists",
		zap.String("idempotency_key", cmd.IdempotencyKey))

	result := UpdateBalanceResult{
		UserBalance:     ub,
		TransactionID:   existedTransaction.TransactionID,
		TransactionTime: existedTransaction.CreatedAt,
	}

	return result, nil
}

func (s *userBalanceService) DecreaseBalance(ctx context.Context, cmd UserBalanceCommand) (UpdateBalanceResult, error) {
	ub, err := s.userRepo.FindByUserID(cmd.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserBalanceNotFound) {
			return UpdateBalanceResult{}, NewServiceError(constants.ErrCodeUserNotFound, err)
		}
		return UpdateBalanceResult{}, NewServiceError(constants.ErrCodeOperationFailed, err)
	}

	if ub.Balance-cmd.Amount < 0 {
		return UpdateBalanceResult{}, NewServiceError(constants.ErrCodeInsufficientBalance, ErrInsufficientBalance)
	}

	transaction := model.Transaction{
		UserID:         cmd.UserID,
		IdempotencyKey: cmd.IdempotencyKey,
		TxType:         model.TxTypeDecrease,
		Amount:         cmd.Amount,
		CreatedAt:      time.Now(),
	}

	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		if err := s.transactionRepo.Create(ctx, &transaction); err != nil {
			s.log.Error("error create user transaction", zap.Error(err))
			return NewServiceError(constants.ErrCodeOperationFailed, err)
		}

		ub.Balance -= cmd.Amount
		ub.UpdatedAt = time.Now()

		if err := s.userRepo.UpdateBalance(ctx, ub.UserID, ub.Balance); err != nil {
			s.log.Error("error create user balance", zap.Error(err))
			return NewServiceError(constants.ErrMsgOperationFailed, err)
		}

		return nil
	})

	if err == nil {
		result := UpdateBalanceResult{
			UserBalance:     ub,
			TransactionID:   transaction.TransactionID,
			TransactionTime: transaction.CreatedAt,
		}

		return result, nil
	}

	if !errors.Is(err, repository.ErrTransactionExisted) {
		return UpdateBalanceResult{}, err
	}

	existedTransaction, err := s.transactionRepo.GetByIdempotencyKey(model.TxTypeDecrease, cmd.IdempotencyKey)
	if err != nil {
		s.log.Error("error get transaction by idempotency key", zap.Error(err))
		return UpdateBalanceResult{}, NewServiceError(constants.ErrCodeOperationFailed, err)
	}

	s.log.Info("Idempotent request transaction already exists",
		zap.String("idempotency_key", cmd.IdempotencyKey))

	result := UpdateBalanceResult{
		UserBalance:     ub,
		TransactionID:   existedTransaction.TransactionID,
		TransactionTime: existedTransaction.CreatedAt,
	}

	return result, nil
}
