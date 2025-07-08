# Otsu Obliterator

Advanced 2D Otsu thresholding application with multiple image quality metrics and processing methods.

## Quick Start

```bash
# Build and run
./build.sh run

# Or manual execution
go run .
```

## Project Structure

```
otsu-obliterator/
├── build/              # All compiled binaries
├── logs/               # Debug and application logs  
├── cmd/quality_check/  # Quality assurance tool
├── *.go               # Application source files
├── build.sh           # Build automation
└── go.mod             # Dependencies
```

## Debug Mode

### Enable Debug Features
```bash
# Build with debug instrumentation (outputs to build/)
go build -tags debug -o build/otsu-obliterator-debug .

# Run with debug logging (logs to logs/)
mkdir -p logs
export LOG_LEVEL=debug
./build/otsu-obliterator-debug 2>&1 | tee logs/debug.log

# Filter debug output
./build/otsu-obliterator-debug 2>&1 | grep -E "(DEBUG|ERROR|WARN)" | tee logs/filtered.log
```

### Debug Output Management
```bash
# View all debug output with log rotation
go run -tags debug . 2>&1 | tee logs/debug-$(date +%Y%m%d-%H%M%S).log

# Filter by category and save to logs/
go run -tags debug . 2>&1 | grep "operation_id" | tee logs/operations.log
go run -tags debug . 2>&1 | grep "memory" | tee logs/memory.log
go run -tags debug . 2>&1 | grep "parameter" | tee logs/parameters.log
go run -tags debug . 2>&1 | grep "system snapshot" | tee logs/monitoring.log
go run -tags debug . 2>&1 | grep "duration_ms" | tee logs/performance.log
```

### Debug Features
- **Resource Monitoring**: Memory, goroutines, GC analysis (5s intervals)
- **Operation Tracing**: Processing pipeline with IDs and timing
- **Parameter History**: UI parameter change tracking
- **Performance Metrics**: Memory allocation and timing data
- **Log Management**: All logs written to `logs/` directory

## Quality Assurance

```bash
go run cmd/quality_check/main.go check    # Full check with external tools
go run cmd/quality_check/main.go fast     # Core checks only  
go run cmd/quality_check/main.go format   # Format validation only
```

### Linting Tools
- **go vet**: Built-in static analysis
- **staticcheck**: Advanced bug detection and style
- **govulncheck**: Security vulnerability scanner
- **ineffassign**: Ineffectual assignment detection
- **gofmt**: Code formatting validation

## Building

```bash
./build.sh build         # Production build (to build/)
./build.sh build debug   # Development build with race detection
./build.sh build all     # Cross-platform builds (all to build/)
./build.sh clean         # Remove build/ directory
```

### Build Output Organization
- All binaries: `build/`
- Debug binaries: `build/*-debug`
- Cross-platform: `build/*-platform-arch`
- No binaries in project root

## Features

### Processing Methods
- **Single Scale**: Standard 2D Otsu thresholding
- **Multi-Scale Pyramid**: Multiple resolution levels
- **Region Adaptive**: Grid-based local thresholding

### Algorithm Parameters
- **Window Size**: Neighborhood size (3-21, adaptive available)
- **Histogram Bins**: 2D histogram bins (auto or 32-256)
- **Smoothing Strength**: Gaussian histogram smoothing (0-5)
- **Neighborhood Types**: Rectangular, circular, distance-weighted

### Preprocessing Options
- **Gaussian Preprocessing**: Blur reduction
- **Adaptive Contrast Enhancement**: CLAHE improvement
- **Homomorphic Filtering**: Illumination correction
- **Anisotropic Diffusion**: Edge-preserving smoothing

### Quality Metrics (DIBCO Standard)
- **F-measure**: Precision/recall harmonic mean
- **Pseudo F-measure**: DIBCO weighted (β=0.5)
- **NRM**: Negative Rate Metric
- **DRD**: Distance Reciprocal Distortion
- **MPM**: Morphological Path Misalignment
- **BFC**: Background/Foreground Contrast
- **Skeleton**: Structural similarity

### User Interface
- **Image Viewer**: Side-by-side comparison with zoom controls
- **Parameter Panel**: Real-time algorithm controls
- **Metrics Display**: Live quality assessment
- **File Operations**: Load/save with format options

## Dependencies

- Go 1.24+
- OpenCV 4.x
- Fyne v2.6.1

### Installation

**macOS**: `brew install opencv`  
**Ubuntu/Debian**: `sudo apt-get install libopencv-dev`  
**Windows**: Download from opencv.org, configure environment

## Architecture

Flat structure (34 files, all <300 LOC):

**Core**: `main.go`, `app.go`, `app_about.go`  
**Processing**: `processing_engine.go`, `processing_methods.go`, `processing_validation.go`, `processing_neighborhood.go`, `processing_otsu.go`, `processing_preprocess.go`, `processing_timeout.go`  
**UI**: `ui_toolbar.go`, `ui_toolbar_metrics.go`, `ui_imageviewer.go`, `ui_parameter_panel.go`, `ui_filesavemenu.go`  
**Metrics/IO**: `metrics.go`, `io_image.go`  
**Debug**: `debug_stubs.go` (release), `debug_system.go`, `debug_monitor.go`, `debug_tracer.go` (debug builds)  

## Algorithm Implementation

### 2D Otsu Thresholding
Liu and Li (1993) extension with integer threshold optimization, 2D histograms using pixel intensity vs neighborhood mean, between-class variance maximization.

### Quality Metrics
DIBCO standards: F-measure, Pseudo F-measure (β=0.5), DRD with 5×5 weighting matrices, MPM using Hausdorff distance, NRM as (FN+FP)/(2*(TP+TN)).

### Performance
- Single scale: O(n²)
- Multi-scale: O(n² log n) 
- Region adaptive: O(n²/g²)

## File Organization

### Build Outputs
```bash
build/
├── otsu-obliterator           # Production binary
├── otsu-obliterator-debug     # Debug binary
├── otsu-obliterator.exe       # Windows build
├── otsu-obliterator-macos-*   # macOS builds
└── otsu-obliterator-linux-*   # Linux builds
```

### Debug Logs
```bash
logs/
├── debug-YYYYMMDD-HHMMSS.log  # Timestamped debug logs
├── operations.log             # Operation tracing
├── memory.log                 # Memory monitoring
├── parameters.log             # Parameter changes
├── performance.log            # Timing data
└── monitoring.log             # System snapshots
```

## Validation

Tested against DIBCO datasets, academic implementations, and standard benchmarks. Metrics match published results within acceptable tolerances.
