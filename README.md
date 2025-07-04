# Otsu Obliterator

Advanced 2D Otsu thresholding application with multiple image quality metrics and processing methods.

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

### Debug Mode

Enable debug logging to troubleshoot issues:

```bash
go run . 2>&1 | grep -E "(DEBUG|ERROR)"
```

## Features

### Processing Methods
- **Single Scale**: Standard 2D Otsu thresholding with integer threshold optimization
- **Multi-Scale Pyramid**: Processes multiple resolution levels for complex documents
- **Region Adaptive**: Applies different thresholds to image regions

### Algorithm Parameters
- **Window Size**: Neighborhood size for local statistics (3-21, adaptive sizing available)
- **Histogram Bins**: Bins for 2D histogram construction (auto-calculated or 32-256)
- **Smoothing Strength**: Gaussian smoothing of histogram (0-5)
- **Neighborhood Types**: Rectangular, circular, or distance-weighted neighborhoods

### Preprocessing Options
- **Gaussian Preprocessing**: Blur reduction before processing
- **Adaptive Contrast Enhancement**: CLAHE-based contrast improvement
- **Homomorphic Filtering**: Illumination variation correction
- **Anisotropic Diffusion**: Edge-preserving noise reduction

### Quality Metrics (DIBCO Standard)
- **F-measure**: Standard precision/recall harmonic mean
- **Pseudo F-measure**: DIBCO standard weighted F-measure (β=0.5)
- **NRM**: Negative Rate Metric for error quantification
- **DRD**: Distance Reciprocal Distortion for visual quality assessment
- **MPM**: Morphological Path Misalignment for object-level accuracy
- **Background/Foreground Contrast**: Clutter and speckle analysis
- **Skeleton Similarity**: Structural accuracy measurement

### Post-Processing
- **Morphological Operations**: Opening and closing with configurable kernel sizes
- **Interpolation Methods**: Nearest, bilinear, and bicubic for scaling operations

### User Interface
- **Image Viewer**: Side-by-side comparison with zoom controls and view modes
- **Parameter Panel**: Organized controls for all algorithm parameters
- **Metrics Display**: Real-time quality assessment with detailed breakdown
- **File Operations**: Load/save with format selection and quality options

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

This application follows a flat structure pattern for applications under 3500 lines of code:

- `main.go` - Application entry point
- `app.go` - Application setup and lifecycle management
- `ui_*.go` - User interface components (toolbar, parameters, image viewer, file operations)
- `processing.go` - Image processing algorithms with validation
- `metrics.go` - Quality measurement calculations
- `io_image.go` - File input/output operations

## Algorithm Implementation

### 2D Otsu Thresholding
The implementation uses the Liu and Li (1993) extension of Otsu's algorithm:
- 2D histogram construction using pixel intensity vs neighborhood mean
- Integer threshold optimization (removes non-standard subpixel approach)
- Between-class variance maximization
- Multiple neighborhood calculation methods

### Quality Metrics
All metrics follow DIBCO (Document Image Binarization Contest) standards:
- **F-measure**: Standard binary classification metric
- **Pseudo F-measure**: Uses β=0.5 weighting as per DIBCO specification
- **DRD**: Implements Lu et al. (2004) methodology with 5×5 weighting matrices
- **MPM**: Object-level evaluation using Hausdorff distance between contours
- **NRM**: Standard negative rate calculation (FN+FP)/(2*(TP+TN))

### Processing Enhancements
- **Multi-scale pyramid**: Processes images at multiple resolutions
- **Region-adaptive thresholding**: Grid-based local threshold optimization
- **Adaptive window sizing**: Dynamic neighborhood size based on image statistics
- **Advanced preprocessing**: Homomorphic filtering and anisotropic diffusion

## Quality Tools

The application includes native Go quality checking:

- Static analysis with go vet
- Race condition detection
- Code formatting validation
- Dependency verification
- Optional staticcheck and govulncheck integration

## Performance Characteristics

### Computational Complexity
- Single scale: O(n²) for n×n images
- Multi-scale: O(n² log n) with pyramid levels
- Region adaptive: O(n²/g²) where g is grid size

## Validation and Testing

The implementation has been validated against:
- DIBCO contest ground truth datasets
- Academic reference implementations
- Standard computer vision benchmarks

Quality metrics match published academic results within acceptable tolerances.

## License & Authors

MIT, Ervins Strauhmanis
