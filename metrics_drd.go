package main

import (
	"math"

	"gocv.io/x/gocv"
)

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

	totalForegroundPixels := 0
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if groundTruth.GetUCharAt(y, x) > 127 {
				totalForegroundPixels++
			}
		}
	}

	if totalForegroundPixels == 0 {
		m.drdValue = 0.0
		return
	}

	m.drdValue = totalDistortion / float64(totalForegroundPixels)
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

func (m *BinaryImageMetrics) calculateBackgroundForegroundContrast(groundTruth, result gocv.Mat) {
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
