package main

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ParameterPanel struct {
	app       *Application
	container *fyne.Container
	widgets   *ParameterWidgets

	lastProcessTime  time.Time
	processingCtx    context.Context
	processingCancel context.CancelFunc
}

type ParameterWidgets struct {
	processingMethodSelect *widget.Select
	windowSizeSlider       *widget.Slider
	windowSizeLabel        *widget.Label
	histBinsSlider         *widget.Slider
	histBinsLabel          *widget.Label
	smoothingSlider        *widget.Slider
	smoothingLabel         *widget.Label
	pyramidLevelsSlider    *widget.Slider
	pyramidLevelsLabel     *widget.Label
	regionGridSlider       *widget.Slider
	regionGridLabel        *widget.Label
	neighborhoodSelect     *widget.Select
	interpolationSelect    *widget.Select
	morphKernelSlider      *widget.Slider
	morphKernelLabel       *widget.Label
	diffusionIterSlider    *widget.Slider
	diffusionIterLabel     *widget.Label
	diffusionKappaSlider   *widget.Slider
	diffusionKappaLabel    *widget.Label

	edgePreservationCheck   *widget.Check
	noiseRobustnessCheck    *widget.Check
	gaussianPreprocessCheck *widget.Check
	useLogCheck             *widget.Check
	normalizeCheck          *widget.Check
	contrastCheck           *widget.Check
	adaptiveWindowCheck     *widget.Check
	morphPostProcessCheck   *widget.Check
	homomorphicCheck        *widget.Check
	anisotropicCheck        *widget.Check
}

func NewParameterPanel(app *Application) *ParameterPanel {
	pp := &ParameterPanel{
		app: app,
	}

	pp.widgets = NewParameterWidgets()
	pp.buildLayout()
	pp.setupParameterListener()

	return pp
}

func NewParameterWidgets() *ParameterWidgets {
	w := &ParameterWidgets{}

	w.processingMethodSelect = widget.NewSelect([]string{
		"Single Scale",
		"Multi-Scale Pyramid",
		"Region Adaptive",
	}, nil)

	w.windowSizeSlider = widget.NewSlider(3, 21)
	w.windowSizeSlider.Step = 2
	w.windowSizeSlider.SetValue(7)
	w.windowSizeLabel = widget.NewLabel("Window Size: 7")

	w.histBinsSlider = widget.NewSlider(0, 256)
	w.histBinsSlider.SetValue(0)
	w.histBinsLabel = widget.NewLabel("Histogram Bins: Auto")

	w.smoothingSlider = widget.NewSlider(0.0, 5.0)
	w.smoothingSlider.SetValue(1.0)
	w.smoothingLabel = widget.NewLabel("Smoothing Strength: 1.0")

	w.pyramidLevelsSlider = widget.NewSlider(1, 5)
	w.pyramidLevelsSlider.SetValue(3)
	w.pyramidLevelsLabel = widget.NewLabel("Pyramid Levels: 3")

	w.regionGridSlider = widget.NewSlider(32, 256)
	w.regionGridSlider.SetValue(64)
	w.regionGridLabel = widget.NewLabel("Region Grid Size: 64")

	w.neighborhoodSelect = widget.NewSelect([]string{
		"Rectangular",
		"Circular",
		"Distance Weighted",
	}, nil)
	w.neighborhoodSelect.SetSelected("Rectangular")

	w.interpolationSelect = widget.NewSelect([]string{
		"Nearest",
		"Bilinear",
		"Bicubic",
	}, nil)
	w.interpolationSelect.SetSelected("Bilinear")

	w.morphKernelSlider = widget.NewSlider(1, 7)
	w.morphKernelSlider.Step = 2
	w.morphKernelSlider.SetValue(3)
	w.morphKernelLabel = widget.NewLabel("Morphological Kernel: 3")

	w.diffusionIterSlider = widget.NewSlider(1, 20)
	w.diffusionIterSlider.SetValue(5)
	w.diffusionIterLabel = widget.NewLabel("Diffusion Iterations: 5")

	w.diffusionKappaSlider = widget.NewSlider(10.0, 100.0)
	w.diffusionKappaSlider.SetValue(30)
	w.diffusionKappaLabel = widget.NewLabel("Diffusion Kappa: 30.0")

	w.edgePreservationCheck = widget.NewCheck("Edge Preservation", nil)
	w.noiseRobustnessCheck = widget.NewCheck("Noise Robustness", nil)
	w.gaussianPreprocessCheck = widget.NewCheck("Gaussian Preprocessing", nil)
	w.gaussianPreprocessCheck.SetChecked(true)
	w.useLogCheck = widget.NewCheck("Use Log Histogram", nil)
	w.normalizeCheck = widget.NewCheck("Normalize Histogram", nil)
	w.normalizeCheck.SetChecked(true)
	w.contrastCheck = widget.NewCheck("Adaptive Contrast Enhancement", nil)
	w.adaptiveWindowCheck = widget.NewCheck("Adaptive Window Sizing", nil)
	w.morphPostProcessCheck = widget.NewCheck("Morphological Post-Processing", nil)
	w.homomorphicCheck = widget.NewCheck("Homomorphic Filtering", nil)
	w.anisotropicCheck = widget.NewCheck("Anisotropic Diffusion", nil)

	return w
}

func (pp *ParameterPanel) buildLayout() {
	basicSection := container.NewVBox(
		widget.NewLabel("Basic Parameters"),
		widget.NewSeparator(),
		container.NewHBox(
			container.NewVBox(pp.widgets.windowSizeLabel, pp.widgets.windowSizeSlider),
			container.NewVBox(pp.widgets.histBinsLabel, pp.widgets.histBinsSlider),
			container.NewVBox(pp.widgets.smoothingLabel, pp.widgets.smoothingSlider),
		),
	)

	methodSection := container.NewVBox(
		widget.NewLabel("Processing Method"),
		widget.NewSeparator(),
		pp.widgets.processingMethodSelect,
		container.NewHBox(
			container.NewVBox(pp.widgets.pyramidLevelsLabel, pp.widgets.pyramidLevelsSlider),
			container.NewVBox(pp.widgets.regionGridLabel, pp.widgets.regionGridSlider),
		),
	)

	algorithmSection := container.NewVBox(
		widget.NewLabel("Algorithm Options"),
		widget.NewSeparator(),
		container.NewHBox(
			pp.widgets.edgePreservationCheck,
			pp.widgets.noiseRobustnessCheck,
		),
		container.NewHBox(
			pp.widgets.gaussianPreprocessCheck,
			pp.widgets.useLogCheck,
		),
		container.NewHBox(
			pp.widgets.normalizeCheck,
			pp.widgets.contrastCheck,
		),
	)

	allSections := container.NewVBox(
		basicSection,
		methodSection,
		algorithmSection,
	)

	scroll := container.NewScroll(allSections)
	scroll.SetMinSize(fyne.NewSize(800, 400))
	pp.container = container.NewBorder(nil, nil, nil, nil, scroll)

	pp.widgets.processingMethodSelect.SetSelected("Single Scale")
}

func (pp *ParameterPanel) setupParameterListener() {
	pp.widgets.windowSizeSlider.OnChanged = func(value float64) {
		intVal := int(value)
		if intVal%2 == 0 {
			intVal++
		}
		pp.widgets.windowSizeLabel.SetText(fmt.Sprintf("Window Size: %d", intVal))
		pp.triggerParameterChange()
	}
}

func (pp *ParameterPanel) triggerParameterChange() {
	now := time.Now()
	if now.Sub(pp.lastProcessTime) < 100*time.Millisecond {
		return
	}
	pp.lastProcessTime = now

	if pp.processingCancel != nil {
		pp.processingCancel()
	}

	pp.processingCtx, pp.processingCancel = context.WithCancel(context.Background())
	go pp.delayedProcessing(pp.processingCtx)
}

func (pp *ParameterPanel) delayedProcessing(ctx context.Context) {
	select {
	case <-time.After(200 * time.Millisecond):
	case <-ctx.Done():
		return
	}

	if pp.app.processing.GetOriginalImage() == nil {
		return
	}

	params := pp.GetCurrentParameters()
	fyne.Do(func() {
		pp.app.toolbar.handleProcessImageWithParams(params)
	})
}

func (pp *ParameterPanel) GetCurrentParameters() *OtsuParameters {
	windowSize := int(pp.widgets.windowSizeSlider.Value)
	if windowSize%2 == 0 {
		windowSize++
	}

	return &OtsuParameters{
		WindowSize:                 windowSize,
		HistogramBins:              int(pp.widgets.histBinsSlider.Value),
		SmoothingStrength:          pp.widgets.smoothingSlider.Value,
		EdgePreservation:           pp.widgets.edgePreservationCheck.Checked,
		NoiseRobustness:            pp.widgets.noiseRobustnessCheck.Checked,
		GaussianPreprocessing:      pp.widgets.gaussianPreprocessCheck.Checked,
		UseLogHistogram:            pp.widgets.useLogCheck.Checked,
		NormalizeHistogram:         pp.widgets.normalizeCheck.Checked,
		ApplyContrastEnhancement:   pp.widgets.contrastCheck.Checked,
		AdaptiveWindowSizing:       pp.widgets.adaptiveWindowCheck.Checked,
		MultiScaleProcessing:       pp.widgets.processingMethodSelect.Selected == "Multi-Scale Pyramid",
		PyramidLevels:              int(pp.widgets.pyramidLevelsSlider.Value),
		NeighborhoodType:           pp.widgets.neighborhoodSelect.Selected,
		InterpolationMethod:        pp.widgets.interpolationSelect.Selected,
		MorphologicalPostProcess:   pp.widgets.morphPostProcessCheck.Checked,
		MorphologicalKernelSize:    int(pp.widgets.morphKernelSlider.Value),
		HomomorphicFiltering:       pp.widgets.homomorphicCheck.Checked,
		AnisotropicDiffusion:       pp.widgets.anisotropicCheck.Checked,
		DiffusionIterations:        int(pp.widgets.diffusionIterSlider.Value),
		DiffusionKappa:             pp.widgets.diffusionKappaSlider.Value,
		RegionAdaptiveThresholding: pp.widgets.processingMethodSelect.Selected == "Region Adaptive",
		RegionGridSize:             int(pp.widgets.regionGridSlider.Value),
	}
}

func (pp *ParameterPanel) GetContainer() *fyne.Container {
	return pp.container
}
