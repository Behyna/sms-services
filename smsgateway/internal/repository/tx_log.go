package repository

import (
	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"gorm.io/gorm"
)

type TxLogRepository interface {
	CreateWithTx(tx *gorm.DB, log *model.TxLog) error
}

type TxLog struct {
	db *gorm.DB
}

func NewTxLogRepository(db *gorm.DB) TxLogRepository {
	return &TxLog{db: db}
}

func (r *TxLog) CreateWithTx(tx *gorm.DB, log *model.TxLog) error {
	if err := tx.Create(log).Error; err != nil {
		return err
	}

	return nil
}
