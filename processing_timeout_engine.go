package main

import (
	"context"
	"fmt"
	"time"

	"gocv.io/x/gocv"
)

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
	if err := validateProcessingInputs(pe.originalImage, params); err != nil {
		return nil, nil, fmt.Errorf("input validation: %w", err)
	}

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

	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
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

	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
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

	metrics, err := CalculateBinaryMetrics(gray, result)
	if err != nil {
		return processedData, nil, fmt.Errorf("metrics calculation: %w", err)
	}

	if err := validateProcessingResult(processedData, metrics); err != nil {
		return processedData, metrics, fmt.Errorf("result validation: %w", err)
	}

	return processedData, metrics, nil
}
