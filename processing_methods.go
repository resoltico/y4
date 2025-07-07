package main

import (
	"gocv.io/x/gocv"
)

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

	// Process regions using efficient RowRange/ColRange operations
	for y := 0; y < rows; y += gridSize {
		endY := min(y+gridSize, rows)
		srcRowRange := src.RowRange(y, endY)
		dstRowRange := result.RowRange(y, endY)

		for x := 0; x < cols; x += gridSize {
			endX := min(x+gridSize, cols)

			// Extract region using matrix slicing
			srcRegion := srcRowRange.ColRange(x, endX)

			if srcRegion.Rows() < 16 || srcRegion.Cols() < 16 {
				srcRegion.Close()
				regionErrors++
				continue
			}

			hasContrast, contrast, err := pe.validateRegionContrast(srcRegion)
			totalContrast += contrast

			if !hasContrast {
				lowContrastRegions++
				debugSystem := GetDebugSystem()
				debugSystem.logger.Debug("skipping low contrast region",
					"x", x, "y", y,
					"contrast", contrast,
					"error", err.Error())
				srcRegion.Close()
				regionsSkipped++
				continue
			}

			regionParams := *params
			regionParams.RegionAdaptiveThresholding = false
			regionResult := pe.processSingleScale(srcRegion, &regionParams)

			if !regionResult.Empty() {
				dstRegion := dstRowRange.ColRange(x, endX)
				regionResult.CopyTo(&dstRegion)
				dstRegion.Close()
				regionsProcessed++
			} else {
				regionErrors++
			}

			srcRegion.Close()
			regionResult.Close()
		}

		srcRowRange.Close()
		dstRowRange.Close()
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
