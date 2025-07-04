package main

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

type ParameterBinder struct {
	app              *Application
	currentParams    *OtsuParameters
	updateInProgress bool
	listeners        []func(*OtsuParameters)

	// Individual bindings for each parameter
	windowSizeBinding        binding.Float
	histogramBinsBinding     binding.Float
	smoothingStrengthBinding binding.Float
	pyramidLevelsBinding     binding.Float
	regionGridSizeBinding    binding.Float
	morphKernelSizeBinding   binding.Float
	diffusionIterBinding     binding.Float
	diffusionKappaBinding    binding.Float

	// Boolean bindings
	edgePreservationBinding     binding.Bool
	noiseRobustnessBinding      binding.Bool
	gaussianPreprocessBinding   binding.Bool
	useLogHistogramBinding      binding.Bool
	normalizeHistogramBinding   binding.Bool
	contrastEnhancementBinding  binding.Bool
	adaptiveWindowBinding       binding.Bool
	multiScaleBinding           binding.Bool
	regionAdaptiveBinding       binding.Bool
	morphPostProcessBinding     binding.Bool
	homomorphicFilteringBinding binding.Bool
	anisotropicDiffusionBinding binding.Bool

	// String bindings
	neighborhoodTypeBinding    binding.String
	interpolationMethodBinding binding.String
}

func NewParameterBinder(app *Application) *ParameterBinder {
	pb := &ParameterBinder{
		app:       app,
		listeners: make([]func(*OtsuParameters), 0),
	}

	pb.createBindings()
	pb.setDefaultValues()
	pb.setupListeners()
	pb.updateCurrentParameters()

	return pb
}

func (pb *ParameterBinder) createBindings() {
	// Create all individual bindings
	pb.windowSizeBinding = binding.NewFloat()
	pb.histogramBinsBinding = binding.NewFloat()
	pb.smoothingStrengthBinding = binding.NewFloat()
	pb.pyramidLevelsBinding = binding.NewFloat()
	pb.regionGridSizeBinding = binding.NewFloat()
	pb.morphKernelSizeBinding = binding.NewFloat()
	pb.diffusionIterBinding = binding.NewFloat()
	pb.diffusionKappaBinding = binding.NewFloat()

	pb.edgePreservationBinding = binding.NewBool()
	pb.noiseRobustnessBinding = binding.NewBool()
	pb.gaussianPreprocessBinding = binding.NewBool()
	pb.useLogHistogramBinding = binding.NewBool()
	pb.normalizeHistogramBinding = binding.NewBool()
	pb.contrastEnhancementBinding = binding.NewBool()
	pb.adaptiveWindowBinding = binding.NewBool()
	pb.multiScaleBinding = binding.NewBool()
	pb.regionAdaptiveBinding = binding.NewBool()
	pb.morphPostProcessBinding = binding.NewBool()
	pb.homomorphicFilteringBinding = binding.NewBool()
	pb.anisotropicDiffusionBinding = binding.NewBool()

	pb.neighborhoodTypeBinding = binding.NewString()
	pb.interpolationMethodBinding = binding.NewString()
}

func (pb *ParameterBinder) setDefaultValues() {
	pb.windowSizeBinding.Set(7.0)
	pb.histogramBinsBinding.Set(0.0)
	pb.smoothingStrengthBinding.Set(1.0)
	pb.pyramidLevelsBinding.Set(3.0)
	pb.regionGridSizeBinding.Set(64.0)
	pb.morphKernelSizeBinding.Set(3.0)
	pb.diffusionIterBinding.Set(5.0)
	pb.diffusionKappaBinding.Set(30.0)

	pb.gaussianPreprocessBinding.Set(true)
	pb.normalizeHistogramBinding.Set(true)

	pb.neighborhoodTypeBinding.Set("Rectangular")
	pb.interpolationMethodBinding.Set("Bilinear")
}

func (pb *ParameterBinder) setupListeners() {
	// Add listeners to all bindings
	pb.windowSizeBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
	pb.histogramBinsBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
	pb.smoothingStrengthBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
	pb.pyramidLevelsBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
	pb.regionGridSizeBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
	pb.morphKernelSizeBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
	pb.diffusionIterBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
	pb.diffusionKappaBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))

	pb.edgePreservationBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
	pb.noiseRobustnessBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
	pb.gaussianPreprocessBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
	pb.useLogHistogramBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
	pb.normalizeHistogramBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
	pb.contrastEnhancementBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
	pb.adaptiveWindowBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
	pb.multiScaleBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
	pb.regionAdaptiveBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
	pb.morphPostProcessBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
	pb.homomorphicFilteringBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
	pb.anisotropicDiffusionBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))

	pb.neighborhoodTypeBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
	pb.interpolationMethodBinding.AddListener(binding.NewDataListener(pb.onParameterChanged))
}

func (pb *ParameterBinder) onParameterChanged() {
	if pb.updateInProgress {
		return
	}

	fyne.Do(func() {
		pb.updateCurrentParameters()
		pb.notifyListeners()
	})
}

func (pb *ParameterBinder) updateCurrentParameters() {
	windowSize, _ := pb.windowSizeBinding.Get()
	histogramBins, _ := pb.histogramBinsBinding.Get()
	smoothingStrength, _ := pb.smoothingStrengthBinding.Get()
	pyramidLevels, _ := pb.pyramidLevelsBinding.Get()
	regionGridSize, _ := pb.regionGridSizeBinding.Get()
	morphKernelSize, _ := pb.morphKernelSizeBinding.Get()
	diffusionIter, _ := pb.diffusionIterBinding.Get()
	diffusionKappa, _ := pb.diffusionKappaBinding.Get()

	edgePreservation, _ := pb.edgePreservationBinding.Get()
	noiseRobustness, _ := pb.noiseRobustnessBinding.Get()
	gaussianPreprocess, _ := pb.gaussianPreprocessBinding.Get()
	useLogHistogram, _ := pb.useLogHistogramBinding.Get()
	normalizeHistogram, _ := pb.normalizeHistogramBinding.Get()
	contrastEnhancement, _ := pb.contrastEnhancementBinding.Get()
	adaptiveWindow, _ := pb.adaptiveWindowBinding.Get()
	multiScale, _ := pb.multiScaleBinding.Get()
	regionAdaptive, _ := pb.regionAdaptiveBinding.Get()
	morphPostProcess, _ := pb.morphPostProcessBinding.Get()
	homomorphicFiltering, _ := pb.homomorphicFilteringBinding.Get()
	anisotropicDiffusion, _ := pb.anisotropicDiffusionBinding.Get()

	neighborhoodType, _ := pb.neighborhoodTypeBinding.Get()
	interpolationMethod, _ := pb.interpolationMethodBinding.Get()

	pb.currentParams = &OtsuParameters{
		WindowSize:                 pb.ensureOddValue(int(windowSize)),
		HistogramBins:              int(histogramBins),
		SmoothingStrength:          smoothingStrength,
		EdgePreservation:           edgePreservation,
		NoiseRobustness:            noiseRobustness,
		GaussianPreprocessing:      gaussianPreprocess,
		UseLogHistogram:            useLogHistogram,
		NormalizeHistogram:         normalizeHistogram,
		ApplyContrastEnhancement:   contrastEnhancement,
		AdaptiveWindowSizing:       adaptiveWindow,
		MultiScaleProcessing:       multiScale,
		PyramidLevels:              int(pyramidLevels),
		NeighborhoodType:           neighborhoodType,
		InterpolationMethod:        interpolationMethod,
		MorphologicalPostProcess:   morphPostProcess,
		MorphologicalKernelSize:    pb.ensureOddValue(int(morphKernelSize)),
		HomomorphicFiltering:       homomorphicFiltering,
		AnisotropicDiffusion:       anisotropicDiffusion,
		DiffusionIterations:        int(diffusionIter),
		DiffusionKappa:             diffusionKappa,
		RegionAdaptiveThresholding: regionAdaptive,
		RegionGridSize:             int(regionGridSize),
	}
}

func (pb *ParameterBinder) ensureOddValue(value int) int {
	if value%2 == 0 {
		return value + 1
	}
	return value
}

func (pb *ParameterBinder) BindSlider(slider *widget.Slider, binding binding.Float, labelWidget *widget.Label, formatFunc func(float64) string) {
	slider.Bind(binding)

	if labelWidget != nil && formatFunc != nil {
		slider.OnChanged = func(value float64) {
			labelWidget.SetText(formatFunc(value))
		}
		// Set initial label value
		if val, err := binding.Get(); err == nil {
			labelWidget.SetText(formatFunc(val))
		}
	}
}

func (pb *ParameterBinder) BindCheck(check *widget.Check, binding binding.Bool) {
	check.Bind(binding)
}

func (pb *ParameterBinder) BindSelect(selector *widget.Select, binding binding.String) {
	selector.Bind(binding)
}

func (pb *ParameterBinder) GetCurrentParameters() *OtsuParameters {
	return pb.currentParams
}

func (pb *ParameterBinder) AddParameterListener(listener func(*OtsuParameters)) {
	pb.listeners = append(pb.listeners, listener)
}

func (pb *ParameterBinder) notifyListeners() {
	for _, listener := range pb.listeners {
		listener(pb.currentParams)
	}
}

func (pb *ParameterBinder) UpdateProcessingMethodDependencies(method string) {
	pb.updateInProgress = true
	defer func() { pb.updateInProgress = false }()

	switch method {
	case "Multi-Scale Pyramid":
		pb.multiScaleBinding.Set(true)
		pb.regionAdaptiveBinding.Set(false)
	case "Region Adaptive":
		pb.multiScaleBinding.Set(false)
		pb.regionAdaptiveBinding.Set(true)
	default:
		pb.multiScaleBinding.Set(false)
		pb.regionAdaptiveBinding.Set(false)
	}

	pb.updateCurrentParameters()
	pb.notifyListeners()
}

// Accessor methods for individual bindings
func (pb *ParameterBinder) GetWindowSizeBinding() binding.Float    { return pb.windowSizeBinding }
func (pb *ParameterBinder) GetHistogramBinsBinding() binding.Float { return pb.histogramBinsBinding }
func (pb *ParameterBinder) GetSmoothingStrengthBinding() binding.Float {
	return pb.smoothingStrengthBinding
}
func (pb *ParameterBinder) GetPyramidLevelsBinding() binding.Float  { return pb.pyramidLevelsBinding }
func (pb *ParameterBinder) GetRegionGridSizeBinding() binding.Float { return pb.regionGridSizeBinding }
func (pb *ParameterBinder) GetMorphKernelSizeBinding() binding.Float {
	return pb.morphKernelSizeBinding
}
func (pb *ParameterBinder) GetDiffusionIterBinding() binding.Float  { return pb.diffusionIterBinding }
func (pb *ParameterBinder) GetDiffusionKappaBinding() binding.Float { return pb.diffusionKappaBinding }

func (pb *ParameterBinder) GetEdgePreservationBinding() binding.Bool {
	return pb.edgePreservationBinding
}
func (pb *ParameterBinder) GetNoiseRobustnessBinding() binding.Bool { return pb.noiseRobustnessBinding }
func (pb *ParameterBinder) GetGaussianPreprocessBinding() binding.Bool {
	return pb.gaussianPreprocessBinding
}
func (pb *ParameterBinder) GetUseLogHistogramBinding() binding.Bool { return pb.useLogHistogramBinding }
func (pb *ParameterBinder) GetNormalizeHistogramBinding() binding.Bool {
	return pb.normalizeHistogramBinding
}
func (pb *ParameterBinder) GetContrastEnhancementBinding() binding.Bool {
	return pb.contrastEnhancementBinding
}
func (pb *ParameterBinder) GetAdaptiveWindowBinding() binding.Bool { return pb.adaptiveWindowBinding }
func (pb *ParameterBinder) GetMorphPostProcessBinding() binding.Bool {
	return pb.morphPostProcessBinding
}
func (pb *ParameterBinder) GetHomomorphicFilteringBinding() binding.Bool {
	return pb.homomorphicFilteringBinding
}
func (pb *ParameterBinder) GetAnisotropicDiffusionBinding() binding.Bool {
	return pb.anisotropicDiffusionBinding
}

func (pb *ParameterBinder) GetNeighborhoodTypeBinding() binding.String {
	return pb.neighborhoodTypeBinding
}
func (pb *ParameterBinder) GetInterpolationMethodBinding() binding.String {
	return pb.interpolationMethodBinding
}

// Format functions
func FormatWindowSize(value float64) string {
	intVal := int(value)
	if intVal%2 == 0 {
		intVal++
	}
	return "Window Size: " + strconv.Itoa(intVal)
}

func FormatHistogramBins(value float64) string {
	intVal := int(value)
	if intVal == 0 {
		return "Histogram Bins: Auto"
	}
	return "Histogram Bins: " + strconv.Itoa(intVal)
}

func FormatSmoothingStrength(value float64) string {
	return "Smoothing Strength: " + strconv.FormatFloat(value, 'f', 1, 64)
}

func FormatPyramidLevels(value float64) string {
	return "Pyramid Levels: " + strconv.Itoa(int(value))
}

func FormatRegionGrid(value float64) string {
	return "Region Grid Size: " + strconv.Itoa(int(value))
}

func FormatMorphKernel(value float64) string {
	intVal := int(value)
	if intVal%2 == 0 {
		intVal++
	}
	return "Morphological Kernel: " + strconv.Itoa(intVal)
}

func FormatDiffusionIter(value float64) string {
	return "Diffusion Iterations: " + strconv.Itoa(int(value))
}

func FormatDiffusionKappa(value float64) string {
	return "Diffusion Kappa: " + strconv.FormatFloat(value, 'f', 1, 64)
}
