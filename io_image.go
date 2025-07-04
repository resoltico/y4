package main

import (
	"bufio"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
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

	actualFormat := determineImageFormat(uriExtension, standardLibFormat)
	bounds := img.Bounds()

	imageData := &ImageData{
		Image:    img,
		Mat:      mat,
		Width:    bounds.Dx(),
		Height:   bounds.Dy(),
		Channels: mat.Channels(),
		Format:   actualFormat,
	}

	return imageData, nil
}

func SaveImageToWriter(writer fyne.URIWriteCloser, imageData *ImageData) error {
	if imageData == nil {
		return fmt.Errorf("no image data to save")
	}

	img, ok := imageData.Image.(image.Image)
	if !ok {
		return fmt.Errorf("image data contains invalid image")
	}

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
	case ".tiff", ".tif":
		return "tiff"
	case ".jpg", ".jpeg":
		return "jpeg"
	case ".png":
		return "png"
	case ".bmp":
		return "bmp"
	case ".gif":
		return "gif"
	case ".webp":
		return "webp"
	default:
		if stdLibFormat != "" {
			return stdLibFormat
		}
		return "unknown"
	}
}
