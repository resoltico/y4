package main

import (
	"fmt"

	"gocv.io/x/gocv"
)

func CalculateBinaryMetrics(groundTruth, result gocv.Mat) (*BinaryImageMetrics, error) {
	defer func() {
		if r := recover(); r != nil {
			debugSystem := GetDebugSystem()
			debugSystem.logger.Error("metrics calculation panicked", "error", r)
		}
	}()

	if err := validateMatForMetrics(groundTruth, "ground truth input"); err != nil {
		return nil, fmt.Errorf("ground truth validation failed: %w", err)
	}

	if err := validateMatForMetrics(result, "result input"); err != nil {
		return nil, fmt.Errorf("result validation failed: %w", err)
	}

	if err := validateMatDimensionsMatch(groundTruth, result, "input matrices"); err != nil {
		return nil, fmt.Errorf("dimension mismatch: %w", err)
	}

	metrics := &BinaryImageMetrics{}

	if err := metrics.calculateConfusionMatrix(groundTruth, result); err != nil {
		return nil, fmt.Errorf("confusion matrix calculation failed: %w", err)
	}

	if err := metrics.calculateDRD(groundTruth, result); err != nil {
		return nil, fmt.Errorf("DRD calculation failed: %w", err)
	}

	if err := metrics.calculateMPM(groundTruth, result); err != nil {
		return nil, fmt.Errorf("MPM calculation failed: %w", err)
	}

	if err := metrics.calculateBackgroundForegroundContrast(groundTruth, result); err != nil {
		return nil, fmt.Errorf("BFC calculation failed: %w", err)
	}

	if err := metrics.calculateSkeletonSimilarity(groundTruth, result); err != nil {
		return nil, fmt.Errorf("skeleton similarity calculation failed: %w", err)
	}

	if err := validateAllMetrics(metrics); err != nil {
		return nil, fmt.Errorf("metrics validation failed: %w", err)
	}

	debugSystem := GetDebugSystem()
	debugSystem.logger.Debug("metrics calculation completed",
		"f_measure", metrics.FMeasure(),
		"pseudo_f_measure", metrics.PseudoFMeasure(),
		"nrm", metrics.NRM(),
		"drd", metrics.DRD(),
		"mpm", metrics.MPM(),
		"bfc", metrics.BackgroundForegroundContrast(),
		"skeleton", metrics.SkeletonSimilarity(),
		"total_pixels", metrics.TotalPixels,
		"true_positives", metrics.TruePositives,
		"true_negatives", metrics.TrueNegatives,
		"false_positives", metrics.FalsePositives,
		"false_negatives", metrics.FalseNegatives,
	)

	return metrics, nil
}
