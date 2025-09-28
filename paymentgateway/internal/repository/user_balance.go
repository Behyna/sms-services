package repository

import (
	"context"
	"errors"

	"github.com/Behyna/sms-services/paymentgateway/internal/model"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var ErrUserBalanceNotFound = errors.New("USER_BALANCE_NOT_FOUND")
var ErrUserBalanceExists = errors.New("USER_BALANCE_EXISTED")

type UserBalanceRepository interface {
	Create(ctx context.Context, ub *model.UserBalance) error
	FindByUserID(userID string) (model.UserBalance, error)
	UpdateBalance(ctx context.Context, userID string, newBalance int64) error
}

type userBalance struct {
	db *gorm.DB
}

func NewUserBalanceRepository(db *gorm.DB) UserBalanceRepository {
	return &userBalance{db: db}
}

func (r *userBalance) Create(ctx context.Context, ub *model.UserBalance) error {
	db := GetTx(ctx, r.db)
	err := db.Create(ub).Error
	if err == nil {
		return nil
	}

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return ErrUserBalanceExists
	}

	return err
}

func (r *userBalance) FindByUserID(userID string) (model.UserBalance, error) {
	var ub model.UserBalance
	err := r.db.Where("user_id = ?", userID).First(&ub).Error
	if err == nil {
		return ub, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.UserBalance{}, ErrUserBalanceNotFound
	}

	return model.UserBalance{}, err
}

func (r *userBalance) UpdateBalance(ctx context.Context, userID string, newBalance int64) error {
	db := GetTx(ctx, r.db)
	if err := db.Model(&model.UserBalance{}).
		Where("user_id = ?", userID).
		Update("balance", newBalance).Error; err != nil {
		return err
	}

	return nil
}
