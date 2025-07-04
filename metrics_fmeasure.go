package main

func (m *BinaryImageMetrics) FMeasure() float64 {
	if m.TruePositives+m.FalsePositives == 0 || m.TruePositives+m.FalseNegatives == 0 {
		return 0.0
	}

	precision := float64(m.TruePositives) / float64(m.TruePositives+m.FalsePositives)
	recall := float64(m.TruePositives) / float64(m.TruePositives+m.FalseNegatives)

	if precision+recall == 0 {
		return 0.0
	}

	return 2 * (precision * recall) / (precision + recall)
}

func (m *BinaryImageMetrics) PseudoFMeasure() float64 {
	if m.TruePositives == 0 {
		return 0.0
	}
	if m.TruePositives+m.FalsePositives == 0 || m.TruePositives+m.FalseNegatives == 0 {
		return 0.0
	}

	precision := float64(m.TruePositives) / float64(m.TruePositives+m.FalsePositives)
	recall := float64(m.TruePositives) / float64(m.TruePositives+m.FalseNegatives)

	// DIBCO standard pseudo F-measure with β = 0.5
	beta := 0.5
	betaSquared := beta * beta

	if betaSquared*precision+recall == 0 {
		return 0.0
	}

	return (1 + betaSquared) * precision * recall / (betaSquared*precision + recall)
}

func (m *BinaryImageMetrics) NRM() float64 {
	fn := float64(m.FalseNegatives)
	fp := float64(m.FalsePositives)
	tp := float64(m.TruePositives)
	tn := float64(m.TrueNegatives)

	// Standard DIBCO NRM calculation
	numerator := fn + fp
	denominator := 2 * (tp + tn)

	if denominator == 0 {
		return 1.0
	}

	return numerator / denominator
}

func (m *BinaryImageMetrics) DRD() float64 {
	return m.drdValue
}

func (m *BinaryImageMetrics) MPM() float64 {
	return m.mpmValue
}

func (m *BinaryImageMetrics) BackgroundForegroundContrast() float64 {
	return m.pbcValue
}

func (m *BinaryImageMetrics) SkeletonSimilarity() float64 {
	return m.skeletonValue
}
