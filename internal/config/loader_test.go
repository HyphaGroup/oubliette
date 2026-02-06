package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUnifiedConfig(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("valid unified config", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "valid.jsonc")
		configJSON := `{
			// Test config
			"server": {
				"address": ":9000"
			},
			"credentials": {
				"github": {"credentials": {}, "default": ""},
				"providers": {"credentials": {}, "default": ""}
			},
			"defaults": {
				"limits": {"max_recursion_depth": 5, "max_agents_per_session": 100, "max_cost_usd": 50.00},
				"agent": {"model": "sonnet", "autonomy": "off", "reasoning": "medium"},
				"container": {"type": "osint"},
				"backup": {"enabled": false, "directory": "backups", "retention": 14, "interval_hours": 12}
			},
			"models": {
				"models": {
					"test": {"model": "test-model", "displayName": "Test", "provider": "test"}
				},
				"defaults": {}
			}
		}`
		_ = os.WriteFile(configPath, []byte(configJSON), 0o644)

		cfg, err := LoadUnifiedConfig(configPath)
		if err != nil {
			t.Fatalf("LoadUnifiedConfig() error = %v", err)
		}
		if cfg.Server.Address != ":9000" {
			t.Errorf("Server.Address = %q, want %q", cfg.Server.Address, ":9000")
		}

		if cfg.Defaults.Limits.MaxRecursionDepth != 5 {
			t.Errorf("Defaults.Limits.MaxRecursionDepth = %d, want %d", cfg.Defaults.Limits.MaxRecursionDepth, 5)
		}
		if cfg.Defaults.Container.Type != "osint" {
			t.Errorf("Defaults.Container.Type = %q, want %q", cfg.Defaults.Container.Type, "osint")
		}
		if len(cfg.Models.Models) != 1 {
			t.Errorf("len(Models.Models) = %d, want 1", len(cfg.Models.Models))
		}
	})

	t.Run("JSONC comments are stripped", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "comments.jsonc")
		configJSON := `{
			// Line comment
			"server": {"address": ":8080"},
			/* Block comment */
			"credentials": {
				"factory": {
					"credentials": {"default": {"api_key": "key"}},
					"default": "default"
				}
			},
			"defaults": {},
			"models": {}
		}`
		_ = os.WriteFile(configPath, []byte(configJSON), 0o644)

		cfg, err := LoadUnifiedConfig(configPath)
		if err != nil {
			t.Fatalf("LoadUnifiedConfig() error = %v", err)
		}
		if cfg.Server.Address != ":8080" {
			t.Errorf("Server.Address = %q, want %q", cfg.Server.Address, ":8080")
		}
	})

	t.Run("applies defaults for missing fields", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "minimal.jsonc")
		configJSON := `{
			"server": {},
			"credentials": {
				"factory": {
					"credentials": {"default": {"api_key": "key"}},
					"default": "default"
				}
			},
			"defaults": {},
			"models": {}
		}`
		_ = os.WriteFile(configPath, []byte(configJSON), 0o644)

		cfg, err := LoadUnifiedConfig(configPath)
		if err != nil {
			t.Fatalf("LoadUnifiedConfig() error = %v", err)
		}
		if cfg.Server.Address != ":8080" {
			t.Errorf("Server.Address = %q, want default %q", cfg.Server.Address, ":8080")
		}
		if cfg.Defaults.Limits.MaxRecursionDepth != 3 {
			t.Errorf("Defaults.Limits.MaxRecursionDepth = %d, want default %d", cfg.Defaults.Limits.MaxRecursionDepth, 3)
		}
		if cfg.Defaults.Container.Type != "dev" {
			t.Errorf("Defaults.Container.Type = %q, want default %q", cfg.Defaults.Container.Type, "dev")
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "invalid.jsonc")
		_ = os.WriteFile(configPath, []byte("not json"), 0o644)

		_, err := LoadUnifiedConfig(configPath)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestFindConfigPath(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("finds config in specified dir", func(t *testing.T) {
		configDir := filepath.Join(tmpDir, "custom")
		_ = os.MkdirAll(configDir, 0o755)
		_ = os.WriteFile(filepath.Join(configDir, "oubliette.jsonc"), []byte("{}"), 0o644)

		path, err := FindConfigPath(configDir)
		if err != nil {
			t.Fatalf("FindConfigPath() error = %v", err)
		}
		if filepath.Base(path) != "oubliette.jsonc" {
			t.Errorf("FindConfigPath() = %q, want oubliette.jsonc", path)
		}
	})

	t.Run("error when config not found", func(t *testing.T) {
		_, err := FindConfigPath(filepath.Join(tmpDir, "nonexistent"))
		if err == nil {
			t.Error("expected error when config not found")
		}
	})
}

func TestLoadAll(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("loads unified config", func(t *testing.T) {
		configDir := filepath.Join(tmpDir, "all")
		_ = os.MkdirAll(configDir, 0o755)

		configJSON := `{
			"server": {"address": ":7000"},
			"credentials": {
				"providers": {
					"credentials": {"default": {"api_key": "test-key", "provider": "anthropic"}},
					"default": "default"
				}
			},
			"defaults": {
				"limits": {"max_recursion_depth": 10, "max_agents_per_session": 100, "max_cost_usd": 25.0},
				"agent": {"model": "gpt-5.1", "autonomy": "high", "reasoning": "low"},
				"container": {"type": "osint"}
			},
			"models": {
				"models": {"sonnet": {"model": "claude-sonnet-4-5", "provider": "anthropic"}}
			}
		}`
		_ = os.WriteFile(filepath.Join(configDir, "oubliette.jsonc"), []byte(configJSON), 0o644)

		cfg, err := LoadAll(configDir)
		if err != nil {
			t.Fatalf("LoadAll() error = %v", err)
		}
		if cfg.Server.Address != ":7000" {
			t.Errorf("Server.Address = %q, want %q", cfg.Server.Address, ":7000")
		}
		provCred, ok := cfg.Credentials.GetDefaultProviderCredential()
		if !ok || provCred.APIKey != "test-key" {
			t.Errorf("Credentials.GetDefaultProviderCredential() key = %q, want %q", provCred.APIKey, "test-key")
		}
		if cfg.ConfigDefaults.Limits.MaxRecursionDepth != 10 {
			t.Errorf("ConfigDefaults.Limits.MaxRecursionDepth = %d, want %d", cfg.ConfigDefaults.Limits.MaxRecursionDepth, 10)
		}
		// Check legacy ProjectDefaults populated from ConfigDefaults
		if cfg.ProjectDefaults.MaxRecursionDepth != 10 {
			t.Errorf("ProjectDefaults.MaxRecursionDepth = %d, want %d", cfg.ProjectDefaults.MaxRecursionDepth, 10)
		}
		if cfg.ProjectDefaults.ContainerType != "osint" {
			t.Errorf("ProjectDefaults.ContainerType = %q, want %q", cfg.ProjectDefaults.ContainerType, "osint")
		}
		// Check models loaded
		if cfg.Models == nil || len(cfg.Models.Models) != 1 {
			t.Errorf("Models not loaded correctly")
		}
	})
}

func TestLoadedConfig_Validate(t *testing.T) {
	t.Run("empty config is valid", func(t *testing.T) {
		// API keys are optional - OpenCode can run without them
		cfg := &LoadedConfig{
			Credentials: &CredentialRegistry{},
		}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Validate() error = %v, want nil (API keys are optional)", err)
		}
	})

	t.Run("config with credentials is valid", func(t *testing.T) {
		cfg := &LoadedConfig{
			Credentials: &CredentialRegistry{
				Providers: ProviderCredentials{
					Credentials: map[string]ProviderCredential{
						"default": {APIKey: "test-key", Provider: "anthropic"},
					},
					Default: "default",
				},
			},
		}
		err := cfg.Validate()
		if err != nil {
			t.Errorf("Validate() error = %v", err)
		}
	})
}
