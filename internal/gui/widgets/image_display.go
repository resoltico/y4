package widgets

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	ImageAreaWidth  = 500
	ImageAreaHeight = 400
)

type ImageDisplay struct {
	container     fyne.CanvasObject
	originalImage *canvas.Image
	previewImage  *canvas.Image
	splitView     *container.Split
}

func NewImageDisplay() *ImageDisplay {
	display := &ImageDisplay{}
	display.createComponents()
	display.setupLayout()
	return display
}

func (id *ImageDisplay) createComponents() {
	id.originalImage = canvas.NewImageFromImage(nil)
	id.originalImage.FillMode = canvas.ImageFillContain
	id.originalImage.ScaleMode = canvas.ImageScaleSmooth
	id.originalImage.SetMinSize(fyne.NewSize(ImageAreaWidth, ImageAreaHeight))

	id.previewImage = canvas.NewImageFromImage(nil)
	id.previewImage.FillMode = canvas.ImageFillContain
	id.previewImage.ScaleMode = canvas.ImageScaleSmooth
	id.previewImage.SetMinSize(fyne.NewSize(ImageAreaWidth, ImageAreaHeight))
}

func (id *ImageDisplay) setupLayout() {
	originalContainer := container.NewBorder(
		widget.NewRichTextFromMarkdown("**Original**"),
		nil, nil, nil,
		id.originalImage,
	)

	previewContainer := container.NewBorder(
		widget.NewRichTextFromMarkdown("**Preview**"),
		nil, nil, nil,
		id.previewImage,
	)

	id.splitView = container.NewHSplit(originalContainer, previewContainer)
	id.splitView.SetOffset(0.5)
	id.container = id.splitView
}

func (id *ImageDisplay) GetContainer() fyne.CanvasObject {
	return id.container
}

func (id *ImageDisplay) SetOriginalImage(img image.Image) {
	id.originalImage.Image = img
	id.originalImage.Refresh()
	id.container.Refresh()
}

func (id *ImageDisplay) SetPreviewImage(img image.Image) {
	id.previewImage.Image = img
	id.previewImage.Refresh()
}

func (id *ImageDisplay) GetSplitView() *container.Split {
	return id.splitView
}
