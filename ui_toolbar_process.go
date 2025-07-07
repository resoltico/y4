package main

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

func (t *Toolbar) handleProcessImage() {
	params := t.app.parameters.GetCurrentParameters()
	t.handleProcessImageWithParams(params)
}

func (t *Toolbar) handleProcessImageWithParams(params *OtsuParameters) {
	originalData := t.app.processing.GetOriginalImage()
	if originalData == nil {
		return
	}

	if t.processingInProgress {
		if t.cancelProcessing != nil {
			t.cancelProcessing()
		}
		return
	}

	t.processingInProgress = true
	t.app.parameters.SetStatus("Processing...")
	t.processButton.SetText("Cancel")

	t.currentProcessingCtx, t.cancelProcessing = context.WithCancel(context.Background())

	go func() {
		defer func() {
			fyne.Do(func() {
				t.processingInProgress = false
				t.processButton.SetText("Process")
			})
		}()

		startTime := time.Now()
		debugSystem := GetDebugSystem()
		imageSize := [2]int{originalData.Width, originalData.Height}

		method := t.getProcessingMethodName(params)
		opID := debugSystem.TraceProcessingStart(method, params, imageSize)

		DebugTraceMemory("before_processing")

		if err := validateOtsuParameters(params, imageSize); err != nil {
			processingDuration := time.Since(startTime)
			debugSystem.TraceValidationError(err, "parameter_validation")
			debugSystem.TraceProcessingEnd(opID, processingDuration, false, err.Error())

			fyne.Do(func() {
				dialog.ShowError(err, t.app.window)
				t.app.parameters.SetStatus("Parameter validation failed")
			})
			return
		}

		result, metrics, err := t.app.processing.ProcessImageWithTimeout(t.currentProcessingCtx, params)
		processingDuration := time.Since(startTime)

		DebugTraceMemory("after_processing")

		if err != nil {
			debugSystem.TraceProcessingEnd(opID, processingDuration, false, err.Error())

			fyne.Do(func() {
				if t.currentProcessingCtx.Err() == context.Canceled {
					t.app.parameters.SetStatus("Processing cancelled")
				} else {
					dialog.ShowError(err, t.app.window)
					t.app.parameters.SetStatus("Processing failed")
				}
			})
			return
		}

		debugSystem.TraceProcessingEnd(opID, processingDuration, true, "")
		debugSystem.TraceImageOperation(opID, method, imageSize, [2]int{result.Width, result.Height}, processingDuration)

		if metrics != nil {
			debugSystem.TraceThresholdCalculation(opID, [2]int{0, 0}, metrics.FMeasure())
		}

		fyne.Do(func() {
			t.app.imageViewer.SetProcessedImage(result.Image)
			t.app.parameters.SetStatus("Processing complete")
			t.app.parameters.SetMetrics(metrics)
			t.app.parameters.SetProcessingDetails(params, result, metrics)
			t.saveButton.Enable()

			DebugTraceParam("ProcessingComplete", method, fmt.Sprintf("duration=%dms", processingDuration.Milliseconds()))
		})
	}()
}

func (t *Toolbar) getProcessingMethodName(params *OtsuParameters) string {
	if params.MultiScaleProcessing {
		return fmt.Sprintf("multi_scale_%d_levels", params.PyramidLevels)
	} else if params.RegionAdaptiveThresholding {
		return fmt.Sprintf("region_adaptive_%d_grid", params.RegionGridSize)
	}
	return "single_scale"
}

func (t *Toolbar) CancelCurrentProcessing() {
	if t.processingInProgress && t.cancelProcessing != nil {
		t.cancelProcessing()
		t.app.parameters.SetStatus("Processing cancelled")
	}
}
