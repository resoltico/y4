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

func safeCountNonZero(mat gocv.Mat, context string) (int, error) {
	if err := validateMat(mat, context); err != nil {
		return 0, err
	}

	if mat.Channels() == 1 {
		return gocv.CountNonZero(mat), nil
	}

	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(mat, &gray, gocv.ColorBGRToGray)

	return gocv.CountNonZero(gray), nil
}

func CalculateBinaryMetrics(groundTruth, result gocv.Mat) *BinaryImageMetrics {
	if err := validateMat(groundTruth, "ground truth"); err != nil {
		return nil
	}
	if err := validateMat(result, "result"); err != nil {
		return nil
	}
	if err := validateMatDimensions(groundTruth, result, "metrics calculation"); err != nil {
		return nil
	}

	metrics := &BinaryImageMetrics{}
	metrics.calculateConfusionMatrix(groundTruth, result)
	metrics.calculateDRD(groundTruth, result)
	metrics.calculateMPM(groundTruth, result)
	metrics.calculateBackgroundForegroundContrast(groundTruth, result)
	metrics.calculateSkeletonSimilarity(groundTruth, result)

	return metrics
}

func (m *BinaryImageMetrics) calculateConfusionMatrix(groundTruth, result gocv.Mat) {
	rows := groundTruth.Rows()
	cols := groundTruth.Cols()
	m.TotalPixels = rows * cols

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			gtValue := groundTruth.GetUCharAt(y, x)
			resValue := result.GetUCharAt(y, x)

			gtBinary := gtValue > 127
			resBinary := resValue > 127

			if gtBinary && resBinary {
				m.TruePositives++
			} else if !gtBinary && !resBinary {
				m.TrueNegatives++
			} else if !gtBinary && resBinary {
				m.FalsePositives++
			} else {
				m.FalseNegatives++
			}
		}
	}
}
