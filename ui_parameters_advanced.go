package main

import (
	"strconv"

	"fyne.io/fyne/v2/widget"
)

func (pp *ParameterPanel) createAdvancedWidgets() {
	// Advanced preprocessing
	pp.homomorphicCheck = widget.NewCheck("Homomorphic Filtering", nil)
	pp.anisotropicCheck = widget.NewCheck("Anisotropic Diffusion", nil)

	pp.diffusionIterSlider = widget.NewSlider(1, 20)
	pp.diffusionIterLabel = widget.NewLabel("Diffusion Iterations: 5")

	pp.diffusionKappaSlider = widget.NewSlider(10.0, 100.0)
	pp.diffusionKappaLabel = widget.NewLabel("Diffusion Kappa: 30.0")
}

func (pp *ParameterPanel) setupAdvancedHandlers() {
	pp.diffusionIterSlider.OnChanged = func(value float64) {
		intValue := int(value)
		pp.diffusionIterLabel.SetText("Diffusion Iterations: " + strconv.Itoa(intValue))
	}

	pp.diffusionKappaSlider.OnChanged = func(value float64) {
		pp.diffusionKappaLabel.SetText("Diffusion Kappa: " + strconv.FormatFloat(value, 'f', 1, 64))
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
