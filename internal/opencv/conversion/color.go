package conversion

import (
	"fmt"

	"otsu-obliterator/internal/opencv/safe"

	"gocv.io/x/gocv"
)

func CvtColorSafe(src *safe.Mat, dst *safe.Mat, code gocv.ColorConversionCode) error {
	if err := safe.ValidateColorConversion(src, code); err != nil {
		return fmt.Errorf("color conversion validation failed: %w", err)
	}

	if err := safe.ValidateMatForOperation(dst, "CvtColor destination"); err != nil {
		return fmt.Errorf("destination mat validation failed: %w", err)
	}

	srcMat := src.GetMat()
	dstMat := dst.GetMat()

	gocv.CvtColor(srcMat, &dstMat, code)

	return nil
}

func ConvertToGrayscale(src *safe.Mat) (*safe.Mat, error) {
	if err := safe.ValidateMatForOperation(src, "ConvertToGrayscale"); err != nil {
		return nil, err
	}

	channels := src.Channels()

	if channels == 1 {
		return src.Clone()
	}

	dst, err := safe.NewMat(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination Mat: %w", err)
	}

	var conversionCode gocv.ColorConversionCode
	switch channels {
	case 3:
		conversionCode = gocv.ColorBGRToGray
	case 4:
		tempBGR, err := safe.NewMat(src.Rows(), src.Cols(), gocv.MatTypeCV8UC3)
		if err != nil {
			dst.Close()
			return nil, fmt.Errorf("failed to create temporary BGR Mat: %w", err)
		}
		defer tempBGR.Close()

		if err := CvtColorSafe(src, tempBGR, gocv.ColorBGRAToBGR); err != nil {
			dst.Close()
			return nil, fmt.Errorf("BGRA to BGR conversion failed: %w", err)
		}

		if err := CvtColorSafe(tempBGR, dst, gocv.ColorBGRToGray); err != nil {
			dst.Close()
			return nil, fmt.Errorf("BGR to Gray conversion failed: %w", err)
		}

		return dst, nil
	default:
		dst.Close()
		return nil, fmt.Errorf("unsupported channel count for grayscale conversion: %d", channels)
	}

	if err := CvtColorSafe(src, dst, conversionCode); err != nil {
		dst.Close()
		return nil, fmt.Errorf("color conversion failed: %w", err)
	}

	return dst, nil
}

func ConvertToBGR(src *safe.Mat) (*safe.Mat, error) {
	if err := safe.ValidateMatForOperation(src, "ConvertToBGR"); err != nil {
		return nil, err
	}

	channels := src.Channels()

	if channels == 3 {
		return src.Clone()
	}

	dst, err := safe.NewMat(src.Rows(), src.Cols(), gocv.MatTypeCV8UC3)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination Mat: %w", err)
	}

	var conversionCode gocv.ColorConversionCode
	switch channels {
	case 1:
		conversionCode = gocv.ColorGrayToBGR
	case 4:
		conversionCode = gocv.ColorBGRAToBGR
	default:
		dst.Close()
		return nil, fmt.Errorf("unsupported channel count for BGR conversion: %d", channels)
	}

	if err := CvtColorSafe(src, dst, conversionCode); err != nil {
		dst.Close()
		return nil, fmt.Errorf("color conversion failed: %w", err)
	}

	return dst, nil
}
