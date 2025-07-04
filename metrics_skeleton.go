package main

import (
	"image"

	"gocv.io/x/gocv"
)

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
	zeros := gocv.NewScalar(0, 0, 0, 0)
	skeleton.SetTo(zeros)

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
