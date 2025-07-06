package main

import (
	"fmt"
	"math"

	"gocv.io/x/gocv"
)

func (pe *ProcessingEngine) validateRegionContrast(src gocv.Mat) (bool, float64, error) {
	if err := validateMatForMetrics(src, "contrast validation"); err != nil {
		return false, 0, err
	}

	minVal, maxVal, _, _ := gocv.MinMaxLoc(src)
	contrast := maxVal - minVal

	if contrast < 5.0 {
		return false, contrast, fmt.Errorf("insufficient contrast: %.2f (minimum 5.0)", contrast)
	}
	return true, contrast, nil
}

func (pe *ProcessingEngine) build2DHistogram(src, neighborhood gocv.Mat, histBins int) [][]float64 {
	if err := validateMatForMetrics(src, "2D histogram source"); err != nil {
		return make([][]float64, histBins)
	}

	if err := validateMatForMetrics(neighborhood, "2D histogram neighborhood"); err != nil {
		return make([][]float64, histBins)
	}

	if err := validateMatDimensionsMatch(src, neighborhood, "2D histogram"); err != nil {
		return make([][]float64, histBins)
	}

	// Validate contrast before processing
	hasContrast, contrast, err := pe.validateRegionContrast(src)
	if !hasContrast {
		debugSystem := GetDebugSystem()
		debugSystem.logger.Warn("skipping region due to insufficient contrast",
			"contrast", contrast,
			"error", err.Error())
		return make([][]float64, histBins) // Return empty histogram
	}

	histogram := make([][]float64, histBins)
	for i := range histogram {
		histogram[i] = make([]float64, histBins)
	}

	rows := src.Rows()
	cols := src.Cols()
	binScale := float64(histBins-1) / 255.0

	// Debug: Check input data ranges
	debugSystem := GetDebugSystem()
	srcMin, srcMax, _, _ := gocv.MinMaxLoc(src)
	neighMin, neighMax, _, _ := gocv.MinMaxLoc(neighborhood)

	debugSystem.logger.Debug("histogram input analysis",
		"src_min", srcMin, "src_max", srcMax,
		"neigh_min", neighMin, "neigh_max", neighMax,
		"hist_bins", histBins, "bin_scale", binScale)

	totalPixels := 0
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
			totalPixels++
		}
	}

	// Debug: Analyze histogram distribution
	nonZeroBins := 0
	maxBinValue := 0.0
	for i := 0; i < histBins; i++ {
		for j := 0; j < histBins; j++ {
			if histogram[i][j] > 0 {
				nonZeroBins++
				if histogram[i][j] > maxBinValue {
					maxBinValue = histogram[i][j]
				}
			}
		}
	}

	debugSystem.logger.Debug("histogram distribution analysis",
		"total_pixels", totalPixels,
		"non_zero_bins", nonZeroBins,
		"max_bin_value", maxBinValue,
		"bins_ratio", float64(nonZeroBins)/float64(histBins*histBins))

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

	debugSystem := GetDebugSystem()

	if totalCount == 0 {
		debugSystem.logger.Error("histogram empty - no pixel data",
			"histogram_bins", histBins)
		return bestThreshold
	}

	// Test thresholds and track variance quality
	varianceData := make([]float64, 0, (histBins-2)*(histBins-2))

	for t1 := 1; t1 < histBins-1; t1++ {
		for t2 := 1; t2 < histBins-1; t2++ {
			variance := pe.calculateVarianceForIntegerThresholds(histogram, t1, t2, totalSum, totalCount)
			varianceData = append(varianceData, variance)

			if variance > maxVariance {
				maxVariance = variance
				bestThreshold = [2]int{t1, t2}
			}
		}
	}

	// Calculate variance statistics for quality assessment
	avgVariance := 0.0
	for _, v := range varianceData {
		avgVariance += v
	}
	avgVariance /= float64(len(varianceData))

	// Quality check - detect poor separation
	varianceRatio := maxVariance / avgVariance

	debugSystem.logger.Debug("Otsu threshold analysis",
		"threshold_t1", bestThreshold[0],
		"threshold_t2", bestThreshold[1],
		"max_variance", maxVariance,
		"avg_variance", avgVariance,
		"variance_ratio", varianceRatio,
		"histogram_bins", histBins,
		"total_count", totalCount)

	if varianceRatio < 1.5 {
		debugSystem.logger.Warn("poor foreground/background separation detected",
			"max_variance", maxVariance,
			"avg_variance", avgVariance,
			"variance_ratio", varianceRatio,
			"threshold_t1", bestThreshold[0],
			"threshold_t2", bestThreshold[1])
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
	if err := validateMatForMetrics(src, "threshold application source"); err != nil {
		return gocv.NewMat()
	}

	if err := validateMatForMetrics(neighborhood, "threshold application neighborhood"); err != nil {
		return gocv.NewMat()
	}

	if err := validateMatDimensionsMatch(src, neighborhood, "threshold application"); err != nil {
		return gocv.NewMat()
	}

	result := gocv.NewMatWithSize(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)
	binScale := float64(histBins-1) / 255.0

	foregroundPixels := 0
	backgroundPixels := 0

	for y := 0; y < src.Rows(); y++ {
		for x := 0; x < src.Cols(); x++ {
			pixelValue := src.GetUCharAt(y, x)
			neighValue := neighborhood.GetUCharAt(y, x)

			pixelBin := int(float64(pixelValue) * binScale)
			neighBin := int(float64(neighValue) * binScale)

			if pixelBin > threshold[0] && neighBin > threshold[1] {
				result.SetUCharAt(y, x, 255)
				foregroundPixels++
			} else {
				result.SetUCharAt(y, x, 0)
				backgroundPixels++
			}
		}
	}

	debugSystem := GetDebugSystem()
	totalPixels := foregroundPixels + backgroundPixels
	foregroundRatio := float64(foregroundPixels) / float64(totalPixels)

	debugSystem.logger.Debug("threshold application results",
		"threshold_t1", threshold[0],
		"threshold_t2", threshold[1],
		"foreground_pixels", foregroundPixels,
		"background_pixels", backgroundPixels,
		"foreground_ratio", foregroundRatio,
		"bin_scale", binScale)

	// Enhanced diagnostic logging for problematic results
	if foregroundPixels == 0 {
		minVal, maxVal, _, _ := gocv.MinMaxLoc(src)
		debugSystem.logger.Error("threshold produced all-background image",
			"threshold_t1", threshold[0],
			"threshold_t2", threshold[1],
			"hist_bins", histBins,
			"src_contrast", maxVal-minVal,
			"src_min", minVal,
			"src_max", maxVal)
	} else if backgroundPixels == 0 {
		debugSystem.logger.Error("threshold produced all-foreground image",
			"threshold_t1", threshold[0],
			"threshold_t2", threshold[1],
			"hist_bins", histBins)
	}

	if err := validateMatForMetrics(result, "threshold application result"); err != nil {
		if result.Empty() {
			result.Close()
			return gocv.NewMat()
		}
	}

	return result
}
