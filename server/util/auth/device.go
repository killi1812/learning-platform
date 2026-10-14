package auth

import "strings"

// ParseDeviceName extracts a clean human-readable device name from a User-Agent header.
func ParseDeviceName(userAgent string) string {
	if strings.TrimSpace(userAgent) == "" {
		return "Unknown Device"
	}

	var browser string
	switch {
	case strings.Contains(userAgent, "Edg/"):
		browser = "Microsoft Edge"
	case strings.Contains(userAgent, "Chrome/") && !strings.Contains(userAgent, "Edg/"):
		browser = "Chrome"
	case strings.Contains(userAgent, "Firefox/"):
		browser = "Firefox"
	case strings.Contains(userAgent, "Safari/") && !strings.Contains(userAgent, "Chrome/"):
		browser = "Safari"
	case strings.Contains(userAgent, "PostmanRuntime/"):
		browser = "Postman"
	default:
		browser = "Web Browser"
	}

	var os string
	switch {
	case strings.Contains(userAgent, "iPhone") || strings.Contains(userAgent, "iPad"):
		os = "iOS"
	case strings.Contains(userAgent, "Android"):
		os = "Android"
	case strings.Contains(userAgent, "Windows"):
		os = "Windows"
	case strings.Contains(userAgent, "Macintosh") || strings.Contains(userAgent, "Mac OS X"):
		os = "macOS"
	case strings.Contains(userAgent, "Linux"):
		os = "Linux"
	default:
		os = ""
	}

	if os != "" {
		return browser + " on " + os
	}
	return browser
}
