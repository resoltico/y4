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

	var minVal, maxVal float64
	gocv.MinMaxLoc(gray, &minVal, &maxVal)

	if maxVal-minVal < 1e-6 {
		return &MatValidationError{
			Context: context,
			Issue:   "matrix contains uniform values",
			MatInfo: fmt.Sprintf("min=%.6f max=%.6f", minVal, maxVal),
		}
	}

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
	gocv.CvtColor(src, &gray, gocv.ColorBGRToGray)
	return gray, nil
}

func ensureBinaryThresholded(src gocv.Mat, context string) (gocv.Mat, error) {
	gray, err := normalizeToGrayscale(src, context)
	if err != nil {
		return gocv.NewMat(), err
	}
	defer gray.Close()

	binary := gocv.NewMat()
	gocv.Threshold(gray, &binary, 127, 255, gocv.ThresholdBinary)
	return binary, nil
}

func calculatePixelStatistics(mat gocv.Mat) (int, int, int, int, error) {
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

	return totalPixels, foregroundPixels, backgroundPixels, 0, nil
}

func validateMetricRange(value float64, metricName string) error {
	if value < 0.0 || value > 1.0 {
		return fmt.Errorf("%s value %.6f outside valid range [0.0, 1.0]", metricName, value)
	}
	return nil
}

func validateMetricNotNaN(value float64, metricName string) error {
	if value != value { // NaN check
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

	metricValues := map[string]float64{
		"F-measure":                      metrics.FMeasure(),
		"Pseudo F-measure":               metrics.PseudoFMeasure(),
		"NRM":                            metrics.NRM(),
		"DRD":                            metrics.DRD(),
		"MPM":                            metrics.MPM(),
		"Background Foreground Contrast": metrics.BackgroundForegroundContrast(),
		"Skeleton Similarity":            metrics.SkeletonSimilarity(),
	}

	for name, value := range metricValues {
		if err := validateMetricNotNaN(value, name); err != nil {
			return err
		}
		if err := validateMetricFinite(value, name); err != nil {
			return err
		}
		if name != "DRD" && name != "MPM" {
			if err := validateMetricRange(value, name); err != nil {
				return err
			}
		}
	}

	confusionMatrixSum := metrics.TruePositives + metrics.TrueNegatives + metrics.FalsePositives + metrics.FalseNegatives
	if confusionMatrixSum != metrics.TotalPixels {
		return fmt.Errorf("confusion matrix sum %d does not match total pixels %d",
			confusionMatrixSum, metrics.TotalPixels)
	}

	return nil
}
