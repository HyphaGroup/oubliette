# Oubliette Design Patterns

This document describes the key design patterns used throughout Oubliette. Understanding these patterns is essential for contributing code that integrates cleanly with the existing architecture.

## Table of Contents

- [Manager Pattern](#manager-pattern)
- [Handler Pattern](#handler-pattern)
- [Configuration Pattern](#configuration-pattern)
- [Per-Entity Locking Pattern](#per-entity-locking-pattern)
- [Ring Buffer Pattern](#ring-buffer-pattern)
- [Session Lifecycle](#session-lifecycle)
- [Workspace Resolution](#workspace-resolution)

---

## Manager Pattern

Managers are the primary abstraction for resource lifecycle operations. Each resource type (projects, sessions, containers) has a dedicated manager.

### Structure

```go
// Manager struct with dependencies and state
type Manager struct {
    dataDir      string           // Base directory for persistence
    dependencies *OtherManager    // Injected dependencies
    locks        *EntityLockMap   // Per-entity locking (if needed)
    index        sync.Map         // In-memory index (if needed)
}

// Constructor with dependency injection
func NewManager(dataDir string, deps *OtherManager) *Manager {
    return &Manager{
        dataDir:      dataDir,
        dependencies: deps,
    }
}
```

### Required Methods

Every manager implements these core operations:

```go
// Create - instantiate new resource
func (m *Manager) Create(ctx context.Context, req CreateRequest) (*Resource, error)

// Get - retrieve by ID (with validation)
func (m *Manager) Get(resourceID string) (*Resource, error)

// List - enumerate with optional filtering
func (m *Manager) List(filter *ListFilter) ([]*Resource, error)

// Delete - remove resource (with safety checks)
func (m *Manager) Delete(resourceID string) error
```

### Key Principles

1. **Context as first parameter** for cancellation and tracing
2. **Validate inputs** using `internal/validation/` before operations
3. **Acquire locks** before reading/writing shared state
4. **Return errors** - don't log and continue
5. **Use atomic writes** for persistence (write to temp file, then rename)

### Example: Project Manager

```go
// internal/project/manager.go

func (m *Manager) Create(req CreateProjectRequest) (*Project, error) {
    // 1. Generate identifiers
    projectID := uuid.New().String()
    
    // 2. Create directory structure
    projectDir := filepath.Join(m.projectsDir, projectID)
    if err := os.MkdirAll(projectDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create directory: %w", err)
    }
    
    // 3. Initialize resource state
    project := &Project{
        ID:        projectID,
        Name:      req.Name,
        CreatedAt: time.Now(),
    }
    
    // 4. Persist with atomic write
    if err := m.saveMetadata(project); err != nil {
        return nil, err
    }
    
    // 5. Update indexes
    m.indexProject(projectID, project.Name)
    
    return project, nil
}

func (m *Manager) Get(projectID string) (*Project, error) {
    // 1. Validate input
    if err := validation.ValidateProjectID(projectID); err != nil {
        return nil, err
    }
    
    // 2. Acquire read lock
    m.projectLocks.RLock(projectID)
    defer m.projectLocks.RUnlock(projectID)
    
    // 3. Load from disk
    data, err := os.ReadFile(m.metadataPath(projectID))
    if err != nil {
        return nil, fmt.Errorf("project %s not found", projectID)
    }
    
    // 4. Parse and return
    var project Project
    if err := json.Unmarshal(data, &project); err != nil {
        return nil, err
    }
    return &project, nil
}
```

### Files

- `internal/project/manager.go` - Project lifecycle
- `internal/session/manager.go` - Session lifecycle
- `internal/session/streaming.go` - Bidirectional streaming sessions

---

## Handler Pattern

Handlers process MCP tool calls. Each handler validates input, performs operations via managers, and returns structured responses.

### Structure

```go
// Handler method on Server
func (s *Server) handleToolName(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, *mcp.McpError, error)
```

### Return Values

- `(*mcp.CallToolResult, nil, nil)` - Success with content
- `(nil, &mcp.McpError{...}, nil)` - MCP-level error (invalid params, not found)
- `(nil, nil, error)` - System error (should not happen normally)

### Handler Template

```go
func (s *Server) handleSomeTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, *mcp.McpError, error) {
    // 1. Extract and validate parameters
    var params struct {
        ProjectID string `json:"project_id"`
        Name      string `json:"name"`
    }
    if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
        return nil, nil, err
    }
    
    // 2. Check authentication/authorization
    authCtx, err := requireProjectAccess(ctx, params.ProjectID)
    if err != nil {
        return nil, &mcp.McpError{Code: 403, Message: err.Error()}, nil
    }
    if !authCtx.CanWrite() {
        return nil, &mcp.McpError{Code: 403, Message: "read-only access"}, nil
    }
    
    // 3. Call manager method(s)
    result, err := s.someManager.DoOperation(ctx, params.ProjectID, params.Name)
    if err != nil {
        // Convert to user-friendly message
        return &mcp.CallToolResult{
            Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: %v", err)}},
        }, nil, nil
    }
    
    // 4. Format response
    return &mcp.CallToolResult{
        Content: []mcp.Content{&mcp.TextContent{Text: formatResult(result)}},
    }, nil, nil
}
```

### Authentication Context

Every handler that accesses resources must check authentication:

```go
// requireProjectAccess extracts auth context and verifies project access
func requireProjectAccess(ctx context.Context, projectID string) (*AuthContext, error) {
    authCtx := auth.FromContext(ctx)
    if authCtx == nil {
        return nil, fmt.Errorf("authentication required")
    }
    // Additional project-specific checks here
    return authCtx, nil
}
```

### Files

- `internal/mcp/handlers_project.go` - Project CRUD handlers
- `internal/mcp/handlers_session.go` - Session management handlers
- `internal/mcp/handlers_container.go` - Container operation handlers
- `internal/mcp/handlers_token.go` - Token management handlers
- `internal/mcp/handlers_workspace.go` - Workspace handlers

---

## Configuration Pattern

Configuration is loaded from a single `oubliette.jsonc` file. See [CONFIGURATION.md](CONFIGURATION.md) for the full reference.

### Loading

```go
cfg, err := config.LoadUnifiedConfig(configDir)
```

### Config Precedence

The server locates its config directory: `--dir` → `OUBLIETTE_HOME` → `./.oubliette` → `~/.oubliette`

### Adding New Configuration

1. Add field to the appropriate struct in `internal/config/`
2. Add JSON tag
3. Set default in `applyDefaults()`
4. Document in `config/oubliette.jsonc.example`

### Files

- `internal/config/unified.go` - Unified config loading
- `internal/config/models.go` - Model definitions
- `internal/config/credentials.go` - Credential management
- `config/oubliette.jsonc.example` - Documented example

---

## Per-Entity Locking Pattern

Provides fine-grained locking per resource instance to prevent concurrent modification conflicts while allowing parallel operations on different resources.

### Structure

```go
type EntityLockMap struct {
    locks sync.Map // entityID -> *sync.RWMutex
}

func (m *EntityLockMap) getOrCreateLock(entityID string) *sync.RWMutex {
    lock, _ := m.locks.LoadOrStore(entityID, &sync.RWMutex{})
    return lock.(*sync.RWMutex)
}

func (m *EntityLockMap) Lock(entityID string)   { m.getOrCreateLock(entityID).Lock() }
func (m *EntityLockMap) Unlock(entityID string) { m.getOrCreateLock(entityID).Unlock() }
func (m *EntityLockMap) RLock(entityID string)  { m.getOrCreateLock(entityID).RLock() }
func (m *EntityLockMap) RUnlock(entityID string){ m.getOrCreateLock(entityID).RUnlock() }
func (m *EntityLockMap) Delete(entityID string) { m.locks.Delete(entityID) }
```

### Usage

```go
// Write operations - exclusive lock
m.locks.Lock(projectID)
defer m.locks.Unlock(projectID)
// ... modify data ...

// Read operations - shared lock
m.locks.RLock(projectID)
defer m.locks.RUnlock(projectID)
// ... read data ...
```

### When to Use

- **Use per-entity locks** for metadata files that can be modified concurrently
- **Use single mutex** for in-memory maps that need atomic compound operations
- **Use sync.Map** for simple key-value caches with independent entries

### Files

- `internal/project/locks.go` - Project-level locking

---

## Ring Buffer Pattern

Used for event streaming with bounded memory and support for client disconnect/reconnect.

### Structure

```go
type EventBuffer struct {
    events     []*BufferedEvent
    maxSize    int
    startIndex int   // Logical index of first buffered event
    dropped    int64 // Count of dropped events
    mu         sync.RWMutex
}

type BufferedEvent struct {
    Index     int         // Monotonically increasing logical index
    Timestamp time.Time
    Event     *StreamEvent
}
```

### Key Operations

```go
// Append - add event, drop oldest if full
func (b *EventBuffer) Append(event *StreamEvent) int {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    index := b.startIndex + len(b.events)
    
    if len(b.events) >= b.maxSize {
        // Drop oldest event
        b.events = b.events[1:]
        b.startIndex++
        b.dropped++
    }
    
    b.events = append(b.events, &BufferedEvent{Index: index, Event: event})
    return index
}

// After - get events after index (for resumption)
func (b *EventBuffer) After(index int) ([]*BufferedEvent, error) {
    b.mu.RLock()
    defer b.mu.RUnlock()
    
    if index < b.startIndex-1 {
        return nil, fmt.Errorf("events purged (oldest: %d)", b.startIndex)
    }
    
    start := index - b.startIndex + 1
    if start >= len(b.events) {
        return []*BufferedEvent{}, nil
    }
    
    result := make([]*BufferedEvent, len(b.events)-start)
    copy(result, b.events[start:])
    return result, nil
}
```

### Client Polling Pattern

```
Client                          Server
  |                               |
  |-- GET /events?since=-1 ----->|  (initial poll)
  |<---- events[0..15] ----------|  (last_index: 15)
  |                               |
  |-- GET /events?since=15 ----->|  (resume from 15)
  |<---- events[16..42] ---------|  (last_index: 42)
  |                               |
  |  [client disconnects]         |
  |                               |
  |-- GET /events?since=42 ----->|  (reconnect, resume)
  |<---- events[43..50] ---------|  (last_index: 50)
```

### Files

- `internal/session/event_buffer.go` - Ring buffer implementation

---

---

## Session Lifecycle

Sessions progress through defined states with specific transitions.

### States

```
┌─────────┐     ┌─────────┐     ┌───────────┐
│ created │────>│ running │────>│ completed │
└─────────┘     └────┬────┘     └───────────┘
                     │
                     v
                ┌─────────┐
                │ failed  │
                └─────────┘
```

### State Transitions

| From | To | Trigger |
|------|-----|---------|
| - | created | `session.Manager.Create()` |
| created | running | `ActiveSession.Start()` |
| running | idle | Turn completes (completion event) |
| idle | running | New message received |
| running/idle | completed | Executor exits or session ended |
| running | failed | Error, timeout, or crash |

### Key Components

```
Session (metadata)          ActiveSession (runtime)
├── SessionID               ├── Session reference
├── ProjectID               ├── StreamingExecutor
├── WorkspaceID             │   ├── stdin (io.Writer)
├── Status                  │   ├── stdout (io.Reader)
├── Depth                   │   └── Events() chan
├── ParentSessionID         ├── EventBuffer (1000 events)
├── ChildSessionIDs         └── cancel func
├── ExplorationID
├── RuntimeSessionID
├── StartedAt
└── CompletedAt
```

### Files

- `internal/session/manager.go` - Session CRUD and persistence
- `internal/session/active.go` - Runtime session state
- `internal/session/types.go` - Session data structures

---

## Workspace Resolution

Workspaces provide isolation within projects. Resolution follows specific rules based on input parameters.

### Resolution Logic

```go
func resolveWorkspace(projectID string, workspaceID string, createWorkspace bool) (string, error) {
    // Case 1: No workspace specified, don't create
    if workspaceID == "" && !createWorkspace {
        return project.DefaultWorkspaceID, nil  // Use default
    }
    
    // Case 2: No workspace specified, create new
    if workspaceID == "" && createWorkspace {
        return createNewWorkspace(projectID), nil  // Generate UUID
    }
    
    // Case 3: Workspace specified, exists
    if workspaceExists(projectID, workspaceID) {
        return workspaceID, nil  // Use as-is
    }
    
    // Case 4: Workspace specified, doesn't exist, don't create
    if !createWorkspace {
        return "", fmt.Errorf("workspace %s not found", workspaceID)
    }
    
    // Case 5: Workspace specified, doesn't exist, create
    return createWorkspaceWithID(projectID, workspaceID), nil
}
```

### Decision Table

| `workspace_id` | `create_workspace` | Result |
|----------------|-------------------|--------|
| empty | false | Default workspace |
| empty | true | New UUID workspace |
| UUID (exists) | false | Specified workspace |
| UUID (exists) | true | Specified workspace |
| UUID (missing) | false | Error |
| UUID (missing) | true | Create with specified UUID |

### Files

- `internal/mcp/handlers_session.go` - `resolveWorkspaceGeneric()`
- `internal/project/manager.go` - `CreateWorkspace()`, `WorkspaceExists()`

---

## Summary

When contributing to Oubliette, follow these patterns:

1. **New resource type?** Create a Manager with Create/Get/List/Delete methods
2. **New MCP tool?** Add a handler following the template, check auth, call manager
3. **New config option?** Add to Config struct, set default, document in example
4. **Concurrent access?** Use per-entity locking for metadata, sync.Map for caches
5. **Streaming data?** Use ring buffer with index-based resumption
6. **Container operations?** Use Runtime interface, both implementations work identically

For questions about patterns not covered here, check existing code in the relevant `internal/` package.
