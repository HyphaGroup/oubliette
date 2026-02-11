package mcp

import (
	"testing"

	"github.com/HyphaGroup/oubliette/internal/auth"
)

func TestIsToolAllowed(t *testing.T) {
	tests := []struct {
		name       string
		tool       ToolDef
		tokenScope string
		projectID  string
		want       bool
	}{
		// Admin scope - full access
		{
			name:       "admin can access admin tool",
			tool:       ToolDef{Name: "token_create", Target: TargetGlobal, Access: AccessAdmin},
			tokenScope: auth.ScopeAdmin,
			want:       true,
		},
		{
			name:       "admin can access global write tool",
			tool:       ToolDef{Name: "project_create", Target: TargetGlobal, Access: AccessWrite},
			tokenScope: auth.ScopeAdmin,
			want:       true,
		},
		{
			name:       "admin can access project read tool",
			tool:       ToolDef{Name: "project_get", Target: TargetProject, Access: AccessRead},
			tokenScope: auth.ScopeAdmin,
			projectID:  "proj-1",
			want:       true,
		},

		// Admin:ro scope - read only
		{
			name:       "admin:ro cannot access admin tool",
			tool:       ToolDef{Name: "token_create", Target: TargetGlobal, Access: AccessAdmin},
			tokenScope: auth.ScopeAdminRO,
			want:       false,
		},
		{
			name:       "admin:ro cannot access global write tool",
			tool:       ToolDef{Name: "project_create", Target: TargetGlobal, Access: AccessWrite},
			tokenScope: auth.ScopeAdminRO,
			want:       false,
		},
		{
			name:       "admin:ro can access global read tool",
			tool:       ToolDef{Name: "project_list", Target: TargetGlobal, Access: AccessRead},
			tokenScope: auth.ScopeAdminRO,
			want:       true,
		},
		{
			name:       "admin:ro can access project read tool",
			tool:       ToolDef{Name: "project_get", Target: TargetProject, Access: AccessRead},
			tokenScope: auth.ScopeAdminRO,
			projectID:  "proj-1",
			want:       true,
		},

		// Project scope - own project only
		{
			name:       "project scope can access own project read",
			tool:       ToolDef{Name: "project_get", Target: TargetProject, Access: AccessRead},
			tokenScope: "project:proj-1",
			projectID:  "proj-1",
			want:       true,
		},
		{
			name:       "project scope can access own project write",
			tool:       ToolDef{Name: "session_spawn", Target: TargetProject, Access: AccessWrite},
			tokenScope: "project:proj-1",
			projectID:  "proj-1",
			want:       true,
		},
		{
			name:       "project scope cannot access other project",
			tool:       ToolDef{Name: "project_get", Target: TargetProject, Access: AccessRead},
			tokenScope: "project:proj-1",
			projectID:  "proj-2",
			want:       false,
		},
		{
			name:       "project scope cannot access global write tool",
			tool:       ToolDef{Name: "project_create", Target: TargetGlobal, Access: AccessWrite},
			tokenScope: "project:proj-1",
			want:       false,
		},
		{
			name:       "project scope can access global read tool",
			tool:       ToolDef{Name: "project_list", Target: TargetGlobal, Access: AccessRead},
			tokenScope: "project:proj-1",
			want:       true,
		},
		{
			name:       "project scope cannot access admin tool",
			tool:       ToolDef{Name: "token_create", Target: TargetGlobal, Access: AccessAdmin},
			tokenScope: "project:proj-1",
			want:       false,
		},

		// Project:ro scope - read only, own project only
		{
			name:       "project:ro can access own project read",
			tool:       ToolDef{Name: "project_get", Target: TargetProject, Access: AccessRead},
			tokenScope: "project:proj-1:ro",
			projectID:  "proj-1",
			want:       true,
		},
		{
			name:       "project:ro cannot access own project write",
			tool:       ToolDef{Name: "session_spawn", Target: TargetProject, Access: AccessWrite},
			tokenScope: "project:proj-1:ro",
			projectID:  "proj-1",
			want:       false,
		},
		{
			name:       "project:ro cannot access other project",
			tool:       ToolDef{Name: "project_get", Target: TargetProject, Access: AccessRead},
			tokenScope: "project:proj-1:ro",
			projectID:  "proj-2",
			want:       false,
		},

		// Legacy read-only scope (treated as admin:ro)
		{
			name:       "read-only can access global read tool",
			tool:       ToolDef{Name: "project_list", Target: TargetGlobal, Access: AccessRead},
			tokenScope: auth.ScopeAdminRO,
			want:       true,
		},
		{
			name:       "read-only cannot access global write tool",
			tool:       ToolDef{Name: "project_create", Target: TargetGlobal, Access: AccessWrite},
			tokenScope: auth.ScopeAdminRO,
			want:       false,
		},

		// Edge cases
		{
			name:       "project scope with empty projectID denied",
			tool:       ToolDef{Name: "project_get", Target: TargetProject, Access: AccessRead},
			tokenScope: "project:proj-1",
			projectID:  "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsToolAllowed(&tt.tool, tt.tokenScope, tt.projectID)
			if got != tt.want {
				t.Errorf("IsToolAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractProjectIDFromArgs(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
		want string
	}{
		{
			name: "has project_id",
			args: map[string]any{"project_id": "proj-123"},
			want: "proj-123",
		},
		{
			name: "empty project_id",
			args: map[string]any{"project_id": ""},
			want: "",
		},
		{
			name: "no project_id",
			args: map[string]any{"other": "value"},
			want: "",
		},
		{
			name: "nil args",
			args: nil,
			want: "",
		},
		{
			name: "project_id wrong type",
			args: map[string]any{"project_id": 123},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractProjectIDFromArgs(tt.args)
			if got != tt.want {
				t.Errorf("ExtractProjectIDFromArgs() = %q, want %q", got, tt.want)
			}
		})
	}
}
