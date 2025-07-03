package widgets

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ParameterPanel struct {
	container              *fyne.Container
	parametersContent      *fyne.Container
	parameterChangeHandler func(string, interface{})

	// Reusable widgets for 2D Otsu
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

	// Reusable widgets for Iterative Triclass
	initialMethod              *widget.Select
	maxIterSlider              *widget.Slider
	maxIterLabel               *widget.Label
	convergencePrecisionSlider *widget.Slider
	convergencePrecisionLabel  *widget.Label
	minTBDSlider               *widget.Slider
	minTBDLabel                *widget.Label
	classSeparationSlider      *widget.Slider
	classSeparationLabel       *widget.Label
	preprocessingCheck         *widget.Check
	cleanupCheck               *widget.Check
	bordersCheck               *widget.Check

	currentAlgorithm string
}

func NewParameterPanel() *ParameterPanel {
	panel := &ParameterPanel{}
	panel.setupPanel()
	panel.createWidgets()
	return panel
}

func (pp *ParameterPanel) setupPanel() {
	pp.parametersContent = container.NewVBox(
		widget.NewLabel("Parameters:"),
	)
	pp.container = container.NewVBox(pp.parametersContent)
}

func (pp *ParameterPanel) createWidgets() {
	// Create 2D Otsu widgets
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

	// Create Iterative Triclass widgets
	pp.initialMethod = widget.NewSelect([]string{"otsu", "mean", "median"}, nil)

	pp.maxIterSlider = widget.NewSlider(5, 15)
	pp.maxIterLabel = widget.NewLabel("Max Iterations: 10")

	pp.convergencePrecisionSlider = widget.NewSlider(0.1, 10.0)
	pp.convergencePrecisionLabel = widget.NewLabel("Convergence Precision: 1.0")

	pp.minTBDSlider = widget.NewSlider(0.001, 0.2)
	pp.minTBDLabel = widget.NewLabel("Min TBD Fraction: 0.010")

	pp.classSeparationSlider = widget.NewSlider(0.1, 0.8)
	pp.classSeparationLabel = widget.NewLabel("Class Separation: 0.50")

	pp.preprocessingCheck = widget.NewCheck("Preprocessing", nil)
	pp.cleanupCheck = widget.NewCheck("Result Cleanup", nil)
	pp.bordersCheck = widget.NewCheck("Preserve Borders", nil)
}

func (pp *ParameterPanel) GetContainer() *fyne.Container {
	return pp.container
}

func (pp *ParameterPanel) SetParameterChangeHandler(handler func(string, interface{})) {
	pp.parameterChangeHandler = handler
	pp.setupEventHandlers()
}

func (pp *ParameterPanel) setupEventHandlers() {
	if pp.parameterChangeHandler == nil {
		return
	}

	// 2D Otsu handlers
	pp.windowSizeSlider.OnChanged = func(value float64) {
		intValue := int(value)
		if intValue%2 == 0 {
			intValue++
		}
		pp.windowSizeLabel.SetText("Window Size: " + strconv.Itoa(intValue))
		pp.parameterChangeHandler("window_size", intValue)
	}

	pp.histBinsSlider.OnChanged = func(value float64) {
		intValue := int(value)
		if intValue == 0 {
			pp.histBinsLabel.SetText("Histogram Bins: Auto")
		} else {
			pp.histBinsLabel.SetText("Histogram Bins: " + strconv.Itoa(intValue))
		}
		pp.parameterChangeHandler("histogram_bins", intValue)
	}

	pp.smoothingStrengthSlider.OnChanged = func(value float64) {
		pp.smoothingStrengthLabel.SetText("Smoothing Strength: " + strconv.FormatFloat(value, 'f', 1, 64))
		pp.parameterChangeHandler("smoothing_strength", value)
	}

	pp.edgePreservationCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("edge_preservation", checked)
	}

	pp.noiseRobustnessCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("noise_robustness", checked)
	}

	pp.gaussianPreprocessCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("gaussian_preprocessing", checked)
	}

	pp.useLogCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("use_log_histogram", checked)
	}

	pp.normalizeCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("normalize_histogram", checked)
	}

	pp.contrastCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("apply_contrast_enhancement", checked)
	}

	// Iterative Triclass handlers
	pp.initialMethod.OnChanged = func(value string) {
		pp.parameterChangeHandler("initial_threshold_method", value)
	}

	pp.maxIterSlider.OnChanged = func(value float64) {
		intValue := int(value)
		pp.maxIterLabel.SetText("Max Iterations: " + strconv.Itoa(intValue))
		pp.parameterChangeHandler("max_iterations", intValue)
	}

	pp.convergencePrecisionSlider.OnChanged = func(value float64) {
		pp.convergencePrecisionLabel.SetText("Convergence Precision: " + strconv.FormatFloat(value, 'f', 1, 64))
		pp.parameterChangeHandler("convergence_precision", value)
	}

	pp.minTBDSlider.OnChanged = func(value float64) {
		pp.minTBDLabel.SetText("Min TBD Fraction: " + strconv.FormatFloat(value, 'f', 3, 64))
		pp.parameterChangeHandler("minimum_tbd_fraction", value)
	}

	pp.classSeparationSlider.OnChanged = func(value float64) {
		pp.classSeparationLabel.SetText("Class Separation: " + strconv.FormatFloat(value, 'f', 2, 64))
		pp.parameterChangeHandler("class_separation", value)
	}

	pp.preprocessingCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("preprocessing", checked)
	}

	pp.cleanupCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("result_cleanup", checked)
	}

	pp.bordersCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("preserve_borders", checked)
	}
}

func (pp *ParameterPanel) UpdateParameters(algorithm string, params map[string]interface{}) {
	if pp.currentAlgorithm == algorithm {
		pp.updateValues(params)
		return
	}

	pp.currentAlgorithm = algorithm
	pp.parametersContent.RemoveAll()
	pp.parametersContent.Add(widget.NewLabel("Parameters:"))

	switch algorithm {
	case "2D Otsu":
		pp.buildOtsu2DParameters(params)
	case "Iterative Triclass":
		pp.buildTriclassParameters(params)
	}

	pp.container.Refresh()
}

func (pp *ParameterPanel) updateValues(params map[string]interface{}) {
	switch pp.currentAlgorithm {
	case "2D Otsu":
		pp.updateOtsu2DValues(params)
	case "Iterative Triclass":
		pp.updateTriclassValues(params)
	}
}

func (pp *ParameterPanel) updateOtsu2DValues(params map[string]interface{}) {
	if windowSize := pp.getIntParam(params, "window_size", 7); windowSize != int(pp.windowSizeSlider.Value) {
		pp.windowSizeSlider.SetValue(float64(windowSize))
	}
	if histBins := pp.getIntParam(params, "histogram_bins", 0); histBins != int(pp.histBinsSlider.Value) {
		pp.histBinsSlider.SetValue(float64(histBins))
	}
	if smoothing := pp.getFloatParam(params, "smoothing_strength", 1.0); smoothing != pp.smoothingStrengthSlider.Value {
		pp.smoothingStrengthSlider.SetValue(smoothing)
	}
	if edgePres := pp.getBoolParam(params, "edge_preservation", false); edgePres != pp.edgePreservationCheck.Checked {
		pp.edgePreservationCheck.SetChecked(edgePres)
	}
	if noiseRob := pp.getBoolParam(params, "noise_robustness", false); noiseRob != pp.noiseRobustnessCheck.Checked {
		pp.noiseRobustnessCheck.SetChecked(noiseRob)
	}
	if gaussian := pp.getBoolParam(params, "gaussian_preprocessing", true); gaussian != pp.gaussianPreprocessCheck.Checked {
		pp.gaussianPreprocessCheck.SetChecked(gaussian)
	}
	if useLog := pp.getBoolParam(params, "use_log_histogram", false); useLog != pp.useLogCheck.Checked {
		pp.useLogCheck.SetChecked(useLog)
	}
	if normalize := pp.getBoolParam(params, "normalize_histogram", true); normalize != pp.normalizeCheck.Checked {
		pp.normalizeCheck.SetChecked(normalize)
	}
	if contrast := pp.getBoolParam(params, "apply_contrast_enhancement", false); contrast != pp.contrastCheck.Checked {
		pp.contrastCheck.SetChecked(contrast)
	}
}

func (pp *ParameterPanel) updateTriclassValues(params map[string]interface{}) {
	if method := pp.getStringParam(params, "initial_threshold_method", "otsu"); method != pp.initialMethod.Selected {
		pp.initialMethod.SetSelected(method)
	}
	if maxIter := pp.getIntParam(params, "max_iterations", 10); maxIter != int(pp.maxIterSlider.Value) {
		pp.maxIterSlider.SetValue(float64(maxIter))
	}
	if precision := pp.getFloatParam(params, "convergence_precision", 1.0); precision != pp.convergencePrecisionSlider.Value {
		pp.convergencePrecisionSlider.SetValue(precision)
	}
	if minTBD := pp.getFloatParam(params, "minimum_tbd_fraction", 0.01); minTBD != pp.minTBDSlider.Value {
		pp.minTBDSlider.SetValue(minTBD)
	}
	if separation := pp.getFloatParam(params, "class_separation", 0.5); separation != pp.classSeparationSlider.Value {
		pp.classSeparationSlider.SetValue(separation)
	}
	if preprocess := pp.getBoolParam(params, "preprocessing", false); preprocess != pp.preprocessingCheck.Checked {
		pp.preprocessingCheck.SetChecked(preprocess)
	}
	if cleanup := pp.getBoolParam(params, "result_cleanup", true); cleanup != pp.cleanupCheck.Checked {
		pp.cleanupCheck.SetChecked(cleanup)
	}
	if borders := pp.getBoolParam(params, "preserve_borders", false); borders != pp.bordersCheck.Checked {
		pp.bordersCheck.SetChecked(borders)
	}
}

func (pp *ParameterPanel) buildOtsu2DParameters(params map[string]interface{}) {
	windowSize := pp.getIntParam(params, "window_size", 7)
	pp.windowSizeSlider.SetValue(float64(windowSize))
	pp.windowSizeLabel.SetText("Window Size: " + strconv.Itoa(windowSize))

	histBins := pp.getIntParam(params, "histogram_bins", 0)
	pp.histBinsSlider.SetValue(float64(histBins))
	if histBins == 0 {
		pp.histBinsLabel.SetText("Histogram Bins: Auto")
	} else {
		pp.histBinsLabel.SetText("Histogram Bins: " + strconv.Itoa(histBins))
	}

	smoothingStrength := pp.getFloatParam(params, "smoothing_strength", 1.0)
	pp.smoothingStrengthSlider.SetValue(smoothingStrength)
	pp.smoothingStrengthLabel.SetText("Smoothing Strength: " + strconv.FormatFloat(smoothingStrength, 'f', 1, 64))

	pp.edgePreservationCheck.SetChecked(pp.getBoolParam(params, "edge_preservation", false))
	pp.noiseRobustnessCheck.SetChecked(pp.getBoolParam(params, "noise_robustness", false))
	pp.gaussianPreprocessCheck.SetChecked(pp.getBoolParam(params, "gaussian_preprocessing", true))
	pp.useLogCheck.SetChecked(pp.getBoolParam(params, "use_log_histogram", false))
	pp.normalizeCheck.SetChecked(pp.getBoolParam(params, "normalize_histogram", true))
	pp.contrastCheck.SetChecked(pp.getBoolParam(params, "apply_contrast_enhancement", false))

	pp.parametersContent.Add(container.NewVBox(
		container.NewHBox(
			container.NewVBox(pp.windowSizeLabel, pp.windowSizeSlider),
			container.NewVBox(pp.histBinsLabel, pp.histBinsSlider),
			container.NewVBox(pp.smoothingStrengthLabel, pp.smoothingStrengthSlider),
		),
		container.NewHBox(pp.edgePreservationCheck, pp.noiseRobustnessCheck),
		container.NewHBox(pp.gaussianPreprocessCheck, pp.useLogCheck),
		container.NewHBox(pp.normalizeCheck, pp.contrastCheck),
	))
}

func (pp *ParameterPanel) buildTriclassParameters(params map[string]interface{}) {
	pp.initialMethod.SetSelected(pp.getStringParam(params, "initial_threshold_method", "otsu"))

	maxIter := pp.getIntParam(params, "max_iterations", 10)
	pp.maxIterSlider.SetValue(float64(maxIter))
	pp.maxIterLabel.SetText("Max Iterations: " + strconv.Itoa(maxIter))

	convergencePrecision := pp.getFloatParam(params, "convergence_precision", 1.0)
	pp.convergencePrecisionSlider.SetValue(convergencePrecision)
	pp.convergencePrecisionLabel.SetText("Convergence Precision: " + strconv.FormatFloat(convergencePrecision, 'f', 1, 64))

	minTBD := pp.getFloatParam(params, "minimum_tbd_fraction", 0.01)
	pp.minTBDSlider.SetValue(minTBD)
	pp.minTBDLabel.SetText("Min TBD Fraction: " + strconv.FormatFloat(minTBD, 'f', 3, 64))

	classSeparation := pp.getFloatParam(params, "class_separation", 0.5)
	pp.classSeparationSlider.SetValue(classSeparation)
	pp.classSeparationLabel.SetText("Class Separation: " + strconv.FormatFloat(classSeparation, 'f', 2, 64))

	pp.preprocessingCheck.SetChecked(pp.getBoolParam(params, "preprocessing", false))
	pp.cleanupCheck.SetChecked(pp.getBoolParam(params, "result_cleanup", true))
	pp.bordersCheck.SetChecked(pp.getBoolParam(params, "preserve_borders", false))

	pp.parametersContent.Add(container.NewVBox(
		container.NewHBox(
			container.NewVBox(widget.NewLabel("Initial Method"), pp.initialMethod),
			container.NewVBox(pp.maxIterLabel, pp.maxIterSlider),
			container.NewVBox(pp.convergencePrecisionLabel, pp.convergencePrecisionSlider),
		),
		container.NewHBox(
			container.NewVBox(pp.minTBDLabel, pp.minTBDSlider),
			container.NewVBox(pp.classSeparationLabel, pp.classSeparationSlider),
		),
		container.NewHBox(pp.preprocessingCheck, pp.cleanupCheck, pp.bordersCheck),
	))
}

func (pp *ParameterPanel) getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if value, ok := params[key].(int); ok {
		return value
	}
	return defaultValue
}

func (pp *ParameterPanel) getStringParam(params map[string]interface{}, key string, defaultValue string) string {
	if value, ok := params[key].(string); ok {
		return value
	}
	return defaultValue
}

func (pp *ParameterPanel) getFloatParam(params map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := params[key].(float64); ok {
		return value
	}
	return defaultValue
}

func (pp *ParameterPanel) getBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if value, ok := params[key].(bool); ok {
		return value
	}
	return defaultValue
}
