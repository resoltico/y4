package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type BasicParameterWidgets struct {
	windowSizeSlider        *widget.Slider
	windowSizeLabel         *widget.Label
	histBinsSlider          *widget.Slider
	histBinsLabel           *widget.Label
	smoothingStrengthSlider *widget.Slider
	smoothingStrengthLabel  *widget.Label
}

func NewBasicParameterWidgets() *BasicParameterWidgets {
	bpw := &BasicParameterWidgets{}

	bpw.windowSizeSlider = widget.NewSlider(3, 21)
	bpw.windowSizeSlider.Step = 2
	bpw.windowSizeLabel = widget.NewLabel("Window Size: 7")

	bpw.histBinsSlider = widget.NewSlider(0, 256)
	bpw.histBinsLabel = widget.NewLabel("Histogram Bins: Auto")

	bpw.smoothingStrengthSlider = widget.NewSlider(0.0, 5.0)
	bpw.smoothingStrengthLabel = widget.NewLabel("Smoothing Strength: 1.0")

	return bpw
}

func (bpw *BasicParameterWidgets) CreateSection() *fyne.Container {
	return container.NewVBox(
		widget.NewLabel("Basic Parameters"),
		widget.NewSeparator(),
		container.NewHBox(
			container.NewVBox(bpw.windowSizeLabel, bpw.windowSizeSlider),
			container.NewVBox(bpw.histBinsLabel, bpw.histBinsSlider),
			container.NewVBox(bpw.smoothingStrengthLabel, bpw.smoothingStrengthSlider),
		),
	)
}

type AlgorithmWidgets struct {
	edgePreservationCheck   *widget.Check
	noiseRobustnessCheck    *widget.Check
	gaussianPreprocessCheck *widget.Check
	useLogCheck             *widget.Check
	normalizeCheck          *widget.Check
	contrastCheck           *widget.Check
}

func NewAlgorithmWidgets() *AlgorithmWidgets {
	aw := &AlgorithmWidgets{}

	aw.edgePreservationCheck = widget.NewCheck("Edge Preservation", nil)
	aw.noiseRobustnessCheck = widget.NewCheck("Noise Robustness", nil)
	aw.gaussianPreprocessCheck = widget.NewCheck("Gaussian Preprocessing", nil)
	aw.useLogCheck = widget.NewCheck("Use Log Histogram", nil)
	aw.normalizeCheck = widget.NewCheck("Normalize Histogram", nil)
	aw.contrastCheck = widget.NewCheck("Adaptive Contrast Enhancement", nil)

	return aw
}

func (aw *AlgorithmWidgets) CreateSection() *fyne.Container {
	return container.NewVBox(
		widget.NewLabel("Algorithm Options"),
		widget.NewSeparator(),
		container.NewHBox(
			aw.edgePreservationCheck,
			aw.noiseRobustnessCheck,
		),
		container.NewHBox(
			aw.gaussianPreprocessCheck,
			aw.useLogCheck,
		),
		container.NewHBox(
			aw.normalizeCheck,
			aw.contrastCheck,
		),
	)
}

type MethodWidgets struct {
	processingMethodSelect *widget.Select
	pyramidLevelsSlider    *widget.Slider
	pyramidLevelsLabel     *widget.Label
	regionGridSlider       *widget.Slider
	regionGridLabel        *widget.Label
}

func NewMethodWidgets() *MethodWidgets {
	mw := &MethodWidgets{}

	mw.processingMethodSelect = widget.NewSelect([]string{
		"Single Scale",
		"Multi-Scale Pyramid",
		"Region Adaptive",
	}, nil)

	mw.pyramidLevelsSlider = widget.NewSlider(1, 5)
	mw.pyramidLevelsLabel = widget.NewLabel("Pyramid Levels: 3")

	mw.regionGridSlider = widget.NewSlider(32, 256)
	mw.regionGridLabel = widget.NewLabel("Region Grid Size: 64")

	return mw
}

func (mw *MethodWidgets) CreateSection() *fyne.Container {
	return container.NewVBox(
		widget.NewLabel("Processing Method"),
		widget.NewSeparator(),
		mw.processingMethodSelect,
		container.NewHBox(
			container.NewVBox(mw.pyramidLevelsLabel, mw.pyramidLevelsSlider),
			container.NewVBox(mw.regionGridLabel, mw.regionGridSlider),
		),
	)
}

type NeighborhoodWidgets struct {
	neighborhoodTypeSelect *widget.Select
	adaptiveWindowCheck    *widget.Check
}

func NewNeighborhoodWidgets() *NeighborhoodWidgets {
	nw := &NeighborhoodWidgets{}

	nw.neighborhoodTypeSelect = widget.NewSelect([]string{
		"Rectangular",
		"Circular",
		"Distance Weighted",
	}, nil)

	nw.adaptiveWindowCheck = widget.NewCheck("Adaptive Window Sizing", nil)

	return nw
}

func (nw *NeighborhoodWidgets) CreateSection() *fyne.Container {
	return container.NewVBox(
		widget.NewLabel("Neighborhood Parameters"),
		widget.NewSeparator(),
		nw.neighborhoodTypeSelect,
		nw.adaptiveWindowCheck,
	)
}

type PreprocessingWidgets struct {
	homomorphicCheck     *widget.Check
	anisotropicCheck     *widget.Check
	diffusionIterSlider  *widget.Slider
	diffusionIterLabel   *widget.Label
	diffusionKappaSlider *widget.Slider
	diffusionKappaLabel  *widget.Label
}

func NewPreprocessingWidgets() *PreprocessingWidgets {
	pw := &PreprocessingWidgets{}

	pw.homomorphicCheck = widget.NewCheck("Homomorphic Filtering", nil)
	pw.anisotropicCheck = widget.NewCheck("Anisotropic Diffusion", nil)
	pw.diffusionIterSlider = widget.NewSlider(1, 20)
	pw.diffusionIterLabel = widget.NewLabel("Diffusion Iterations: 5")
	pw.diffusionKappaSlider = widget.NewSlider(10.0, 100.0)
	pw.diffusionKappaLabel = widget.NewLabel("Diffusion Kappa: 30.0")

	return pw
}

func (pw *PreprocessingWidgets) CreateSection() *fyne.Container {
	return container.NewVBox(
		widget.NewLabel("Advanced Preprocessing"),
		widget.NewSeparator(),
		container.NewHBox(
			pw.homomorphicCheck,
			pw.anisotropicCheck,
		),
		container.NewHBox(
			container.NewVBox(pw.diffusionIterLabel, pw.diffusionIterSlider),
			container.NewVBox(pw.diffusionKappaLabel, pw.diffusionKappaSlider),
		),
	)
}

type PostprocessingWidgets struct {
	interpolationSelect   *widget.Select
	morphPostProcessCheck *widget.Check
	morphKernelSlider     *widget.Slider
	morphKernelLabel      *widget.Label
}

func NewPostprocessingWidgets() *PostprocessingWidgets {
	ppw := &PostprocessingWidgets{}

	ppw.interpolationSelect = widget.NewSelect([]string{
		"Nearest",
		"Bilinear",
		"Bicubic",
	}, nil)

	ppw.morphPostProcessCheck = widget.NewCheck("Morphological Post-Processing", nil)
	ppw.morphKernelSlider = widget.NewSlider(1, 7)
	ppw.morphKernelLabel = widget.NewLabel("Morphological Kernel: 3")

	return ppw
}

func (ppw *PostprocessingWidgets) CreateSection() *fyne.Container {
	return container.NewVBox(
		widget.NewLabel("Post-Processing"),
		widget.NewSeparator(),
		ppw.interpolationSelect,
		ppw.morphPostProcessCheck,
		container.NewVBox(ppw.morphKernelLabel, ppw.morphKernelSlider),
	)
}
