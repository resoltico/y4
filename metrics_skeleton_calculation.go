package main

import (
	"image"

	"gocv.io/x/gocv"
)

func (m *BinaryImageMetrics) calculateSkeletonSimilarity(groundTruth, result gocv.Mat) error {
	gtSkeleton, err := extractSkeletonSafely(groundTruth, "skeleton ground truth")
	if err != nil {
		return err
	}
	defer gtSkeleton.Close()

	resSkeleton, err := extractSkeletonSafely(result, "skeleton result")
	if err != nil {
		return err
	}
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

func extractSkeletonSafely(src gocv.Mat, context string) (gocv.Mat, error) {
	if err := validateMatForMetrics(src, context); err != nil {
		return gocv.NewMat(), err
	}

	binary, err := ensureBinaryThresholded(src, context)
	if err != nil {
		return gocv.NewMat(), err
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

	return skeleton, nil
}

func calculateMedialAxisTransform(binary gocv.Mat) (gocv.Mat, error) {
	if err := validateMatForMetrics(binary, "medial axis transform"); err != nil {
		return gocv.NewMat(), err
	}

	distanceMap := gocv.NewMat()
	defer distanceMap.Close()

	gocv.DistanceTransform(binary, &distanceMap, gocv.DistL2, 3)

	localMaxima := gocv.NewMat()
	defer localMaxima.Close()

	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Point{X: 3, Y: 3})
	defer kernel.Close()

	dilated := gocv.NewMat()
	defer dilated.Close()
	gocv.MorphologyEx(distanceMap, &dilated, gocv.MorphDilate, kernel)

	gocv.Compare(distanceMap, dilated, &localMaxima, gocv.CompareEQ)

	skeleton := gocv.NewMat()
	gocv.BitwiseAnd(binary, localMaxima, &skeleton)

	return skeleton, nil
}

func applyZhangSuenThinning(binary gocv.Mat) (gocv.Mat, error) {
	if err := validateMatForMetrics(binary, "Zhang-Suen thinning"); err != nil {
		return gocv.NewMat(), err
	}

	rows := binary.Rows()
	cols := binary.Cols()

	result := binary.Clone()
	temp := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)
	defer temp.Close()

	maxIterations := 100
	changed := true

	for iteration := 0; iteration < maxIterations && changed; iteration++ {
		changed = false
		result.CopyTo(&temp)

		for y := 1; y < rows-1; y++ {
			for x := 1; x < cols-1; x++ {
				if temp.GetUCharAt(y, x) == 0 {
					continue
				}

				neighbors := getNeighborValues(temp, x, y)
				transitions := countTransitions(neighbors)
				nonZeroNeighbors := countNonZeroNeighbors(neighbors)

				if nonZeroNeighbors >= 2 && nonZeroNeighbors <= 6 &&
					transitions == 1 &&
					shouldRemovePixel(neighbors, iteration%2) {
					result.SetUCharAt(y, x, 0)
					changed = true
				}
			}
		}
	}

	return result, nil
}

func getNeighborValues(mat gocv.Mat, x, y int) [8]uint8 {
	var neighbors [8]uint8
	positions := [][2]int{
		{0, -1}, {1, -1}, {1, 0}, {1, 1},
		{0, 1}, {-1, 1}, {-1, 0}, {-1, -1},
	}

	for i, pos := range positions {
		nx, ny := x+pos[0], y+pos[1]
		if nx >= 0 && nx < mat.Cols() && ny >= 0 && ny < mat.Rows() {
			neighbors[i] = mat.GetUCharAt(ny, nx)
		}
	}

	return neighbors
}

func countTransitions(neighbors [8]uint8) int {
	transitions := 0
	for i := 0; i < 8; i++ {
		current := neighbors[i] > 0
		next := neighbors[(i+1)%8] > 0
		if !current && next {
			transitions++
		}
	}
	return transitions
}

func countNonZeroNeighbors(neighbors [8]uint8) int {
	count := 0
	for _, val := range neighbors {
		if val > 0 {
			count++
		}
	}
	return count
}

func shouldRemovePixel(neighbors [8]uint8, iteration int) bool {
	if iteration == 0 {
		return (neighbors[0] == 0 || neighbors[2] == 0 || neighbors[4] == 0) &&
			(neighbors[2] == 0 || neighbors[4] == 0 || neighbors[6] == 0)
	} else {
		return (neighbors[0] == 0 || neighbors[2] == 0 || neighbors[6] == 0) &&
			(neighbors[0] == 0 || neighbors[4] == 0 || neighbors[6] == 0)
	}
}
