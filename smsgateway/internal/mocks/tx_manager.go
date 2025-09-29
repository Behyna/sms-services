package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type TxManager struct {
	mock.Mock
}

func (t *TxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	args := t.Called(ctx, fn)

	if args.Error(0) != nil {
		return args.Error(0)
	}

	txCtx := context.WithValue(ctx, "tx", "mock_tx")
	return fn(txCtx)
}

func (t *TxManager) GetTx(ctx context.Context) context.Context {
	args := t.Called(ctx)
	return args.Get(0).(context.Context)
}
