package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/Behyna/sms-services/paymentgateway/internal/metrics"
)

// HTTPMetricsMiddleware collects HTTP request metrics
func HTTPMetricsMiddleware(m *metrics.Metrics, logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Increment in-flight requests
		m.HTTPRequestsInFlight.Inc()
		defer m.HTTPRequestsInFlight.Dec()

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Extract request/response details
		method := c.Method()
		path := c.Route().Path
		if path == "" {
			path = c.Path()
		}
		statusCode := strconv.Itoa(c.Response().StatusCode())
		responseSize := len(c.Response().Body())

		// Record metrics
		m.RecordHTTPRequest(method, path, statusCode, duration, responseSize)

		// Log slow requests
		if duration > time.Second {
			logger.Warn("Slow HTTP request",
				zap.String("method", method),
				zap.String("path", path),
				zap.String("status_code", statusCode),
				zap.Duration("duration", duration),
				zap.Int("response_size", responseSize),
			)
		}

		return err
	}
}

// HealthCheckMiddleware provides a simple health check endpoint
func HealthCheckMiddleware(serviceName string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Path() == "/health" {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"status":    "healthy",
				"timestamp": time.Now().Unix(),
				"service":   serviceName,
			})
		}
		return c.Next()
	}
}
