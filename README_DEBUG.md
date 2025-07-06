# Debug System Guide - Otsu Obliterator

## Overview

The debug system provides comprehensive monitoring, tracing, and performance analysis with build-time toggles for production/development environments.

## Build Configurations

### Debug Build (Full Instrumentation)
```bash
# Always outputs to build/ directory
go build -tags debug -o build/otsu-obliterator-debug .
go run -tags debug .
```

### Release Build (Minimal Overhead)
```bash
# Always outputs to build/ directory  
go build -o build/otsu-obliterator .
go run .
```

## Log Management

### Log Directory Structure
```bash
mkdir -p logs  # Create logs directory
logs/
├── debug-YYYYMMDD-HHMMSS.log  # Timestamped debug sessions
├── operations.log             # Operation tracing only
├── memory.log                 # Memory monitoring only
├── parameters.log             # Parameter changes only
├── performance.log            # Timing data only
├── monitoring.log             # System snapshots only
└── filtered-YYYYMMDD.log      # Custom filtered logs
```

### Log Levels & Filtering

#### Core System Monitoring
```bash
# Redirect all logs to logs/ directory
./build/otsu-obliterator-debug 2>&1 | tee logs/debug-$(date +%Y%m%d-%H%M%S).log

# Filter errors and warnings to logs/
./build/otsu-obliterator-debug 2>&1 | grep -E "(ERROR|WARN)" | tee logs/errors.log

# System health monitoring to logs/
./build/otsu-obliterator-debug 2>&1 | grep -E "DEBUG.*system" | tee logs/system.log
```

#### Algorithm Analysis
```bash
# Algorithm debugging to logs/
./build/otsu-obliterator-debug 2>&1 | grep -E "histogram|threshold|variance" | tee logs/algorithm.log

# Processing methods to logs/
./build/otsu-obliterator-debug 2>&1 | grep -E "neighborhood|pyramid|region" | tee logs/processing.log

# Preprocessing steps to logs/
./build/otsu-obliterator-debug 2>&1 | grep -E "preprocessing|diffusion|blur" | tee logs/preprocessing.log
```

#### Performance Monitoring
```bash
# Resource usage to logs/
./build/otsu-obliterator-debug 2>&1 | grep -E "memory|duration|goroutines" | tee logs/performance.log

# Operation timing to logs/
./build/otsu-obliterator-debug 2>&1 | grep -E "operation_id.*completed" | tee logs/operations.log

# Runtime metrics to logs/
./build/otsu-obliterator-debug 2>&1 | grep -E "gc_cycles|cgo_calls" | tee logs/runtime.log
```

#### Parameter Tracking
```bash
# UI parameter changes to logs/
./build/otsu-obliterator-debug 2>&1 | grep -E "parameter.*changed" | tee logs/parameters.log

# Operation snapshots to logs/
./build/otsu-obliterator-debug 2>&1 | grep -E "snapshot.*recorded" | tee logs/snapshots.log

# Specific parameter monitoring to logs/
./build/otsu-obliterator-debug 2>&1 | grep -E "WindowSize|HistogramBins" | tee logs/ui-params.log
```

#### Validation & Quality
```bash
# Input validation to logs/
./build/otsu-obliterator-debug 2>&1 | grep -E "validation|matrix.*empty" | tee logs/validation.log

# Quality metrics to logs/
./build/otsu-obliterator-debug 2>&1 | grep -E "metrics.*completed" | tee logs/metrics.log

# Classification results to logs/
./build/otsu-obliterator-debug 2>&1 | grep -E "confusion.*matrix" | tee logs/classification.log
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
# Step 1: Check input validation (save to logs/)
./build/otsu-obliterator-debug 2>&1 | grep -E "validation.*failed|matrix.*empty" | tee logs/failures.log

# Step 2: Analyze histogram construction
./build/otsu-obliterator-debug 2>&1 | grep -E "histogram.*analysis|non_zero_bins" | tee logs/histogram-debug.log

# Step 3: Examine threshold calculation
./build/otsu-obliterator-debug 2>&1 | grep -E "threshold.*analysis|variance.*ratio" | tee logs/threshold-debug.log

# Step 4: Verify output generation
./build/otsu-obliterator-debug 2>&1 | grep -E "threshold.*results|foreground_ratio" | tee logs/output-debug.log
```

### Performance Analysis
```bash
# Memory leak detection (save to logs/)
./build/otsu-obliterator-debug 2>&1 | grep -E "memory_leaked|heap_alloc" | tee logs/memory-leaks.log

# Processing bottlenecks (save sorted timing to logs/)
./build/otsu-obliterator-debug 2>&1 | grep -E "duration_ms" | sort -k6 -n | tee logs/timing-sorted.log

# Resource threshold violations
./build/otsu-obliterator-debug 2>&1 | grep -E "high.*usage.*detected|threshold.*exceeded" | tee logs/thresholds.log
```

### Multi-Scale Debugging
```bash
# Pyramid construction issues (to logs/)
./build/otsu-obliterator-debug 2>&1 | grep -E "pyramid.*failed|upsampling.*failed" | tee logs/pyramid-errors.log

# Level-specific problems
./build/otsu-obliterator-debug 2>&1 | grep -E "level.*validation|pyramid.*level" | tee logs/pyramid-levels.log

# Combination errors
./build/otsu-obliterator-debug 2>&1 | grep -E "combination.*failed|dimension.*mismatch" | tee logs/pyramid-combination.log
```

## Operation ID Tracking

Each processing operation gets a unique ID for end-to-end tracing:

```bash
# Track specific operation (save to logs/)
./build/otsu-obliterator-debug 2>&1 | grep "operation_id=5" | tee logs/operation-5.log

# Follow operation lifecycle
./build/otsu-obliterator-debug 2>&1 | grep -E "operation_id.*started|operation_id.*completed" | tee logs/operation-lifecycle.log
```

## Debug Environment Variables

```bash
export LOG_LEVEL=debug     # Force debug level
export LOG_LEVEL=info      # Reduce verbosity
export LOG_LEVEL=warn      # Minimal logging

# Run with log level and save output
LOG_LEVEL=debug ./build/otsu-obliterator-debug 2>&1 | tee logs/debug-session.log
```

## Common Debug Patterns

### All-Black Output Investigation
```bash
./build/otsu-obliterator-debug 2>&1 | grep -E "histogram.*empty|poor.*separation|all-background" | tee logs/black-output.log
```

### Memory Growth Analysis
```bash
./build/otsu-obliterator-debug 2>&1 | grep -E "heap_alloc_mb" | awk '{print $NF}' | sort -n | tee logs/memory-growth.log
```

### Parameter Change Tracking
```bash
./build/otsu-obliterator-debug 2>&1 | grep -E "parameter.*changed" | tail -10 | tee logs/recent-params.log
```

### Processing Method Comparison
```bash
# Single scale (to logs/)
./build/otsu-obliterator-debug 2>&1 | grep -E "method=single_scale.*completed" | tee logs/single-scale.log

# Multi-scale (to logs/)
./build/otsu-obliterator-debug 2>&1 | grep -E "method=multi_scale.*completed" | tee logs/multi-scale.log

# Region adaptive (to logs/)
./build/otsu-obliterator-debug 2>&1 | grep -E "method=region_adaptive.*completed" | tee logs/region-adaptive.log
```

## Integration with Quality Tools

```bash
# Combine with quality checks (save to logs/)
go run cmd/quality_check/main.go check 2>&1 | grep -E "DEBUG|ERROR" | tee logs/quality-debug.log

# Performance profiling with debug (save to logs/)
go run -tags debug . -cpuprofile=logs/cpu.prof 2>&1 | grep "duration_ms" | tee logs/profiling.log
```

### Quality Check Linters
- **go vet**: Static analysis and potential bugs
- **staticcheck**: Advanced code quality and style
- **govulncheck**: Security vulnerability detection  
- **ineffassign**: Unused variable assignments
- **gofmt**: Code formatting compliance

## Build System Integration

```bash
# Debug build with race detection (outputs to build/)
./build.sh build debug

# Release build (outputs to build/)
./build.sh build

# All builds go to build/, logs go to logs/
./build.sh build all 2>&1 | tee logs/build.log
```

## Advanced Filtering Examples

### Time-based Analysis
```bash
# Recent operations only (save to logs/)
./build/otsu-obliterator-debug 2>&1 | grep "$(date '+%H:%M')" | grep -E "operation.*completed" | tee logs/recent-ops.log
```

### Error Context Reconstruction
```bash
# Find operation that failed (with context, save to logs/)
./build/otsu-obliterator-debug 2>&1 | grep -B5 -A5 "ERROR.*processing.*failed" | tee logs/error-context.log
```

### Statistical Summary
```bash
# Average processing times (save to logs/)
./build/otsu-obliterator-debug 2>&1 | grep "duration_ms" | awk -F'=' '{sum+=$NF; count++} END {print "Avg:", sum/count "ms"}' | tee logs/timing-stats.log
```

## Log Rotation and Management

```bash
# Daily log rotation
LOG_DATE=$(date +%Y%m%d)
./build/otsu-obliterator-debug 2>&1 | tee logs/debug-${LOG_DATE}.log

# Size-based log management
find logs/ -name "*.log" -size +10M -exec gzip {} \;

# Clean old logs (keep last 7 days)
find logs/ -name "*.log" -mtime +7 -delete
```