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

	regionWidth := endX - x
	regionHeight := endY - y

	if err := validateImageDimensions(regionWidth, regionHeight, "region grid cell"); err != nil {
		return gocv.NewMat(), false
	}

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

	actualLevels := levels
	for i := 1; i <= levels; i++ {
		testRows := src.Rows() / (1 << i)
		testCols := src.Cols() / (1 << i)
		if testRows < 64 || testCols < 64 {
			actualLevels = i - 1
			break
		}
	}
	levels = actualLevels

	debugSystem := GetDebugSystem()
	debugSystem.logger.Debug("multi-scale pyramid levels calculated",
		"requested_levels", params.PyramidLevels,
		"actual_levels", levels,
		"source_size", fmt.Sprintf("%dx%d", src.Cols(), src.Rows()))

	if levels < 1 {
		debugSystem.logger.Warn("insufficient levels for multi-scale processing, using single scale")
		return pe.processSingleScale(src, params)
	}

	// Build pyramid using resize instead of unreliable PyrDown
	pyramid := make([]gocv.Mat, levels+1)
	pyramid[0] = src.Clone()

	for i := 1; i <= levels; i++ {
		prevLevel := pyramid[i-1]
		targetRows := prevLevel.Rows() / 2
		targetCols := prevLevel.Cols() / 2

		pyramid[i] = gocv.NewMat()
		err := gocv.Resize(prevLevel, &pyramid[i],
			image.Point{X: targetCols, Y: targetRows},
			0, 0, gocv.InterpolationArea)

		if err != nil {
			debugSystem.logger.Error("pyramid level resize failed", "level", i, "error", err)
			pyramid[i].Close()
			pyramid[i] = prevLevel.Clone()
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

	// Process each level with scale-appropriate parameters
	results := make([]gocv.Mat, levels+1)
	for i := 0; i <= levels; i++ {
		scaleParams := *params
		scaleParams.MultiScaleProcessing = false
		scaleParams.WindowSize = max(3, params.WindowSize/(1<<i))
		if scaleParams.WindowSize%2 == 0 {
			scaleParams.WindowSize++
		}

		if scaleParams.HistogramBins > 0 {
			scaleParams.HistogramBins = max(32, params.HistogramBins/(1<<i))
		}

		results[i] = pe.processSingleScale(pyramid[i], &scaleParams)
	}

	defer func() {
		for i := 1; i <= levels; i++ {
			results[i].Close()
		}
	}()

	// Combine results using weighted blending instead of OR
	combined := results[0].Clone()
	combinedFloat := gocv.NewMat()
	defer combinedFloat.Close()
	combined.ConvertTo(&combinedFloat, gocv.MatTypeCV32F)

	for i := levels - 1; i >= 0; i-- {
		if i == 0 {
			break
		}

		upsampled := gocv.NewMat()
		targetSize := image.Point{X: results[i].Cols(), Y: results[i].Rows()}

		err := gocv.Resize(results[i+1], &upsampled, targetSize, 0, 0, gocv.InterpolationLinear)
		if err != nil {
			debugSystem.logger.Error("upsampling failed", "level", i, "error", err)
			upsampled.Close()
			continue
		}

		upsampledFloat := gocv.NewMat()
		upsampled.ConvertTo(&upsampledFloat, gocv.MatTypeCV32F)

		// Weighted combination: fine details get higher weight
		weight := 0.3 / float64(i+1)

		gocv.AddWeighted(combinedFloat, 1.0-weight, upsampledFloat, weight, 0, &combinedFloat)

		upsampled.Close()
		upsampledFloat.Close()
	}

	// Convert back to binary
	result := gocv.NewMat()
	combinedFloat.ConvertTo(&result, gocv.MatTypeCV8U)

	if err := validateMatForMetrics(result, "multi-scale result"); err != nil {
		result.Close()
		return gocv.NewMat()
	}

	return result
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

	if gridSize < 32 {
		debugSystem := GetDebugSystem()
		debugSystem.logger.Warn("grid size too small, increasing",
			"old_size", gridSize, "new_size", 32)
		gridSize = 32
	}

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

			regionParams := *params
			regionParams.RegionAdaptiveThresholding = false
			regionResult := pe.processSingleScale(roi, &regionParams)

			if !regionResult.Empty() {
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

	debugSystem.TraceContrastAnalysis(0, totalRegions, lowContrastRegions, avgContrast)

	if err := validateMatForMetrics(result, "region adaptive result"); err != nil {
		result.Close()
		return gocv.NewMat()
	}

	return result
}
