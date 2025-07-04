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

	windowSizeSlider        *widget.Slider
	windowSizeLabel         *widget.Label
	histBinsSlider          *widget.Slider
	histBinsLabel           *widget.Label
	smoothingStrengthSlider *widget.Slider
	smoothingStrengthLabel  *widget.Label
	edgePreservationCheck   *widget.Check
	noiseRobustnessCheck    *widget.Check
	gaussianPreprocessCheck *widget.Check
	useLogCheck             *widget.Check
	normalizeCheck          *widget.Check
	contrastCheck           *widget.Check
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
	pp.windowSizeSlider = widget.NewSlider(3, 21)
	pp.windowSizeSlider.Step = 2
	pp.windowSizeLabel = widget.NewLabel("Window Size: 7")

	pp.histBinsSlider = widget.NewSlider(0, 256)
	pp.histBinsLabel = widget.NewLabel("Histogram Bins: Auto")

	pp.smoothingStrengthSlider = widget.NewSlider(0.0, 5.0)
	pp.smoothingStrengthLabel = widget.NewLabel("Smoothing Strength: 1.0")

	pp.edgePreservationCheck = widget.NewCheck("Edge Preservation (MAOTSU)", nil)
	pp.noiseRobustnessCheck = widget.NewCheck("Noise Robustness", nil)
	pp.gaussianPreprocessCheck = widget.NewCheck("Gaussian Preprocessing", nil)
	pp.useLogCheck = widget.NewCheck("Use Log Histogram", nil)
	pp.normalizeCheck = widget.NewCheck("Normalize Histogram", nil)
	pp.contrastCheck = widget.NewCheck("Apply Contrast Enhancement", nil)
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
}

func (pp *ParameterPanel) buildLayout() {
	sliderRow := container.NewHBox(
		container.NewVBox(pp.windowSizeLabel, pp.windowSizeSlider),
		container.NewVBox(pp.histBinsLabel, pp.histBinsSlider),
		container.NewVBox(pp.smoothingStrengthLabel, pp.smoothingStrengthSlider),
	)

	checkRow1 := container.NewHBox(
		pp.edgePreservationCheck,
		pp.noiseRobustnessCheck,
	)

	checkRow2 := container.NewHBox(
		pp.gaussianPreprocessCheck,
		pp.useLogCheck,
	)

	checkRow3 := container.NewHBox(
		pp.normalizeCheck,
		pp.contrastCheck,
	)

	pp.container = container.NewVBox(
		widget.NewLabel("2D Otsu Parameters:"),
		sliderRow,
		checkRow1,
		checkRow2,
		checkRow3,
	)
}

func (pp *ParameterPanel) setDefaults() {
	pp.windowSizeSlider.SetValue(7)
	pp.histBinsSlider.SetValue(0)
	pp.smoothingStrengthSlider.SetValue(1.0)
	pp.gaussianPreprocessCheck.SetChecked(true)
	pp.normalizeCheck.SetChecked(true)
}

func (pp *ParameterPanel) GetParameters() *OtsuParameters {
	windowSize := int(pp.windowSizeSlider.Value)
	if windowSize%2 == 0 {
		windowSize++
	}

	histBins := int(pp.histBinsSlider.Value)

	return &OtsuParameters{
		WindowSize:              windowSize,
		HistogramBins:           histBins,
		SmoothingStrength:       pp.smoothingStrengthSlider.Value,
		EdgePreservation:        pp.edgePreservationCheck.Checked,
		NoiseRobustness:         pp.noiseRobustnessCheck.Checked,
		GaussianPreprocessing:   pp.gaussianPreprocessCheck.Checked,
		UseLogHistogram:         pp.useLogCheck.Checked,
		NormalizeHistogram:      pp.normalizeCheck.Checked,
		ApplyContrastEnhancement: pp.contrastCheck.Checked,
	}
}

func (pp *ParameterPanel) GetContainer() *fyne.Container {
	return pp.container
}