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
	drdValue       float64
	mpmValue       float64
	pbcValue       float64
	skeletonValue  float64
}

func (m *BinaryImageMetrics) FMeasure() float64 {
	precision := m.Precision()
	recall := m.Recall()
	if precision+recall == 0 {
		return 0.0
	}
	return 2.0 * (precision * recall) / (precision + recall)
}

func (m *BinaryImageMetrics) PseudoFMeasure() float64 {
	precision := m.Precision()
	recall := m.Recall()
	beta := 0.5
	betaSq := beta * beta
	if betaSq*precision+recall == 0 {
		return 0.0
	}
	return (1.0 + betaSq) * (precision * recall) / (betaSq*precision + recall)
}

func (m *BinaryImageMetrics) NRM() float64 {
	if m.TotalPixels == 0 {
		return 0.0
	}
	return float64(m.FalsePositives+m.FalseNegatives) / float64(2*(m.TruePositives+m.TrueNegatives))
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

func (m *BinaryImageMetrics) Precision() float64 {
	if m.TruePositives+m.FalsePositives == 0 {
		return 0.0
	}
	return float64(m.TruePositives) / float64(m.TruePositives+m.FalsePositives)
}

func (m *BinaryImageMetrics) Recall() float64 {
	if m.TruePositives+m.FalseNegatives == 0 {
		return 0.0
	}
	return float64(m.TruePositives) / float64(m.TruePositives+m.FalseNegatives)
}

func (m *BinaryImageMetrics) calculateConfusionMatrix(groundTruth, result gocv.Mat) error {
	if err := validateMatDimensionsMatch(groundTruth, result, "confusion matrix"); err != nil {
		return err
	}

	gtBinary, err := ensureBinaryThresholded(groundTruth, "confusion matrix ground truth")
	if err != nil {
		return err
	}
	defer gtBinary.Close()

	resBinary, err := ensureBinaryThresholded(result, "confusion matrix result")
	if err != nil {
		return err
	}
	defer resBinary.Close()

	// Use calculatePixelStatistics for validation and debugging
	totalPixels, gtForeground, gtBackground, _, err := calculatePixelStatistics(gtBinary)
	if err != nil {
		return err
	}

	resTotalPixels, resForeground, resBackground, _, err := calculatePixelStatistics(resBinary)
	if err != nil {
		return err
	}

	if totalPixels != resTotalPixels {
		return fmt.Errorf("pixel count mismatch: %d vs %d", totalPixels, resTotalPixels)
	}

	// Validate reasonable foreground/background ratios
	gtForegroundRatio := float64(gtForeground) / float64(totalPixels)
	resForegroundRatio := float64(resForeground) / float64(totalPixels)

	if gtForegroundRatio < 0.01 || gtForegroundRatio > 0.99 {
		debugSystem := GetDebugSystem()
		debugSystem.logger.Warn("ground truth has extreme foreground ratio",
			"ratio", gtForegroundRatio,
			"foreground", gtForeground,
			"background", gtBackground,
			"total", totalPixels)
	}

	if resForegroundRatio < 0.01 || resForegroundRatio > 0.99 {
		debugSystem := GetDebugSystem()
		debugSystem.logger.Warn("result has extreme foreground ratio",
			"ratio", resForegroundRatio,
			"foreground", resForeground,
			"background", resBackground,
			"total", resTotalPixels)
	}

	// Log pixel statistics for debugging
	debugSystem := GetDebugSystem()
	debugSystem.logger.Debug("confusion matrix pixel statistics",
		"gt_foreground", gtForeground,
		"gt_background", gtBackground,
		"gt_ratio", gtForegroundRatio,
		"res_foreground", resForeground,
		"res_background", resBackground,
		"res_ratio", resForegroundRatio,
		"total_pixels", totalPixels)

	rows := gtBinary.Rows()
	cols := gtBinary.Cols()
	m.TotalPixels = totalPixels
	m.TruePositives = 0
	m.TrueNegatives = 0
	m.FalsePositives = 0
	m.FalseNegatives = 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			gtValue := gtBinary.GetUCharAt(y, x) > 127
			resValue := resBinary.GetUCharAt(y, x) > 127

			if gtValue && resValue {
				m.TruePositives++
			} else if !gtValue && !resValue {
				m.TrueNegatives++
			} else if !gtValue && resValue {
				m.FalsePositives++
			} else {
				m.FalseNegatives++
			}
		}
	}

	return nil
}

func (m *BinaryImageMetrics) calculateDRD(groundTruth, result gocv.Mat) error {
	gtBinary, err := createBinaryMask(groundTruth, 127)
	if err != nil {
		return fmt.Errorf("DRD ground truth binary mask: %w", err)
	}
	defer gtBinary.Close()

	resBinary, err := createBinaryMask(result, 127)
	if err != nil {
		return fmt.Errorf("DRD result binary mask: %w", err)
	}
	defer resBinary.Close()

	rows := gtBinary.Rows()
	cols := gtBinary.Cols()

	weightMatrix := m.createDRDWeightMatrix()

	totalDistortion := 0.0
	totalErrorPixels := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			gtValue := gtBinary.GetUCharAt(y, x) > 127
			resValue := resBinary.GetUCharAt(y, x) > 127

			if gtValue != resValue {
				totalErrorPixels++
				distortion := m.calculatePixelDRD(gtBinary, x, y, weightMatrix)
				totalDistortion += distortion
			}
		}
	}

	if totalErrorPixels == 0 {
		m.drdValue = 0.0
		return nil
	}

	totalForegroundPixels := 0
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if gtBinary.GetUCharAt(y, x) > 127 {
				totalForegroundPixels++
			}
		}
	}

	if totalForegroundPixels == 0 {
		m.drdValue = 0.0
		return nil
	}

	m.drdValue = totalDistortion / float64(totalForegroundPixels)
	return nil
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

func (m *BinaryImageMetrics) calculateMPM(groundTruth, result gocv.Mat) error {
	gtContours, err := m.extractContoursWithValidation(groundTruth, "MPM ground truth")
	if err != nil {
		return err
	}

	resContours, err := m.extractContoursWithValidation(result, "MPM result")
	if err != nil {
		return err
	}

	if len(gtContours) == 0 && len(resContours) == 0 {
		m.mpmValue = 0.0
		return nil
	}

	totalMismatch := 0.0
	totalObjects := 0

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
			fallbackDistance := float64(groundTruth.Rows() + groundTruth.Cols())
			totalMismatch += fallbackDistance
		}
	}

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

		if minDistance > 5.0 {
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
		return nil
	}

	m.mpmValue = totalMismatch / float64(totalObjects)
	return nil
}

func (m *BinaryImageMetrics) extractContoursWithValidation(mat gocv.Mat, context string) ([][]image.Point, error) {
	if err := validateMatForMetrics(mat, context); err != nil {
		return [][]image.Point{}, err
	}

	binary, err := createBinaryMask(mat, 127)
	if err != nil {
		return [][]image.Point{}, fmt.Errorf("contour extraction binary mask: %w", err)
	}
	defer binary.Close()

	defer func() {
		if r := recover(); r != nil {
			return
		}
	}()

	contours := gocv.FindContours(binary, gocv.RetrievalExternal, gocv.ChainApproxSimple)

	if contours.IsNil() {
		return [][]image.Point{}, nil
	}
	defer contours.Close()

	size := contours.Size()
	if size == 0 {
		return [][]image.Point{}, nil
	}

	result := make([][]image.Point, 0, size)
	for i := 0; i < size; i++ {
		contour := contours.At(i)
		if !contour.IsNil() {
			points := contour.ToPoints()
			if len(points) > 10 {
				result = append(result, points)
			}
		}
	}

	return result, nil
}

func (m *BinaryImageMetrics) calculateContourDistance(contour1, contour2 []image.Point) float64 {
	if len(contour1) == 0 || len(contour2) == 0 {
		return math.Inf(1)
	}

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

func (m *BinaryImageMetrics) calculateBackgroundForegroundContrast(groundTruth, result gocv.Mat) error {
	gtBinary, err := createBinaryMask(groundTruth, 127)
	if err != nil {
		return fmt.Errorf("BFC ground truth binary mask: %w", err)
	}
	defer gtBinary.Close()

	resBinary, err := createBinaryMask(result, 127)
	if err != nil {
		return fmt.Errorf("BFC result binary mask: %w", err)
	}
	defer resBinary.Close()

	rows := gtBinary.Rows()
	cols := gtBinary.Cols()

	backgroundErrors := 0
	foregroundErrors := 0
	totalBackground := 0
	totalForeground := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			gtValue := gtBinary.GetUCharAt(y, x) > 127
			resValue := resBinary.GetUCharAt(y, x) > 127

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
	return nil
}

func (m *BinaryImageMetrics) calculateSkeletonSimilarity(groundTruth, result gocv.Mat) error {
	gtSkeleton := m.extractSkeleton(groundTruth)
	defer gtSkeleton.Close()
	resSkeleton := m.extractSkeleton(result)
	defer resSkeleton.Close()

	if gtSkeleton.Empty() || resSkeleton.Empty() {
		m.skeletonValue = 0.0
		return nil
	}

	intersection, err := performMatrixOperation(gtSkeleton, resSkeleton, "and")
	if err != nil {
		return err
	}
	defer intersection.Close()

	unionMat, err := performMatrixOperation(gtSkeleton, resSkeleton, "or")
	if err != nil {
		return err
	}
	defer unionMat.Close()

	intersectionPixels, err := calculateSafeCountNonZero(intersection, "skeleton intersection")
	if err != nil {
		return err
	}

	unionPixels, err := calculateSafeCountNonZero(unionMat, "skeleton union")
	if err != nil {
		return err
	}

	if unionPixels == 0 {
		m.skeletonValue = 0.0
		return nil
	}

	m.skeletonValue = float64(intersectionPixels) / float64(unionPixels)
	return nil
}

func (m *BinaryImageMetrics) extractSkeleton(src gocv.Mat) gocv.Mat {
	if src.Empty() {
		return gocv.NewMat()
	}

	binary, err := createBinaryMask(src, 127)
	if err != nil {
		return gocv.NewMat()
	}
	defer binary.Close()

	skeleton := gocv.NewMatWithSize(binary.Rows(), binary.Cols(), gocv.MatTypeCV8UC1)
	zeros := gocv.NewScalar(0, 0, 0, 0)
	skeleton.SetTo(zeros)

	temp := gocv.NewMat()
	defer temp.Close()

	element := gocv.GetStructuringElement(gocv.MorphCross, image.Point{X: 3, Y: 3})
	defer element.Close()

	workingCopy := binary.Clone()
	defer workingCopy.Close()

	maxIterations := 100
	iteration := 0

	for iteration < maxIterations {
		gocv.MorphologyEx(workingCopy, &temp, gocv.MorphOpen, element)
		gocv.BitwiseNot(temp, &temp)
		gocv.BitwiseAnd(workingCopy, temp, &temp)
		gocv.BitwiseOr(skeleton, temp, &skeleton)
		gocv.MorphologyEx(workingCopy, &workingCopy, gocv.MorphErode, element)

		nonZeroCount, err := calculateSafeCountNonZero(workingCopy, "skeleton iteration")
		if err != nil || nonZeroCount == 0 {
			break
		}

		iteration++
	}

	return skeleton
}
