package main

import (
	"fmt"

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

func validateMat(mat gocv.Mat, context string) error {
	if mat.Empty() {
		return fmt.Errorf("%s: matrix is empty", context)
	}
	if mat.Rows() <= 0 || mat.Cols() <= 0 {
		return fmt.Errorf("%s: invalid dimensions %dx%d", context, mat.Rows(), mat.Cols())
	}
	return nil
}

func validateMatDimensions(mat1, mat2 gocv.Mat, context string) error {
	if mat1.Rows() != mat2.Rows() || mat1.Cols() != mat2.Cols() {
		return fmt.Errorf("%s: dimension mismatch %dx%d vs %dx%d",
			context, mat1.Rows(), mat1.Cols(), mat2.Rows(), mat2.Cols())
	}
	return nil
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

	return nil
}
