package docker

import (
	"testing"
)

func TestParseMemoryString(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"", 0},
		{"0", 0},
		{"1024", 1024},
		{"1K", 1024},
		{"1k", 1024},
		{"1M", 1024 * 1024},
		{"1m", 1024 * 1024},
		{"1G", 1024 * 1024 * 1024},
		{"1g", 1024 * 1024 * 1024},
		{"4G", 4 * 1024 * 1024 * 1024},
		{"2048M", 2048 * 1024 * 1024},
		{"512K", 512 * 1024},
		{"1T", 1024 * 1024 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseMemoryString(tt.input)
			if result != tt.expected {
				t.Errorf("parseMemoryString(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildResourceConstraints(t *testing.T) {
	tests := []struct {
		name    string
		memory  string
		cpus    int
		wantMem int64
		wantCPU int64
	}{
		{"empty", "", 0, 0, 0},
		{"memory only", "4G", 0, 4 * 1024 * 1024 * 1024, 0},
		{"cpus only", "", 4, 0, 4e9},
		{"both", "2G", 2, 2 * 1024 * 1024 * 1024, 2e9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resources := buildResourceConstraints(tt.memory, tt.cpus)
			if resources.Memory != tt.wantMem {
				t.Errorf("Memory = %d, want %d", resources.Memory, tt.wantMem)
			}
			if resources.NanoCPUs != tt.wantCPU {
				t.Errorf("NanoCPUs = %d, want %d", resources.NanoCPUs, tt.wantCPU)
			}
		})
	}
}
