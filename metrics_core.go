package main

import (
	"fmt"

	"gocv.io/x/gocv"
)

type BinaryImageMetrics struct {
	TruePositives  int
	TrueNegatives  int
	FalsePositives int
	FalseNegatives int
	TotalPixels    int

	drdValue      float64
	mpmValue      float64
	pbcValue      float64
	skeletonValue float64
}

func CalculateBinaryMetrics(groundTruth, result gocv.Mat) (*BinaryImageMetrics, error) {
	if err := validateMatForMetrics(groundTruth, "ground truth"); err != nil {
		return nil, fmt.Errorf("ground truth validation: %w", err)
	}
	if err := validateMatForMetrics(result, "result"); err != nil {
		return nil, fmt.Errorf("result validation: %w", err)
	}
	if err := validateMatDimensionsMatch(groundTruth, result, "metrics calculation"); err != nil {
		return nil, fmt.Errorf("dimension validation: %w", err)
	}

	metrics := &BinaryImageMetrics{}

	if err := metrics.calculateConfusionMatrix(groundTruth, result); err != nil {
		return nil, fmt.Errorf("confusion matrix calculation: %w", err)
	}

	if err := metrics.calculateDRD(groundTruth, result); err != nil {
		return nil, fmt.Errorf("DRD calculation: %w", err)
	}

	if err := metrics.calculateMPM(groundTruth, result); err != nil {
		return nil, fmt.Errorf("MPM calculation: %w", err)
	}

	if err := metrics.calculateBackgroundForegroundContrast(groundTruth, result); err != nil {
		return nil, fmt.Errorf("BFC calculation: %w", err)
	}

	if err := metrics.calculateSkeletonSimilarity(groundTruth, result); err != nil {
		return nil, fmt.Errorf("skeleton similarity calculation: %w", err)
	}

	if err := validateAllMetrics(metrics); err != nil {
		return nil, fmt.Errorf("metrics validation: %w", err)
	}

	return metrics, nil
}

func (m *BinaryImageMetrics) calculateConfusionMatrix(groundTruth, result gocv.Mat) error {
	gtBinary, err := ensureBinaryThresholded(groundTruth, "ground truth confusion matrix")
	if err != nil {
		return err
	}
	defer gtBinary.Close()

	resBinary, err := ensureBinaryThresholded(result, "result confusion matrix")
	if err != nil {
		return err
	}
	defer resBinary.Close()

	rows := gtBinary.Rows()
	cols := gtBinary.Cols()
	m.TotalPixels = rows * cols

	m.TruePositives = 0
	m.TrueNegatives = 0
	m.FalsePositives = 0
	m.FalseNegatives = 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			gtValue := gtBinary.GetUCharAt(y, x) > 127
			resValue := resBinary.GetUCharAt(y, x) > 127

			if gtValue && resValue {
				m.TruePositives++
			} else if !gtValue && !resValue {
				m.TrueNegatives++
			} else if !gtValue && resValue {
				m.FalsePositives++
			} else {
				m.FalseNegatives++
			}
		}
	}

	return nil
}

func (m *BinaryImageMetrics) FMeasure() float64 {
	if m.TruePositives == 0 {
		return 0.0
	}

	precision := float64(m.TruePositives) / float64(m.TruePositives+m.FalsePositives)
	recall := float64(m.TruePositives) / float64(m.TruePositives+m.FalseNegatives)

	if precision == 0.0 || recall == 0.0 {
		return 0.0
	}

	return 2.0 * (precision * recall) / (precision + recall)
}

func (m *BinaryImageMetrics) PseudoFMeasure() float64 {
	if m.TruePositives == 0 {
		return 0.0
	}

	precision := float64(m.TruePositives) / float64(m.TruePositives+m.FalsePositives)
	recall := float64(m.TruePositives) / float64(m.TruePositives+m.FalseNegatives)

	if precision == 0.0 || recall == 0.0 {
		return 0.0
	}

	beta := 0.5
	betaSquared := beta * beta

	numerator := (1.0 + betaSquared) * precision * recall
	denominator := betaSquared*precision + recall

	if denominator == 0.0 {
		return 0.0
	}

	return numerator / denominator
}

func (m *BinaryImageMetrics) NRM() float64 {
	fn := float64(m.FalseNegatives)
	fp := float64(m.FalsePositives)
	tp := float64(m.TruePositives)
	tn := float64(m.TrueNegatives)

	numerator := fn + fp
	denominator := 2.0 * (tp + tn)

	if denominator == 0.0 {
		return 1.0
	}

	return numerator / denominator
}

func (m *BinaryImageMetrics) DRD() float64 {
	return m.drdValue
}

func (m *BinaryImageMetrics) MPM() float64 {
	return m.mpmValue
}

func (m *BinaryImageMetrics) BackgroundForegroundContrast() float64 {
	return m.pbcValue
}

func (m *BinaryImageMetrics) SkeletonSimilarity() float64 {
	return m.skeletonValue
}

func calculateSafeCountNonZero(mat gocv.Mat, context string) (int, error) {
	if err := validateMatForMetrics(mat, context); err != nil {
		return 0, err
	}

	var gray gocv.Mat
	if mat.Channels() == 1 {
		gray = mat
	} else {
		gray = gocv.NewMat()
		defer gray.Close()
		gocv.CvtColor(mat, &gray, gocv.ColorBGRToGray)
	}

	return gocv.CountNonZero(gray), nil
}

func createBinaryMask(mat gocv.Mat, threshold uint8) (gocv.Mat, error) {
	gray, err := normalizeToGrayscale(mat, "binary mask creation")
	if err != nil {
		return gocv.NewMat(), err
	}
	defer gray.Close()

	binary := gocv.NewMat()
	gocv.Threshold(gray, &binary, float64(threshold), 255, gocv.ThresholdBinary)
	return binary, nil
}

func performMatrixOperation(mat1, mat2 gocv.Mat, operation string) (gocv.Mat, error) {
	if err := validateMatDimensionsMatch(mat1, mat2, operation); err != nil {
		return gocv.NewMat(), err
	}

	result := gocv.NewMat()

	switch operation {
	case "and":
		gocv.BitwiseAnd(mat1, mat2, &result)
	case "or":
		gocv.BitwiseOr(mat1, mat2, &result)
	case "xor":
		gocv.BitwiseXor(mat1, mat2, &result)
	case "not":
		gocv.BitwiseNot(mat1, &result)
	default:
		result.Close()
		return gocv.NewMat(), fmt.Errorf("unsupported matrix operation: %s", operation)
	}

	return result, nil
}
