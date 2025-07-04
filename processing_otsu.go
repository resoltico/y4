package main

import (
	"math"

	"gocv.io/x/gocv"
)

func (pe *ProcessingEngine) build2DHistogram(src, neighborhood gocv.Mat, histBins int) [][]float64 {
	histogram := make([][]float64, histBins)
	for i := range histogram {
		histogram[i] = make([]float64, histBins)
	}

	rows := src.Rows()
	cols := src.Cols()
	binScale := float64(histBins-1) / 255.0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue := src.GetUCharAt(y, x)
			neighValue := neighborhood.GetUCharAt(y, x)

			pixelBin := int(float64(pixelValue) * binScale)
			neighBin := int(float64(neighValue) * binScale)

			if pixelBin >= histBins {
				pixelBin = histBins - 1
			}
			if neighBin >= histBins {
				neighBin = histBins - 1
			}

			histogram[pixelBin][neighBin]++
		}
	}

	return histogram
}

func (pe *ProcessingEngine) applyLogScaling(histogram [][]float64) {
	histBins := len(histogram)
	for i := 0; i < histBins; i++ {
		for j := 0; j < histBins; j++ {
			if histogram[i][j] > 0 {
				histogram[i][j] = math.Log1p(histogram[i][j])
			}
		}
	}
}

func (pe *ProcessingEngine) normalizeHistogram(histogram [][]float64) {
	histBins := len(histogram)
	total := 0.0

	for i := 0; i < histBins; i++ {
		for j := 0; j < histBins; j++ {
			total += histogram[i][j]
		}
	}

	if total > 0 {
		invTotal := 1.0 / total
		for i := 0; i < histBins; i++ {
			for j := 0; j < histBins; j++ {
				histogram[i][j] *= invTotal
			}
		}
	}
}

func (pe *ProcessingEngine) smoothHistogram(histogram [][]float64, sigma float64) {
	histBins := len(histogram)
	kernelRadius := int(sigma * 3)
	kernelSize := kernelRadius*2 + 1

	kernel := make([][]float64, kernelSize)
	for i := range kernel {
		kernel[i] = make([]float64, kernelSize)
	}

	sum := 0.0
	invSigmaSq := 1.0 / (2.0 * sigma * sigma)

	for i := 0; i < kernelSize; i++ {
		for j := 0; j < kernelSize; j++ {
			x := float64(i - kernelRadius)
			y := float64(j - kernelRadius)
			value := math.Exp(-(x*x + y*y) * invSigmaSq)
			kernel[i][j] = value
			sum += value
		}
	}

	for i := 0; i < kernelSize; i++ {
		for j := 0; j < kernelSize; j++ {
			kernel[i][j] /= sum
		}
	}

	smoothed := make([][]float64, histBins)
	for i := range smoothed {
		smoothed[i] = make([]float64, histBins)
	}

	for i := 0; i < histBins; i++ {
		for j := 0; j < histBins; j++ {
			value := 0.0

			for ki := 0; ki < kernelSize; ki++ {
				for kj := 0; kj < kernelSize; kj++ {
					hi := i + ki - kernelRadius
					hj := j + kj - kernelRadius

					if hi >= 0 && hi < histBins && hj >= 0 && hj < histBins {
						value += histogram[hi][hj] * kernel[ki][kj]
					}
				}
			}

			smoothed[i][j] = value
		}
	}

	for i := 0; i < histBins; i++ {
		copy(histogram[i], smoothed[i])
	}
}

func (pe *ProcessingEngine) find2DOtsuThresholdInteger(histogram [][]float64) [2]int {
	histBins := len(histogram)
	bestThreshold := [2]int{histBins / 2, histBins / 2}
	maxVariance := 0.0

	totalSum := 0.0
	totalCount := 0.0
	for i := 0; i < histBins; i++ {
		for j := 0; j < histBins; j++ {
			weight := histogram[i][j]
			totalSum += float64(i*histBins+j) * weight
			totalCount += weight
		}
	}

	if totalCount == 0 {
		return bestThreshold
	}

	for t1 := 1; t1 < histBins-1; t1++ {
		for t2 := 1; t2 < histBins-1; t2++ {
			variance := pe.calculateVarianceForIntegerThresholds(histogram, t1, t2, totalSum, totalCount)
			if variance > maxVariance {
				maxVariance = variance
				bestThreshold = [2]int{t1, t2}
			}
		}
	}

	return bestThreshold
}

func (pe *ProcessingEngine) calculateVarianceForIntegerThresholds(histogram [][]float64, t1, t2 int, totalSum, totalCount float64) float64 {
	histBins := len(histogram)
	var w0, w1, sum0, sum1 float64

	for i := 0; i <= t1; i++ {
		for j := 0; j <= t2; j++ {
			weight := histogram[i][j]
			w0 += weight
			sum0 += float64(i*histBins+j) * weight
		}
	}

	for i := t1 + 1; i < histBins; i++ {
		for j := t2 + 1; j < histBins; j++ {
			weight := histogram[i][j]
			w1 += weight
			sum1 += float64(i*histBins+j) * weight
		}
	}

	if w0 > 0 && w1 > 0 {
		mean0 := sum0 / w0
		mean1 := sum1 / w1
		meanDiff := mean0 - mean1
		return w0 * w1 * meanDiff * meanDiff
	}

	return 0.0
}

func (pe *ProcessingEngine) applyThreshold(src, neighborhood gocv.Mat, threshold [2]int, histBins int) gocv.Mat {
	result := gocv.NewMatWithSize(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)
	binScale := float64(histBins-1) / 255.0

	for y := 0; y < src.Rows(); y++ {
		for x := 0; x < src.Cols(); x++ {
			pixelValue := src.GetUCharAt(y, x)
			neighValue := neighborhood.GetUCharAt(y, x)

			pixelBin := int(float64(pixelValue) * binScale)
			neighBin := int(float64(neighValue) * binScale)

			if pixelBin > threshold[0] && neighBin > threshold[1] {
				result.SetUCharAt(y, x, 255)
			} else {
				result.SetUCharAt(y, x, 0)
			}
		}
	}

	return result
}
