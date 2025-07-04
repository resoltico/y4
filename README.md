# Otsu Obliterator

Advanced 2D Otsu thresholding application with multiple image quality metrics.

## Quality Assurance

### Quick Check
```bash
go run cmd/quality_check/main.go check
```

### Fast Check (without external tools)
```bash
go run cmd/quality_check/main.go fast
```

### Format Check Only
```bash
go run cmd/quality_check/main.go format
```

## Building

### Development Build
```bash
./build.sh build debug
```

### Production Build
```bash
./build.sh build
```

### Cross-platform Build
```bash
./build.sh build all
```

## Running

### Direct Execution
```bash
./build.sh run
```

### Manual Execution
```bash
./build/otsu-obliterator
```

## Features

- 2D Otsu thresholding with multiple parameters
- Six image quality metrics (F-measure, pseudo F-measure, DRD, MPM, NRM, PBC)
- Real-time parameter adjustment
- Multiple preprocessing options
- Cross-platform support

## Dependencies

- Go 1.24+
- OpenCV 4.x
- Fyne v2.6.1

## Installation

### macOS
```bash
brew install opencv
```

### Ubuntu/Debian
```bash
sudo apt-get install libopencv-dev
```

### Windows
Download OpenCV from opencv.org and configure environment variables.

## Architecture

This application follows a flat structure pattern optimized for applications under 2000 lines of code:

- `main.go` - Application entry point
- `app.go` - Application setup and lifecycle
- `ui_*.go` - User interface components
- `processing.go` - Image processing algorithms
- `metrics.go` - Quality measurement calculations
- `io_image.go` - File input/output operations

## Quality Tools

The application includes native Go quality checking:

- Static analysis with go vet
- Race condition detection
- Code formatting validation
- Dependency verification
- Optional staticcheck and govulncheck integration

## Metrics

### F-measure
Standard pixel accuracy using harmonic mean of precision and recall.

### Pseudo F-measure
Weighted accuracy with reduced bias toward foreground objects.

### DRD (Distance Reciprocal Distortion)
Visual distortion measurement using 5×5 weighting matrices.

### MPM (Morphological Path Misalignment)
Object-by-object evaluation with spatial relationship penalties.

### NRM (Normalized Root Mean Square)
Direct pixel mismatch calculation between result and ground truth.

### PBC (Peak Background Contrast)
Background clutter and foreground speckle analysis.