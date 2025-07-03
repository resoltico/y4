#!/usr/bin/env bash

# Otsu Obliterator Quality Check Script
# Performs comprehensive code quality auditing, linting, and validation

set -o errexit    # Exit on any command failure
set -o nounset    # Exit on undefined variables
set -o pipefail   # Exit on pipe failures

# Get script directory for relative paths
__dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
__file="${__dir}/$(basename "${BASH_SOURCE[0]}")"

# Project configuration
readonly PROJECT_NAME="otsu-obliterator"
readonly GO_VERSION_REQUIRED="1.24"
readonly COVERAGE_THRESHOLD=75
readonly MAX_COMPLEXITY=15
readonly MAX_LINE_LENGTH=120

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly BLUE='\033[0;34m'
readonly YELLOW='\033[1;33m'
readonly PURPLE='\033[0;35m'
readonly NC='\033[0m'

# Counters
CHECKS_TOTAL=0
CHECKS_PASSED=0
CHECKS_FAILED=0
WARNINGS_COUNT=0

# Logging functions
log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $*"
}

success() {
    echo -e "${GREEN}✓${NC} $*"
    CHECKS_PASSED=$((CHECKS_PASSED + 1))
}

fail() {
    echo -e "${RED}✗${NC} $*" >&2
    CHECKS_FAILED=$((CHECKS_FAILED + 1))
}

warn() {
    echo -e "${YELLOW}⚠${NC} $*"
    WARNINGS_COUNT=$((WARNINGS_COUNT + 1))
}

info() {
    echo -e "${PURPLE}ℹ${NC} $*"
}

# Check counter
check() {
    CHECKS_TOTAL=$((CHECKS_TOTAL + 1))
}

# Environment validation
validate_environment() {
    log "Validating development environment..."
    
    # Check Go version
    check
    if command -v go &> /dev/null; then
        local go_version
        go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
        if [[ "$(printf '%s\n' "${GO_VERSION_REQUIRED}" "${go_version}" | sort -V | head -n1)" == "${GO_VERSION_REQUIRED}" ]]; then
            success "Go version ${go_version} meets requirement (>= ${GO_VERSION_REQUIRED})"
        else
            fail "Go version ${go_version} does not meet requirement (>= ${GO_VERSION_REQUIRED})"
        fi
    else
        fail "Go not found in PATH"
    fi
    
    # Check go.mod exists
    check
    if [[ -f "go.mod" ]]; then
        success "go.mod file exists"
    else
        fail "go.mod file not found"
    fi
    
    # Validate module name
    check
    local module_name
    module_name=$(head -n1 go.mod | cut -d' ' -f2)
    if [[ "${module_name}" == "${PROJECT_NAME}" ]]; then
        success "Module name matches project: ${module_name}"
    else
        fail "Module name mismatch: expected '${PROJECT_NAME}', got '${module_name}'"
    fi
}

# Dependency verification
check_dependencies() {
    log "Checking dependencies..."
    
    # Auto-install govulncheck for vulnerability checking
    auto_install_tool "govulncheck" "go install golang.org/x/vuln/cmd/govulncheck@latest"
    
    # Verify go mod tidy
    check
    if go mod tidy -diff &> /dev/null; then
        success "Dependencies are properly tidied"
    else
        fail "Dependencies require tidying (run 'go mod tidy')"
    fi
    
    # Check for security vulnerabilities
    check
    if command -v govulncheck &> /dev/null; then
        if govulncheck ./...; then
            success "No known security vulnerabilities found"
        else
            fail "Security vulnerabilities detected"
        fi
    else
        warn "govulncheck auto-install failed"
    fi
    
    # Verify module integrity
    check
    if go mod verify; then
        success "Module checksums verified"
    else
        fail "Module verification failed"
    fi
}

# Code formatting checks
check_formatting() {
    log "Checking code formatting..."
    
    # Auto-install goimports if missing
    auto_install_tool "goimports" "go install golang.org/x/tools/cmd/goimports@latest"
    
    # Check gofmt
    check
    local unformatted_files
    unformatted_files=$(gofmt -l . | grep -v vendor/ || true)
    if [[ -z "${unformatted_files}" ]]; then
        success "All Go files are properly formatted"
    else
        fail "Unformatted files found:"
        echo "${unformatted_files}" | sed 's/^/    /'
        info "Run 'gofmt -w .' to fix formatting"
    fi
    
    # Check goimports
    check
    if command -v goimports &> /dev/null; then
        local import_issues
        import_issues=$(goimports -l . | grep -v vendor/ || true)
        if [[ -z "${import_issues}" ]]; then
            success "All imports are properly organized"
        else
            fail "Import organization issues found:"
            echo "${import_issues}" | sed 's/^/    /'
            info "Run 'goimports -w .' to fix imports"
        fi
    else
        fail "goimports auto-install failed"
    fi
}

# Static analysis
run_static_analysis() {
    log "Running static analysis..."
    
    # Auto-install tools
    auto_install_tool "staticcheck" "go install honnef.co/go/tools/cmd/staticcheck@latest"
    auto_install_tool "ineffassign" "go install github.com/gordonklaus/ineffassign@latest"
    auto_install_tool "misspell" "go install github.com/client9/misspell/cmd/misspell@latest"
    
    # Auto-install golangci-lint (special case)
    if ! command -v golangci-lint &> /dev/null; then
        log "Auto-installing golangci-lint..."
        if curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$(go env GOPATH)/bin" latest 2>/dev/null; then
            success "Auto-installed golangci-lint"
        else
            warn "Failed to auto-install golangci-lint"
        fi
    fi
    
    # go vet
    check
    if go vet ./...; then
        success "go vet found no issues"
    else
        fail "go vet found issues"
    fi
    
    # staticcheck
    check
    if command -v staticcheck &> /dev/null; then
        if [[ -f "staticcheck.conf" ]]; then
            if staticcheck -f stylish ./...; then
                success "staticcheck found no issues (using staticcheck.conf)"
            else
                fail "staticcheck found issues"
            fi
        else
            if staticcheck ./...; then
                success "staticcheck found no issues (default config)"
            else
                fail "staticcheck found issues"
            fi
        fi
    else
        fail "staticcheck auto-install failed"
    fi
    
    # golangci-lint
    check
    if command -v golangci-lint &> /dev/null; then
        if [[ -f ".golangci.yml" ]]; then
            if golangci-lint run; then
                success "golangci-lint found no issues (using .golangci.yml)"
            else
                fail "golangci-lint found issues"
            fi
        else
            warn "No .golangci.yml config found, using defaults"
            if golangci-lint run --timeout=5m; then
                success "golangci-lint found no issues (default config)"
            else
                fail "golangci-lint found issues"
            fi
        fi
    else
        warn "golangci-lint not available - install manually from https://golangci-lint.run/"
    fi
    
    # ineffassign
    check
    if command -v ineffassign &> /dev/null; then
        if ineffassign ./...; then
            success "ineffassign found no issues"
        else
            fail "ineffassign found inefficient assignments"
        fi
    else
        warn "ineffassign auto-install failed"
    fi
    
    # misspell
    check
    if command -v misspell &> /dev/null; then
        local misspelled
        misspelled=$(misspell -error . | grep -v vendor/ || true)
        if [[ -z "${misspelled}" ]]; then
            success "misspell found no issues"
        else
            fail "misspell found spelling errors:"
            echo "${misspelled}" | sed 's/^/    /'
        fi
    else
        warn "misspell auto-install failed"
    fi
}

# Security checks
check_security() {
    log "Running security checks..."
    
    # Auto-install tools
    auto_install_tool "gosec" "go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"
    auto_install_tool "govulncheck" "go install golang.org/x/vuln/cmd/govulncheck@latest"
    
    # gosec
    check
    if command -v gosec &> /dev/null; then
        if gosec -quiet ./...; then
            success "gosec found no security issues"
        else
            fail "gosec found security issues"
        fi
    else
        warn "gosec auto-install failed"
    fi
    
    # Check for hardcoded secrets (basic patterns)
    check
    local secret_patterns=(
        "password.*=.*['\"][^'\"]*['\"]"
        "token.*=.*['\"][^'\"]*['\"]"
        "secret.*=.*['\"][^'\"]*['\"]"
        "key.*=.*['\"][^'\"]*['\"]"
    )
    
    local secrets_found=false
    for pattern in "${secret_patterns[@]}"; do
        if grep -r -i -E "${pattern}" --include="*.go" . | grep -v "_test.go"; then
            secrets_found=true
        fi
    done
    
    if [[ "${secrets_found}" == "false" ]]; then
        success "No potential hardcoded secrets found"
    else
        fail "Potential hardcoded secrets detected"
    fi
}

# Test coverage analysis
check_test_coverage() {
    log "Analyzing test coverage..."
    
    # Run tests with coverage
    check
    local coverage_file="coverage.out"
    if go test -race -coverprofile="${coverage_file}" -covermode=atomic ./...; then
        success "All tests pass with race detection"
        
        # Calculate coverage percentage
        if [[ -f "${coverage_file}" ]]; then
            local coverage_percent
            coverage_percent=$(go tool cover -func="${coverage_file}" | grep total | awk '{print $3}' | sed 's/%//')
            
            check
            if (( $(echo "${coverage_percent} >= ${COVERAGE_THRESHOLD}" | bc -l) )); then
                success "Test coverage: ${coverage_percent}% (meets ${COVERAGE_THRESHOLD}% threshold)"
            else
                fail "Test coverage: ${coverage_percent}% (below ${COVERAGE_THRESHOLD}% threshold)"
            fi
            
            # Generate HTML coverage report
            go tool cover -html="${coverage_file}" -o coverage.html
            info "Coverage report generated: coverage.html"
        fi
    else
        fail "Tests failed or race conditions detected"
    fi
    
    # Clean up coverage file
    [[ -f "${coverage_file}" ]] && rm "${coverage_file}"
}

# Code complexity analysis
check_complexity() {
    log "Analyzing code complexity..."
    
    # Auto-install gocyclo
    auto_install_tool "gocyclo" "go install github.com/fzipp/gocyclo/cmd/gocyclo@latest"
    
    check
    if command -v gocyclo &> /dev/null; then
        local complex_functions
        complex_functions=$(gocyclo -over "${MAX_COMPLEXITY}" . | grep -v vendor/ || true)
        if [[ -z "${complex_functions}" ]]; then
            success "No functions exceed complexity threshold (${MAX_COMPLEXITY})"
        else
            fail "Functions with high complexity (>${MAX_COMPLEXITY}):"
            echo "${complex_functions}" | sed 's/^/    /'
        fi
    else
        warn "gocyclo auto-install failed"
    fi
}

# Code quality metrics
check_code_metrics() {
    log "Checking code quality metrics..."
    
    # Check line length
    check
    local long_lines
    long_lines=$(find . -name "*.go" -not -path "./vendor/*" -exec grep -l ".\{${MAX_LINE_LENGTH}\}" {} \; || true)
    if [[ -z "${long_lines}" ]]; then
        success "All lines are within ${MAX_LINE_LENGTH} character limit"
    else
        warn "Files with lines exceeding ${MAX_LINE_LENGTH} characters:"
        echo "${long_lines}" | sed 's/^/    /'
    fi
    
    # Check for TODO/FIXME comments
    check
    local todos
    todos=$(grep -r -n -E "(TODO|FIXME|XXX|HACK)" --include="*.go" . | grep -v vendor/ || true)
    if [[ -z "${todos}" ]]; then
        success "No TODO/FIXME comments found"
    else
        info "TODO/FIXME comments found (${#todos[@]} items):"
        echo "${todos}" | head -10 | sed 's/^/    /'
        if [[ $(echo "${todos}" | wc -l) -gt 10 ]]; then
            info "... and $(($(echo "${todos}" | wc -l) - 10)) more"
        fi
    fi
}

# Performance benchmarks
run_benchmarks() {
    log "Running performance benchmarks..."
    
    check
    if go test -bench=. -benchmem ./... | grep -E "(Benchmark|PASS|FAIL)"; then
        success "Benchmarks completed successfully"
    else
        warn "No benchmarks found or benchmarks failed"
    fi
}

# Documentation checks
check_documentation() {
    log "Checking documentation..."
    
    # Check README exists
    check
    if [[ -f "README.md" ]]; then
        success "README.md exists"
    else
        fail "README.md not found"
    fi
    
    # Check package documentation
    check
    if command -v godoc &> /dev/null; then
        local undocumented
        undocumented=$(go list ./... | xargs -I {} sh -c 'go doc {} 2>/dev/null || echo "Missing: {}"' | grep "Missing:" || true)
        if [[ -z "${undocumented}" ]]; then
            success "All packages have documentation"
        else
            warn "Packages missing documentation:"
            echo "${undocumented}" | sed 's/^/    /'
        fi
    else
        warn "godoc not available"
    fi
}

# Build verification
verify_build() {
    log "Verifying build..."
    
    # Clean build
    check
    if go clean -cache && go build -v ./...; then
        success "Clean build successful"
    else
        fail "Build failed"
    fi
    
    # Cross-compilation check
    check
    local platforms=("linux/amd64" "darwin/amd64" "darwin/arm64" "windows/amd64")
    local build_failures=0
    
    for platform in "${platforms[@]}"; do
        IFS='/' read -r goos goarch <<< "${platform}"
        if GOOS="${goos}" GOARCH="${goarch}" go build -o /tmp/test-build ./cmd/otsu-obliterator > /dev/null 2>&1; then
            info "Cross-compile success: ${platform}"
        else
            warn "Cross-compile failed: ${platform}"
            ((build_failures++))
        fi
    done
    
    if [[ ${build_failures} -eq 0 ]]; then
        success "All cross-compilation targets successful"
    else
        warn "${build_failures} cross-compilation targets failed"
    fi
    
    # Clean up test binary
    rm -f /tmp/test-build
}

# Git repository checks
check_git_status() {
    log "Checking Git repository status..."
    
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        warn "Not in a Git repository"
        return
    fi
    
    # Check for uncommitted changes
    check
    if git diff-index --quiet HEAD --; then
        success "No uncommitted changes"
    else
        warn "Uncommitted changes detected"
        git status --porcelain | sed 's/^/    /'
    fi
    
    # Check branch status
    check
    local branch
    branch=$(git rev-parse --abbrev-ref HEAD)
    if [[ "${branch}" != "main" && "${branch}" != "master" ]]; then
        info "Current branch: ${branch}"
    else
        success "On main branch: ${branch}"
    fi
}

# Generate summary report
generate_summary() {
    echo ""
    echo "=================================="
    echo "Quality Check Summary Report"
    echo "=================================="
    echo "Total Checks: ${CHECKS_TOTAL}"
    echo "Passed: ${CHECKS_PASSED}"
    echo "Failed: ${CHECKS_FAILED}"
    echo "Warnings: ${WARNINGS_COUNT}"
    echo ""
    
    local success_rate
    if [[ ${CHECKS_TOTAL} -gt 0 ]]; then
        success_rate=$(( (CHECKS_PASSED * 100) / CHECKS_TOTAL ))
        echo "Success Rate: ${success_rate}%"
    else
        echo "Success Rate: N/A"
    fi
    
    echo ""
    if [[ ${CHECKS_FAILED} -eq 0 ]]; then
        echo -e "${GREEN}🎉 All quality checks passed!${NC}"
        if [[ ${WARNINGS_COUNT} -gt 0 ]]; then
            echo -e "${YELLOW}⚠ ${WARNINGS_COUNT} warnings should be addressed${NC}"
        fi
    else
        echo -e "${RED}❌ ${CHECKS_FAILED} quality checks failed${NC}"
        echo -e "${YELLOW}⚠ ${WARNINGS_COUNT} warnings found${NC}"
    fi
    echo ""
}

# Auto-install missing tools
auto_install_tool() {
    local tool_name="$1"
    local install_cmd="$2"
    
    if ! command -v "${tool_name}" &> /dev/null; then
        log "Auto-installing ${tool_name}..."
        if eval "${install_cmd}"; then
            success "Auto-installed ${tool_name}"
            return 0
        else
            warn "Failed to auto-install ${tool_name}"
            return 1
        fi
    fi
    return 0
}

# Install missing tools
install_tools() {
    log "Installing missing Go tools..."
    
    local tools=(
        "goimports:go install golang.org/x/tools/cmd/goimports@latest"
        "staticcheck:go install honnef.co/go/tools/cmd/staticcheck@latest"
        "ineffassign:go install github.com/gordonklaus/ineffassign@latest"
        "misspell:go install github.com/client9/misspell/cmd/misspell@latest"
        "gosec:go install github.com/securego/gosec/v2/cmd/gosec@latest"
        "gocyclo:go install github.com/fzipp/gocyclo/cmd/gocyclo@latest"
        "govulncheck:go install golang.org/x/vuln/cmd/govulncheck@latest"
    )
    
    for tool_entry in "${tools[@]}"; do
        IFS=':' read -r tool_name install_cmd <<< "${tool_entry}"
        auto_install_tool "${tool_name}" "${install_cmd}"
    done
    
    # Special handling for golangci-lint (different install method)
    if ! command -v golangci-lint &> /dev/null; then
        log "Auto-installing golangci-lint..."
        if curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$(go env GOPATH)/bin" latest; then
            success "Auto-installed golangci-lint"
        else
            warn "Failed to auto-install golangci-lint - install manually from https://golangci-lint.run/"
        fi
    fi
}

# Help function
show_help() {
    cat << 'EOF'
Quality Check Script for Otsu Obliterator

Usage: ./quality_check.sh [OPTIONS]

OPTIONS:
  check             Run all quality checks (default)
  install-tools     Install missing Go development tools
  fast              Run essential checks only (skip benchmarks)
  coverage          Run only test coverage analysis
  security          Run only security checks
  format            Check only code formatting
  help              Show this help message

EXAMPLES:
  ./quality_check.sh              # Run all checks
  ./quality_check.sh fast         # Run essential checks
  ./quality_check.sh coverage     # Check test coverage only
  ./quality_check.sh install-tools # Install development tools

The script performs comprehensive code quality analysis including:
- Environment validation
- Dependency verification  
- Code formatting checks
- Static analysis
- Security scanning
- Test coverage analysis
- Code complexity metrics
- Documentation validation
- Build verification
- Git repository status

Exit codes:
  0 - All checks passed
  1 - One or more checks failed
  2 - Invalid arguments
EOF
}

# Main execution
main() {
    local command="${1:-check}"
    
    case "${command}" in
        "check"|"")
            validate_environment
            check_dependencies
            check_formatting
            run_static_analysis
            check_security
            check_test_coverage
            check_complexity
            check_code_metrics
            run_benchmarks
            check_documentation
            verify_build
            check_git_status
            ;;
        "fast")
            validate_environment
            check_dependencies
            check_formatting
            run_static_analysis
            check_test_coverage
            verify_build
            ;;
        "coverage")
            check_test_coverage
            ;;
        "security")
            check_security
            ;;
        "format")
            check_formatting
            ;;
        "install-tools")
            install_tools
            exit 0
            ;;
        "help"|"--help"|"-h")
            show_help
            exit 0
            ;;
        *)
            echo "Unknown command: ${command}" >&2
            echo "Use './quality_check.sh help' for usage information" >&2
            exit 2
            ;;
    esac
    
    generate_summary
    
    # Exit with appropriate code
    if [[ ${CHECKS_FAILED} -eq 0 ]]; then
        exit 0
    else
        exit 1
    fi
}

# Run main function
main "$@"