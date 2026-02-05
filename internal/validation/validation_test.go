package validation

import (
	"testing"
)

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid UUID", "550e8400-e29b-41d4-a716-446655440000", false},
		{"valid UUID uppercase", "550E8400-E29B-41D4-A716-446655440000", false},
		{"empty", "", true},
		{"not a UUID", "not-a-uuid", true},
		{"path traversal attempt", "../../../etc/passwd", true},
		{"SQL injection attempt", "'; DROP TABLE projects; --", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUUID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUUID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateProjectID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid UUID", "550e8400-e29b-41d4-a716-446655440000", false},
		{"empty", "", true},
		{"invalid format", "not-a-uuid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProjectID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProjectID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSessionID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		// UUID format
		{"valid UUID session", "550e8400-e29b-41d4-a716-446655440000", false},
		// Child session format
		{"valid child session", "child_550e8400-e29b-41d4-a716-446655440000_1", false},
		{"valid nested child", "child_child_550e8400_2_3", false},
		// Gogol session format (gogol_YYYYMMDD_HHMMSS_HEXRANDOM)
		{"valid gogol session", "gogol_20250101_120000_abc12345", false},
		{"valid gogol short hex", "gogol_20250101_120000_a1", false},
		{"valid gogol long hex", "gogol_20250101_120000_abcdef0123456789", false},
		{"valid gogol uppercase hex", "gogol_20250101_120000_ABCDEF01", false},
		// Invalid formats
		{"empty", "", true},
		{"invalid child format", "child_", true},
		{"invalid child format two parts", "child_x", true},
		{"not a valid ID", "not-valid", true},
		{"gogol wrong date format", "gogol_2025-01-01_120000_abc123", true},
		{"gogol wrong time format", "gogol_20250101_12:00:00_abc123", true},
		{"gogol non-hex random", "gogol_20250101_120000_ghijkl", true},
		{"gogol missing underscore", "gogol20250101_120000_abc123", true},
		{"gogol missing random", "gogol_20250101_120000_", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSessionID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSessionID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateWorkspaceID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid UUID", "550e8400-e29b-41d4-a716-446655440000", false},
		{"empty", "", true},
		{"invalid format", "not-a-uuid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkspaceID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWorkspaceID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{"simple path", "foo/bar", "foo/bar", false},
		{"single component", "filename.txt", "filename.txt", false},
		{"with underscore", "my_file.txt", "my_file.txt", false},
		{"with dash", "my-file.txt", "my-file.txt", false},
		{"trailing slash", "foo/bar/", "foo/bar/", false},
		{"empty", "", "", true},
		{"path traversal", "../../../etc/passwd", "", true},
		{"path traversal in middle", "foo/../../../etc/passwd", "", true},
		{"absolute path", "/etc/passwd", "", true},
		{"unsafe chars semicolon", "foo;rm -rf /", "", true},
		{"unsafe chars space", "foo bar", "", true},
		{"unsafe chars ampersand", "foo&bar", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SanitizePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("SanitizePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SanitizePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateContainerID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid short ID", "abc123def456", false},
		{"valid long ID", "abc123def456abc123def456abc123def456abc123def456abc123def456abc1", false},
		{"valid uppercase", "ABC123DEF456", false},
		{"empty", "", true},
		{"too short", "abc123", true},
		{"too long", "abc123def456abc123def456abc123def456abc123def456abc123def456abc12345", true},
		{"invalid chars", "abc123def456xyz!", true},
		{"invalid chars space", "abc123 def456", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContainerID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContainerID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
