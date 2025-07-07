package main

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

// Real Gaussian pyramid using exact OpenCV 5x5 kernel and pixel rejection
func (pe *ProcessingEngine) processMultiScalePyramid(src gocv.Mat, params *OtsuParameters) gocv.Mat {
	if err := validateMatForMetrics(src, "multi-scale processing"); err != nil {
		return gocv.NewMat()
	}

	levels := params.PyramidLevels
	if levels <= 0 {
		levels = 3
	}

	actualLevels := levels
	for i := 1; i <= levels; i++ {
		testRows := src.Rows() / (1 << i)
		testCols := src.Cols() / (1 << i)
		if testRows < 64 || testCols < 64 {
			actualLevels = i - 1
			break
		}
	}
	levels = actualLevels

	debugSystem := GetDebugSystem()
	debugSystem.logger.Debug("pyramid levels calculated",
		"requested_levels", params.PyramidLevels,
		"actual_levels", levels,
		"source_size", fmt.Sprintf("%dx%d", src.Cols(), src.Rows()))

	if levels < 1 {
		debugSystem.logger.Warn("insufficient levels for pyramid, using single scale")
		return pe.processSingleScale(src, params)
	}

	// Build Gaussian pyramid using exact OpenCV method
	pyramid := make([]gocv.Mat, levels+1)
	pyramid[0] = src.Clone()

	for i := 1; i <= levels; i++ {
		pyramid[i] = pe.pyrDownExact(pyramid[i-1])
		if pyramid[i].Empty() {
			debugSystem.logger.Error("pyramid construction failed", "level", i)
			for j := 1; j < i; j++ {
				pyramid[j].Close()
			}
			return pe.processSingleScale(src, params)
		}
	}

	defer func() {
		for i := 1; i <= levels; i++ {
			pyramid[i].Close()
		}
	}()

	// Process each level with scale-appropriate parameters
	results := make([]gocv.Mat, levels+1)
	for i := 0; i <= levels; i++ {
		scaleParams := *params
		scaleParams.MultiScaleProcessing = false
		scaleParams.WindowSize = max(3, params.WindowSize/(1<<i))
		if scaleParams.WindowSize%2 == 0 {
			scaleParams.WindowSize++
		}
		
		if scaleParams.HistogramBins > 0 {
			scaleParams.HistogramBins = max(32, params.HistogramBins/(1<<i))
		}
		
		results[i] = pe.processSingleScale(pyramid[i], &scaleParams)
	}

	defer func() {
		for i := 1; i <= levels; i++ {
			results[i].Close()
		}
	}()

	// Reconstruct using Laplacian pyramid approach
	reconstructed := results[levels].Clone()
	defer reconstructed.Close()

	for i := levels - 1; i >= 0; i-- {
		upsampled := pe.pyrUpExact(reconstructed, results[i].Rows(), results[i].Cols())
		if upsampled.Empty() {
			debugSystem.logger.Error("upsampling failed", "level", i)
			continue
		}

		combined := gocv.NewMat()
		weight := 0.7 // Favor finer scale details
		gocv.AddWeighted(results[i], weight, upsampled, 1.0-weight, 0, &combined)
		
		reconstructed.Close()
		upsampled.Close()
		reconstructed = combined
	}

	if err := validateMatForMetrics(reconstructed, "pyramid result"); err != nil {
		return gocv.NewMat()
	}

	return reconstructed.Clone()
}

// Create exact 5x5 Gaussian kernel matching OpenCV coefficients
func (pe *ProcessingEngine) createPyramidKernel() gocv.Mat {
	// Exact OpenCV pyramid kernel: [1,4,6,4,1] pattern normalized by 1/256
	kernelData := []float32{
		1.0/256, 4.0/256, 6.0/256, 4.0/256, 1.0/256,
		4.0/256, 16.0/256, 24.0/256, 16.0/256, 4.0/256,
		6.0/256, 24.0/256, 36.0/256, 24.0/256, 6.0/256,
		4.0/256, 16.0/256, 24.0/256, 16.0/256, 4.0/256,
		1.0/256, 4.0/256, 6.0/256, 4.0/256, 1.0/256,
	}
	
	kernel, err := gocv.NewMatFromBytes(5, 5, gocv.MatTypeCV32F, (*[80]byte)(kernelData)[:])
	if err != nil {
		return gocv.NewMat()
	}
	
	return kernel
}

// Real pyrDown: exact 5x5 kernel + even pixel rejection
func (pe *ProcessingEngine) pyrDownExact(src gocv.Mat) gocv.Mat {
	if err := validateMatForMetrics(src, "pyrDown input"); err != nil {
		return gocv.NewMat()
	}

	// Step 1: Apply exact 5x5 Gaussian kernel
	kernel := pe.createPyramidKernel()
	defer kernel.Close()
	
	blurred := gocv.NewMat()
	defer blurred.Close()
	
	err := gocv.Filter2D(src, &blurred, -1, kernel, image.Point{X: -1, Y: -1}, 0, gocv.BorderDefault)
	if err != nil {
		return gocv.NewMat()
	}

	// Step 2: Downsample by rejecting even rows and columns
	rows := blurred.Rows()
	cols := blurred.Cols()
	newRows := (rows + 1) / 2
	newCols := (cols + 1) / 2

	result := gocv.NewMatWithSize(newRows, newCols, src.Type())

	// Extract odd-indexed pixels using row/column operations
	for y := 0; y < newRows; y++ {
		for x := 0; x < newCols; x++ {
			srcY := y * 2
			srcX := x * 2
			
			if srcY < rows && srcX < cols {
				switch src.Type() {
				case gocv.MatTypeCV8UC1:
					val := blurred.GetUCharAt(srcY, srcX)
					result.SetUCharAt(y, x, val)
				case gocv.MatTypeCV8UC3:
					for c := 0; c < 3; c++ {
						val := blurred.GetUCharAt3(srcY, srcX, c)
						result.SetUCharAt3(y, x, c, val)
					}
				}
			}
		}
	}

	return result
}

// Real pyrUp: insert zeros + blur with kernel×4
func (pe *ProcessingEngine) pyrUpExact(src gocv.Mat, targetRows, targetCols int) gocv.Mat {
	if err := validateMatForMetrics(src, "pyrUp input"); err != nil {
		return gocv.NewMat()
	}

	rows := src.Rows()
	cols := src.Cols()

	// Step 1: Insert zero rows and columns (upsample to double size)
	upsampled := gocv.NewMatWithSize(rows*2, cols*2, src.Type())
	zeros := gocv.NewScalar(0, 0, 0, 0)
	upsampled.SetTo(zeros)

	// Copy original pixels to odd positions
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			dstY := y * 2
			dstX := x * 2
			
			switch src.Type() {
			case gocv.MatTypeCV8UC1:
				val := src.GetUCharAt(y, x)
				upsampled.SetUCharAt(dstY, dstX, val)
			case gocv.MatTypeCV8UC3:
				for c := 0; c < 3; c++ {
					val := src.GetUCharAt3(y, x, c)
					upsampled.SetUCharAt3(dstY, dstX, c, val)
				}
			}
		}
	}

	// Step 2: Apply kernel×4 (OpenCV multiplies coefficients by 4 for upsampling)
	kernelData := []float32{
		4.0/256, 16.0/256, 24.0/256, 16.0/256, 4.0/256,
		16.0/256, 64.0/256, 96.0/256, 64.0/256, 16.0/256,
		24.0/256, 96.0/256, 144.0/256, 96.0/256, 24.0/256,
		16.0/256, 64.0/256, 96.0/256, 64.0/256, 16.0/256,
		4.0/256, 16.0/256, 24.0/256, 16.0/256, 4.0/256,
	}
	
	kernel, err := gocv.NewMatFromBytes(5, 5, gocv.MatTypeCV32F, (*[80]byte)(kernelData)[:])
	if err != nil {
		upsampled.Close()
		return gocv.NewMat()
	}
	defer kernel.Close()

	blurred := gocv.NewMat()
	defer blurred.Close()
	
	err = gocv.Filter2D(upsampled, &blurred, -1, kernel, image.Point{X: -1, Y: -1}, 0, gocv.BorderDefault)
	upsampled.Close()
	
	if err != nil {
		return gocv.NewMat()
	}

	// Step 3: Resize to exact target dimensions if needed
	if blurred.Rows() != targetRows || blurred.Cols() != targetCols {
		result := gocv.NewMat()
		gocv.Resize(blurred, &result, image.Point{X: targetCols, Y: targetRows}, 0, 0, gocv.InterpolationLinear)
		return result
	}

	return blurred.Clone()
}