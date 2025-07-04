package main

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

type ParameterBinder struct {
	app              *Application
	paramsBinding    binding.DataMap
	currentParams    *OtsuParameters
	updateInProgress bool
	listeners        []func(*OtsuParameters)
}

func NewParameterBinder(app *Application) *ParameterBinder {
	boundParams := map[string]interface{}{
		"WindowSize":                 7,
		"HistogramBins":              0,
		"SmoothingStrength":          1.0,
		"EdgePreservation":           false,
		"NoiseRobustness":            false,
		"GaussianPreprocessing":      true,
		"UseLogHistogram":            false,
		"NormalizeHistogram":         true,
		"ApplyContrastEnhancement":   false,
		"AdaptiveWindowSizing":       false,
		"MultiScaleProcessing":       false,
		"PyramidLevels":              3,
		"NeighborhoodType":           "Rectangular",
		"InterpolationMethod":        "Bilinear",
		"MorphologicalPostProcess":   false,
		"MorphologicalKernelSize":    3,
		"HomomorphicFiltering":       false,
		"AnisotropicDiffusion":       false,
		"DiffusionIterations":        5,
		"DiffusionKappa":             30.0,
		"RegionAdaptiveThresholding": false,
		"RegionGridSize":             64,
	}

	pb := &ParameterBinder{
		app:           app,
		paramsBinding: binding.BindUntypedMap(&boundParams),
		currentParams: pb.convertToOtsuParameters(boundParams),
		listeners:     make([]func(*OtsuParameters), 0),
	}

	pb.setupParameterListener()
	return pb
}

func (pb *ParameterBinder) convertToOtsuParameters(params map[string]interface{}) *OtsuParameters {
	getInt := func(key string, def int) int {
		if val, ok := params[key]; ok {
			if intVal, ok := val.(int); ok {
				return intVal
			}
		}
		return def
	}

	getFloat := func(key string, def float64) float64 {
		if val, ok := params[key]; ok {
			if floatVal, ok := val.(float64); ok {
				return floatVal
			}
		}
		return def
	}

	getBool := func(key string, def bool) bool {
		if val, ok := params[key]; ok {
			if boolVal, ok := val.(bool); ok {
				return boolVal
			}
		}
		return def
	}

	getString := func(key string, def string) string {
		if val, ok := params[key]; ok {
			if stringVal, ok := val.(string); ok {
				return stringVal
			}
		}
		return def
	}

	return &OtsuParameters{
		WindowSize:                 pb.ensureOddWindowSize(getInt("WindowSize", 7)),
		HistogramBins:              getInt("HistogramBins", 0),
		SmoothingStrength:          getFloat("SmoothingStrength", 1.0),
		EdgePreservation:           getBool("EdgePreservation", false),
		NoiseRobustness:            getBool("NoiseRobustness", false),
		GaussianPreprocessing:      getBool("GaussianPreprocessing", true),
		UseLogHistogram:            getBool("UseLogHistogram", false),
		NormalizeHistogram:         getBool("NormalizeHistogram", true),
		ApplyContrastEnhancement:   getBool("ApplyContrastEnhancement", false),
		AdaptiveWindowSizing:       getBool("AdaptiveWindowSizing", false),
		MultiScaleProcessing:       getBool("MultiScaleProcessing", false),
		PyramidLevels:              getInt("PyramidLevels", 3),
		NeighborhoodType:           getString("NeighborhoodType", "Rectangular"),
		InterpolationMethod:        getString("InterpolationMethod", "Bilinear"),
		MorphologicalPostProcess:   getBool("MorphologicalPostProcess", false),
		MorphologicalKernelSize:    pb.ensureOddKernelSize(getInt("MorphologicalKernelSize", 3)),
		HomomorphicFiltering:       getBool("HomomorphicFiltering", false),
		AnisotropicDiffusion:       getBool("AnisotropicDiffusion", false),
		DiffusionIterations:        getInt("DiffusionIterations", 5),
		DiffusionKappa:             getFloat("DiffusionKappa", 30.0),
		RegionAdaptiveThresholding: getBool("RegionAdaptiveThresholding", false),
		RegionGridSize:             getInt("RegionGridSize", 64),
	}
}

func (pb *ParameterBinder) setupParameterListener() {
	pb.paramsBinding.AddListener(binding.NewDataListener(func() {
		if pb.updateInProgress {
			return
		}

		fyne.Do(func() {
			pb.updateCurrentParameters()
			pb.notifyListeners()
		})
	}))
}

func (pb *ParameterBinder) updateCurrentParameters() {
	boundParams, err := pb.paramsBinding.Get()
	if err == nil {
		pb.currentParams = pb.convertToOtsuParameters(boundParams)
	}
}

func (pb *ParameterBinder) ensureOddWindowSize(size int) int {
	if size%2 == 0 {
		return size + 1
	}
	return size
}

func (pb *ParameterBinder) ensureOddKernelSize(size int) int {
	if size%2 == 0 {
		return size + 1
	}
	return size
}

func (pb *ParameterBinder) BindSlider(slider *widget.Slider, fieldName string, labelWidget *widget.Label, formatFunc func(float64) string) error {
	binding, err := pb.paramsBinding.GetItem(fieldName)
	if err != nil {
		return fmt.Errorf("get field binding for %s: %w", fieldName, err)
	}

	slider.OnChanged = func(value float64) {
		pb.updateInProgress = true
		binding.Set(value)
		if labelWidget != nil && formatFunc != nil {
			labelWidget.SetText(formatFunc(value))
		}
		pb.updateInProgress = false
		pb.updateCurrentParameters()
		pb.notifyListeners()
	}

	// Set initial value
	if val, err := binding.Get(); err == nil {
		if floatVal, ok := val.(float64); ok {
			slider.SetValue(floatVal)
		} else if intVal, ok := val.(int); ok {
			slider.SetValue(float64(intVal))
		}
	}

	return nil
}

func (pb *ParameterBinder) BindCheck(check *widget.Check, fieldName string) error {
	binding, err := pb.paramsBinding.GetItem(fieldName)
	if err != nil {
		return fmt.Errorf("get field binding for %s: %w", fieldName, err)
	}

	check.OnChanged = func(checked bool) {
		pb.updateInProgress = true
		binding.Set(checked)
		pb.updateInProgress = false
		pb.updateCurrentParameters()
		pb.notifyListeners()
	}

	// Set initial value
	if val, err := binding.Get(); err == nil {
		if boolVal, ok := val.(bool); ok {
			check.SetChecked(boolVal)
		}
	}

	return nil
}

func (pb *ParameterBinder) BindSelect(selector *widget.Select, fieldName string) error {
	binding, err := pb.paramsBinding.GetItem(fieldName)
	if err != nil {
		return fmt.Errorf("get field binding for %s: %w", fieldName, err)
	}

	selector.OnChanged = func(selected string) {
		pb.updateInProgress = true
		binding.Set(selected)
		pb.updateInProgress = false
		pb.updateCurrentParameters()
		pb.notifyListeners()
	}

	// Set initial value
	if val, err := binding.Get(); err == nil {
		if stringVal, ok := val.(string); ok {
			selector.SetSelected(stringVal)
		}
	}

	return nil
}

func (pb *ParameterBinder) SetFieldValue(fieldName string, value interface{}) error {
	pb.updateInProgress = true
	defer func() { pb.updateInProgress = false }()

	binding, err := pb.paramsBinding.GetItem(fieldName)
	if err != nil {
		return fmt.Errorf("get field binding for %s: %w", fieldName, err)
	}

	return binding.Set(value)
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
	switch method {
	case "Multi-Scale Pyramid":
		pb.SetFieldValue("MultiScaleProcessing", true)
		pb.SetFieldValue("RegionAdaptiveThresholding", false)
	case "Region Adaptive":
		pb.SetFieldValue("MultiScaleProcessing", false)
		pb.SetFieldValue("RegionAdaptiveThresholding", true)
	default:
		pb.SetFieldValue("MultiScaleProcessing", false)
		pb.SetFieldValue("RegionAdaptiveThresholding", false)
	}
}

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
