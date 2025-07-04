package main

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type Application struct {
	fyneApp fyne.App
	window  fyne.Window
	ctx     context.Context
	cancel  context.CancelFunc

	toolbar     *Toolbar
	imageViewer *ImageViewer
	parameters  *ParameterPanel
	processing  *ProcessingEngine
}

func NewApplication(fyneApp fyne.App, window fyne.Window, ctx context.Context, cancel context.CancelFunc) *Application {
	app := &Application{
		fyneApp: fyneApp,
		window:  window,
		ctx:     ctx,
		cancel:  cancel,
	}

	app.processing = NewProcessingEngine()
	app.toolbar = NewToolbar(app)
	app.imageViewer = NewImageViewer()
	app.parameters = NewParameterPanel(app)

	app.setupWindow()
	app.setupMenu()

	return app
}

func (a *Application) setupWindow() {
	a.window.Resize(fyne.NewSize(1200, 800))
	a.window.CenterOnScreen()
	a.window.SetMaster()

	content := container.NewVBox(
		a.toolbar.GetContainer(),
		a.imageViewer.GetContainer(),
		a.parameters.GetContainer(),
	)

	a.window.SetContent(content)

	a.window.SetCloseIntercept(func() {
		a.cancel()
		a.window.Close()
	})
}

func (a *Application) setupMenu() {
	aboutAction := func() {
		a.showAbout()
	}

	fileMenu := fyne.NewMenu("File")
	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("About", aboutAction),
	)

	mainMenu := fyne.NewMainMenu(fileMenu, helpMenu)
	a.window.SetMainMenu(mainMenu)
}

func (a *Application) showAbout() {
	metadata := a.fyneApp.Metadata()

	nameLabel := widget.NewLabel(metadata.Name)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}
	nameLabel.Alignment = fyne.TextAlignCenter

	versionLabel := widget.NewLabel("Version " + metadata.Version)
	versionLabel.Alignment = fyne.TextAlignCenter

	authorLabel := widget.NewLabel("Author: Ervins Strauhmanis")
	authorLabel.Alignment = fyne.TextAlignCenter

	licenseLabel := widget.NewLabel("License: MIT")
	licenseLabel.Alignment = fyne.TextAlignCenter

	content := container.NewVBox(
		widget.NewSeparator(),
		nameLabel,
		versionLabel,
		widget.NewSeparator(),
		authorLabel,
		licenseLabel,
		widget.NewSeparator(),
	)

	dialog.NewCustom("About", "Close", content, a.window).Show()
}

func (a *Application) Run() error {
	a.window.ShowAndRun()
	return nil
}
