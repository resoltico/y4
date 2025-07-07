package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
)

const (
	AppName       = "Otsu Obliterator"
	AppID         = "com.imageprocessing.otsu-obliterator"
	AppVersion    = "1.0.0"
	AppExecutable = "otsu-obliterator"
	DeveloperName = "Ervins Strauhmanis"
	Copyright     = "© 2025 Ervins Strauhmanis"
)

type PackageConfig struct {
	AppName       string
	AppID         string
	AppVersion    string
	AppExecutable string
	DeveloperName string
	Copyright     string
	SourceBinary  string
	IconPath      string
	OutputDir     string
	AppDir        string
	DMGPath       string
	MinVersion    string
}

type PackageStats struct {
	BinarySize  int64
	AppSize     int64
	DMGSize     int64
	ProcessTime time.Duration
}

func main() {
	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "package":
		handlePackage()
	case "clean":
		handleClean()
	case "verify":
		handleVerify()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		showUsage()
		os.Exit(1)
	}
}

func showUsage() {
	fmt.Printf(`macOS App Packager for %s

Usage: go run cmd/package/main.go [COMMAND] [OPTIONS]

COMMANDS:
  package [binary_path]    Create .app bundle and .dmg (default: build/otsu-obliterator)
  clean                    Remove all packaging artifacts  
  verify [app_path]        Verify .app bundle structure

EXAMPLES:
  go run cmd/package/main.go package                          # Package default binary
  go run cmd/package/main.go package build/otsu-obliterator  # Package specific binary
  go run cmd/package/main.go verify dist/Otsu\ Obliterator.app
  go run cmd/package/main.go clean

OUTPUT:
  dist/Otsu Obliterator.app     - macOS application bundle
  dist/Otsu-Obliterator.dmg     - Disk image for distribution
`, AppName)
}

func handlePackage() {
	binaryPath := "build/otsu-obliterator"
	if len(os.Args) > 2 {
		binaryPath = os.Args[2]
	}

	config := &PackageConfig{
		AppName:       AppName,
		AppID:         AppID,
		AppVersion:    AppVersion,
		AppExecutable: AppExecutable,
		DeveloperName: DeveloperName,
		Copyright:     Copyright,
		SourceBinary:  binaryPath,
		IconPath:      "icon.png",
		OutputDir:     "dist",
	}

	if err := packageApp(config); err != nil {
		fmt.Printf("❌ Packaging failed: %v\n", err)
		os.Exit(1)
	}
}

func handleClean() {
	dirs := []string{"dist", "tmp/packaging"}
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			fmt.Printf("⚠️  Failed to remove %s: %v\n", dir, err)
		} else {
			fmt.Printf("🗑️  Removed %s\n", dir)
		}
	}
	fmt.Println("✅ Cleanup complete")
}

func handleVerify() {
	appPath := "dist/Otsu Obliterator.app"
	if len(os.Args) > 2 {
		appPath = os.Args[2]
	}

	if err := verifyAppBundle(appPath); err != nil {
		fmt.Printf("❌ Verification failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ %s verified successfully\n", appPath)
}

func packageApp(config *PackageConfig) error {
	startTime := time.Now()
	stats := &PackageStats{}

	fmt.Printf("📦 Packaging %s v%s\n", config.AppName, config.AppVersion)

	// Validate source binary
	if err := validateBinary(config.SourceBinary); err != nil {
		return fmt.Errorf("binary validation: %w", err)
	}

	binaryInfo, err := os.Stat(config.SourceBinary)
	if err != nil {
		return err
	}
	stats.BinarySize = binaryInfo.Size()

	// Detect minimum macOS version
	minVersion, err := detectMinimumVersion(config.SourceBinary)
	if err != nil {
		fmt.Printf("⚠️  Could not detect minimum version, using 10.15: %v\n", err)
		minVersion = "10.15"
	} else {
		fmt.Printf("📋 Detected minimum macOS version: %s\n", minVersion)
	}
	config.MinVersion = minVersion

	// Setup paths
	config.AppDir = filepath.Join(config.OutputDir, config.AppName+".app")
	config.DMGPath = filepath.Join(config.OutputDir, strings.ReplaceAll(config.AppName, " ", "-")+".dmg")

	// Create directory structure
	if err := createDirectoryStructure(config); err != nil {
		return fmt.Errorf("directory creation: %w", err)
	}

	// Create Info.plist
	if err := createInfoPlist(config); err != nil {
		return fmt.Errorf("Info.plist creation: %w", err)
	}

	// Copy binary
	if err := copyBinary(config); err != nil {
		return fmt.Errorf("binary copy: %w", err)
	}

	// Create icon
	if err := createIcon(config); err != nil {
		fmt.Printf("⚠️  Icon creation failed (non-fatal): %v\n", err)
	}

	// Set permissions
	if err := setPermissions(config); err != nil {
		return fmt.Errorf("permissions: %w", err)
	}

	// Calculate app size
	if appSize, err := calculateDirectorySize(config.AppDir); err == nil {
		stats.AppSize = appSize
	}

	// Create DMG
	if err := createDMG(config); err != nil {
		return fmt.Errorf("DMG creation: %w", err)
	}

	// Calculate DMG size
	if dmgInfo, err := os.Stat(config.DMGPath); err == nil {
		stats.DMGSize = dmgInfo.Size()
	}

	stats.ProcessTime = time.Since(startTime)
	printStats(stats)

	fmt.Printf("✅ Package created successfully:\n")
	fmt.Printf("   📱 App Bundle: %s\n", config.AppDir)
	fmt.Printf("   💿 DMG: %s\n", config.DMGPath)

	return nil
}

func validateBinary(binaryPath string) error {
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("binary not found: %s", binaryPath)
	}

	// Check if it's executable
	file, err := os.Open(binaryPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read first 4 bytes to check for Mach-O header
	header := make([]byte, 4)
	if _, err := file.Read(header); err != nil {
		return err
	}

	// Mach-O magic numbers
	isMachO := (header[0] == 0xfe && header[1] == 0xed && header[2] == 0xfa && header[3] == 0xce) || // 32-bit
		(header[0] == 0xfe && header[1] == 0xed && header[2] == 0xfa && header[3] == 0xcf) || // 64-bit
		(header[0] == 0xcf && header[1] == 0xfa && header[2] == 0xed && header[3] == 0xfe) || // 64-bit reverse
		(header[0] == 0xce && header[1] == 0xfa && header[2] == 0xed && header[3] == 0xfe) // 32-bit reverse

	if !isMachO {
		return fmt.Errorf("binary is not a valid Mach-O executable")
	}

	return nil
}

func detectMinimumVersion(binaryPath string) (string, error) {
	cmd := exec.Command("otool", "-l", binaryPath)
	output, err := cmd.Output()
	if err != nil {
		return "10.15", err
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if strings.Contains(line, "LC_BUILD_VERSION") {
			// Look for minos line in next few lines
			for j := i + 1; j < i+10 && j < len(lines); j++ {
				if strings.Contains(lines[j], "minos") {
					fields := strings.Fields(strings.TrimSpace(lines[j]))
					if len(fields) >= 2 {
						version := fields[1]
						// Convert major.minor format (e.g., "15.0" -> "15.0")
						return formatMacOSVersion(version), nil
					}
				}
			}
		}
		// Fallback: check for older LC_VERSION_MIN_MACOSX
		if strings.Contains(line, "LC_VERSION_MIN_MACOSX") && i+2 < len(lines) {
			versionLine := strings.TrimSpace(lines[i+2])
			if strings.HasPrefix(versionLine, "version") {
				fields := strings.Fields(versionLine)
				if len(fields) >= 2 {
					return formatMacOSVersion(fields[1]), nil
				}
			}
		}
	}

	return "10.15", fmt.Errorf("no version info found")
}

func formatMacOSVersion(version string) string {
	// Handle different version formats
	re := regexp.MustCompile(`(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(version)
	if len(matches) >= 3 {
		major := matches[1]
		minor := matches[2]

		// Convert new versioning (15.0 = macOS 15.0) vs old (10.15 = macOS 10.15)
		if major == "10" {
			return fmt.Sprintf("%s.%s", major, minor)
		} else {
			return fmt.Sprintf("%s.0", major)
		}
	}
	return version
}

func createDirectoryStructure(config *PackageConfig) error {
	dirs := []string{
		config.OutputDir,
		config.AppDir,
		filepath.Join(config.AppDir, "Contents"),
		filepath.Join(config.AppDir, "Contents", "MacOS"),
		filepath.Join(config.AppDir, "Contents", "Resources"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func createInfoPlist(config *PackageConfig) error {
	const infoPlistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundleDevelopmentRegion</key>
	<string>en</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleIdentifier</key>
	<string>{{.AppID}}</string>
	<key>CFBundleExecutable</key>
	<string>{{.AppExecutable}}</string>
	<key>CFBundleIconFile</key>
	<string>{{.AppName}}.icns</string>
	<key>CFBundleDisplayName</key>
	<string>{{.AppName}}</string>
	<key>CFBundleName</key>
	<string>{{.AppName}}</string>
	<key>CFBundleVersion</key>
	<string>{{.AppVersion}}</string>
	<key>CFBundleShortVersionString</key>
	<string>{{.AppVersion}}</string>
	<key>CFBundleSignature</key>
	<string>????</string>
	<key>NSHumanReadableCopyright</key>
	<string>{{.Copyright}}</string>
	<key>LSMinimumSystemVersion</key>
	<string>{{.MinVersion}}</string>
	<key>NSHighResolutionCapable</key>
	<true/>
	<key>LSApplicationCategoryType</key>
	<string>public.app-category.graphics-design</string>
</dict>
</plist>`

	tmpl, err := template.New("infoplist").Parse(infoPlistTemplate)
	if err != nil {
		return err
	}

	plistPath := filepath.Join(config.AppDir, "Contents", "Info.plist")
	file, err := os.Create(plistPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, config)
}

func copyBinary(config *PackageConfig) error {
	srcFile, err := os.Open(config.SourceBinary)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstPath := filepath.Join(config.AppDir, "Contents", "MacOS", config.AppExecutable)
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func createIcon(config *PackageConfig) error {
	iconSrcPath := config.IconPath
	if _, err := os.Stat(iconSrcPath); os.IsNotExist(err) {
		return fmt.Errorf("icon file not found: %s", iconSrcPath)
	}

	iconsetDir := filepath.Join(config.OutputDir, "tmp.iconset")
	if err := os.MkdirAll(iconsetDir, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(iconsetDir)

	// Create multiple icon sizes using sips
	sizes := []struct {
		size int
		name string
	}{
		{16, "icon_16x16.png"},
		{32, "icon_16x16@2x.png"},
		{32, "icon_32x32.png"},
		{64, "icon_32x32@2x.png"},
		{128, "icon_128x128.png"},
		{256, "icon_128x128@2x.png"},
		{256, "icon_256x256.png"},
		{512, "icon_256x256@2x.png"},
		{512, "icon_512x512.png"},
		{1024, "icon_512x512@2x.png"},
	}

	for _, s := range sizes {
		outputPath := filepath.Join(iconsetDir, s.name)
		cmd := exec.Command("sips", "-z", fmt.Sprintf("%d", s.size), fmt.Sprintf("%d", s.size), iconSrcPath, "--out", outputPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create icon %s: %w", s.name, err)
		}
	}

	// Convert iconset to icns
	icnsPath := filepath.Join(config.AppDir, "Contents", "Resources", config.AppName+".icns")
	cmd := exec.Command("iconutil", "-c", "icns", iconsetDir, "-o", icnsPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create icns file: %w", err)
	}

	return nil
}

func setPermissions(config *PackageConfig) error {
	binaryPath := filepath.Join(config.AppDir, "Contents", "MacOS", config.AppExecutable)
	return os.Chmod(binaryPath, 0755)
}

func createDMG(config *PackageConfig) error {
	// Remove existing DMG
	os.Remove(config.DMGPath)

	cmd := exec.Command("hdiutil", "create",
		"-volname", config.AppName,
		"-srcfolder", config.AppDir,
		"-ov",
		"-format", "UDZO",
		config.DMGPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hdiutil failed: %w\nOutput: %s", err, output)
	}

	return nil
}

func calculateDirectorySize(dirPath string) (int64, error) {
	var size int64
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func printStats(stats *PackageStats) {
	fmt.Printf("\n📊 Package Statistics:\n")
	fmt.Printf("   Binary Size: %s\n", formatBytes(stats.BinarySize))
	if stats.AppSize > 0 {
		fmt.Printf("   App Bundle: %s\n", formatBytes(stats.AppSize))
	}
	if stats.DMGSize > 0 {
		fmt.Printf("   DMG Size: %s\n", formatBytes(stats.DMGSize))
	}
	fmt.Printf("   Build Time: %v\n", stats.ProcessTime.Round(time.Millisecond))
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func verifyAppBundle(appPath string) error {
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		return fmt.Errorf("app bundle not found: %s", appPath)
	}

	// Check required files
	requiredFiles := []string{
		filepath.Join(appPath, "Contents", "Info.plist"),
		filepath.Join(appPath, "Contents", "MacOS", AppExecutable),
	}

	for _, file := range requiredFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return fmt.Errorf("required file missing: %s", file)
		}
	}

	// Check executable permissions
	execPath := filepath.Join(appPath, "Contents", "MacOS", AppExecutable)
	info, err := os.Stat(execPath)
	if err != nil {
		return err
	}

	if info.Mode()&0111 == 0 {
		return fmt.Errorf("executable lacks execute permissions: %s", execPath)
	}

	fmt.Printf("✅ App bundle structure is valid\n")
	fmt.Printf("   Info.plist: ✓\n")
	fmt.Printf("   Executable: ✓\n")
	fmt.Printf("   Permissions: ✓\n")

	return nil
}
