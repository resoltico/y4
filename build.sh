#!/usr/bin/env bash

set -euo pipefail

readonly BINARY_NAME="otsu-obliterator"
readonly VERSION="${VERSION:-1.0.0}"
readonly BUILD_DIR="${BUILD_DIR:-build}"

readonly LDFLAGS="-s -w -X main.version=${VERSION}"
readonly BUILD_TAGS="netgo"

readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'

log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $*"
}

success() {
    echo -e "${GREEN}✓${NC} $*"
}

error() {
    echo -e "${RED}✗${NC} $*" >&2
}

check_deps() {
    log "Checking dependencies..."
    
    if ! command -v go &> /dev/null; then
        error "Go not found. Install Go 1.24+ from https://golang.org/"
        exit 1
    fi
    
    local go_version
    go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
    if [[ "$(printf '%s\n' "1.24" "${go_version}" | sort -V | head -n1)" != "1.24" ]]; then
        error "Go version ${go_version} detected. Go 1.24+ required"
        exit 1
    fi
    
    success "Go ${go_version} detected"
    
    if ! pkg-config --exists opencv4 && ! pkg-config --exists opencv; then
        error "OpenCV not found. Install: brew install opencv (macOS) or apt-get install libopencv-dev (Ubuntu)"
        exit 1
    fi
    
    success "OpenCV found"
    
    if [[ ! -f "go.mod" ]]; then
        error "go.mod not found"
        exit 1
    fi
    
    success "Dependencies verified"
}

clean() {
    log "Cleaning build artifacts..."
    
    if [[ -d "${BUILD_DIR}" ]]; then
        rm -rf "${BUILD_DIR}"
        success "Removed ${BUILD_DIR}/"
    fi
    
    go clean -cache
    go clean -testcache
    
    find . -name "*.prof" -delete 2>/dev/null || true
    find . -name "coverage.*" -delete 2>/dev/null || true
    
    success "Build cache cleaned"
}

build() {
    local target="${1:-default}"
    local output_name="${BINARY_NAME}"
    local build_env=""
    local extra_flags=""
    
    check_deps
    
    case "${target}" in
        "default"|"")
            log "Building ${BINARY_NAME} for current platform"
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
            exit 1
            ;;
    esac
    
    mkdir -p "${BUILD_DIR}"
    
    local build_cmd="go build ${extra_flags} -ldflags \"${LDFLAGS}\" -o \"${BUILD_DIR}/${output_name}\" ."
    
    if [[ -n "${build_env}" ]]; then
        eval "env ${build_env} ${build_cmd}"
    else
        eval "${build_cmd}"
    fi
    
    if [[ -f "${BUILD_DIR}/${output_name}" ]]; then
        local size
        size=$(du -h "${BUILD_DIR}/${output_name}" | cut -f1)
        success "Built: ${BUILD_DIR}/${output_name} (${size})"
    else
        error "Build failed - binary not found: ${BUILD_DIR}/${output_name}"
        exit 1
    fi
}

run_tests() {
    log "Running tests..."
    
    if go test -tags "${BUILD_TAGS}" -race -short -v ./...; then
        success "Tests passed"
    else
        error "Tests failed"
        exit 1
    fi
}

run() {
    check_deps
    build "debug"
    export LOG_LEVEL="${LOG_LEVEL:-info}"
    "./${BUILD_DIR}/${BINARY_NAME}"
}

show_help() {
    cat << 'EOF'
Otsu Obliterator Build System

Usage: ./build.sh [COMMAND] [OPTIONS]

COMMANDS:
  build [target]    Build binary for target platform
  run              Build and run application
  test             Run tests
  clean            Remove build artifacts
  deps             Verify dependencies
  help             Show this help message

BUILD TARGETS:
  default          Current platform
  debug            With race detection
  windows          Windows 64-bit
  macos            macOS Intel 64-bit  
  macos-arm64      macOS Apple Silicon
  linux            Linux 64-bit
  all              All supported platforms

EXAMPLES:
  ./build.sh build              # Build for current platform
  ./build.sh build macos-arm64  # Cross-compile for macOS ARM
  ./build.sh run                # Build and run with debugging
  ./build.sh clean && ./build.sh build all  # Clean rebuild all platforms

For more information, see README.md
EOF
}

main() {
    local command="${1:-help}"
    
    case "${command}" in
        "build")
            build "${2:-}"
            ;;
        "run")
            run
            ;;
        "test")
            run_tests
            ;;
        "clean")
            clean
            ;;
        "deps")
            check_deps
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

main "$@"