package main

import "fmt"

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
