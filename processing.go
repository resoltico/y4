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
	WindowSize               int
	HistogramBins            int
	SmoothingStrength        float64
	EdgePreservation         bool
	NoiseRobustness          bool
	GaussianPreprocessing    bool
	UseLogHistogram          bool
	NormalizeHistogram       bool
	ApplyContrastEnhancement bool
}

func NewProcessingEngine() *ProcessingEngine {
	return &ProcessingEngine{}
}

func (pe *ProcessingEngine) SetOriginalImage(data *ImageData) {
	pe.originalImage = data
}

func (pe *ProcessingEngine) GetOriginalImage() *ImageData {
	return pe.originalImage
}

func (pe *ProcessingEngine) GetProcessedImage() *ImageData {
	return pe.processedImage
}

func (pe *ProcessingEngine) ProcessImage(params *OtsuParameters) (*ImageData, *BinaryImageMetrics, error) {
	if pe.originalImage == nil {
		return nil, nil, fmt.Errorf("no original image loaded")
	}

	gray := pe.convertToGrayscale(pe.originalImage.Mat)
	defer gray.Close()

	working := gray
	if params.GaussianPreprocessing {
		blurred := pe.applyGaussianBlur(gray, params.SmoothingStrength)
		defer blurred.Close()
		working = blurred
	}

	if params.ApplyContrastEnhancement {
		enhanced := pe.applyCLAHE(working)
		defer enhanced.Close()
		working = enhanced
	}

	neighborhood := pe.calculateNeighborhoodMean(working, params.WindowSize)
	defer neighborhood.Close()

	histBins := params.HistogramBins
	if histBins == 0 {
		histBins = pe.calculateHistogramBins(working)
	}

	histogram := pe.build2DHistogram(working, neighborhood, histBins)

	if params.UseLogHistogram {
		pe.applyLogScaling(histogram)
	}

	if params.NormalizeHistogram {
		pe.normalizeHistogram(histogram)
	}

	if params.SmoothingStrength > 0 {
		pe.smoothHistogram(histogram, params.SmoothingStrength)
	}

	threshold := pe.find2DOtsuThreshold(histogram)

	result := pe.applyThreshold(working, neighborhood, threshold, histBins)
	defer result.Close()

	if params.NoiseRobustness {
		cleaned := pe.applyNoiseReduction(result)
		defer cleaned.Close()
		result = cleaned
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

	return processedData, metrics, nil
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

func (pe *ProcessingEngine) applyCLAHE(src gocv.Mat) gocv.Mat {
	dst := gocv.NewMat()
	clahe := gocv.NewCLAHE()
	defer clahe.Close()
	clahe.Apply(src, &dst)
	return dst
}

func (pe *ProcessingEngine) calculateNeighborhoodMean(src gocv.Mat, windowSize int) gocv.Mat {
	dst := gocv.NewMat()
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Point{X: windowSize, Y: windowSize})
	defer kernel.Close()
	gocv.MorphologyEx(src, &dst, gocv.MorphOpen, kernel)
	return dst
}

func (pe *ProcessingEngine) calculateHistogramBins(src gocv.Mat) int {
	rows := src.Rows()
	cols := src.Cols()
	totalPixels := rows * cols

	baseBins := 64
	if totalPixels > 1000000 {
		baseBins = 128
	} else if totalPixels < 100000 {
		baseBins = 32
	}

	return baseBins
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

func (pe *ProcessingEngine) find2DOtsuThreshold(histogram [][]float64) [2]float64 {
	histBins := len(histogram)
	bestThreshold := [2]float64{float64(histBins) / 2.0, float64(histBins) / 2.0}
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

	subPixelStep := 0.1

	for t := 1.0; t < float64(histBins-1); t += subPixelStep {
		variance := pe.calculateVarianceForThresholds(histogram, t, t, totalSum, totalCount)
		if variance > maxVariance {
			maxVariance = variance
			bestThreshold = [2]float64{t, t}
		}
	}

	return bestThreshold
}

func (pe *ProcessingEngine) calculateVarianceForThresholds(histogram [][]float64, t1, t2, totalSum, totalCount float64) float64 {
	histBins := len(histogram)
	var w0, w1, sum0, sum1 float64

	t1Int := int(t1)
	t2Int := int(t2)

	for i := 0; i <= t1Int; i++ {
		for j := 0; j <= t2Int; j++ {
			if float64(i) <= t1 && float64(j) <= t2 {
				weight := histogram[i][j]
				w0 += weight
				sum0 += float64(i*histBins+j) * weight
			}
		}
	}

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

func (pe *ProcessingEngine) applyThreshold(src, neighborhood gocv.Mat, threshold [2]float64, histBins int) gocv.Mat {
	result := gocv.NewMatWithSize(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)
	binScale := float64(histBins-1) / 255.0

	for y := 0; y < src.Rows(); y++ {
		for x := 0; x < src.Cols(); x++ {
			pixelValue := src.GetUCharAt(y, x)
			neighValue := neighborhood.GetUCharAt(y, x)

			pixelBin := float64(pixelValue) * binScale
			neighBin := float64(neighValue) * binScale

			if pixelBin > threshold[0] && neighBin > threshold[1] {
				result.SetUCharAt(y, x, 255)
			} else {
				result.SetUCharAt(y, x, 0)
			}
		}
	}

	return result
}

func (pe *ProcessingEngine) applyNoiseReduction(src gocv.Mat) gocv.Mat {
	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 3, Y: 3})
	defer kernel.Close()

	opened := gocv.NewMat()
	defer opened.Close()
	gocv.MorphologyEx(src, &opened, gocv.MorphOpen, kernel)

	result := gocv.NewMat()
	gocv.MorphologyEx(opened, &result, gocv.MorphClose, kernel)

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
