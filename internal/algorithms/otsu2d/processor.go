package otsu2d

import (
	"context"
	"fmt"
	"image"

	"otsu-obliterator/internal/opencv/conversion"
	"otsu-obliterator/internal/opencv/safe"

	"gocv.io/x/gocv"
)

type Processor struct {
	name string
}

func NewProcessor() *Processor {
	return &Processor{
		name: "2D Otsu",
	}
}

func (p *Processor) GetName() string {
	return p.name
}

func (p *Processor) GetDefaultParameters() map[string]interface{} {
	return map[string]interface{}{
		"window_size":                7,
		"histogram_bins":             0, // auto-calculated
		"smoothing_strength":         1.0,
		"edge_preservation":          false,
		"noise_robustness":           false,
		"gaussian_preprocessing":     true,
		"use_log_histogram":          false,
		"normalize_histogram":        true,
		"apply_contrast_enhancement": false,
	}
}

func (p *Processor) ValidateParameters(params map[string]interface{}) error {
	if windowSize, ok := params["window_size"].(int); ok {
		if windowSize < 3 || windowSize > 21 || windowSize%2 == 0 {
			return fmt.Errorf("window_size must be odd number between 3 and 21, got: %d", windowSize)
		}
	}

	if histBins, ok := params["histogram_bins"].(int); ok {
		if histBins != 0 && (histBins < 16 || histBins > 256) {
			return fmt.Errorf("histogram_bins must be 0 (auto) or between 16 and 256, got: %d", histBins)
		}
	}

	if smoothing, ok := params["smoothing_strength"].(float64); ok {
		if smoothing < 0.0 || smoothing > 5.0 {
			return fmt.Errorf("smoothing_strength must be between 0.0 and 5.0, got: %f", smoothing)
		}
	}

	return nil
}

func (p *Processor) Process(input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	return p.ProcessWithContext(context.Background(), input, params)
}

func (p *Processor) ProcessWithContext(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	if err := safe.ValidateMatForOperation(input, "2D Otsu processing"); err != nil {
		return nil, err
	}

	if err := p.ValidateParameters(params); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	gray, err := conversion.ConvertToGrayscale(input)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to grayscale: %w", err)
	}
	defer gray.Close()

	working := gray
	var processed *safe.Mat

	// Apply MAOTSU preprocessing when edge preservation is enabled
	if p.getBoolParam(params, "edge_preservation") {
		processed, err = p.applyMAOTSUPreprocessing(gray)
		if err != nil {
			return nil, fmt.Errorf("MAOTSU preprocessing failed: %w", err)
		}
		working = processed
		defer processed.Close()
	}

	// Gaussian preprocessing
	var blurred *safe.Mat
	if p.getBoolParam(params, "gaussian_preprocessing") {
		blurred, err = p.applyGaussianBlur(working, p.getFloatParam(params, "smoothing_strength"))
		if err != nil {
			return nil, fmt.Errorf("gaussian preprocessing failed: %w", err)
		}
		working = blurred
		defer blurred.Close()
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Contrast enhancement if requested
	var enhanced *safe.Mat
	if p.getBoolParam(params, "apply_contrast_enhancement") {
		enhanced, err = p.applyCLAHE(working)
		if err != nil {
			return nil, fmt.Errorf("contrast enhancement failed: %w", err)
		}
		working = enhanced
		defer enhanced.Close()
	}

	// Calculate neighborhood using summed area table for performance
	neighborhood, err := p.calculateNeighborhoodMeanFast(working, params)
	if err != nil {
		return nil, fmt.Errorf("neighborhood calculation failed: %w", err)
	}
	defer neighborhood.Close()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Calculate histogram bins automatically if set to 0
	histBins := p.getIntParam(params, "histogram_bins")
	if histBins == 0 {
		histBins = p.calculateOptimalBinCount(working)
	}

	// Build 2D histogram using summed area table
	histogram := p.build2DHistogramWithSummedTable(working, neighborhood, histBins, params)

	// Apply histogram processing
	if p.getBoolParam(params, "use_log_histogram") {
		p.applyLogScaling(histogram)
	}

	if p.getBoolParam(params, "normalize_histogram") {
		p.normalizeHistogram(histogram)
	}

	if p.getFloatParam(params, "smoothing_strength") > 0 {
		p.smoothHistogram(histogram, p.getFloatParam(params, "smoothing_strength"))
	}

	// Find threshold using diagonal projection optimization
	threshold := p.find2DOtsuThresholdWithDiagonalProjection(histogram)

	result, err := safe.NewMat(working.Rows(), working.Cols(), gocv.MatTypeCV8UC1)
	if err != nil {
		return nil, fmt.Errorf("failed to create result Mat: %w", err)
	}

	if err := p.applyThresholdWithSubpixelPrecision(working, neighborhood, result, threshold, histBins, params); err != nil {
		result.Close()
		return nil, fmt.Errorf("threshold application failed: %w", err)
	}

	// Apply noise robustness post-processing if enabled
	if p.getBoolParam(params, "noise_robustness") {
		cleaned, err := p.applyNoiseRobustnessFilter(result)
		if err != nil {
			result.Close()
			return nil, fmt.Errorf("noise robustness filtering failed: %w", err)
		}
		result.Close()
		result = cleaned
	}

	return result, nil
}

// applyMAOTSUPreprocessing combines median and average filtering for edge preservation
func (p *Processor) applyMAOTSUPreprocessing(src *safe.Mat) (*safe.Mat, error) {
	// Apply median filter for noise reduction
	median, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}
	defer median.Close()

	srcMat := src.GetMat()
	medianMat := median.GetMat()
	gocv.MedianBlur(srcMat, &medianMat, 5)

	// Apply Gaussian filter for smoothing
	gaussian, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}
	defer gaussian.Close()

	gaussianMat := gaussian.GetMat()
	gocv.GaussianBlur(medianMat, &gaussianMat, image.Point{X: 5, Y: 5}, 1.0, 1.0, gocv.BorderDefault)

	// Combine median and gaussian results with weighted average
	result, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	rows := src.Rows()
	cols := src.Cols()
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			medianVal, _ := median.GetUCharAt(y, x)
			gaussianVal, _ := gaussian.GetUCharAt(y, x)

			// Weight 0.6 for median (noise reduction) and 0.4 for gaussian (smoothing)
			combined := uint8(float64(medianVal)*0.6 + float64(gaussianVal)*0.4)
			result.SetUCharAt(y, x, combined)
		}
	}

	return result, nil
}

// calculateNeighborhoodMeanFast uses summed area table for O(1) neighborhood calculations
func (p *Processor) calculateNeighborhoodMeanFast(src *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	windowSize := p.getIntParam(params, "window_size")

	dst, err := safe.NewMat(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)
	if err != nil {
		return nil, err
	}

	rows := src.Rows()
	cols := src.Cols()
	halfWindow := windowSize / 2

	// Build summed area table
	sumTable := make([][]int64, rows+1)
	for i := range sumTable {
		sumTable[i] = make([]int64, cols+1)
	}

	for y := 1; y <= rows; y++ {
		for x := 1; x <= cols; x++ {
			pixelVal, _ := src.GetUCharAt(y-1, x-1)
			sumTable[y][x] = int64(pixelVal) + sumTable[y-1][x] + sumTable[y][x-1] - sumTable[y-1][x-1]
		}
	}

	// Calculate neighborhood means using summed area table
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			y1 := max(0, y-halfWindow)
			x1 := max(0, x-halfWindow)
			y2 := min(rows-1, y+halfWindow)
			x2 := min(cols-1, x+halfWindow)

			area := int64((y2 - y1 + 1) * (x2 - x1 + 1))
			sum := sumTable[y2+1][x2+1] - sumTable[y1][x2+1] - sumTable[y2+1][x1] + sumTable[y1][x1]

			mean := uint8(sum / area)
			dst.SetUCharAt(y, x, mean)
		}
	}

	return dst, nil
}

// calculateOptimalBinCount determines histogram bins based on image characteristics
func (p *Processor) calculateOptimalBinCount(src *safe.Mat) int {
	rows := src.Rows()
	cols := src.Cols()
	totalPixels := rows * cols

	// Calculate image dynamic range
	var minVal, maxVal uint8 = 255, 0
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			val, _ := src.GetUCharAt(y, x)
			if val < minVal {
				minVal = val
			}
			if val > maxVal {
				maxVal = val
			}
		}
	}

	dynamicRange := int(maxVal - minVal)

	// Adaptive bin calculation based on dynamic range and image size
	baseBins := 64
	if dynamicRange < 50 {
		baseBins = 32
	} else if dynamicRange > 200 {
		baseBins = 128
	}

	// Adjust for image size
	if totalPixels > 1000000 { // Large images
		baseBins = min(baseBins*2, 256)
	} else if totalPixels < 100000 { // Small images
		baseBins = max(baseBins/2, 16)
	}

	return baseBins
}

// build2DHistogramWithSummedTable builds histogram using summed area table optimization
func (p *Processor) build2DHistogramWithSummedTable(src, neighborhood *safe.Mat, histBins int, params map[string]interface{}) [][]float64 {
	histogram := make([][]float64, histBins)
	for i := range histogram {
		histogram[i] = make([]float64, histBins)
	}

	rows := src.Rows()
	cols := src.Cols()
	binScale := float64(histBins-1) / 255.0

	// Direct histogram accumulation with bounds checking
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue, err := src.GetUCharAt(y, x)
			if err != nil {
				continue
			}

			neighValue, err := neighborhood.GetUCharAt(y, x)
			if err != nil {
				continue
			}

			pixelBin := int(float64(pixelValue) * binScale)
			neighBin := int(float64(neighValue) * binScale)

			// Clamp to valid range
			pixelBin = max(0, min(pixelBin, histBins-1))
			neighBin = max(0, min(neighBin, histBins-1))

			histogram[pixelBin][neighBin]++
		}
	}

	return histogram
}

// find2DOtsuThresholdWithDiagonalProjection uses diagonal projection to reduce search complexity
func (p *Processor) find2DOtsuThresholdWithDiagonalProjection(histogram [][]float64) [2]float64 {
	histBins := len(histogram)
	bestThreshold := [2]float64{float64(histBins) / 2.0, float64(histBins) / 2.0}
	maxVariance := 0.0

	// Calculate total sum and count for the histogram
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

	// Diagonal projection optimization: search along main patterns
	subPixelStep := 0.1

	// Main diagonal search
	for t := 1.0; t < float64(histBins-1); t += subPixelStep {
		variance := p.calculateVarianceForThresholds(histogram, t, t, totalSum, totalCount)
		if variance > maxVariance {
			maxVariance = variance
			bestThreshold = [2]float64{t, t}
		}
	}

	// Off-diagonal search around best diagonal point
	searchRadius := 5.0
	centerT1, centerT2 := bestThreshold[0], bestThreshold[1]

	for t1 := maxFloat(1.0, centerT1-searchRadius); t1 < minFloat(float64(histBins-1), centerT1+searchRadius); t1 += subPixelStep {
		for t2 := maxFloat(1.0, centerT2-searchRadius); t2 < minFloat(float64(histBins-1), centerT2+searchRadius); t2 += subPixelStep {
			variance := p.calculateVarianceForThresholds(histogram, t1, t2, totalSum, totalCount)
			if variance > maxVariance {
				maxVariance = variance
				bestThreshold = [2]float64{t1, t2}
			}
		}
	}

	return bestThreshold
}

// calculateVarianceForThresholds computes between-class variance for given thresholds
func (p *Processor) calculateVarianceForThresholds(histogram [][]float64, t1, t2, totalSum, totalCount float64) float64 {
	histBins := len(histogram)
	var w0, w1, sum0, sum1 float64

	t1Int := int(t1)
	t2Int := int(t2)

	// Calculate class 0 (background) statistics
	for i := 0; i <= t1Int; i++ {
		for j := 0; j <= t2Int; j++ {
			if float64(i) <= t1 && float64(j) <= t2 {
				weight := histogram[i][j]
				interpolationFactor := 1.0

				if i == t1Int && float64(i) > t1 {
					interpolationFactor *= (1.0 - (t1 - float64(t1Int)))
				}
				if j == t2Int && float64(j) > t2 {
					interpolationFactor *= (1.0 - (t2 - float64(t2Int)))
				}

				weightInterpolated := weight * interpolationFactor
				w0 += weightInterpolated
				sum0 += float64(i*histBins+j) * weightInterpolated
			}
		}
	}

	// Calculate class 1 (foreground) statistics
	for i := t1Int + 1; i < histBins; i++ {
		for j := t2Int + 1; j < histBins; j++ {
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

// applyThresholdWithSubpixelPrecision applies threshold with sub-pixel interpolation
func (p *Processor) applyThresholdWithSubpixelPrecision(src, neighborhood, dst *safe.Mat, threshold [2]float64, histBins int, params map[string]interface{}) error {
	rows := src.Rows()
	cols := src.Cols()
	binScale := float64(histBins-1) / 255.0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue, err := src.GetUCharAt(y, x)
			if err != nil {
				return fmt.Errorf("failed to get pixel at (%d,%d): %w", x, y, err)
			}

			neighValue, err := neighborhood.GetUCharAt(y, x)
			if err != nil {
				return fmt.Errorf("failed to get neighborhood pixel at (%d,%d): %w", x, y, err)
			}

			pixelBin := float64(pixelValue) * binScale
			neighBin := float64(neighValue) * binScale

			var value uint8
			if pixelBin > threshold[0] && neighBin > threshold[1] {
				value = 255
			} else {
				value = 0
			}

			if err := dst.SetUCharAt(y, x, value); err != nil {
				return fmt.Errorf("failed to set pixel at (%d,%d): %w", x, y, err)
			}
		}
	}

	return nil
}

// applyNoiseRobustnessFilter applies morphological operations for noise reduction
func (p *Processor) applyNoiseRobustnessFilter(src *safe.Mat) (*safe.Mat, error) {
	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 3, Y: 3})
	defer kernel.Close()

	// Opening operation to remove small noise
	opened, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	srcMat := src.GetMat()
	openedMat := opened.GetMat()
	gocv.MorphologyEx(srcMat, &openedMat, gocv.MorphOpen, kernel)

	// Closing operation to fill small gaps
	result, err := safe.NewMat(opened.Rows(), opened.Cols(), opened.Type())
	if err != nil {
		opened.Close()
		return nil, err
	}

	resultMat := result.GetMat()
	gocv.MorphologyEx(openedMat, &resultMat, gocv.MorphClose, kernel)

	opened.Close()
	return result, nil
}

func (p *Processor) applyGaussianBlur(src *safe.Mat, sigma float64) (*safe.Mat, error) {
	dst, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	srcMat := src.GetMat()
	dstMat := dst.GetMat()

	kernelSize := int(sigma*6) + 1
	if kernelSize%2 == 0 {
		kernelSize++
	}
	kernelSize = max(3, min(kernelSize, 15))

	gocv.GaussianBlur(srcMat, &dstMat, image.Point{X: kernelSize, Y: kernelSize}, sigma, sigma, gocv.BorderDefault)

	return dst, nil
}

func (p *Processor) applyCLAHE(src *safe.Mat) (*safe.Mat, error) {
	dst, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to create destination Mat: %w", err)
	}

	clahe := gocv.NewCLAHE()
	defer clahe.Close()

	srcMat := src.GetMat()
	dstMat := dst.GetMat()

	clahe.Apply(srcMat, &dstMat)

	return dst, nil
}

func (p *Processor) getBoolParam(params map[string]interface{}, key string) bool {
	if value, ok := params[key].(bool); ok {
		return value
	}
	return false
}

func (p *Processor) getIntParam(params map[string]interface{}, key string) int {
	if value, ok := params[key].(int); ok {
		return value
	}
	return 0
}

func (p *Processor) getFloatParam(params map[string]interface{}, key string) float64 {
	if value, ok := params[key].(float64); ok {
		return value
	}
	return 0.0
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
