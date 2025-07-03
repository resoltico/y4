package pipeline

import (
	"bufio"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path/filepath"
	"strings"

	"otsu-obliterator/internal/logger"
	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/opencv/safe"

	"fyne.io/fyne/v2"
	"gocv.io/x/gocv"
)

type imageLoader struct {
	memoryManager *memory.Manager
	logger        logger.Logger
}

func (l *imageLoader) LoadFromReader(reader fyne.URIReadCloser) (*ImageData, error) {
	originalURI := reader.URI()
	uriExtension := strings.ToLower(filepath.Ext(originalURI.Path()))

	bufferedReader := bufio.NewReader(reader)
	data, err := io.ReadAll(bufferedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	return l.LoadFromBytes(data, uriExtension)
}

func (l *imageLoader) LoadFromBytes(data []byte, format string) (*ImageData, error) {
	img, standardLibFormat, err := image.Decode(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image with standard library: %w", err)
	}

	mat, err := gocv.IMDecode(data, gocv.IMReadColor)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image with OpenCV: %w", err)
	}
	defer mat.Close()

	safeMat, err := safe.NewMatFromMatWithTracker(mat, l.memoryManager, "loaded_image")
	if err != nil {
		return nil, fmt.Errorf("failed to create safe Mat: %w", err)
	}

	actualFormat := l.determineActualFormat(format, standardLibFormat)
	bounds := img.Bounds()

	imageData := &ImageData{
		Image:    img,
		Mat:      safeMat,
		Width:    bounds.Dx(),
		Height:   bounds.Dy(),
		Channels: safeMat.Channels(),
		Format:   actualFormat,
	}

	l.logger.Info("ImageLoader", "image loaded", map[string]interface{}{
		"width":    imageData.Width,
		"height":   imageData.Height,
		"channels": imageData.Channels,
		"format":   actualFormat,
	})

	return imageData, nil
}

func (l *imageLoader) determineActualFormat(uriExtension, stdLibFormat string) string {
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
