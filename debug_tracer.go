//go:build debug

package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type ParameterTracer struct {
	logger           *slog.Logger
	parameterHistory map[string][]ParameterChange
	operationParams  map[int64]ParameterSnapshot
	mutex            sync.RWMutex
}

type ParameterChange struct {
	Timestamp time.Time   `json:"timestamp"`
	Field     string      `json:"field"`
	OldValue  interface{} `json:"old_value"`
	NewValue  interface{} `json:"new_value"`
}

type ParameterSnapshot struct {
	OperationID int64           `json:"operation_id"`
	Method      string          `json:"method"`
	Timestamp   time.Time       `json:"timestamp"`
	Parameters  *OtsuParameters `json:"parameters"`
}

func NewParameterTracer(logger *slog.Logger) *ParameterTracer {
	return &ParameterTracer{
		logger:           logger,
		parameterHistory: make(map[string][]ParameterChange),
		operationParams:  make(map[int64]ParameterSnapshot),
	}
}

func (pt *ParameterTracer) TraceParameters(operationID int64, method string, params *OtsuParameters) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	snapshot := ParameterSnapshot{
		OperationID: operationID,
		Method:      method,
		Timestamp:   time.Now(),
		Parameters:  pt.cloneParameters(params),
	}

	pt.operationParams[operationID] = snapshot

	paramJSON, _ := json.Marshal(params)
	pt.logger.Debug("parameter snapshot recorded",
		"operation_id", operationID,
		"method", method,
		"parameters", string(paramJSON),
	)
}

func (pt *ParameterTracer) TraceParameterChange(field string, oldValue, newValue interface{}) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	change := ParameterChange{
		Timestamp: time.Now(),
		Field:     field,
		OldValue:  oldValue,
		NewValue:  newValue,
	}

	if pt.parameterHistory[field] == nil {
		pt.parameterHistory[field] = make([]ParameterChange, 0)
	}

	pt.parameterHistory[field] = append(pt.parameterHistory[field], change)

	pt.logger.Debug("parameter change traced",
		"field", field,
		"old_value", oldValue,
		"new_value", newValue,
		"change_count", len(pt.parameterHistory[field]),
	)
}

func (pt *ParameterTracer) GetParameterHistory(field string) []ParameterChange {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	history := pt.parameterHistory[field]
	if history == nil {
		return []ParameterChange{}
	}

	result := make([]ParameterChange, len(history))
	copy(result, history)
	return result
}

func (pt *ParameterTracer) GetOperationParameters(operationID int64) (ParameterSnapshot, bool) {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	snapshot, exists := pt.operationParams[operationID]
	return snapshot, exists
}

func (pt *ParameterTracer) AnalyzeParameterPatterns() map[string]interface{} {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	analysis := make(map[string]interface{})

	changeFreq := make(map[string]int)
	for field, changes := range pt.parameterHistory {
		changeFreq[field] = len(changes)
	}
	analysis["change_frequency"] = changeFreq

	recentChanges := make(map[string]time.Time)
	for field, changes := range pt.parameterHistory {
		if len(changes) > 0 {
			recentChanges[field] = changes[len(changes)-1].Timestamp
		}
	}
	analysis["recent_changes"] = recentChanges

	methodCombinations := make(map[string]int)
	for _, snapshot := range pt.operationParams {
		key := pt.generateParameterKey(snapshot.Parameters)
		methodCombinations[key]++
	}
	analysis["parameter_combinations"] = methodCombinations

	return analysis
}

func (pt *ParameterTracer) generateParameterKey(params *OtsuParameters) string {
	return fmt.Sprintf("method_%t_%t_window_%d_bins_%d_smooth_%.1f",
		params.MultiScaleProcessing,
		params.RegionAdaptiveThresholding,
		params.WindowSize,
		params.HistogramBins,
		params.SmoothingStrength,
	)
}

func (pt *ParameterTracer) DumpParameterHistory() {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	pt.logger.Info("parameter history dump",
		"total_fields_tracked", len(pt.parameterHistory),
		"total_operations", len(pt.operationParams),
	)

	for field, changes := range pt.parameterHistory {
		if len(changes) > 0 {
			pt.logger.Info("field history",
				"field", field,
				"total_changes", len(changes),
				"first_change", changes[0].Timestamp,
				"last_change", changes[len(changes)-1].Timestamp,
				"last_value", changes[len(changes)-1].NewValue,
			)
		}
	}
}

func (pt *ParameterTracer) cloneParameters(params *OtsuParameters) *OtsuParameters {
	return &OtsuParameters{
		WindowSize:                 params.WindowSize,
		HistogramBins:              params.HistogramBins,
		SmoothingStrength:          params.SmoothingStrength,
		EdgePreservation:           params.EdgePreservation,
		NoiseRobustness:            params.NoiseRobustness,
		GaussianPreprocessing:      params.GaussianPreprocessing,
		UseLogHistogram:            params.UseLogHistogram,
		NormalizeHistogram:         params.NormalizeHistogram,
		ApplyContrastEnhancement:   params.ApplyContrastEnhancement,
		AdaptiveWindowSizing:       params.AdaptiveWindowSizing,
		MultiScaleProcessing:       params.MultiScaleProcessing,
		PyramidLevels:              params.PyramidLevels,
		NeighborhoodType:           params.NeighborhoodType,
		InterpolationMethod:        params.InterpolationMethod,
		MorphologicalPostProcess:   params.MorphologicalPostProcess,
		MorphologicalKernelSize:    params.MorphologicalKernelSize,
		HomomorphicFiltering:       params.HomomorphicFiltering,
		AnisotropicDiffusion:       params.AnisotropicDiffusion,
		DiffusionIterations:        params.DiffusionIterations,
		DiffusionKappa:             params.DiffusionKappa,
		RegionAdaptiveThresholding: params.RegionAdaptiveThresholding,
		RegionGridSize:             params.RegionGridSize,
	}
}

func (pt *ParameterTracer) CleanupOldHistory(maxAge time.Duration) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	cutoff := time.Now().Add(-maxAge)
	cleaned := 0

	for field, changes := range pt.parameterHistory {
		newChanges := make([]ParameterChange, 0)
		for _, change := range changes {
			if change.Timestamp.After(cutoff) {
				newChanges = append(newChanges, change)
			} else {
				cleaned++
			}
		}
		pt.parameterHistory[field] = newChanges
	}

	for operationID, snapshot := range pt.operationParams {
		if snapshot.Timestamp.Before(cutoff) {
			delete(pt.operationParams, operationID)
			cleaned++
		}
	}

	pt.logger.Info("parameter history cleanup completed",
		"items_cleaned", cleaned,
		"cutoff_time", cutoff,
	)
}
