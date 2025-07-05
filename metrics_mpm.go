package main

import (
	"image"
	"math"

	"gocv.io/x/gocv"
)

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
