package pipeline

import (
	"context"
	"fmt"

	"otsu-obliterator/internal/algorithms"
	"otsu-obliterator/internal/logger"
	"otsu-obliterator/internal/opencv/bridge"
	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/opencv/safe"
)

type imageProcessor struct {
	memoryManager    *memory.Manager
	logger           logger.Logger
	algorithmManager *algorithms.Manager
}

type ContextualAlgorithm interface {
	ProcessWithContext(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error)
}

func (p *imageProcessor) ProcessImage(inputData *ImageData, algorithm algorithms.Algorithm, params map[string]interface{}) (*ImageData, error) {
	return p.ProcessImageWithContext(context.Background(), inputData, algorithm, params)
}

func (p *imageProcessor) ProcessImageWithContext(ctx context.Context, inputData *ImageData, algorithm algorithms.Algorithm, params map[string]interface{}) (*ImageData, error) {
	if err := safe.ValidateMatForOperation(inputData.Mat, "ProcessImage"); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	var resultMat *safe.Mat
	var err error

	if contextualAlg, ok := algorithm.(ContextualAlgorithm); ok {
		resultMat, err = contextualAlg.ProcessWithContext(ctx, inputData.Mat, params)
	} else {
		resultMat, err = algorithm.Process(inputData.Mat, params)
	}

	if err != nil {
		return nil, fmt.Errorf("algorithm processing failed: %w", err)
	}

	if resultMat == nil {
		return nil, fmt.Errorf("algorithm returned nil result")
	}

	select {
	case <-ctx.Done():
		p.memoryManager.ReleaseMat(resultMat, "processing_result")
		return nil, ctx.Err()
	default:
	}

	resultImage, err := bridge.MatToImage(resultMat)
	if err != nil {
		p.memoryManager.ReleaseMat(resultMat, "processing_result")
		return nil, fmt.Errorf("Mat to image conversion failed: %w", err)
	}

	if resultImage == nil {
		p.memoryManager.ReleaseMat(resultMat, "processing_result")
		return nil, fmt.Errorf("Mat to image conversion returned nil")
	}

	bounds := resultImage.Bounds()
	processedData := &ImageData{
		Image:       resultImage,
		Mat:         resultMat,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		Channels:    resultMat.Channels(),
		Format:      inputData.Format,
		OriginalURI: inputData.OriginalURI,
	}

	p.logger.Info("ImageProcessor", "processing completed", map[string]interface{}{
		"algorithm":   algorithm.GetName(),
		"input_size":  fmt.Sprintf("%dx%d", inputData.Width, inputData.Height),
		"output_size": fmt.Sprintf("%dx%d", processedData.Width, processedData.Height),
		"image_type":  fmt.Sprintf("%T", resultImage),
	})

	return processedData, nil
}
