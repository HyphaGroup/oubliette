package droid

import (
	"encoding/json"
	"fmt"

	"github.com/HyphaGroup/oubliette/internal/agent"
)

func (r *Runtime) parseOutput(raw []byte) (*agent.ExecuteResponse, error) {
	// Strip any leading control characters (bell, etc.) from Docker output
	for len(raw) > 0 && raw[0] < 32 && raw[0] != '\n' && raw[0] != '\t' {
		raw = raw[1:]
	}

	var output struct {
		Type       string `json:"type"`
		Subtype    string `json:"subtype"`
		IsError    bool   `json:"is_error"`
		DurationMs int    `json:"duration_ms"`
		NumTurns   int    `json:"num_turns"`
		Result     string `json:"result"`
		SessionID  string `json:"session_id"`
	}

	if err := json.Unmarshal(raw, &output); err != nil {
		return nil, fmt.Errorf("failed to parse Droid output: %w", err)
	}

	if output.IsError {
		return nil, fmt.Errorf("droid execution error: %s", output.Result)
	}

	return &agent.ExecuteResponse{
		SessionID:  output.SessionID,
		Result:     output.Result,
		DurationMs: output.DurationMs,
		NumTurns:   output.NumTurns,
	}, nil
}
