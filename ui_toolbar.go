package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type Toolbar struct {
	app       *Application
	container *fyne.Container

	loadButton    *widget.Button
	saveButton    *widget.Button
	processButton *widget.Button
	statusLabel   *widget.Label
	metricsLabel  *widget.Label
}

func NewToolbar(app *Application) *Toolbar {
	t := &Toolbar{
		app: app,
	}

	t.createButtons()
	t.createLabels()
	t.buildLayout()

	return t
}

func (t *Toolbar) createButtons() {
	t.loadButton = widget.NewButton("Load Image", t.handleLoadImage)
	t.loadButton.Importance = widget.HighImportance

	t.saveButton = widget.NewButton("Save Result", t.handleSaveImage)
	t.saveButton.Importance = widget.HighImportance
	t.saveButton.Disable()

	t.processButton = widget.NewButton("Process", t.handleProcessImage)
	t.processButton.Importance = widget.HighImportance
	t.processButton.Disable()
}

func (t *Toolbar) createLabels() {
	t.statusLabel = widget.NewLabel("Ready")
	t.metricsLabel = widget.NewLabel("No metrics available")
}

func (t *Toolbar) buildLayout() {
	leftSection := container.NewHBox(t.loadButton, t.saveButton)
	centerSection := container.NewHBox(t.processButton)
	rightSection := container.NewVBox(t.statusLabel, t.metricsLabel)

	t.container = container.NewBorder(
		nil, nil, leftSection, rightSection, centerSection,
	)
}

func (t *Toolbar) handleLoadImage() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()

		t.SetStatus("Loading image...")

		imageData, loadErr := LoadImageFromReader(reader)
		if loadErr != nil {
			dialog.ShowError(loadErr, t.app.window)
			t.SetStatus("Load failed")
			return
		}

		fyne.Do(func() {
			t.app.imageViewer.SetOriginalImage(imageData.Image)
			t.app.processing.SetOriginalImage(imageData)
			t.processButton.Enable()
			t.SetStatus("Image loaded")
		})
	}, t.app.window)
}

func (t *Toolbar) handleSaveImage() {
	processedData := t.app.processing.GetProcessedImage()
	if processedData == nil {
		return
	}

	dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}
		defer writer.Close()

		t.SetStatus("Saving image...")

		saveErr := SaveImageToWriter(writer, processedData)
		if saveErr != nil {
			dialog.ShowError(saveErr, t.app.window)
			t.SetStatus("Save failed")
			return
		}

		t.SetStatus("Image saved")
	}, t.app.window)
}

func (t *Toolbar) handleProcessImage() {
	originalData := t.app.processing.GetOriginalImage()
	if originalData == nil {
		return
	}

	t.SetStatus("Processing...")
	t.processButton.Disable()

	go func() {
		params := t.app.parameters.GetParameters()
		result, metrics, err := t.app.processing.ProcessImage(params)

		fyne.Do(func() {
			t.processButton.Enable()

			if err != nil {
				dialog.ShowError(err, t.app.window)
				t.SetStatus("Processing failed")
				return
			}

			t.app.imageViewer.SetProcessedImage(result.Image)
			t.SetStatus("Processing complete")
			t.SetMetrics(metrics)
			t.saveButton.Enable()
		})
	}()
}

func (t *Toolbar) SetStatus(status string) {
	t.statusLabel.SetText(status)
}

func (t *Toolbar) SetMetrics(metrics *BinaryImageMetrics) {
	if metrics == nil {
		t.metricsLabel.SetText("No metrics available")
		return
	}

	text := "F: %.3f | pF: %.3f | DRD: %.3f | MPM: %.3f | NRM: %.3f | PBC: %.3f"
	t.metricsLabel.SetText(
		fmt.Sprintf(text,
			metrics.FMeasure(),
			metrics.PseudoFMeasure(),
			metrics.DRD(),
			metrics.MPM(),
			metrics.NRM(),
			metrics.PBC(),
		),
	)
}

func (t *Toolbar) GetContainer() *fyne.Container {
	return t.container
}
