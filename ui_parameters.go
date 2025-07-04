package main

import (
	"context"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ParameterPanel struct {
	app       *Application
	container *fyne.Container
	binder    *ParameterBinder

	// Basic parameter widgets
	windowSizeSlider        *widget.Slider
	windowSizeLabel         *widget.Label
	histBinsSlider          *widget.Slider
	histBinsLabel           *widget.Label
	smoothingStrengthSlider *widget.Slider
	smoothingStrengthLabel  *widget.Label

	// Algorithm toggles
	edgePreservationCheck   *widget.Check
	noiseRobustnessCheck    *widget.Check
	gaussianPreprocessCheck *widget.Check
	useLogCheck             *widget.Check
	normalizeCheck          *widget.Check
	contrastCheck           *widget.Check

	// Processing method selection
	processingMethodSelect *widget.Select
	pyramidLevelsSlider    *widget.Slider
	pyramidLevelsLabel     *widget.Label
	regionGridSlider       *widget.Slider
	regionGridLabel        *widget.Label

	// Neighborhood parameters
	neighborhoodTypeSelect *widget.Select
	adaptiveWindowCheck    *widget.Check

	// Interpolation and post-processing
	interpolationSelect   *widget.Select
	morphPostProcessCheck *widget.Check
	morphKernelSlider     *widget.Slider
	morphKernelLabel      *widget.Label

	// Advanced preprocessing
	homomorphicCheck     *widget.Check
	anisotropicCheck     *widget.Check
	diffusionIterSlider  *widget.Slider
	diffusionIterLabel   *widget.Label
	diffusionKappaSlider *widget.Slider
	diffusionKappaLabel  *widget.Label

	lastProcessTime  time.Time
	processingCtx    context.Context
	processingCancel context.CancelFunc
}

func NewParameterPanel(app *Application) *ParameterPanel {
	pp := &ParameterPanel{
		app: app,
	}

	pp.binder = NewParameterBinder(app)
	pp.createWidgets()
	pp.setupBindings()
	pp.buildLayout()
	pp.setupParameterListener()

	return pp
}

func (pp *ParameterPanel) createWidgets() {
	// Basic parameters
	pp.windowSizeSlider = widget.NewSlider(3, 21)
	pp.windowSizeSlider.Step = 2
	pp.windowSizeLabel = widget.NewLabel("Window Size: 7")

	pp.histBinsSlider = widget.NewSlider(0, 256)
	pp.histBinsLabel = widget.NewLabel("Histogram Bins: Auto")

	pp.smoothingStrengthSlider = widget.NewSlider(0.0, 5.0)
	pp.smoothingStrengthLabel = widget.NewLabel("Smoothing Strength: 1.0")

	// Algorithm toggles
	pp.edgePreservationCheck = widget.NewCheck("Edge Preservation", nil)
	pp.noiseRobustnessCheck = widget.NewCheck("Noise Robustness", nil)
	pp.gaussianPreprocessCheck = widget.NewCheck("Gaussian Preprocessing", nil)
	pp.useLogCheck = widget.NewCheck("Use Log Histogram", nil)
	pp.normalizeCheck = widget.NewCheck("Normalize Histogram", nil)
	pp.contrastCheck = widget.NewCheck("Adaptive Contrast Enhancement", nil)

	// Processing method selection
	pp.processingMethodSelect = widget.NewSelect([]string{
		"Single Scale",
		"Multi-Scale Pyramid",
		"Region Adaptive",
	}, nil)

	pp.pyramidLevelsSlider = widget.NewSlider(1, 5)
	pp.pyramidLevelsLabel = widget.NewLabel("Pyramid Levels: 3")

	pp.regionGridSlider = widget.NewSlider(32, 256)
	pp.regionGridLabel = widget.NewLabel("Region Grid Size: 64")

	// Neighborhood parameters
	pp.neighborhoodTypeSelect = widget.NewSelect([]string{
		"Rectangular",
		"Circular",
		"Distance Weighted",
	}, nil)

	pp.adaptiveWindowCheck = widget.NewCheck("Adaptive Window Sizing", nil)

	// Interpolation and post-processing
	pp.interpolationSelect = widget.NewSelect([]string{
		"Nearest",
		"Bilinear",
		"Bicubic",
	}, nil)

	pp.morphPostProcessCheck = widget.NewCheck("Morphological Post-Processing", nil)
	pp.morphKernelSlider = widget.NewSlider(1, 7)
	pp.morphKernelLabel = widget.NewLabel("Morphological Kernel: 3")

	// Advanced preprocessing
	pp.homomorphicCheck = widget.NewCheck("Homomorphic Filtering", nil)
	pp.anisotropicCheck = widget.NewCheck("Anisotropic Diffusion", nil)
	pp.diffusionIterSlider = widget.NewSlider(1, 20)
	pp.diffusionIterLabel = widget.NewLabel("Diffusion Iterations: 5")
	pp.diffusionKappaSlider = widget.NewSlider(10.0, 100.0)
	pp.diffusionKappaLabel = widget.NewLabel("Diffusion Kappa: 30.0")
}

func (pp *ParameterPanel) setupBindings() {
	// Bind sliders with proper data binding
	pp.binder.BindSlider(pp.windowSizeSlider, "WindowSize", pp.windowSizeLabel, FormatWindowSize)
	pp.binder.BindSlider(pp.histBinsSlider, "HistogramBins", pp.histBinsLabel, FormatHistogramBins)
	pp.binder.BindSlider(pp.smoothingStrengthSlider, "SmoothingStrength", pp.smoothingStrengthLabel, FormatSmoothingStrength)
	pp.binder.BindSlider(pp.pyramidLevelsSlider, "PyramidLevels", pp.pyramidLevelsLabel, FormatPyramidLevels)
	pp.binder.BindSlider(pp.regionGridSlider, "RegionGridSize", pp.regionGridLabel, FormatRegionGrid)
	pp.binder.BindSlider(pp.morphKernelSlider, "MorphologicalKernelSize", pp.morphKernelLabel, FormatMorphKernel)
	pp.binder.BindSlider(pp.diffusionIterSlider, "DiffusionIterations", pp.diffusionIterLabel, FormatDiffusionIter)
	pp.binder.BindSlider(pp.diffusionKappaSlider, "DiffusionKappa", pp.diffusionKappaLabel, FormatDiffusionKappa)

	// Bind checkboxes
	pp.binder.BindCheck(pp.edgePreservationCheck, "EdgePreservation")
	pp.binder.BindCheck(pp.noiseRobustnessCheck, "NoiseRobustness")
	pp.binder.BindCheck(pp.gaussianPreprocessCheck, "GaussianPreprocessing")
	pp.binder.BindCheck(pp.useLogCheck, "UseLogHistogram")
	pp.binder.BindCheck(pp.normalizeCheck, "NormalizeHistogram")
	pp.binder.BindCheck(pp.contrastCheck, "ApplyContrastEnhancement")
	pp.binder.BindCheck(pp.adaptiveWindowCheck, "AdaptiveWindowSizing")
	pp.binder.BindCheck(pp.morphPostProcessCheck, "MorphologicalPostProcess")
	pp.binder.BindCheck(pp.homomorphicCheck, "HomomorphicFiltering")
	pp.binder.BindCheck(pp.anisotropicCheck, "AnisotropicDiffusion")

	// Bind selects
	pp.binder.BindSelect(pp.neighborhoodTypeSelect, "NeighborhoodType")
	pp.binder.BindSelect(pp.interpolationSelect, "InterpolationMethod")

	// Processing method requires special handling
	pp.processingMethodSelect.OnChanged = func(method string) {
		pp.handleProcessingMethodChange(method)
	}

	// Morphological and diffusion controls visibility
	pp.morphPostProcessCheck.OnChanged = func(checked bool) {
		pp.toggleMorphologicalControls(checked)
	}

	pp.anisotropicCheck.OnChanged = func(checked bool) {
		pp.toggleDiffusionControls(checked)
	}
}

func (pp *ParameterPanel) setupParameterListener() {
	pp.binder.AddParameterListener(func(params *OtsuParameters) {
		// Debounce parameter changes to avoid excessive processing
		now := time.Now()
		if now.Sub(pp.lastProcessTime) < 100*time.Millisecond {
			return
		}
		pp.lastProcessTime = now

		// Cancel any ongoing processing
		if pp.processingCancel != nil {
			pp.processingCancel()
		}

		// Start new processing with timeout
		pp.processingCtx, pp.processingCancel = context.WithCancel(context.Background())
		go pp.triggerProcessing(pp.processingCtx, params)
	})
}

func (pp *ParameterPanel) triggerProcessing(ctx context.Context, params *OtsuParameters) {
	// Wait a bit to batch rapid parameter changes
	select {
	case <-time.After(200 * time.Millisecond):
		// Proceed with processing
	case <-ctx.Done():
		return // Cancelled
	}

	// Check if we still have an image to process
	if pp.app.processing.GetOriginalImage() == nil {
		return
	}

	// Trigger processing through the toolbar
	fyne.Do(func() {
		pp.app.toolbar.handleProcessImageWithParams(params)
	})
}

func (pp *ParameterPanel) handleProcessingMethodChange(method string) {
	pp.binder.UpdateProcessingMethodDependencies(method)
	pp.updateMethodVisibility(method)

	// Debug trace parameter change
	DebugTraceParam("ProcessingMethod", pp.getLastProcessingMethod(), method)
}

func (pp *ParameterPanel) getLastProcessingMethod() string {
	params := pp.binder.GetCurrentParameters()
	if params.MultiScaleProcessing {
		return "Multi-Scale Pyramid"
	} else if params.RegionAdaptiveThresholding {
		return "Region Adaptive"
	}
	return "Single Scale"
}

func (pp *ParameterPanel) updateMethodVisibility(method string) {
	switch method {
	case "Multi-Scale Pyramid":
		pp.pyramidLevelsSlider.Show()
		pp.pyramidLevelsLabel.Show()
		pp.regionGridSlider.Hide()
		pp.regionGridLabel.Hide()
	case "Region Adaptive":
		pp.pyramidLevelsSlider.Hide()
		pp.pyramidLevelsLabel.Hide()
		pp.regionGridSlider.Show()
		pp.regionGridLabel.Show()
	default:
		pp.pyramidLevelsSlider.Hide()
		pp.pyramidLevelsLabel.Hide()
		pp.regionGridSlider.Hide()
		pp.regionGridLabel.Hide()
	}
}

func (pp *ParameterPanel) toggleMorphologicalControls(enabled bool) {
	if enabled {
		pp.morphKernelSlider.Show()
		pp.morphKernelLabel.Show()
	} else {
		pp.morphKernelSlider.Hide()
		pp.morphKernelLabel.Hide()
	}
}

func (pp *ParameterPanel) toggleDiffusionControls(enabled bool) {
	if enabled {
		pp.diffusionIterSlider.Show()
		pp.diffusionIterLabel.Show()
		pp.diffusionKappaSlider.Show()
		pp.diffusionKappaLabel.Show()
	} else {
		pp.diffusionIterSlider.Hide()
		pp.diffusionIterLabel.Hide()
		pp.diffusionKappaSlider.Hide()
		pp.diffusionKappaLabel.Hide()
	}
}

func (pp *ParameterPanel) buildLayout() {
	// Basic parameters section
	basicSection := container.NewVBox(
		widget.NewLabel("Basic Parameters"),
		widget.NewSeparator(),
		container.NewHBox(
			container.NewVBox(pp.windowSizeLabel, pp.windowSizeSlider),
			container.NewVBox(pp.histBinsLabel, pp.histBinsSlider),
			container.NewVBox(pp.smoothingStrengthLabel, pp.smoothingStrengthSlider),
		),
	)

	// Processing method section
	methodSection := container.NewVBox(
		widget.NewLabel("Processing Method"),
		widget.NewSeparator(),
		pp.processingMethodSelect,
		container.NewHBox(
			container.NewVBox(pp.pyramidLevelsLabel, pp.pyramidLevelsSlider),
			container.NewVBox(pp.regionGridLabel, pp.regionGridSlider),
		),
	)

	// Neighborhood parameters section
	neighborhoodSection := container.NewVBox(
		widget.NewLabel("Neighborhood Parameters"),
		widget.NewSeparator(),
		pp.neighborhoodTypeSelect,
		pp.adaptiveWindowCheck,
	)

	// Algorithm options section
	algorithmSection := container.NewVBox(
		widget.NewLabel("Algorithm Options"),
		widget.NewSeparator(),
		container.NewHBox(
			pp.edgePreservationCheck,
			pp.noiseRobustnessCheck,
		),
		container.NewHBox(
			pp.gaussianPreprocessCheck,
			pp.useLogCheck,
		),
		container.NewHBox(
			pp.normalizeCheck,
			pp.contrastCheck,
		),
	)

	// Preprocessing section
	preprocessingSection := container.NewVBox(
		widget.NewLabel("Advanced Preprocessing"),
		widget.NewSeparator(),
		container.NewHBox(
			pp.homomorphicCheck,
			pp.anisotropicCheck,
		),
		container.NewHBox(
			container.NewVBox(pp.diffusionIterLabel, pp.diffusionIterSlider),
			container.NewVBox(pp.diffusionKappaLabel, pp.diffusionKappaSlider),
		),
	)

	// Post-processing section
	postprocessingSection := container.NewVBox(
		widget.NewLabel("Post-Processing"),
		widget.NewSeparator(),
		pp.interpolationSelect,
		pp.morphPostProcessCheck,
		container.NewVBox(pp.morphKernelLabel, pp.morphKernelSlider),
	)

	// Create scrollable container with all sections
	allSections := container.NewVBox(
		basicSection,
		methodSection,
		neighborhoodSection,
		algorithmSection,
		preprocessingSection,
		postprocessingSection,
	)

	scroll := container.NewScroll(allSections)
	scroll.SetMinSize(fyne.NewSize(800, 400))
	pp.container = container.NewBorder(nil, nil, nil, nil, scroll)

	pp.setInitialValues()
}

func (pp *ParameterPanel) setInitialValues() {
	// Set default values through the binder
	pp.binder.SetFieldValue("WindowSize", 7)
	pp.binder.SetFieldValue("HistogramBins", 0)
	pp.binder.SetFieldValue("SmoothingStrength", 1.0)
	pp.binder.SetFieldValue("PyramidLevels", 3)
	pp.binder.SetFieldValue("RegionGridSize", 64)
	pp.binder.SetFieldValue("MorphologicalKernelSize", 3)
	pp.binder.SetFieldValue("DiffusionIterations", 5)
	pp.binder.SetFieldValue("DiffusionKappa", 30.0)

	pp.processingMethodSelect.SetSelected("Single Scale")
	pp.binder.SetFieldValue("NeighborhoodType", "Rectangular")
	pp.binder.SetFieldValue("InterpolationMethod", "Bilinear")

	pp.binder.SetFieldValue("GaussianPreprocessing", true)
	pp.binder.SetFieldValue("NormalizeHistogram", true)

	// Hide method-specific controls initially
	pp.toggleMorphologicalControls(false)
	pp.toggleDiffusionControls(false)
	pp.updateMethodVisibility("Single Scale")
}

func (pp *ParameterPanel) GetCurrentParameters() *OtsuParameters {
	return pp.binder.GetCurrentParameters()
}

func (pp *ParameterPanel) GetContainer() *fyne.Container {
	return pp.container
}
