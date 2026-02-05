package droid

import (
	"fmt"
	"strings"

	"github.com/HyphaGroup/oubliette/internal/agent"
)

func (r *Runtime) buildCommand(req *agent.ExecuteRequest) string {
	parts := []string{"droid", "exec"}

	// For stream-jsonrpc mode, don't pass prompt as argument - it's sent via stdin
	if !req.StreamJSONRPC {
		// Prompt (with system prompt prepended if provided)
		prompt := req.Prompt
		if req.SystemPrompt != "" {
			prompt = fmt.Sprintf("%s\n\n---\n\n%s", req.SystemPrompt, req.Prompt)
		}
		parts = append(parts, shellEscape(prompt))
	}

	// Model selection
	model := req.Model
	if model == "" {
		model = r.defaultModel
	}
	parts = append(parts, "-m", model)

	// Autonomy level
	autonomy := req.AutonomyLevel
	if autonomy == "" {
		autonomy = r.defaultAutonomy
	}

	// Handle skip-permissions-unsafe as special flag
	if autonomy == "skip-permissions-unsafe" {
		parts = append(parts, "--skip-permissions-unsafe")
	} else if autonomy != "read-only" && autonomy != "" {
		parts = append(parts, "--auto", autonomy)
	}

	// Reasoning effort
	if req.ReasoningLevel != "" && req.ReasoningLevel != "off" {
		parts = append(parts, "-r", req.ReasoningLevel)
	}

	// Session continuation
	if req.SessionID != "" {
		parts = append(parts, "-s", req.SessionID)
	}

	// Output format
	if req.StreamJSONRPC {
		// Bidirectional streaming - input and output via JSON-RPC
		parts = append(parts, "-o", "stream-jsonrpc", "--input-format", "stream-jsonrpc")
	} else {
		// Single-turn execution with JSON output
		parts = append(parts, "-o", "json")
	}

	// Working directory
	if req.WorkingDir != "" {
		parts = append(parts, "--cwd", req.WorkingDir)
	}

	// Tool filtering
	if len(req.EnabledTools) > 0 {
		parts = append(parts, "--enabled-tools", strings.Join(req.EnabledTools, ","))
	}
	if len(req.DisabledTools) > 0 {
		parts = append(parts, "--disabled-tools", strings.Join(req.DisabledTools, ","))
	}

	// Spec mode
	if req.UseSpec {
		parts = append(parts, "--use-spec")
	}

	return strings.Join(parts, " ")
}

func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
