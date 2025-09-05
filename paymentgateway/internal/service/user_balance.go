package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Behyna/sms-services/paymentgateway/internal/constants"
	"github.com/Behyna/sms-services/paymentgateway/internal/metrics"
	"github.com/Behyna/sms-services/paymentgateway/internal/model"
	"github.com/Behyna/sms-services/paymentgateway/internal/repository"
	"go.uber.org/zap"
)

type UserBalanceService interface {
	CreateUser(ctx context.Context, userID int64, initialBalance float64) (*model.UserBalance, error)
	GetBalance(userID int64) (model.UserBalance, error)
	IncreaseBalance(ctx context.Context, userID int64, amount float64) (*model.UserBalance, error)
	DecreaseBalance(ctx context.Context, userID int64, amount float64) (*model.UserBalance, error)
}

type userBalanceService struct {
	log             *zap.Logger
	userRepo        repository.UserBalanceRepository
	transactionRepo repository.TransactionRepository
	metrics         *metrics.Metrics
}

func NewUserBalanceService(userRepo repository.UserBalanceRepository, log *zap.Logger, transactionRepo repository.TransactionRepository, metrics *metrics.Metrics) UserBalanceService {
	return &userBalanceService{userRepo: userRepo, log: log, transactionRepo: transactionRepo, metrics: metrics}
}

func (s *userBalanceService) CreateUser(ctx context.Context, userID int64, initialBalance float64) (*model.UserBalance, error) {
	start := time.Now()
	createAt := time.Now()
	ub := &model.UserBalance{
		UserID:    userID,
		Balance:   initialBalance,
		UpdatedAt: createAt,
		CreatedAt: createAt,
	}
	transaction := &model.Transaction{
		UserID:    userID,
		Amount:    initialBalance,
		TxType:    model.TxTypeIncrease,
		CreatedAt: createAt,
	}

	err := s.transactionRepo.Transaction(ctx, func(ctx context.Context) error {
		// Record transaction creation metrics
		transactionStart := time.Now()
		if err := s.transactionRepo.Create(transaction); err != nil {
			s.log.Error("error create user transaction", zap.Error(err))
			s.metrics.RecordDBQuery("insert", "transactions", "error", time.Since(transactionStart))
			return err
		}
		s.metrics.RecordDBQuery("insert", "transactions", "success", time.Since(transactionStart))

		// Record user balance creation metrics
		userBalanceStart := time.Now()
		if err := s.userRepo.Create(ub); err != nil {
			s.log.Error("error create user balance", zap.Error(err))
			s.metrics.RecordDBQuery("insert", "user_balances", "error", time.Since(userBalanceStart))
			return err
		}
		s.metrics.RecordDBQuery("insert", "user_balances", "success", time.Since(userBalanceStart))

		return nil
	})
	if err != nil {
		s.log.Error("Failed to create user balance",
			zap.Int64("user_id", userID),
			zap.Float64("initial_balance", initialBalance),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)
		return &model.UserBalance{}, err
	}

	s.log.Info("User balance created successfully",
		zap.Int64("user_id", userID),
		zap.Float64("initial_balance", initialBalance),
		zap.Duration("total_duration", time.Since(start)),
	)

	return ub, nil
}

func (s *userBalanceService) GetBalance(userID int64) (model.UserBalance, error) {
	start := time.Now()

	userBalance, err := s.userRepo.FindByUserID(userID)
	duration := time.Since(start)

	if err != nil {
		s.log.Error("Failed to get user balance",
			zap.Int64("user_id", userID),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		s.metrics.RecordDBQuery("select", "user_balances", "error", duration)
		return model.UserBalance{}, err
	}

	s.metrics.RecordDBQuery("select", "user_balances", "success", duration)
	s.metrics.UpdateUserBalance(fmt.Sprintf("%d", userID), userBalance.Balance)

	s.log.Debug("User balance retrieved successfully",
		zap.Int64("user_id", userID),
		zap.Float64("balance", userBalance.Balance),
		zap.Duration("duration", duration),
	)

	return userBalance, nil
}

func (s *userBalanceService) IncreaseBalance(ctx context.Context, userID int64, amount float64) (*model.UserBalance, error) {
	ub, err := s.userRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	err = s.transactionRepo.Transaction(ctx, func(ctx context.Context) error {
		createdAt := time.Now()
		transaction := &model.Transaction{
			UserID:    userID,
			Amount:    amount,
			TxType:    model.TxTypeIncrease,
			CreatedAt: createdAt,
		}
		if err := s.transactionRepo.Create(transaction); err != nil {
			s.log.Error("error create user transaction", zap.Error(err))
			return err
		}

		ub.Balance += amount
		ub.UpdatedAt = createdAt

		if err := s.userRepo.UpdateBalance(ub.UserID, ub.Balance); err != nil {
			s.log.Error("error create user balance", zap.Error(err))
			return err
		}
		return nil
	})

	return &ub, err
}

func (s *userBalanceService) DecreaseBalance(ctx context.Context, userID int64, amount float64) (*model.UserBalance, error) {
	ub, err := s.userRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	if ub.Balance-amount < 0 {
		return nil, constants.ErrInsufficientBalance
	}

	err = s.transactionRepo.Transaction(ctx, func(ctx context.Context) error {
		createdAt := time.Now()
		transaction := &model.Transaction{
			UserID:    userID,
			Amount:    amount,
			TxType:    model.TxTypeDecrease,
			CreatedAt: createdAt,
		}
		if err := s.transactionRepo.Create(transaction); err != nil {
			s.log.Error("error create user transaction", zap.Error(err))
			return err
		}

		ub.Balance -= amount
		ub.UpdatedAt = createdAt

		if err := s.userRepo.UpdateBalance(ub.UserID, ub.Balance); err != nil {
			s.log.Error("error create user balance", zap.Error(err))
			return err
		}
		return nil
	})

	return &ub, err
}
