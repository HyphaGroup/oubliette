package agent

import (
	"testing"
)

func TestDetectRuntimeType(t *testing.T) {
	tests := []struct {
		name          string
		factoryAPIKey string
		expected      RuntimeType
	}{
		{
			name:          "with factory API key returns droid",
			factoryAPIKey: "fk-test123",
			expected:      RuntimeTypeDroid,
		},
		{
			name:          "without factory API key returns opencode",
			factoryAPIKey: "",
			expected:      RuntimeTypeOpenCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectRuntimeType(tt.factoryAPIKey)
			if result != tt.expected {
				t.Errorf("DetectRuntimeType(%q) = %q, want %q", tt.factoryAPIKey, result, tt.expected)
			}
		})
	}
}

func TestRuntimeTypeConstants(t *testing.T) {
	// Verify runtime type constants have expected values
	if RuntimeTypeDroid != "droid" {
		t.Errorf("RuntimeTypeDroid = %q, want 'droid'", RuntimeTypeDroid)
	}
	if RuntimeTypeOpenCode != "opencode" {
		t.Errorf("RuntimeTypeOpenCode = %q, want 'opencode'", RuntimeTypeOpenCode)
	}
	if RuntimeTypeAuto != "auto" {
		t.Errorf("RuntimeTypeAuto = %q, want 'auto'", RuntimeTypeAuto)
	}
}
