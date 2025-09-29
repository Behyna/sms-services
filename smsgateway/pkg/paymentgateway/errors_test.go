package paymentgateway_test

import (
	"testing"

	"github.com/Behyna/sms-services/smsgateway/pkg/paymentgateway"
	"github.com/stretchr/testify/assert"
)

func TestMapStatusToError(t *testing.T) {
	testCases := []struct {
		name          string
		statusCode    int
		expectedError error
	}{
		{
			name:          "NotFound",
			statusCode:    404,
			expectedError: paymentgateway.ErrUserNotFound,
		},
		{
			name:          "UnprocessableEntity",
			statusCode:    422,
			expectedError: paymentgateway.ErrValidationFailed,
		},
		{
			name:          "InternalServerError",
			statusCode:    500,
			expectedError: paymentgateway.ErrServerError,
		},
		{
			name:          "BadGateway",
			statusCode:    502,
			expectedError: paymentgateway.ErrServerError,
		},
		{
			name:          "BadRequest",
			statusCode:    400,
			expectedError: paymentgateway.ErrServerError,
		},
		{
			name:          "Conflict",
			statusCode:    409,
			expectedError: paymentgateway.ErrInsufficientBalance,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := paymentgateway.MapStatusToError(tc.statusCode)

			assert.Error(t, err, "Expected an error for status code %d", tc.statusCode)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}
