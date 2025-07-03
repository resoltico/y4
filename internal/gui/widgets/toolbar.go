package widgets

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type Toolbar struct {
	container     *fyne.Container
	loadButton    *widget.Button
	saveButton    *widget.Button
	processButton *widget.Button
	statusLabel   *widget.Label
	metricsLabel  *widget.Label

	loadHandler    func()
	saveHandler    func()
	processHandler func()
}

func NewToolbar() *Toolbar {
	toolbar := &Toolbar{}
	toolbar.createComponents()
	toolbar.buildLayout()
	return toolbar
}

func (t *Toolbar) createComponents() {
	t.loadButton = widget.NewButton("Load", t.onLoadClicked)
	t.loadButton.Importance = widget.HighImportance

	t.saveButton = widget.NewButton("Save", t.onSaveClicked)
	t.saveButton.Importance = widget.HighImportance

	t.processButton = widget.NewButton("Process", t.onProcessClicked)
	t.processButton.Importance = widget.HighImportance

	t.statusLabel = widget.NewLabel("Ready")
	t.metricsLabel = widget.NewLabel("PSNR: -- | SSIM: --")
}

func (t *Toolbar) buildLayout() {
	background := canvas.NewRectangle(color.RGBA{R: 250, G: 249, B: 245, A: 255})
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeWidth = 1.0
	border.StrokeColor = color.RGBA{R: 231, G: 231, B: 231, A: 255}

	leftSection := container.NewHBox(t.loadButton, t.saveButton)
	centerSection := container.NewHBox(t.processButton)
	statusSection := container.NewHBox(t.statusLabel)
	rightSection := container.NewHBox(t.metricsLabel)

	content := container.NewBorder(
		nil, nil,
		leftSection,
		rightSection,
		container.NewHBox(centerSection, widget.NewSeparator(), statusSection),
	)

	t.container = container.NewStack(
		border,
		container.NewPadded(
			container.NewStack(background, container.NewPadded(content)),
		),
	)
}

func (t *Toolbar) onLoadClicked() {
	if t.loadHandler != nil {
		t.loadHandler()
	}
}

func (t *Toolbar) onSaveClicked() {
	if t.saveHandler != nil {
		t.saveHandler()
	}
}

func (t *Toolbar) onProcessClicked() {
	if t.processHandler != nil {
		t.processHandler()
	}
}

func (t *Toolbar) onAlgorithmChanged(algorithm string) {
	// No-op: only 2D Otsu supported
}

func (t *Toolbar) GetContainer() *fyne.Container {
	return t.container
}

func (t *Toolbar) SetLoadHandler(handler func()) {
	t.loadHandler = handler
}

func (t *Toolbar) SetSaveHandler(handler func()) {
	t.saveHandler = handler
}

func (t *Toolbar) SetProcessHandler(handler func()) {
	t.processHandler = handler
}

func (t *Toolbar) SetAlgorithmChangeHandler(handler func(string)) {
	// No-op: algorithm selection removed
}

func (t *Toolbar) SetStatus(status string) {
	t.statusLabel.SetText(status)
}

func (t *Toolbar) SetProgress(progress string) {
	// Progress display removed - handled through status messages
}

func (t *Toolbar) SetStage(stage string) {
	// Stage display removed - handled through status messages
}

func (t *Toolbar) SetMetrics(psnr, ssim float64) {
	if psnr > 0 && ssim > 0 {
		text := fmt.Sprintf("PSNR: %.2f dB | SSIM: %.4f", psnr, ssim)
		t.metricsLabel.SetText(text)
	} else {
		t.metricsLabel.SetText("PSNR: -- | SSIM: --")
	}
}
