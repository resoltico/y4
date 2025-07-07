package main

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

// Real Gaussian pyramid using 5x5 kernel and proper downsampling
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

	// Build Gaussian pyramid using proper downsampling
	pyramid := make([]gocv.Mat, levels+1)
	pyramid[0] = src.Clone()

	for i := 1; i <= levels; i++ {
		pyramid[i] = pe.pyrDownProper(pyramid[i-1])
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
		upsampled := pe.pyrUpProper(reconstructed, results[i].Rows(), results[i].Cols())
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

// Create 5x5 Gaussian kernel using manual construction
func (pe *ProcessingEngine) createPyramidKernel() gocv.Mat {
	// OpenCV pyramid kernel coefficients
	coeffs := []float32{
		1.0 / 256, 4.0 / 256, 6.0 / 256, 4.0 / 256, 1.0 / 256,
		4.0 / 256, 16.0 / 256, 24.0 / 256, 16.0 / 256, 4.0 / 256,
		6.0 / 256, 24.0 / 256, 36.0 / 256, 24.0 / 256, 6.0 / 256,
		4.0 / 256, 16.0 / 256, 24.0 / 256, 16.0 / 256, 4.0 / 256,
		1.0 / 256, 4.0 / 256, 6.0 / 256, 4.0 / 256, 1.0 / 256,
	}

	// Create kernel manually to avoid unsafe.Pointer issues
	kernel := gocv.NewMatWithSize(5, 5, gocv.MatTypeCV32F)
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			kernel.SetFloatAt(i, j, coeffs[i*5+j])
		}
	}
	return kernel
}

// Proper pyrDown: 5x5 Gaussian blur + efficient subsampling
func (pe *ProcessingEngine) pyrDownProper(src gocv.Mat) gocv.Mat {
	if err := validateMatForMetrics(src, "pyrDown input"); err != nil {
		return gocv.NewMat()
	}

	// Apply 5x5 Gaussian kernel
	kernel := pe.createPyramidKernel()
	defer kernel.Close()

	blurred := gocv.NewMat()
	defer blurred.Close()

	gocv.Filter2D(src, &blurred, -1, kernel, image.Point{X: -1, Y: -1}, 0, gocv.BorderDefault)

	// Efficient subsampling using matrix slicing
	rows := blurred.Rows()
	cols := blurred.Cols()
	newRows := (rows + 1) / 2
	newCols := (cols + 1) / 2

	result := gocv.NewMatWithSize(newRows, newCols, src.Type())

	// Extract every 2nd row and column using bulk operations
	for y := 0; y < newRows; y++ {
		srcRow := blurred.RowRange(y*2, y*2+1)
		dstRow := result.RowRange(y, y+1)

		// Copy every 2nd column in the row
		for x := 0; x < newCols; x++ {
			srcCol := srcRow.ColRange(x*2, x*2+1)
			dstCol := dstRow.ColRange(x, x+1)
			srcCol.CopyTo(&dstCol)
			srcCol.Close()
			dstCol.Close()
		}

		srcRow.Close()
		dstRow.Close()
	}

	return result
}

// Proper pyrUp: insert zeros + blur with matrix operations
func (pe *ProcessingEngine) pyrUpProper(src gocv.Mat, targetRows, targetCols int) gocv.Mat {
	if err := validateMatForMetrics(src, "pyrUp input"); err != nil {
		return gocv.NewMat()
	}

	rows := src.Rows()
	cols := src.Cols()

	// Upsample by inserting zeros using bulk operations
	upsampled := gocv.NewMatWithSize(rows*2, cols*2, src.Type())
	zeros := gocv.NewScalar(0, 0, 0, 0)
	upsampled.SetTo(zeros)

	// Copy source to even positions using RowRange/ColRange
	for y := 0; y < rows; y++ {
		srcRow := src.RowRange(y, y+1)
		dstRow := upsampled.RowRange(y*2, y*2+1)

		for x := 0; x < cols; x++ {
			srcCol := srcRow.ColRange(x, x+1)
			dstCol := dstRow.ColRange(x*2, x*2+1)
			srcCol.CopyTo(&dstCol)
			srcCol.Close()
			dstCol.Close()
		}

		srcRow.Close()
		dstRow.Close()
	}

	// Apply 4x kernel for upsampling
	coeffs4x := []float32{
		4.0 / 256, 16.0 / 256, 24.0 / 256, 16.0 / 256, 4.0 / 256,
		16.0 / 256, 64.0 / 256, 96.0 / 256, 64.0 / 256, 16.0 / 256,
		24.0 / 256, 96.0 / 256, 144.0 / 256, 96.0 / 256, 24.0 / 256,
		16.0 / 256, 64.0 / 256, 96.0 / 256, 64.0 / 256, 16.0 / 256,
		4.0 / 256, 16.0 / 256, 24.0 / 256, 16.0 / 256, 4.0 / 256,
	}

	kernel := gocv.NewMatWithSize(5, 5, gocv.MatTypeCV32F)
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			kernel.SetFloatAt(i, j, coeffs4x[i*5+j])
		}
	}
	defer kernel.Close()

	blurred := gocv.NewMat()
	defer blurred.Close()

	gocv.Filter2D(upsampled, &blurred, -1, kernel, image.Point{X: -1, Y: -1}, 0, gocv.BorderDefault)
	upsampled.Close()

	// Resize to exact target dimensions if needed
	if blurred.Rows() != targetRows || blurred.Cols() != targetCols {
		result := gocv.NewMat()
		gocv.Resize(blurred, &result, image.Point{X: targetCols, Y: targetRows}, 0, 0, gocv.InterpolationLinear)
		return result
	}

	return blurred.Clone()
}
