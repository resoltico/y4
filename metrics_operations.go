package main

import (
	"fmt"

	"gocv.io/x/gocv"
)

func calculateSafeCountNonZero(mat gocv.Mat, context string) (int, error) {
	if err := validateMatForMetrics(mat, context); err != nil {
		return 0, err
	}

	var gray gocv.Mat
	if mat.Channels() == 1 {
		gray = mat
	} else {
		gray = gocv.NewMat()
		defer gray.Close()
		gocv.CvtColor(mat, &gray, gocv.ColorBGRToGray)
	}

	return gocv.CountNonZero(gray), nil
}

func createBinaryMask(mat gocv.Mat, threshold uint8) (gocv.Mat, error) {
	if err := validateMatForMetrics(mat, "binary mask creation"); err != nil {
		return gocv.NewMat(), err
	}

	gray, err := normalizeToGrayscale(mat, "binary mask creation")
	if err != nil {
		return gocv.NewMat(), err
	}
	defer gray.Close()

	if err := validateBinaryMat(gray, "binary mask input"); err != nil {
		return gocv.NewMat(), fmt.Errorf("binary mask validation: %w", err)
	}

	binary := gocv.NewMat()
	gocv.Threshold(gray, &binary, float32(threshold), 255, gocv.ThresholdBinary)

	// Validate output
	if err := validateBinaryMat(binary, "binary mask output"); err != nil {
		binary.Close()
		return gocv.NewMat(), fmt.Errorf("binary mask output validation: %w", err)
	}

	return binary, nil
}

func performMatrixOperation(mat1, mat2 gocv.Mat, operation string) (gocv.Mat, error) {
	if err := validateMatDimensionsMatch(mat1, mat2, operation); err != nil {
		return gocv.NewMat(), err
	}

	if err := validateMatForMetrics(mat1, operation+" input1"); err != nil {
		return gocv.NewMat(), err
	}

	if err := validateMatForMetrics(mat2, operation+" input2"); err != nil {
		return gocv.NewMat(), err
	}

	result := gocv.NewMat()

	switch operation {
	case "and":
		gocv.BitwiseAnd(mat1, mat2, &result)
	case "or":
		gocv.BitwiseOr(mat1, mat2, &result)
	case "xor":
		gocv.BitwiseXor(mat1, mat2, &result)
	case "not":
		gocv.BitwiseNot(mat1, &result)
	default:
		result.Close()
		return gocv.NewMat(), fmt.Errorf("unsupported matrix operation: %s", operation)
	}

	// Validate result
	if err := validateMatForMetrics(result, operation+" result"); err != nil {
		result.Close()
		return gocv.NewMat(), fmt.Errorf("matrix operation result validation: %w", err)
	}

	return result, nil
}
