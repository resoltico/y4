package main

import (
	"math"

	"gocv.io/x/gocv"
)

func (pe *ProcessingEngine) calculateAdaptiveWindowSize(src gocv.Mat) int {
	if err := validateMatForMetrics(src, "adaptive window size calculation"); err != nil {
		return 7 // safe default
	}

	rows, cols := src.Rows(), src.Cols()
	minDimension := min(rows, cols)

	if minDimension < 100 {
		return 3
	} else if minDimension < 500 {
		return 5
	} else if minDimension < 1000 {
		return 7
	} else {
		return 9
	}
}

func (pe *ProcessingEngine) calculateHistogramBins(src gocv.Mat) int {
	if err := validateMatForMetrics(src, "histogram bins calculation"); err != nil {
		return 64 // safe default
	}

	rows, cols := src.Rows(), src.Cols()
	pixelCount := rows * cols

	if pixelCount < 10000 {
		return 32
	} else if pixelCount < 100000 {
		return 64
	} else if pixelCount < 1000000 {
		return 128
	} else {
		return 256
	}
}

func (pe *ProcessingEngine) calculateNeighborhood(src gocv.Mat, windowSize int, neighborhoodType string) gocv.Mat {
	if err := validateMatForMetrics(src, "neighborhood calculation"); err != nil {
		return gocv.NewMat()
	}

	if err := validateImageDimensions(src.Cols(), src.Rows(), "neighborhood calculation"); err != nil {
		return gocv.NewMat()
	}

	switch neighborhoodType {
	case "Circular":
		return pe.calculateCircularNeighborhood(src, windowSize)
	case "Distance Weighted":
		return pe.calculateDistanceWeightedNeighborhood(src, windowSize)
	default:
		return pe.calculateRectangularNeighborhood(src, windowSize)
	}
}

func (pe *ProcessingEngine) calculateRectangularNeighborhood(src gocv.Mat, windowSize int) gocv.Mat {
	result := gocv.NewMatWithSize(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)

	halfWindow := windowSize / 2
	rows, cols := src.Rows(), src.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			sum := 0
			count := 0

			for dy := -halfWindow; dy <= halfWindow; dy++ {
				for dx := -halfWindow; dx <= halfWindow; dx++ {
					ny, nx := y+dy, x+dx
					if ny >= 0 && ny < rows && nx >= 0 && nx < cols {
						sum += int(src.GetUCharAt(ny, nx))
						count++
					}
				}
			}

			if count > 0 {
				result.SetUCharAt(y, x, uint8(sum/count))
			}
		}
	}

	if err := validateMatForMetrics(result, "rectangular neighborhood result"); err != nil {
		result.Close()
		return gocv.NewMat()
	}

	return result
}

func (pe *ProcessingEngine) calculateCircularNeighborhood(src gocv.Mat, windowSize int) gocv.Mat {
	result := gocv.NewMatWithSize(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)

	radius := float64(windowSize / 2)
	rows, cols := src.Rows(), src.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			sum := 0
			count := 0

			for dy := -int(radius); dy <= int(radius); dy++ {
				for dx := -int(radius); dx <= int(radius); dx++ {
					distance := math.Sqrt(float64(dx*dx + dy*dy))
					if distance <= radius {
						ny, nx := y+dy, x+dx
						if ny >= 0 && ny < rows && nx >= 0 && nx < cols {
							sum += int(src.GetUCharAt(ny, nx))
							count++
						}
					}
				}
			}

			if count > 0 {
				result.SetUCharAt(y, x, uint8(sum/count))
			}
		}
	}

	if err := validateMatForMetrics(result, "circular neighborhood result"); err != nil {
		result.Close()
		return gocv.NewMat()
	}

	return result
}

func (pe *ProcessingEngine) calculateDistanceWeightedNeighborhood(src gocv.Mat, windowSize int) gocv.Mat {
	result := gocv.NewMatWithSize(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)

	halfWindow := windowSize / 2
	rows, cols := src.Rows(), src.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			weightedSum := 0.0
			totalWeight := 0.0

			for dy := -halfWindow; dy <= halfWindow; dy++ {
				for dx := -halfWindow; dx <= halfWindow; dx++ {
					ny, nx := y+dy, x+dx
					if ny >= 0 && ny < rows && nx >= 0 && nx < cols {
						distance := math.Sqrt(float64(dx*dx + dy*dy))
						weight := 1.0 / (1.0 + distance)

						value := float64(src.GetUCharAt(ny, nx))
						weightedSum += value * weight
						totalWeight += weight
					}
				}
			}

			if totalWeight > 0 {
				result.SetUCharAt(y, x, uint8(weightedSum/totalWeight))
			}
		}
	}

	if err := validateMatForMetrics(result, "distance weighted neighborhood result"); err != nil {
		result.Close()
		return gocv.NewMat()
	}

	return result
}
