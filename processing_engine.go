package main

import (
	"fmt"
	"image"
	"image/color"

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
		return nil, nil, fmt.Errorf("no original image loaded")
	}

	if err := validateMatForMetrics(pe.originalImage.Mat, "original image processing"); err != nil {
		return nil, nil, fmt.Errorf("original image validation: %w", err)
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

	metrics, err := CalculateBinaryMetrics(gray, result)
	if err != nil {
		return processedData, nil, fmt.Errorf("metrics calculation: %w", err)
	}

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
