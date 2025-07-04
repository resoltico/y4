package main

import (
	"strconv"

	"fyne.io/fyne/v2/widget"
)

func (pp *ParameterPanel) createMethodWidgets() {
	// Processing method selection
	pp.processingMethodSelect = widget.NewSelect([]string{
		"Single Scale",
		"Multi-Scale Pyramid",
		"Region Adaptive",
	}, nil)

	pp.pyramidLevelsSlider = widget.NewSlider(1, 5)
	pp.pyramidLevelsLabel = widget.NewLabel("Pyramid Levels: 3")

	pp.regionGridSlider = widget.NewSlider(32, 256)
	pp.regionGridLabel = widget.NewLabel("Region Grid Size: 64")

	// Neighborhood parameters
	pp.neighborhoodTypeSelect = widget.NewSelect([]string{
		"Rectangular",
		"Circular",
		"Distance Weighted",
	}, nil)

	pp.adaptiveWindowCheck = widget.NewCheck("Adaptive Window Sizing", nil)

	// Interpolation and post-processing
	pp.interpolationSelect = widget.NewSelect([]string{
		"Nearest",
		"Bilinear",
		"Bicubic",
	}, nil)

	pp.morphPostProcessCheck = widget.NewCheck("Morphological Post-Processing", nil)
	pp.morphKernelSlider = widget.NewSlider(1, 7)
	pp.morphKernelLabel = widget.NewLabel("Morphological Kernel: 3")
}

func (pp *ParameterPanel) setupMethodHandlers() {
	pp.pyramidLevelsSlider.OnChanged = func(value float64) {
		intValue := int(value)
		pp.pyramidLevelsLabel.SetText("Pyramid Levels: " + strconv.Itoa(intValue))
	}

	pp.regionGridSlider.OnChanged = func(value float64) {
		intValue := int(value)
		pp.regionGridLabel.SetText("Region Grid Size: " + strconv.Itoa(intValue))
	}

	pp.morphKernelSlider.OnChanged = func(value float64) {
		intValue := int(value)
		if intValue%2 == 0 {
			intValue++
		}
		pp.morphKernelLabel.SetText("Morphological Kernel: " + strconv.Itoa(intValue))
	}

	pp.processingMethodSelect.OnChanged = func(method string) {
		pp.updateMethodSpecificControls(method)
	}

	pp.morphPostProcessCheck.OnChanged = func(checked bool) {
		if checked {
			pp.morphKernelSlider.Show()
			pp.morphKernelLabel.Show()
		} else {
			pp.morphKernelSlider.Hide()
			pp.morphKernelLabel.Hide()
		}
	}
}

func (pp *ParameterPanel) updateMethodSpecificControls(method string) {
	switch method {
	case "Multi-Scale Pyramid":
		pp.pyramidLevelsSlider.Show()
		pp.pyramidLevelsLabel.Show()
		pp.regionGridSlider.Hide()
		pp.regionGridLabel.Hide()
	case "Region Adaptive":
		pp.pyramidLevelsSlider.Hide()
		pp.pyramidLevelsLabel.Hide()
		pp.regionGridSlider.Show()
		pp.regionGridLabel.Show()
	default:
		pp.pyramidLevelsSlider.Hide()
		pp.pyramidLevelsLabel.Hide()
		pp.regionGridSlider.Hide()
		pp.regionGridLabel.Hide()
	}
}
