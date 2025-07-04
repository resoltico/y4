package main

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type Toolbar struct {
	app       *Application
	container *fyne.Container

	loadButton    *widget.Button
	saveButton    *widget.Button
	processButton *widget.Button
	statusLabel   *widget.Label
	metricsLabel  *widget.Label
	detailsLabel  *widget.Label
	fileSaveMenu  *FileSaveMenu

	processingInProgress bool
	currentProcessingCtx context.Context
	cancelProcessing     context.CancelFunc
}

func NewToolbar(app *Application) *Toolbar {
	t := &Toolbar{
		app: app,
	}

	t.createButtons()
	t.createLabels()
	t.fileSaveMenu = NewFileSaveMenu(app.window)
	t.buildLayout()

	return t
}

func (t *Toolbar) createButtons() {
	t.loadButton = widget.NewButton("Load Image", t.handleLoadImage)
	t.loadButton.Importance = widget.HighImportance

	t.saveButton = widget.NewButton("Save Result", t.handleSaveImage)
	t.saveButton.Importance = widget.HighImportance
	t.saveButton.Disable()

	t.processButton = widget.NewButton("Process", t.handleProcessImage)
	t.processButton.Importance = widget.HighImportance
	t.processButton.Disable()
}

func (t *Toolbar) createLabels() {
	t.statusLabel = widget.NewLabel("Ready")
	t.metricsLabel = widget.NewLabel("No metrics available")
	t.detailsLabel = widget.NewLabel("Load an image to begin processing")
}

func (t *Toolbar) buildLayout() {
	leftSection := container.NewHBox(t.loadButton, t.saveButton)
	centerSection := container.NewHBox(t.processButton)

	metricsSection := container.NewVBox(
		t.statusLabel,
		t.metricsLabel,
		t.detailsLabel,
	)

	t.container = container.NewBorder(
		nil, nil, leftSection, metricsSection, centerSection,
	)
}

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

		t.SetStatus("Loading image...")
		DebugTraceMemory("before_image_load")

		imageData, loadErr := LoadImageFromReader(reader)
		loadDuration := time.Since(startTime)

		if loadErr != nil {
			debugSystem.TraceProcessingEnd(opID, loadDuration, false, loadErr.Error())
			dialog.ShowError(loadErr, t.app.window)
			t.SetStatus("Load failed")
			return
		}

		debugSystem.TraceProcessingEnd(opID, loadDuration, true, "")
		debugSystem.TraceImageOperation(opID, "load", [2]int{0, 0}, [2]int{imageData.Width, imageData.Height}, loadDuration)
		DebugTraceMemory("after_image_load")

		fyne.Do(func() {
			t.app.imageViewer.SetOriginalImage(imageData.Image)
			t.app.processing.SetOriginalImage(imageData)
			t.processButton.Enable()
			t.SetStatus("Image loaded")
			t.SetDetails(fmt.Sprintf("Image: %dx%d pixels, %d channels, %s format",
				imageData.Width, imageData.Height, imageData.Channels, imageData.Format))

			DebugTraceParam("ImageLoaded", "none", fmt.Sprintf("%dx%d", imageData.Width, imageData.Height))
		})
	}, t.app.window)
}

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
		// Cancel current processing
		if t.cancelProcessing != nil {
			t.cancelProcessing()
		}
		return
	}

	t.processingInProgress = true
	t.SetStatus("Processing...")
	t.processButton.SetText("Cancel")

	// Create processing context with timeout
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

		// Validate parameters before processing
		if err := validateOtsuParameters(params, imageSize); err != nil {
			processingDuration := time.Since(startTime)
			debugSystem.TraceValidationError(err, "parameter_validation")
			debugSystem.TraceProcessingEnd(opID, processingDuration, false, err.Error())

			fyne.Do(func() {
				dialog.ShowError(err, t.app.window)
				t.SetStatus("Parameter validation failed")
			})
			return
		}

		// Process with timeout and validation
		result, metrics, err := t.app.processing.ProcessImageWithTimeout(t.currentProcessingCtx, params)
		processingDuration := time.Since(startTime)

		DebugTraceMemory("after_processing")

		if err != nil {
			debugSystem.TraceProcessingEnd(opID, processingDuration, false, err.Error())

			fyne.Do(func() {
				if t.currentProcessingCtx.Err() == context.Canceled {
					t.SetStatus("Processing cancelled")
				} else {
					dialog.ShowError(err, t.app.window)
					t.SetStatus("Processing failed")
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
			t.SetStatus("Processing complete")
			t.SetMetrics(metrics)
			t.SetProcessingDetails(params, result, metrics)
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

func (t *Toolbar) SetStatus(status string) {
	t.statusLabel.SetText("Status: " + status)
}

func (t *Toolbar) SetDetails(details string) {
	t.detailsLabel.SetText(details)
}

func (t *Toolbar) SetMetrics(metrics *BinaryImageMetrics) {
	if metrics == nil {
		t.metricsLabel.SetText("No metrics available")
		return
	}

	basicMetrics := fmt.Sprintf("F: %.3f | pF: %.3f | NRM: %.3f | DRD: %.3f",
		metrics.FMeasure(),
		metrics.PseudoFMeasure(),
		metrics.NRM(),
		metrics.DRD(),
	)

	t.metricsLabel.SetText(basicMetrics)

	// Debug trace metrics
	debugSystem := GetDebugSystem()
	debugSystem.logger.Info("metrics calculated",
		"f_measure", metrics.FMeasure(),
		"pseudo_f_measure", metrics.PseudoFMeasure(),
		"nrm", metrics.NRM(),
		"drd", metrics.DRD(),
		"mpm", metrics.MPM(),
		"bfc", metrics.BackgroundForegroundContrast(),
		"skeleton", metrics.SkeletonSimilarity(),
	)
}

func (t *Toolbar) SetProcessingDetails(params *OtsuParameters, result *ImageData, metrics *BinaryImageMetrics) {
	if params == nil || result == nil || metrics == nil {
		return
	}

	processingMethod := t.getProcessingMethodDisplayName(params)

	advancedMetrics := fmt.Sprintf("MPM: %.3f | BFC: %.3f | Skeleton: %.3f | Method: %s",
		metrics.MPM(),
		metrics.BackgroundForegroundContrast(),
		metrics.SkeletonSimilarity(),
		processingMethod,
	)

	algorithmDetails := fmt.Sprintf("Window: %d | Bins: %d | Neighborhood: %s | Preprocessing: %s",
		params.WindowSize,
		params.HistogramBins,
		params.NeighborhoodType,
		t.getPreprocessingDescription(params),
	)

	confusionMatrix := fmt.Sprintf("TP: %d | TN: %d | FP: %d | FN: %d | Total: %d",
		metrics.TruePositives,
		metrics.TrueNegatives,
		metrics.FalsePositives,
		metrics.FalseNegatives,
		metrics.TotalPixels,
	)

	detailsText := fmt.Sprintf("%s\n%s\n%s",
		advancedMetrics,
		algorithmDetails,
		confusionMatrix,
	)

	t.SetDetails(detailsText)
}

func (t *Toolbar) getProcessingMethodDisplayName(params *OtsuParameters) string {
	if params.MultiScaleProcessing {
		return fmt.Sprintf("Multi-Scale (%d levels)", params.PyramidLevels)
	} else if params.RegionAdaptiveThresholding {
		return fmt.Sprintf("Region Adaptive (%dx%d grid)", params.RegionGridSize, params.RegionGridSize)
	}
	return "Single Scale"
}

func (t *Toolbar) getPreprocessingDescription(params *OtsuParameters) string {
	var steps []string

	if params.HomomorphicFiltering {
		steps = append(steps, "Homomorphic")
	}
	if params.AnisotropicDiffusion {
		steps = append(steps, fmt.Sprintf("Diffusion(%d)", params.DiffusionIterations))
	}
	if params.GaussianPreprocessing {
		steps = append(steps, "Gaussian")
	}
	if params.ApplyContrastEnhancement {
		steps = append(steps, "CLAHE")
	}

	if len(steps) == 0 {
		return "None"
	}

	result := ""
	for i, step := range steps {
		if i > 0 {
			result += "+"
		}
		result += step
	}
	return result
}

func (t *Toolbar) GetContainer() *fyne.Container {
	return t.container
}

func (t *Toolbar) CancelCurrentProcessing() {
	if t.processingInProgress && t.cancelProcessing != nil {
		t.cancelProcessing()
		t.SetStatus("Processing cancelled")
	}
}
