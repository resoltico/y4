package main

import (
	"strconv"

	"fyne.io/fyne/v2"
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
	pp.createBasicWidgets()
	pp.createAlgorithmWidgets()
	pp.setupBasicHandlers()
	pp.buildLayout()
	pp.setDefaults()
	return pp
}

func (pp *ParameterPanel) createBasicWidgets() {
	// Basic parameters
	pp.windowSizeSlider = widget.NewSlider(3, 21)
	pp.windowSizeSlider.Step = 2
	pp.windowSizeLabel = widget.NewLabel("Window Size: 7")

	pp.histBinsSlider = widget.NewSlider(0, 256)
	pp.histBinsLabel = widget.NewLabel("Histogram Bins: Auto")

	pp.smoothingStrengthSlider = widget.NewSlider(0.0, 5.0)
	pp.smoothingStrengthLabel = widget.NewLabel("Smoothing Strength: 1.0")
}

func (pp *ParameterPanel) createAlgorithmWidgets() {
	// Algorithm toggles
	pp.edgePreservationCheck = widget.NewCheck("Edge Preservation", nil)
	pp.noiseRobustnessCheck = widget.NewCheck("Noise Robustness", nil)
	pp.gaussianPreprocessCheck = widget.NewCheck("Gaussian Preprocessing", nil)
	pp.useLogCheck = widget.NewCheck("Use Log Histogram", nil)
	pp.normalizeCheck = widget.NewCheck("Normalize Histogram", nil)
	pp.contrastCheck = widget.NewCheck("Adaptive Contrast Enhancement", nil)
}

func (pp *ParameterPanel) setupBasicHandlers() {
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
}
