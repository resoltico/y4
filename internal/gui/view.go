package gui

import (
	"fmt"
	"image"

	"otsu-obliterator/internal/gui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type View struct {
	window     fyne.Window
	controller *Controller

	toolbar        *widgets.Toolbar
	imageDisplay   *widgets.ImageDisplay
	parameterPanel *widgets.ParameterPanel
	mainContainer  *fyne.Container
}

func NewView(window fyne.Window) *View {
	view := &View{
		window: window,
	}

	view.setupComponents()
	view.setupLayout()

	return view
}

func (v *View) SetController(controller *Controller) {
	v.controller = controller
	v.setupEventHandlers()
}

func (v *View) setupComponents() {
	v.toolbar = widgets.NewToolbar()
	v.imageDisplay = widgets.NewImageDisplay()
	v.parameterPanel = widgets.NewParameterPanel()
}

func (v *View) setupLayout() {
	v.mainContainer = container.NewVBox(
		v.imageDisplay.GetContainer(),
		v.toolbar.GetContainer(),
		v.parameterPanel.GetContainer(),
	)
}

func (v *View) setupEventHandlers() {
	if v.controller == nil {
		return
	}

	v.toolbar.SetLoadHandler(v.controller.LoadImage)
	v.toolbar.SetSaveHandler(v.controller.SaveImage)
	v.toolbar.SetProcessHandler(v.controller.ProcessImage)
	v.toolbar.SetAlgorithmChangeHandler(v.controller.ChangeAlgorithm)

	v.parameterPanel.SetParameterChangeHandler(v.controller.UpdateParameter)
}

func (v *View) GetMainContainer() *fyne.Container {
	return v.mainContainer
}

func (v *View) SetOriginalImage(img image.Image) {
	if v.controller != nil {
		v.controller.logger.Debug("View", "SetOriginalImage called", map[string]interface{}{
			"image_nil":  img == nil,
			"image_type": fmt.Sprintf("%T", img),
		})
	}

	v.imageDisplay.SetOriginalImage(img)

	if v.controller != nil {
		v.controller.logger.Debug("View", "SetOriginalImage completed", nil)
	}
}

func (v *View) SetPreviewImage(img image.Image) {
	if v.controller != nil {
		v.controller.logger.Debug("View", "SetPreviewImage called", map[string]interface{}{
			"image_nil":  img == nil,
			"image_type": fmt.Sprintf("%T", img),
		})
	}

	v.imageDisplay.SetPreviewImage(img)

	if v.controller != nil {
		v.controller.logger.Debug("View", "SetPreviewImage completed", nil)
	}
}

func (v *View) UpdateParameterPanel(algorithm string, params map[string]interface{}) {
	v.parameterPanel.UpdateParameters(algorithm, params)
}

func (v *View) SetStatus(status string) {
	v.toolbar.SetStatus(status)
}

func (v *View) SetProgress(progress string) {
	// Progress is now handled through status messages
}

func (v *View) SetStage(stage string) {
	// Stages are now handled through status messages
}

func (v *View) SetMetrics(psnr, ssim float64) {
	v.toolbar.SetMetrics(psnr, ssim)
}

func (v *View) ShowError(title string, err error) {
	dialog.ShowError(err, v.window)
}

func (v *View) ShowFileDialog(callback func(fyne.URIReadCloser, error)) {
	dialog.ShowFileOpen(callback, v.window)
}

func (v *View) ShowSaveDialog(callback func(fyne.URIWriteCloser, error)) {
	dialog.ShowFileSave(callback, v.window)
}

func (v *View) ShowFormatSelectionDialog(callback func(string, bool)) {
	content := widget.NewLabel("No file extension detected. Please choose a format:")

	formatSelect := widget.NewSelect([]string{"PNG", "JPEG"}, nil)
	formatSelect.SetSelected("PNG")

	form := container.NewVBox(
		content,
		formatSelect,
	)

	dialog.ShowCustomConfirm("Choose File Format", "Save", "Cancel",
		form, func(confirmed bool) {
			if confirmed && formatSelect.Selected != "" {
				callback(formatSelect.Selected, true)
			} else {
				callback("", false)
			}
		}, v.window)
}

func (v *View) GetWindow() fyne.Window {
	return v.window
}

func (v *View) Show() {
	v.window.SetContent(v.mainContainer)
	v.window.Show()
}

func (v *View) Shutdown() {
}
