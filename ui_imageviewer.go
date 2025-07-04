package main

import (
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
	iv.container = container.NewBorder(nil, nil, nil, nil, iv.splitView)
}

func (iv *ImageViewer) SetOriginalImage(img image.Image) {
	iv.originalImage.Image = img
	iv.originalImage.Refresh()
}

func (iv *ImageViewer) SetProcessedImage(img image.Image) {
	iv.processedImage.Image = img
	iv.processedImage.Refresh()
}

func (iv *ImageViewer) GetContainer() *fyne.Container {
	return iv.container
}
