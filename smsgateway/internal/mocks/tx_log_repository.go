package mocks

import (
	"context"

	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"github.com/stretchr/testify/mock"
)

type TxLogRepository struct {
	mock.Mock
}

func (t *TxLogRepository) Create(ctx context.Context, log *model.TxLog) error {
	args := t.Called(ctx, log)
	return args.Error(0)
}

func (t *TxLogRepository) Update(log *model.TxLog) error {
	args := t.Called(log)
	return args.Error(0)
}

func (t *TxLogRepository) UpdateByMessageID(ctx context.Context, log *model.TxLog) error {
	args := t.Called(ctx, log)
	return args.Error(0)
}

func (t *TxLogRepository) UpdateForPermFailed(ctx context.Context, log *model.TxLog) error {
	args := t.Called(ctx, log)
	return args.Error(0)
}

func (t *TxLogRepository) FindUnpublishedFailed(limit int) ([]model.TxLog, error) {
	args := t.Called(limit)
	return args.Get(0).([]model.TxLog), args.Error(1)
}

func (t *TxLogRepository) FindUnpublishedCreated(limit int) ([]model.TxLog, error) {
	args := t.Called(limit)
	return args.Get(0).([]model.TxLog), args.Error(1)
}

func (t *TxLogRepository) GetByID(id int64) (*model.TxLog, error) {
	args := t.Called(id)
	return args.Get(0).(*model.TxLog), args.Error(1)
}
