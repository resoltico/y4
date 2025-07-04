package main

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

type ValidationError struct {
	Context string
	Field   string
	Value   interface{}
	Reason  string
}

func (ve *ValidationError) Error() string {
	return fmt.Sprintf("%s: invalid %s value %v - %s", ve.Context, ve.Field, ve.Value, ve.Reason)
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

func validateImageMat(mat gocv.Mat, context string) error {
	if mat.Empty() {
		return &ValidationError{
			Context: context,
			Field:   "image",
			Value:   "empty",
			Reason:  "matrix contains no data",
		}
	}

	rows := mat.Rows()
	cols := mat.Cols()

	if rows <= 0 || cols <= 0 {
		return &ValidationError{
			Context: context,
			Field:   "dimensions",
			Value:   fmt.Sprintf("%dx%d", cols, rows),
			Reason:  "width and height must be positive",
		}
	}

	if rows < 3 || cols < 3 {
		return &ValidationError{
			Context: context,
			Field:   "dimensions",
			Value:   fmt.Sprintf("%dx%d", cols, rows),
			Reason:  "minimum size 3x3 required for processing",
		}
	}

	matType := mat.Type()
	if matType != gocv.MatTypeCV8UC1 && matType != gocv.MatTypeCV8UC3 && matType != gocv.MatTypeCV8UC4 {
		return &ValidationError{
			Context: context,
			Field:   "type",
			Value:   matType,
			Reason:  "only 8-bit unsigned images supported",
		}
	}

	return nil
}

func validateKernelSize(kernelSize int, imageWidth, imageHeight int, context string) error {
	if kernelSize < 1 {
		return &ValidationError{
			Context: context,
			Field:   "kernel_size",
			Value:   kernelSize,
			Reason:  "must be positive",
		}
	}

	if kernelSize%2 == 0 {
		return &ValidationError{
			Context: context,
			Field:   "kernel_size",
			Value:   kernelSize,
			Reason:  "must be odd number",
		}
	}

	if kernelSize >= imageWidth || kernelSize >= imageHeight {
		return &ValidationError{
			Context: context,
			Field:   "kernel_size",
			Value:   kernelSize,
			Reason:  fmt.Sprintf("must be smaller than image dimensions %dx%d", imageWidth, imageHeight),
		}
	}

	return nil
}

func safeMatCreation(rows, cols int, matType gocv.MatType, context string) (gocv.Mat, error) {
	if rows <= 0 || cols <= 0 {
		return gocv.Mat{}, &ValidationError{
			Context: context,
			Field:   "dimensions",
			Value:   fmt.Sprintf("%dx%d", cols, rows),
			Reason:  "dimensions must be positive",
		}
	}

	if rows > 32768 || cols > 32768 {
		return gocv.Mat{}, &ValidationError{
			Context: context,
			Field:   "dimensions",
			Value:   fmt.Sprintf("%dx%d", cols, rows),
			Reason:  "dimensions exceed maximum size 32768x32768",
		}
	}

	mat := gocv.NewMatWithSize(rows, cols, matType)
	if mat.Empty() {
		return gocv.Mat{}, &ValidationError{
			Context: context,
			Field:   "allocation",
			Value:   fmt.Sprintf("%dx%d type %d", cols, rows, matType),
			Reason:  "failed to allocate matrix memory",
		}
	}

	return mat, nil
}

func safeMatOperation(operation func() error, mats []gocv.Mat, context string) error {
	defer func() {
		if r := recover(); r != nil {
			for _, mat := range mats {
				if !mat.Empty() {
					mat.Close()
				}
			}
		}
	}()

	for i, mat := range mats {
		if err := validateImageMat(mat, fmt.Sprintf("%s mat[%d]", context, i)); err != nil {
			return err
		}
	}

	return operation()
}

func validateContourData(contours [][]image.Point, context string) error {
	if len(contours) == 0 {
		return nil // Empty contours is valid
	}

	for i, contour := range contours {
		if len(contour) < 3 {
			return &ValidationError{
				Context: context,
				Field:   "contour_points",
				Value:   fmt.Sprintf("contour %d has %d points", i, len(contour)),
				Reason:  "contours must have at least 3 points",
			}
		}

		for j, point := range contour {
			if point.X < 0 || point.Y < 0 {
				return &ValidationError{
					Context: context,
					Field:   "contour_coordinates",
					Value:   fmt.Sprintf("contour %d point %d: (%d,%d)", i, j, point.X, point.Y),
					Reason:  "coordinates must be non-negative",
				}
			}
		}
	}

	return nil
}
