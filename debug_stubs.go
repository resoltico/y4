//go:build !debug

package main

import (
	"log/slog"
	"time"
)

type DebugSystem struct {
	logger *slog.Logger
}

type DebugConfig struct {
	LogLevel      slog.Level
	EnableTracing bool
	EnableMonitor bool
	OutputFile    string
	ConsoleOutput bool
}

func InitDebugSystem(config DebugConfig) *DebugSystem {
	return &DebugSystem{
		logger: slog.Default(),
	}
}

func GetDebugSystem() *DebugSystem {
	return &DebugSystem{
		logger: slog.Default(),
	}
}

func (ds *DebugSystem) TraceProcessingStart(method string, params *OtsuParameters, imageSize [2]int) int64 {
	return 0
}

func (ds *DebugSystem) TraceProcessingEnd(operationID int64, duration time.Duration, success bool, errorMsg string) {
}

func (ds *DebugSystem) TraceParameterChange(field string, oldValue, newValue interface{}) {
}

func (ds *DebugSystem) TraceImageOperation(operationID int64, operation string, inputSize, outputSize [2]int, duration time.Duration) {
}

func (ds *DebugSystem) TraceThresholdCalculation(operationID int64, threshold [2]int, variance float64) {
}

func (ds *DebugSystem) TraceValidationError(err error, context string) {
}

func (ds *DebugSystem) DumpSystemState() {
}

func (ds *DebugSystem) Close() error {
	return nil
}

func DebugTraceStart(method string, params *OtsuParameters, imageSize [2]int) int64 {
	return 0
}

func DebugTraceEnd(operationID int64, duration time.Duration, err error) {
}

func DebugTraceParam(field string, oldValue, newValue interface{}) {
}

func DebugTraceMat(operationID int64, operation string, mat interface{}) {
}

func DebugTraceMemory(context string) {
}
