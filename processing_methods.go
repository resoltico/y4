package main

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

func (pe *ProcessingEngine) extractSafeRegion(src gocv.Mat, x, y, gridSize int) (gocv.Mat, bool) {
	rows, cols := src.Rows(), src.Cols()
	endY := min(y+gridSize, rows)
	endX := min(x+gridSize, cols)

	// Validate region dimensions
	regionWidth := endX - x
	regionHeight := endY - y

	if err := validateImageDimensions(regionWidth, regionHeight, "region grid cell"); err != nil {
		return gocv.NewMat(), false
	}

	// Ensure minimum meaningful size for processing
	if regionWidth < 16 || regionHeight < 16 {
		return gocv.NewMat(), false
	}

	roi := src.Region(image.Rect(x, y, endX, endY))
	if err := validateMatForMetrics(roi, "region ROI"); err != nil {
		roi.Close()
		return gocv.NewMat(), false
	}

	return roi, true
}

func (pe *ProcessingEngine) processSingleScale(src gocv.Mat, params *OtsuParameters) gocv.Mat {
	if err := validateMatForMetrics(src, "single scale processing"); err != nil {
		return gocv.NewMat()
	}

	windowSize := params.WindowSize
	if params.AdaptiveWindowSizing {
		windowSize = pe.calculateAdaptiveWindowSize(src)
	}

	neighborhood := pe.calculateNeighborhood(src, windowSize, params.NeighborhoodType)
	defer neighborhood.Close()

	histBins := params.HistogramBins
	if histBins == 0 {
		histBins = pe.calculateHistogramBins(src)
	}

	histogram := pe.build2DHistogram(src, neighborhood, histBins)

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
	result := pe.applyThreshold(src, neighborhood, threshold, histBins)

	if err := validateMatForMetrics(result, "single scale result"); err != nil {
		result.Close()
		return gocv.NewMat()
	}

	return result
}

func (pe *ProcessingEngine) processMultiScale(src gocv.Mat, params *OtsuParameters) gocv.Mat {
	if err := validateMatForMetrics(src, "multi-scale processing"); err != nil {
		return gocv.NewMat()
	}

	levels := params.PyramidLevels
	if levels <= 0 {
		levels = 3
	}

	// Calculate maximum usable levels based on minimum size requirements
	actualLevels := levels
	for i := 1; i <= levels; i++ {
		testRows := src.Rows() / (1 << i)
		testCols := src.Cols() / (1 << i)
		if testRows < 32 || testCols < 32 {
			actualLevels = i - 1
			break
		}
	}
	levels = actualLevels

	debugSystem := GetDebugSystem()
	debugSystem.logger.Debug("multi-scale pyramid planning",
		"requested_levels", params.PyramidLevels,
		"actual_levels", levels,
		"source_size", fmt.Sprintf("%dx%d", src.Cols(), src.Rows()))

	if levels < 1 {
		debugSystem.logger.Warn("image too small for multi-scale processing, using single scale")
		return pe.processSingleScale(src, params)
	}

	pyramid := make([]gocv.Mat, levels+1)
	pyramid[0] = src.Clone()

	for i := 1; i <= levels; i++ {
		pyramid[i] = gocv.NewMat()
		if err := gocv.PyrDown(pyramid[i-1], &pyramid[i], image.Point{}, gocv.BorderDefault); err != nil {
			pyramid[i].Close()
			pyramid[i] = pyramid[i-1].Clone()
		}

		if err := validateMatForMetrics(pyramid[i], "pyramid level"); err != nil {
			debugSystem.logger.Warn("pyramid level validation failed", "level", i, "error", err)
		}
	}

	defer func() {
		for i := 1; i <= levels; i++ {
			pyramid[i].Close()
		}
	}()

	results := make([]gocv.Mat, levels+1)
	for i := 0; i <= levels; i++ {
		scaleParams := *params
		scaleParams.MultiScaleProcessing = false
		scaleParams.WindowSize = max(3, params.WindowSize/(1<<i))
		results[i] = pe.processSingleScale(pyramid[i], &scaleParams)
	}

	defer func() {
		for i := 1; i <= levels; i++ {
			results[i].Close()
		}
	}()

	for i := levels - 1; i >= 0; i-- {
		upsampled := gocv.NewMat()
		if err := gocv.PyrUp(results[i+1], &upsampled, image.Point{}, gocv.BorderDefault); err != nil {
			upsampled.Close()
			debugSystem.logger.Error("pyramid upsampling failed", "level", i, "error", err)
			continue
		}

		if err := validateMatForMetrics(upsampled, "pyramid upsampled"); err != nil {
			upsampled.Close()
			debugSystem.logger.Error("upsampled matrix validation failed", "level", i, "error", err)
			continue
		}

		// Resize upsampled to match current level dimensions
		if upsampled.Rows() != results[i].Rows() || upsampled.Cols() != results[i].Cols() {
			resized := gocv.NewMat()
			gocv.Resize(upsampled, &resized, image.Point{X: results[i].Cols(), Y: results[i].Rows()}, 0, 0, gocv.InterpolationLinear)
			upsampled.Close()
			upsampled = resized
		}

		combined, err := performMatrixOperation(results[i], upsampled, "or")
		if err != nil {
			upsampled.Close()
			debugSystem.logger.Error("pyramid combination failed", "level", i, "error", err)
			continue
		}

		results[i].Close()
		upsampled.Close()
		results[i] = combined
	}

	if err := validateMatForMetrics(results[0], "multi-scale result"); err != nil {
		results[0].Close()
		return gocv.NewMat()
	}

	return results[0]
}

func (pe *ProcessingEngine) processRegionAdaptive(src gocv.Mat, params *OtsuParameters) gocv.Mat {
	if err := validateMatForMetrics(src, "region adaptive processing"); err != nil {
		return gocv.NewMat()
	}

	rows, cols := src.Rows(), src.Cols()

	if err := validateImageDimensions(cols, rows, "region adaptive dimensions"); err != nil {
		return gocv.NewMat()
	}

	gridSize := params.RegionGridSize
	if gridSize <= 0 {
		gridSize = max(rows/8, cols/8)
	}

	// Ensure minimum grid size for meaningful processing
	if gridSize < 32 {
		debugSystem := GetDebugSystem()
		debugSystem.logger.Warn("grid size too small, increasing",
			"old_size", gridSize, "new_size", 32)
		gridSize = 32
	}

	// Validate grid size makes sense for image dimensions
	if gridSize > min(rows, cols)/2 {
		debugSystem := GetDebugSystem()
		debugSystem.logger.Warn("grid size too large for image dimensions",
			"grid_size", gridSize,
			"image_rows", rows,
			"image_cols", cols)
		gridSize = min(rows, cols) / 4
	}

	result := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)

	regionsProcessed := 0
	regionErrors := 0
	regionsSkipped := 0
	lowContrastRegions := 0
	totalContrast := 0.0

	for y := 0; y < rows; y += gridSize {
		for x := 0; x < cols; x += gridSize {
			roi, isValid := pe.extractSafeRegion(src, x, y, gridSize)
			if !isValid {
				regionErrors++
				continue
			}

			// Check contrast before processing
			hasContrast, contrast, err := pe.validateRegionContrast(roi)
			totalContrast += contrast

			if !hasContrast {
				lowContrastRegions++
				debugSystem := GetDebugSystem()
				debugSystem.logger.Debug("skipping low contrast region",
					"x", x, "y", y,
					"contrast", contrast,
					"error", err.Error())
				roi.Close()
				regionsSkipped++
				continue
			}

			// Process region normally
			regionParams := *params
			regionParams.RegionAdaptiveThresholding = false
			regionResult := pe.processSingleScale(roi, &regionParams)

			if !regionResult.Empty() {
				// Copy successful result
				endY := min(y+regionResult.Rows(), rows)
				endX := min(x+regionResult.Cols(), cols)

				rowRange := result.RowRange(y, endY)
				colRange := rowRange.ColRange(x, endX)
				regionResult.CopyTo(&colRange)
				rowRange.Close()
				colRange.Close()
				regionsProcessed++
			} else {
				regionErrors++
			}

			roi.Close()
			regionResult.Close()
		}
	}

	debugSystem := GetDebugSystem()
	totalRegions := regionsProcessed + regionErrors + regionsSkipped
	avgContrast := 0.0
	if totalRegions > 0 {
		avgContrast = totalContrast / float64(totalRegions)
	}

	debugSystem.logger.Info("region adaptive processing complete",
		"regions_processed", regionsProcessed,
		"regions_skipped", regionsSkipped,
		"region_errors", regionErrors,
		"low_contrast_regions", lowContrastRegions,
		"average_contrast", avgContrast,
		"grid_size", gridSize,
		"image_dimensions", []int{cols, rows})

	// Log contrast analysis summary
	debugSystem.TraceContrastAnalysis(0, totalRegions, lowContrastRegions, avgContrast)

	if err := validateMatForMetrics(result, "region adaptive result"); err != nil {
		result.Close()
		return gocv.NewMat()
	}

	return result
}
