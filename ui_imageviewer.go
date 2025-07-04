package main

import (
	"fmt"
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ImageViewer struct {
	container      *fyne.Container
	originalImage  *canvas.Image
	processedImage *canvas.Image
	splitView      *container.Split
	viewModeSelect *widget.Select
	zoomSlider     *widget.Slider
	zoomLabel      *widget.Label
}

func NewImageViewer() *ImageViewer {
	iv := &ImageViewer{}
	iv.createImages()
	iv.createControls()
	iv.buildLayout()
	iv.setupHandlers()
	return iv
}

func (iv *ImageViewer) createImages() {
	iv.originalImage = canvas.NewImageFromImage(nil)
	iv.originalImage.FillMode = canvas.ImageFillContain
	iv.originalImage.ScaleMode = canvas.ImageScaleSmooth
	iv.originalImage.SetMinSize(fyne.NewSize(400, 300))

	iv.processedImage = canvas.NewImageFromImage(nil)
	iv.processedImage.FillMode = canvas.ImageFillContain
	iv.processedImage.ScaleMode = canvas.ImageScaleSmooth
	iv.processedImage.SetMinSize(fyne.NewSize(400, 300))
}

func (iv *ImageViewer) createControls() {
	iv.viewModeSelect = widget.NewSelect([]string{
		"Side by Side",
		"Original Only",
		"Processed Only",
		"Overlay Comparison",
	}, nil)
	iv.viewModeSelect.SetSelected("Side by Side")

	iv.zoomSlider = widget.NewSlider(0.1, 3.0)
	iv.zoomSlider.SetValue(1.0)
	iv.zoomSlider.Step = 0.1

	iv.zoomLabel = widget.NewLabel("Zoom: 100%")
}

func (iv *ImageViewer) setupHandlers() {
	iv.viewModeSelect.OnChanged = func(mode string) {
		iv.updateViewMode(mode)
	}

	iv.zoomSlider.OnChanged = func(value float64) {
		iv.zoomLabel.SetText(fmt.Sprintf("Zoom: %.0f%%", value*100))
		iv.updateZoom(value)
	}
}

func (iv *ImageViewer) buildLayout() {
	originalContainer := container.NewBorder(
		widget.NewLabel("Original"),
		nil, nil, nil,
		iv.originalImage,
	)

	processedContainer := container.NewBorder(
		widget.NewLabel("Processed"),
		nil, nil, nil,
		iv.processedImage,
	)

	iv.splitView = container.NewHSplit(originalContainer, processedContainer)
	iv.splitView.SetOffset(0.5)

	controlsContainer := container.NewHBox(
		widget.NewLabel("View:"),
		iv.viewModeSelect,
		widget.NewSeparator(),
		iv.zoomLabel,
		iv.zoomSlider,
	)

	iv.container = container.NewBorder(
		controlsContainer,
		nil, nil, nil,
		iv.splitView,
	)
}

func (iv *ImageViewer) updateViewMode(mode string) {
	switch mode {
	case "Original Only":
		iv.splitView.SetOffset(1.0)
	case "Processed Only":
		iv.splitView.SetOffset(0.0)
	case "Overlay Comparison":
		// For overlay mode, we'll show both images with transparency
		iv.splitView.SetOffset(0.5)
		iv.createOverlayView()
	default: // "Side by Side"
		iv.splitView.SetOffset(0.5)
	}
}

func (iv *ImageViewer) updateZoom(zoomLevel float64) {
	newSize := fyne.NewSize(
		float32(400.0*zoomLevel),
		float32(300.0*zoomLevel),
	)

	iv.originalImage.SetMinSize(newSize)
	iv.processedImage.SetMinSize(newSize)

	iv.originalImage.Refresh()
	iv.processedImage.Refresh()
}

func (iv *ImageViewer) createOverlayView() {
	if iv.originalImage.Image == nil || iv.processedImage.Image == nil {
		return
	}

	// Create a simple overlay by showing both images with visual indicators
	// This is a simplified overlay - for true overlay, we'd need to composite the images
	iv.originalImage.FillMode = canvas.ImageFillOriginal
	iv.processedImage.FillMode = canvas.ImageFillOriginal
}

func (iv *ImageViewer) SetOriginalImage(img image.Image) {
	iv.originalImage.Image = img
	iv.originalImage.Refresh()

	if img != nil {
		// Reset zoom when new image is loaded
		iv.zoomSlider.SetValue(1.0)
		iv.updateZoom(1.0)
	}
}

func (iv *ImageViewer) SetProcessedImage(img image.Image) {
	iv.processedImage.Image = img
	iv.processedImage.Refresh()
}

func (iv *ImageViewer) GetContainer() *fyne.Container {
	return iv.container
}

func (iv *ImageViewer) ResetView() {
	iv.viewModeSelect.SetSelected("Side by Side")
	iv.zoomSlider.SetValue(1.0)
	iv.updateViewMode("Side by Side")
	iv.updateZoom(1.0)
}
