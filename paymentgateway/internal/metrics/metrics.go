package metrics

import (
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	// HTTP Metrics
	HTTPRequestsTotal     *prometheus.CounterVec
	HTTPRequestDuration   *prometheus.HistogramVec
	HTTPRequestsInFlight  prometheus.Gauge
	HTTPResponseSizeBytes *prometheus.HistogramVec

	// Business Metrics
	UserBalanceCreated      prometheus.Counter
	UserBalanceCreationErrs prometheus.Counter
	BalanceRetrievalTotal   *prometheus.CounterVec
	TransactionsCreated     *prometheus.CounterVec
	TransactionErrors       *prometheus.CounterVec
	CurrentUserBalances     *prometheus.GaugeVec

	// Database Metrics
	DBConnectionsInUse prometheus.Gauge
	DBConnectionsIdle  prometheus.Gauge
	DBQueryDuration    *prometheus.HistogramVec
	DBQueriesTotal     *prometheus.CounterVec
	DBConnectionErrors prometheus.Counter

	// System Metrics
	ServiceUptime    prometheus.Gauge
	ServiceVersion   *prometheus.GaugeVec
	Goroutines       prometheus.Gauge
	MemoryUsageBytes *prometheus.GaugeVec

	// Validation Metrics
	ValidationErrors   *prometheus.CounterVec
	ValidationDuration *prometheus.HistogramVec
}

func NewMetrics() *Metrics {
	return &Metrics{
		// HTTP Metrics
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "paymentgateway_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status_code"},
		),
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "paymentgateway_http_request_duration_seconds",
				Help:    "Duration of HTTP requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status_code"},
		),
		HTTPRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "paymentgateway_http_requests_in_flight",
				Help: "Number of HTTP requests currently being served",
			},
		),
		HTTPResponseSizeBytes: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "paymentgateway_http_response_size_bytes",
				Help:    "Size of HTTP responses in bytes",
				Buckets: []float64{100, 1000, 10_000, 100_000, 1_000_000},
			},
			[]string{"method", "path", "status_code"},
		),

		// Business Metrics
		UserBalanceCreated: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "paymentgateway_user_balance_created_total",
				Help: "Total number of user balances created",
			},
		),
		UserBalanceCreationErrs: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "paymentgateway_user_balance_creation_errors_total",
				Help: "Total number of user balance creation errors",
			},
		),
		BalanceRetrievalTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "paymentgateway_balance_retrieval_total",
				Help: "Total number of balance retrievals",
			},
			[]string{"status"},
		),
		TransactionsCreated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "paymentgateway_transactions_created_total",
				Help: "Total number of transactions created",
			},
			[]string{"tx_type"},
		),
		TransactionErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "paymentgateway_transaction_errors_total",
				Help: "Total number of transaction errors",
			},
			[]string{"tx_type", "error_type"},
		),
		CurrentUserBalances: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "paymentgateway_current_user_balances",
				Help: "Current balance amounts for users",
			},
			[]string{"user_id"},
		),

		// Database Metrics
		DBConnectionsInUse: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "paymentgateway_db_connections_in_use",
				Help: "Number of database connections currently in use",
			},
		),
		DBConnectionsIdle: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "paymentgateway_db_connections_idle",
				Help: "Number of idle database connections",
			},
		),
		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "paymentgateway_db_query_duration_seconds",
				Help:    "Duration of database queries in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
			},
			[]string{"operation", "table"},
		),
		DBQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "paymentgateway_db_queries_total",
				Help: "Total number of database queries",
			},
			[]string{"operation", "table", "status"},
		),
		DBConnectionErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "paymentgateway_db_connection_errors_total",
				Help: "Total number of database connection errors",
			},
		),

		// System Metrics
		ServiceUptime: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "paymentgateway_service_uptime_seconds",
				Help: "Service uptime in seconds",
			},
		),
		ServiceVersion: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "paymentgateway_service_version_info",
				Help: "Service version information (labels: version, commit, build_date)",
			},
			[]string{"version", "commit", "build_date"},
		),
		Goroutines: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "paymentgateway_goroutines",
				Help: "Number of goroutines currently running",
			},
		),
		MemoryUsageBytes: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "paymentgateway_memory_usage_bytes",
				Help: "Memory usage in bytes",
			},
			[]string{"type"},
		),

		// Validation Metrics
		ValidationErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "paymentgateway_validation_errors_total",
				Help: "Total number of validation errors",
			},
			[]string{"field", "tag"},
		),
		ValidationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "paymentgateway_validation_duration_seconds",
				Help:    "Duration of validation operations in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5},
			},
			[]string{"endpoint"},
		),
	}
}

// --- Recording Methods ---

func (m *Metrics) RecordHTTPRequest(method, path, statusCode string, duration time.Duration, responseSize int) {
	m.HTTPRequestsTotal.WithLabelValues(method, path, statusCode).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path, statusCode).Observe(duration.Seconds())
	m.HTTPResponseSizeBytes.WithLabelValues(method, path, statusCode).Observe(float64(responseSize))
}

func (m *Metrics) RecordUserBalanceCreated() {
	m.UserBalanceCreated.Inc()
}

func (m *Metrics) RecordUserBalanceCreationError() {
	m.UserBalanceCreationErrs.Inc()
}

func (m *Metrics) RecordBalanceRetrieval(status string) {
	m.BalanceRetrievalTotal.WithLabelValues(status).Inc()
}

func (m *Metrics) RecordTransactionCreated(txType string) {
	m.TransactionsCreated.WithLabelValues(txType).Inc()
}

func (m *Metrics) RecordTransactionError(txType, errorType string) {
	m.TransactionErrors.WithLabelValues(txType, errorType).Inc()
}

func (m *Metrics) UpdateUserBalance(userID string, balance int64) {
	m.CurrentUserBalances.WithLabelValues(userID).Set(float64(balance))
}

func (m *Metrics) RecordDBQuery(operation, table, status string, duration time.Duration) {
	m.DBQueriesTotal.WithLabelValues(operation, table, status).Inc()
	m.DBQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

func (m *Metrics) RecordDBConnectionError() {
	m.DBConnectionErrors.Inc()
}

func (m *Metrics) RecordValidationError(field, tag string) {
	m.ValidationErrors.WithLabelValues(field, tag).Inc()
}

func (m *Metrics) RecordValidationDuration(endpoint string, duration time.Duration) {
	m.ValidationDuration.WithLabelValues(endpoint).Observe(duration.Seconds())
}

// UpdateSystemMetrics updates system-level metrics (goroutines, uptime, memory).
func (m *Metrics) UpdateSystemMetrics(uptime time.Duration, memStats *runtime.MemStats) {
	m.ServiceUptime.Set(uptime.Seconds())
	m.Goroutines.Set(float64(runtime.NumGoroutine()))

	m.MemoryUsageBytes.WithLabelValues("alloc").Set(float64(memStats.Alloc))
	m.MemoryUsageBytes.WithLabelValues("total_alloc").Set(float64(memStats.TotalAlloc))
	m.MemoryUsageBytes.WithLabelValues("sys").Set(float64(memStats.Sys))
	m.MemoryUsageBytes.WithLabelValues("heap_alloc").Set(float64(memStats.HeapAlloc))
	m.MemoryUsageBytes.WithLabelValues("heap_sys").Set(float64(memStats.HeapSys))
}

// SetServiceVersion sets the service version information (only once per start).
func (m *Metrics) SetServiceVersion(version, commit, buildDate string) {
	m.ServiceVersion.WithLabelValues(version, commit, buildDate).Set(1)
}
