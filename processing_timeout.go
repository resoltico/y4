package main

import (
	"context"
	"fmt"
	"time"
)

type TimeoutConfig struct {
	SingleScale    time.Duration
	MultiScale     time.Duration
	RegionAdaptive time.Duration
	Preprocessing  time.Duration
	Histogram      time.Duration
}

var DefaultTimeouts = TimeoutConfig{
	SingleScale:    30 * time.Second,
	MultiScale:     120 * time.Second,
	RegionAdaptive: 60 * time.Second,
	Preprocessing:  15 * time.Second,
	Histogram:      10 * time.Second,
}

type TimeoutError struct {
	Operation string
	Duration  time.Duration
	Context   string
}

func (te *TimeoutError) Error() string {
	return fmt.Sprintf("%s operation timed out after %v in %s", te.Operation, te.Duration, te.Context)
}

type ProcessingResult struct {
	Data    *ImageData
	Metrics *BinaryImageMetrics
	Error   error
}

func withProcessingTimeout(ctx context.Context, timeout time.Duration, operation string, fn func() (*ImageData, *BinaryImageMetrics, error)) (*ImageData, *BinaryImageMetrics, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan ProcessingResult, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- ProcessingResult{
					Data:    nil,
					Metrics: nil,
					Error:   fmt.Errorf("operation panicked: %v", r),
				}
			}
		}()

		data, metrics, err := fn()
		done <- ProcessingResult{
			Data:    data,
			Metrics: metrics,
			Error:   err,
		}
	}()

	select {
	case result := <-done:
		return result.Data, result.Metrics, result.Error
	case <-ctx.Done():
		return nil, nil, &TimeoutError{
			Operation: operation,
			Duration:  timeout,
			Context:   "processing engine",
		}
	}
}

func (pe *ProcessingEngine) ProcessImageWithTimeout(ctx context.Context, params *OtsuParameters) (*ImageData, *BinaryImageMetrics, error) {
	if pe.originalImage == nil {
		return nil, nil, fmt.Errorf("no original image loaded")
	}

	if err := validateMatForMetrics(pe.originalImage.Mat, "original image"); err != nil {
		return nil, nil, fmt.Errorf("image validation failed: %w", err)
	}

	imageSize := [2]int{pe.originalImage.Width, pe.originalImage.Height}
	if err := validateOtsuParameters(params, imageSize); err != nil {
		return nil, nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	timeout := pe.calculateTimeout(params)

	return withProcessingTimeout(ctx, timeout, "image processing", func() (*ImageData, *BinaryImageMetrics, error) {
		return pe.processImageSafely(ctx, params)
	})
}
