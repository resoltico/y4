package main

import (
	"bufio"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"gocv.io/x/gocv"
)

func LoadImageFromReader(reader fyne.URIReadCloser) (*ImageData, error) {
	originalURI := reader.URI()
	uriExtension := strings.ToLower(filepath.Ext(originalURI.Path()))

	bufferedReader := bufio.NewReader(reader)
	data, err := io.ReadAll(bufferedReader)
	if err != nil {
		return nil, fmt.Errorf("read image data: %w", err)
	}

	img, standardLibFormat, err := image.Decode(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("decode image with standard library: %w", err)
	}

	// Use IMReadUnchanged to preserve alpha channels
	mat, err := gocv.IMDecode(data, gocv.IMReadUnchanged)
	if err != nil {
		return nil, fmt.Errorf("decode image with OpenCV: %w", err)
	}

	// Handle transparency by compositing with white background
	if mat.Channels() == 4 {
		composited := compositeTransparencyWithWhiteBackground(mat)
		mat.Close()
		mat = composited
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if err := validateImageDimensions(width, height, "image loading"); err != nil {
		mat.Close()
		return nil, fmt.Errorf("image dimension validation: %w", err)
	}

	if err := validateMatForMetrics(mat, "loaded image"); err != nil {
		mat.Close()
		return nil, fmt.Errorf("loaded image matrix validation: %w", err)
	}

	actualFormat := determineImageFormat(uriExtension, standardLibFormat)

	imageData := &ImageData{
		Image:    img,
		Mat:      mat,
		Width:    width,
		Height:   height,
		Channels: mat.Channels(),
		Format:   actualFormat,
	}

	return imageData, nil
}

func compositeTransparencyWithWhiteBackground(src gocv.Mat) gocv.Mat {
	if src.Channels() != 4 {
		return src.Clone()
	}

	rows, cols := src.Rows(), src.Cols()

	// Split BGRA channels
	channels := gocv.Split(src)
	defer func() {
		for _, ch := range channels {
			ch.Close()
		}
	}()

	if len(channels) != 4 {
		return src.Clone()
	}

	bgr := channels[:3]  // B, G, R channels
	alpha := channels[3] // Alpha channel

	// Create result matrix for BGR output
	result := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC3)

	// Debug logging for transparency processing
	debugSystem := GetDebugSystem()
	debugSystem.logger.Debug("transparency composition starting",
		"rows", rows, "cols", cols,
		"alpha_channels", len(channels),
		"bgr_channels", len(bgr))

	// Create result by merging blended BGR channels
	blendedChannels := make([]gocv.Mat, 3)
	for c := 0; c < 3; c++ {
		blendedChannels[c] = gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)
		defer blendedChannels[c].Close()
	}

	// Perform alpha blending per pixel
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			alphaValue := float64(alpha.GetUCharAt(y, x)) / 255.0
			invAlpha := 1.0 - alphaValue

			// Blend each BGR channel with white (255)
			for c := 0; c < 3; c++ {
				foreground := float64(bgr[c].GetUCharAt(y, x))
				blended := (foreground * alphaValue) + (255.0 * invAlpha)

				if blended > 255.0 {
					blended = 255.0
				}
				if blended < 0.0 {
					blended = 0.0
				}

				// Write to individual channel matrix
				blendedChannels[c].SetUCharAt(y, x, uint8(blended))
			}
		}
	}

	// Merge blended channels into final result
	gocv.Merge(blendedChannels, &result)

	debugSystem.logger.Debug("transparency composition completed",
		"result_channels", result.Channels(),
		"result_type", result.Type())

	return result
}

func SaveImageToWriter(writer fyne.URIWriteCloser, imageData *ImageData) error {
	if imageData == nil {
		return fmt.Errorf("no image data to save")
	}

	// Validate image data before saving
	if err := validateImageDimensions(imageData.Width, imageData.Height, "image saving"); err != nil {
		return fmt.Errorf("save image validation: %w", err)
	}

	if err := validateMatForMetrics(imageData.Mat, "save image"); err != nil {
		return fmt.Errorf("save image matrix validation: %w", err)
	}

	img := imageData.Image
	ext := strings.ToLower(writer.URI().Extension())

	var err error
	switch ext {
	case ".jpg", ".jpeg":
		err = jpeg.Encode(writer, img, &jpeg.Options{Quality: 95})
	case ".png":
		err = png.Encode(writer, img)
	default:
		err = png.Encode(writer, img)
	}

	if err != nil {
		return fmt.Errorf("encode image: %w", err)
	}

	return nil
}

func determineImageFormat(uriExtension, stdLibFormat string) string {
	switch uriExtension {
	case ".jpg", ".jpeg":
		return "jpeg"
	case ".png":
		return "png"
	default:
		if stdLibFormat != "" {
			return stdLibFormat
		}
		return "unknown"
	}
}
