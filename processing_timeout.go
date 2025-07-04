package main

import (
	"context"
	"fmt"
	"image"
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

func withProcessingTimeout[T any](ctx context.Context, timeout time.Duration, operation string, fn func() (T, error)) (T, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type result struct {
		value T
		err   error
	}

	done := make(chan result, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				var zero T
				done <- result{zero, fmt.Errorf("operation panicked: %v", r)}
			}
		}()

		value, err := fn()
		done <- result{value, err}
	}()

	select {
	case r := <-done:
		return r.value, r.err
	case <-ctx.Done():
		var zero T
		return zero, &TimeoutError{
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
		homomorphic, err := pe.applyHomomorphicFilteringWithTimeout(ctx, working)
		if err != nil {
			return nil, nil, fmt.Errorf("homomorphic filtering failed: %w", err)
		}
		working.Close()
		working = homomorphic
	}

	if params.AnisotropicDiffusion {
		diffused, err := pe.applyAnisotropicDiffusionWithTimeout(ctx, working, params.DiffusionIterations, params.DiffusionKappa)
		if err != nil {
			return nil, nil, fmt.Errorf("anisotropic diffusion failed: %w", err)
		}
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

	result, err := pe.processWithMethod(ctx, working, params)
	if err != nil {
		return nil, nil, fmt.Errorf("processing method failed: %w", err)
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

func (pe *ProcessingEngine) processWithMethod(ctx context.Context, src gocv.Mat, params *OtsuParameters) (gocv.Mat, error) {
	if params.MultiScaleProcessing {
		return pe.processMultiScaleWithTimeout(ctx, src, params)
	} else if params.RegionAdaptiveThresholding {
		return pe.processRegionAdaptiveWithTimeout(ctx, src, params)
	} else {
		return pe.processSingleScaleWithTimeout(ctx, src, params)
	}
}

func (pe *ProcessingEngine) processSingleScaleWithTimeout(ctx context.Context, src gocv.Mat, params *OtsuParameters) (gocv.Mat, error) {
	return withProcessingTimeout(ctx, DefaultTimeouts.SingleScale, "single scale processing", func() (gocv.Mat, error) {
		return pe.processSingleScaleBounded(src, params)
	})
}

func (pe *ProcessingEngine) processSingleScaleBounded(src gocv.Mat, params *OtsuParameters) (gocv.Mat, error) {
	windowSize := params.WindowSize
	if params.AdaptiveWindowSizing {
		windowSize = pe.calculateAdaptiveWindowSize(src)
	}

	if err := validateKernelSize(windowSize, src.Cols(), src.Rows(), "window size"); err != nil {
		return gocv.Mat{}, err
	}

	neighborhood := pe.calculateNeighborhood(src, windowSize, params.NeighborhoodType)
	defer neighborhood.Close()

	histBins := params.HistogramBins
	if histBins == 0 {
		histBins = pe.calculateHistogramBins(src)
	}

	histogram := pe.build2DHistogram(src, neighborhood, histBins)
	if err := validateHistogramData(histogram, "2D histogram"); err != nil {
		return gocv.Mat{}, err
	}

	if params.UseLogHistogram {
		pe.applyLogScaling(histogram)
	}

	if params.NormalizeHistogram {
		pe.normalizeHistogram(histogram)
	}

	if params.SmoothingStrength > 0 {
		pe.smoothHistogram(histogram, params.SmoothingStrength)
	}

	threshold := pe.find2DOtsuThresholdInteger(histogram)
	if err := validateThreshold(threshold, histBins, "Otsu threshold"); err != nil {
		return gocv.Mat{}, err
	}

	return pe.applyThreshold(src, neighborhood, threshold, histBins), nil
}

func (pe *ProcessingEngine) processMultiScaleWithTimeout(ctx context.Context, src gocv.Mat, params *OtsuParameters) (gocv.Mat, error) {
	timeout := time.Duration(params.PyramidLevels) * 30 * time.Second
	return withProcessingTimeout(ctx, timeout, "multi-scale processing", func() (gocv.Mat, error) {
		return pe.processMultiScaleBounded(src, params)
	})
}

func (pe *ProcessingEngine) processMultiScaleBounded(src gocv.Mat, params *OtsuParameters) (gocv.Mat, error) {
	levels := min(params.PyramidLevels, 8)
	if levels <= 0 {
		levels = 3
	}

	minSize := 16
	actualLevels := 0
	testRows, testCols := src.Rows(), src.Cols()
	for actualLevels < levels && testRows >= minSize && testCols >= minSize {
		testRows /= 2
		testCols /= 2
		actualLevels++
	}

	if actualLevels == 0 {
		return pe.processSingleScaleBounded(src, params)
	}

	pyramid := make([]gocv.Mat, actualLevels+1)
	pyramid[0] = src.Clone()
	defer pyramid[0].Close()

	for i := 1; i <= actualLevels; i++ {
		pyramid[i] = gocv.NewMat()
		defer pyramid[i].Close()

		if err := safeMatOperation(func() error {
			gocv.PyrDown(pyramid[i-1], &pyramid[i], image.Point{}, gocv.BorderDefault)
			return nil
		}, []gocv.Mat{pyramid[i-1]}, fmt.Sprintf("pyramid level %d", i)); err != nil {
			return gocv.Mat{}, err
		}
	}

	results := make([]gocv.Mat, actualLevels+1)
	for i := 0; i <= actualLevels; i++ {
		scaleParams := *params
		scaleParams.MultiScaleProcessing = false
		scaleParams.WindowSize = max(3, params.WindowSize/(1<<i))

		var err error
		results[i], err = pe.processSingleScaleBounded(pyramid[i], &scaleParams)
		if err != nil {
			for j := 0; j < i; j++ {
				results[j].Close()
			}
			return gocv.Mat{}, fmt.Errorf("pyramid level %d processing failed: %w", i, err)
		}
		defer results[i].Close()
	}

	for i := actualLevels - 1; i >= 0; i-- {
		upsampled := gocv.NewMat()
		defer upsampled.Close()

		gocv.PyrUp(results[i+1], &upsampled, image.Point{}, gocv.BorderDefault)

		combined := gocv.NewMat()
		defer combined.Close()

		gocv.BitwiseOr(results[i], upsampled, &combined)

		results[i].Close()
		results[i] = combined.Clone()
	}

	return results[0].Clone(), nil
}

func (pe *ProcessingEngine) processRegionAdaptiveWithTimeout(ctx context.Context, src gocv.Mat, params *OtsuParameters) (gocv.Mat, error) {
	gridComplexity := (src.Rows() * src.Cols()) / (params.RegionGridSize * params.RegionGridSize)
	timeout := DefaultTimeouts.RegionAdaptive + time.Duration(gridComplexity/1000)*time.Second

	return withProcessingTimeout(ctx, timeout, "region adaptive processing", func() (gocv.Mat, error) {
		return pe.processRegionAdaptiveBounded(src, params)
	})
}

func (pe *ProcessingEngine) processRegionAdaptiveBounded(src gocv.Mat, params *OtsuParameters) (gocv.Mat, error) {
	rows, cols := src.Rows(), src.Cols()
	gridSize := max(8, min(params.RegionGridSize, min(rows, cols)/2))

	result, err := safeMatCreation(rows, cols, gocv.MatTypeCV8UC1, "region adaptive result")
	if err != nil {
		return gocv.Mat{}, err
	}

	processedRegions := 0
	maxRegions := (rows/gridSize + 1) * (cols/gridSize + 1)

	for y := 0; y < rows; y += gridSize {
		for x := 0; x < cols; x += gridSize {
			if processedRegions > maxRegions {
				return gocv.Mat{}, fmt.Errorf("region processing exceeded maximum regions %d", maxRegions)
			}

			endY := min(y+gridSize, rows)
			endX := min(x+gridSize, cols)

			if endX-x < 8 || endY-y < 8 {
				continue
			}

			roi := src.Region(image.Rect(x, y, endX, endY))
			defer roi.Close()

			regionParams := *params
			regionParams.RegionAdaptiveThresholding = false
			regionParams.WindowSize = min(regionParams.WindowSize, min(endX-x, endY-y)/2)
			if regionParams.WindowSize%2 == 0 {
				regionParams.WindowSize--
			}
			if regionParams.WindowSize < 3 {
				regionParams.WindowSize = 3
			}

			regionResult, err := pe.processSingleScaleBounded(roi, &regionParams)
			if err != nil {
				return gocv.Mat{}, fmt.Errorf("region [%d,%d] processing failed: %w", x, y, err)
			}
			defer regionResult.Close()

			resultROI := result.Region(image.Rect(x, y, endX, endY))
			regionResult.CopyTo(&resultROI)
			resultROI.Close()

			processedRegions++
		}
	}

	return result, nil
}

func (pe *ProcessingEngine) applyHomomorphicFilteringWithTimeout(ctx context.Context, src gocv.Mat) (gocv.Mat, error) {
	return withProcessingTimeout(ctx, DefaultTimeouts.Preprocessing, "homomorphic filtering", func() (gocv.Mat, error) {
		return pe.applyHomomorphicFiltering(src), nil
	})
}

func (pe *ProcessingEngine) applyAnisotropicDiffusionWithTimeout(ctx context.Context, src gocv.Mat, iterations int, kappa float64) (gocv.Mat, error) {
	timeout := time.Duration(iterations) * 2 * time.Second
	return withProcessingTimeout(ctx, timeout, "anisotropic diffusion", func() (gocv.Mat, error) {
		return pe.applyAnisotropicDiffusion(src, iterations, kappa), nil
	})
}
