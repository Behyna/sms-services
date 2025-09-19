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
	CreateUser(ctx context.Context, cmd UserBalanceCommand) (*model.UserBalance, error)
	GetBalance(userID string) (model.UserBalance, error)
	IncreaseBalance(ctx context.Context, cmd UserBalanceCommand) (*model.UserBalance, error)
	DecreaseBalance(ctx context.Context, cmd UserBalanceCommand) (*model.UserBalance, error)
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

func (s *userBalanceService) CreateUser(ctx context.Context, cmd UserBalanceCommand) (*model.UserBalance, error) {
	start := time.Now()
	createAt := time.Now()
	ub := &model.UserBalance{
		UserID:    cmd.UserID,
		Balance:   cmd.Amount,
		UpdatedAt: createAt,
		CreatedAt: createAt,
	}
	transaction := &model.Transaction{
		UserID:    cmd.UserID,
		Amount:    cmd.Amount,
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
			zap.String("user_id", cmd.UserID),
			zap.Int64("initial_balance", cmd.Amount),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)
		return &model.UserBalance{}, err
	}

	s.log.Info("User balance created successfully",
		zap.String("user_id", cmd.UserID),
		zap.Int64("initial_balance", cmd.Amount),
		zap.Duration("total_duration", time.Since(start)),
	)

	return ub, nil
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

func (s *userBalanceService) IncreaseBalance(ctx context.Context, cmd UserBalanceCommand) (*model.UserBalance, error) {
	ub, err := s.userRepo.FindByUserID(cmd.UserID)
	if err != nil {
		return nil, err
	}

	err = s.transactionRepo.Transaction(ctx, func(ctx context.Context) error {
		createdAt := time.Now()
		transaction := &model.Transaction{
			UserID:    cmd.UserID,
			Amount:    cmd.Amount,
			TxType:    model.TxTypeIncrease,
			CreatedAt: createdAt,
		}
		if err := s.transactionRepo.Create(transaction); err != nil {
			s.log.Error("error create user transaction", zap.Error(err))
			return err
		}

		ub.Balance += cmd.Amount
		ub.UpdatedAt = createdAt

		if err := s.userRepo.UpdateBalance(ub.UserID, ub.Balance); err != nil {
			s.log.Error("error create user balance", zap.Error(err))
			return err
		}
		return nil
	})

	return &ub, err
}

func (s *userBalanceService) DecreaseBalance(ctx context.Context, cmd UserBalanceCommand) (*model.UserBalance, error) {
	ub, err := s.userRepo.FindByUserID(cmd.UserID)
	if err != nil {
		return nil, err
	}

	if ub.Balance-cmd.Amount < 0 {
		return nil, constants.ErrInsufficientBalance
	}

	err = s.transactionRepo.Transaction(ctx, func(ctx context.Context) error {
		createdAt := time.Now()
		transaction := &model.Transaction{
			UserID:    cmd.UserID,
			Amount:    cmd.Amount,
			TxType:    model.TxTypeDecrease,
			CreatedAt: createdAt,
		}
		if err := s.transactionRepo.Create(transaction); err != nil {
			s.log.Error("error create user transaction", zap.Error(err))
			return err
		}

		ub.Balance -= cmd.Amount
		ub.UpdatedAt = createdAt

		if err := s.userRepo.UpdateBalance(ub.UserID, ub.Balance); err != nil {
			s.log.Error("error create user balance", zap.Error(err))
			return err
		}
		return nil
	})

	return &ub, err
}
