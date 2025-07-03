package otsu2d

import (
	"math"
)

func (p *Processor) smoothHistogram(histogram [][]float64, sigma float64) {
	if sigma <= 0.0 {
		return
	}

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

	// Normalize kernel
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

	// Copy back
	for i := 0; i < histBins; i++ {
		copy(histogram[i], smoothed[i])
	}
}

func (p *Processor) applyLogScaling(histogram [][]float64) {
	histBins := len(histogram)

	for i := 0; i < histBins; i++ {
		for j := 0; j < histBins; j++ {
			if histogram[i][j] > 0 {
				histogram[i][j] = math.Log1p(histogram[i][j])
			}
		}
	}
}

func (p *Processor) normalizeHistogram(histogram [][]float64) {
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
