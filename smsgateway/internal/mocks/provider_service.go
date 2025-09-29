package mocks

import (
	"context"

	"github.com/Behyna/sms-services/smsgateway/pkg/smsprovider"
	"github.com/stretchr/testify/mock"
)

type ProviderService struct {
	mock.Mock
}

func (p *ProviderService) SendWithRetry(ctx context.Context, fromMSISDN, toMSISDN, text string) (smsprovider.Response, error) {
	args := p.Called(ctx, fromMSISDN, toMSISDN, text)
	return args.Get(0).(smsprovider.Response), args.Error(1)
}
