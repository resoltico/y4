package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

func (t *Toolbar) handleLoadImage() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			DebugTraceParam("LoadDialog", "closed", "cancelled_or_error")
			return
		}
		defer reader.Close()

		startTime := time.Now()
		debugSystem := GetDebugSystem()
		opID := debugSystem.TraceProcessingStart("image_load", &OtsuParameters{}, [2]int{0, 0})

		t.app.parameters.SetStatus("Loading image...")
		DebugTraceMemory("before_image_load")

		imageData, loadErr := LoadImageFromReader(reader)
		loadDuration := time.Since(startTime)

		if loadErr != nil {
			debugSystem.TraceProcessingEnd(opID, loadDuration, false, loadErr.Error())
			dialog.ShowError(loadErr, t.app.window)
			t.app.parameters.SetStatus("Load failed")
			return
		}

		debugSystem.TraceProcessingEnd(opID, loadDuration, true, "")
		debugSystem.TraceImageOperation(opID, "load", [2]int{0, 0}, [2]int{imageData.Width, imageData.Height}, loadDuration)
		DebugTraceMemory("after_image_load")

		fyne.Do(func() {
			t.app.imageViewer.SetOriginalImage(imageData.Image)
			t.app.processing.SetOriginalImage(imageData)
			t.processButton.Enable()
			t.app.parameters.SetStatus("Image loaded")
			t.app.parameters.SetDetails(fmt.Sprintf("Image: %dx%d pixels, %d channels, %s format",
				imageData.Width, imageData.Height, imageData.Channels, imageData.Format))

			DebugTraceParam("ImageLoaded", "none", fmt.Sprintf("%dx%d", imageData.Width, imageData.Height))
		})
	}, t.app.window)
}
