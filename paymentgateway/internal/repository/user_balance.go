package repository

import (
	"errors"

	"github.com/Behyna/sms-services/paymentgateway/internal/constants"
	"github.com/Behyna/sms-services/paymentgateway/internal/model"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type UserBalanceRepository interface {
	Create(ub *model.UserBalance) error
	FindByUserID(userID string) (model.UserBalance, error)
	UpdateBalance(userID string, newBalance int64) error
}

type userBalance struct {
	db *gorm.DB
}

func NewUserBalanceRepository(db *gorm.DB) UserBalanceRepository {
	return &userBalance{db: db}
}

func (r *userBalance) Create(ub *model.UserBalance) error {
	if err := r.db.Create(&ub).Error; err != nil {
		var mysqlErr *mysql.MySQLError

		if errors.As(err, &mysqlErr) {
			if mysqlErr.Number == 1062 {
				return constants.ErrUserBalanceAlreadyExists
			}
		}
	}
	return nil
}

func (r *userBalance) FindByUserID(userID string) (model.UserBalance, error) {
	var ub model.UserBalance
	if err := r.db.Where("user_id = ?", userID).First(&ub).Error; err != nil {
		return model.UserBalance{}, err
	}
	return ub, nil
}

func (r *userBalance) UpdateBalance(userID string, newBalance int64) error {
	if err := r.db.Model(&model.UserBalance{}).
		Where("user_id = ?", userID).
		Update("balance", newBalance).Error; err != nil {
		return err
	}
	return nil
}
