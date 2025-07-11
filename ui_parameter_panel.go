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

	// Status and metrics widgets
	statusLabel  *widget.Label
	metricsLabel *widget.Label
	detailsLabel *widget.Label

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
	pp.createStatusMetricsWidgets()
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

func (pp *ParameterPanel) createStatusMetricsWidgets() {
	pp.statusLabel = widget.NewLabel("Ready")
	pp.metricsLabel = widget.NewLabel("No metrics available")
	pp.detailsLabel = widget.NewLabel("Load an image to begin processing")
}

func (pp *ParameterPanel) buildLayout() {
	basicSection := container.NewVBox(
		createSectionHeader("Basic Parameters"),
		container.NewVBox(pp.widgets.windowSizeLabel, pp.widgets.windowSizeSlider),
		container.NewVBox(pp.widgets.histBinsLabel, pp.widgets.histBinsSlider),
		container.NewVBox(pp.widgets.smoothingLabel, pp.widgets.smoothingSlider),
	)

	methodSection := container.NewVBox(
		createSectionHeader("Processing Method"),
		pp.widgets.processingMethodSelect,
		container.NewVBox(pp.widgets.pyramidLevelsLabel, pp.widgets.pyramidLevelsSlider),
		container.NewVBox(pp.widgets.regionGridLabel, pp.widgets.regionGridSlider),
	)

	algorithmSection := container.NewVBox(
		createSectionHeader("Algorithm Options"),
		pp.widgets.edgePreservationCheck,
		pp.widgets.noiseRobustnessCheck,
		pp.widgets.gaussianPreprocessCheck,
		pp.widgets.useLogCheck,
		pp.widgets.normalizeCheck,
		pp.widgets.contrastCheck,
	)

	statusMetricsSection := container.NewVBox(
		createSectionHeader("Status & Metrics"),
		pp.statusLabel,
		pp.metricsLabel,
		pp.detailsLabel,
	)

	allSections := container.NewHBox(
		basicSection,
		methodSection,
		algorithmSection,
		statusMetricsSection,
	)

	pp.container = allSections
	pp.widgets.processingMethodSelect.SetSelected("Single Scale")
}

func (pp *ParameterPanel) resetToDefaults() {
	pp.widgets.windowSizeSlider.SetValue(7)
	pp.widgets.histBinsSlider.SetValue(0)
	pp.widgets.smoothingSlider.SetValue(1.0)
	pp.widgets.pyramidLevelsSlider.SetValue(3)
	pp.widgets.regionGridSlider.SetValue(64)
	pp.widgets.morphKernelSlider.SetValue(3)
	pp.widgets.diffusionIterSlider.SetValue(5)
	pp.widgets.diffusionKappaSlider.SetValue(30)

	pp.widgets.processingMethodSelect.SetSelected("Single Scale")
	pp.widgets.neighborhoodSelect.SetSelected("Rectangular")
	pp.widgets.interpolationSelect.SetSelected("Bilinear")

	pp.widgets.edgePreservationCheck.SetChecked(false)
	pp.widgets.noiseRobustnessCheck.SetChecked(false)
	pp.widgets.gaussianPreprocessCheck.SetChecked(true)
	pp.widgets.useLogCheck.SetChecked(false)
	pp.widgets.normalizeCheck.SetChecked(true)
	pp.widgets.contrastCheck.SetChecked(false)
	pp.widgets.adaptiveWindowCheck.SetChecked(false)
	pp.widgets.morphPostProcessCheck.SetChecked(false)
	pp.widgets.homomorphicCheck.SetChecked(false)
	pp.widgets.anisotropicCheck.SetChecked(false)

	pp.updateLabels()
	pp.triggerParameterChange()
}

func (pp *ParameterPanel) updateLabels() {
	pp.widgets.windowSizeLabel.SetText(fmt.Sprintf("Window Size: %.0f", pp.widgets.windowSizeSlider.Value))
	if pp.widgets.histBinsSlider.Value == 0 {
		pp.widgets.histBinsLabel.SetText("Histogram Bins: Auto")
	} else {
		pp.widgets.histBinsLabel.SetText(fmt.Sprintf("Histogram Bins: %.0f", pp.widgets.histBinsSlider.Value))
	}
	pp.widgets.smoothingLabel.SetText(fmt.Sprintf("Smoothing Strength: %.1f", pp.widgets.smoothingSlider.Value))
	pp.widgets.pyramidLevelsLabel.SetText(fmt.Sprintf("Pyramid Levels: %.0f", pp.widgets.pyramidLevelsSlider.Value))
	pp.widgets.regionGridLabel.SetText(fmt.Sprintf("Region Grid Size: %.0f", pp.widgets.regionGridSlider.Value))
	pp.widgets.morphKernelLabel.SetText(fmt.Sprintf("Morphological Kernel: %.0f", pp.widgets.morphKernelSlider.Value))
	pp.widgets.diffusionIterLabel.SetText(fmt.Sprintf("Diffusion Iterations: %.0f", pp.widgets.diffusionIterSlider.Value))
	pp.widgets.diffusionKappaLabel.SetText(fmt.Sprintf("Diffusion Kappa: %.1f", pp.widgets.diffusionKappaSlider.Value))
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

	pp.widgets.histBinsSlider.OnChanged = func(value float64) {
		if value == 0 {
			pp.widgets.histBinsLabel.SetText("Histogram Bins: Auto")
		} else {
			pp.widgets.histBinsLabel.SetText(fmt.Sprintf("Histogram Bins: %.0f", value))
		}
		pp.triggerParameterChange()
	}

	pp.widgets.smoothingSlider.OnChanged = func(value float64) {
		pp.widgets.smoothingLabel.SetText(fmt.Sprintf("Smoothing Strength: %.1f", value))
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

func (pp *ParameterPanel) SetStatus(status string) {
	pp.statusLabel.SetText("Status: " + status)
}

func (pp *ParameterPanel) SetDetails(details string) {
	pp.detailsLabel.SetText(details)
}

func (pp *ParameterPanel) SetMetrics(metrics *BinaryImageMetrics) {
	if metrics == nil {
		pp.metricsLabel.SetText("No metrics available")
		return
	}

	basicMetrics := fmt.Sprintf("F: %.3f | pF: %.3f | NRM: %.3f | DRD: %.3f",
		metrics.FMeasure(),
		metrics.PseudoFMeasure(),
		metrics.NRM(),
		metrics.DRD(),
	)

	pp.metricsLabel.SetText(basicMetrics)

	debugSystem := GetDebugSystem()
	debugSystem.logger.Info("metrics calculated",
		"f_measure", metrics.FMeasure(),
		"pseudo_f_measure", metrics.PseudoFMeasure(),
		"nrm", metrics.NRM(),
		"drd", metrics.DRD(),
		"mpm", metrics.MPM(),
		"bfc", metrics.BackgroundForegroundContrast(),
		"skeleton", metrics.SkeletonSimilarity(),
	)
}

func (pp *ParameterPanel) SetProcessingDetails(params *OtsuParameters, result *ImageData, metrics *BinaryImageMetrics) {
	if params == nil || result == nil || metrics == nil {
		return
	}

	allMetrics := fmt.Sprintf("MPM: %.3f | BFC: %.3f | Skeleton: %.3f",
		metrics.MPM(),
		metrics.BackgroundForegroundContrast(),
		metrics.SkeletonSimilarity(),
	)

	pp.SetDetails(allMetrics)
}

func (pp *ParameterPanel) GetContainer() *fyne.Container {
	return pp.container
}
