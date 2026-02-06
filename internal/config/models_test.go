package config

import (
	"encoding/json"
	"testing"
)

func TestModelRegistry_GetModel(t *testing.T) {
	registry := &ModelRegistry{
		Models: map[string]ModelDefinition{
			"sonnet": {Model: "claude-sonnet-4-5", DisplayName: "Sonnet 4.5", Provider: "anthropic", APIKey: "sk-xxx"},
			"opus":   {Model: "claude-opus-4-5", DisplayName: "Opus 4.5", Provider: "anthropic", APIKey: "sk-yyy"},
		},
	}

	t.Run("existing model", func(t *testing.T) {
		model, ok := registry.GetModel("sonnet")
		if !ok {
			t.Error("expected to find model")
		}
		if model.Model != "claude-sonnet-4-5" {
			t.Errorf("Model = %q, want %q", model.Model, "claude-sonnet-4-5")
		}
	})

	t.Run("missing model", func(t *testing.T) {
		_, ok := registry.GetModel("nonexistent")
		if ok {
			t.Error("expected model not found")
		}
	})
}

func TestModelDefinition_ExtraHeadersJSON(t *testing.T) {
	def := ModelDefinition{
		Model:           "claude-opus-4-6",
		DisplayName:     "Opus 4.6 1M",
		BaseURL:         "https://api.anthropic.com",
		MaxOutputTokens: 128000,
		Provider:        "anthropic",
		ExtraHeaders: map[string]string{
			"anthropic-beta": "context-1m-2025-08-07",
		},
	}

	data, err := json.Marshal(def)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var parsed ModelDefinition
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if parsed.ExtraHeaders["anthropic-beta"] != "context-1m-2025-08-07" {
		t.Errorf("ExtraHeaders not round-tripped: got %v", parsed.ExtraHeaders)
	}
	if parsed.MaxOutputTokens != 128000 {
		t.Errorf("MaxOutputTokens: got %d, want 128000", parsed.MaxOutputTokens)
	}
}

func TestModelDefinition_NoExtraHeadersOmitted(t *testing.T) {
	def := ModelDefinition{
		Model:    "claude-sonnet-4-5",
		Provider: "anthropic",
	}

	data, err := json.Marshal(def)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	s := string(data)
	if containsStr(s, "extraHeaders") {
		t.Error("extraHeaders should be omitted when empty")
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestModelRegistry_HasModel(t *testing.T) {
	registry := &ModelRegistry{
		Models: map[string]ModelDefinition{
			"sonnet": {},
		},
	}

	if !registry.HasModel("sonnet") {
		t.Error("expected HasModel(sonnet) = true")
	}
	if registry.HasModel("nonexistent") {
		t.Error("expected HasModel(nonexistent) = false")
	}
}

func TestModelRegistry_ListModels(t *testing.T) {
	registry := &ModelRegistry{
		Models: map[string]ModelDefinition{
			"sonnet": {Model: "claude-sonnet-4-5", DisplayName: "Sonnet 4.5", Provider: "anthropic", APIKey: "sk-xxx"},
			"opus":   {Model: "claude-opus-4-5", DisplayName: "Opus 4.5", Provider: "anthropic", APIKey: "sk-yyy"},
		},
	}

	models := registry.ListModels()
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}

	// ModelInfo should not contain API keys
	for _, m := range models {
		if m.Name == "" {
			t.Error("model name should not be empty")
		}
		if m.DisplayName == "" {
			t.Error("model displayName should not be empty")
		}
	}
}

func TestModelRegistry_ResolveModel(t *testing.T) {
	registry := &ModelRegistry{
		Models: map[string]ModelDefinition{
			"sonnet": {Model: "claude-sonnet-4-5", DisplayName: "Sonnet 4.5", Provider: "anthropic"},
			"opus":   {Model: "claude-opus-4-5", DisplayName: "Opus 4.5", Provider: "anthropic"},
		},
	}

	t.Run("resolves shorthand", func(t *testing.T) {
		resolved := registry.ResolveModel("sonnet")
		if resolved != "claude-sonnet-4-5" {
			t.Errorf("ResolveModel(sonnet) = %q, want %q", resolved, "claude-sonnet-4-5")
		}
	})

	t.Run("passes through unknown model", func(t *testing.T) {
		resolved := registry.ResolveModel("claude-custom-model")
		if resolved != "claude-custom-model" {
			t.Errorf("ResolveModel(claude-custom-model) = %q, want %q", resolved, "claude-custom-model")
		}
	})
}
