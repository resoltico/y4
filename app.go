package main

import (
	"context"
	"log/slog"

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

	debugSystem *DebugSystem
}

func NewApplication(fyneApp fyne.App, window fyne.Window, ctx context.Context, cancel context.CancelFunc) *Application {
	app := &Application{
		fyneApp: fyneApp,
		window:  window,
		ctx:     ctx,
		cancel:  cancel,
	}

	// Initialize debug system first
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

	// Create main split between image viewer and parameters
	mainContent := container.NewHSplit(
		a.imageViewer.GetContainer(),
		a.parameters.GetContainer(),
	)
	mainContent.SetOffset(0.7) // Give more space to image viewer

	// Combine toolbar and main content
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
	// Cancel any ongoing processing
	if a.toolbar != nil {
		a.toolbar.CancelCurrentProcessing()
	}

	// Close debug system
	if a.debugSystem != nil {
		a.debugSystem.DumpSystemState()
		a.debugSystem.Close()
	}

	// Cancel application context
	a.cancel()

	a.debugSystem.logger.Info("application cleanup completed")
}

func (a *Application) setupMenu() {
	aboutAction := func() {
		a.showAbout()
	}

	helpAction := func() {
		a.showHelp()
	}

	debugAction := func() {
		a.showDebugInfo()
	}

	fileMenu := fyne.NewMenu("File")
	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("About", aboutAction),
		fyne.NewMenuItem("User Guide", helpAction),
		fyne.NewMenuItem("Debug Info", debugAction),
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

	descriptionLabel := widget.NewLabel("Advanced 2D Otsu thresholding application with multiple image quality metrics")
	descriptionLabel.Alignment = fyne.TextAlignCenter
	descriptionLabel.Wrapping = fyne.TextWrapWord

	authorLabel := widget.NewLabel("Author: Ervins Strauhmanis")
	authorLabel.Alignment = fyne.TextAlignCenter

	licenseLabel := widget.NewLabel("License: MIT")
	licenseLabel.Alignment = fyne.TextAlignCenter

	featuresLabel := widget.NewLabel("Features: Multi-scale processing, Region-adaptive thresholding, Advanced preprocessing, Comprehensive metrics")
	featuresLabel.Alignment = fyne.TextAlignCenter
	featuresLabel.Wrapping = fyne.TextWrapWord

	techLabel := widget.NewLabel("Built with Go 1.24, Fyne v2.6.1, GoCV v0.41.0")
	techLabel.Alignment = fyne.TextAlignCenter

	content := container.NewVBox(
		widget.NewSeparator(),
		nameLabel,
		versionLabel,
		widget.NewSeparator(),
		descriptionLabel,
		widget.NewSeparator(),
		featuresLabel,
		widget.NewSeparator(),
		techLabel,
		widget.NewSeparator(),
		authorLabel,
		licenseLabel,
		widget.NewSeparator(),
	)

	dialog.NewCustom("About", "Close", content, a.window).Show()
}

func (a *Application) showHelp() {
	helpText := `OTSU OBLITERATOR USER GUIDE

PROCESSING METHODS:
• Single Scale: Standard 2D Otsu thresholding
• Multi-Scale Pyramid: Processes multiple resolution levels
• Region Adaptive: Applies different thresholds to image regions

BASIC PARAMETERS:
• Window Size: Neighborhood size for local statistics (3-21)
• Histogram Bins: Bins for 2D histogram (0=auto, 32-256)
• Smoothing Strength: Gaussian smoothing of histogram (0-5)

NEIGHBORHOOD TYPES:
• Rectangular: Standard square neighborhood
• Circular: Circular neighborhood shape
• Distance Weighted: Weighted by distance from center

PREPROCESSING OPTIONS:
• Gaussian Preprocessing: Blur before processing
• Adaptive Contrast Enhancement: CLAHE contrast improvement
• Homomorphic Filtering: Illumination correction
• Anisotropic Diffusion: Edge-preserving smoothing

QUALITY METRICS:
• F-measure: Standard precision/recall harmonic mean
• Pseudo F-measure: DIBCO standard weighted F-measure
• NRM: Negative Rate Metric for error quantification
• DRD: Distance Reciprocal Distortion for visual quality
• MPM: Morphological Path Misalignment for object accuracy
• BFC: Background/Foreground Contrast analysis
• Skeleton: Skeleton similarity for structural accuracy

POST-PROCESSING:
• Morphological Post-Processing: Opening/closing operations
• Interpolation Method: For scaling operations

TIPS:
• Start with default settings for most images
• Use Multi-Scale for complex documents
• Enable Adaptive Window Sizing for varying text sizes
• Apply Homomorphic Filtering for uneven illumination
• Use Anisotropic Diffusion for noisy images
• Check Debug Info menu for performance analysis`

	helpLabel := widget.NewLabel(helpText)
	helpLabel.Wrapping = fyne.TextWrapWord

	helpScroll := container.NewScroll(helpLabel)
	helpScroll.SetMinSize(fyne.NewSize(600, 500))

	dialog.NewCustom("User Guide", "Close", helpScroll, a.window).Show()
}

func (a *Application) showDebugInfo() {
	if a.debugSystem == nil {
		dialog.ShowInformation("Debug Info", "Debug system not available", a.window)
		return
	}

	a.debugSystem.DumpSystemState()

	debugText := `DEBUG SYSTEM STATUS

The debug system provides monitoring and tracing capabilities.

Use 'go run . 2>&1 | grep -E "(DEBUG|ERROR|WARN)"' to filter logs.`

	debugLabel := widget.NewLabel(debugText)
	debugLabel.Wrapping = fyne.TextWrapWord

	dumpButton := widget.NewButton("Dump Current State", func() {
		a.debugSystem.DumpSystemState()
	})

	content := container.NewVBox(
		debugLabel,
		widget.NewSeparator(),
		dumpButton,
	)

	debugScroll := container.NewScroll(content)
	debugScroll.SetMinSize(fyne.NewSize(600, 500))

	dialog.NewCustom("Debug Information", "Close", debugScroll, a.window).Show()
}

func (a *Application) ShowAndRun() {
	a.window.ShowAndRun()
}
