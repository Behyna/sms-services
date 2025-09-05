package metrics

import (
	"runtime"
	"time"

	"go.uber.org/zap"
)

// SystemCollector collects system-level metrics
type SystemCollector struct {
	metrics   *Metrics
	logger    *zap.Logger
	startTime time.Time
	ticker    *time.Ticker
	stopCh    chan struct{}
}

// NewSystemCollector creates a new system metrics collector
func NewSystemCollector(metrics *Metrics, logger *zap.Logger) *SystemCollector {
	return &SystemCollector{
		metrics:   metrics,
		logger:    logger,
		startTime: time.Now(),
		stopCh:    make(chan struct{}),
	}
}

// Start begins collecting system metrics at regular intervals
func (sc *SystemCollector) Start(interval time.Duration) {
	sc.ticker = time.NewTicker(interval)

	// Set initial service version (can be updated with actual build info)
	sc.metrics.SetServiceVersion("1.0.0", "unknown", time.Now().Format("2006-01-02"))

	go sc.collectLoop()
	sc.logger.Info("System metrics collector started", zap.Duration("interval", interval))
}

// Stop stops the system metrics collector
func (sc *SystemCollector) Stop() {
	if sc.ticker != nil {
		sc.ticker.Stop()
	}
	close(sc.stopCh)
	sc.logger.Info("System metrics collector stopped")
}

// collectLoop runs the collection loop
func (sc *SystemCollector) collectLoop() {
	// Collect initial metrics
	sc.collect()

	for {
		select {
		case <-sc.ticker.C:
			sc.collect()
		case <-sc.stopCh:
			return
		}
	}
}

// collect gathers and updates system metrics
func (sc *SystemCollector) collect() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Calculate uptime
	uptime := time.Since(sc.startTime)

	// Update metrics using the correct signature
	sc.metrics.UpdateSystemMetrics(uptime, &memStats)

	// Log system stats periodically (every 10 minutes)
	if uptime.Minutes() > 0 && int(uptime.Minutes())%10 == 0 {
		sc.logger.Info("System metrics snapshot",
			zap.Duration("uptime", uptime),
			zap.Int("goroutines", runtime.NumGoroutine()),
			zap.Uint64("alloc_mb", memStats.Alloc/1024/1024),
			zap.Uint64("sys_mb", memStats.Sys/1024/1024),
			zap.Uint32("gc_count", memStats.NumGC),
		)
	}
}

// GetUptimeSeconds returns the service uptime in seconds
func (sc *SystemCollector) GetUptimeSeconds() float64 {
	return time.Since(sc.startTime).Seconds()
}

// GetMemoryStats returns current memory statistics
func (sc *SystemCollector) GetMemoryStats() map[string]float64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]float64{
		"alloc":         float64(memStats.Alloc),
		"total_alloc":   float64(memStats.TotalAlloc),
		"sys":           float64(memStats.Sys),
		"heap_alloc":    float64(memStats.HeapAlloc),
		"heap_sys":      float64(memStats.HeapSys),
		"heap_idle":     float64(memStats.HeapIdle),
		"heap_inuse":    float64(memStats.HeapInuse),
		"heap_released": float64(memStats.HeapReleased),
		"stack_inuse":   float64(memStats.StackInuse),
		"stack_sys":     float64(memStats.StackSys),
		"gc_sys":        float64(memStats.GCSys),
		"other_sys":     float64(memStats.OtherSys),
	}
}
