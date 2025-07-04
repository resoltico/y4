package main

import (
	"fmt"
)

func validateOtsuParameters(params *OtsuParameters, imageSize [2]int) error {
	if params == nil {
		return &ValidationError{
			Context: "parameter validation",
			Field:   "params",
			Value:   nil,
			Reason:  "parameters object is nil",
		}
	}

	width, height := imageSize[0], imageSize[1]

	if params.WindowSize < 3 || params.WindowSize > 21 {
		return &ValidationError{
			Context: "parameter validation",
			Field:   "WindowSize",
			Value:   params.WindowSize,
			Reason:  "must be between 3 and 21",
		}
	}

	if params.WindowSize%2 == 0 {
		return &ValidationError{
			Context: "parameter validation",
			Field:   "WindowSize",
			Value:   params.WindowSize,
			Reason:  "must be odd number",
		}
	}

	if params.WindowSize >= min(width, height) {
		return &ValidationError{
			Context: "parameter validation",
			Field:   "WindowSize",
			Value:   params.WindowSize,
			Reason:  fmt.Sprintf("must be smaller than image dimensions %dx%d", width, height),
		}
	}

	if params.HistogramBins < 0 || params.HistogramBins > 256 {
		return &ValidationError{
			Context: "parameter validation",
			Field:   "HistogramBins",
			Value:   params.HistogramBins,
			Reason:  "must be 0 (auto) or between 1 and 256",
		}
	}

	if params.SmoothingStrength < 0.0 || params.SmoothingStrength > 10.0 {
		return &ValidationError{
			Context: "parameter validation",
			Field:   "SmoothingStrength",
			Value:   params.SmoothingStrength,
			Reason:  "must be between 0.0 and 10.0",
		}
	}

	if params.PyramidLevels < 1 || params.PyramidLevels > 8 {
		return &ValidationError{
			Context: "parameter validation",
			Field:   "PyramidLevels",
			Value:   params.PyramidLevels,
			Reason:  "must be between 1 and 8",
		}
	}

	minImageSize := min(width, height)
	maxPyramidLevels := 0
	testSize := minImageSize
	for testSize >= 16 {
		testSize /= 2
		maxPyramidLevels++
	}

	if params.PyramidLevels > maxPyramidLevels {
		return &ValidationError{
			Context: "parameter validation",
			Field:   "PyramidLevels",
			Value:   params.PyramidLevels,
			Reason:  fmt.Sprintf("maximum %d levels for %dx%d image", maxPyramidLevels, width, height),
		}
	}

	if params.RegionGridSize < 8 || params.RegionGridSize > min(width, height)/2 {
		maxGrid := min(width, height) / 2
		return &ValidationError{
			Context: "parameter validation",
			Field:   "RegionGridSize",
			Value:   params.RegionGridSize,
			Reason:  fmt.Sprintf("must be between 8 and %d for %dx%d image", maxGrid, width, height),
		}
	}

	if params.MorphologicalKernelSize < 1 || params.MorphologicalKernelSize > 15 {
		return &ValidationError{
			Context: "parameter validation",
			Field:   "MorphologicalKernelSize",
			Value:   params.MorphologicalKernelSize,
			Reason:  "must be between 1 and 15",
		}
	}

	if params.MorphologicalKernelSize%2 == 0 {
		return &ValidationError{
			Context: "parameter validation",
			Field:   "MorphologicalKernelSize",
			Value:   params.MorphologicalKernelSize,
			Reason:  "must be odd number",
		}
	}

	if params.DiffusionIterations < 1 || params.DiffusionIterations > 50 {
		return &ValidationError{
			Context: "parameter validation",
			Field:   "DiffusionIterations",
			Value:   params.DiffusionIterations,
			Reason:  "must be between 1 and 50",
		}
	}

	if params.DiffusionKappa < 1.0 || params.DiffusionKappa > 100.0 {
		return &ValidationError{
			Context: "parameter validation",
			Field:   "DiffusionKappa",
			Value:   params.DiffusionKappa,
			Reason:  "must be between 1.0 and 100.0",
		}
	}

	validNeighborhoods := map[string]bool{
		"Rectangular":       true,
		"Circular":          true,
		"Distance Weighted": true,
	}
	if !validNeighborhoods[params.NeighborhoodType] {
		return &ValidationError{
			Context: "parameter validation",
			Field:   "NeighborhoodType",
			Value:   params.NeighborhoodType,
			Reason:  "must be Rectangular, Circular, or Distance Weighted",
		}
	}

	validInterpolations := map[string]bool{
		"Nearest":  true,
		"Bilinear": true,
		"Bicubic":  true,
	}
	if !validInterpolations[params.InterpolationMethod] {
		return &ValidationError{
			Context: "parameter validation",
			Field:   "InterpolationMethod",
			Value:   params.InterpolationMethod,
			Reason:  "must be Nearest, Bilinear, or Bicubic",
		}
	}

	return nil
}

func validateHistogramData(histogram [][]float64, context string) error {
	if len(histogram) == 0 {
		return &ValidationError{
			Context: context,
			Field:   "histogram",
			Value:   "empty",
			Reason:  "histogram has no data",
		}
	}

	bins := len(histogram)
	if bins < 16 || bins > 256 {
		return &ValidationError{
			Context: context,
			Field:   "histogram_bins",
			Value:   bins,
			Reason:  "histogram must have between 16 and 256 bins",
		}
	}

	for i, row := range histogram {
		if len(row) != bins {
			return &ValidationError{
				Context: context,
				Field:   "histogram_shape",
				Value:   fmt.Sprintf("row %d has %d bins, expected %d", i, len(row), bins),
				Reason:  "histogram must be square matrix",
			}
		}

		for j, value := range row {
			if value < 0 {
				return &ValidationError{
					Context: context,
					Field:   "histogram_value",
					Value:   fmt.Sprintf("[%d,%d] = %f", i, j, value),
					Reason:  "histogram values must be non-negative",
				}
			}
		}
	}

	return nil
}

func validateThreshold(threshold [2]int, histogramBins int, context string) error {
	t1, t2 := threshold[0], threshold[1]

	if t1 < 0 || t1 >= histogramBins {
		return &ValidationError{
			Context: context,
			Field:   "threshold_t1",
			Value:   t1,
			Reason:  fmt.Sprintf("must be between 0 and %d", histogramBins-1),
		}
	}

	if t2 < 0 || t2 >= histogramBins {
		return &ValidationError{
			Context: context,
			Field:   "threshold_t2",
			Value:   t2,
			Reason:  fmt.Sprintf("must be between 0 and %d", histogramBins-1),
		}
	}

	return nil
}
