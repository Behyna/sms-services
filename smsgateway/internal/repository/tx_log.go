package repository

import (
	"context"
	"errors"

	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"gorm.io/gorm"
)

var ErrTxLogNotFound = errors.New("TXLOG_NOT_FOUND")

type TxLogRepository interface {
	Create(ctx context.Context, log *model.TxLog) error
	Update(log *model.TxLog) error
	UpdateByMessageID(ctx context.Context, log *model.TxLog) error
	UpdateForPermFailed(ctx context.Context, log *model.TxLog) error
	FindUnpublishedFailed(limit int) ([]model.TxLog, error)
	FindUnpublishedCreated(limit int) ([]model.TxLog, error)
	GetByID(id int64) (*model.TxLog, error)
}

type TxLog struct {
	db *gorm.DB
}

func NewTxLogRepository(db *gorm.DB) TxLogRepository {
	return &TxLog{db: db}
}

func (r *TxLog) Create(ctx context.Context, log *model.TxLog) error {
	db := GetTx(ctx, r.db)
	if err := db.Create(log).Error; err != nil {
		return err
	}

	return nil
}

func (r *TxLog) Update(log *model.TxLog) error {
	return r.db.Model(log).Where("id = ?", log.ID).Updates(log).Error
}

func (r *TxLog) UpdateByMessageID(ctx context.Context, log *model.TxLog) error {
	db := GetTx(ctx, r.db)
	return db.Model(log).Where("message_id = ?", log.MessageID).Updates(log).Error
}

func (r *TxLog) UpdateForPermFailed(ctx context.Context, log *model.TxLog) error {
	db := GetTx(ctx, r.db)
	return db.Model(log).Where("message_id = ?", log.MessageID).
		Select("state", "published", "published_at", "last_error", "updated_at").Updates(log).Error
}

func (r *TxLog) FindUnpublishedFailed(limit int) ([]model.TxLog, error) {
	var txLogs []model.TxLog

	err := r.db.Preload("Message").Where("state = ? AND published = ?",
		model.TxLogStateFailed, false).Limit(limit).Find(&txLogs).Error

	if err != nil {
		return nil, err
	}

	return txLogs, nil
}

func (r *TxLog) FindUnpublishedCreated(limit int) ([]model.TxLog, error) {
	var txLogs []model.TxLog

	err := r.db.Preload("Message").
		Where("state = ? AND published = ?",
			model.TxLogStateCreated, false).Order("created_at ASC").Limit(limit).Find(&txLogs).Error

	if err != nil {
		return nil, err
	}

	return txLogs, nil
}

func (r *TxLog) GetByID(id int64) (*model.TxLog, error) {
	var txLog model.TxLog

	err := r.db.Where("id = ?", id).First(&txLog).Error
	if err == nil {
		return &txLog, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTxLogNotFound
	}

	return nil, err
}
