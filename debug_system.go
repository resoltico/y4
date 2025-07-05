//go:build debug

package main

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"

	"gocv.io/x/gocv"
)

type DebugSystem struct {
	logger       *slog.Logger
	tracer       *ParameterTracer
	monitor      *ResourceMonitor
	enabled      bool
	startTime    time.Time
	operationID  int64
	operationMux sync.Mutex
}

type DebugConfig struct {
	LogLevel      slog.Level
	EnableTracing bool
	EnableMonitor bool
	OutputFile    string
	ConsoleOutput bool
}

var debugSystem *DebugSystem
var debugOnce sync.Once

func InitDebugSystem(config DebugConfig) *DebugSystem {
	debugOnce.Do(func() {
		debugSystem = newDebugSystem(config)
	})
	return debugSystem
}

func GetDebugSystem() *DebugSystem {
	if debugSystem == nil {
		return InitDebugSystem(DebugConfig{
			LogLevel:      slog.LevelInfo,
			EnableTracing: true,
			EnableMonitor: true,
			ConsoleOutput: true,
		})
	}
	return debugSystem
}

func newDebugSystem(config DebugConfig) *DebugSystem {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level:     config.LogLevel,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   "timestamp",
					Value: slog.StringValue(time.Now().Format("15:04:05.000")),
				}
			}
			return a
		},
	}

	if config.ConsoleOutput {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else if config.OutputFile != "" {
		file, err := os.OpenFile(config.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			handler = slog.NewTextHandler(os.Stdout, opts)
		} else {
			handler = slog.NewJSONHandler(file, opts)
		}
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)

	ds := &DebugSystem{
		logger:    logger,
		enabled:   true,
		startTime: time.Now(),
	}

	if config.EnableTracing {
		ds.tracer = NewParameterTracer(logger)
	}

	if config.EnableMonitor {
		ds.monitor = NewResourceMonitor(logger)
		ds.monitor.Start()
	}

	ds.logger.Info("debug system initialized",
		"log_level", config.LogLevel.String(),
		"tracing_enabled", config.EnableTracing,
		"monitoring_enabled", config.EnableMonitor,
	)

	return ds
}

func (ds *DebugSystem) nextOperationID() int64 {
	ds.operationMux.Lock()
	defer ds.operationMux.Unlock()
	ds.operationID++
	return ds.operationID
}

func (ds *DebugSystem) TraceProcessingStart(method string, params *OtsuParameters, imageSize [2]int) int64 {
	if !ds.enabled {
		return 0
	}

	opID := ds.nextOperationID()

	ds.logger.Info("processing operation started",
		"operation_id", opID,
		"method", method,
		"image_width", imageSize[0],
		"image_height", imageSize[1],
		"window_size", params.WindowSize,
		"histogram_bins", params.HistogramBins,
		"smoothing_strength", params.SmoothingStrength,
	)

	if ds.tracer != nil {
		ds.tracer.TraceParameters(opID, method, params)
	}

	if ds.monitor != nil {
		ds.monitor.RecordOperationStart(opID, method)
	}

	return opID
}

func (ds *DebugSystem) TraceProcessingEnd(operationID int64, duration time.Duration, success bool, errorMsg string) {
	if !ds.enabled || operationID == 0 {
		return
	}

	logArgs := []interface{}{
		"operation_id", operationID,
		"duration_ms", duration.Milliseconds(),
		"success", success,
	}

	if errorMsg != "" {
		logArgs = append(logArgs, "error", errorMsg)
		ds.logger.Error("processing operation failed", logArgs...)
	} else {
		ds.logger.Info("processing operation completed", logArgs...)
	}

	if ds.monitor != nil {
		ds.monitor.RecordOperationEnd(operationID, duration, success)
	}
}

func (ds *DebugSystem) TraceParameterChange(field string, oldValue, newValue interface{}) {
	if !ds.enabled {
		return
	}

	ds.logger.Debug("parameter changed",
		"field", field,
		"old_value", oldValue,
		"new_value", newValue,
		"timestamp", time.Now().UnixMilli(),
	)

	if ds.tracer != nil {
		ds.tracer.TraceParameterChange(field, oldValue, newValue)
	}
}

func (ds *DebugSystem) TraceImageOperation(operationID int64, operation string, inputSize, outputSize [2]int, duration time.Duration) {
	if !ds.enabled {
		return
	}

	ds.logger.Debug("image operation completed",
		"operation_id", operationID,
		"operation", operation,
		"input_size", fmt.Sprintf("%dx%d", inputSize[0], inputSize[1]),
		"output_size", fmt.Sprintf("%dx%d", outputSize[0], outputSize[1]),
		"duration_ms", duration.Milliseconds(),
	)
}

func (ds *DebugSystem) TraceThresholdCalculation(operationID int64, threshold [2]int, variance float64) {
	if !ds.enabled {
		return
	}

	ds.logger.Debug("threshold calculation",
		"operation_id", operationID,
		"threshold_t1", threshold[0],
		"threshold_t2", threshold[1],
		"between_class_variance", variance,
	)
}

func (ds *DebugSystem) TraceValidationError(err error, context string) {
	if !ds.enabled {
		return
	}

	ds.logger.Error("validation error",
		"error", err.Error(),
		"context", context,
		"error_type", fmt.Sprintf("%T", err),
	)
}

func (ds *DebugSystem) DumpSystemState() {
	if !ds.enabled {
		return
	}

	uptime := time.Since(ds.startTime)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	ds.logger.Info("system state dump",
		"uptime", uptime.String(),
		"total_operations", ds.operationID,
		"heap_alloc_mb", bytesToMB(m.HeapAlloc),
		"total_alloc_mb", bytesToMB(m.TotalAlloc),
		"gc_cycles", m.NumGC,
		"goroutines", runtime.NumGoroutine(),
	)

	if ds.monitor != nil {
		ds.monitor.DumpStats()
	}
}

func (ds *DebugSystem) Close() error {
	if ds.monitor != nil {
		ds.monitor.Stop()
	}

	ds.logger.Info("debug system shutdown",
		"total_uptime", time.Since(ds.startTime).String(),
		"total_operations", ds.operationID,
	)

	return nil
}

type MatInfo struct {
	Rows     int
	Cols     int
	Type     gocv.MatType
	Channels int
	Empty    bool
}

func GetMatInfo(mat gocv.Mat) MatInfo {
	return MatInfo{
		Rows:     mat.Rows(),
		Cols:     mat.Cols(),
		Type:     mat.Type(),
		Channels: mat.Channels(),
		Empty:    mat.Empty(),
	}
}

func bytesToMB(bytes uint64) float64 {
	return float64(bytes) / 1024 / 1024
}

func DebugTraceStart(method string, params *OtsuParameters, imageSize [2]int) int64 {
	ds := GetDebugSystem()
	return ds.TraceProcessingStart(method, params, imageSize)
}

func DebugTraceEnd(operationID int64, duration time.Duration, err error) {
	ds := GetDebugSystem()
	success := err == nil
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}
	ds.TraceProcessingEnd(operationID, duration, success, errorMsg)
}

func DebugTraceParam(field string, oldValue, newValue interface{}) {
	ds := GetDebugSystem()
	ds.TraceParameterChange(field, oldValue, newValue)
}

func DebugTraceMat(operationID int64, operation string, mat gocv.Mat) {
	ds := GetDebugSystem()
	ds.logger.Debug("mat operation",
		"operation_id", operationID,
		"operation", operation,
		"mat_rows", mat.Rows(),
		"mat_cols", mat.Cols(),
		"mat_type", mat.Type(),
		"mat_channels", mat.Channels(),
		"mat_empty", mat.Empty(),
	)
}

func DebugTraceMemory(context string) {
	ds := GetDebugSystem()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	ds.logger.Debug("memory usage",
		"context", context,
		"heap_alloc_mb", bytesToMB(m.HeapAlloc),
		"heap_sys_mb", bytesToMB(m.HeapSys),
		"heap_objects", m.HeapObjects,
		"gc_cycles", m.NumGC,
		"goroutines", runtime.NumGoroutine(),
	)
}
