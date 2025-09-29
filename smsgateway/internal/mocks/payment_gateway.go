package mocks

import (
	"context"

	"github.com/Behyna/sms-services/smsgateway/pkg/paymentgateway"
	"github.com/stretchr/testify/mock"
)

type PaymentGateway struct {
	mock.Mock
}

func (p *PaymentGateway) Charge(ctx context.Context, request paymentgateway.UpdateUserBalanceRequest) (paymentgateway.Response, error) {
	args := p.Called(ctx, request)
	return args.Get(0).(paymentgateway.Response), args.Error(1)
}

func (p *PaymentGateway) Refund(ctx context.Context, request paymentgateway.UpdateUserBalanceRequest) (paymentgateway.Response, error) {
	args := p.Called(ctx, request)
	return args.Get(0).(paymentgateway.Response), args.Error(1)
}
