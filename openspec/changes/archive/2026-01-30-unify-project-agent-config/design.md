# Design: Unified Project Agent Configuration

## Architecture Overview

```
config/config-defaults.json (server defaults)
              ↓
         project_create
              ↓
    ┌─────────────────────────────────────┐
    │  projects/<id>/config.json          │
    │  (canonical source of truth)        │
    └─────────────────────────────────────┘
              ↓
    ┌─────────────────────────────────────┐
    │  internal/agent/config/             │
    │  ├── types.go      (canonical)      │
    │  ├── droid.go      (translator)     │
    │  └── opencode.go   (translator)     │
    └─────────────────────────────────────┘
              ↓                ↓
    .factory/mcp.json    opencode.json
    .factory/settings.json
```

## Canonical Config Schema

### Root Structure

```go
// internal/agent/config/types.go

type ProjectConfig struct {
    // Project identity (from current metadata.json)
    ID                 string    `json:"id"`
    Name               string    `json:"name"`
    Description        string    `json:"description,omitempty"`
    CreatedAt          time.Time `json:"created_at"`
    DefaultWorkspaceID string    `json:"default_workspace_id"`
    
    // Container settings
    Container ContainerConfig `json:"container"`
    
    // Agent runtime settings
    Agent AgentConfig `json:"agent"`
    
    // Resource limits
    Limits LimitsConfig `json:"limits"`
    
    // Isolation settings (existing)
    WorkspaceIsolation bool     `json:"workspace_isolation,omitempty"`
    ProtectedPaths     []string `json:"protected_paths,omitempty"`
}

type ContainerConfig struct {
    Type          string `json:"type"`                     // base, dev, osint
    ImageName     string `json:"image_name"`
    HasDockerfile bool   `json:"has_dockerfile,omitempty"`
    Status        string `json:"status,omitempty"`         // runtime state
    ID            string `json:"id,omitempty"`             // runtime state
}

type AgentConfig struct {
    Runtime       string                `json:"runtime"`                  // droid, opencode
    Model         string                `json:"model"`                    // e.g., claude-sonnet-4-5-20250929
    Autonomy      string                `json:"autonomy"`                 // low, medium, high
    Reasoning     string                `json:"reasoning,omitempty"`      // off, low, medium, high
    DisabledTools []string              `json:"disabled_tools,omitempty"`
    MCPServers    map[string]MCPServer  `json:"mcp_servers"`
    Permissions   map[string]any        `json:"permissions,omitempty"`    // OpenCode-style permissions
}

type MCPServer struct {
    Type        string            `json:"type"`                  // stdio, http
    Command     string            `json:"command,omitempty"`     // for stdio
    Args        []string          `json:"args,omitempty"`        // for stdio
    URL         string            `json:"url,omitempty"`         // for http
    Headers     map[string]string `json:"headers,omitempty"`     // for http
    Env         map[string]string `json:"env,omitempty"`
    Disabled    bool              `json:"disabled,omitempty"`
}

type LimitsConfig struct {
    MaxRecursionDepth   int     `json:"max_recursion_depth"`
    MaxAgentsPerSession int     `json:"max_agents_per_session"`
    MaxCostUSD          float64 `json:"max_cost_usd"`
}
```

## Translation Layer

### Droid Translation (`internal/agent/config/droid.go`)

```go
type DroidMCPConfig struct {
    MCPServers map[string]DroidMCPServer `json:"mcpServers"`
}

type DroidMCPServer struct {
    Type     string            `json:"type"`              // stdio, http
    Command  string            `json:"command,omitempty"`
    Args     []string          `json:"args,omitempty"`
    URL      string            `json:"url,omitempty"`
    Env      map[string]string `json:"env,omitempty"`
    Disabled bool              `json:"disabled,omitempty"`
}

func ToDroidMCPConfig(cfg *AgentConfig) *DroidMCPConfig {
    servers := make(map[string]DroidMCPServer)
    for name, srv := range cfg.MCPServers {
        servers[name] = DroidMCPServer{
            Type:     srv.Type,  // stdio/http same in canonical
            Command:  srv.Command,
            Args:     srv.Args,
            URL:      srv.URL,
            Env:      srv.Env,
            Disabled: srv.Disabled,
        }
    }
    return &DroidMCPConfig{MCPServers: servers}
}

// Settings translation maps to existing .factory/settings.json format
func ToDroidSettings(cfg *AgentConfig, models []ModelDefinition) *Settings {
    // Maps autonomy -> autonomyMode
    // Maps model -> sessionModel 
    // Maps reasoning -> reasoningEffort
    // ... existing logic from project/settings.go
}
```

### OpenCode Translation (`internal/agent/config/opencode.go`)

```go
type OpenCodeConfig struct {
    Schema     string                     `json:"$schema"`
    Model      string                     `json:"model"`
    Permission any                        `json:"permission"` // string or map
    Tools      map[string]bool            `json:"tools,omitempty"`
    MCP        map[string]OpenCodeMCP     `json:"mcp"`
    Provider   map[string]ProviderConfig  `json:"provider,omitempty"`
}

type OpenCodeMCP struct {
    Type        string            `json:"type"`                  // local, remote
    Command     []string          `json:"command,omitempty"`     // array for local
    URL         string            `json:"url,omitempty"`         // for remote
    Headers     map[string]string `json:"headers,omitempty"`
    Environment map[string]string `json:"environment,omitempty"`
    Enabled     bool              `json:"enabled"`
    OAuth       *OAuthConfig      `json:"oauth,omitempty"`
}

func ToOpenCodeConfig(cfg *AgentConfig) *OpenCodeConfig {
    // 1. Translate MCP servers
    mcp := make(map[string]OpenCodeMCP)
    for name, srv := range cfg.MCPServers {
        ocMCP := OpenCodeMCP{
            Enabled: !srv.Disabled,
        }
        switch srv.Type {
        case "stdio":
            ocMCP.Type = "local"
            // Combine command + args into array
            ocMCP.Command = append([]string{srv.Command}, srv.Args...)
            ocMCP.Environment = srv.Env
        case "http":
            ocMCP.Type = "remote"
            ocMCP.URL = srv.URL
            ocMCP.Headers = srv.Headers
        }
        mcp[name] = ocMCP
    }
    
    // 2. Translate model (add provider prefix)
    model := translateModelToOpenCodeFormat(cfg.Model)
    
    // 3. Translate autonomy to permissions
    permission := translateAutonomyToPermissions(cfg.Autonomy, cfg.Permissions)
    
    // 4. Translate disabled tools
    tools := make(map[string]bool)
    for _, t := range cfg.DisabledTools {
        tools[t] = false
    }
    
    return &OpenCodeConfig{
        Schema:     "https://opencode.ai/config.json",
        Model:      model,
        Permission: permission,
        Tools:      tools,
        MCP:        mcp,
    }
}

func translateModelToOpenCodeFormat(model string) string {
    switch {
    case strings.HasPrefix(model, "claude-"):
        return "anthropic/" + model
    case strings.HasPrefix(model, "gpt-"):
        return "openai/" + model
    case strings.HasPrefix(model, "gemini-"):
        return "google/" + model
    default:
        return model
    }
}

func translateAutonomyToPermissions(autonomy string, custom map[string]any) any {
    if len(custom) > 0 {
        return custom  // Use custom permissions if provided
    }
    switch autonomy {
    case "off":
        return "allow"  // String "allow" = unrestricted (skip-permissions-unsafe equivalent)
    case "high":
        return map[string]any{
            "*": "allow",
            "external_directory": "ask",
            "doom_loop": "ask",
        }
    case "medium":
        return map[string]any{
            "read": "allow",
            "edit": "allow",
            "bash": map[string]string{
                "*": "ask",
                "git *": "allow",
            },
        }
    case "low":
        return map[string]any{
            "read": "allow",
            "edit": "ask",
            "bash": "ask",
        }
    default:
        return "allow"  // Default to unrestricted if unknown
    }
}

// translateAutonomyToDroidFlag returns the Droid CLI flag for the autonomy level
func translateAutonomyToDroidFlag(autonomy string) []string {
    switch autonomy {
    case "off":
        return []string{"--skip-permissions-unsafe"}
    case "high":
        return []string{"--auto", "high"}
    case "medium":
        return []string{"--auto", "medium"}
    case "low":
        return []string{"--auto", "low"}
    default:
        return []string{"--skip-permissions-unsafe"}  // Default to unrestricted
    }
}

// translateReasoningToDroidFlag returns the Droid CLI flag for reasoning effort
func translateReasoningToDroidFlag(reasoning string) []string {
    switch reasoning {
    case "off":
        return []string{"-r", "off"}
    case "low":
        return []string{"-r", "low"}
    case "medium":
        return []string{"-r", "medium"}
    case "high":
        return []string{"-r", "high"}
    default:
        return []string{"-r", "medium"}  // Default to medium
    }
}

// translateReasoningToOpenCode configures reasoning for OpenCode
// OpenCode uses model variants or provider-specific options:
// - Anthropic: thinking.budgetTokens (high=default budget, max=maximum)
// - OpenAI: reasoningEffort (none, minimal, low, medium, high, xhigh)
// - Google: low, high variants
func translateReasoningToOpenCode(reasoning string, model string) map[string]any {
    // Provider detection from model prefix
    switch {
    case strings.HasPrefix(model, "anthropic/") || strings.HasPrefix(model, "claude-"):
        // Anthropic uses thinking.budgetTokens
        switch reasoning {
        case "off":
            return nil  // No thinking block
        case "low":
            return map[string]any{"thinking": map[string]any{"type": "enabled", "budgetTokens": 4000}}
        case "medium":
            return map[string]any{"thinking": map[string]any{"type": "enabled", "budgetTokens": 16000}}
        case "high":
            return map[string]any{"thinking": map[string]any{"type": "enabled", "budgetTokens": 32000}}
        default:
            return map[string]any{"thinking": map[string]any{"type": "enabled", "budgetTokens": 16000}}
        }
    case strings.HasPrefix(model, "openai/") || strings.HasPrefix(model, "gpt-"):
        // OpenAI uses reasoningEffort
        switch reasoning {
        case "off":
            return map[string]any{"reasoningEffort": "none"}
        case "low":
            return map[string]any{"reasoningEffort": "low"}
        case "medium":
            return map[string]any{"reasoningEffort": "medium"}
        case "high":
            return map[string]any{"reasoningEffort": "high"}
        default:
            return map[string]any{"reasoningEffort": "medium"}
        }
    case strings.HasPrefix(model, "google/") || strings.HasPrefix(model, "gemini-"):
        // Google uses low/high variants
        switch reasoning {
        case "off", "low":
            return map[string]any{"variant": "low"}
        case "medium", "high":
            return map[string]any{"variant": "high"}
        default:
            return map[string]any{"variant": "high"}
        }
    default:
        return nil  // Unknown provider, no reasoning config
    }
}
```

## Project Manager Changes

### Config File Operations

```go
// internal/project/manager.go

const configFileName = "config.json"

func (m *Manager) loadConfig(projectID string) (*config.ProjectConfig, error) {
    configPath := filepath.Join(m.projectsDir, projectID, configFileName)
    
    data, err := os.ReadFile(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read config: %w", err)
    }
    
    var cfg config.ProjectConfig
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }
    return &cfg, nil
}

func (m *Manager) saveConfig(cfg *config.ProjectConfig) error {
    configPath := filepath.Join(m.projectsDir, cfg.ID, configFileName)
    // ... save logic ...
}
```

### Runtime Config Generation

```go
// internal/project/manager.go

func (m *Manager) generateRuntimeConfigs(cfg *config.ProjectConfig) error {
    projectDir := filepath.Join(m.projectsDir, cfg.ID)
    
    // Always generate both runtime configs
    
    // 1. Droid: .factory/mcp.json
    droidMCP := agentconfig.ToDroidMCPConfig(&cfg.Agent)
    if err := writeJSON(filepath.Join(projectDir, ".factory", "mcp.json"), droidMCP); err != nil {
        return fmt.Errorf("failed to write droid mcp config: %w", err)
    }
    
    // 2. Droid: .factory/settings.json  
    droidSettings := agentconfig.ToDroidSettings(&cfg.Agent, m.modelRegistry)
    if err := writeJSON(filepath.Join(projectDir, ".factory", "settings.json"), droidSettings); err != nil {
        return fmt.Errorf("failed to write droid settings: %w", err)
    }
    
    // 3. OpenCode: opencode.json
    openCodeCfg := agentconfig.ToOpenCodeConfig(&cfg.Agent)
    if err := writeJSON(filepath.Join(projectDir, "opencode.json"), openCodeCfg); err != nil {
        return fmt.Errorf("failed to write opencode config: %w", err)
    }
    
    return nil
}
```

## Server Defaults Schema

Rename `config/project-defaults.json` to `config/config-defaults.json` and expand:

```json
{
  "limits": {
    "max_recursion_depth": 3,
    "max_agents_per_session": 50,
    "max_cost_usd": 10.0
  },
  "agent": {
    "runtime": "droid",
    "model": "claude-sonnet-4-5-20250929",
    "autonomy": "off",
    "reasoning": "medium",
    "mcp_servers": {
      "oubliette-parent": {
        "type": "stdio",
        "command": "/usr/local/bin/oubliette-client",
        "args": ["/mcp/relay.sock"]
      }
    }
  },
  "container": {
    "type": "dev"
  }
}
```

Note: `autonomy: "off"` is the default because agents run in isolated containers with no human to respond to permission prompts.

## API Changes

### CreateProjectRequest

Simplify to match canonical config structure:

```go
type CreateProjectRequest struct {
    Name               string
    Description        string
    GitHubToken        string
    RemoteURL          string
    InitGit            bool
    
    // Container
    ContainerType string
    
    // Agent (all optional, uses defaults)
    AgentRuntime    string
    Model           string
    Autonomy        string
    Reasoning       string
    DisabledTools   []string
    MCPServers      map[string]MCPServer  // Additional MCP servers
    Permissions     map[string]any        // Custom permissions
    
    // Limits (all optional, uses defaults)
    MaxRecursionDepth   *int
    MaxAgentsPerSession *int
    MaxCostUSD          *float64
    
    // Isolation
    WorkspaceIsolation bool
    ProtectedPaths     []string
}
```

Note: `IncludedModels` and `SessionModel` are removed - the new system uses a single `Model` field. Model variants can be added later if needed.

## File Layout After Change

```
projects/<id>/
├── config.json              # NEW: canonical config (replaces metadata.json)
├── .factory/
│   ├── mcp.json             # GENERATED: Droid MCP config
│   ├── settings.json        # GENERATED: Droid settings
│   ├── commands/            # unchanged
│   ├── hooks/               # unchanged
│   ├── droids/              # NEW: custom droid definitions (Droid runtime)
│   └── skills/              # NEW: skill definitions (Droid runtime)
├── .opencode/
│   ├── agents/              # NEW: agent definitions (OpenCode runtime)
│   └── skills/              # NEW: skill definitions (OpenCode runtime)
├── opencode.json            # NEW GENERATED: OpenCode config
├── workspaces/
│   └── <uuid>/
│       ├── metadata.json    # unchanged (workspace metadata)
│       ├── .factory/        # workspace-specific overrides
│       └── opencode.json    # workspace-specific (future)
└── sessions/                # unchanged
```

## Read-Only Config Mounts

Config files are mounted read-only in containers to prevent agents from self-modification.

### Container Mount Strategy

```go
// internal/container/mounts.go (or similar)

func getConfigMounts(projectDir string) []Mount {
    return []Mount{
        // Canonical config - read-only
        {
            Source:   filepath.Join(projectDir, "config.json"),
            Target:   "/workspace/config.json",
            ReadOnly: true,
        },
        // OpenCode config - read-only
        {
            Source:   filepath.Join(projectDir, "opencode.json"),
            Target:   "/workspace/opencode.json",
            ReadOnly: true,
        },
        // Droid .factory directory - read-only for config files
        // Note: .factory/commands and .factory/hooks may need write access
        {
            Source:   filepath.Join(projectDir, ".factory", "mcp.json"),
            Target:   "/workspace/.factory/mcp.json",
            ReadOnly: true,
        },
        {
            Source:   filepath.Join(projectDir, ".factory", "settings.json"),
            Target:   "/workspace/.factory/settings.json",
            ReadOnly: true,
        },
    }
}
```

### What's Read-Only vs Writable

| Path | Mount Type | Reason |
|------|------------|--------|
| `config.json` | Read-only | Canonical config - admin only |
| `opencode.json` | Read-only | Generated config - admin only |
| `.factory/mcp.json` | Read-only | Generated config - admin only |
| `.factory/settings.json` | Read-only | Generated config - admin only |
| `.factory/commands/` | Writable | Agent may create custom commands |
| `.factory/hooks/` | Writable | Agent may modify hooks |
| `.factory/droids/` | Writable | Agent may create custom droids |
| `.factory/skills/` | Writable | Agent may create skills |
| `.opencode/agents/` | Writable | Agent may create OpenCode agents |
| `.opencode/skills/` | Writable | Agent may create OpenCode skills |
| `workspaces/<uuid>/` | Writable | Agent working directory |

### Security Rationale

1. **Prevent privilege escalation**: Agent cannot grant itself higher autonomy
2. **Prevent config corruption**: Malformed config would break future sessions
3. **Audit trail**: Config changes require external action (MCP or host access)
4. **Reproducibility**: Same config always produces same behavior
