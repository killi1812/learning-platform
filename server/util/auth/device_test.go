package auth_test

import (
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/auth"
	"github.com/stretchr/testify/assert"
)

func TestParseDeviceName(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		expected  string
	}{
		{
			name:      "Empty User-Agent",
			userAgent: "",
			expected:  "Unknown Device",
		},
		{
			name:      "Whitespace only",
			userAgent: "   ",
			expected:  "Unknown Device",
		},
		{
			name:      "Chrome on Linux",
			userAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			expected:  "Chrome on Linux",
		},
		{
			name:      "Firefox on Windows",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/121.0",
			expected:  "Firefox on Windows",
		},
		{
			name:      "Safari on macOS",
			userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
			expected:  "Safari on macOS",
		},
		{
			name:      "Edge on Windows",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
			expected:  "Microsoft Edge on Windows",
		},
		{
			name:      "Chrome on Android",
			userAgent: "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
			expected:  "Chrome on Android",
		},
		{
			name:      "Safari on iOS",
			userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1",
			expected:  "Safari on iOS",
		},
		{
			name:      "Postman",
			userAgent: "PostmanRuntime/7.36.0",
			expected:  "Postman",
		},
		{
			name:      "Unknown curl",
			userAgent: "curl/7.88.1",
			expected:  "Web Browser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := auth.ParseDeviceName(tt.userAgent)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
