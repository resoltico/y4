package main

import (
	"image"
	"math"

	"gocv.io/x/gocv"
)

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
