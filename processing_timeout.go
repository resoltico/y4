package main

import (
	"context"
	"fmt"
	"time"

	"gocv.io/x/gocv"
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

	if err := validateImageMat(pe.originalImage.Mat, "original image"); err != nil {
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

func (pe *ProcessingEngine) calculateTimeout(params *OtsuParameters) time.Duration {
	baseTimeout := DefaultTimeouts.SingleScale

	if params.MultiScaleProcessing {
		baseTimeout = DefaultTimeouts.MultiScale
		baseTimeout += time.Duration(params.PyramidLevels) * 15 * time.Second
	} else if params.RegionAdaptiveThresholding {
		baseTimeout = DefaultTimeouts.RegionAdaptive
		gridComplexity := (pe.originalImage.Width * pe.originalImage.Height) / (params.RegionGridSize * params.RegionGridSize)
		baseTimeout += time.Duration(gridComplexity/1000) * time.Second
	}

	if params.HomomorphicFiltering {
		baseTimeout += DefaultTimeouts.Preprocessing
	}
	if params.AnisotropicDiffusion {
		baseTimeout += time.Duration(params.DiffusionIterations) * 2 * time.Second
	}

	return baseTimeout
}

func (pe *ProcessingEngine) processImageSafely(ctx context.Context, params *OtsuParameters) (*ImageData, *BinaryImageMetrics, error) {
	gray := pe.convertToGrayscale(pe.originalImage.Mat)
	defer gray.Close()

	working := gray.Clone()
	defer working.Close()

	if params.HomomorphicFiltering {
		homomorphic := pe.applyHomomorphicFiltering(working)
		working.Close()
		working = homomorphic
	}

	if params.AnisotropicDiffusion {
		diffused := pe.applyAnisotropicDiffusion(working, params.DiffusionIterations, params.DiffusionKappa)
		working.Close()
		working = diffused
	}

	if params.GaussianPreprocessing {
		blurred := pe.applyGaussianBlur(working, params.SmoothingStrength)
		working.Close()
		working = blurred
	}

	if params.ApplyContrastEnhancement {
		enhanced := pe.applyAdaptiveContrastEnhancement(working)
		working.Close()
		working = enhanced
	}

	var result gocv.Mat
	if params.MultiScaleProcessing {
		result = pe.processMultiScale(working, params)
	} else if params.RegionAdaptiveThresholding {
		result = pe.processRegionAdaptive(working, params)
	} else {
		result = pe.processSingleScale(working, params)
	}
	defer result.Close()

	if params.MorphologicalPostProcess {
		morphed := pe.applyMorphologicalPostProcessing(result, params.MorphologicalKernelSize)
		result.Close()
		result = morphed
	}

	resultImage := pe.matToImage(result)

	processedData := &ImageData{
		Image:    resultImage,
		Mat:      result.Clone(),
		Width:    resultImage.Bounds().Dx(),
		Height:   resultImage.Bounds().Dy(),
		Channels: 1,
		Format:   pe.originalImage.Format,
	}

	pe.processedImage = processedData

	metrics := CalculateBinaryMetrics(pe.originalImage.Mat, result)
	if metrics == nil {
		return processedData, nil, fmt.Errorf("metrics calculation failed")
	}

	return processedData, metrics, nil
}
