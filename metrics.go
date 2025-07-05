package main

import (
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

func (m *BinaryImageMetrics) FMeasure() float64 {
	if m.TruePositives+m.FalsePositives == 0 || m.TruePositives+m.FalseNegatives == 0 {
		return 0.0
	}

	precision := float64(m.TruePositives) / float64(m.TruePositives+m.FalsePositives)
	recall := float64(m.TruePositives) / float64(m.TruePositives+m.FalseNegatives)

	if precision+recall == 0 {
		return 0.0
	}

	return 2 * (precision * recall) / (precision + recall)
}

func (m *BinaryImageMetrics) PseudoFMeasure() float64 {
	if m.TruePositives == 0 {
		return 0.0
	}
	if m.TruePositives+m.FalsePositives == 0 || m.TruePositives+m.FalseNegatives == 0 {
		return 0.0
	}

	precision := float64(m.TruePositives) / float64(m.TruePositives+m.FalsePositives)
	recall := float64(m.TruePositives) / float64(m.TruePositives+m.FalseNegatives)

	beta := 0.5
	betaSquared := beta * beta

	if betaSquared*precision+recall == 0 {
		return 0.0
	}

	return (1 + betaSquared) * precision * recall / (betaSquared*precision + recall)
}

func (m *BinaryImageMetrics) NRM() float64 {
	fn := float64(m.FalseNegatives)
	fp := float64(m.FalsePositives)
	tp := float64(m.TruePositives)
	tn := float64(m.TrueNegatives)

	numerator := fn + fp
	denominator := 2 * (tp + tn)

	if denominator == 0 {
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
