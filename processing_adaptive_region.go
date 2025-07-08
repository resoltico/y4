package main

import (
	"fmt"
	"image"
	"math"

	"gocv.io/x/gocv"
)

// Complete region adaptive processing implementation
func (pe *ProcessingEngine) processRegionAdaptive(src gocv.Mat, params *OtsuParameters) gocv.Mat {
	if err := validateMatForMetrics(src, "region adaptive processing"); err != nil {
		return gocv.NewMat()
	}

	rows, cols := src.Rows(), src.Cols()

	if err := validateImageDimensions(cols, rows, "region adaptive dimensions"); err != nil {
		return gocv.NewMat()
	}

	debugSystem := GetDebugSystem()

	// Check if overlapping regions should be used
	useOverlapping := pe.shouldUseOverlappingRegions(src, params)

	if useOverlapping {
		debugSystem.logger.Info("using overlapping regions for complex image")
		return pe.processOverlappingRegions(src, params)
	}

	// Standard non-overlapping region processing
	gridSize := pe.calculateAdaptiveGridSize(src)

	if gridSize > intMin(rows, cols)/2 {
		debugSystem.logger.Warn("grid size too large for image dimensions, falling back to single scale",
			"grid_size", gridSize,
			"image_rows", rows,
			"image_cols", cols)
		return pe.processSingleScaleAdaptive(src, params)
	}

	// Initialize result matrix to background (BLACK = 0)
	result := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)
	backgroundScalar := gocv.NewScalar(255, 0, 0, 0) // BLACK = background
	result.SetTo(backgroundScalar)

	regionsProcessed := 0
	regionErrors := 0
	regionsSkipped := 0
	lowContrastRegions := 0
	totalContrast := 0.0

	// Debug tracking
	totalForegroundPixels := 0
	totalBackgroundPixels := 0

	// Process regions using efficient row/column operations
	for y := 0; y < rows; y += gridSize {
		endY := intMin(y+gridSize, rows)

		for x := 0; x < cols; x += gridSize {
			endX := intMin(x+gridSize, cols)

			// Extract region using matrix slicing
			srcRegion := src.Region(image.Rect(x, y, endX, endY))

			if srcRegion.Rows() < 16 || srcRegion.Cols() < 16 {
				srcRegion.Close()
				regionErrors++
				continue
			}

			hasContrast, contrast, _ := pe.validateRegionContrastAdaptive(srcRegion)
			totalContrast += contrast

			if !hasContrast {
				lowContrastRegions++
				debugSystem.logger.Debug("region quality analysis",
					"x", x, "y", y,
					"width", endX-x, "height", endY-y,
					"has_contrast", false,
					"contrast", contrast,
					"entropy", 0)

				srcRegion.Close()
				regionsSkipped++
				// Region remains initialized background (BLACK) - consistent
				regionPixels := (endX - x) * (endY - y)
				totalBackgroundPixels += regionPixels
				continue
			}

			debugSystem.logger.Debug("region quality analysis",
				"x", x, "y", y,
				"width", endX-x, "height", endY-y,
				"has_contrast", true,
				"contrast", contrast,
				"entropy", 0)

			regionParams := *params
			regionParams.RegionAdaptiveThresholding = false
			regionResult := pe.processSingleScaleAdaptive(srcRegion, &regionParams)

			if !regionResult.Empty() {
				// Count pixels in this region result
				regionForeground, regionErr := calculateSafeCountNonZero(regionResult, "region result")
				if regionErr == nil {
					regionPixels := (endX - x) * (endY - y)
					regionBackground := regionPixels - regionForeground
					totalForegroundPixels += regionForeground
					totalBackgroundPixels += regionBackground

					debugSystem.logger.Debug("region processing result",
						"x", x, "y", y,
						"region_pixels", regionPixels,
						"foreground_pixels", regionForeground,
						"background_pixels", regionBackground,
						"foreground_ratio", float64(regionForeground)/float64(regionPixels))
				}

				dstRegion := result.Region(image.Rect(x, y, endX, endY))
				regionResult.CopyTo(&dstRegion)
				dstRegion.Close()
				regionsProcessed++
			} else {
				regionErrors++
				// Failed region remains background
				regionPixels := (endX - x) * (endY - y)
				totalBackgroundPixels += regionPixels
			}

			srcRegion.Close()
			regionResult.Close()
		}
	}

	totalRegions := regionsProcessed + regionErrors + regionsSkipped
	avgContrast := 0.0
	if totalRegions > 0 {
		avgContrast = totalContrast / float64(totalRegions)
	}

	// Validate final result
	minVal, maxVal, _, _ := gocv.MinMaxLoc(result)
	finalForeground, _ := calculateSafeCountNonZero(result, "final result")
	totalPixels := rows * cols
	finalBackground := totalPixels - finalForeground
	finalForegroundRatio := float64(finalForeground) / float64(totalPixels)

	debugSystem.logger.Info("region adaptive processing complete",
		"regions_processed", regionsProcessed,
		"regions_skipped", regionsSkipped,
		"region_errors", regionErrors,
		"low_contrast_regions", lowContrastRegions,
		"average_contrast", avgContrast,
		"grid_size", gridSize,
		"image_dimensions", []int{cols, rows},
		"result_min_value", float64(minVal),
		"result_max_value", float64(maxVal),
		"final_foreground_pixels", finalForeground,
		"final_background_pixels", finalBackground,
		"final_foreground_ratio", finalForegroundRatio,
		"total_pixels", totalPixels)

	debugSystem.TraceContrastAnalysis(0, totalRegions, lowContrastRegions, avgContrast)

	// Check for uniform output
	if minVal == maxVal {
		debugSystem.logger.Error("uniform output detected",
			"uniform_value", float64(minVal),
			"total_regions", totalRegions,
			"processed_regions", regionsProcessed,
			"skipped_regions", regionsSkipped)

		// Apply global fallback
		result.Close()
		globalResult := gocv.NewMat()
		gocv.Threshold(src, &globalResult, 0, 255, gocv.ThresholdBinary+gocv.ThresholdOtsu)
		debugSystem.logger.Info("applied global Otsu fallback")
		return globalResult
	}

	if err := validateMatForMetrics(result, "region adaptive result"); err != nil {
		result.Close()
		return gocv.NewMat()
	}

	return result
}

func (pe *ProcessingEngine) processSingleScaleAdaptive(src gocv.Mat, params *OtsuParameters) gocv.Mat {
	if err := validateMatForMetrics(src, "single scale adaptive processing"); err != nil {
		return gocv.NewMat()
	}

	windowSize := params.WindowSize
	if params.AdaptiveWindowSizing {
		windowSize = pe.calculateAdaptiveWindowSize(src)
	}

	neighborhood := pe.calculateNeighborhood(src, windowSize, params.NeighborhoodType)
	defer neighborhood.Close()

	histBins := params.HistogramBins
	if histBins == 0 {
		histBins = pe.calculateHistogramBins(src)
	}

	histogram := pe.build2DHistogram(src, neighborhood, histBins)

	if params.UseLogHistogram {
		pe.applyLogScaling(histogram)
	}

	if params.NormalizeHistogram {
		pe.normalizeHistogram(histogram)
	}

	if params.SmoothingStrength > 0 {
		pe.smoothHistogram(histogram, params.SmoothingStrength)
	}

	threshold := pe.find2DOtsuThresholdInteger(histogram)
	result := pe.applyThreshold(src, neighborhood, threshold, histBins)

	if err := validateMatForMetrics(result, "single scale adaptive result"); err != nil {
		result.Close()
		return gocv.NewMat()
	}

	return result
}

func (pe *ProcessingEngine) validateRegionContrastAdaptive(src gocv.Mat) (bool, float64, error) {
	if err := validateMatForMetrics(src, "contrast validation"); err != nil {
		return false, 0, err
	}

	minVal, maxVal, _, _ := gocv.MinMaxLoc(src)
	contrast := float64(maxVal - minVal)

	if contrast < 15.0 {
		return false, contrast, fmt.Errorf("insufficient contrast: %.2f (minimum 15.0)", contrast)
	}
	return true, contrast, nil
}

func (pe *ProcessingEngine) calculateAdaptiveGridSize(src gocv.Mat) int {
	if err := validateMatForMetrics(src, "adaptive grid size calculation"); err != nil {
		return 64 // fallback grid size
	}

	rows, cols := src.Rows(), src.Cols()

	// Calculate histogram-based complexity
	hist := gocv.NewMat()
	defer hist.Close()

	mask := gocv.NewMat() // empty mask
	defer mask.Close()

	channels := []int{0}
	histSize := []int{64} // reduce bins for speed
	ranges := []float64{0, 256}

	err := gocv.CalcHist([]gocv.Mat{src}, channels, mask, &hist, histSize, ranges, false)
	if err != nil {
		debugSystem := GetDebugSystem()
		debugSystem.logger.Warn("histogram calculation failed for grid sizing", "error", err.Error())
		return intMax(64, intMin(rows, cols)/8)
	}

	entropy := calculateHistogramEntropy(hist)
	contrast := calculateRegionContrast(src)

	baseSize := intMin(rows, cols) / 6 // less aggressive than /8

	debugSystem := GetDebugSystem()
	debugSystem.logger.Debug("grid size calculation",
		"entropy", entropy,
		"contrast", contrast,
		"base_size", baseSize,
		"image_dimensions", fmt.Sprintf("%dx%d", cols, rows))

	// Adapt based on image complexity
	if entropy > 6.5 && contrast > 30.0 {
		gridSize := intMax(32, baseSize/2) // fine grid for complex regions
		debugSystem.logger.Debug("using fine grid for complex image", "grid_size", gridSize)
		return gridSize
	}

	if entropy < 4.0 || contrast < 15.0 {
		gridSize := intMax(96, baseSize*3/2) // coarser for uniform regions
		debugSystem.logger.Debug("using coarse grid for uniform image", "grid_size", gridSize)
		return gridSize
	}

	gridSize := intMax(64, baseSize) // standard grid
	debugSystem.logger.Debug("using standard grid", "grid_size", gridSize)
	return gridSize
}

func calculateHistogramEntropy(hist gocv.Mat) float64 {
	if hist.Empty() {
		return 0.0
	}

	rows := hist.Rows()
	total := 0.0
	entropy := 0.0

	// Calculate total
	for i := 0; i < rows; i++ {
		value := float64(hist.GetFloatAt(i, 0))
		total += value
	}

	if total == 0 {
		return 0.0
	}

	// Calculate entropy
	for i := 0; i < rows; i++ {
		value := float64(hist.GetFloatAt(i, 0))
		if value > 0 {
			probability := value / total
			entropy -= probability * math.Log2(probability)
		}
	}

	return entropy
}

func calculateRegionContrast(src gocv.Mat) float64 {
	if err := validateMatForMetrics(src, "region contrast calculation"); err != nil {
		return 0.0
	}

	minVal, maxVal, _, _ := gocv.MinMaxLoc(src)
	return float64(maxVal - minVal)
}

func (pe *ProcessingEngine) shouldUseOverlappingRegions(src gocv.Mat, params *OtsuParameters) bool {
	entropy := pe.calculateImageEntropy(src)
	contrast := calculateRegionContrast(src)

	complexityThreshold := 10.0
	contrastThreshold := 25.0

	isComplex := entropy > complexityThreshold && contrast > contrastThreshold

	debugSystem := GetDebugSystem()
	debugSystem.logger.Debug("complexity analysis for overlap decision",
		"entropy", entropy,
		"contrast", contrast,
		"is_complex", isComplex,
		"complexity_threshold", complexityThreshold,
		"contrast_threshold", contrastThreshold)

	return isComplex
}

func (pe *ProcessingEngine) calculateImageEntropy(src gocv.Mat) float64 {
	if err := validateMatForMetrics(src, "image entropy calculation"); err != nil {
		return 0.0
	}

	hist := gocv.NewMat()
	defer hist.Close()

	mask := gocv.NewMat()
	defer mask.Close()

	channels := []int{0}
	histSize := []int{256}
	ranges := []float64{0, 256}

	err := gocv.CalcHist([]gocv.Mat{src}, channels, mask, &hist, histSize, ranges, false)
	if err != nil {
		return 0.0
	}

	return calculateHistogramEntropy(hist)
}

func (pe *ProcessingEngine) processOverlappingRegions(src gocv.Mat, params *OtsuParameters) gocv.Mat {
	if err := validateMatForMetrics(src, "overlapping regions processing"); err != nil {
		return gocv.NewMat()
	}

	rows, cols := src.Rows(), src.Cols()
	gridSize := pe.calculateAdaptiveGridSize(src)
	overlap := gridSize / 4 // 25% overlap

	result := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)
	weights := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV32F)

	// Initialize matrices to background
	backgroundScalar := gocv.NewScalar(255, 0, 0, 0) // BLACK = background
	zeroScalar := gocv.NewScalar(0, 0, 0, 0)         // Zero weights

	result.SetTo(backgroundScalar)
	weights.SetTo(zeroScalar)

	defer result.Close()
	defer weights.Close()

	regionsProcessed := 0
	regionsSkipped := 0

	debugSystem := GetDebugSystem()
	debugSystem.logger.Debug("starting overlapping region processing",
		"grid_size", gridSize,
		"overlap", overlap,
		"total_regions_estimate", (rows/gridSize+1)*(cols/gridSize+1))

	for y := 0; y < rows; y += gridSize - overlap {
		endY := intMin(y+gridSize, rows)

		for x := 0; x < cols; x += gridSize - overlap {
			endX := intMin(x+gridSize, cols)

			// Validate region size
			regionWidth := endX - x
			regionHeight := endY - y
			if regionWidth < 16 || regionHeight < 16 {
				regionsSkipped++
				continue
			}

			regionResult := pe.processRegionWithMultilevelFallback(src, x, y, endX, endY, params)
			if regionResult.Empty() {
				regionsSkipped++
				continue
			}

			regionWeight := pe.createGaussianWeight(regionWidth, regionHeight)

			// Extract target regions using Region
			targetRegion := result.Region(image.Rect(x, y, endX, endY))
			targetWeights := weights.Region(image.Rect(x, y, endX, endY))

			pe.blendRegionWeighted(regionResult, regionWeight, &targetRegion, &targetWeights)

			targetRegion.Close()
			targetWeights.Close()
			regionResult.Close()
			regionWeight.Close()

			regionsProcessed++
		}
	}

	pe.normalizeByWeights(&result, weights)

	debugSystem.logger.Info("overlapping region processing completed",
		"regions_processed", regionsProcessed,
		"regions_skipped", regionsSkipped,
		"processing_rate", float64(regionsProcessed)/float64(regionsProcessed+regionsSkipped))

	if err := validateMatForMetrics(result, "overlapping regions result"); err != nil {
		result.Close()
		return gocv.NewMat()
	}

	return result.Clone()
}

func (pe *ProcessingEngine) processRegionWithMultilevelFallback(src gocv.Mat, x, y, endX, endY int, params *OtsuParameters) gocv.Mat {
	// Extract region using efficient matrix slicing
	region := src.Region(image.Rect(x, y, endX, endY))
	defer region.Close()

	debugSystem := GetDebugSystem()

	// Level 1: Quality analysis
	hasContrast, contrast, entropy := pe.analyzeRegionQuality(region)

	debugSystem.logger.Debug("region quality analysis",
		"x", x, "y", y,
		"width", endX-x, "height", endY-y,
		"has_contrast", hasContrast,
		"contrast", contrast,
		"entropy", entropy)

	// Return empty Mat for zero-contrast regions (let caller handle background)
	if !hasContrast {
		debugSystem.logger.Debug("returning empty result for zero-contrast region")
		return gocv.NewMat()
	}

	// Level 1: Standard 2D Otsu for high-quality regions
	if hasContrast && contrast > 20.0 && entropy > 5.0 {
		if pe.detectBimodalDistribution(region) {
			debugSystem.logger.Debug("using standard 2D Otsu for high-quality bimodal region")
			return pe.processSingleScaleAdaptive(region, params)
		}
	}

	// Level 2: Adaptive window growing for medium-quality regions
	if contrast > 10.0 && entropy > 3.0 {
		debugSystem.logger.Debug("using adaptive window growing for medium-quality region")
		expandedRegion := pe.expandRegionAdaptively(src, x, y, endX, endY)
		if !expandedRegion.Empty() {
			defer expandedRegion.Close()
			if err := validateMatForMetrics(expandedRegion, "expanded region"); err == nil {
				return pe.processSingleScaleAdaptive(expandedRegion, params)
			}
		}
	}

	// Level 3: Global method fallback for low-quality regions
	debugSystem.logger.Debug("using global method fallback for low-quality region")
	globalParams := *params
	globalParams.AdaptiveWindowSizing = true
	globalParams.SmoothingStrength = 2.0
	globalParams.GaussianPreprocessing = true

	return pe.processSingleScaleAdaptive(region, &globalParams)
}

func (pe *ProcessingEngine) analyzeRegionQuality(region gocv.Mat) (bool, float64, float64) {
	if err := validateMatForMetrics(region, "region quality analysis"); err != nil {
		return false, 0.0, 0.0
	}

	// Calculate contrast
	minVal, maxVal, _, _ := gocv.MinMaxLoc(region)
	contrast := float64(maxVal - minVal)

	// Calculate entropy using histogram
	hist := gocv.NewMat()
	defer hist.Close()

	mask := gocv.NewMat()
	defer mask.Close()

	channels := []int{0}
	histSize := []int{64}
	ranges := []float64{0, 256}

	err := gocv.CalcHist([]gocv.Mat{region}, channels, mask, &hist, histSize, ranges, false)
	if err != nil {
		return contrast > 15.0, contrast, 0.0
	}

	entropy := calculateHistogramEntropy(hist)

	return contrast > 15.0, contrast, entropy
}

func (pe *ProcessingEngine) detectBimodalDistribution(region gocv.Mat) bool {
	if err := validateMatForMetrics(region, "bimodal detection"); err != nil {
		return false
	}

	hist := gocv.NewMat()
	defer hist.Close()

	mask := gocv.NewMat()
	defer mask.Close()

	channels := []int{0}
	histSize := []int{64}
	ranges := []float64{0, 256}

	err := gocv.CalcHist([]gocv.Mat{region}, channels, mask, &hist, histSize, ranges, false)
	if err != nil {
		return false
	}

	// Extract histogram data
	histData := make([]float32, 64)
	for i := 0; i < 64; i++ {
		histData[i] = hist.GetFloatAt(i, 0)
	}

	peaks := pe.findHistogramPeaks(histData)
	valleys := pe.findHistogramValleys(histData)

	// Bimodal detection: 2 significant peaks with deep valley between
	if len(peaks) >= 2 && len(valleys) >= 1 {
		peakSeparation := int(math.Abs(float64(peaks[1] - peaks[0])))
		valleyDepth := (histData[peaks[0]]+histData[peaks[1]])/2 - histData[valleys[0]]
		maxPeak := floatMax(histData[peaks[0]], histData[peaks[1]])

		return peakSeparation > 10 && valleyDepth > 0.2*maxPeak
	}

	return false
}

func (pe *ProcessingEngine) findHistogramPeaks(data []float32) []int {
	peaks := make([]int, 0)

	if len(data) < 3 {
		return peaks
	}

	maxValue := pe.findMaxValue(data)
	threshold := 0.1 * maxValue

	for i := 1; i < len(data)-1; i++ {
		if data[i] > data[i-1] && data[i] > data[i+1] && data[i] > threshold {
			peaks = append(peaks, i)
		}
	}

	return peaks
}

func (pe *ProcessingEngine) findHistogramValleys(data []float32) []int {
	valleys := make([]int, 0)

	if len(data) < 3 {
		return valleys
	}

	for i := 1; i < len(data)-1; i++ {
		if data[i] < data[i-1] && data[i] < data[i+1] {
			valleys = append(valleys, i)
		}
	}

	return valleys
}

func (pe *ProcessingEngine) findMaxValue(data []float32) float32 {
	if len(data) == 0 {
		return 0
	}

	maxVal := data[0]
	for _, val := range data[1:] {
		if val > maxVal {
			maxVal = val
		}
	}
	return maxVal
}

func (pe *ProcessingEngine) expandRegionAdaptively(src gocv.Mat, x, y, endX, endY int) gocv.Mat {
	rows, cols := src.Rows(), src.Cols()

	// Calculate expansion based on current region contrast
	currentRegion := src.Region(image.Rect(x, y, endX, endY))
	defer currentRegion.Close()

	_, contrast, _ := pe.analyzeRegionQuality(currentRegion)

	// Adaptive expansion factor based on contrast
	expansionFactor := 1.5
	if contrast < 10.0 {
		expansionFactor = 2.0
	} else if contrast > 25.0 {
		expansionFactor = 1.2
	}

	currentWidth := endX - x
	currentHeight := endY - y

	expandWidth := int(float64(currentWidth) * expansionFactor)
	expandHeight := int(float64(currentHeight) * expansionFactor)

	// Calculate new bounds with expansion
	newX := intMax(0, x-(expandWidth-currentWidth)/2)
	newY := intMax(0, y-(expandHeight-currentHeight)/2)
	newEndX := intMin(cols, newX+expandWidth)
	newEndY := intMin(rows, newY+expandHeight)

	// Validate expansion result
	if newEndX <= newX || newEndY <= newY {
		return gocv.NewMat()
	}

	if newEndX-newX < 32 || newEndY-newY < 32 {
		return gocv.NewMat() // expansion too small
	}

	debugSystem := GetDebugSystem()
	debugSystem.logger.Debug("region expansion",
		"original", fmt.Sprintf("%d,%d-%d,%d", x, y, endX, endY),
		"expanded", fmt.Sprintf("%d,%d-%d,%d", newX, newY, newEndX, newEndY),
		"expansion_factor", expansionFactor,
		"contrast", contrast)

	return src.Region(image.Rect(newX, newY, newEndX, newEndY))
}

func (pe *ProcessingEngine) createGaussianWeight(width, height int) gocv.Mat {
	weight := gocv.NewMatWithSize(height, width, gocv.MatTypeCV32F)

	centerX, centerY := float64(width)/2, float64(height)/2
	sigma := float64(intMin(width, height)) / 6

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dx, dy := float64(x)-centerX, float64(y)-centerY
			distance := math.Sqrt(dx*dx + dy*dy)
			w := math.Exp(-(distance * distance) / (2 * sigma * sigma))
			weight.SetFloatAt(y, x, float32(w))
		}
	}

	return weight
}

func (pe *ProcessingEngine) blendRegionWeighted(newRegion, regionWeight gocv.Mat, targetRegion, targetWeights *gocv.Mat) {
	if err := validateMatForMetrics(newRegion, "blend new region"); err != nil {
		return
	}

	if err := validateMatForMetrics(regionWeight, "blend region weight"); err != nil {
		return
	}

	// Convert to float for blending calculations
	newFloat := gocv.NewMat()
	defer newFloat.Close()
	if err := newRegion.ConvertTo(&newFloat, gocv.MatTypeCV32F); err != nil {
		return
	}

	currentFloat := gocv.NewMat()
	defer currentFloat.Close()
	if err := targetRegion.ConvertTo(&currentFloat, gocv.MatTypeCV32F); err != nil {
		return
	}

	// Weighted sum: result = (current*currentWeights + new*newWeights) / (currentWeights + newWeights)
	weighted1 := gocv.NewMat()
	weighted2 := gocv.NewMat()
	defer weighted1.Close()
	defer weighted2.Close()

	gocv.Multiply(currentFloat, *targetWeights, &weighted1)
	gocv.Multiply(newFloat, regionWeight, &weighted2)

	combined := gocv.NewMat()
	defer combined.Close()
	gocv.Add(weighted1, weighted2, &combined)

	newWeights := gocv.NewMat()
	defer newWeights.Close()
	gocv.Add(*targetWeights, regionWeight, &newWeights)

	// Create threshold matrix for avoiding division by zero
	thresholdMat := gocv.NewMatWithSize(newWeights.Rows(), newWeights.Cols(), gocv.MatTypeCV32F)
	defer thresholdMat.Close()
	thresholdScalar := gocv.NewScalar(1e-6, 0, 0, 0)
	thresholdMat.SetTo(thresholdScalar)

	mask := gocv.NewMat()
	defer mask.Close()
	gocv.Compare(newWeights, thresholdMat, &mask, gocv.CompareGT)

	// Apply mask to avoid zero division
	nonZeroCount, _ := calculateSafeCountNonZero(mask, "blend weights check")
	if nonZeroCount > 0 {
		gocv.Divide(combined, newWeights, &combined)
		combined.CopyTo(targetRegion)
		newWeights.CopyTo(targetWeights)
	}
}

func (pe *ProcessingEngine) normalizeByWeights(result *gocv.Mat, weights gocv.Mat) {
	if err := validateMatForMetrics(*result, "normalize result"); err != nil {
		return
	}

	if err := validateMatForMetrics(weights, "normalize weights"); err != nil {
		return
	}

	// Create threshold matrix for sufficient weight
	thresholdMat := gocv.NewMatWithSize(weights.Rows(), weights.Cols(), gocv.MatTypeCV32F)
	defer thresholdMat.Close()
	thresholdScalar := gocv.NewScalar(0.1, 0, 0, 0)
	thresholdMat.SetTo(thresholdScalar)

	mask := gocv.NewMat()
	defer mask.Close()
	gocv.Compare(weights, thresholdMat, &mask, gocv.CompareGT)

	// Convert result to 8-bit if needed
	if result.Type() != gocv.MatTypeCV8UC1 {
		converted := gocv.NewMat()
		defer converted.Close()
		if err := result.ConvertTo(&converted, gocv.MatTypeCV8UC1); err == nil {
			converted.CopyTo(result)
		}
	}

	debugSystem := GetDebugSystem()
	nonZeroWeights, err := calculateSafeCountNonZero(mask, "weight normalization")
	if err == nil {
		totalPixels := result.Rows() * result.Cols()
		debugSystem.logger.Debug("weight normalization completed",
			"pixels_with_weights", nonZeroWeights,
			"total_pixels", totalPixels,
			"coverage_ratio", float64(nonZeroWeights)/float64(totalPixels))
	}
}

// Helper functions for math operations
func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func intMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func floatMax(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
