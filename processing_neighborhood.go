package main

import (
	"image"
	"math"

	"gocv.io/x/gocv"
)

func (pe *ProcessingEngine) calculateAdaptiveWindowSize(src gocv.Mat) int {
	rows, cols := src.Rows(), src.Cols()

	var intensity float64
	totalPixels := rows * cols

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			intensity += float64(src.GetUCharAt(y, x))
		}
	}

	meanIntensity := intensity / float64(totalPixels)

	var variance float64
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			diff := float64(src.GetUCharAt(y, x)) - meanIntensity
			variance += diff * diff
		}
	}
	variance /= float64(totalPixels)

	baseWindow := 7
	varianceScale := variance / 1000.0
	adaptiveWindow := int(float64(baseWindow) * (1.0 + varianceScale))

	if adaptiveWindow%2 == 0 {
		adaptiveWindow++
	}

	return max(3, min(adaptiveWindow, 21))
}

func (pe *ProcessingEngine) calculateNeighborhood(src gocv.Mat, windowSize int, neighborhoodType string) gocv.Mat {
	switch neighborhoodType {
	case "circular":
		return pe.calculateCircularNeighborhood(src, windowSize)
	case "distance_weighted":
		return pe.calculateDistanceWeightedNeighborhood(src, windowSize)
	default:
		return pe.calculateRectangularNeighborhood(src, windowSize)
	}
}

func (pe *ProcessingEngine) calculateRectangularNeighborhood(src gocv.Mat, windowSize int) gocv.Mat {
	dst := gocv.NewMat()
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Point{X: windowSize, Y: windowSize})
	defer kernel.Close()
	gocv.MorphologyEx(src, &dst, gocv.MorphOpen, kernel)
	return dst
}

func (pe *ProcessingEngine) calculateCircularNeighborhood(src gocv.Mat, windowSize int) gocv.Mat {
	dst := gocv.NewMat()
	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: windowSize, Y: windowSize})
	defer kernel.Close()
	gocv.MorphologyEx(src, &dst, gocv.MorphOpen, kernel)
	return dst
}

func (pe *ProcessingEngine) calculateDistanceWeightedNeighborhood(src gocv.Mat, windowSize int) gocv.Mat {
	rows, cols := src.Rows(), src.Cols()
	dst := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)

	radius := windowSize / 2

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			var weightedSum, totalWeight float64

			for dy := -radius; dy <= radius; dy++ {
				for dx := -radius; dx <= radius; dx++ {
					ny, nx := y+dy, x+dx
					if ny >= 0 && ny < rows && nx >= 0 && nx < cols {
						distance := math.Sqrt(float64(dx*dx + dy*dy))
						if distance <= float64(radius) {
							weight := 1.0 / (1.0 + distance)
							pixel := float64(src.GetUCharAt(ny, nx))
							weightedSum += pixel * weight
							totalWeight += weight
						}
					}
				}
			}

			if totalWeight > 0 {
				dst.SetUCharAt(y, x, uint8(weightedSum/totalWeight))
			} else {
				dst.SetUCharAt(y, x, src.GetUCharAt(y, x))
			}
		}
	}

	return dst
}

func (pe *ProcessingEngine) calculateHistogramBins(src gocv.Mat) int {
	rows := src.Rows()
	cols := src.Cols()
	totalPixels := rows * cols

	if totalPixels > 1000000 {
		return 128
	} else if totalPixels < 100000 {
		return 32
	}
	return 64
}
