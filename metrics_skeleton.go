package main

import (
	"image"

	"gocv.io/x/gocv"
)

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

	binary, err := ensureBinaryThresholded(src, "skeleton extraction")
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
