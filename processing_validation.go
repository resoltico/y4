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
	return validateMatForMetrics(mat, context)
}

func validateMatDimensions(mat1, mat2 gocv.Mat, context string) error {
	return validateMatDimensionsMatch(mat1, mat2, context)
}

func validateImageMat(mat gocv.Mat, context string) error {
	return validateMatForMetrics(mat, context)
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

	if params.DiffusionIterations < 1 || params.DiffusionIterations > 50 {
		return &ValidationError{
			Context: "parameter validation",
			Field:   "DiffusionIterations",
			Value:   params.DiffusionIterations,
			Reason:  "must be between 1 and 50",
		}
	}

	if params.DiffusionKappa < 1.0 || params.DiffusionKappa > 200.0 {
		return &ValidationError{
			Context: "parameter validation",
			Field:   "DiffusionKappa",
			Value:   params.DiffusionKappa,
			Reason:  "must be between 1.0 and 200.0",
		}
	}

	if params.RegionGridSize < 16 || params.RegionGridSize > 512 {
		return &ValidationError{
			Context: "parameter validation",
			Field:   "RegionGridSize",
			Value:   params.RegionGridSize,
			Reason:  "must be between 16 and 512",
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

	return nil
}

func validateProcessingInputs(originalImage *ImageData, params *OtsuParameters) error {
	if originalImage == nil {
		return fmt.Errorf("original image is nil")
	}

	if err := validateMatForMetrics(originalImage.Mat, "processing input"); err != nil {
		return fmt.Errorf("original image validation: %w", err)
	}

	imageSize := [2]int{originalImage.Width, originalImage.Height}
	if err := validateOtsuParameters(params, imageSize); err != nil {
		return fmt.Errorf("parameter validation: %w", err)
	}

	return nil
}

func validateProcessingResult(result *ImageData, metrics *BinaryImageMetrics) error {
	if result == nil {
		return fmt.Errorf("processing result is nil")
	}

	if err := validateMatForMetrics(result.Mat, "processing result"); err != nil {
		return fmt.Errorf("result image validation: %w", err)
	}

	if metrics != nil {
		if err := validateAllMetrics(metrics); err != nil {
			return fmt.Errorf("metrics validation: %w", err)
		}
	}

	return nil
}

func validateImageDimensions(width, height int, context string) error {
	if width <= 0 || height <= 0 {
		return &ValidationError{
			Context: context,
			Field:   "dimensions",
			Value:   fmt.Sprintf("%dx%d", width, height),
			Reason:  "width and height must be positive",
		}
	}

	if width < 3 || height < 3 {
		return &ValidationError{
			Context: context,
			Field:   "dimensions",
			Value:   fmt.Sprintf("%dx%d", width, height),
			Reason:  "minimum size 3x3 required for processing",
		}
	}

	if width > 32768 || height > 32768 {
		return &ValidationError{
			Context: context,
			Field:   "dimensions",
			Value:   fmt.Sprintf("%dx%d", width, height),
			Reason:  "exceeds maximum dimensions 32768x32768",
		}
	}

	return nil
}
