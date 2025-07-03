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
}

func (pp *ParameterPanel) UpdateParameters(algorithm string, params map[string]interface{}) {
	if pp.currentAlgorithm == algorithm {
		pp.updateValues(params)
		return
	}

	pp.currentAlgorithm = algorithm
	pp.parametersContent.RemoveAll()
	pp.parametersContent.Add(widget.NewLabel("Parameters:"))

	if algorithm == "2D Otsu" {
		pp.buildOtsu2DParameters(params)
	}

	pp.container.Refresh()
}

func (pp *ParameterPanel) updateValues(params map[string]interface{}) {
	if pp.currentAlgorithm == "2D Otsu" {
		pp.updateOtsu2DValues(params)
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
