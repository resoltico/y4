package main

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func createSectionHeader(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.TextStyle = fyne.TextStyle{Bold: true}
	return label
}

type ImageViewer struct {
	splitContainer *container.Split
	originalImage  *canvas.Image
	processedImage *canvas.Image
}

func NewImageViewer() *ImageViewer {
	iv := &ImageViewer{}
	iv.createImages()
	iv.buildLayout()
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

func (iv *ImageViewer) buildLayout() {
	originalContainer := container.NewBorder(
		createSectionHeader("Original"),
		nil, nil, nil,
		iv.originalImage,
	)

	processedContainer := container.NewBorder(
		createSectionHeader("Processed"),
		nil, nil, nil,
		iv.processedImage,
	)

	// Split container handles its own sizing - no wrapper needed
	iv.splitContainer = container.NewHSplit(originalContainer, processedContainer)
	iv.splitContainer.SetOffset(0.5)

	debugSystem := GetDebugSystem()
	DebugLogUILayout(debugSystem.logger, "split_container", iv.splitContainer)
	DebugLogImageSizing(debugSystem.logger, "original_image", iv.originalImage)
	DebugLogImageSizing(debugSystem.logger, "processed_image", iv.processedImage)
}

func (iv *ImageViewer) SetOriginalImage(img image.Image) {
	iv.originalImage.Image = img
	iv.originalImage.Refresh()

	debugSystem := GetDebugSystem()
	DebugLogImageSizing(debugSystem.logger, "original_after_set", iv.originalImage)
	DebugLogLayoutRefresh(debugSystem.logger, "image_viewer", iv.splitContainer, "original_image_set")
}

func (iv *ImageViewer) SetProcessedImage(img image.Image) {
	iv.processedImage.Image = img
	iv.processedImage.Refresh()

	debugSystem := GetDebugSystem()
	DebugLogImageSizing(debugSystem.logger, "processed_after_set", iv.processedImage)
	DebugLogLayoutRefresh(debugSystem.logger, "image_viewer", iv.splitContainer, "processed_image_set")
}

func (iv *ImageViewer) GetContainer() *fyne.Container {
	// Use border layout to ensure split container fills available space
	return container.NewBorder(nil, nil, nil, nil, iv.splitContainer)
}
