package main

import (
	"context"
	"image/color"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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

	// Create toolbar background
	toolbarBg := canvas.NewRectangle(color.RGBA{R: 250, G: 249, B: 245, A: 255})
	toolbarContainer := container.NewStack(toolbarBg, a.toolbar.GetContainer())

	content := container.NewVBox(
		a.imageViewer.GetContainer(),
		toolbarContainer,
		a.parameters.GetContainer(),
	)

	a.window.SetContent(content)

	// Comprehensive debug logging
	debugSystem := GetDebugSystem()
	DebugLogWindowSizing(debugSystem.logger, a.window, "after_setup")
	DebugLogContainerHierarchy(debugSystem.logger, "main_window", content, 0)
	DebugLogUILayout(debugSystem.logger, "main_vbox", content)
	DebugLogUILayout(debugSystem.logger, "image_viewer_in_window", a.imageViewer.GetContainer())

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
