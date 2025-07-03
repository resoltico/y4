package triclass

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
		name: "Iterative Triclass",
	}
}

func (p *Processor) GetName() string {
	return p.name
}

func (p *Processor) GetDefaultParameters() map[string]interface{} {
	return map[string]interface{}{
		"initial_threshold_method": "otsu",
		"histogram_bins":           0, // auto-calculated
		"convergence_precision":    1.0,
		"max_iterations":           10,
		"minimum_tbd_fraction":     0.01,
		"class_separation":         0.5,
		"preprocessing":            false,
		"result_cleanup":           true,
		"preserve_borders":         false,
	}
}

func (p *Processor) ValidateParameters(params map[string]interface{}) error {
	if method, ok := params["initial_threshold_method"].(string); ok {
		if method != "otsu" && method != "mean" && method != "median" {
			return fmt.Errorf("initial_threshold_method must be 'otsu', 'mean', or 'median', got: %s", method)
		}
	}

	if histBins, ok := params["histogram_bins"].(int); ok {
		if histBins != 0 && (histBins < 16 || histBins > 256) {
			return fmt.Errorf("histogram_bins must be 0 (auto) or between 16 and 256, got: %d", histBins)
		}
	}

	if precision, ok := params["convergence_precision"].(float64); ok {
		if precision < 0.1 || precision > 10.0 {
			return fmt.Errorf("convergence_precision must be between 0.1 and 10.0, got: %f", precision)
		}
	}

	if maxIter, ok := params["max_iterations"].(int); ok {
		if maxIter < 5 || maxIter > 15 {
			return fmt.Errorf("max_iterations must be between 5 and 15, got: %d", maxIter)
		}
	}

	if fraction, ok := params["minimum_tbd_fraction"].(float64); ok {
		if fraction < 0.001 || fraction > 0.2 {
			return fmt.Errorf("minimum_tbd_fraction must be between 0.001 and 0.2, got: %f", fraction)
		}
	}

	if separation, ok := params["class_separation"].(float64); ok {
		if separation < 0.1 || separation > 0.8 {
			return fmt.Errorf("class_separation must be between 0.1 and 0.8, got: %f", separation)
		}
	}

	return nil
}

func (p *Processor) Process(input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	return p.ProcessWithContext(context.Background(), input, params)
}

func (p *Processor) ProcessWithContext(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	if err := safe.ValidateMatForOperation(input, "Iterative Triclass processing"); err != nil {
		return nil, err
	}

	if err := p.ValidateParameters(params); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	gray, err := conversion.ConvertToGrayscale(input)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to grayscale: %w", err)
	}
	defer gray.Close()

	working := gray
	if p.getBoolParam(params, "preprocessing") {
		preprocessed, err := p.applyAdvancedPreprocessing(gray)
		if err != nil {
			return nil, fmt.Errorf("preprocessing failed: %w", err)
		}
		working = preprocessed
		defer preprocessed.Close()
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	result, err := p.performIterativeTriclassWithIntelligentConvergence(ctx, working, params)
	if err != nil {
		return nil, fmt.Errorf("iterative processing failed: %w", err)
	}

	if p.getBoolParam(params, "result_cleanup") {
		cleaned, err := p.applyMorphologicalCleanup(result)
		if err != nil {
			result.Close()
			return nil, fmt.Errorf("cleanup failed: %w", err)
		}
		result.Close()
		result = cleaned
	}

	return result, nil
}

// performIterativeTriclassWithIntelligentConvergence implements iterative triclass with intelligent stopping
func (p *Processor) performIterativeTriclassWithIntelligentConvergence(ctx context.Context, working *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	maxIterations := p.getIntParam(params, "max_iterations")
	convergencePrecision := p.getFloatParam(params, "convergence_precision")
	minTBDFraction := p.getFloatParam(params, "minimum_tbd_fraction")

	result, err := safe.NewMat(working.Rows(), working.Cols(), working.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to create result Mat: %w", err)
	}

	currentRegion, err := working.Clone()
	if err != nil {
		result.Close()
		return nil, fmt.Errorf("failed to clone working Mat: %w", err)
	}
	defer currentRegion.Close()

	previousThreshold := -1.0
	totalPixels := float64(currentRegion.Rows() * currentRegion.Cols())
	convergenceHistory := make([]float64, 0, maxIterations)

	for iteration := 0; iteration < maxIterations; iteration++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		nonZeroPixels := p.countNonZeroPixels(currentRegion)
		if nonZeroPixels == 0 {
			break
		}

		threshold := p.calculateThresholdForRegionWithAutoMethod(currentRegion, params)

		// Intelligent convergence detection
		convergence := abs(threshold - previousThreshold)
		convergenceHistory = append(convergenceHistory, convergence)

		// Check for convergence stability over multiple iterations
		if len(convergenceHistory) >= 3 && p.isConverged(convergenceHistory, convergencePrecision) {
			break
		}

		previousThreshold = threshold

		foregroundMask, backgroundMask, tbdMask, err := p.segmentRegionWithAdaptiveGaps(currentRegion, threshold, params)
		if err != nil {
			return nil, fmt.Errorf("segmentation failed at iteration %d: %w", iteration, err)
		}

		tbdCount := p.countNonZeroPixels(tbdMask)
		tbdFraction := float64(tbdCount) / totalPixels

		p.updateResult(result, foregroundMask)

		foregroundMask.Close()
		backgroundMask.Close()

		if tbdFraction < minTBDFraction {
			tbdMask.Close()
			break
		}

		newRegion, err := p.extractTBDRegion(working, tbdMask)
		tbdMask.Close()
		if err != nil {
			return nil, fmt.Errorf("TBD region extraction failed: %w", err)
		}

		currentRegion.Close()
		currentRegion = newRegion
	}

	return result, nil
}

// isConverged checks if convergence is stable over multiple iterations
func (p *Processor) isConverged(history []float64, precision float64) bool {
	if len(history) < 3 {
		return false
	}

	// Check last 3 convergence values are all below precision
	for i := len(history) - 3; i < len(history); i++ {
		if history[i] > precision {
			return false
		}
	}

	return true
}

// applyAdvancedPreprocessing applies guided filtering and noise reduction
func (p *Processor) applyAdvancedPreprocessing(src *safe.Mat) (*safe.Mat, error) {
	// Apply guided filtering for edge-preserving smoothing
	guided, err := p.applyGuidedFilter(src, 8, 0.2)
	if err != nil {
		return nil, err
	}
	defer guided.Close()

	// Apply non-local means denoising
	denoised, err := p.applyNonLocalMeansDenoising(guided)
	if err != nil {
		return nil, err
	}

	return denoised, nil
}

// applyGuidedFilter implements guided filtering for edge preservation
func (p *Processor) applyGuidedFilter(src *safe.Mat, radius int, epsilon float64) (*safe.Mat, error) {
	// Simplified guided filter implementation
	// In production, this would use more sophisticated guided filtering
	result, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	rows := src.Rows()
	cols := src.Cols()

	// Build integral image for fast box filtering
	integral := make([][]int64, rows+1)
	for i := range integral {
		integral[i] = make([]int64, cols+1)
	}

	for y := 1; y <= rows; y++ {
		for x := 1; x <= cols; x++ {
			val, _ := src.GetUCharAt(y-1, x-1)
			integral[y][x] = int64(val) + integral[y-1][x] + integral[y][x-1] - integral[y-1][x-1]
		}
	}

	// Apply guided filtering with box filter approximation
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			y1 := max(0, y-radius)
			x1 := max(0, x-radius)
			y2 := min(rows-1, y+radius)
			x2 := min(cols-1, x+radius)

			area := int64((y2 - y1 + 1) * (x2 - x1 + 1))
			sum := integral[y2+1][x2+1] - integral[y1][x2+1] - integral[y2+1][x1] + integral[y1][x1]

			mean := uint8(sum / area)
			result.SetUCharAt(y, x, mean)
		}
	}

	return result, nil
}

// applyNonLocalMeansDenoising applies advanced denoising
func (p *Processor) applyNonLocalMeansDenoising(src *safe.Mat) (*safe.Mat, error) {
	result, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	srcMat := src.GetMat()
	resultMat := result.GetMat()

	// Use OpenCV's non-local means denoising
	gocv.FastNlMeansDenoising(srcMat, &resultMat)

	return result, nil
}

// calculateThresholdForRegionWithAutoMethod uses automatic method selection
func (p *Processor) calculateThresholdForRegionWithAutoMethod(region *safe.Mat, params map[string]interface{}) float64 {
	method := p.getStringParam(params, "initial_threshold_method")
	histBins := p.getIntParam(params, "histogram_bins")

	if histBins == 0 {
		histBins = p.calculateAdaptiveHistogramBins(region)
	}

	histogram := p.calculateHistogram(region, histBins)

	switch method {
	case "mean":
		return p.calculateMeanThresholdWithSubpixelPrecision(histogram, histBins)
	case "median":
		return p.calculateMedianThresholdWithSubpixelPrecision(histogram, histBins)
	default:
		return p.calculateOtsuThresholdWithSubpixelPrecision(histogram, histBins)
	}
}

// calculateAdaptiveHistogramBins determines optimal bin count based on region characteristics
func (p *Processor) calculateAdaptiveHistogramBins(region *safe.Mat) int {
	rows := region.Rows()
	cols := region.Cols()
	totalPixels := rows * cols

	// Calculate dynamic range of the region
	var minVal, maxVal uint8 = 255, 0
	nonZeroPixels := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			val, _ := region.GetUCharAt(y, x)
			if val > 0 {
				nonZeroPixels++
				if val < minVal {
					minVal = val
				}
				if val > maxVal {
					maxVal = val
				}
			}
		}
	}

	if nonZeroPixels == 0 {
		return 64
	}

	dynamicRange := int(maxVal - minVal)

	// Adaptive calculation based on dynamic range and pixel count
	baseBins := 64
	if dynamicRange < 30 {
		baseBins = 32
	} else if dynamicRange > 150 {
		baseBins = 128
	}

	// Adjust for region size
	if nonZeroPixels < totalPixels/4 { // Small regions
		baseBins = max(baseBins/2, 16)
	}

	return baseBins
}

// segmentRegionWithAdaptiveGaps uses adaptive gap calculation for better segmentation
func (p *Processor) segmentRegionWithAdaptiveGaps(region *safe.Mat, threshold float64, params map[string]interface{}) (*safe.Mat, *safe.Mat, *safe.Mat, error) {
	rows := region.Rows()
	cols := region.Cols()

	foreground, err := safe.NewMat(rows, cols, region.Type())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create foreground Mat: %w", err)
	}

	background, err := safe.NewMat(rows, cols, region.Type())
	if err != nil {
		foreground.Close()
		return nil, nil, nil, fmt.Errorf("failed to create background Mat: %w", err)
	}

	tbd, err := safe.NewMat(rows, cols, region.Type())
	if err != nil {
		foreground.Close()
		background.Close()
		return nil, nil, nil, fmt.Errorf("failed to create TBD Mat: %w", err)
	}

	classSeparation := p.getFloatParam(params, "class_separation")

	// Adaptive gap calculation based on threshold value and image characteristics
	adaptiveGap := classSeparation
	if threshold < 50 {
		adaptiveGap *= 1.5 // Increase gap for dark images
	} else if threshold > 200 {
		adaptiveGap *= 0.8 // Decrease gap for bright images
	}

	lowerThreshold := threshold * (1.0 - adaptiveGap)
	upperThreshold := threshold * (1.0 + adaptiveGap)

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue, err := region.GetUCharAt(y, x)
			if err != nil {
				continue
			}

			if pixelValue > 0 {
				pixelFloat := float64(pixelValue)
				if pixelFloat > upperThreshold {
					foreground.SetUCharAt(y, x, 255)
				} else if pixelFloat < lowerThreshold {
					background.SetUCharAt(y, x, 255)
				} else {
					tbd.SetUCharAt(y, x, 255)
				}
			}
		}
	}

	return foreground, background, tbd, nil
}

// calculateOtsuThresholdWithSubpixelPrecision implements sub-pixel Otsu calculation
func (p *Processor) calculateOtsuThresholdWithSubpixelPrecision(histogram []int, histBins int) float64 {
	total := 0
	for i := 0; i < histBins; i++ {
		total += histogram[i]
	}

	if total == 0 {
		return 127.5
	}

	sum := 0.0
	for i := 0; i < histBins; i++ {
		sum += float64(i) * float64(histogram[i])
	}

	sumB := 0.0
	wB := 0
	maxVariance := 0.0
	bestThreshold := 127.5
	invTotal := 1.0 / float64(total)
	binToValue := 255.0 / float64(histBins-1)

	// Sub-pixel precision search
	subPixelStep := 0.1
	for t := 0.0; t < float64(histBins); t += subPixelStep {
		tInt := int(t)
		if tInt >= histBins {
			break
		}

		// Interpolated weight calculation
		weight := float64(histogram[tInt])
		if tInt+1 < histBins {
			fraction := t - float64(tInt)
			weight = weight*(1.0-fraction) + float64(histogram[tInt+1])*fraction
		}

		wB += int(weight)
		if wB == 0 {
			continue
		}

		wF := total - wB
		if wF == 0 {
			break
		}

		sumB += t * weight

		mB := sumB / float64(wB)
		mF := (sum - sumB) / float64(wF)
		meanDiff := mB - mF

		varBetween := float64(wB) * float64(wF) * invTotal * meanDiff * meanDiff

		if varBetween > maxVariance {
			maxVariance = varBetween
			bestThreshold = t * binToValue
		}
	}

	return bestThreshold
}

// calculateMeanThresholdWithSubpixelPrecision calculates mean with interpolation
func (p *Processor) calculateMeanThresholdWithSubpixelPrecision(histogram []int, histBins int) float64 {
	totalPixels := 0
	weightedSum := 0.0

	for i := 0; i < histBins; i++ {
		totalPixels += histogram[i]
		weightedSum += float64(i) * float64(histogram[i])
	}

	if totalPixels == 0 {
		return 127.5
	}

	meanBin := weightedSum / float64(totalPixels)
	return meanBin * 255.0 / float64(histBins-1)
}

// calculateMedianThresholdWithSubpixelPrecision calculates median with interpolation
func (p *Processor) calculateMedianThresholdWithSubpixelPrecision(histogram []int, histBins int) float64 {
	totalPixels := 0
	for i := 0; i < histBins; i++ {
		totalPixels += histogram[i]
	}

	if totalPixels == 0 {
		return 127.5
	}

	halfPixels := float64(totalPixels) / 2.0
	cumSum := 0.0

	for i := 0; i < histBins; i++ {
		cumSum += float64(histogram[i])
		if cumSum >= halfPixels {
			// Sub-pixel interpolation for median
			if i > 0 && cumSum > halfPixels {
				excess := cumSum - halfPixels
				fraction := excess / float64(histogram[i])
				interpolatedBin := float64(i) - fraction
				return interpolatedBin * 255.0 / float64(histBins-1)
			}
			return float64(i) * 255.0 / float64(histBins-1)
		}
	}

	return 127.5
}

// applyMorphologicalCleanup applies advanced morphological operations
func (p *Processor) applyMorphologicalCleanup(src *safe.Mat) (*safe.Mat, error) {
	kernel3 := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 3, Y: 3})
	defer kernel3.Close()

	kernel5 := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 5, Y: 5})
	defer kernel5.Close()

	// Opening operation to remove small noise
	opened, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}
	defer opened.Close()

	srcMat := src.GetMat()
	openedMat := opened.GetMat()
	gocv.MorphologyEx(srcMat, &openedMat, gocv.MorphOpen, kernel3)

	// Closing operation to fill small gaps
	closed, err := safe.NewMat(opened.Rows(), opened.Cols(), opened.Type())
	if err != nil {
		return nil, err
	}
	defer closed.Close()

	closedMat := closed.GetMat()
	gocv.MorphologyEx(openedMat, &closedMat, gocv.MorphClose, kernel5)

	// Final median filtering for additional noise reduction
	result, err := safe.NewMat(closed.Rows(), closed.Cols(), closed.Type())
	if err != nil {
		return nil, err
	}

	resultMat := result.GetMat()
	gocv.MedianBlur(closedMat, &resultMat, 3)

	return result, nil
}

func (p *Processor) countNonZeroPixels(mat *safe.Mat) int {
	rows := mat.Rows()
	cols := mat.Cols()
	count := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if value, err := mat.GetUCharAt(y, x); err == nil && value > 0 {
				count++
			}
		}
	}

	return count
}

func (p *Processor) calculateHistogram(src *safe.Mat, histBins int) []int {
	histogram := make([]int, histBins)
	rows := src.Rows()
	cols := src.Cols()
	binScale := float64(histBins-1) / 255.0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if pixelValue, err := src.GetUCharAt(y, x); err == nil && pixelValue > 0 {
				bin := int(float64(pixelValue) * binScale)
				bin = max(0, min(bin, histBins-1))
				histogram[bin]++
			}
		}
	}

	return histogram
}

func (p *Processor) updateResult(result, foregroundMask *safe.Mat) {
	rows := result.Rows()
	cols := result.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if value, err := foregroundMask.GetUCharAt(y, x); err == nil && value > 0 {
				result.SetUCharAt(y, x, 255)
			}
		}
	}
}

func (p *Processor) extractTBDRegion(original, tbdMask *safe.Mat) (*safe.Mat, error) {
	result, err := safe.NewMat(original.Rows(), original.Cols(), original.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to create TBD region Mat: %w", err)
	}

	rows := original.Rows()
	cols := original.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if tbdValue, err := tbdMask.GetUCharAt(y, x); err == nil && tbdValue > 0 {
				if origValue, err := original.GetUCharAt(y, x); err == nil {
					result.SetUCharAt(y, x, origValue)
				}
			}
		}
	}

	return result, nil
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

func (p *Processor) getStringParam(params map[string]interface{}, key string) string {
	if value, ok := params[key].(string); ok {
		return value
	}
	return ""
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
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
