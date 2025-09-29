package mocks

import (
	"context"

	"github.com/Behyna/sms-services/smsgateway/internal/service"
	"github.com/stretchr/testify/mock"
)

type PaymentService struct {
	mock.Mock
}

func (p *PaymentService) Charge(ctx context.Context, cmd service.ChargePaymentCommand) error {
	args := p.Called(ctx, cmd)
	return args.Error(0)
}

func (p *PaymentService) Refund(ctx context.Context, cmd service.RefundPaymentCommand) error {
	args := p.Called(ctx, cmd)
	return args.Error(0)
}
