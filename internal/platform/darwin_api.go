//go:build darwin

package platform

// DarwinAPI implements WindowAPI for macOS platform
type DarwinAPI struct{}

// NewDarwinAPI creates a new macOS API instance
func NewDarwinAPI() *DarwinAPI {
	return &DarwinAPI{}
}

// NewWindowAPI creates a new WindowAPI instance for macOS
func NewWindowAPI() WindowAPI {
	return NewDarwinAPI()
}

// GetCurrentAppName gets the name of the currently active application on macOS
func (d *DarwinAPI) GetCurrentAppName() string {
	// TODO: Implement using Cocoa/AppKit APIs
	// For now, return placeholder
	return "macos-app-placeholder"
}

// GetCurrentAppInfo gets detailed information about the currently active application on macOS
func (d *DarwinAPI) GetCurrentAppInfo() *AppInfo {
	// TODO: Implement using Cocoa/AppKit APIs
	// Possible approaches:
	// - Use NSWorkspace.sharedWorkspace.frontmostApplication
	// - Use CGWindowListCopyWindowInfo
	// - Use Accessibility APIs

	return &AppInfo{
		Name:     "macos-app-placeholder",
		IconPath: "",
		ExePath:  "/Applications/Placeholder.app",
	}
}
