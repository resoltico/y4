package gui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"otsu-obliterator/internal/algorithms"
	"otsu-obliterator/internal/logger"
	"otsu-obliterator/internal/pipeline"

	"fyne.io/fyne/v2"
)

type Controller struct {
	view             *View
	coordinator      pipeline.ProcessingCoordinator
	algorithmManager *algorithms.Manager
	logger           logger.Logger

	mu                sync.RWMutex
	currentAlgorithm  string
	currentParameters map[string]interface{}
	processingActive  bool

	processCtx    context.Context
	processCancel context.CancelFunc
}

func NewController(coord pipeline.ProcessingCoordinator, log logger.Logger) *Controller {
	return &Controller{
		coordinator:       coord,
		algorithmManager:  algorithms.NewManager(),
		logger:            log,
		currentAlgorithm:  "2D Otsu",
		currentParameters: make(map[string]interface{}),
	}
}

func (c *Controller) SetView(view *View) {
	c.view = view
	c.initializeDefaultParameters()
}

func (c *Controller) initializeDefaultParameters() {
	params := c.algorithmManager.GetParameters(c.currentAlgorithm)
	c.mu.Lock()
	c.currentParameters = params
	c.mu.Unlock()

	fyne.Do(func() {
		c.view.UpdateParameterPanel(c.currentAlgorithm, params)
	})
}

func (c *Controller) LoadImage() {
	c.view.ShowFileDialog(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			c.handleError("File selection error", err)
			return
		}
		if reader == nil {
			return
		}

		fyne.Do(func() {
			c.updateStatus("Loading image...")
			c.updateStage("Reading file data...")
		})

		go func() {
			defer reader.Close()

			start := time.Now()
			imageData, loadErr := c.coordinator.LoadImage(reader)

			fyne.Do(func() {
				if loadErr != nil {
					c.handleError("Image load error", loadErr)
					c.updateStatus("Ready")
					c.updateStage("")
					return
				}

				c.view.SetPreviewImage(nil)
				c.view.SetOriginalImage(imageData.Image)
				c.updateStatus("Image loaded")
				c.updateStage("")

				c.view.GetMainContainer().Refresh()

				c.logger.Info("Controller", "image loaded", map[string]interface{}{
					"width":     imageData.Width,
					"height":    imageData.Height,
					"format":    imageData.Format,
					"load_time": time.Since(start),
				})
			})
		}()
	})
}

func (c *Controller) SaveImage() {
	processedImg := c.coordinator.GetProcessedImage()
	if processedImg == nil {
		c.handleError("Save error", fmt.Errorf("no processed image to save"))
		return
	}

	c.view.ShowSaveDialog(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			c.handleError("File save error", err)
			return
		}
		if writer == nil {
			return
		}

		ext := strings.ToLower(writer.URI().Extension())
		if ext == "" {
			c.showFormatSelectionDialog(writer, processedImg)
			return
		}

		c.saveImageWithWriter(writer, processedImg)
	})
}

func (c *Controller) ChangeAlgorithm(algorithm string) {
	// Only support 2D Otsu algorithm
	if algorithm != "2D Otsu" {
		c.handleError("Algorithm error", fmt.Errorf("unsupported algorithm: %s", algorithm))
		return
	}

	c.mu.Lock()
	c.currentAlgorithm = algorithm
	c.mu.Unlock()

	if err := c.algorithmManager.SetCurrentAlgorithm(algorithm); err != nil {
		c.handleError("Algorithm change error", err)
		return
	}

	params := c.algorithmManager.GetParameters(algorithm)
	c.mu.Lock()
	c.currentParameters = params
	c.mu.Unlock()

	fyne.Do(func() {
		c.view.UpdateParameterPanel(algorithm, params)
	})
}

func (c *Controller) UpdateParameter(name string, value interface{}) {
	c.mu.Lock()
	algorithm := c.currentAlgorithm
	c.currentParameters[name] = value
	c.mu.Unlock()

	err := c.algorithmManager.SetParameter(algorithm, name, value)
	if err != nil {
		c.handleError("Parameter update error", err)
		return
	}
}

func (c *Controller) ProcessImage() {
	if c.isProcessing() {
		return
	}

	originalImg := c.coordinator.GetOriginalImage()
	if originalImg == nil {
		c.handleError("Processing error", fmt.Errorf("no image loaded"))
		return
	}

	c.setProcessing(true)
	fyne.Do(func() {
		c.updateStatus("Initializing...")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	c.mu.Lock()
	c.processCtx = ctx
	c.processCancel = cancel
	c.mu.Unlock()

	go func() {
		defer func() {
			c.setProcessing(false)
			fyne.Do(func() {
				c.updateStatus("Ready")
			})
			cancel()
		}()

		algorithm := c.getCurrentAlgorithm()
		params := c.getCurrentParameters()

		stages := c.getProcessingStages()
		stageCount := len(stages)

		for i, stage := range stages {
			select {
			case <-ctx.Done():
				return
			default:
			}

			fyne.Do(func() {
				stageProgress := fmt.Sprintf("Stage %d/%d: %s", i+1, stageCount, stage)
				c.updateStatus(stageProgress)
			})

			time.Sleep(300 * time.Millisecond)
		}

		start := time.Now()
		processedImg, err := c.coordinator.ProcessImageWithContext(ctx, algorithm, params)
		processingTime := time.Since(start)

		fyne.Do(func() {
			if err != nil {
				c.handleError("Processing error", err)
				c.updateStatus("Processing failed")
				return
			}

			if processedImg != nil {
				c.view.SetPreviewImage(processedImg.Image)
				c.updateMetrics(originalImg, processedImg)
				c.updateStatus("Processing completed")

				c.logger.Info("Controller", "processing completed", map[string]interface{}{
					"algorithm":       algorithm,
					"width":           processedImg.Width,
					"height":          processedImg.Height,
					"processing_time": processingTime,
				})
			} else {
				c.updateStatus("Processing failed - no result")
			}
		})
	}()
}

func (c *Controller) getProcessingStages() []string {
	return []string{
		"Converting to grayscale",
		"Applying preprocessing",
		"Building 2D histogram",
		"Finding threshold",
		"Applying threshold",
		"Finalizing result",
	}
}

func (c *Controller) CancelProcessing() {
	c.mu.Lock()
	if c.processCancel != nil {
		c.processCancel()
	}
	c.mu.Unlock()
}

func (c *Controller) updateStatus(status string) {
	c.view.SetStatus(status)
}

func (c *Controller) updateStage(stage string) {
	c.view.SetStage(stage)
}

func (c *Controller) updateMetrics(original, processed *pipeline.ImageData) {
	psnr := c.coordinator.CalculatePSNR(original, processed)
	ssim := c.coordinator.CalculateSSIM(original, processed)
	c.view.SetMetrics(psnr, ssim)
}

func (c *Controller) handleError(title string, err error) {
	c.logger.Error("Controller", err, map[string]interface{}{
		"title": title,
	})

	fyne.Do(func() {
		c.view.ShowError(title, err)
	})
}

func (c *Controller) isProcessing() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.processingActive
}

func (c *Controller) setProcessing(active bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.processingActive = active
}

func (c *Controller) getCurrentAlgorithm() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentAlgorithm
}

func (c *Controller) getCurrentParameters() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range c.currentParameters {
		result[k] = v
	}
	return result
}

func (c *Controller) Shutdown() {
	c.CancelProcessing()
	c.logger.Info("Controller", "shutdown completed", nil)
}

func (c *Controller) showFormatSelectionDialog(writer fyne.URIWriteCloser, processedImg *pipeline.ImageData) {
	originalPath := writer.URI().Path()
	writer.Close()

	if err := os.Remove(originalPath); err != nil {
		c.logger.Debug("Controller", "failed to remove empty file", map[string]interface{}{
			"path":  originalPath,
			"error": err.Error(),
		})
	}

	fyne.Do(func() {
		c.view.ShowFormatSelectionDialog(func(format string, confirmed bool) {
			if !confirmed {
				return
			}

			c.saveImageWithFormat(originalPath, processedImg, format)
		})
	})
}

func (c *Controller) saveImageWithFormat(imagePath string, processedImg *pipeline.ImageData, format string) {
	fyne.Do(func() {
		c.updateStatus("Saving image...")
		c.updateStage("Writing file...")
	})

	go func() {
		ext := ".png"
		if format == "JPEG" {
			ext = ".jpg"
		}

		finalPath := imagePath + ext

		file, err := os.Create(finalPath)
		if err != nil {
			fyne.Do(func() {
				c.handleError("File create error", err)
				c.updateStage("")
			})
			return
		}
		defer file.Close()

		saveErr := c.coordinator.SaveImageToWriter(file, processedImg, format)

		fyne.Do(func() {
			if saveErr != nil {
				c.handleError("Image save error", saveErr)
			} else {
				c.updateStatus("Image saved")
				c.logger.Info("Controller", "image saved with format", map[string]interface{}{
					"path":   finalPath,
					"format": format,
				})
			}
			c.updateStage("")
		})
	}()
}

func (c *Controller) saveImageWithWriter(writer fyne.URIWriteCloser, processedImg *pipeline.ImageData) {
	fyne.Do(func() {
		c.updateStatus("Saving image...")
		c.updateStage("Writing file...")
	})

	go func() {
		defer writer.Close()

		start := time.Now()
		saveErr := c.coordinator.SaveImage(writer, processedImg)

		fyne.Do(func() {
			if saveErr != nil {
				c.handleError("Image save error", saveErr)
			} else {
				c.updateStatus("Image saved")
				c.logger.Info("Controller", "image saved", map[string]interface{}{
					"path":      writer.URI().Path(),
					"save_time": time.Since(start),
				})
			}
			c.updateStage("")
		})
	}()
}
