package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	ProjectName = "otsu-obliterator"
	GoVersion   = "1.24"
	ColorGreen  = "\033[0;32m"
	ColorRed    = "\033[0;31m"
	ColorYellow = "\033[1;33m"
	ColorReset  = "\033[0m"
)

type QualityChecker struct {
	checksPassed int
	checksFailed int
	gopath       string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/quality_check/main.go [check|fast|format]")
		os.Exit(1)
	}

	qc := &QualityChecker{}

	gopath, err := qc.runCommand("go", "env", "GOPATH")
	if err != nil {
		fmt.Printf("%s✗%s Could not determine GOPATH\n", ColorRed, ColorReset)
		os.Exit(1)
	}
	qc.gopath = strings.TrimSpace(gopath)

	command := os.Args[1]
	switch command {
	case "check":
		qc.runAllChecks()
	case "fast":
		qc.runFastChecks()
	case "format":
		qc.checkFormatting()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}

	qc.generateSummary()
	if qc.checksFailed > 0 {
		os.Exit(1)
	}
}

func (qc *QualityChecker) runAllChecks() {
	qc.validateEnvironment()
	qc.ensureTools()
	qc.checkFormatting()
	qc.runCoreChecks()
	qc.runExternalTools()
	qc.checkBuild()
}

func (qc *QualityChecker) runFastChecks() {
	qc.validateEnvironment()
	qc.checkFormatting()
	qc.runCoreChecks()
	qc.checkBuild()
}

func (qc *QualityChecker) validateEnvironment() {
	fmt.Println("Validating environment...")

	output, err := qc.runCommand("go", "version")
	if err != nil {
		qc.fail("Go not found")
		return
	}

	versionRegex := regexp.MustCompile(`go(\d+\.\d+)`)
	matches := versionRegex.FindStringSubmatch(output)
	if len(matches) < 2 {
		qc.fail("Unable to parse Go version")
		return
	}

	version := matches[1]
	if version != GoVersion {
		qc.fail(fmt.Sprintf("Go version %s required, found %s", GoVersion, version))
		return
	}
	qc.success(fmt.Sprintf("Go version %s matches requirement", version))

	if !qc.fileExists("go.mod") {
		qc.fail("go.mod not found")
		return
	}
	qc.success("go.mod exists")

	moduleName := qc.extractModuleName()
	if moduleName != ProjectName {
		qc.fail(fmt.Sprintf("Module name mismatch: expected '%s', got '%s'", ProjectName, moduleName))
		return
	}
	qc.success(fmt.Sprintf("Module name matches project ('%s')", moduleName))
}

func (qc *QualityChecker) ensureTools() {
	fmt.Println("Ensuring tools are available...")

	tools := []struct {
		name       string
		binaryPath string
		installCmd []string
	}{
		{
			name:       "staticcheck",
			binaryPath: qc.gopath + "/bin/staticcheck",
			installCmd: []string{"go", "install", "honnef.co/go/tools/cmd/staticcheck@latest"},
		},
		{
			name:       "govulncheck",
			binaryPath: qc.gopath + "/bin/govulncheck",
			installCmd: []string{"go", "install", "golang.org/x/vuln/cmd/govulncheck@latest"},
		},
	}

	for _, tool := range tools {
		if qc.fileExists(tool.binaryPath) {
			qc.success(fmt.Sprintf("%s is available", tool.name))
		} else {
			qc.warn(fmt.Sprintf("%s not found, installing...", tool.name))
			if err := qc.runCommandSilent(tool.installCmd[0], tool.installCmd[1:]...); err != nil {
				qc.fail(fmt.Sprintf("Failed to install %s", tool.name))
			} else {
				qc.success(fmt.Sprintf("%s installed", tool.name))
			}
		}
	}
}

func (qc *QualityChecker) checkFormatting() {
	fmt.Println("Checking code formatting...")

	output, err := qc.runCommand("gofmt", "-l", ".")
	if err != nil {
		qc.fail("gofmt check failed")
		return
	}

	unformatted := strings.TrimSpace(output)
	if unformatted == "" {
		qc.success("All Go files formatted")
	} else {
		files := strings.Split(unformatted, "\n")
		qc.fail(fmt.Sprintf("Unformatted files found: %s", strings.Join(files, ", ")))
		fmt.Printf("   Run: gofmt -w %s\n", strings.Join(files, " "))
	}
}

func (qc *QualityChecker) runCoreChecks() {
	fmt.Println("Running core quality checks...")

	checks := []struct {
		name string
		args []string
	}{
		{"go vet", []string{"go", "vet", "./..."}},
		{"tests with race detection", []string{"go", "test", "-race", "-short", "./..."}},
		{"dependency management", []string{"go", "mod", "tidy", "-diff"}},
		{"module verification", []string{"go", "mod", "verify"}},
	}

	for _, check := range checks {
		if err := qc.runCommandSilent(check.args[0], check.args[1:]...); err != nil {
			qc.fail(fmt.Sprintf("%s failed", check.name))
		} else {
			qc.success(fmt.Sprintf("%s passed", check.name))
		}
	}
}

func (qc *QualityChecker) runExternalTools() {
	fmt.Println("Running external tools...")

	staticcheckPath := qc.gopath + "/bin/staticcheck"
	if qc.fileExists(staticcheckPath) {
		output, err := qc.runCommand(staticcheckPath, "-checks=all,-SA1019", "./...")
		if err != nil {
			qc.fail("staticcheck found issues:")
			if strings.TrimSpace(output) != "" {
				fmt.Print(output)
			}
		} else {
			qc.success("staticcheck passed")
		}
	} else {
		qc.warn("staticcheck not available")
	}

	govulncheckPath := qc.gopath + "/bin/govulncheck"
	if qc.fileExists(govulncheckPath) {
		output, err := qc.runCommand(govulncheckPath, "./...")
		if err != nil {
			qc.fail("Security vulnerabilities detected:")
			if strings.TrimSpace(output) != "" {
				fmt.Print(output)
			}
		} else {
			qc.success("No security vulnerabilities found")
		}
	} else {
		qc.warn("govulncheck not available")
	}
}

func (qc *QualityChecker) checkBuild() {
	fmt.Println("Verifying build...")

	if err := qc.runCommandSilent("go", "build", "-v", "."); err != nil {
		qc.fail("Build failed")
		return
	}
	qc.success("Build successful")

	if qc.fileExists(ProjectName) {
		os.Remove(ProjectName)
	}
}

func (qc *QualityChecker) success(message string) {
	fmt.Printf("%s✓%s %s\n", ColorGreen, ColorReset, message)
	qc.checksPassed++
}

func (qc *QualityChecker) fail(message string) {
	fmt.Printf("%s✗%s %s\n", ColorRed, ColorReset, message)
	qc.checksFailed++
}

func (qc *QualityChecker) warn(message string) {
	fmt.Printf("%s⚠%s %s\n", ColorYellow, ColorReset, message)
}

func (qc *QualityChecker) generateSummary() {
	fmt.Println("\n==================================")
	fmt.Println("Quality Check Summary")
	fmt.Println("==================================")
	fmt.Printf("Passed: %d\n", qc.checksPassed)
	fmt.Printf("Failed: %d\n\n", qc.checksFailed)

	if qc.checksFailed == 0 {
		fmt.Printf("%sAll quality checks passed%s\n", ColorGreen, ColorReset)
	} else {
		fmt.Printf("%s%d quality checks failed%s\n", ColorRed, qc.checksFailed, ColorReset)
	}
}

func (qc *QualityChecker) fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func (qc *QualityChecker) runCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (qc *QualityChecker) runCommandSilent(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	return cmd.Run()
}

func (qc *QualityChecker) extractModuleName() string {
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 {
		return ""
	}

	parts := strings.Fields(lines[0])
	if len(parts) < 2 || parts[0] != "module" {
		return ""
	}

	modulePath := parts[1]
	pathParts := strings.Split(modulePath, "/")
	return pathParts[len(pathParts)-1]
}
