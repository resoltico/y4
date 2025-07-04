package main

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

type ValidationError struct {
	Context string
	Field   string
	Value   interface{}
	Reason  string
}

func (ve *ValidationError) Error() string {
	return fmt.Sprintf("%s: invalid %s value %v - %s", ve.Context, ve.Field, ve.Value, ve.Reason)
}

func validateImageMat(mat gocv.Mat, context string) error {
	if mat.Empty() {
		return &ValidationError{
			Context: context,
			Field:   "image",
			Value:   "empty",
			Reason:  "matrix contains no data",
		}
	}

	rows := mat.Rows()
	cols := mat.Cols()

	if rows <= 0 || cols <= 0 {
		return &ValidationError{
			Context: context,
			Field:   "dimensions",
			Value:   fmt.Sprintf("%dx%d", cols, rows),
			Reason:  "width and height must be positive",
		}
	}

	if rows < 3 || cols < 3 {
		return &ValidationError{
			Context: context,
			Field:   "dimensions",
			Value:   fmt.Sprintf("%dx%d", cols, rows),
			Reason:  "minimum size 3x3 required for processing",
		}
	}

	matType := mat.Type()
	if matType != gocv.MatTypeCV8UC1 && matType != gocv.MatTypeCV8UC3 && matType != gocv.MatTypeCV8UC4 {
		return &ValidationError{
			Context: context,
			Field:   "type",
			Value:   matType,
			Reason:  "only 8-bit unsigned images supported",
		}
	}

	return nil
}

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

func validateKernelSize(kernelSize int, imageWidth, imageHeight int, context string) error {
	if kernelSize < 1 {
		return &ValidationError{
			Context: context,
			Field:   "kernel_size",
			Value:   kernelSize,
			Reason:  "must be positive",
		}
	}

	if kernelSize%2 == 0 {
		return &ValidationError{
			Context: context,
			Field:   "kernel_size",
			Value:   kernelSize,
			Reason:  "must be odd number",
		}
	}

	if kernelSize >= imageWidth || kernelSize >= imageHeight {
		return &ValidationError{
			Context: context,
			Field:   "kernel_size",
			Value:   kernelSize,
			Reason:  fmt.Sprintf("must be smaller than image dimensions %dx%d", imageWidth, imageHeight),
		}
	}

	return nil
}

func safeMatCreation(rows, cols int, matType gocv.MatType, context string) (gocv.Mat, error) {
	if rows <= 0 || cols <= 0 {
		return gocv.Mat{}, &ValidationError{
			Context: context,
			Field:   "dimensions",
			Value:   fmt.Sprintf("%dx%d", cols, rows),
			Reason:  "dimensions must be positive",
		}
	}

	if rows > 32768 || cols > 32768 {
		return gocv.Mat{}, &ValidationError{
			Context: context,
			Field:   "dimensions",
			Value:   fmt.Sprintf("%dx%d", cols, rows),
			Reason:  "dimensions exceed maximum size 32768x32768",
		}
	}

	mat := gocv.NewMatWithSize(rows, cols, matType)
	if mat.Empty() {
		return gocv.Mat{}, &ValidationError{
			Context: context,
			Field:   "allocation",
			Value:   fmt.Sprintf("%dx%d type %d", cols, rows, matType),
			Reason:  "failed to allocate matrix memory",
		}
	}

	return mat, nil
}

func safeMatOperation(operation func() error, mats []gocv.Mat, context string) error {
	defer func() {
		if r := recover(); r != nil {
			for _, mat := range mats {
				if !mat.Empty() {
					mat.Close()
				}
			}
		}
	}()

	for i, mat := range mats {
		if err := validateImageMat(mat, fmt.Sprintf("%s mat[%d]", context, i)); err != nil {
			return err
		}
	}

	return operation()
}

func validateContourData(contours [][]image.Point, context string) error {
	if len(contours) == 0 {
		return nil // Empty contours is valid
	}

	for i, contour := range contours {
		if len(contour) < 3 {
			return &ValidationError{
				Context: context,
				Field:   "contour_points",
				Value:   fmt.Sprintf("contour %d has %d points", i, len(contour)),
				Reason:  "contours must have at least 3 points",
			}
		}

		for j, point := range contour {
			if point.X < 0 || point.Y < 0 {
				return &ValidationError{
					Context: context,
					Field:   "contour_coordinates",
					Value:   fmt.Sprintf("contour %d point %d: (%d,%d)", i, j, point.X, point.Y),
					Reason:  "coordinates must be non-negative",
				}
			}
		}
	}

	return nil
}
