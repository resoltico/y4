package main

import (
	"fmt"
	"image/jpeg"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

type FileSaveMenu struct {
	window fyne.Window
}

type ImageFormat struct {
	Name      string
	Extension string
	MimeType  string
}

var SupportedFormats = []ImageFormat{
	{"PNG", ".png", "image/png"},
	{"JPEG", ".jpg", "image/jpeg"},
	{"JPEG", ".jpeg", "image/jpeg"},
}

func NewFileSaveMenu(window fyne.Window) *FileSaveMenu {
	return &FileSaveMenu{
		window: window,
	}
}

func (fsm *FileSaveMenu) ShowSaveDialog(imageData *ImageData, callback func(fyne.URIWriteCloser, error)) {
	if imageData == nil {
		callback(nil, fmt.Errorf("no image data to save"))
		return
	}

	formatSelect := widget.NewSelect([]string{"PNG", "JPEG"}, nil)
	formatSelect.SetSelected("PNG")

	qualitySlider := widget.NewSlider(1, 100)
	qualitySlider.SetValue(95)
	qualitySlider.Hide()

	qualityLabel := widget.NewLabel("Quality: 95")
	qualityLabel.Hide()

	formatSelect.OnChanged = func(format string) {
		if format == "JPEG" {
			qualitySlider.Show()
			qualityLabel.Show()
		} else {
			qualitySlider.Hide()
			qualityLabel.Hide()
		}
	}

	qualitySlider.OnChanged = func(value float64) {
		qualityLabel.SetText(fmt.Sprintf("Quality: %.0f", value))
	}

	formatContainer := container.NewVBox(
		widget.NewLabel("Format:"),
		formatSelect,
		qualityLabel,
		qualitySlider,
	)

	customDialog := dialog.NewCustomConfirm(
		"Save Options",
		"Save",
		"Cancel",
		formatContainer,
		func(save bool) {
			if save {
				fsm.showFileSaveDialogWithFormat(imageData, formatSelect.Selected, int(qualitySlider.Value), callback)
			}
		},
		fsm.window,
	)

	customDialog.Show()
}

func (fsm *FileSaveMenu) showFileSaveDialogWithFormat(imageData *ImageData, format string, quality int, callback func(fyne.URIWriteCloser, error)) {
	var extension string
	switch format {
	case "JPEG":
		extension = ".jpg"
	case "PNG":
		extension = ".png"
	default:
		extension = ".png"
	}

	saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			callback(nil, err)
			return
		}

		originalURI := writer.URI().String()
		if !strings.HasSuffix(strings.ToLower(originalURI), extension) {
			newURI := originalURI + extension
			uri := storage.NewURI(newURI)
			if newWriter, err := storage.Writer(uri); err == nil {
				writer.Close()
				writer = newWriter
			}
		}

		if format == "JPEG" {
			fsm.saveAsJPEG(writer, imageData, quality, callback)
		} else {
			SaveImageToWriter(writer, imageData)
			callback(writer, nil)
		}
	}, fsm.window)

	saveDialog.SetFileName("processed_image" + extension)
	saveDialog.Show()
}

func (fsm *FileSaveMenu) saveAsJPEG(writer fyne.URIWriteCloser, imageData *ImageData, quality int, callback func(fyne.URIWriteCloser, error)) {
	defer writer.Close()

	img := imageData.Image
	jpegOptions := &jpeg.Options{Quality: quality}

	if err := jpeg.Encode(writer, img, jpegOptions); err != nil {
		callback(nil, fmt.Errorf("encode JPEG: %w", err))
		return
	}

	callback(writer, nil)
}

func (fsm *FileSaveMenu) GetSupportedFormats() []ImageFormat {
	return SupportedFormats
}
