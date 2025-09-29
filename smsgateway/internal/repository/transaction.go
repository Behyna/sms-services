package repository

import (
	"context"

	"gorm.io/gorm"
)

type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type TransactionManager struct {
	db *gorm.DB
}

func NewTransactionManager(db *gorm.DB) TxManager {
	return &TransactionManager{db: db}
}

func (tm *TransactionManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return tm.db.Transaction(func(tx *gorm.DB) error {
		ctx = context.WithValue(ctx, "tx", tx)
		return fn(ctx)
	})
}

func GetTx(ctx context.Context, db *gorm.DB) *gorm.DB {
	tx, ok := ctx.Value("tx").(*gorm.DB)
	if !ok {
		return db
	}
	return tx
}
