package main

import (
	"fmt"
	"image"
	"math"

	"gocv.io/x/gocv"
)

type BinaryImageMetrics struct {
	TruePositives  int
	TrueNegatives  int
	FalsePositives int
	FalseNegatives int
	TotalPixels    int

	drdValue      float64
	mpmValue      float64
	pbcValue      float64
	skeletonValue float64
	ocrValue      float64
}

type SkeletonMetrics struct {
	SkeletonSimilarity     float64
	BranchPointAccuracy    float64
	EndPointAccuracy       float64
	StrokeWidthConsistency float64
}

type OCRMetrics struct {
	CharacterAccuracy float64
	WordAccuracy      float64
	EditDistance      float64
}

func validateMat(mat gocv.Mat, context string) error {
	if mat.Empty() {
		return fmt.Errorf("%s: matrix is empty", context)
	}
	if mat.Rows() <= 0 || mat.Cols() <= 0 {
		return fmt.Errorf("%s: invalid dimensions %dx%d", context, mat.Rows(), mat.Cols())
	}
	return nil
}

func validateMatDimensions(mat1, mat2 gocv.Mat, context string) error {
	if mat1.Rows() != mat2.Rows() || mat1.Cols() != mat2.Cols() {
		return fmt.Errorf("%s: dimension mismatch %dx%d vs %dx%d",
			context, mat1.Rows(), mat1.Cols(), mat2.Rows(), mat2.Cols())
	}
	return nil
}

func CalculateBinaryMetrics(groundTruth, result gocv.Mat) *BinaryImageMetrics {
	if err := validateMat(groundTruth, "ground truth"); err != nil {
		return nil
	}
	if err := validateMat(result, "result"); err != nil {
		return nil
	}
	if err := validateMatDimensions(groundTruth, result, "metrics calculation"); err != nil {
		return nil
	}

	metrics := &BinaryImageMetrics{}
	metrics.calculateConfusionMatrix(groundTruth, result)
	metrics.calculateDRD(groundTruth, result)
	metrics.calculateMPM(groundTruth, result)
	metrics.calculateBackgroundForegroundContrast(groundTruth, result)
	metrics.calculateSkeletonSimilarity(groundTruth, result)

	return metrics
}

func (m *BinaryImageMetrics) calculateConfusionMatrix(groundTruth, result gocv.Mat) {
	rows := groundTruth.Rows()
	cols := groundTruth.Cols()
	m.TotalPixels = rows * cols

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			gtValue := groundTruth.GetUCharAt(y, x)
			resValue := result.GetUCharAt(y, x)

			gtBinary := gtValue > 127
			resBinary := resValue > 127

			if gtBinary && resBinary {
				m.TruePositives++
			} else if !gtBinary && !resBinary {
				m.TrueNegatives++
			} else if !gtBinary && resBinary {
				m.FalsePositives++
			} else {
				m.FalseNegatives++
			}
		}
	}
}

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

func (m *BinaryImageMetrics) calculateDRD(groundTruth, result gocv.Mat) {
	rows := groundTruth.Rows()
	cols := groundTruth.Cols()

	weightMatrix := m.createDRDWeightMatrix()

	totalDistortion := 0.0
	totalErrorPixels := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			gtValue := groundTruth.GetUCharAt(y, x) > 127
			resValue := result.GetUCharAt(y, x) > 127

			if gtValue != resValue {
				totalErrorPixels++
				distortion := m.calculatePixelDRD(groundTruth, x, y, weightMatrix)
				totalDistortion += distortion
			}
		}
	}

	if totalErrorPixels == 0 {
		m.drdValue = 0.0
		return
	}

	totalForegroundPixels := 0
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if groundTruth.GetUCharAt(y, x) > 127 {
				totalForegroundPixels++
			}
		}
	}

	if totalForegroundPixels == 0 {
		m.drdValue = 0.0
		return
	}

	// DIBCO standard DRD normalization
	m.drdValue = totalDistortion / float64(totalForegroundPixels)
}

func (m *BinaryImageMetrics) createDRDWeightMatrix() [][]float64 {
	size := 5
	center := size / 2
	matrix := make([][]float64, size)

	for i := range matrix {
		matrix[i] = make([]float64, size)
		for j := range matrix[i] {
			dx := float64(i - center)
			dy := float64(j - center)
			distance := math.Sqrt(dx*dx + dy*dy)

			if distance == 0 {
				matrix[i][j] = 1.0
			} else {
				matrix[i][j] = 1.0 / distance
			}
		}
	}

	return matrix
}

func (m *BinaryImageMetrics) calculatePixelDRD(groundTruth gocv.Mat, x, y int, weightMatrix [][]float64) float64 {
	rows := groundTruth.Rows()
	cols := groundTruth.Cols()
	size := len(weightMatrix)
	center := size / 2

	weightedSum := 0.0
	totalWeight := 0.0

	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			nx := x + i - center
			ny := y + j - center

			if nx >= 0 && nx < cols && ny >= 0 && ny < rows {
				gtValue := groundTruth.GetUCharAt(ny, nx) > 127
				weight := weightMatrix[i][j]

				if gtValue {
					weightedSum += weight
				}
				totalWeight += weight
			}
		}
	}

	if totalWeight == 0 {
		return 0.0
	}

	return weightedSum / totalWeight
}

func (m *BinaryImageMetrics) calculateMPM(groundTruth, result gocv.Mat) {
	gtContours := m.extractContoursWithValidation(groundTruth, "ground truth")
	resContours := m.extractContoursWithValidation(result, "result")

	if len(gtContours) == 0 && len(resContours) == 0 {
		m.mpmValue = 0.0
		return
	}

	totalMismatch := 0.0
	totalObjects := 0

	// Calculate misalignment for ground truth objects
	for _, gtContour := range gtContours {
		if len(gtContour) == 0 {
			continue
		}
		totalObjects++

		minDistance := math.Inf(1)
		for _, resContour := range resContours {
			if len(resContour) == 0 {
				continue
			}
			distance := m.calculateContourDistance(gtContour, resContour)
			if distance < minDistance {
				minDistance = distance
			}
		}

		if minDistance != math.Inf(1) {
			totalMismatch += minDistance
		} else {
			// No matching contour found - use maximum possible distance
			fallbackDistance := float64(groundTruth.Rows() + groundTruth.Cols())
			totalMismatch += fallbackDistance
		}
	}

	// Calculate misalignment for result objects not in ground truth
	for _, resContour := range resContours {
		if len(resContour) == 0 {
			continue
		}

		minDistance := math.Inf(1)
		for _, gtContour := range gtContours {
			if len(gtContour) == 0 {
				continue
			}
			distance := m.calculateContourDistance(resContour, gtContour)
			if distance < minDistance {
				minDistance = distance
			}
		}

		// Only count result contours that don't match any ground truth contour
		if minDistance > 5.0 { // Threshold for "matching"
			totalObjects++
			if minDistance != math.Inf(1) {
				totalMismatch += minDistance
			} else {
				fallbackDistance := float64(groundTruth.Rows() + groundTruth.Cols())
				totalMismatch += fallbackDistance
			}
		}
	}

	if totalObjects == 0 {
		m.mpmValue = 0.0
		return
	}

	m.mpmValue = totalMismatch / float64(totalObjects)
}

func (m *BinaryImageMetrics) extractContoursWithValidation(mat gocv.Mat, context string) [][]image.Point {
	if err := validateMat(mat, context); err != nil {
		return [][]image.Point{}
	}

	contours := gocv.FindContours(mat, gocv.RetrievalExternal, gocv.ChainApproxSimple)

	if contours.IsNil() {
		return [][]image.Point{}
	}

	defer func() {
		if r := recover(); r != nil {
			// Recovery from potential OpenCV segfault
		}
	}()

	size := contours.Size()
	if size == 0 {
		contours.Close()
		return [][]image.Point{}
	}

	result := make([][]image.Point, 0, size)
	for i := 0; i < size; i++ {
		contour := contours.At(i)
		if !contour.IsNil() {
			points := contour.ToPoints()
			// Filter out very small contours (noise)
			if len(points) > 10 {
				result = append(result, points)
			}
		}
	}

	contours.Close()
	return result
}

func (m *BinaryImageMetrics) calculateContourDistance(contour1, contour2 []image.Point) float64 {
	if len(contour1) == 0 || len(contour2) == 0 {
		return math.Inf(1)
	}

	// Use Hausdorff distance for more accurate contour comparison
	distance1 := m.calculateDirectedHausdorffDistance(contour1, contour2)
	distance2 := m.calculateDirectedHausdorffDistance(contour2, contour1)

	return math.Max(distance1, distance2)
}

func (m *BinaryImageMetrics) calculateDirectedHausdorffDistance(contour1, contour2 []image.Point) float64 {
	maxDistance := 0.0

	for _, p1 := range contour1 {
		minDistance := math.Inf(1)

		for _, p2 := range contour2 {
			dx := float64(p1.X - p2.X)
			dy := float64(p1.Y - p2.Y)
			distance := math.Sqrt(dx*dx + dy*dy)

			if distance < minDistance {
				minDistance = distance
			}
		}

		if minDistance > maxDistance && minDistance != math.Inf(1) {
			maxDistance = minDistance
		}
	}

	return maxDistance
}

func (m *BinaryImageMetrics) calculateBackgroundForegroundContrast(groundTruth, result gocv.Mat) {
	rows := groundTruth.Rows()
	cols := groundTruth.Cols()

	backgroundErrors := 0
	foregroundErrors := 0
	totalBackground := 0
	totalForeground := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			gtValue := groundTruth.GetUCharAt(y, x) > 127
			resValue := result.GetUCharAt(y, x) > 127

			if gtValue {
				totalForeground++
				if !resValue {
					foregroundErrors++
				}
			} else {
				totalBackground++
				if resValue {
					backgroundErrors++
				}
			}
		}
	}

	backgroundClutter := 0.0
	if totalBackground > 0 {
		backgroundClutter = float64(backgroundErrors) / float64(totalBackground)
	}

	foregroundSpeckle := 0.0
	if totalForeground > 0 {
		foregroundSpeckle = float64(foregroundErrors) / float64(totalForeground)
	}

	m.pbcValue = (backgroundClutter + foregroundSpeckle) / 2.0
}

func (m *BinaryImageMetrics) calculateSkeletonSimilarity(groundTruth, result gocv.Mat) {
	gtSkeleton := m.extractSkeleton(groundTruth)
	defer gtSkeleton.Close()
	resSkeleton := m.extractSkeleton(result)
	defer resSkeleton.Close()

	if gtSkeleton.Empty() || resSkeleton.Empty() {
		m.skeletonValue = 0.0
		return
	}

	// Calculate skeleton overlap
	intersection := gocv.NewMat()
	defer intersection.Close()
	gocv.BitwiseAnd(gtSkeleton, resSkeleton, &intersection)

	unionMat := gocv.NewMat()
	defer unionMat.Close()
	gocv.BitwiseOr(gtSkeleton, resSkeleton, &unionMat)

	intersectionPixels := gocv.CountNonZero(intersection)
	unionPixels := gocv.CountNonZero(unionMat)

	if unionPixels == 0 {
		m.skeletonValue = 0.0
		return
	}

	// Jaccard similarity index
	m.skeletonValue = float64(intersectionPixels) / float64(unionPixels)
}

func (m *BinaryImageMetrics) extractSkeleton(src gocv.Mat) gocv.Mat {
	if src.Empty() {
		return gocv.NewMat()
	}

	// Convert to binary if not already
	binary := gocv.NewMat()
	gocv.Threshold(src, &binary, 127, 255, gocv.ThresholdBinary)

	// Apply morphological skeletonization using iterative thinning
	skeleton := gocv.NewMat()
	temp := gocv.NewMat()
	defer temp.Close()

	// Initialize skeleton as zeros
	skeleton = gocv.NewMatWithSize(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)
	gocv.SetTo(&skeleton, gocv.NewScalar(0, 0, 0, 0), gocv.NewMat())

	element := gocv.GetStructuringElement(gocv.MorphCross, image.Point{X: 3, Y: 3})
	defer element.Close()

	for {
		gocv.MorphologyEx(binary, &temp, gocv.MorphOpen, element)
		gocv.BitwiseNot(temp, &temp)
		gocv.BitwiseAnd(binary, temp, &temp)
		gocv.BitwiseOr(skeleton, temp, &skeleton)
		gocv.MorphologyEx(binary, &binary, gocv.MorphErode, element)

		if gocv.CountNonZero(binary) == 0 {
			break
		}
	}

	binary.Close()
	return skeleton
}
