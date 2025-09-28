package repository

import (
	"context"
	"errors"

	"github.com/Behyna/sms-services/paymentgateway/internal/model"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var ErrTransactionExisted = errors.New("TRANSACTION_EXISTED")

type TransactionRepository interface {
	Create(ctx context.Context, tx *model.Transaction) error
	GetByIdempotencyKey(txType model.TxType, idempotencyKey string) (*model.Transaction, error)
}

type transaction struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) TransactionRepository {
	return &transaction{db: db}
}

func (t *transaction) Create(ctx context.Context, tx *model.Transaction) error {
	db := GetTx(ctx, t.db)
	err := db.Create(tx).Error
	if err == nil {
		return nil
	}

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return ErrTransactionExisted
	}

	return err
}

func (t *transaction) GetByIdempotencyKey(txType model.TxType, idempotencyKey string) (*model.Transaction, error) {
	var tx model.Transaction
	err := t.db.Where("tx_type = ? AND idempotency_key = ?", txType, idempotencyKey).First(&tx).Error
	if err == nil {
		return &tx, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return nil, err
}
