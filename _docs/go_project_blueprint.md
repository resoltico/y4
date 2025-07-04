# Go 1.24 Project Design Blueprint

## Core Principles

### Technology Stack Constraints
- **Go 1.24 exclusively** - Use latest language features, no version compatibility
- **Fyne v2.6.1** - Leverage new threading model and performance improvements
- **GoCV v0.41.0** - Use current API methods only
- **Zero backwards compatibility** - No legacy pattern support
- **No graceful degradation** - Fail fast on unsupported features

### Architecture Mandates

#### Flat Structure Requirements
- Single package per application (≤2000 LOC)
- File naming: `prefix_concern.go` (ui_*, processing_*, metrics_*, io_*)
- No `internal/` directories
- No package hierarchies beyond main
- Direct imports only

#### Direct Instantiation Pattern
```go
// Mandatory: Direct construction
app := &Application{
    engine:  NewProcessingEngine(),
    toolbar: NewToolbar(),
}

// Forbidden: Dependency injection
type Container interface {
    Resolve(name string) interface{}
}
```

#### Error Handling Specifications
- Direct error returns: `func() (result, error)`
- No wrapped error types or error chains
- No custom error handling frameworks
- Fail fast: `if err != nil { return nil, err }`

### LOC Reduction Rules

#### Eliminate These Patterns (-65% LOC)
1. **Package Hierarchies**: No nested package structures
2. **Interface Abstractions**: Direct struct usage
3. **Factory Patterns**: Direct constructors only
4. **Builder Patterns**: Struct literals with initialization
5. **Middleware Chains**: Direct function calls
6. **Event Systems**: Direct method invocation

#### Memory Management (-50% LOC)
- Use `defer mat.Close()` exclusively
- No custom memory tracking
- No object pools
- No reference counting
- Direct GoCV Mat lifecycle

#### Configuration Simplification (-45% LOC)
- Struct-based parameters only
- No validation frameworks
- No configuration files
- No environment variable systems
- Direct parameter passing

## File Organization Standards

### Mandatory File Structure
```
project/
├── go.mod              # Go 1.24 tool directive
├── main.go             # Entry point only
├── app.go              # Application setup
├── ui_toolbar.go       # Toolbar widgets
├── ui_parameters.go    # Parameter controls
├── ui_imageviewer.go   # Image display
├── processing.go       # Core algorithms
├── metrics.go          # Quality measurements
├── io_image.go         # File operations
├── build.sh           # Build automation
└── quality_check.sh   # Quality validation
```

### File Naming Conventions
- `ui_*`: All GUI components
- `processing_*`: Algorithm implementations
- `metrics_*`: Quality measurements
- `io_*`: Input/output operations
- `config_*`: Configuration management
- No generic names (util.go, helper.go, common.go)

## Go 1.24 Specific Requirements

### Tool Management
```go
// go.mod tool directive mandatory
tool (
    honnef.co/go/tools/cmd/staticcheck v0.5.1
    golang.org/x/vuln/cmd/govulncheck v1.1.3
)
```

### Threading with Fyne v2.6+
```go
// Mandatory pattern for UI updates
go func() {
    result := heavyOperation()
    fyne.Do(func() {
        widget.SetContent(result)
    })
}()
```

### Performance Optimizations
- Swiss Tables map implementation usage
- Small object allocation patterns
- Direct function calls over interfaces
- Escape analysis considerations

## Quality Standards

### Build System Requirements
- Use `go build` exclusively (never `fyne build`)
- Cross-compilation support mandatory
- Race detection in debug builds
- Static linking where possible

### Quality Check Specifications
```bash
# Core checks (100% reliable)
go vet ./...
go test -race -short ./...
test -z "$(gofmt -l .)" || exit 1

# External tools (conditional)
command -v staticcheck && staticcheck -checks="all,-SA1019" ./...
```

### Testing Constraints
- No testing frameworks beyond standard library
- No mocking libraries
- No benchmark infrastructure
- Race detection mandatory
- Short tests only in CI

## Forbidden Patterns

### Never Implement
- Dependency injection containers
- Observer/event bus patterns
- Middleware chain systems
- Complex error wrapping
- Configuration file parsers
- Custom logging frameworks
- Memory management abstractions
- Interface{} usage for type safety
- Reflection-based solutions
- Generic utility functions

### Legacy API Avoidance
- No deprecated Fyne widgets
- No obsolete GoCV functions
- No pre-1.24 Go patterns
- No compatibility shims
- No version detection code

## Implementation Guidelines

### Function Design
- Single responsibility per function
- Maximum 50 lines per function
- Direct parameter passing
- No variadic functions unless necessary
- Return early pattern mandatory

### Struct Design
- Composition over inheritance
- No embedded interfaces
- Direct field access
- No getter/setter methods
- Initialization in constructors

### Concurrency Rules
- Fyne v2.6+ threading model only
- No custom goroutine pools
- Context cancellation mandatory
- No shared mutable state
- Channel communication over mutexes

## Metrics Implementation Standards

### Required Metrics (6 mandatory)
- F-measure: Standard pixel accuracy
- Pseudo F-measure: Weighted accuracy
- DRD: Distance reciprocal distortion
- MPM: Morphological path misalignment
- NRM: Normalized root mean square
- PBC: Peak background contrast

### Implementation Requirements
- Direct calculation functions
- No metric frameworks
- Matrix operations using GoCV
- Performance over abstraction

## Build and Deployment

### Build Script Mandates
- Cross-platform compilation
- Dependency verification
- Static analysis integration
- Binary size optimization
- Debug symbol stripping

### Quality Assurance
- Under 200 lines quality script
- Go 1.24 tool directive usage
- Container-based CI preferred
- No auto-installation fallbacks

## Success Metrics

### Performance Targets
- 15-25% faster startup time
- 10-20% smaller binary size
- 5-10% lower memory usage
- Sub-second build times

### Code Quality Targets
- ≤2000 LOC total project
- ≤50 LOC average file size
- Zero external dependencies beyond core stack
- 100% test coverage for algorithms
- Zero linter warnings

## Enforcement Rules

### Code Review Requirements
- Flat structure compliance check
- Direct instantiation verification
- Legacy pattern detection
- LOC impact assessment
- Performance measurement validation

### Automatic Rejection Criteria
- Any `internal/` package usage
- Interface-based dependency injection
- Custom error type definitions
- Testing framework imports
- Backwards compatibility code
- Deprecated API usage

This blueprint ensures consistent, high-performance Go applications that leverage modern language features while maintaining simplicity and efficiency.