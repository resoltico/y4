# Go 1.24 Project Blueprint: Defensive Flat Architecture

## Core Principles

### Technology Stack
- **Go 1.24 exclusively** - Use latest language features
- **Flat package structure** - Single main package for applications ≤2500 LOC
- **Direct instantiation** - No dependency injection containers
- **Native tooling** - Go-based quality assurance over external scripts
- **Defensive programming** - Input validation and error context without architectural complexity

### Architecture Mandates

#### Flat Structure with Safety Guards
- Single package per application (≤2500 LOC)
- File naming: `prefix_concern.go` (ui_*, processing_*, metrics_*, io_*)
- No `internal/` directories
- No package hierarchies beyond main
- Validation functions for external library boundaries

#### Direct Instantiation with Validation
```go
// Required: Direct construction with validation
app := &Application{
    engine:  NewProcessingEngine(),
    toolbar: NewToolbar(),
}

// Required: Input validation at boundaries
func ProcessImage(img ImageData) (*Result, error) {
    if err := validateImageData(img); err != nil {
        return nil, fmt.Errorf("process image: %w", err)
    }
    // Processing logic
}
```

#### Error Handling with Context
- Direct error returns: `func() (result, error)`
- Context in error messages: `fmt.Errorf("function context: %w", err)`
- Validation at external library boundaries
- Resource cleanup with defer statements

## File Organization Standards

### Mandatory File Structure
```
project/
├── go.mod              # Go 1.24 with native tool management
├── main.go             # Entry point only
├── app.go              # Application setup
├── ui_toolbar.go       # Toolbar widgets
├── ui_parameters.go    # Parameter controls
├── ui_imageviewer.go   # Image display
├── processing.go       # Core algorithms with validation
├── metrics.go          # Quality measurements with safety
├── io_image.go         # File operations
├── quality_check.go    # Native Go quality assurance
└── build.sh           # Build automation
```

### File Naming Conventions
- `ui_*`: All GUI components
- `processing_*`: Algorithm implementations with input validation
- `metrics_*`: Quality measurements with bounds checking
- `io_*`: Input/output operations with error context
- No generic names (util.go, helper.go, common.go)

## Defensive Programming Patterns

### External Library Boundaries
```go
// Required for CGO libraries (OpenCV, C libraries)
func validateMat(mat ExternalMat, context string) error {
    if mat.Empty() {
        return fmt.Errorf("%s: matrix is empty", context)
    }
    if mat.Rows() <= 0 || mat.Cols() <= 0 {
        return fmt.Errorf("%s: invalid dimensions %dx%d", context, mat.Rows(), mat.Cols())
    }
    return nil
}

// Required for operations that can segfault
func safeExternalOperation(data ExternalData) (Result, error) {
    defer func() {
        if r := recover(); r != nil {
            // Log recovery without full structured logging
        }
    }()
    
    if err := validateExternalData(data); err != nil {
        return Result{}, err
    }
    
    return performOperation(data)
}
```

### Resource Management
```go
// Required pattern for external resources
func ProcessWithCleanup(input Input) (*Result, error) {
    resource := acquireExternalResource(input)
    defer resource.Close()
    
    if err := validateResource(resource, "process input"); err != nil {
        return nil, err
    }
    
    return process(resource)
}
```

## Quality Assurance Standards

### Native Go Quality Checking
```go
// quality_check.go - Replace external scripts
func main() {
    checks := []QualityCheck{
        {name: "go vet", cmd: []string{"go", "vet", "./..."}},
        {name: "tests with race detection", cmd: []string{"go", "test", "-race", "./..."}},
        {name: "formatting", cmd: []string{"gofmt", "-l", "."}},
    }
    
    for _, check := range checks {
        if err := runCheck(check); err != nil {
            fail(check.name)
        } else {
            success(check.name)
        }
    }
}
```

### Build System Requirements
- Use `go build` exclusively
- Cross-compilation support
- Race detection in debug builds
- Native quality tools over external scripts

## LOC Management

### Target Ranges
- **≤1500 LOC**: Pure flat structure, minimal validation
- **1500-2500 LOC**: Flat structure with defensive programming
- **>2500 LOC**: Consider architectural patterns beyond this blueprint

### LOC Allocation Guidelines
- Core business logic: 60-70%
- User interface: 15-25%
- Input validation and safety: 10-15%
- Quality assurance tooling: 5-10%

## Go 1.24 Specific Features

### Native Tool Management
```go
// go.mod - Tool dependencies
tool (
    honnef.co/go/tools/cmd/staticcheck v0.5.1
    golang.org/x/vuln/cmd/govulncheck v1.1.3
)
```

### Performance Optimizations
- Swiss Tables map implementation usage
- Small object allocation patterns
- Direct function calls over interfaces
- Minimal escape analysis considerations

## Implementation Guidelines

### Function Design with Safety
- Single responsibility per function
- Maximum 50 lines per function
- Input validation for external library calls
- Context in error messages
- Early returns for invalid states

### Struct Design
- Composition over inheritance
- No embedded interfaces
- Direct field access
- Validation in constructors
- Resource cleanup methods

### Concurrency with Modern Fyne
```go
// Required pattern for UI updates
go func() {
    result, err := heavyOperation()
    if err != nil {
        // Handle error with context
        return
    }
    
    fyne.Do(func() {
        widget.SetContent(result)
    })
}()
```

## Required Patterns

### Always Implement
- Input validation at external library boundaries
- Context in error messages
- Resource cleanup with defer
- Panic recovery for segfault-prone operations
- Native Go quality checking
- Direct error propagation

### Never Implement
- Dependency injection containers
- Complex middleware systems
- Custom error wrapping frameworks
- External scripting for quality checks
- Package hierarchies for small applications
- Interface abstractions without clear benefit

## Success Metrics

### Performance Targets
- Sub-second build times
- Quality checks complete in <5 seconds
- Zero segfaults in production
- Minimal memory allocation overhead

### Code Quality Targets
- ≤2500 LOC total project
- Zero race conditions
- 100% external library operation validation
- Zero linter warnings
- Native tooling dependency only

## Enforcement Rules

### Code Review Requirements
- Flat structure compliance verification
- External library boundary validation check
- Error context presence validation
- Resource cleanup pattern verification
- Native quality tool usage confirmation

### Automatic Rejection Criteria
- Any `internal/` package usage for applications ≤2500 LOC
- Missing validation for external library operations
- Generic error messages without context
- External scripts for quality checking
- Missing resource cleanup
- Unhandled potential segfault scenarios

This blueprint ensures reliable, maintainable Go applications that leverage flat architecture benefits while preventing common pitfalls through defensive programming patterns.