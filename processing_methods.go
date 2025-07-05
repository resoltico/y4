package main

import (
	"image"

	"gocv.io/x/gocv"
)

func (pe *ProcessingEngine) processSingleScale(src gocv.Mat, params *OtsuParameters) gocv.Mat {
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
	return pe.applyThreshold(src, neighborhood, threshold, histBins)
}

func (pe *ProcessingEngine) processMultiScale(src gocv.Mat, params *OtsuParameters) gocv.Mat {
	levels := params.PyramidLevels
	if levels <= 0 {
		levels = 3
	}

	pyramid := make([]gocv.Mat, levels+1)
	pyramid[0] = src.Clone()

	for i := 1; i <= levels; i++ {
		pyramid[i] = gocv.NewMat()
		if err := gocv.PyrDown(pyramid[i-1], &pyramid[i], image.Point{}, gocv.BorderDefault); err != nil {
			pyramid[i].Close()
			pyramid[i] = pyramid[i-1].Clone()
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
			continue
		}

		combined := gocv.NewMat()
		gocv.BitwiseOr(results[i], upsampled, &combined)

		results[i].Close()
		upsampled.Close()
		results[i] = combined
	}

	return results[0]
}

func (pe *ProcessingEngine) processRegionAdaptive(src gocv.Mat, params *OtsuParameters) gocv.Mat {
	rows, cols := src.Rows(), src.Cols()
	gridSize := params.RegionGridSize
	if gridSize <= 0 {
		gridSize = max(rows/8, cols/8)
	}

	result := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)

	for y := 0; y < rows; y += gridSize {
		for x := 0; x < cols; x += gridSize {
			endY := min(y+gridSize, rows)
			endX := min(x+gridSize, cols)

			roi := src.Region(image.Rect(x, y, endX, endY))
			regionParams := *params
			regionParams.RegionAdaptiveThresholding = false
			regionResult := pe.processSingleScale(roi, &regionParams)

			rowRange := result.RowRange(y, endY)
			colRange := rowRange.ColRange(x, endX)
			regionResult.CopyTo(&colRange)

			roi.Close()
			regionResult.Close()
			rowRange.Close()
			colRange.Close()
		}
	}

	return result
}
