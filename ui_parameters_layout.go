package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (pp *ParameterPanel) buildLayout() {
	pp.createMethodWidgets()
	pp.createAdvancedWidgets()
	pp.setupMethodHandlers()
	pp.setupAdvancedHandlers()

	// Basic parameters section
	basicSection := container.NewVBox(
		widget.NewLabel("Basic Parameters"),
		widget.NewSeparator(),
		container.NewHBox(
			container.NewVBox(pp.windowSizeLabel, pp.windowSizeSlider),
			container.NewVBox(pp.histBinsLabel, pp.histBinsSlider),
			container.NewVBox(pp.smoothingStrengthLabel, pp.smoothingStrengthSlider),
		),
	)

	// Processing method section
	methodSection := container.NewVBox(
		widget.NewLabel("Processing Method"),
		widget.NewSeparator(),
		pp.processingMethodSelect,
		container.NewHBox(
			container.NewVBox(pp.pyramidLevelsLabel, pp.pyramidLevelsSlider),
			container.NewVBox(pp.regionGridLabel, pp.regionGridSlider),
		),
	)

	// Neighborhood parameters section
	neighborhoodSection := container.NewVBox(
		widget.NewLabel("Neighborhood Parameters"),
		widget.NewSeparator(),
		pp.neighborhoodTypeSelect,
		pp.adaptiveWindowCheck,
	)

	// Algorithm options section
	algorithmSection := container.NewVBox(
		widget.NewLabel("Algorithm Options"),
		widget.NewSeparator(),
		container.NewHBox(
			pp.edgePreservationCheck,
			pp.noiseRobustnessCheck,
		),
		container.NewHBox(
			pp.gaussianPreprocessCheck,
			pp.useLogCheck,
		),
		container.NewHBox(
			pp.normalizeCheck,
			pp.contrastCheck,
		),
	)

	// Preprocessing section
	preprocessingSection := container.NewVBox(
		widget.NewLabel("Advanced Preprocessing"),
		widget.NewSeparator(),
		container.NewHBox(
			pp.homomorphicCheck,
			pp.anisotropicCheck,
		),
		container.NewHBox(
			container.NewVBox(pp.diffusionIterLabel, pp.diffusionIterSlider),
			container.NewVBox(pp.diffusionKappaLabel, pp.diffusionKappaSlider),
		),
	)

	// Post-processing section
	postprocessingSection := container.NewVBox(
		widget.NewLabel("Post-Processing"),
		widget.NewSeparator(),
		pp.interpolationSelect,
		pp.morphPostProcessCheck,
		container.NewVBox(pp.morphKernelLabel, pp.morphKernelSlider),
	)

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
}

func (pp *ParameterPanel) setDefaults() {
	pp.windowSizeSlider.SetValue(7)
	pp.histBinsSlider.SetValue(0)
	pp.smoothingStrengthSlider.SetValue(1.0)
	pp.pyramidLevelsSlider.SetValue(3)
	pp.regionGridSlider.SetValue(64)
	pp.morphKernelSlider.SetValue(3)
	pp.diffusionIterSlider.SetValue(5)
	pp.diffusionKappaSlider.SetValue(30.0)

	pp.processingMethodSelect.SetSelected("Single Scale")
	pp.neighborhoodTypeSelect.SetSelected("Rectangular")
	pp.interpolationSelect.SetSelected("Bilinear")

	pp.gaussianPreprocessCheck.SetChecked(true)
	pp.normalizeCheck.SetChecked(true)

	// Hide method-specific controls initially
	pp.pyramidLevelsSlider.Hide()
	pp.pyramidLevelsLabel.Hide()
	pp.regionGridSlider.Hide()
	pp.regionGridLabel.Hide()

	// Hide morphological controls initially
	pp.morphKernelSlider.Hide()
	pp.morphKernelLabel.Hide()

	// Hide diffusion controls initially
	pp.diffusionIterSlider.Hide()
	pp.diffusionIterLabel.Hide()
	pp.diffusionKappaSlider.Hide()
	pp.diffusionKappaLabel.Hide()
}

func (pp *ParameterPanel) GetContainer() *fyne.Container {
	return pp.container
}
