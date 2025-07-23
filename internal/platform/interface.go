package platform

// WindowAPI defines the interface for platform-specific window operations
type WindowAPI interface {
	GetCurrentAppName() string
	GetCurrentAppInfo() *AppInfo
}

// AppInfo contains information about an application
type AppInfo struct {
	Name     string `json:"name"`
	IconPath string `json:"iconPath"`
	ExePath  string `json:"exePath"`
}
