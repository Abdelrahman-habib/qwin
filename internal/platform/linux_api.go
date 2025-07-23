//go:build linux

package platform

// LinuxAPI implements WindowAPI for Linux platform
type LinuxAPI struct{}

// NewLinuxAPI creates a new Linux API instance
func NewLinuxAPI() *LinuxAPI {
	return &LinuxAPI{}
}

// NewWindowAPI creates a new WindowAPI instance for Linux
func NewWindowAPI() WindowAPI {
	return NewLinuxAPI()
}

// GetCurrentAppName gets the name of the currently active application on Linux
func (l *LinuxAPI) GetCurrentAppName() string {
	// TODO: Implement using X11/Wayland APIs
	// For now, return placeholder
	return "linux-app-placeholder"
}

// GetCurrentAppInfo gets detailed information about the currently active application on Linux
func (l *LinuxAPI) GetCurrentAppInfo() *AppInfo {
	// TODO: Implement using X11/Wayland APIs
	// Possible approaches:
	// - Use X11: XGetInputFocus, XGetWindowProperty
	// - Use Wayland: wlr-foreign-toplevel-management protocol
	// - Parse /proc filesystem for process info

	return &AppInfo{
		Name:     "linux-app-placeholder",
		IconPath: "",
		ExePath:  "/usr/bin/placeholder",
	}
}
