# macOS App Packaging Guide

This solution creates a native macOS `.app` bundle and distributable `.dmg` file from your Go binary.

## Quick Start

```bash
# 1. Setup packaging (one-time)
mkdir -p cmd/package
# Copy the Go packager code to cmd/package/main.go
# Copy the shell script as package.sh

# 2. Make script executable
chmod +x package.sh

# 3. Build your app first
./build.sh build

# 4. Package the app
./package.sh package
```

## File Structure

```
otsu-obliterator/
├── cmd/
│   └── package/
│       └── main.go          # Go packaging tool
├── build/
│   └── otsu-obliterator     # Your built binary
├── dist/                    # Output directory (created)
│   ├── Otsu Obliterator.app # macOS app bundle
│   └── Otsu-Obliterator.dmg # Distribution disk image
├── icon.png                 # App icon (1024x1024 recommended)
├── package.sh              # Packaging script
└── ...
```

## Requirements

- **macOS**: Required for `hdiutil`, `iconutil`, `sips`
- **Go 1.24+**: For building the packaging tool
- **Xcode Command Line Tools**: `xcode-select --install`

## Commands

### Package Application
```bash
./package.sh package                    # Use build/otsu-obliterator
./package.sh package build/debug-build  # Use specific binary
```

### Clean Artifacts
```bash
./package.sh clean
```

### Verify App Bundle
```bash
./package.sh verify "dist/Otsu Obliterator.app"
```

## Icon Requirements

Place a PNG icon as `icon.png` in your project root:
- **Recommended**: 1024x1024 pixels
- **Format**: PNG with transparency
- **Quality**: High resolution for Retina displays

The packager automatically creates all required icon sizes (16x16 to 1024x1024) and converts to `.icns` format.

## Output Files

### App Bundle Structure
```
Otsu Obliterator.app/
└── Contents/
    ├── Info.plist           # App metadata
    ├── MacOS/
    │   └── otsu-obliterator # Your executable
    └── Resources/
        └── Otsu Obliterator.icns # App icon
```

### DMG Contents
- Compressed disk image with your `.app` bundle
- Optimized for distribution
- Double-click to mount and drag to Applications

## Integration with Build System

Add to your `build.sh`:

```bash
package() {
    build "macos"  # or your preferred target
    ./package.sh package
}
```

## Info.plist Configuration

The packager creates a complete `Info.plist` with:
- **Bundle ID**: `com.imageprocessing.otsu-obliterator`
- **App Category**: Graphics & Design
- **System Requirements**: macOS 10.13+
- **Retina Support**: Enabled
- **Executable Permissions**: Properly set

## Troubleshooting

### "Binary not found"
```bash
./build.sh build  # Build binary first
```

### "sips command not found"
```bash
xcode-select --install
```

### "Icon conversion failed"
Ensure `icon.png` exists and is a valid PNG file. The packager will continue without an icon if this fails.

### "DMG creation failed"
Check disk space and ensure no existing DMG is mounted.

## Advanced Usage

### Custom Binary Location
```bash
./package.sh package /path/to/custom/binary
```

### Verify Before Distribution
```bash
./package.sh verify "dist/Otsu Obliterator.app"
```

### Manual Steps (if needed)
```bash
# Build binary
go build -o build/otsu-obliterator .

# Package
go run cmd/package/main.go package build/otsu-obliterator

# Verify
go run cmd/package/main.go verify "dist/Otsu Obliterator.app"
```

## Distribution

The created `.dmg` file is ready for distribution:
1. Upload to your website
2. Share via email/cloud storage  
3. Users double-click to mount
4. Drag app to Applications folder

No code signing required for basic distribution, though users may see security warnings on first launch.