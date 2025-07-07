package main

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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
	detailsLabel  *widget.Label
	fileSaveMenu  *FileSaveMenu

	processingInProgress bool
	currentProcessingCtx context.Context
	cancelProcessing     context.CancelFunc
}

func NewToolbar(app *Application) *Toolbar {
	t := &Toolbar{
		app: app,
	}

	t.createButtons()
	t.createLabels()
	t.fileSaveMenu = NewFileSaveMenu(app.window)
	t.buildLayout()

	return t
}

func (t *Toolbar) createButtons() {
	t.loadButton = widget.NewButton("Load", t.handleLoadImage)
	t.loadButton.Importance = widget.HighImportance

	t.saveButton = widget.NewButton("Save", t.handleSaveImage)
	t.saveButton.Importance = widget.HighImportance
	t.saveButton.Disable()

	t.processButton = widget.NewButton("Process", t.handleProcessImage)
	t.processButton.Importance = widget.HighImportance
	t.processButton.Disable()
}

func (t *Toolbar) createLabels() {
	t.statusLabel = widget.NewLabel("Ready")
	t.metricsLabel = widget.NewLabel("No metrics available")
	t.detailsLabel = widget.NewLabel("Load an image to begin processing")
}

func (t *Toolbar) buildLayout() {
	buttonsSection := container.NewHBox(
		t.loadButton,
		t.saveButton,
		t.processButton,
	)

	metricsSection := container.NewVBox(
		t.statusLabel,
		t.metricsLabel,
		t.detailsLabel,
	)

	t.container = container.NewBorder(
		nil, nil, buttonsSection, metricsSection, nil,
	)
}

func (t *Toolbar) SetStatus(status string) {
	t.statusLabel.SetText("Status: " + status)
}

func (t *Toolbar) SetDetails(details string) {
	t.detailsLabel.SetText(details)
}

func (t *Toolbar) GetContainer() *fyne.Container {
	return t.container
}
