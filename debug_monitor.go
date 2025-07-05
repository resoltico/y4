//go:build debug

package main

import (
	"context"
	"log/slog"
	"runtime"
	"sync"
	"time"
)

type ResourceMonitor struct {
	logger           *slog.Logger
	operations       map[int64]*OperationStats
	matAllocations   int64
	matDeallocations int64
	peakMemoryUsage  uint64
	startTime        time.Time
	ticker           *time.Ticker
	ctx              context.Context
	cancel           context.CancelFunc
	mutex            sync.RWMutex
	running          bool
}

type OperationStats struct {
	ID             int64
	Method         string
	StartTime      time.Time
	EndTime        time.Time
	Duration       time.Duration
	Success        bool
	PeakMemory     uint64
	InitialMemory  uint64
	GoroutineCount int
	MatAllocations int64
}

func NewResourceMonitor(logger *slog.Logger) *ResourceMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &ResourceMonitor{
		logger:     logger,
		operations: make(map[int64]*OperationStats),
		startTime:  time.Now(),
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (rm *ResourceMonitor) Start() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if rm.running {
		return
	}

	rm.running = true
	rm.ticker = time.NewTicker(5 * time.Second)

	go rm.monitorLoop()

	rm.logger.Info("resource monitor started", "interval", "5s")
}

func (rm *ResourceMonitor) Stop() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if !rm.running {
		return
	}

	rm.running = false
	rm.cancel()

	if rm.ticker != nil {
		rm.ticker.Stop()
	}

	rm.logger.Info("resource monitor stopped")
}

func (rm *ResourceMonitor) monitorLoop() {
	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-rm.ticker.C:
			rm.captureSystemSnapshot()
			rm.checkResourceThresholds()
		}
	}
}

func (rm *ResourceMonitor) captureSystemSnapshot() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	if m.HeapAlloc > rm.peakMemoryUsage {
		rm.peakMemoryUsage = m.HeapAlloc
	}

	rm.logger.Debug("system snapshot",
		"heap_alloc_mb", bytesToMB(m.HeapAlloc),
		"heap_sys_mb", bytesToMB(m.HeapSys),
		"heap_objects", m.HeapObjects,
		"goroutines", runtime.NumGoroutine(),
		"gc_cycles", m.NumGC,
		"cgo_calls", runtime.NumCgoCall(),
	)
}

func (rm *ResourceMonitor) checkResourceThresholds() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	const maxHeapMB = 512
	const maxGoroutines = 100

	if bytesToMB(m.HeapAlloc) > maxHeapMB {
		rm.logger.Warn("high memory usage detected",
			"heap_alloc_mb", bytesToMB(m.HeapAlloc),
			"threshold_mb", maxHeapMB,
		)
	}

	goroutineCount := runtime.NumGoroutine()
	if goroutineCount > maxGoroutines {
		rm.logger.Warn("high goroutine count detected",
			"goroutines", goroutineCount,
			"threshold", maxGoroutines,
		)
	}

	if m.NumGC > 0 {
		gcPause := time.Duration(m.PauseNs[(m.NumGC+255)%256])
		if gcPause > 10*time.Millisecond {
			rm.logger.Warn("long gc pause detected",
				"pause_ms", gcPause.Milliseconds(),
				"gc_cycle", m.NumGC,
			)
		}
	}
}

func (rm *ResourceMonitor) RecordOperationStart(operationID int64, method string) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats := &OperationStats{
		ID:             operationID,
		Method:         method,
		StartTime:      time.Now(),
		InitialMemory:  m.HeapAlloc,
		GoroutineCount: runtime.NumGoroutine(),
		MatAllocations: rm.matAllocations,
	}

	rm.operations[operationID] = stats

	rm.logger.Debug("operation monitoring started",
		"operation_id", operationID,
		"method", method,
		"initial_memory_mb", bytesToMB(stats.InitialMemory),
		"goroutines", stats.GoroutineCount,
	)
}

func (rm *ResourceMonitor) RecordOperationEnd(operationID int64, duration time.Duration, success bool) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	stats, exists := rm.operations[operationID]
	if !exists {
		rm.logger.Warn("operation end recorded without start",
			"operation_id", operationID,
		)
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats.EndTime = time.Now()
	stats.Duration = duration
	stats.Success = success
	stats.PeakMemory = m.HeapAlloc

	memoryDelta := int64(stats.PeakMemory) - int64(stats.InitialMemory)
	matDelta := rm.matAllocations - stats.MatAllocations

	rm.logger.Info("operation monitoring completed",
		"operation_id", operationID,
		"method", stats.Method,
		"duration_ms", duration.Milliseconds(),
		"success", success,
		"memory_delta_mb", bytesToMB(uint64(abs(memoryDelta))),
		"mat_allocations", matDelta,
		"peak_memory_mb", bytesToMB(stats.PeakMemory),
	)

	if !success {
		rm.logger.Error("failed operation resource usage",
			"operation_id", operationID,
			"method", stats.Method,
			"memory_leaked_mb", bytesToMB(uint64(max(0, memoryDelta))),
		)
	}
}

func (rm *ResourceMonitor) DumpStats() {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	uptime := time.Since(rm.startTime)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	completedOps := 0
	successfulOps := 0
	totalDuration := time.Duration(0)

	for _, stats := range rm.operations {
		if !stats.EndTime.IsZero() {
			completedOps++
			if stats.Success {
				successfulOps++
			}
			totalDuration += stats.Duration
		}
	}

	avgDuration := time.Duration(0)
	if completedOps > 0 {
		avgDuration = totalDuration / time.Duration(completedOps)
	}

	rm.logger.Info("resource monitor statistics",
		"uptime", uptime.String(),
		"peak_memory_mb", bytesToMB(rm.peakMemoryUsage),
		"current_memory_mb", bytesToMB(m.HeapAlloc),
		"total_operations", len(rm.operations),
		"completed_operations", completedOps,
		"successful_operations", successfulOps,
		"success_rate", float64(successfulOps)/float64(max(1, completedOps)),
		"average_duration_ms", avgDuration.Milliseconds(),
		"mat_allocations", rm.matAllocations,
		"mat_deallocations", rm.matDeallocations,
		"mat_balance", rm.matAllocations-rm.matDeallocations,
		"gc_cycles", m.NumGC,
		"goroutines", runtime.NumGoroutine(),
	)
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
