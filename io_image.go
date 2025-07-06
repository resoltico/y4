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

	mat, err := gocv.IMDecode(data, gocv.IMReadColor)
	if err != nil {
		return nil, fmt.Errorf("decode image with OpenCV: %w", err)
	}

	// Validate loaded image dimensions and matrix
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
