# Otsu Obliterator

A high-performance image processing application implementing advanced Otsu thresholding algorithms with real-time preview and memory-safe OpenCV operations.

## Features

- **2D Otsu Thresholding** - Advanced implementation with neighborhood analysis and sub-pixel precision
- **Iterative Triclass** - Multi-threshold segmentation with convergence detection and floating-point accuracy
- **Quality Modes** - Fast (integer precision) vs Best (sub-pixel convergence) processing modes
- **Real-time Preview** - Live parameter adjustment with immediate visual feedback
- **Memory Safety** - Automatic OpenCV Mat lifecycle management with leak detection
- **Cross-platform** - Native builds for Windows, macOS, and Linux

## Quick Start

```bash
# Clone and build
git clone <repository-url>
cd otsu-obliterator

# Install dependencies
./build.sh deps

# Build and run immediately
./build.sh build && ./build/otsu-obliterator
```

## Build Options

### Production Builds
```bash
# Current platform
./build.sh build && ./build/otsu-obliterator

# Cross-platform
./build.sh build windows && ./build/otsu-obliterator.exe
./build.sh build macos-arm64 && ./build/otsu-obliterator-macos-arm64
./build.sh build linux && ./build/otsu-obliterator-linux-amd64
```

### Development Builds
```bash
# With memory profiling
./build.sh build profile && ./build/otsu-obliterator

# Debug with race detection
./build.sh debug memory
```

### Distribution Packages
```bash
# Current platform package
./build.sh package

# Platform-specific packages
./build.sh package windows    # Creates installer
./build.sh package macos      # Creates .app bundle
./build.sh package linux      # Creates distribution archive
```

## Critical Build Information

**Important**: This project uses `go build` instead of `fyne build` to preserve proper application entry points. The build script automatically handles this correctly.

**Do NOT manually use**:
```bash
fyne build -o binary ./cmd/otsu-obliterator  # Wrong - bypasses main()
```

**Always use the build script**:
```bash
./build.sh build [target]  # Correct - preserves main() execution
```

This ensures proper menu initialization and About dialog functionality. If you see missing menus or empty About dialogs, verify you're using the build script rather than manual `fyne build`.

## Requirements

- **Go 1.24+** - Required for modern language features
- **OpenCV 4.11.0+** - Computer vision operations
- **CGO enabled** - For OpenCV bindings
- **Fyne tool** - For packaging (auto-installed when needed)

### Platform-specific Installation

**macOS:**
```bash
brew install opencv
```

**Ubuntu/Debian:**
```bash
sudo apt-get install libopencv-dev
```

**Windows:**
Follow [GoCV installation guide](https://gocv.io/getting-started/)

## Usage

1. **Load Image** - Click Load button or drag image file
2. **Select Algorithm** - Choose between 2D Otsu or Iterative Triclass
3. **Choose Quality** - Fast (integer precision) or Best (sub-pixel precision)
4. **Adjust Parameters** - Use sliders for real-time tuning
5. **Process** - Click Process button for thresholding
6. **Save Result** - Export processed image in PNG/JPEG format

### Quality Modes

**Fast Mode:**
- Integer-based calculations for speed
- Standard threshold detection
- Suitable for real-time processing

**Best Mode:**
- Sub-pixel precision with 0.1 step interpolation
- Floating-point calculations throughout
- Higher accuracy edge detection
- 3-5x slower processing time

### Algorithm Parameters

**2D Otsu:**
- Window Size: Neighborhood analysis window (3-21, odd numbers)
- Histogram Bins: Threshold precision (16-256)
- Pixel Weight Factor: Balance between pixel and neighborhood values (0.0-1.0)
- Smoothing Sigma: Gaussian smoothing strength (0.0-5.0)

**Iterative Triclass:**
- Max Iterations: Convergence limit (1-20)
- Convergence Epsilon: Threshold stability requirement (0.1-10.0)
- Gap Factor: Separation between threshold classes (0.0-1.0)
- Min TBD Fraction: Minimum "to be determined" pixel ratio (0.001-0.2)

## Performance

**Memory Management:**
- Automatic OpenCV Mat cleanup prevents leaks
- Pool-based object reuse reduces allocation overhead
- Real-time memory monitoring with statistics

**Processing Speed:**
- Fast mode: Integer calculations for maximum speed
- Best mode: Sub-pixel precision for quality
- Context-based cancellation for responsiveness
- Multi-threaded operations where applicable

## Development

### Build Workflow
```bash
# Complete development cycle
make dev

# Individual steps
./build.sh format  # Code formatting
./build.sh lint    # Static analysis
./build.sh test    # Unit tests with coverage
./build.sh bench   # Performance benchmarks
```

### Memory Debugging
```bash
# Monitor Mat object lifecycle
./build.sh debug memory

# Check for memory leaks
LOG_LEVEL=debug ./build/otsu-obliterator
```

### Testing
```bash
# Run tests with coverage
./build.sh test

# View coverage report
open coverage.html
```

### Packaging
```bash
# Create distribution packages
make dist

# Platform-specific packaging
make package-windows
make package-macos
make package-linux
```

## Architecture

- **MVC Pattern** - Clean separation of GUI, business logic, and data
- **Pipeline Processing** - Modular image processing workflow
- **Memory Safety** - Wrapper around OpenCV Mat objects with automatic cleanup
- **Context Propagation** - Cancellation and timeout support throughout
- **Quality Modes** - Computational precision levels independent of parameter settings

## About Window

Access application information via Help → About menu, showing:
- Application name and version
- Author and license information
- Runtime environment details
- Build configuration

**Note**: If the About dialog appears empty or menus are missing, ensure you built using `./build.sh build` rather than manual commands.

## Troubleshooting

**Build Issues:**
```bash
# Verify dependencies
./build.sh deps

# Clean and rebuild
./build.sh clean && ./build.sh build
```

**Missing Menus/About Dialog:**
- **Cause**: Using `fyne build` instead of `go build`
- **Solution**: Always use `./build.sh build [target]`
- **Verification**: Look for "MAIN: Starting main function" in debug output

**Runtime Issues:**
- Ensure OpenCV is properly installed and accessible
- Check that image files are in supported formats (PNG, JPEG, TIFF, BMP)
- Monitor memory usage with debug builds if processing large images

**Performance Issues:**
- Use production builds (`./build.sh build`) not debug builds
- Choose Fast quality mode for speed, Best for accuracy
- Close unused images to free memory
- Reduce histogram bins for faster processing on large images

**Packaging Issues:**
- Ensure Fyne tool is installed: `go install fyne.io/fyne/v2/cmd/fyne@latest`
- Verify FyneApp.toml configuration is complete
- Check platform-specific dependencies are met

## License, Author

MIT, Ervins Strauhmanis

## Acknowledgments

- Built with [GoCV](https://gocv.io/) for OpenCV bindings
- GUI powered by [Fyne](https://fyne.io/)
- Logging via [zerolog](https://github.com/rs/zerolog)