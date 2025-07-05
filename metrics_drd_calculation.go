package main

import (
	"math"

	"gocv.io/x/gocv"
)

func (m *BinaryImageMetrics) calculateDRD(groundTruth, result gocv.Mat) error {
	gtBinary, err := ensureBinaryThresholded(groundTruth, "DRD ground truth")
	if err != nil {
		return err
	}
	defer gtBinary.Close()

	resBinary, err := ensureBinaryThresholded(result, "DRD result")
	if err != nil {
		return err
	}
	defer resBinary.Close()

	rows := gtBinary.Rows()
	cols := gtBinary.Cols()

	weightMatrix := createDRDWeightMatrix()

	totalDistortion := 0.0
	totalErrorPixels := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			gtValue := gtBinary.GetUCharAt(y, x) > 127
			resValue := resBinary.GetUCharAt(y, x) > 127

			if gtValue != resValue {
				totalErrorPixels++
				distortion := calculatePixelDRD(gtBinary, x, y, weightMatrix)
				totalDistortion += distortion
			}
		}
	}

	if totalErrorPixels == 0 {
		m.drdValue = 0.0
		return nil
	}

	totalForegroundPixels := 0
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if gtBinary.GetUCharAt(y, x) > 127 {
				totalForegroundPixels++
			}
		}
	}

	if totalForegroundPixels == 0 {
		m.drdValue = 0.0
		return nil
	}

	m.drdValue = totalDistortion / float64(totalForegroundPixels)
	return nil
}

func createDRDWeightMatrix() [][]float64 {
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

func calculatePixelDRD(groundTruth gocv.Mat, x, y int, weightMatrix [][]float64) float64 {
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

func (m *BinaryImageMetrics) calculateBackgroundForegroundContrast(groundTruth, result gocv.Mat) error {
	gtBinary, err := ensureBinaryThresholded(groundTruth, "BFC ground truth")
	if err != nil {
		return err
	}
	defer gtBinary.Close()

	resBinary, err := ensureBinaryThresholded(result, "BFC result")
	if err != nil {
		return err
	}
	defer resBinary.Close()

	rows := gtBinary.Rows()
	cols := gtBinary.Cols()

	backgroundErrors := 0
	foregroundErrors := 0
	totalBackground := 0
	totalForeground := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			gtValue := gtBinary.GetUCharAt(y, x) > 127
			resValue := resBinary.GetUCharAt(y, x) > 127

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
	return nil
}

func calculateLocalNeighborhoodStatistics(mat gocv.Mat, x, y, windowSize int) (float64, float64, error) {
	rows := mat.Rows()
	cols := mat.Cols()
	halfWindow := windowSize / 2

	sum := 0.0
	count := 0

	for dy := -halfWindow; dy <= halfWindow; dy++ {
		for dx := -halfWindow; dx <= halfWindow; dx++ {
			nx := x + dx
			ny := y + dy

			if nx >= 0 && nx < cols && ny >= 0 && ny < rows {
				value := float64(mat.GetUCharAt(ny, nx))
				sum += value
				count++
			}
		}
	}

	if count == 0 {
		return 0.0, 0.0, nil
	}

	mean := sum / float64(count)

	sumSquaredDiff := 0.0
	for dy := -halfWindow; dy <= halfWindow; dy++ {
		for dx := -halfWindow; dx <= halfWindow; dx++ {
			nx := x + dx
			ny := y + dy

			if nx >= 0 && nx < cols && ny >= 0 && ny < rows {
				value := float64(mat.GetUCharAt(ny, nx))
				diff := value - mean
				sumSquaredDiff += diff * diff
			}
		}
	}

	variance := sumSquaredDiff / float64(count)

	return mean, variance, nil
}

func applyAdaptiveFiltering(mat gocv.Mat, kernelSize int) (gocv.Mat, error) {
	if err := validateMatForMetrics(mat, "adaptive filtering"); err != nil {
		return gocv.NewMat(), err
	}

	result := gocv.NewMat()

	switch kernelSize {
	case 3:
		gocv.MedianBlur(mat, &result, 3)
	case 5:
		gocv.MedianBlur(mat, &result, 5)
	default:
		gocv.GaussianBlur(mat, &result, [2]int{kernelSize, kernelSize}, 0, 0, gocv.BorderDefault)
	}

	return result, nil
}
