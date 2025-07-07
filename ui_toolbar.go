package main

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Toolbar struct {
	app       *Application
	container *fyne.Container

	loadButton    *widget.Button
	saveButton    *widget.Button
	processButton *widget.Button
	resetButton   *widget.Button
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
	t.fileSaveMenu = NewFileSaveMenu(app.window)
	t.buildThemedLayout()

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

	t.resetButton = widget.NewButton("Reset", t.handleReset)
}

func (t *Toolbar) buildThemedLayout() {
	buttonsSection := container.NewHBox(
		t.loadButton,
		t.saveButton,
		t.processButton,
		t.resetButton,
	)

	// Create background rectangle using theme header color
	bgRect := canvas.NewRectangle(theme.Color(theme.ColorNameHeaderBackground))

	// Use border layout to overlay buttons on themed background
	t.container = container.NewBorder(
		nil, nil, buttonsSection, nil, bgRect,
	)
}

func (t *Toolbar) handleReset() {
	t.app.parameters.resetToDefaults()
}

func (t *Toolbar) GetContainer() *fyne.Container {
	return t.container
}
