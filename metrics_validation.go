package main

import (
	"fmt"

	"gocv.io/x/gocv"
)

type MatValidationError struct {
	Context string
	Issue   string
	MatInfo string
}

func (e *MatValidationError) Error() string {
	return fmt.Sprintf("%s: %s (%s)", e.Context, e.Issue, e.MatInfo)
}

type ValidationError struct {
	Context string
	Field   string
	Value   interface{}
	Reason  string
}

func (ve *ValidationError) Error() string {
	return fmt.Sprintf("%s: invalid %s value %v - %s", ve.Context, ve.Field, ve.Value, ve.Reason)
}

func validateMatForMetrics(mat gocv.Mat, context string) error {
	if mat.Empty() {
		return &MatValidationError{
			Context: context,
			Issue:   "matrix contains no data",
			MatInfo: "empty",
		}
	}

	rows := mat.Rows()
	cols := mat.Cols()
	matType := mat.Type()

	if rows <= 0 || cols <= 0 {
		return &MatValidationError{
			Context: context,
			Issue:   "invalid matrix dimensions",
			MatInfo: fmt.Sprintf("%dx%d", cols, rows),
		}
	}

	if rows < 3 || cols < 3 {
		return &MatValidationError{
			Context: context,
			Issue:   "matrix too small for metrics calculation",
			MatInfo: fmt.Sprintf("%dx%d (minimum 3x3)", cols, rows),
		}
	}

	if rows > 32768 || cols > 32768 {
		return &MatValidationError{
			Context: context,
			Issue:   "matrix exceeds maximum dimensions",
			MatInfo: fmt.Sprintf("%dx%d (maximum 32768x32768)", cols, rows),
		}
	}

	supportedTypes := []gocv.MatType{
		gocv.MatTypeCV8UC1,
		gocv.MatTypeCV8UC3,
		gocv.MatTypeCV8UC4,
	}

	typeSupported := false
	for _, supportedType := range supportedTypes {
		if matType == supportedType {
			typeSupported = true
			break
		}
	}

	if !typeSupported {
		return &MatValidationError{
			Context: context,
			Issue:   "unsupported matrix type",
			MatInfo: fmt.Sprintf("type %d (supported: CV8UC1, CV8UC3, CV8UC4)", int(matType)),
		}
	}

	return nil
}

func validateMatDimensionsMatch(mat1, mat2 gocv.Mat, context string) error {
	if mat1.Rows() != mat2.Rows() || mat1.Cols() != mat2.Cols() {
		return &MatValidationError{
			Context: context,
			Issue:   "matrix dimensions do not match",
			MatInfo: fmt.Sprintf("%dx%d vs %dx%d", mat1.Cols(), mat1.Rows(), mat2.Cols(), mat2.Rows()),
		}
	}
	return nil
}

func validateBinaryMat(mat gocv.Mat, context string) error {
	if err := validateMatForMetrics(mat, context); err != nil {
		return err
	}

	var gray gocv.Mat
	if mat.Channels() > 1 {
		gray = gocv.NewMat()
		defer gray.Close()
		gocv.CvtColor(mat, &gray, gocv.ColorBGRToGray)
	} else {
		gray = mat
	}

	minVal, maxVal, _, _ := gocv.MinMaxLoc(gray)
	valueRange := float64(maxVal - minVal)

	debugSystem := GetDebugSystem()

	if valueRange < 1e-6 {
		debugSystem.logger.Error("binary matrix validation failed - uniform values detected",
			"context", context,
			"min_value", float64(minVal),
			"max_value", float64(maxVal),
			"value_range", valueRange,
			"matrix_dimensions", fmt.Sprintf("%dx%d", mat.Cols(), mat.Rows()),
			"matrix_type", int(mat.Type()),
		)

		return &MatValidationError{
			Context: context,
			Issue:   "matrix contains uniform values",
			MatInfo: fmt.Sprintf("min=%.6f max=%.6f", float64(minVal), float64(maxVal)),
		}
	}

	debugSystem.logger.Debug("binary matrix validation passed",
		"context", context,
		"min_value", float64(minVal),
		"max_value", float64(maxVal),
		"value_range", valueRange,
	)

	return nil
}

func normalizeToGrayscale(src gocv.Mat, context string) (gocv.Mat, error) {
	if err := validateMatForMetrics(src, context); err != nil {
		return gocv.NewMat(), err
	}

	if src.Channels() == 1 {
		return src.Clone(), nil
	}

	gray := gocv.NewMat()

	// Handle different channel counts including transparency
	switch src.Channels() {
	case 3:
		gocv.CvtColor(src, &gray, gocv.ColorBGRToGray)
	case 4:
		// Convert BGRA to BGR first, then to grayscale
		bgr := gocv.NewMat()
		defer bgr.Close()
		gocv.CvtColor(src, &bgr, gocv.ColorBGRAToBGR)
		gocv.CvtColor(bgr, &gray, gocv.ColorBGRToGray)
	default:
		gray.Close()
		return gocv.NewMat(), &MatValidationError{
			Context: context,
			Issue:   "unsupported channel count for grayscale conversion",
			MatInfo: fmt.Sprintf("channels=%d", src.Channels()),
		}
	}

	return gray, nil
}

func ensureBinaryThresholded(src gocv.Mat, context string) (gocv.Mat, error) {
	gray, err := normalizeToGrayscale(src, context)
	if err != nil {
		return gocv.NewMat(), err
	}
	defer gray.Close()

	debugSystem := GetDebugSystem()

	// Check input statistics before thresholding
	minVal, maxVal, _, _ := gocv.MinMaxLoc(gray)
	inputRange := float64(maxVal - minVal)

	debugSystem.logger.Debug("binary thresholding input analysis",
		"context", context,
		"input_min", float64(minVal),
		"input_max", float64(maxVal),
		"input_range", inputRange,
	)

	if err := validateBinaryMat(gray, context+" grayscale"); err != nil {
		debugSystem.logger.Error("binary thresholding input validation failed",
			"context", context,
			"error", err.Error(),
		)
		return gocv.NewMat(), fmt.Errorf("binary validation failed: %w", err)
	}

	binary := gocv.NewMat()
	gocv.Threshold(gray, &binary, 127, 255, gocv.ThresholdBinary)

	// Validate thresholding result
	binaryMin, binaryMax, _, _ := gocv.MinMaxLoc(binary)
	binaryRange := float64(binaryMax - binaryMin)
	foregroundPixels, _ := calculateSafeCountNonZero(binary, context+" binary result")
	totalPixels := binary.Rows() * binary.Cols()
	foregroundRatio := float64(foregroundPixels) / float64(totalPixels)

	debugSystem.logger.Debug("binary thresholding result analysis",
		"context", context,
		"output_min", float64(binaryMin),
		"output_max", float64(binaryMax),
		"output_range", binaryRange,
		"foreground_pixels", foregroundPixels,
		"total_pixels", totalPixels,
		"foreground_ratio", foregroundRatio,
	)

	if binaryRange < 1e-6 {
		debugSystem.logger.Error("binary thresholding produced uniform output",
			"context", context,
			"uniform_value", float64(binaryMin),
			"input_range", inputRange,
		)
	}

	return binary, nil
}

func calculatePixelStatistics(mat gocv.Mat) (int, int, int, int, error) {
	if err := validateMatForMetrics(mat, "pixel statistics"); err != nil {
		return 0, 0, 0, 0, err
	}

	binary, err := ensureBinaryThresholded(mat, "pixel statistics")
	if err != nil {
		return 0, 0, 0, 0, err
	}
	defer binary.Close()

	rows := binary.Rows()
	cols := binary.Cols()
	totalPixels := rows * cols

	foregroundPixels := gocv.CountNonZero(binary)
	backgroundPixels := totalPixels - foregroundPixels

	debugSystem := GetDebugSystem()
	debugSystem.logger.Debug("pixel statistics calculated",
		"total_pixels", totalPixels,
		"foreground_pixels", foregroundPixels,
		"background_pixels", backgroundPixels,
		"foreground_ratio", float64(foregroundPixels)/float64(totalPixels),
	)

	return totalPixels, foregroundPixels, backgroundPixels, 0, nil
}

func validateMetricRange(value float64, metricName string) error {
	if value < 0.0 || value > 1.0 {
		return fmt.Errorf("%s value %.6f outside valid range [0.0, 1.0]", metricName, value)
	}
	return nil
}

func validateMetricNotNaN(value float64, metricName string) error {
	if value != value {
		return fmt.Errorf("%s value is NaN", metricName)
	}
	return nil
}

func validateMetricFinite(value float64, metricName string) error {
	if value > 1e10 || value < -1e10 {
		return fmt.Errorf("%s value %.6f is not finite", metricName, value)
	}
	return nil
}

func validateAllMetrics(metrics *BinaryImageMetrics) error {
	if metrics == nil {
		return fmt.Errorf("metrics object is nil")
	}

	debugSystem := GetDebugSystem()

	metricValues := map[string]float64{
		"F-measure":                      metrics.FMeasure(),
		"Pseudo F-measure":               metrics.PseudoFMeasure(),
		"NRM":                            metrics.NRM(),
		"DRD":                            metrics.DRD(),
		"MPM":                            metrics.MPM(),
		"Background Foreground Contrast": metrics.BackgroundForegroundContrast(),
		"Skeleton Similarity":            metrics.SkeletonSimilarity(),
	}

	debugSystem.logger.Debug("validating calculated metrics",
		"f_measure", metricValues["F-measure"],
		"pseudo_f_measure", metricValues["Pseudo F-measure"],
		"nrm", metricValues["NRM"],
		"drd", metricValues["DRD"],
		"mpm", metricValues["MPM"],
		"bfc", metricValues["Background Foreground Contrast"],
		"skeleton", metricValues["Skeleton Similarity"],
	)

	for name, value := range metricValues {
		if err := validateMetricNotNaN(value, name); err != nil {
			debugSystem.logger.Error("metric validation failed - NaN detected",
				"metric_name", name,
				"error", err.Error(),
			)
			return err
		}
		if err := validateMetricFinite(value, name); err != nil {
			debugSystem.logger.Error("metric validation failed - infinite value",
				"metric_name", name,
				"metric_value", value,
				"error", err.Error(),
			)
			return err
		}
		if name != "DRD" && name != "MPM" {
			if err := validateMetricRange(value, name); err != nil {
				debugSystem.logger.Error("metric validation failed - out of range",
					"metric_name", name,
					"metric_value", value,
					"error", err.Error(),
				)
				return err
			}
		}
	}

	confusionMatrixSum := metrics.TruePositives + metrics.TrueNegatives + metrics.FalsePositives + metrics.FalseNegatives
	if confusionMatrixSum != metrics.TotalPixels {
		debugSystem.logger.Error("confusion matrix validation failed",
			"confusion_sum", confusionMatrixSum,
			"total_pixels", metrics.TotalPixels,
			"true_positives", metrics.TruePositives,
			"true_negatives", metrics.TrueNegatives,
			"false_positives", metrics.FalsePositives,
			"false_negatives", metrics.FalseNegatives,
		)
		return fmt.Errorf("confusion matrix sum %d does not match total pixels %d",
			confusionMatrixSum, metrics.TotalPixels)
	}

	debugSystem.logger.Debug("metrics validation passed",
		"confusion_matrix_sum", confusionMatrixSum,
		"total_pixels", metrics.TotalPixels,
	)

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

	if err := validateImageDimensions(width, height, "parameter validation"); err != nil {
		return err
	}

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

func validateTransparencyHandling(mat gocv.Mat, context string) error {
	if mat.Empty() {
		return &MatValidationError{
			Context: context,
			Issue:   "transparency validation on empty matrix",
			MatInfo: "empty",
		}
	}

	channels := mat.Channels()

	switch channels {
	case 1, 3:
		// Standard cases - no transparency
		return nil
	case 4:
		// BGRA format with alpha channel
		debugSystem := GetDebugSystem()
		debugSystem.logger.Debug("transparency detected in matrix",
			"context", context,
			"channels", channels,
			"dimensions", fmt.Sprintf("%dx%d", mat.Cols(), mat.Rows()),
		)
		return nil
	default:
		return &MatValidationError{
			Context: context,
			Issue:   "unsupported channel count",
			MatInfo: fmt.Sprintf("channels=%d (supported: 1, 3, 4)", channels),
		}
	}
}

func validateProcessingInputs(originalImage *ImageData, params *OtsuParameters) error {
	if originalImage == nil {
		return fmt.Errorf("original image is nil")
	}

	debugSystem := GetDebugSystem()
	debugSystem.logger.Debug("validating processing inputs",
		"image_width", originalImage.Width,
		"image_height", originalImage.Height,
		"image_channels", originalImage.Channels,
		"image_format", originalImage.Format,
	)

	if err := validateMatForMetrics(originalImage.Mat, "processing input"); err != nil {
		debugSystem.logger.Error("processing input matrix validation failed",
			"error", err.Error(),
		)
		return fmt.Errorf("original image validation: %w", err)
	}

	if err := validateTransparencyHandling(originalImage.Mat, "processing input"); err != nil {
		debugSystem.logger.Error("transparency validation failed",
			"error", err.Error(),
		)
		return fmt.Errorf("transparency validation: %w", err)
	}

	imageSize := [2]int{originalImage.Width, originalImage.Height}
	if err := validateOtsuParameters(params, imageSize); err != nil {
		debugSystem.logger.Error("parameter validation failed",
			"error", err.Error(),
		)
		return fmt.Errorf("parameter validation: %w", err)
	}

	debugSystem.logger.Debug("processing inputs validation passed")
	return nil
}

func validateProcessingResult(result *ImageData, metrics *BinaryImageMetrics) error {
	if result == nil {
		return fmt.Errorf("processing result is nil")
	}

	debugSystem := GetDebugSystem()
	debugSystem.logger.Debug("validating processing result",
		"result_width", result.Width,
		"result_height", result.Height,
		"result_channels", result.Channels,
	)

	if err := validateMatForMetrics(result.Mat, "processing result"); err != nil {
		debugSystem.logger.Error("processing result matrix validation failed",
			"error", err.Error(),
		)
		return fmt.Errorf("result image validation: %w", err)
	}

	// Check for uniform output in result
	minVal, maxVal, _, _ := gocv.MinMaxLoc(result.Mat)
	if minVal == maxVal {
		debugSystem.logger.Error("processing result contains uniform values",
			"uniform_value", float64(minVal),
			"result_dimensions", fmt.Sprintf("%dx%d", result.Width, result.Height),
		)
	} else {
		debugSystem.logger.Debug("processing result has valid value range",
			"min_value", float64(minVal),
			"max_value", float64(maxVal),
			"value_range", float64(maxVal-minVal),
		)
	}

	if metrics != nil {
		if err := validateAllMetrics(metrics); err != nil {
			debugSystem.logger.Error("metrics validation failed",
				"error", err.Error(),
			)
			return fmt.Errorf("metrics validation: %w", err)
		}
		debugSystem.logger.Debug("metrics validation passed")
	}

	debugSystem.logger.Debug("processing result validation passed")
	return nil
}
