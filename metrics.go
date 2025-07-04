package main

import (
	"image"
	"math"

	"gocv.io/x/gocv"
)

type BinaryImageMetrics struct {
	TruePositives  int
	TrueNegatives  int
	FalsePositives int
	FalseNegatives int
	TotalPixels    int

	drdValue float64
	mpmValue float64
	pbcValue float64
}

func CalculateBinaryMetrics(groundTruth, result gocv.Mat) *BinaryImageMetrics {
	if groundTruth.Rows() != result.Rows() || groundTruth.Cols() != result.Cols() {
		return nil
	}

	metrics := &BinaryImageMetrics{}
	metrics.calculateConfusionMatrix(groundTruth, result)
	metrics.calculateDRD(groundTruth, result)
	metrics.calculateMPM(groundTruth, result)
	metrics.calculatePBC(groundTruth, result)

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

	precision := float64(m.TruePositives) / float64(m.TruePositives+m.FalsePositives)
	recall := float64(m.TruePositives) / float64(m.TruePositives+m.FalseNegatives)

	weightedPrecision := precision * 0.8
	weightedRecall := recall * 1.2

	if weightedPrecision+weightedRecall == 0 {
		return 0.0
	}

	return 2 * (weightedPrecision * weightedRecall) / (weightedPrecision + weightedRecall)
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

func (m *BinaryImageMetrics) PBC() float64 {
	return m.pbcValue
}

func (m *BinaryImageMetrics) calculateDRD(groundTruth, result gocv.Mat) {
	rows := groundTruth.Rows()
	cols := groundTruth.Cols()

	weightMatrix := m.createDRDWeightMatrix()

	totalDistortion := 0.0
	totalErrorPixels := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			gtValue := groundTruth.GetUCharAt(y, x) > 127
			resValue := result.GetUCharAt(y, x) > 127

			if gtValue != resValue {
				totalErrorPixels++
				distortion := m.calculatePixelDRD(groundTruth, x, y, weightMatrix)
				totalDistortion += distortion
			}
		}
	}

	if totalErrorPixels == 0 {
		m.drdValue = 0.0
		return
	}

	m.drdValue = totalDistortion / float64(totalErrorPixels)
}

func (m *BinaryImageMetrics) createDRDWeightMatrix() [][]float64 {
	size := 5
	center := size / 2
	matrix := make([][]float64, size)

	for i := range matrix {
		matrix[i] = make([]float64, size)
		for j := range matrix[i] {
			dx := float64(i - center)
			dy := float64(j - center)
			distance := math.Sqrt(dx*dx + dy*dy)

			if distance == 0 {
				matrix[i][j] = 1.0
			} else {
				matrix[i][j] = 1.0 / distance
			}
		}
	}

	return matrix
}

func (m *BinaryImageMetrics) calculatePixelDRD(groundTruth gocv.Mat, x, y int, weightMatrix [][]float64) float64 {
	rows := groundTruth.Rows()
	cols := groundTruth.Cols()
	size := len(weightMatrix)
	center := size / 2

	weightedSum := 0.0
	totalWeight := 0.0

	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			nx := x + i - center
			ny := y + j - center

			if nx >= 0 && nx < cols && ny >= 0 && ny < rows {
				gtValue := groundTruth.GetUCharAt(ny, nx) > 127
				weight := weightMatrix[i][j]

				if gtValue {
					weightedSum += weight
				}
				totalWeight += weight
			}
		}
	}

	if totalWeight == 0 {
		return 0.0
	}

	return weightedSum / totalWeight
}

func (m *BinaryImageMetrics) calculateMPM(groundTruth, result gocv.Mat) {
	rows := groundTruth.Rows()
	cols := groundTruth.Cols()

	gtContours := m.extractContours(groundTruth)
	resContours := m.extractContours(result)

	if len(gtContours) == 0 && len(resContours) == 0 {
		m.mpmValue = 0.0
		return
	}

	totalMismatch := 0.0
	totalObjects := float64(len(gtContours) + len(resContours))

	for _, gtContour := range gtContours {
		minDistance := math.Inf(1)
		for _, resContour := range resContours {
			distance := m.calculateContourDistance(gtContour, resContour)
			if distance < minDistance {
				minDistance = distance
			}
		}
		if minDistance != math.Inf(1) {
			totalMismatch += minDistance
		} else {
			totalMismatch += float64(rows + cols)
		}
	}

	for _, resContour := range resContours {
		minDistance := math.Inf(1)
		for _, gtContour := range gtContours {
			distance := m.calculateContourDistance(resContour, gtContour)
			if distance < minDistance {
				minDistance = distance
			}
		}
		if minDistance != math.Inf(1) {
			totalMismatch += minDistance
		} else {
			totalMismatch += float64(rows + cols)
		}
	}

	if totalObjects == 0 {
		m.mpmValue = 0.0
		return
	}

	m.mpmValue = totalMismatch / totalObjects
}

func (m *BinaryImageMetrics) extractContours(mat gocv.Mat) [][]image.Point {
	contours := gocv.FindContours(mat, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	defer contours.Close()

	result := make([][]image.Point, contours.Size())
	for i := 0; i < contours.Size(); i++ {
		contour := contours.At(i)
		result[i] = contour.ToPoints()
		contour.Close()
	}

	return result
}

func (m *BinaryImageMetrics) calculateContourDistance(contour1, contour2 []image.Point) float64 {
	if len(contour1) == 0 || len(contour2) == 0 {
		return math.Inf(1)
	}

	minDistance := math.Inf(1)

	for _, p1 := range contour1 {
		for _, p2 := range contour2 {
			dx := float64(p1.X - p2.X)
			dy := float64(p1.Y - p2.Y)
			distance := math.Sqrt(dx*dx + dy*dy)

			if distance < minDistance {
				minDistance = distance
			}
		}
	}

	return minDistance
}

func (m *BinaryImageMetrics) calculatePBC(groundTruth, result gocv.Mat) {
	rows := groundTruth.Rows()
	cols := groundTruth.Cols()

	backgroundErrors := 0
	foregroundErrors := 0
	totalBackground := 0
	totalForeground := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			gtValue := groundTruth.GetUCharAt(y, x) > 127
			resValue := result.GetUCharAt(y, x) > 127

			if gtValue {
				totalForeground++
				if !resValue {
					foregroundErrors++
				}
			} else {
				totalBackground++
				if resValue {
					backgroundErrors++
				}
			}
		}
	}

	backgroundClutter := 0.0
	if totalBackground > 0 {
		backgroundClutter = float64(backgroundErrors) / float64(totalBackground)
	}

	foregroundSpeckle := 0.0
	if totalForeground > 0 {
		foregroundSpeckle = float64(foregroundErrors) / float64(totalForeground)
	}

	m.pbcValue = (backgroundClutter + foregroundSpeckle) / 2.0
}
