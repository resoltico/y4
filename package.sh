#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PACKAGE_TOOL="cmd/package/main.go"

readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
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

show_help() {
    cat << 'EOF'
macOS Packaging Script for Otsu Obliterator

Usage: ./package.sh [COMMAND] [OPTIONS]

COMMANDS:
  package [binary]     Build .app and .dmg (default: build/otsu-obliterator)
  clean               Remove packaging artifacts
  verify [app]        Verify .app bundle structure
  setup               Initialize packaging environment
  help                Show this help message

EXAMPLES:
  ./package.sh package                    # Package default binary
  ./package.sh package build/debug-build # Package specific binary
  ./package.sh clean                      # Clean packaging artifacts
  ./package.sh verify dist/Otsu\ Obliterator.app

REQUIREMENTS:
  - macOS with Xcode Command Line Tools
  - Go 1.24+
  - Built binary in build/ directory
  - Optional: icon.png for custom icon
EOF
}

check_requirements() {
    log "Checking requirements..."
    
    # Check if we're on macOS
    if [[ "$(uname)" != "Darwin" ]]; then
        error "This script must be run on macOS"
        exit 1
    fi
    
    # Check for required tools
    local tools=("go" "hdiutil" "iconutil" "sips")
    for tool in "${tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            error "Required tool not found: $tool"
            case "$tool" in
                "go")
                    echo "Install Go from: https://golang.org/dl/"
                    ;;
                "hdiutil"|"iconutil"|"sips")
                    echo "Install Xcode Command Line Tools: xcode-select --install"
                    ;;
            esac
            exit 1
        fi
    done
    
    success "All required tools found"
    
    # Check Go version
    local go_version
    go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
    if [[ "${go_version}" < "1.24" ]]; then
        error "Go 1.24+ required, found ${go_version}"
        exit 1
    fi
    
    success "Go ${go_version} is compatible"
}

setup_packaging() {
    log "Setting up packaging environment..."
    
    # Create packaging tool if it doesn't exist
    if [[ ! -f "${PACKAGE_TOOL}" ]]; then
        error "Package tool not found: ${PACKAGE_TOOL}"
        echo "Please create the packaging tool first"
        exit 1
    fi
    
    # Ensure directories exist
    mkdir -p build dist
    
    # Check if main binary exists
    if [[ ! -f "build/otsu-obliterator" ]]; then
        log "Building main binary..."
        if ! go build -o build/otsu-obliterator .; then
            error "Failed to build main binary"
            exit 1
        fi
        success "Built main binary"
    fi
    
    success "Packaging environment ready"
}

package_app() {
    local binary_path="${1:-build/otsu-obliterator}"
    
    check_requirements
    setup_packaging
    
    log "Packaging application..."
    
    if [[ ! -f "${binary_path}" ]]; then
        error "Binary not found: ${binary_path}"
        echo "Build the binary first: ./build.sh build"
        exit 1
    fi
    
    # Run the packaging tool
    if ! go run "${PACKAGE_TOOL}" package "${binary_path}"; then
        error "Packaging failed"
        exit 1
    fi
    
    success "Packaging completed successfully"
    
    # Show final output
    if [[ -d "dist/Otsu Obliterator.app" ]]; then
        echo ""
        echo "📱 App Bundle: dist/Otsu Obliterator.app"
        echo "💿 DMG File: dist/Otsu-Obliterator.dmg"
        echo ""
        echo "To test the app:"
        echo "  open 'dist/Otsu Obliterator.app'"
        echo ""
        echo "To mount the DMG:"
        echo "  open dist/Otsu-Obliterator.dmg"
    fi
}

clean_artifacts() {
    log "Cleaning packaging artifacts..."
    
    if go run "${PACKAGE_TOOL}" clean; then
        success "Cleanup completed"
    else
        error "Cleanup failed"
        exit 1
    fi
}

verify_app() {
    local app_path="${1:-dist/Otsu Obliterator.app}"
    
    log "Verifying app bundle: ${app_path}"
    
    if go run "${PACKAGE_TOOL}" verify "${app_path}"; then
        success "Verification completed"
    else
        error "Verification failed"
        exit 1
    fi
}

main() {
    local command="${1:-help}"
    
    case "${command}" in
        "package")
            package_app "${2:-}"
            ;;
        "clean")
            clean_artifacts
            ;;
        "verify")
            verify_app "${2:-}"
            ;;
        "setup")
            setup_packaging
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