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

	t.SetStatus("Preparing save...")

	t.fileSaveMenu.ShowSaveDialog(processedData, func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, t.app.window)
			t.SetStatus("Save failed")
			return
		}

		if writer != nil {
			t.SetStatus("Image saved")
			DebugTraceParam("ImageSaved", "none", writer.URI().String())
		}
	})
}
