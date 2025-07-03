package bridge

import (
	"fmt"
	"image"
	"image/color"

	"otsu-obliterator/internal/opencv/safe"
)

func MatToImage(mat *safe.Mat) (image.Image, error) {
	if err := safe.ValidateMatForOperation(mat, "MatToImage"); err != nil {
		return nil, err
	}

	rows := mat.Rows()
	cols := mat.Cols()
	channels := mat.Channels()

	if rows == 0 || cols == 0 {
		return nil, fmt.Errorf("Mat has zero dimensions: %dx%d", cols, rows)
	}

	switch channels {
	case 1:
		return matToGray(mat, rows, cols)
	case 3:
		return matToRGBA(mat, rows, cols)
	case 4:
		return matToRGBAWithAlpha(mat, rows, cols)
	default:
		return nil, fmt.Errorf("unsupported number of channels: %d", channels)
	}
}

func matToGray(mat *safe.Mat, rows, cols int) (*image.Gray, error) {
	img := image.NewGray(image.Rect(0, 0, cols, rows))

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			value, err := mat.GetUCharAt(y, x)
			if err != nil {
				return nil, fmt.Errorf("failed to get pixel at (%d,%d): %w", x, y, err)
			}
			img.SetGray(x, y, color.Gray{Y: value})
		}
	}

	return img, nil
}

func matToRGBA(mat *safe.Mat, rows, cols int) (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, cols, rows))

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			b, err := mat.GetUCharAt3(y, x, 0)
			if err != nil {
				return nil, fmt.Errorf("failed to get B channel at (%d,%d): %w", x, y, err)
			}

			g, err := mat.GetUCharAt3(y, x, 1)
			if err != nil {
				return nil, fmt.Errorf("failed to get G channel at (%d,%d): %w", x, y, err)
			}

			r, err := mat.GetUCharAt3(y, x, 2)
			if err != nil {
				return nil, fmt.Errorf("failed to get R channel at (%d,%d): %w", x, y, err)
			}

			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	return img, nil
}

func matToRGBAWithAlpha(mat *safe.Mat, rows, cols int) (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, cols, rows))

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			b, err := mat.GetUCharAt3(y, x, 0)
			if err != nil {
				return nil, fmt.Errorf("failed to get B channel at (%d,%d): %w", x, y, err)
			}

			g, err := mat.GetUCharAt3(y, x, 1)
			if err != nil {
				return nil, fmt.Errorf("failed to get G channel at (%d,%d): %w", x, y, err)
			}

			r, err := mat.GetUCharAt3(y, x, 2)
			if err != nil {
				return nil, fmt.Errorf("failed to get R channel at (%d,%d): %w", x, y, err)
			}

			a, err := mat.GetUCharAt3(y, x, 3)
			if err != nil {
				return nil, fmt.Errorf("failed to get A channel at (%d,%d): %w", x, y, err)
			}

			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}

	return img, nil
}

func ImageToMat(img image.Image) (*safe.Mat, error) {
	if img == nil {
		return nil, fmt.Errorf("input image is nil")
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	switch typedImg := img.(type) {
	case *image.Gray:
		return grayToMat(typedImg, width, height)
	case *image.RGBA:
		return rgbaToMat(typedImg, width, height)
	case *image.NRGBA:
		return nrgbaToMat(typedImg, width, height)
	default:
		return genericImageToMat(img, width, height)
	}
}

func grayToMat(img *image.Gray, width, height int) (*safe.Mat, error) {
	mat, err := safe.NewMat(height, width, 0) // CV_8UC1
	if err != nil {
		return nil, err
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gray := img.GrayAt(x, y)
			if err := mat.SetUCharAt(y, x, gray.Y); err != nil {
				mat.Close()
				return nil, fmt.Errorf("failed to set pixel at (%d,%d): %w", x, y, err)
			}
		}
	}

	return mat, nil
}

func rgbaToMat(img *image.RGBA, width, height int) (*safe.Mat, error) {
	mat, err := safe.NewMat(height, width, 16) // CV_8UC3
	if err != nil {
		return nil, err
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			rgba := img.RGBAAt(x, y)

			if err := mat.SetUCharAt3(y, x, 0, rgba.B); err != nil {
				mat.Close()
				return nil, fmt.Errorf("failed to set B channel at (%d,%d): %w", x, y, err)
			}

			if err := mat.SetUCharAt3(y, x, 1, rgba.G); err != nil {
				mat.Close()
				return nil, fmt.Errorf("failed to set G channel at (%d,%d): %w", x, y, err)
			}

			if err := mat.SetUCharAt3(y, x, 2, rgba.R); err != nil {
				mat.Close()
				return nil, fmt.Errorf("failed to set R channel at (%d,%d): %w", x, y, err)
			}
		}
	}

	return mat, nil
}

func nrgbaToMat(img *image.NRGBA, width, height int) (*safe.Mat, error) {
	mat, err := safe.NewMat(height, width, 16) // CV_8UC3
	if err != nil {
		return nil, err
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			nrgba := img.NRGBAAt(x, y)

			if err := mat.SetUCharAt3(y, x, 0, nrgba.B); err != nil {
				mat.Close()
				return nil, fmt.Errorf("failed to set B channel at (%d,%d): %w", x, y, err)
			}

			if err := mat.SetUCharAt3(y, x, 1, nrgba.G); err != nil {
				mat.Close()
				return nil, fmt.Errorf("failed to set G channel at (%d,%d): %w", x, y, err)
			}

			if err := mat.SetUCharAt3(y, x, 2, nrgba.R); err != nil {
				mat.Close()
				return nil, fmt.Errorf("failed to set R channel at (%d,%d): %w", x, y, err)
			}
		}
	}

	return mat, nil
}

func genericImageToMat(img image.Image, width, height int) (*safe.Mat, error) {
	mat, err := safe.NewMat(height, width, 16) // CV_8UC3
	if err != nil {
		return nil, err
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x, y).RGBA()

			// Convert from 16-bit to 8-bit
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			if err := mat.SetUCharAt3(y, x, 0, b8); err != nil {
				mat.Close()
				return nil, fmt.Errorf("failed to set B channel at (%d,%d): %w", x, y, err)
			}

			if err := mat.SetUCharAt3(y, x, 1, g8); err != nil {
				mat.Close()
				return nil, fmt.Errorf("failed to set G channel at (%d,%d): %w", x, y, err)
			}

			if err := mat.SetUCharAt3(y, x, 2, r8); err != nil {
				mat.Close()
				return nil, fmt.Errorf("failed to set R channel at (%d,%d): %w", x, y, err)
			}
		}
	}

	return mat, nil
}
