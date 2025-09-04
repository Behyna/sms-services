package repository

import (
	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"gorm.io/gorm"
)

type TxLogRepository interface {
	CreateWithTx(tx *gorm.DB, log *model.TxLog) error
	Update(log *model.TxLog) error
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

func (r *TxLog) Update(log *model.TxLog) error {
	return r.db.Model(log).Where("message_id = ?", log.MessageID).Updates(map[string]interface{}{
		"published":    log.Published,
		"published_at": log.PublishedAt,
	}).Error
}
