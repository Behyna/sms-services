package mocks

import (
	"context"
	"time"

	"github.com/Behyna/sms-services/smsgateway/internal/model"
	"github.com/stretchr/testify/mock"
)

type MessageRepository struct {
	mock.Mock
}

func (m *MessageRepository) Create(ctx context.Context, message *model.Message) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MessageRepository) Update(ctx context.Context, message *model.Message) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MessageRepository) UpdateForSending(ctx context.Context, message *model.Message, staleThreshold time.Time) error {
	args := m.Called(ctx, message, staleThreshold)
	return args.Error(0)
}

func (m *MessageRepository) GetByID(id int64) (*model.Message, error) {
	args := m.Called(id)
	return args.Get(0).(*model.Message), args.Error(1)
}

func (m *MessageRepository) GetByUserID(userID string, limit, offset int) ([]model.Message, error) {
	args := m.Called(userID, limit, offset)
	return args.Get(0).([]model.Message), args.Error(1)
}

func (m *MessageRepository) CountByUserID(userID string) (int, error) {
	args := m.Called(userID)
	return args.Int(0), args.Error(1)
}
