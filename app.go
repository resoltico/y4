package main

import (
	"context"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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

	debugSystem *DebugSystem
}

func NewApplication(fyneApp fyne.App, window fyne.Window, ctx context.Context, cancel context.CancelFunc) *Application {
	app := &Application{
		fyneApp: fyneApp,
		window:  window,
		ctx:     ctx,
		cancel:  cancel,
	}

	app.debugSystem = InitDebugSystem(DebugConfig{
		LogLevel:      slog.LevelDebug,
		EnableTracing: true,
		EnableMonitor: true,
		ConsoleOutput: true,
	})

	app.processing = NewProcessingEngine()
	app.imageViewer = NewImageViewer()
	app.parameters = NewParameterPanel(app)
	app.toolbar = NewToolbar(app)

	app.setupWindow()
	app.setupMenu()

	app.debugSystem.logger.Info("application initialized",
		"debug_enabled", true,
		"tracing_enabled", true,
		"monitoring_enabled", true,
	)

	return app
}

func (a *Application) setupWindow() {
	a.window.Resize(fyne.NewSize(1400, 900))
	a.window.CenterOnScreen()
	a.window.SetMaster()

	mainContent := container.NewHSplit(
		a.imageViewer.GetContainer(),
		a.parameters.GetContainer(),
	)
	mainContent.SetOffset(0.7)

	content := container.NewVBox(
		a.toolbar.GetContainer(),
		widget.NewSeparator(),
		mainContent,
	)

	a.window.SetContent(content)

	a.window.SetCloseIntercept(func() {
		a.cleanup()
		a.window.Close()
	})
}

func (a *Application) cleanup() {
	if a.toolbar != nil {
		a.toolbar.CancelCurrentProcessing()
	}

	if a.debugSystem != nil {
		a.debugSystem.DumpSystemState()
		a.debugSystem.Close()
	}

	a.cancel()

	a.debugSystem.logger.Info("application cleanup completed")
}

func (a *Application) setupMenu() {
	fileMenu := fyne.NewMenu("File")
	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("About", a.showAbout),
		fyne.NewMenuItem("Debug Info", a.showDebugInfo),
	)

	mainMenu := fyne.NewMainMenu(fileMenu, helpMenu)
	a.window.SetMainMenu(mainMenu)
}

func (a *Application) ShowAndRun() {
	a.window.ShowAndRun()
}
