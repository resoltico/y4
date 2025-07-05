package main

import (
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

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
