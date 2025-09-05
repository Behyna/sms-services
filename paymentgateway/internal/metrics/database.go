package metrics

import (
	"database/sql"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DatabaseMetricsCollector wraps database operations with metrics collection
type DatabaseMetricsCollector struct {
	metrics *Metrics
	logger  *zap.Logger
	db      *gorm.DB
	sqlDB   *sql.DB
	ticker  *time.Ticker
	stopCh  chan struct{}
}

// NewDatabaseMetricsCollector creates a new database metrics collector
func NewDatabaseMetricsCollector(metrics *Metrics, logger *zap.Logger, db *gorm.DB) *DatabaseMetricsCollector {
	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("Failed to get sql.DB from gorm.DB", zap.Error(err))
		metrics.RecordDBConnectionError()
	}

	return &DatabaseMetricsCollector{
		metrics: metrics,
		logger:  logger,
		db:      db,
		sqlDB:   sqlDB,
		stopCh:  make(chan struct{}),
	}
}

// Start begins collecting database metrics at regular intervals
func (dmc *DatabaseMetricsCollector) Start(interval time.Duration) {
	if dmc.sqlDB == nil {
		dmc.logger.Warn("Cannot start database metrics collector: sqlDB is nil")
		return
	}

	dmc.ticker = time.NewTicker(interval)
	go dmc.collectLoop()
	dmc.logger.Info("Database metrics collector started", zap.Duration("interval", interval))
}

// Stop stops the database metrics collector
func (dmc *DatabaseMetricsCollector) Stop() {
	if dmc.ticker != nil {
		dmc.ticker.Stop()
	}
	close(dmc.stopCh)
	dmc.logger.Info("Database metrics collector stopped")
}

// collectLoop runs the collection loop
func (dmc *DatabaseMetricsCollector) collectLoop() {
	// Collect initial metrics
	dmc.collect()

	for {
		select {
		case <-dmc.ticker.C:
			dmc.collect()
		case <-dmc.stopCh:
			return
		}
	}
}

// collect gathers and updates database metrics
func (dmc *DatabaseMetricsCollector) collect() {
	if dmc.sqlDB == nil {
		return
	}

	stats := dmc.sqlDB.Stats()

	// Update connection metrics
	dmc.metrics.DBConnectionsInUse.Set(float64(stats.InUse))
	dmc.metrics.DBConnectionsIdle.Set(float64(stats.Idle))

	// Log connection stats periodically
	dmc.logger.Debug("Database connection stats",
		zap.Int("open_connections", stats.OpenConnections),
		zap.Int("in_use", stats.InUse),
		zap.Int("idle", stats.Idle),
		zap.Int64("wait_count", stats.WaitCount),
		zap.Duration("wait_duration", stats.WaitDuration),
		zap.Int64("max_idle_closed", stats.MaxIdleClosed),
		zap.Int64("max_idle_time_closed", stats.MaxIdleTimeClosed),
		zap.Int64("max_lifetime_closed", stats.MaxLifetimeClosed),
	)
}

// GetConnectionStats returns current database connection statistics
func (dmc *DatabaseMetricsCollector) GetConnectionStats() sql.DBStats {
	if dmc.sqlDB == nil {
		return sql.DBStats{}
	}
	return dmc.sqlDB.Stats()
}

// WithMetrics wraps a database operation with timing metrics
func (dmc *DatabaseMetricsCollector) WithMetrics(operation, table string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	status := "success"
	if err != nil {
		status = "error"
		if err == gorm.ErrRecordNotFound {
			status = "not_found"
		}
	}

	dmc.metrics.RecordDBQuery(operation, table, status, duration)

	// Log slow queries
	if duration > 100*time.Millisecond {
		dmc.logger.Warn("Slow database query",
			zap.String("operation", operation),
			zap.String("table", table),
			zap.String("status", status),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
	}

	return err
}

// HealthCheck performs a database health check
func (dmc *DatabaseMetricsCollector) HealthCheck() error {
	if dmc.sqlDB == nil {
		err := sql.ErrConnDone
		dmc.metrics.RecordDBConnectionError()
		return err
	}

	return dmc.WithMetrics("ping", "health_check", func() error {
		return dmc.sqlDB.Ping()
	})
}

// GetDatabaseInfo returns basic database information
func (dmc *DatabaseMetricsCollector) GetDatabaseInfo() map[string]interface{} {
	info := map[string]interface{}{
		"driver": "mysql",
	}

	if dmc.sqlDB != nil {
		stats := dmc.sqlDB.Stats()
		info["connection_stats"] = map[string]interface{}{
			"open_connections":     stats.OpenConnections,
			"connections_in_use":   stats.InUse,
			"idle_connections":     stats.Idle,
			"wait_count":           stats.WaitCount,
			"wait_duration_ms":     stats.WaitDuration.Milliseconds(),
			"max_idle_closed":      stats.MaxIdleClosed,
			"max_idle_time_closed": stats.MaxIdleTimeClosed,
			"max_lifetime_closed":  stats.MaxLifetimeClosed,
		}
	}

	return info
}
