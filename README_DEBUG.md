# Debug System Guide - Otsu Obliterator

## Overview

The debug system provides comprehensive monitoring, tracing, and performance analysis with build-time toggles for production/development environments.

## Build Configurations

### Debug Build (Full Instrumentation)
```bash
go build -tags debug -o otsu-obliterator .
go run -tags debug .
```

### Release Build (Minimal Overhead)
```bash
go build -o otsu-obliterator .
go run .
```

## Log Levels & Filtering

### Available Levels
- **DEBUG**: Detailed algorithm internals, parameter tracking, performance data
- **INFO**: Application lifecycle, operation completion, high-level status
- **WARN**: Non-fatal issues, performance concerns, algorithm warnings
- **ERROR**: Failures, validation errors, critical issues

### Filtering Commands

#### Core System Monitoring
```bash
./otsu-obliterator 2>&1 | grep -E "(ERROR|WARN)"                    # Errors only
./otsu-obliterator 2>&1 | grep -E "(INFO|ERROR|WARN)"               # Status + errors
./otsu-obliterator 2>&1 | grep -E "DEBUG.*system"                   # System health
```

#### Algorithm Analysis
```bash
./otsu-obliterator 2>&1 | grep -E "histogram|threshold|variance"    # Otsu algorithm
./otsu-obliterator 2>&1 | grep -E "neighborhood|pyramid|region"     # Processing methods
./otsu-obliterator 2>&1 | grep -E "preprocessing|diffusion|blur"    # Image preprocessing
```

#### Performance Monitoring
```bash
./otsu-obliterator 2>&1 | grep -E "memory|duration|goroutines"      # Resource usage
./otsu-obliterator 2>&1 | grep -E "operation_id.*completed"         # Timing data
./otsu-obliterator 2>&1 | grep -E "gc_cycles|cgo_calls"             # Runtime metrics
```

#### Parameter Tracking
```bash
./otsu-obliterator 2>&1 | grep -E "parameter.*changed"              # UI changes
./otsu-obliterator 2>&1 | grep -E "snapshot.*recorded"              # Operation parameters
./otsu-obliterator 2>&1 | grep -E "WindowSize|HistogramBins"        # Specific params
```

#### Validation & Quality
```bash
./otsu-obliterator 2>&1 | grep -E "validation|matrix.*empty"        # Input validation
./otsu-obliterator 2>&1 | grep -E "metrics.*completed"              # Quality metrics
./otsu-obliterator 2>&1 | grep -E "confusion.*matrix"               # Classification results
```

## Debug Keywords by Category

### Algorithm Internals
- `histogram input analysis` - Value ranges before processing
- `histogram distribution analysis` - Bin usage statistics
- `Otsu threshold analysis` - Variance calculations and quality
- `threshold application results` - Pixel classification counts
- `poor foreground/background separation` - Algorithm warnings

### Processing Pipeline
- `single scale processing` - Standard Otsu method
- `multi-scale processing` - Pyramid method status
- `region adaptive processing` - Grid-based method
- `pyramid level validation` - Multi-scale errors
- `neighborhood calculation` - Window processing

### Performance Monitoring
- `operation monitoring started` - Resource tracking begin
- `operation monitoring completed` - Resource tracking end
- `system snapshot` - 5-second interval metrics
- `memory usage` - Heap allocation tracking
- `duration_ms` - Operation timing

### Quality Assurance
- `metrics calculation completed` - DIBCO standard results
- `confusion matrix pixel statistics` - Classification analysis
- `binary mask validation` - Input/output verification
- `processing result validation` - Final quality checks

## Diagnostic Workflows

### Algorithm Failure Investigation
```bash
# Step 1: Check input validation
./otsu-obliterator 2>&1 | grep -E "validation.*failed|matrix.*empty"

# Step 2: Analyze histogram construction
./otsu-obliterator 2>&1 | grep -E "histogram.*analysis|non_zero_bins"

# Step 3: Examine threshold calculation
./otsu-obliterator 2>&1 | grep -E "threshold.*analysis|variance.*ratio"

# Step 4: Verify output generation
./otsu-obliterator 2>&1 | grep -E "threshold.*results|foreground_ratio"
```

### Performance Analysis
```bash
# Memory leak detection
./otsu-obliterator 2>&1 | grep -E "memory_leaked|heap_alloc"

# Processing bottlenecks
./otsu-obliterator 2>&1 | grep -E "duration_ms" | sort -k6 -n

# Resource threshold violations
./otsu-obliterator 2>&1 | grep -E "high.*usage.*detected|threshold.*exceeded"
```

### Multi-Scale Debugging
```bash
# Pyramid construction issues
./otsu-obliterator 2>&1 | grep -E "pyramid.*failed|upsampling.*failed"

# Level-specific problems
./otsu-obliterator 2>&1 | grep -E "level.*validation|pyramid.*level"

# Combination errors
./otsu-obliterator 2>&1 | grep -E "combination.*failed|dimension.*mismatch"
```

## Operation ID Tracking

Each processing operation gets a unique ID for end-to-end tracing:

```bash
# Track specific operation
./otsu-obliterator 2>&1 | grep "operation_id=5"

# Follow operation lifecycle
./otsu-obliterator 2>&1 | grep -E "operation_id.*started|operation_id.*completed"
```

## Debug Environment Variables

```bash
export LOG_LEVEL=debug     # Force debug level
export LOG_LEVEL=info      # Reduce verbosity
export LOG_LEVEL=warn      # Minimal logging
```

## Common Debug Patterns

### All-Black Output Investigation
```bash
./otsu-obliterator 2>&1 | grep -E "histogram.*empty|poor.*separation|all-background"
```

### Memory Growth Analysis
```bash
./otsu-obliterator 2>&1 | grep -E "heap_alloc_mb" | awk '{print $NF}' | sort -n
```

### Parameter Change Tracking
```bash
./otsu-obliterator 2>&1 | grep -E "parameter.*changed" | tail -10
```

### Processing Method Comparison
```bash
# Single scale
./otsu-obliterator 2>&1 | grep -E "method=single_scale.*completed"

# Multi-scale  
./otsu-obliterator 2>&1 | grep -E "method=multi_scale.*completed"

# Region adaptive
./otsu-obliterator 2>&1 | grep -E "method=region_adaptive.*completed"
```

## Integration with Quality Tools

```bash
# Combine with quality checks
go run cmd/quality_check/main.go check 2>&1 | grep -E "DEBUG|ERROR"

# Performance profiling with debug
go run -tags debug . -cpuprofile=cpu.prof 2>&1 | grep "duration_ms"
```

## Build System Integration

```bash
# Debug build with race detection
./build.sh build debug

# Release build (debug stubs only)
./build.sh build
```

## Advanced Filtering Examples

### Time-based Analysis
```bash
# Recent operations only
./otsu-obliterator 2>&1 | grep "$(date '+%H:%M')" | grep -E "operation.*completed"
```

### Error Context Reconstruction
```bash
# Find operation that failed
./otsu-obliterator 2>&1 | grep -B5 -A5 "ERROR.*processing.*failed"
```

### Statistical Summary
```bash
# Average processing times
./otsu-obliterator 2>&1 | grep "duration_ms" | awk -F'=' '{sum+=$NF; count++} END {print "Avg:", sum/count "ms"}'
```