package main

import (
	"image"
	"math"

	"gocv.io/x/gocv"
)

func (m *BinaryImageMetrics) calculateMPM(groundTruth, result gocv.Mat) error {
	gtContours, err := extractContoursSafely(groundTruth, "MPM ground truth")
	if err != nil {
		return err
	}

	resContours, err := extractContoursSafely(result, "MPM result")
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
			distance := calculateContourDistance(gtContour, resContour)
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
			distance := calculateContourDistance(resContour, gtContour)
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

func extractContoursSafely(mat gocv.Mat, context string) ([][]image.Point, error) {
	if err := validateMatForMetrics(mat, context); err != nil {
		return [][]image.Point{}, err
	}

	binary, err := ensureBinaryThresholded(mat, context)
	if err != nil {
		return [][]image.Point{}, err
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

func calculateContourDistance(contour1, contour2 []image.Point) float64 {
	if len(contour1) == 0 || len(contour2) == 0 {
		return math.Inf(1)
	}

	distance1 := calculateDirectedHausdorffDistance(contour1, contour2)
	distance2 := calculateDirectedHausdorffDistance(contour2, contour1)

	return math.Max(distance1, distance2)
}

func calculateDirectedHausdorffDistance(contour1, contour2 []image.Point) float64 {
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

func calculateChamferDistance(binary1, binary2 gocv.Mat) (float64, error) {
	if err := validateMatDimensionsMatch(binary1, binary2, "chamfer distance"); err != nil {
		return 0.0, err
	}

	dist1 := gocv.NewMat()
	defer dist1.Close()
	gocv.DistanceTransform(binary1, &dist1, gocv.DistL2, 3)

	dist2 := gocv.NewMat()
	defer dist2.Close()
	gocv.DistanceTransform(binary2, &dist2, gocv.DistL2, 3)

	rows := binary1.Rows()
	cols := binary1.Cols()
	totalDistance := 0.0
	pixelCount := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if binary1.GetUCharAt(y, x) > 127 {
				distance := float64(dist2.GetFloatAt(y, x))
				totalDistance += distance
				pixelCount++
			}
			if binary2.GetUCharAt(y, x) > 127 {
				distance := float64(dist1.GetFloatAt(y, x))
				totalDistance += distance
				pixelCount++
			}
		}
	}

	if pixelCount == 0 {
		return 0.0, nil
	}

	return totalDistance / float64(pixelCount), nil
}

func calculateBoundingBoxOverlap(contour1, contour2 []image.Point) float64 {
	if len(contour1) == 0 || len(contour2) == 0 {
		return 0.0
	}

	rect1 := calculateBoundingRect(contour1)
	rect2 := calculateBoundingRect(contour2)

	intersection := rect1.Intersect(rect2)
	if intersection.Empty() {
		return 0.0
	}

	area1 := rect1.Dx() * rect1.Dy()
	area2 := rect2.Dx() * rect2.Dy()
	intersectionArea := intersection.Dx() * intersection.Dy()

	union := area1 + area2 - intersectionArea
	if union == 0 {
		return 0.0
	}

	return float64(intersectionArea) / float64(union)
}

func calculateBoundingRect(contour []image.Point) image.Rectangle {
	if len(contour) == 0 {
		return image.Rectangle{}
	}

	minX, minY := contour[0].X, contour[0].Y
	maxX, maxY := contour[0].X, contour[0].Y

	for _, point := range contour {
		if point.X < minX {
			minX = point.X
		}
		if point.X > maxX {
			maxX = point.X
		}
		if point.Y < minY {
			minY = point.Y
		}
		if point.Y > maxY {
			maxY = point.Y
		}
	}

	return image.Rect(minX, minY, maxX+1, maxY+1)
}
