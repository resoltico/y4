package main

import (
	"fmt"
	"log"

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
			log.Printf("[DEBUG] Load dialog cancelled or error: %v", err)
			return
		}
		defer reader.Close()

		log.Printf("[DEBUG] Loading image from: %s", reader.URI().String())
		t.SetStatus("Loading image...")

		imageData, loadErr := LoadImageFromReader(reader)
		if loadErr != nil {
			log.Printf("[ERROR] Image load failed: %v", loadErr)
			dialog.ShowError(loadErr, t.app.window)
			t.SetStatus("Load failed")
			return
		}

		log.Printf("[DEBUG] Image loaded successfully: %dx%d, %d channels, format: %s",
			imageData.Width, imageData.Height, imageData.Channels, imageData.Format)

		fyne.Do(func() {
			t.app.imageViewer.SetOriginalImage(imageData.Image)
			t.app.processing.SetOriginalImage(imageData)
			t.processButton.Enable()
			t.SetStatus("Image loaded")
			t.SetDetails(fmt.Sprintf("Image: %dx%d pixels, %d channels, %s format",
				imageData.Width, imageData.Height, imageData.Channels, imageData.Format))
		})
	}, t.app.window)
}

func (t *Toolbar) handleSaveImage() {
	processedData := t.app.processing.GetProcessedImage()
	if processedData == nil {
		log.Printf("[DEBUG] No processed image to save")
		return
	}

	log.Printf("[DEBUG] Starting save dialog")
	t.SetStatus("Preparing save...")

	t.fileSaveMenu.ShowSaveDialog(processedData, func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			log.Printf("[ERROR] Save failed: %v", err)
			dialog.ShowError(err, t.app.window)
			t.SetStatus("Save failed")
			return
		}

		if writer != nil {
			log.Printf("[DEBUG] Image saved successfully to: %s", writer.URI().String())
			t.SetStatus("Image saved")
		}
	})
}

func (t *Toolbar) handleProcessImage() {
	originalData := t.app.processing.GetOriginalImage()
	if originalData == nil {
		log.Printf("[ERROR] No original image for processing")
		return
	}

	log.Printf("[DEBUG] Starting image processing")
	t.SetStatus("Processing...")
	t.processButton.Disable()

	go func() {
		params := t.app.parameters.GetParameters()
		log.Printf("[DEBUG] Processing parameters: method=%s, window=%d, bins=%d",
			getProcessingMethodName(params), params.WindowSize, params.HistogramBins)

		result, metrics, err := t.app.processing.ProcessImage(params)

		if err != nil {
			log.Printf("[ERROR] Processing failed: %v", err)
		} else {
			log.Printf("[DEBUG] Processing completed successfully")
			if metrics != nil {
				log.Printf("[DEBUG] Metrics calculated: F=%.3f, pF=%.3f, NRM=%.3f, DRD=%.3f",
					metrics.FMeasure(), metrics.PseudoFMeasure(), metrics.NRM(), metrics.DRD())
			}
		}

		fyne.Do(func() {
			t.processButton.Enable()

			if err != nil {
				dialog.ShowError(err, t.app.window)
				t.SetStatus("Processing failed")
				return
			}

			t.app.imageViewer.SetProcessedImage(result.Image)
			t.SetStatus("Processing complete")
			t.SetMetrics(metrics)
			t.SetProcessingDetails(params, result, metrics)
			t.saveButton.Enable()
		})
	}()
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
}

func (t *Toolbar) SetProcessingDetails(params *OtsuParameters, result *ImageData, metrics *BinaryImageMetrics) {
	if params == nil || result == nil || metrics == nil {
		return
	}

	processingMethod := getProcessingMethodName(params)

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

func getProcessingMethodName(params *OtsuParameters) string {
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
