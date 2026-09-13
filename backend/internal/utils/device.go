package utils

import "strings"

type DeviceInfo struct {
	OS         string
	Browser    string
	DeviceType string
}

// ParseUserAgent extracts basic OS, browser, and device classification from User-Agent string
func ParseUserAgent(ua string) DeviceInfo {
	if ua == "" {
		return DeviceInfo{
			OS:         "Unknown",
			Browser:    "Unknown",
			DeviceType: "Unknown",
		}
	}

	lower := strings.ToLower(ua)

	// Device classification
	deviceType := "Desktop"
	if strings.Contains(lower, "mobile") || strings.Contains(lower, "android") || strings.Contains(lower, "iphone") {
		deviceType = "Mobile"
	} else if strings.Contains(lower, "ipad") || strings.Contains(lower, "tablet") {
		deviceType = "Tablet"
	}

	// OS detection
	os := "Unknown OS"
	switch {
	case strings.Contains(lower, "windows"):
		os = "Windows"
	case strings.Contains(lower, "macintosh") || strings.Contains(lower, "mac os"):
		os = "macOS"
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad"):
		os = "iOS"
	case strings.Contains(lower, "android"):
		os = "Android"
	case strings.Contains(lower, "linux"):
		os = "Linux"
	}

	// Browser detection (order matters due to shared tokens)
	browser := "Unknown Browser"
	switch {
	case strings.Contains(lower, "edg/"):
		browser = "Edge"
	case strings.Contains(lower, "opr/") || strings.Contains(lower, "opera"):
		browser = "Opera"
	case strings.Contains(lower, "chrome/") && !strings.Contains(lower, "edg/"):
		browser = "Chrome"
	case strings.Contains(lower, "safari/") && !strings.Contains(lower, "chrome/"):
		browser = "Safari"
	case strings.Contains(lower, "firefox/"):
		browser = "Firefox"
	}

	return DeviceInfo{
		OS:         os,
		Browser:    browser,
		DeviceType: deviceType,
	}
}
