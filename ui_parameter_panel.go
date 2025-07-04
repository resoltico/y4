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

	// Widget groups
	basicWidgets        *BasicParameterWidgets
	algorithmWidgets    *AlgorithmWidgets
	methodWidgets       *MethodWidgets
	neighborhoodWidgets *NeighborhoodWidgets
	preprocessWidgets   *PreprocessingWidgets
	postprocessWidgets  *PostprocessingWidgets

	lastProcessTime  time.Time
	processingCtx    context.Context
	processingCancel context.CancelFunc
}

func NewParameterPanel(app *Application) *ParameterPanel {
	pp := &ParameterPanel{
		app: app,
	}

	pp.binder = NewParameterBinder(app)
	pp.createWidgetGroups()
	pp.setupBindings()
	pp.buildLayout()
	pp.setupParameterListener()

	return pp
}

func (pp *ParameterPanel) createWidgetGroups() {
	pp.basicWidgets = NewBasicParameterWidgets()
	pp.algorithmWidgets = NewAlgorithmWidgets()
	pp.methodWidgets = NewMethodWidgets()
	pp.neighborhoodWidgets = NewNeighborhoodWidgets()
	pp.preprocessWidgets = NewPreprocessingWidgets()
	pp.postprocessWidgets = NewPostprocessingWidgets()
}

func (pp *ParameterPanel) setupBindings() {
	// Bind basic parameters
	pp.binder.BindSlider(pp.basicWidgets.windowSizeSlider, pp.binder.GetWindowSizeBinding(), pp.basicWidgets.windowSizeLabel, FormatWindowSize)
	pp.binder.BindSlider(pp.basicWidgets.histBinsSlider, pp.binder.GetHistogramBinsBinding(), pp.basicWidgets.histBinsLabel, FormatHistogramBins)
	pp.binder.BindSlider(pp.basicWidgets.smoothingStrengthSlider, pp.binder.GetSmoothingStrengthBinding(), pp.basicWidgets.smoothingStrengthLabel, FormatSmoothingStrength)

	// Bind algorithm toggles
	pp.binder.BindCheck(pp.algorithmWidgets.edgePreservationCheck, pp.binder.GetEdgePreservationBinding())
	pp.binder.BindCheck(pp.algorithmWidgets.noiseRobustnessCheck, pp.binder.GetNoiseRobustnessBinding())
	pp.binder.BindCheck(pp.algorithmWidgets.gaussianPreprocessCheck, pp.binder.GetGaussianPreprocessBinding())
	pp.binder.BindCheck(pp.algorithmWidgets.useLogCheck, pp.binder.GetUseLogHistogramBinding())
	pp.binder.BindCheck(pp.algorithmWidgets.normalizeCheck, pp.binder.GetNormalizeHistogramBinding())
	pp.binder.BindCheck(pp.algorithmWidgets.contrastCheck, pp.binder.GetContrastEnhancementBinding())

	// Bind method parameters
	pp.binder.BindSlider(pp.methodWidgets.pyramidLevelsSlider, pp.binder.GetPyramidLevelsBinding(), pp.methodWidgets.pyramidLevelsLabel, FormatPyramidLevels)
	pp.binder.BindSlider(pp.methodWidgets.regionGridSlider, pp.binder.GetRegionGridSizeBinding(), pp.methodWidgets.regionGridLabel, FormatRegionGrid)

	// Bind neighborhood parameters
	pp.binder.BindSelect(pp.neighborhoodWidgets.neighborhoodTypeSelect, pp.binder.GetNeighborhoodTypeBinding())
	pp.binder.BindCheck(pp.neighborhoodWidgets.adaptiveWindowCheck, pp.binder.GetAdaptiveWindowBinding())

	// Bind preprocessing parameters
	pp.binder.BindCheck(pp.preprocessWidgets.homomorphicCheck, pp.binder.GetHomomorphicFilteringBinding())
	pp.binder.BindCheck(pp.preprocessWidgets.anisotropicCheck, pp.binder.GetAnisotropicDiffusionBinding())
	pp.binder.BindSlider(pp.preprocessWidgets.diffusionIterSlider, pp.binder.GetDiffusionIterBinding(), pp.preprocessWidgets.diffusionIterLabel, FormatDiffusionIter)
	pp.binder.BindSlider(pp.preprocessWidgets.diffusionKappaSlider, pp.binder.GetDiffusionKappaBinding(), pp.preprocessWidgets.diffusionKappaLabel, FormatDiffusionKappa)

	// Bind postprocessing parameters
	pp.binder.BindSelect(pp.postprocessWidgets.interpolationSelect, pp.binder.GetInterpolationMethodBinding())
	pp.binder.BindCheck(pp.postprocessWidgets.morphPostProcessCheck, pp.binder.GetMorphPostProcessBinding())
	pp.binder.BindSlider(pp.postprocessWidgets.morphKernelSlider, pp.binder.GetMorphKernelSizeBinding(), pp.postprocessWidgets.morphKernelLabel, FormatMorphKernel)

	// Setup special handlers
	pp.setupSpecialHandlers()
}

func (pp *ParameterPanel) setupSpecialHandlers() {
	// Processing method requires special handling
	pp.methodWidgets.processingMethodSelect.OnChanged = func(method string) {
		pp.handleProcessingMethodChange(method)
	}

	// Toggle control visibility based on checkbox states
	pp.postprocessWidgets.morphPostProcessCheck.OnChanged = func(checked bool) {
		pp.toggleMorphologicalControls(checked)
	}

	pp.preprocessWidgets.anisotropicCheck.OnChanged = func(checked bool) {
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
		pp.methodWidgets.pyramidLevelsSlider.Show()
		pp.methodWidgets.pyramidLevelsLabel.Show()
		pp.methodWidgets.regionGridSlider.Hide()
		pp.methodWidgets.regionGridLabel.Hide()
	case "Region Adaptive":
		pp.methodWidgets.pyramidLevelsSlider.Hide()
		pp.methodWidgets.pyramidLevelsLabel.Hide()
		pp.methodWidgets.regionGridSlider.Show()
		pp.methodWidgets.regionGridLabel.Show()
	default:
		pp.methodWidgets.pyramidLevelsSlider.Hide()
		pp.methodWidgets.pyramidLevelsLabel.Hide()
		pp.methodWidgets.regionGridSlider.Hide()
		pp.methodWidgets.regionGridLabel.Hide()
	}
}

func (pp *ParameterPanel) toggleMorphologicalControls(enabled bool) {
	if enabled {
		pp.postprocessWidgets.morphKernelSlider.Show()
		pp.postprocessWidgets.morphKernelLabel.Show()
	} else {
		pp.postprocessWidgets.morphKernelSlider.Hide()
		pp.postprocessWidgets.morphKernelLabel.Hide()
	}
}

func (pp *ParameterPanel) toggleDiffusionControls(enabled bool) {
	if enabled {
		pp.preprocessWidgets.diffusionIterSlider.Show()
		pp.preprocessWidgets.diffusionIterLabel.Show()
		pp.preprocessWidgets.diffusionKappaSlider.Show()
		pp.preprocessWidgets.diffusionKappaLabel.Show()
	} else {
		pp.preprocessWidgets.diffusionIterSlider.Hide()
		pp.preprocessWidgets.diffusionIterLabel.Hide()
		pp.preprocessWidgets.diffusionKappaSlider.Hide()
		pp.preprocessWidgets.diffusionKappaLabel.Hide()
	}
}

func (pp *ParameterPanel) buildLayout() {
	// Create sections using widget groups
	basicSection := pp.basicWidgets.CreateSection()
	methodSection := pp.methodWidgets.CreateSection()
	neighborhoodSection := pp.neighborhoodWidgets.CreateSection()
	algorithmSection := pp.algorithmWidgets.CreateSection()
	preprocessingSection := pp.preprocessWidgets.CreateSection()
	postprocessingSection := pp.postprocessWidgets.CreateSection()

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
	// Set default values and initial visibility
	pp.methodWidgets.processingMethodSelect.SetSelected("Single Scale")
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
