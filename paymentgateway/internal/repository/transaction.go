package repository

import (
	"context"
	"errors"

	"github.com/Behyna/sms-services/paymentgateway/internal/constants"
	"github.com/Behyna/sms-services/paymentgateway/internal/model"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type TransactionRepository interface {
	Create(ub *model.Transaction) error
	Transaction(ctx context.Context, txBody func(ctx context.Context) error) error
}

type transaction struct {
	db *gorm.DB
}

func (t transaction) Transaction(ctx context.Context, txBody func(ctx context.Context) error) error {
	err := t.db.Transaction(func(tx *gorm.DB) error {
		ctxWithTx := context.WithValue(ctx, constants.TransactionKey{}, tx)
		return txBody(ctxWithTx)
	})
	if err != nil {
		return err
	}
	return nil
}

func NewTransactionRepository(db *gorm.DB) TransactionRepository {
	return &transaction{db: db}
}

func (r *transaction) Create(ub *model.Transaction) error {
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
