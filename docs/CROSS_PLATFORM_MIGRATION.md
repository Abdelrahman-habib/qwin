# Cross-Platform Migration Guide

This document outlines how to migrate qwin from Windows-only to cross-platform support.

## Current State

- ✅ Windows implementation complete (`internal/platform/windows_api.go`)
- ✅ Platform abstraction layer ready (`internal/platform/interface.go`)
- ✅ Build tags added for platform separation
- ✅ Stub implementations created for Linux and macOS
- ✅ Factory pattern implemented for automatic platform detection

## Migration Steps

### Phase 1: Linux Support

#### 1.1 Implement Linux APIs

Replace stub implementation in `internal/platform/linux_api.go`:

**Option A: X11 (Traditional)**

```go
// Use X11 libraries
import "github.com/BurntSushi/xgb/xproto"
// Get active window, process info, icons
```

**Option B: Wayland (Modern)**

```go
// Use Wayland protocols
// wlr-foreign-toplevel-management for window info
```

**Option C: Mixed Approach**

```go
// Detect X11 vs Wayland and use appropriate APIs
```

#### 1.2 Add Linux Dependencies

Update `go.mod`:

```go
// Add Linux-specific dependencies
github.com/BurntSushi/xgb v0.0.0-20210121224620-deaf085860bc
// or other X11/Wayland libraries
```

#### 1.3 Update Wails Configuration

Update `wails.json` for Linux builds:

```json
{
  "linux": {
    "icon": "build/appicon.png",
    "deb": {
      "depends": ["libgtk-3-0", "libwebkit2gtk-4.0-37"]
    }
  }
}
```

### Phase 2: macOS Support

#### 2.1 Implement macOS APIs

Replace stub implementation in `internal/platform/darwin_api.go`:

```go
/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices
#include <Cocoa/Cocoa.h>
#include <ApplicationServices/ApplicationServices.h>

// C helper functions for Objective-C calls
*/
import "C"

// Implement using CGO and Objective-C
```

#### 2.2 Add macOS Dependencies

```go
// CGO-based implementation, no additional Go dependencies needed
// But requires macOS SDK and Xcode tools
```

### Phase 3: Update CI/CD

#### 3.1 Replace Build Workflow

1. Rename current `build.yml` to `build-windows-only.yml.backup`
2. Rename `build-multiplatform.yml.template` to `build.yml`
3. Test all platforms

#### 3.2 Update CodeQL Workflow

The current CodeQL workflow should work as-is once platform abstraction is complete.

#### 3.3 Update Security Workflow

No changes needed - will work across platforms.

## Implementation Priority

### High Priority (Core Functionality)

1. **Active window detection** - Essential for screen time tracking
2. **Process name extraction** - Required for app identification
3. **Basic icon extraction** - For UI display

### Medium Priority (Enhanced Features)

1. **High-quality icon extraction** - Better UI experience
2. **Window title extraction** - More detailed tracking
3. **Process path detection** - Security and categorization

### Low Priority (Nice to Have)

1. **Window position/size** - Advanced analytics
2. **Multiple monitor support** - Complex setups
3. **Accessibility integration** - Enhanced tracking

## Testing Strategy

### Local Testing

```bash
# Test Windows build
GOOS=windows go build ./...

# Test Linux build (requires Linux or WSL)
GOOS=linux go build ./...

# Test macOS build (requires macOS)
GOOS=darwin go build ./...
```

### CI Testing

The multi-platform workflow will test all platforms automatically.

## Rollback Plan

If cross-platform support causes issues:

1. **Revert to Windows-only**:

   ```bash
   git checkout HEAD~1 -- .github/workflows/build.yml
   ```

2. **Remove platform files**:

   ```bash
   rm internal/platform/linux_api.go
   rm internal/platform/darwin_api.go
   rm internal/platform/factory.go
   ```

3. **Remove build tags**:
   Remove `//go:build windows` from `windows_api.go`

## Resources

### Linux Development

- [X11 Programming](https://tronche.com/gui/x/xlib/)
- [Wayland Protocols](https://wayland.freedesktop.org/docs/html/)
- [Go X11 Bindings](https://github.com/BurntSushi/xgb)

### macOS Development

- [Cocoa Documentation](https://developer.apple.com/documentation/cocoa)
- [CGO with Objective-C](https://pkg.go.dev/cmd/cgo)
- [ApplicationServices Framework](https://developer.apple.com/documentation/applicationservices)

### Cross-Platform Wails

- [Wails Cross-Platform Guide](https://wails.io/docs/guides/cross-platform)
- [Platform-Specific Builds](https://wails.io/docs/reference/cli#build)

## Current Status: Ready for Migration ✅

Your codebase is now prepared for cross-platform migration. The Windows implementation will continue working unchanged, and you can implement other platforms incrementally.
