package types

// AppUsage represents usage data for a single application
type AppUsage struct {
	Name     string `json:"name"`
	Duration int64  `json:"duration"` // in seconds
	IconPath string `json:"iconPath"`
	ExePath  string `json:"exePath"`
}

// UsageData represents the complete usage data
type UsageData struct {
	TotalTime int64      `json:"totalTime"` // in seconds
	Apps      []AppUsage `json:"apps"`
}
