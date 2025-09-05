package metrics

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// HTTPMetricsMiddleware creates a middleware that collects HTTP metrics
func HTTPMetricsMiddleware(metrics *Metrics, logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Increment in-flight requests
		metrics.HTTPRequestsInFlight.Inc()
		defer metrics.HTTPRequestsInFlight.Dec()

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Get response details
		method := c.Method()
		path := c.Route().Path
		if path == "" {
			path = c.Path()
		}
		statusCode := strconv.Itoa(c.Response().StatusCode())
		responseSize := len(c.Response().Body())

		// Record metrics
		metrics.RecordHTTPRequest(method, path, statusCode, duration, responseSize)

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
func HealthCheckMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Path() == "/health" {
			return c.Status(200).JSON(fiber.Map{
				"status":    "healthy",
				"timestamp": time.Now().Unix(),
				"service":   "paymentgateway",
			})
		}
		return c.Next()
	}
}
