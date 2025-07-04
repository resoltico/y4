package main

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ParameterPanel struct {
	app       *Application
	container *fyne.Container

	// Basic parameters
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
}

func NewParameterPanel(app *Application) *ParameterPanel {
	pp := &ParameterPanel{
		app: app,
	}
	pp.createWidgets()
	pp.setupHandlers()
	pp.buildLayout()
	pp.setDefaults()
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

func (pp *ParameterPanel) setupHandlers() {
	pp.windowSizeSlider.OnChanged = func(value float64) {
		intValue := int(value)
		if intValue%2 == 0 {
			intValue++
		}
		pp.windowSizeLabel.SetText("Window Size: " + strconv.Itoa(intValue))
	}

	pp.histBinsSlider.OnChanged = func(value float64) {
		intValue := int(value)
		if intValue == 0 {
			pp.histBinsLabel.SetText("Histogram Bins: Auto")
		} else {
			pp.histBinsLabel.SetText("Histogram Bins: " + strconv.Itoa(intValue))
		}
	}

	pp.smoothingStrengthSlider.OnChanged = func(value float64) {
		pp.smoothingStrengthLabel.SetText("Smoothing Strength: " + strconv.FormatFloat(value, 'f', 1, 64))
	}

	pp.pyramidLevelsSlider.OnChanged = func(value float64) {
		intValue := int(value)
		pp.pyramidLevelsLabel.SetText("Pyramid Levels: " + strconv.Itoa(intValue))
	}

	pp.regionGridSlider.OnChanged = func(value float64) {
		intValue := int(value)
		pp.regionGridLabel.SetText("Region Grid Size: " + strconv.Itoa(intValue))
	}

	pp.morphKernelSlider.OnChanged = func(value float64) {
		intValue := int(value)
		if intValue%2 == 0 {
			intValue++
		}
		pp.morphKernelLabel.SetText("Morphological Kernel: " + strconv.Itoa(intValue))
	}

	pp.diffusionIterSlider.OnChanged = func(value float64) {
		intValue := int(value)
		pp.diffusionIterLabel.SetText("Diffusion Iterations: " + strconv.Itoa(intValue))
	}

	pp.diffusionKappaSlider.OnChanged = func(value float64) {
		pp.diffusionKappaLabel.SetText("Diffusion Kappa: " + strconv.FormatFloat(value, 'f', 1, 64))
	}

	pp.processingMethodSelect.OnChanged = func(method string) {
		pp.updateMethodSpecificControls(method)
	}

	pp.morphPostProcessCheck.OnChanged = func(checked bool) {
		if checked {
			pp.morphKernelSlider.Show()
			pp.morphKernelLabel.Show()
		} else {
			pp.morphKernelSlider.Hide()
			pp.morphKernelLabel.Hide()
		}
	}

	pp.anisotropicCheck.OnChanged = func(checked bool) {
		if checked {
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
}

func (pp *ParameterPanel) updateMethodSpecificControls(method string) {
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
}

func (pp *ParameterPanel) setDefaults() {
	pp.windowSizeSlider.SetValue(7)
	pp.histBinsSlider.SetValue(0)
	pp.smoothingStrengthSlider.SetValue(1.0)
	pp.pyramidLevelsSlider.SetValue(3)
	pp.regionGridSlider.SetValue(64)
	pp.morphKernelSlider.SetValue(3)
	pp.diffusionIterSlider.SetValue(5)
	pp.diffusionKappaSlider.SetValue(30.0)

	pp.processingMethodSelect.SetSelected("Single Scale")
	pp.neighborhoodTypeSelect.SetSelected("Rectangular")
	pp.interpolationSelect.SetSelected("Bilinear")

	pp.gaussianPreprocessCheck.SetChecked(true)
	pp.normalizeCheck.SetChecked(true)

	// Hide method-specific controls initially
	pp.pyramidLevelsSlider.Hide()
	pp.pyramidLevelsLabel.Hide()
	pp.regionGridSlider.Hide()
	pp.regionGridLabel.Hide()

	// Hide morphological controls initially
	pp.morphKernelSlider.Hide()
	pp.morphKernelLabel.Hide()

	// Hide diffusion controls initially
	pp.diffusionIterSlider.Hide()
	pp.diffusionIterLabel.Hide()
	pp.diffusionKappaSlider.Hide()
	pp.diffusionKappaLabel.Hide()
}

func (pp *ParameterPanel) GetParameters() *OtsuParameters {
	windowSize := int(pp.windowSizeSlider.Value)
	if windowSize%2 == 0 {
		windowSize++
	}

	histBins := int(pp.histBinsSlider.Value)
	pyramidLevels := int(pp.pyramidLevelsSlider.Value)
	regionGridSize := int(pp.regionGridSlider.Value)
	morphKernelSize := int(pp.morphKernelSlider.Value)
	if morphKernelSize%2 == 0 {
		morphKernelSize++
	}
	diffusionIterations := int(pp.diffusionIterSlider.Value)

	multiScaleProcessing := pp.processingMethodSelect.Selected == "Multi-Scale Pyramid"
	regionAdaptiveThresholding := pp.processingMethodSelect.Selected == "Region Adaptive"

	return &OtsuParameters{
		WindowSize:                 windowSize,
		HistogramBins:              histBins,
		SmoothingStrength:          pp.smoothingStrengthSlider.Value,
		EdgePreservation:           pp.edgePreservationCheck.Checked,
		NoiseRobustness:            pp.noiseRobustnessCheck.Checked,
		GaussianPreprocessing:      pp.gaussianPreprocessCheck.Checked,
		UseLogHistogram:            pp.useLogCheck.Checked,
		NormalizeHistogram:         pp.normalizeCheck.Checked,
		ApplyContrastEnhancement:   pp.contrastCheck.Checked,
		AdaptiveWindowSizing:       pp.adaptiveWindowCheck.Checked,
		MultiScaleProcessing:       multiScaleProcessing,
		PyramidLevels:              pyramidLevels,
		NeighborhoodType:           pp.neighborhoodTypeSelect.Selected,
		InterpolationMethod:        pp.interpolationSelect.Selected,
		MorphologicalPostProcess:   pp.morphPostProcessCheck.Checked,
		MorphologicalKernelSize:    morphKernelSize,
		HomomorphicFiltering:       pp.homomorphicCheck.Checked,
		AnisotropicDiffusion:       pp.anisotropicCheck.Checked,
		DiffusionIterations:        diffusionIterations,
		DiffusionKappa:             pp.diffusionKappaSlider.Value,
		RegionAdaptiveThresholding: regionAdaptiveThresholding,
		RegionGridSize:             regionGridSize,
	}
}

func (pp *ParameterPanel) GetContainer() *fyne.Container {
	return pp.container
}
