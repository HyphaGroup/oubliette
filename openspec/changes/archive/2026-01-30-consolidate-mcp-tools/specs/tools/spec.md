# Spec: Consolidated MCP Tools

Resource-based tools with action parameter replacing separate CRUD tools.

## ADDED Requirements

### Requirement: Action Parameter Pattern

All consolidated tools SHALL require an `action` parameter to specify the operation.

#### Scenario: Valid action executes
- **GIVEN** a call to `project` tool with `action: "create"`
- **WHEN** the tool is invoked
- **THEN** the create operation executes
- **AND** response matches previous `project_create` behavior

#### Scenario: Invalid action returns error
- **GIVEN** a call to `project` tool with `action: "invalid"`
- **WHEN** the tool is invoked
- **THEN** error returned: "unknown action 'invalid' for project tool; valid actions: create, list, get, delete"

#### Scenario: Missing action returns error
- **GIVEN** a call to `project` tool without `action` parameter
- **WHEN** the tool is invoked
- **THEN** error returned: "action parameter is required"

### Requirement: Project Tool

The `project` tool SHALL support actions: create, list, get, delete.

#### Scenario: project action=create
- **GIVEN** `action: "create"` with `name` and optional `source_url`
- **WHEN** invoked
- **THEN** creates project (same as old `project_create`)

#### Scenario: project action=list
- **GIVEN** `action: "list"`
- **WHEN** invoked
- **THEN** returns all projects (same as old `project_list`)

#### Scenario: project action=get
- **GIVEN** `action: "get"` with `project_id`
- **WHEN** invoked
- **THEN** returns project details (same as old `project_get`)

#### Scenario: project action=delete
- **GIVEN** `action: "delete"` with `project_id`
- **WHEN** invoked
- **THEN** deletes project (same as old `project_delete`)

### Requirement: Container Tool

The `container` tool SHALL support actions: start, stop, logs, exec.

#### Scenario: container action=start
- **GIVEN** `action: "start"` with `project_id`
- **WHEN** invoked
- **THEN** starts container (same as old `container_start`)

#### Scenario: container action=stop
- **GIVEN** `action: "stop"` with `project_id`
- **WHEN** invoked
- **THEN** stops container (same as old `container_stop`)

#### Scenario: container action=logs
- **GIVEN** `action: "logs"` with `project_id`
- **WHEN** invoked
- **THEN** returns logs (same as old `container_logs`)

#### Scenario: container action=exec
- **GIVEN** `action: "exec"` with `project_id` and `command`
- **WHEN** invoked
- **THEN** executes command (same as old `container_exec`)

### Requirement: Session Tool

The `session` tool SHALL support actions: spawn, message, get, list, end, events, cleanup.

#### Scenario: session action=spawn
- **GIVEN** `action: "spawn"` with `project_id` and optional workspace/prompt params
- **WHEN** invoked
- **THEN** spawns session (same as old `session_spawn`)

#### Scenario: session action=message
- **GIVEN** `action: "message"` with `project_id`, `workspace_id`, `message`
- **WHEN** invoked
- **THEN** sends message (same as old `session_message`)

#### Scenario: session action=get
- **GIVEN** `action: "get"` with `project_id`, `session_id`
- **WHEN** invoked
- **THEN** returns session (same as old `session_get`)

#### Scenario: session action=list
- **GIVEN** `action: "list"` with `project_id`
- **WHEN** invoked
- **THEN** returns sessions (same as old `session_list`)

#### Scenario: session action=end
- **GIVEN** `action: "end"` with `project_id`, `session_id`
- **WHEN** invoked
- **THEN** ends session (same as old `session_end`)

#### Scenario: session action=events
- **GIVEN** `action: "events"` with `project_id`, `session_id`
- **WHEN** invoked
- **THEN** returns events (same as old `session_events`)

#### Scenario: session action=cleanup
- **GIVEN** `action: "cleanup"` with `project_id`
- **WHEN** invoked
- **THEN** cleans old sessions (same as old `session_cleanup`)

### Requirement: Workspace Tool

The `workspace` tool SHALL support actions: list, delete.

#### Scenario: workspace action=list
- **GIVEN** `action: "list"` with `project_id`
- **WHEN** invoked
- **THEN** returns workspaces (same as old `workspace_list`)

#### Scenario: workspace action=delete
- **GIVEN** `action: "delete"` with `project_id`, `workspace_id`
- **WHEN** invoked
- **THEN** deletes workspace (same as old `workspace_delete`)

### Requirement: Token Tool

The `token` tool SHALL support actions: create, list, revoke.

#### Scenario: token action=create
- **GIVEN** `action: "create"` with scope and optional params
- **WHEN** invoked
- **THEN** creates token (same as old `token_create`)

#### Scenario: token action=list
- **GIVEN** `action: "list"`
- **WHEN** invoked
- **THEN** returns tokens (same as old `token_list`)

#### Scenario: token action=revoke
- **GIVEN** `action: "revoke"` with `token_id`
- **WHEN** invoked
- **THEN** revokes token (same as old `token_revoke`)

### Requirement: Schedule Tool

The `schedule` tool SHALL support actions: create, list, get, update, delete, trigger.

#### Scenario: schedule action=create
- **GIVEN** `action: "create"` with cron, prompt, targets
- **WHEN** invoked
- **THEN** creates schedule (same as old `schedule_create`)

#### Scenario: schedule action=list
- **GIVEN** `action: "list"`
- **WHEN** invoked
- **THEN** returns schedules (same as old `schedule_list`)

#### Scenario: schedule action=get
- **GIVEN** `action: "get"` with `schedule_id`
- **WHEN** invoked
- **THEN** returns schedule (same as old `schedule_get`)

#### Scenario: schedule action=update
- **GIVEN** `action: "update"` with `schedule_id` and fields
- **WHEN** invoked
- **THEN** updates schedule (same as old `schedule_update`)

#### Scenario: schedule action=delete
- **GIVEN** `action: "delete"` with `schedule_id`
- **WHEN** invoked
- **THEN** deletes schedule (same as old `schedule_delete`)

#### Scenario: schedule action=trigger
- **GIVEN** `action: "trigger"` with `schedule_id`
- **WHEN** invoked
- **THEN** triggers schedule (same as old `schedule_trigger`)

### Requirement: Standalone Tools Unchanged

The following tools SHALL remain as separate tools with no changes.

#### Scenario: project_options unchanged
- **WHEN** `project_options` is called
- **THEN** behavior identical to current

#### Scenario: project_changes unchanged
- **WHEN** `project_changes` is called
- **THEN** behavior identical to current

#### Scenario: project_tasks unchanged
- **WHEN** `project_tasks` is called
- **THEN** behavior identical to current

#### Scenario: image_rebuild unchanged
- **WHEN** `image_rebuild` is called
- **THEN** behavior identical to current

#### Scenario: caller_tool_response unchanged
- **WHEN** `caller_tool_response` is called
- **THEN** behavior identical to current

#### Scenario: config_limits unchanged
- **WHEN** `config_limits` is called
- **THEN** behavior identical to current

## REMOVED Requirements

### Requirement: project_create Tool
**Reason**: Consolidated into `project` tool with `action: "create"`

### Requirement: project_list Tool
**Reason**: Consolidated into `project` tool with `action: "list"`

### Requirement: project_get Tool
**Reason**: Consolidated into `project` tool with `action: "get"`

### Requirement: project_delete Tool
**Reason**: Consolidated into `project` tool with `action: "delete"`

### Requirement: container_start Tool
**Reason**: Consolidated into `container` tool with `action: "start"`

### Requirement: container_stop Tool
**Reason**: Consolidated into `container` tool with `action: "stop"`

### Requirement: container_logs Tool
**Reason**: Consolidated into `container` tool with `action: "logs"`

### Requirement: container_exec Tool
**Reason**: Consolidated into `container` tool with `action: "exec"`

### Requirement: session_spawn Tool
**Reason**: Consolidated into `session` tool with `action: "spawn"`

### Requirement: session_message Tool
**Reason**: Consolidated into `session` tool with `action: "message"`

### Requirement: session_get Tool
**Reason**: Consolidated into `session` tool with `action: "get"`

### Requirement: session_list Tool
**Reason**: Consolidated into `session` tool with `action: "list"`

### Requirement: session_end Tool
**Reason**: Consolidated into `session` tool with `action: "end"`

### Requirement: session_events Tool
**Reason**: Consolidated into `session` tool with `action: "events"`

### Requirement: session_cleanup Tool
**Reason**: Consolidated into `session` tool with `action: "cleanup"`

### Requirement: workspace_list Tool
**Reason**: Consolidated into `workspace` tool with `action: "list"`

### Requirement: workspace_delete Tool
**Reason**: Consolidated into `workspace` tool with `action: "delete"`

### Requirement: token_create Tool
**Reason**: Consolidated into `token` tool with `action: "create"`

### Requirement: token_list Tool
**Reason**: Consolidated into `token` tool with `action: "list"`

### Requirement: token_revoke Tool
**Reason**: Consolidated into `token` tool with `action: "revoke"`

### Requirement: schedule_create Tool
**Reason**: Consolidated into `schedule` tool with `action: "create"`

### Requirement: schedule_list Tool
**Reason**: Consolidated into `schedule` tool with `action: "list"`

### Requirement: schedule_get Tool
**Reason**: Consolidated into `schedule` tool with `action: "get"`

### Requirement: schedule_update Tool
**Reason**: Consolidated into `schedule` tool with `action: "update"`

### Requirement: schedule_delete Tool
**Reason**: Consolidated into `schedule` tool with `action: "delete"`

### Requirement: schedule_trigger Tool
**Reason**: Consolidated into `schedule` tool with `action: "trigger"`
