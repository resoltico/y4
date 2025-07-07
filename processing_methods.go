package main

import (
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

	return pe.processMultiScalePyramid(src, params)
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
