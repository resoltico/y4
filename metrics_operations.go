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
	gray, err := normalizeToGrayscale(mat, "binary mask creation")
	if err != nil {
		return gocv.NewMat(), err
	}
	defer gray.Close()

	binary := gocv.NewMat()
	gocv.Threshold(gray, &binary, float32(threshold), 255, gocv.ThresholdBinary)
	return binary, nil
}

func performMatrixOperation(mat1, mat2 gocv.Mat, operation string) (gocv.Mat, error) {
	if err := validateMatDimensionsMatch(mat1, mat2, operation); err != nil {
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

	return result, nil
}
