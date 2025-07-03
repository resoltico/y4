#!/usr/bin/env bash

# Otsu Obliterator Build Script
# Follows bash best practices and Go project conventions

set -o errexit    # Exit on any command failure
set -o nounset    # Exit on undefined variables
set -o pipefail   # Exit on pipe failures

# Get script directory for relative paths
__dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
__file="${__dir}/$(basename "${BASH_SOURCE[0]}")"
__base="$(basename "${__file}" .sh)"

# Project configuration
readonly BINARY_NAME="otsu-obliterator"
readonly VERSION="${VERSION:-1.0.0}"
readonly BUILD_DIR="${BUILD_DIR:-build}"
readonly CMD_DIR="${CMD_DIR:-cmd/${BINARY_NAME}}"

# Build configuration
readonly LDFLAGS="-s -w -X main.version=${VERSION}"
readonly BUILD_TAGS="matprofile"

# Platform detection
readonly OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
readonly ARCH="$(uname -m)"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly BLUE='\033[0;34m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m'

# Logging functions
log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $*"
}

success() {
    echo -e "${GREEN}✓${NC} $*"
}

error() {
    echo -e "${RED}✗${NC} $*" >&2
}

warn() {
    echo -e "${YELLOW}⚠${NC} $*"
}

# Help function with auto-documentation
show_help() {
    cat << 'EOF'
Otsu Obliterator Build System

CRITICAL: This script uses 'go build' instead of 'fyne build' to preserve 
proper main() entry points. This ensures menu initialization and About 
dialog functionality work correctly.

Usage: ./build.sh [COMMAND] [OPTIONS]

COMMANDS:
  build [target]    Build binary for target platform
  run              Build and run application  
  debug [type]     Run with debugging enabled
  test             Run tests with coverage
  bench            Run benchmarks
  clean            Remove build artifacts and clean cache
  deps             Install and verify dependencies
  format           Format code and organize imports
  lint             Run static analysis
  audit            Run quality control checks
  package [target] Create distribution packages
  help             Show this help message

BUILD TARGETS:
  default          Current platform (auto-detected)
  profile          With profiling enabled
  debug            With race detection
  windows          Windows 64-bit
  macos            macOS Intel 64-bit  
  macos-arm64      macOS Apple Silicon
  linux            Linux 64-bit
  all              All supported platforms

DEBUG TYPES:
  basic            LOG_LEVEL=debug
  memory           Memory debugging with MatProfile
  race             Race condition detection

PACKAGE TARGETS:
  windows          Windows installer (.exe)
  macos            macOS application bundle (.app)
  linux            Linux distribution package

EXAMPLES:
  ./build.sh build              # Build for current platform
  ./build.sh build macos-arm64  # Cross-compile for macOS ARM
  ./build.sh debug memory       # Run with memory debugging
  ./build.sh clean && ./build.sh build all  # Clean rebuild all platforms
  ./build.sh audit              # Full quality control check

TROUBLESHOOTING:
  - Missing menus/About dialog: Ensure using this script, not 'fyne build'
  - Build failures: Run './build.sh deps' to verify dependencies
  - Memory issues: Use './build.sh debug memory' for diagnostics

For more information, see README.md
EOF
}

# Dependency checking with detailed feedback
check_deps() {
    log "Checking dependencies..."
    local errors=0
    
    # Check Go installation
    if ! command -v go &> /dev/null; then
        error "Go not found. Install Go 1.21+ from https://golang.org/"
        ((errors++))
    else
        local go_version
        go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
        if [[ "$(printf '%s\n' "1.21" "${go_version}" | sort -V | head -n1)" != "1.21" ]]; then
            warn "Go version ${go_version} detected. Go 1.21+ recommended"
        else
            success "Go ${go_version} detected"
        fi
    fi
    
    # Check OpenCV
    if ! pkg-config --exists opencv4 && ! pkg-config --exists opencv; then
        warn "OpenCV not found - some features may be limited"
        warn "Install: brew install opencv (macOS) or apt-get install libopencv-dev (Ubuntu)"
    else
        success "OpenCV found"
    fi
    
    # Verify Go modules
    if [[ ! -f "go.mod" ]]; then
        error "go.mod not found. Run 'go mod init' first"
        ((errors++))
    fi
    
    # Check cmd directory
    if [[ ! -d "${CMD_DIR}" ]]; then
        error "Command directory '${CMD_DIR}' not found"
        ((errors++))
    fi
    
    if [[ ${errors} -gt 0 ]]; then
        error "${errors} dependency error(s) found"
        exit 1
    fi
    
    success "All dependencies verified"
}

# Smart build cache management
clean_build_cache() {
    log "Cleaning build cache and artifacts..."
    
    # Remove build directory
    if [[ -d "${BUILD_DIR}" ]]; then
        rm -rf "${BUILD_DIR}"
        success "Removed ${BUILD_DIR}/"
    fi
    
    # Clean Go cache
    go clean -cache
    go clean -testcache
    go clean -modcache 2>/dev/null || true
    
    # Remove common artifacts
    find . -name "*.prof" -delete 2>/dev/null || true
    find . -name "coverage.*" -delete 2>/dev/null || true
    
    success "Build cache cleaned"
}

# Auto-clean obsolete builds on version changes
auto_clean_obsolete() {
    local version_file="${BUILD_DIR}/.version"
    
    if [[ -f "${version_file}" ]]; then
        local old_version
        old_version=$(<"${version_file}")
        if [[ "${old_version}" != "${VERSION}" ]]; then
            log "Version changed (${old_version} → ${VERSION}), cleaning obsolete builds"
            clean_build_cache
        fi
    fi
    
    mkdir -p "${BUILD_DIR}"
    echo "${VERSION}" > "${version_file}"
}

# Enhanced build function with progress and validation
build() {
    local target="${1:-default}"
    local output_name="${BINARY_NAME}"
    local build_env=""
    local extra_flags=""
    
    check_deps
    auto_clean_obsolete
    
    case "${target}" in
        "default"|"")
            log "Building ${BINARY_NAME} for current platform (${OS}/${ARCH})"
            ;;
        "profile")
            extra_flags="-tags ${BUILD_TAGS} -race"
            log "Building with profiling and race detection"
            ;;
        "debug")
            extra_flags="-tags ${BUILD_TAGS} -race -gcflags=all=-N -l"
            log "Building debug version with race detection"
            ;;
        "windows")
            output_name="${BINARY_NAME}.exe"
            build_env="GOOS=windows GOARCH=amd64"
            extra_flags="-tags ${BUILD_TAGS}"
            log "Cross-compiling for Windows AMD64"
            ;;
        "macos")
            output_name="${BINARY_NAME}-macos-amd64"
            build_env="GOOS=darwin GOARCH=amd64"
            extra_flags="-tags ${BUILD_TAGS}"
            log "Cross-compiling for macOS Intel"
            ;;
        "macos-arm64")
            output_name="${BINARY_NAME}-macos-arm64"
            build_env="GOOS=darwin GOARCH=arm64"
            extra_flags="-tags ${BUILD_TAGS}"
            log "Cross-compiling for macOS Apple Silicon"
            ;;
        "linux")
            output_name="${BINARY_NAME}-linux-amd64"
            build_env="GOOS=linux GOARCH=amd64"
            extra_flags="-tags ${BUILD_TAGS}"
            log "Cross-compiling for Linux AMD64"
            ;;
        "all")
            log "Building for all supported platforms"
            build "windows"
            build "macos"
            build "macos-arm64"
            build "linux"
            return 0
            ;;
        *)
            error "Unknown build target: ${target}"
            error "Supported: default, profile, debug, windows, macos, macos-arm64, linux, all"
            exit 1
            ;;
    esac
    
    mkdir -p "${BUILD_DIR}"
    
    # Build command construction
    local build_cmd="go build ${extra_flags} -ldflags \"${LDFLAGS}\" -o \"${BUILD_DIR}/${output_name}\" \"./${CMD_DIR}\""
    
    # Execute build
    if [[ -n "${build_env}" ]]; then
        eval "env ${build_env} ${build_cmd}"
    else
        eval "${build_cmd}"
    fi
    
    # Verify build
    if [[ -f "${BUILD_DIR}/${output_name}" ]]; then
        local size
        size=$(du -h "${BUILD_DIR}/${output_name}" | cut -f1)
        success "Built: ${BUILD_DIR}/${output_name} (${size})"
        
        # Show binary info for verification
        if command -v file &> /dev/null; then
            file "${BUILD_DIR}/${output_name}" | sed 's/^/  /'
        fi
    else
        error "Build failed - binary not found: ${BUILD_DIR}/${output_name}"
        exit 1
    fi
}

# Enhanced test runner with coverage
run_tests() {
    log "Running tests with coverage..."
    
    # Create coverage directory
    mkdir -p coverage
    
    # Run tests with race detection and coverage
    if go test -tags "${BUILD_TAGS}" -race -coverprofile=coverage/coverage.out -covermode=atomic -v ./...; then
        # Generate coverage report
        go tool cover -html=coverage/coverage.out -o coverage/coverage.html
        
        # Show coverage summary
        local coverage_pct
        coverage_pct=$(go tool cover -func=coverage/coverage.out | grep total | grep -oE '[0-9]+\.[0-9]+%')
        success "Tests passed - Coverage: ${coverage_pct}"
        log "Coverage report: coverage/coverage.html"
    else
        error "Tests failed"
        exit 1
    fi
}

# Enhanced debug runner with environment setup
run_debug() {
    local debug_type="${1:-basic}"
    
    check_deps
    
    case "${debug_type}" in
        "basic")
            log "Running with basic debugging"
            env LOG_LEVEL=debug go run -tags "${BUILD_TAGS}" "./${CMD_DIR}"
            ;;
        "memory")
            log "Running with memory debugging and MatProfile"
            env LOG_LEVEL=debug GOMAXPROCS=1 go run -tags "${BUILD_TAGS}" -race "./${CMD_DIR}"
            ;;
        "race")
            log "Running with race condition detection"
            env LOG_LEVEL=debug go run -tags "${BUILD_TAGS}" -race "./${CMD_DIR}"
            ;;
        *)
            error "Unknown debug type: ${debug_type}"
            error "Supported: basic, memory, race"
            exit 1
            ;;
    esac
}

# Quality control audit
run_audit() {
    log "Running quality control audit..."
    
    # Format check
    log "Checking code formatting..."
    if [[ -n "$(gofmt -l .)" ]]; then
        error "Code not properly formatted. Run './build.sh format'"
        exit 1
    fi
    
    # Vet check
    log "Running go vet..."
    go vet ./...
    
    # Module verification
    log "Verifying modules..."
    go mod verify
    go mod tidy -diff
    
    # Tests
    run_tests
    
    # Static analysis (if available)
    if command -v staticcheck &> /dev/null; then
        log "Running static analysis..."
        staticcheck ./...
    else
        warn "staticcheck not found - install with: go install honnef.co/go/tools/cmd/staticcheck@latest"
    fi
    
    success "Quality control audit completed"
}

# Main command dispatcher
main() {
    local command="${1:-help}"
    
    case "${command}" in
        "build")
            build "${2:-}"
            ;;
        "run")
            check_deps
            build "profile"
            export LOG_LEVEL="${LOG_LEVEL:-info}"
            export GOMAXPROCS="${GOMAXPROCS:-$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)}"
            "./${BUILD_DIR}/${BINARY_NAME}"
            ;;
        "debug")
            run_debug "${2:-basic}"
            ;;
        "test")
            run_tests
            ;;
        "bench")
            log "Running benchmarks..."
            go test -tags "${BUILD_TAGS}" -bench=. -benchmem ./...
            success "Benchmarks completed"
            ;;
        "clean")
            clean_build_cache
            ;;
        "deps")
            log "Installing and verifying dependencies..."
            go mod download
            go mod verify
            go mod tidy
            check_deps
            success "Dependencies updated"
            ;;
        "format")
            log "Formatting code..."
            go fmt ./...
            if command -v goimports &> /dev/null; then
                goimports -w .
            fi
            success "Code formatted"
            ;;
        "lint")
            log "Running linters..."
            go vet ./...
            if command -v golangci-lint &> /dev/null; then
                golangci-lint run
            else
                warn "golangci-lint not found. Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
            fi
            success "Linting completed"
            ;;
        "audit")
            run_audit
            ;;
        "package")
            log "Packaging not yet implemented - use build for now"
            warn "Coming soon: Distribution packaging"
            ;;
        "help"|"--help"|"-h")
            show_help
            ;;
        *)
            error "Unknown command: ${command}"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# Trap for cleanup on exit
cleanup() {
    local exit_code=$?
    if [[ ${exit_code} -ne 0 ]]; then
        error "Build script failed with exit code ${exit_code}"
    fi
    exit ${exit_code}
}

trap cleanup EXIT ERR

# Run main function
main "$@"