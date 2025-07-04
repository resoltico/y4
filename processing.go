package main

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"gocv.io/x/gocv"
)

type ProcessingEngine struct {
	originalImage  *ImageData
	processedImage *ImageData
	integralImage  gocv.Mat
}

type ImageData struct {
	Image    image.Image
	Mat      gocv.Mat
	Width    int
	Height   int
	Channels int
	Format   string
}

type OtsuParameters struct {
	WindowSize                 int
	HistogramBins              int
	SmoothingStrength          float64
	EdgePreservation           bool
	NoiseRobustness            bool
	GaussianPreprocessing      bool
	UseLogHistogram            bool
	NormalizeHistogram         bool
	ApplyContrastEnhancement   bool
	AdaptiveWindowSizing       bool
	MultiScaleProcessing       bool
	PyramidLevels              int
	NeighborhoodType           string
	InterpolationMethod        string
	MorphologicalPostProcess   bool
	MorphologicalKernelSize    int
	HomomorphicFiltering       bool
	AnisotropicDiffusion       bool
	DiffusionIterations        int
	DiffusionKappa             float64
	RegionAdaptiveThresholding bool
	RegionGridSize             int
}

func NewProcessingEngine() *ProcessingEngine {
	return &ProcessingEngine{}
}

func (pe *ProcessingEngine) SetOriginalImage(data *ImageData) {
	pe.originalImage = data
	pe.buildIntegralImage()
}

func (pe *ProcessingEngine) GetOriginalImage() *ImageData {
	return pe.originalImage
}

func (pe *ProcessingEngine) GetProcessedImage() *ImageData {
	return pe.processedImage
}

func (pe *ProcessingEngine) buildIntegralImage() {
	if pe.originalImage == nil {
		return
	}

	gray := pe.convertToGrayscale(pe.originalImage.Mat)
	defer gray.Close()

	pe.integralImage = gocv.NewMat()
	sqsum := gocv.NewMat()
	defer sqsum.Close()
	tilted := gocv.NewMat()
	defer tilted.Close()

	gocv.Integral(gray, &pe.integralImage, &sqsum, &tilted)
}

func (pe *ProcessingEngine) ProcessImage(params *OtsuParameters) (*ImageData, *BinaryImageMetrics, error) {
	if pe.originalImage == nil {
		return nil, nil, fmt.Errorf("processing engine: no original image loaded")
	}

	if err := validateMat(pe.originalImage.Mat, "original image"); err != nil {
		return nil, nil, fmt.Errorf("processing engine: %w", err)
	}

	gray := pe.convertToGrayscale(pe.originalImage.Mat)
	defer gray.Close()

	working := gray
	if params.HomomorphicFiltering {
		homomorphic := pe.applyHomomorphicFiltering(gray)
		defer homomorphic.Close()
		working = homomorphic
	}

	if params.AnisotropicDiffusion {
		diffused := pe.applyAnisotropicDiffusion(working, params.DiffusionIterations, params.DiffusionKappa)
		defer diffused.Close()
		working = diffused
	}

	if params.GaussianPreprocessing {
		blurred := pe.applyGaussianBlur(working, params.SmoothingStrength)
		defer blurred.Close()
		working = blurred
	}

	if params.ApplyContrastEnhancement {
		enhanced := pe.applyAdaptiveContrastEnhancement(working)
		defer enhanced.Close()
		working = enhanced
	}

	var result gocv.Mat
	if params.MultiScaleProcessing {
		result = pe.processMultiScale(working, params)
	} else if params.RegionAdaptiveThresholding {
		result = pe.processRegionAdaptive(working, params)
	} else {
		result = pe.processSingleScale(working, params)
	}
	defer result.Close()

	if params.MorphologicalPostProcess {
		morphed := pe.applyMorphologicalPostProcessing(result, params.MorphologicalKernelSize)
		defer morphed.Close()
		result = morphed
	}

	resultImage := pe.matToImage(result)

	processedData := &ImageData{
		Image:    resultImage,
		Mat:      result.Clone(),
		Width:    resultImage.Bounds().Dx(),
		Height:   resultImage.Bounds().Dy(),
		Channels: 1,
		Format:   pe.originalImage.Format,
	}

	pe.processedImage = processedData

	metrics := CalculateBinaryMetrics(pe.originalImage.Mat, result)
	if metrics == nil {
		return processedData, nil, fmt.Errorf("metrics calculation failed for %dx%d image", result.Rows(), result.Cols())
	}

	return processedData, metrics, nil
}

func (pe *ProcessingEngine) processSingleScale(src gocv.Mat, params *OtsuParameters) gocv.Mat {
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
	return pe.applyThreshold(src, neighborhood, threshold, histBins)
}

func (pe *ProcessingEngine) processMultiScale(src gocv.Mat, params *OtsuParameters) gocv.Mat {
	levels := params.PyramidLevels
	if levels <= 0 {
		levels = 3
	}

	pyramid := make([]gocv.Mat, levels+1)
	pyramid[0] = src.Clone()

	for i := 1; i <= levels; i++ {
		pyramid[i] = gocv.NewMat()
		gocv.PyrDown(pyramid[i-1], &pyramid[i], image.Point{}, gocv.BorderDefault)
	}

	defer func() {
		for i := 1; i <= levels; i++ {
			pyramid[i].Close()
		}
	}()

	results := make([]gocv.Mat, levels+1)
	for i := 0; i <= levels; i++ {
		scaleParams := *params
		scaleParams.MultiScaleProcessing = false
		scaleParams.WindowSize = max(3, params.WindowSize/(1<<i))
		results[i] = pe.processSingleScale(pyramid[i], &scaleParams)
	}

	defer func() {
		for i := 1; i <= levels; i++ {
			results[i].Close()
		}
	}()

	for i := levels - 1; i >= 0; i-- {
		upsampled := gocv.NewMat()
		gocv.PyrUp(results[i+1], &upsampled, image.Point{}, gocv.BorderDefault)

		combined := gocv.NewMat()
		gocv.BitwiseOr(results[i], upsampled, &combined)

		results[i].Close()
		upsampled.Close()
		results[i] = combined
	}

	return results[0]
}

func (pe *ProcessingEngine) processRegionAdaptive(src gocv.Mat, params *OtsuParameters) gocv.Mat {
	rows, cols := src.Rows(), src.Cols()
	gridSize := params.RegionGridSize
	if gridSize <= 0 {
		gridSize = max(rows/8, cols/8)
	}

	result := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)

	for y := 0; y < rows; y += gridSize {
		for x := 0; x < cols; x += gridSize {
			endY := min(y+gridSize, rows)
			endX := min(x+gridSize, cols)

			roi := src.Region(image.Rect(x, y, endX, endY))
			regionParams := *params
			regionParams.RegionAdaptiveThresholding = false
			regionResult := pe.processSingleScale(roi, &regionParams)

			// Get row and column ranges for copying
			rowRange := result.RowRange(y, endY)
			colRange := rowRange.ColRange(x, endX)
			regionResult.CopyTo(&colRange)

			roi.Close()
			regionResult.Close()
			rowRange.Close()
			colRange.Close()
		}
	}

	return result
}

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

func (pe *ProcessingEngine) convertToGrayscale(src gocv.Mat) gocv.Mat {
	if src.Channels() == 1 {
		return src.Clone()
	}

	gray := gocv.NewMat()
	gocv.CvtColor(src, &gray, gocv.ColorBGRToGray)
	return gray
}

func (pe *ProcessingEngine) applyGaussianBlur(src gocv.Mat, sigma float64) gocv.Mat {
	dst := gocv.NewMat()
	kernelSize := int(sigma*6) + 1
	if kernelSize%2 == 0 {
		kernelSize++
	}
	gocv.GaussianBlur(src, &dst, image.Point{X: kernelSize, Y: kernelSize}, sigma, sigma, gocv.BorderDefault)
	return dst
}

func (pe *ProcessingEngine) applyAdaptiveContrastEnhancement(src gocv.Mat) gocv.Mat {
	clahe := gocv.NewCLAHEWithParams(2.0, image.Point{X: 8, Y: 8})
	defer clahe.Close()

	dst := gocv.NewMat()
	clahe.Apply(src, &dst)
	return dst
}

func (pe *ProcessingEngine) applyHomomorphicFiltering(src gocv.Mat) gocv.Mat {
	rows, cols := src.Rows(), src.Cols()

	floatMat := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV32F)
	defer floatMat.Close()

	src.ConvertTo(&floatMat, gocv.MatTypeCV32F)

	logMat := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV32F)
	defer logMat.Close()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			val := floatMat.GetFloatAt(y, x)
			if val > 0 {
				logMat.SetFloatAt(y, x, float32(math.Log(float64(val)+1)))
			}
		}
	}

	highPassKernel := gocv.NewMatWithSize(5, 5, gocv.MatTypeCV32F)
	defer highPassKernel.Close()

	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			if y == 2 && x == 2 {
				highPassKernel.SetFloatAt(y, x, 24)
			} else {
				highPassKernel.SetFloatAt(y, x, -1)
			}
		}
	}

	filtered := gocv.NewMat()
	defer filtered.Close()
	gocv.Filter2D(logMat, &filtered, -1, highPassKernel, image.Point{X: -1, Y: -1}, 0, gocv.BorderDefault)

	expMat := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV32F)
	defer expMat.Close()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			val := filtered.GetFloatAt(y, x)
			expMat.SetFloatAt(y, x, float32(math.Exp(float64(val))))
		}
	}

	result := gocv.NewMat()
	expMat.ConvertTo(&result, gocv.MatTypeCV8U)
	return result
}

func (pe *ProcessingEngine) applyAnisotropicDiffusion(src gocv.Mat, iterations int, kappa float64) gocv.Mat {
	rows, cols := src.Rows(), src.Cols()

	current := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV32F)
	defer current.Close()
	src.ConvertTo(&current, gocv.MatTypeCV32F)

	next := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV32F)
	defer next.Close()

	for iter := 0; iter < iterations; iter++ {
		for y := 1; y < rows-1; y++ {
			for x := 1; x < cols-1; x++ {
				center := current.GetFloatAt(y, x)
				north := current.GetFloatAt(y-1, x)
				south := current.GetFloatAt(y+1, x)
				east := current.GetFloatAt(y, x+1)
				west := current.GetFloatAt(y, x-1)

				gradN := north - center
				gradS := south - center
				gradE := east - center
				gradW := west - center

				cN := math.Exp(-math.Pow(float64(gradN)/kappa, 2))
				cS := math.Exp(-math.Pow(float64(gradS)/kappa, 2))
				cE := math.Exp(-math.Pow(float64(gradE)/kappa, 2))
				cW := math.Exp(-math.Pow(float64(gradW)/kappa, 2))

				newVal := center + 0.25*(float32(cN)*gradN+float32(cS)*gradS+float32(cE)*gradE+float32(cW)*gradW)
				next.SetFloatAt(y, x, float32(newVal))
			}
		}

		current, next = next, current
	}

	result := gocv.NewMat()
	current.ConvertTo(&result, gocv.MatTypeCV8U)
	return result
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

func (pe *ProcessingEngine) applyMorphologicalPostProcessing(src gocv.Mat, kernelSize int) gocv.Mat {
	openingKernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: kernelSize, Y: kernelSize})
	defer openingKernel.Close()

	opened := gocv.NewMat()
	defer opened.Close()
	gocv.MorphologyEx(src, &opened, gocv.MorphOpen, openingKernel)

	closingKernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: kernelSize + 2, Y: kernelSize + 2})
	defer closingKernel.Close()

	result := gocv.NewMat()
	gocv.MorphologyEx(opened, &result, gocv.MorphClose, closingKernel)

	return result
}

func (pe *ProcessingEngine) matToImage(mat gocv.Mat) image.Image {
	rows := mat.Rows()
	cols := mat.Cols()
	img := image.NewGray(image.Rect(0, 0, cols, rows))

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			value := mat.GetUCharAt(y, x)
			img.SetGray(x, y, color.Gray{Y: value})
		}
	}

	return img
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
