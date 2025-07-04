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

	intersection := gocv.NewMat()
	defer intersection.Close()
	gocv.BitwiseAnd(gtSkeleton, resSkeleton, &intersection)

	unionMat := gocv.NewMat()
	defer unionMat.Close()
	gocv.BitwiseOr(gtSkeleton, resSkeleton, &unionMat)

	intersectionPixels, err := safeCountNonZero(intersection, "skeleton intersection")
	if err != nil {
		m.skeletonValue = 0.0
		return
	}

	unionPixels, err := safeCountNonZero(unionMat, "skeleton union")
	if err != nil {
		m.skeletonValue = 0.0
		return
	}

	if unionPixels == 0 {
		m.skeletonValue = 0.0
		return
	}

	m.skeletonValue = float64(intersectionPixels) / float64(unionPixels)
}

func (m *BinaryImageMetrics) extractSkeleton(src gocv.Mat) gocv.Mat {
	if src.Empty() {
		return gocv.NewMat()
	}

	var gray gocv.Mat
	if src.Channels() > 1 {
		gray = gocv.NewMat()
		defer gray.Close()
		gocv.CvtColor(src, &gray, gocv.ColorBGRToGray)
	} else {
		gray = src
	}

	binary := gocv.NewMat()
	gocv.Threshold(gray, &binary, 127, 255, gocv.ThresholdBinary)

	skeleton := gocv.NewMatWithSize(gray.Rows(), gray.Cols(), gocv.MatTypeCV8UC1)
	zeros := gocv.NewScalar(0, 0, 0, 0)
	skeleton.SetTo(zeros)

	temp := gocv.NewMat()
	defer temp.Close()

	element := gocv.GetStructuringElement(gocv.MorphCross, image.Point{X: 3, Y: 3})
	defer element.Close()

	for {
		gocv.MorphologyEx(binary, &temp, gocv.MorphOpen, element)
		gocv.BitwiseNot(temp, &temp)
		gocv.BitwiseAnd(binary, temp, &temp)
		gocv.BitwiseOr(skeleton, temp, &skeleton)
		gocv.MorphologyEx(binary, &binary, gocv.MorphErode, element)

		nonZeroCount, err := safeCountNonZero(binary, "skeleton iteration")
		if err != nil || nonZeroCount == 0 {
			break
		}
	}

	binary.Close()
	return skeleton
}
