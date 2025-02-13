package solver

import (

	"net/url"
	"testing"
)

func TestTokenIdURLEscapingWithQueryEscape(t *testing.T) {
	tests := []struct {
		name     string
		tokenId  string
		expected string
	}{
		{
			name:     "Complex token ID with forward slash",
			tokenId:  "M0Elwz5/odEcC2fYQJ750BjcKKhjrQTyDtjTpnZOaQY",
			expected: "M0Elwz5/odEcC2fYQJ750BjcKKhjrQTyDtjTpnZOaQY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			escaped := url.QueryEscape(tt.tokenId)
			if escaped != tt.expected {
				t.Errorf("url.QueryEscape(%q) = %q, want %q", tt.tokenId, escaped, tt.expected)
			}

			// Verify we can unescape back to original
			unescaped, err := url.QueryUnescape(escaped)
			if err != nil {
				t.Errorf("Failed to unescape: %v", err)
			}
			if unescaped != tt.tokenId {
				t.Errorf("url.QueryUnescape(%q) = %q, want %q", escaped, unescaped, tt.tokenId)
			}
		})
	}
}
