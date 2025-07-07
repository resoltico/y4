package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

func (t *Toolbar) handleSaveImage() {
	processedData := t.app.processing.GetProcessedImage()
	if processedData == nil {
		return
	}

	t.app.parameters.SetStatus("Preparing save...")

	t.fileSaveMenu.ShowSaveDialog(processedData, func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, t.app.window)
			t.app.parameters.SetStatus("Save failed")
			return
		}

		if writer != nil {
			t.app.parameters.SetStatus("Image saved")
			DebugTraceParam("ImageSaved", "none", writer.URI().String())
		}
	})
}
