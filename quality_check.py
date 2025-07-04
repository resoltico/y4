#!/usr/bin/env python3

import subprocess
import sys
import os
import re

# Constants
PROJECT_NAME = "otsu-obliterator"
GO_VERSION_REQUIRED = "1.24"
GREEN = '\033[0;32m'
RED = '\033[0;31m'
YELLOW = '\033[1;33m'
NC = '\033[0m'

CHECKS_PASSED = 0
CHECKS_FAILED = 0

def success(message):
    global CHECKS_PASSED
    print(f"{GREEN}✓{NC} {message}")
    CHECKS_PASSED += 1

def fail(message):
    global CHECKS_FAILED
    print(f"{RED}✗{NC} {message}", file=sys.stderr)
    CHECKS_FAILED += 1

def warn(message):
    print(f"{YELLOW}⚠{NC} {message}")

def debug(message):
    print(f"[DEBUG] {message}")

# Run a command with the current environment
def run_command(command):
    try:
        env = os.environ.copy()
        result = subprocess.run(command, shell=True, capture_output=True, text=True, env=env)
        if result.returncode == 0:
            return True, result.stdout.strip()
        else:
            return False, result.stderr.strip()
    except Exception as e:
        return False, str(e)

def get_gopath():
    success, output = run_command("go env GOPATH")
    if success:
        return output
    else:
        fail("Failed to get GOPATH")
        sys.exit(1)

def install_tool(tool_name, install_command):
    debug(f"Checking if {tool_name} is installed...")
    installed, _ = run_command(f"command -v {tool_name}")
    if not installed:
        warn(f"{tool_name} not available, attempting to install...")
        success_install, install_output = run_command(install_command)
        if success_install:
            success(f"{tool_name} installed successfully")
            gopath = get_gopath()
            bin_path = os.path.join(gopath, "bin")
            os.environ["PATH"] += os.pathsep + bin_path
            debug(f"Added {bin_path} to PATH")
        else:
            fail(f"Failed to install {tool_name}: {install_output}")
    else:
        success(f"{tool_name} is already installed")

def validate_environment():
    debug("Entering validate_environment")
    print("Validating environment...")
    
    go_installed, _ = run_command("command -v go")
    if not go_installed:
        fail("Go not found in PATH")
        debug("Go not found, exiting validate_environment")
        return False
    
    debug("Go command found")
    
    go_version_success, go_version_output = run_command("go version")
    if not go_version_success:
        fail("Unable to determine Go version")
        debug("Failed to get Go version, exiting validate_environment")
        return False
    
    go_version = re.search(r'go(\d+\.\d+)', go_version_output)
    if go_version:
        go_version = go_version.group(1)
        debug(f"Go version extracted: {go_version}")
        if go_version < GO_VERSION_REQUIRED:
            fail(f"Go version {go_version} below requirement (>= {GO_VERSION_REQUIRED})")
            debug("Go version too low, exiting validate_environment")
            return False
        else:
            success(f"Go version {go_version} meets requirement (>= {GO_VERSION_REQUIRED})")
    else:
        fail("Unable to parse Go version")
        debug("Go version parsing failed, exiting validate_environment")
        return False
    
    debug("Checking for go.mod")
    if os.path.isfile("go.mod"):
        success("go.mod exists")
        with open("go.mod", "r") as f:
            first_line = f.readline().strip()
            module_path = first_line.split(" ")[1] if len(first_line.split(" ")) > 1 else ""
            module_name = module_path.split("/")[-1]
            debug(f"Module path from go.mod: {module_path}")
            if module_name == PROJECT_NAME:
                success(f"Module name matches project (got '{module_name}')")
            else:
                fail(f"Module name mismatch: expected '{PROJECT_NAME}', got '{module_name}' (from '{module_path}')")
    else:
        fail("go.mod not found")
        debug("go.mod missing, continuing")
    
    debug("Exiting validate_environment")
    return True

def check_tools():
    debug("Entering check_tools")
    print("Checking available tools...")
    install_tool("staticcheck", "go install honnef.co/go/tools/cmd/staticcheck@latest")
    install_tool("govulncheck", "go install golang.org/x/vuln/cmd/govulncheck@latest")
    debug("Exiting check_tools")

def check_formatting():
    debug("Entering check_formatting")
    print("Checking code formatting...")
    success_cmd, output = run_command("gofmt -l . | grep -v vendor/ || true")
    if not output:
        success("All Go files formatted correctly")
    else:
        fail(f"Unformatted files found: {output}")
    debug("Exiting check_formatting")

def run_core_checks():
    debug("Entering run_core_checks")
    print("Running core quality checks...")
    for cmd, desc in [
        ("go vet ./...", "go vet passed"),
        ("go test -race -short ./...", "Tests passed with race detection"),
        ("go mod tidy -diff", "Dependencies properly managed"),
        ("go mod verify", "Module checksums verified")
    ]:
        success_cmd, _ = run_command(cmd)
        if success_cmd:
            success(desc)
        else:
            fail(f"{desc.split(' ')[0]} failed")
    debug("Exiting run_core_checks")

def run_external_tools():
    debug("Entering run_external_tools")
    print("Running external tools (conditional)...")
    if run_command("command -v staticcheck")[0]:
        success_cmd, _ = run_command("staticcheck -checks='all,-SA1019' ./...")
        if success_cmd:
            success("staticcheck passed")
        else:
            fail("staticcheck found issues")
    else:
        warn("staticcheck not available")
    
    if run_command("command -v govulncheck")[0]:
        success_cmd, _ = run_command("govulncheck ./...")
        if success_cmd:
            success("No security vulnerabilities found")
        else:
            fail("Security vulnerabilities detected")
    else:
        warn("govulncheck not available")
    debug("Exiting run_external_tools")

def check_build():
    debug("Entering check_build")
    print("Verifying build...")
    success_cmd, _ = run_command("go build -v .")
    if success_cmd:
        success("Build successful")
        if os.path.exists(PROJECT_NAME):
            os.remove(PROJECT_NAME)
    else:
        fail("Build failed")
    debug("Exiting check_build")

def generate_summary():
    debug("Entering generate_summary")
    print("\n==================================")
    print("Quality Check Summary")
    print("==================================")
    print(f"Passed: {CHECKS_PASSED}")
    print(f"Failed: {CHECKS_FAILED}\n")
    if CHECKS_FAILED == 0:
        print(f"{GREEN}All quality checks passed{NC}")
    else:
        print(f"{RED}{CHECKS_FAILED} quality checks failed{NC}")
    debug("Exiting generate_summary")

def main():
    debug("Entering main")
    command = sys.argv[1] if len(sys.argv) > 1 else "check"
    
    if command in ["check", ""]:
        validate_environment()
        check_tools()
        check_formatting()
        run_core_checks()
        run_external_tools()
        check_build()
    elif command == "fast":
        validate_environment()
        check_formatting()
        run_core_checks()
        check_build()
    elif command == "format":
        check_formatting()
    else:
        print("Usage: quality_check.py [check|fast|format]")
        sys.exit(1)
    
    generate_summary()
    
    if CHECKS_FAILED == 0:
        sys.exit(0)
    else:
        sys.exit(1)
    debug("Exiting main")

if __name__ == "__main__":
    main()