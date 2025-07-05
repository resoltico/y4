package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (a *Application) showAbout() {
	metadata := a.fyneApp.Metadata()

	nameLabel := widget.NewLabel(metadata.Name)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}
	nameLabel.Alignment = fyne.TextAlignCenter

	versionLabel := widget.NewLabel("Version " + metadata.Version)
	versionLabel.Alignment = fyne.TextAlignCenter

	descriptionLabel := widget.NewLabel("Advanced 2D Otsu thresholding application with multiple image quality metrics")
	descriptionLabel.Alignment = fyne.TextAlignCenter
	descriptionLabel.Wrapping = fyne.TextWrapWord

	authorLabel := widget.NewLabel("Author: Ervins Strauhmanis")
	authorLabel.Alignment = fyne.TextAlignCenter

	licenseLabel := widget.NewLabel("License: MIT")
	licenseLabel.Alignment = fyne.TextAlignCenter

	featuresLabel := widget.NewLabel("Features: Multi-scale processing, Region-adaptive thresholding, Advanced preprocessing, Comprehensive metrics")
	featuresLabel.Alignment = fyne.TextAlignCenter
	featuresLabel.Wrapping = fyne.TextWrapWord

	techLabel := widget.NewLabel("Built with Go 1.24, Fyne v2.6.1, GoCV v0.41.0")
	techLabel.Alignment = fyne.TextAlignCenter

	content := container.NewVBox(
		widget.NewSeparator(),
		nameLabel,
		versionLabel,
		widget.NewSeparator(),
		descriptionLabel,
		widget.NewSeparator(),
		featuresLabel,
		widget.NewSeparator(),
		techLabel,
		widget.NewSeparator(),
		authorLabel,
		licenseLabel,
		widget.NewSeparator(),
	)

	dialog.NewCustom("About", "Close", content, a.window).Show()
}

func (a *Application) showDebugInfo() {
	if a.debugSystem == nil {
		dialog.ShowInformation("Debug Info", "Debug system not available", a.window)
		return
	}

	a.debugSystem.DumpSystemState()

	debugText := `DEBUG SYSTEM STATUS

The debug system provides monitoring and tracing capabilities.

Use 'go run . 2>&1 | grep -E "(DEBUG|ERROR|WARN)"' to filter logs.`

	debugLabel := widget.NewLabel(debugText)
	debugLabel.Wrapping = fyne.TextWrapWord

	dumpButton := widget.NewButton("Dump Current State", func() {
		a.debugSystem.DumpSystemState()
	})

	content := container.NewVBox(
		debugLabel,
		widget.NewSeparator(),
		dumpButton,
	)

	debugScroll := container.NewScroll(content)
	debugScroll.SetMinSize(fyne.NewSize(600, 500))

	dialog.NewCustom("Debug Information", "Close", debugScroll, a.window).Show()
}
