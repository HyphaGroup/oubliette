# Spec: Workspace Isolation Mode

## ADDED Requirements

### Requirement: Project-level workspace isolation setting

The system MUST support a project-level setting to control workspace isolation.

#### Scenario: Create project with isolation enabled
- **GIVEN** a user creating a new project
- **WHEN** `project_create` is called with `workspace_isolation: true`
- **THEN** the project metadata includes `workspace_isolation: true`
- **AND** subsequent workspaces for this project use isolated mounts

#### Scenario: Create project with isolation disabled (default)
- **GIVEN** a user creating a new project
- **WHEN** `project_create` is called without `workspace_isolation` parameter
- **THEN** the project metadata includes `workspace_isolation: false`
- **AND** subsequent workspaces use full project mounts (current behavior)

#### Scenario: Query project isolation setting
- **GIVEN** a project with `workspace_isolation: true`
- **WHEN** `project_get` is called
- **THEN** the response includes `workspace_isolation: true`

### Requirement: Isolated workspace mounts

The system MUST mount only the workspace directory when isolation is enabled.

#### Scenario: Container mount in isolated mode
- **GIVEN** a project with `workspace_isolation: true`
- **AND** a workspace with UUID `abc-123`
- **WHEN** a session is started for that workspace
- **THEN** the container mounts `projects/<id>/workspaces/abc-123/` at `/workspace`
- **AND** the container working directory is `/workspace`
- **AND** the agent cannot access paths outside `/workspace`

#### Scenario: Container mount in non-isolated mode
- **GIVEN** a project with `workspace_isolation: false`
- **AND** a workspace with UUID `abc-123`
- **WHEN** a session is started for that workspace
- **THEN** the container mounts `projects/<id>/` at `/workspace`
- **AND** the container working directory is `/workspace/workspaces/abc-123`
- **AND** the agent can access the full project directory

### Requirement: AGENTS.md copying for isolated workspaces

The system MUST copy project-level AGENTS.md to isolated workspaces.

#### Scenario: Workspace creation with project AGENTS.md
- **GIVEN** a project with `workspace_isolation: true`
- **AND** an `AGENTS.md` file exists at the project root
- **WHEN** a new workspace is created
- **THEN** `AGENTS.md` is copied to the workspace root
- **AND** the agent sees `AGENTS.md` at `/workspace/AGENTS.md`

#### Scenario: Workspace creation without project AGENTS.md
- **GIVEN** a project with `workspace_isolation: true`
- **AND** no `AGENTS.md` file exists at the project root
- **WHEN** a new workspace is created
- **THEN** the workspace is created without `AGENTS.md`
- **AND** no error occurs

#### Scenario: Workspace already has AGENTS.md
- **GIVEN** a project with `workspace_isolation: true`
- **AND** a workspace that already contains `AGENTS.md`
- **WHEN** workspace creation or session start occurs
- **THEN** the existing workspace `AGENTS.md` is preserved
- **AND** project-level `AGENTS.md` is NOT copied over it

### Requirement: Isolation prevents cross-workspace access

The system MUST prevent agents from accessing other workspaces when isolation is enabled.

#### Scenario: Agent attempts to access parent directory
- **GIVEN** a project with `workspace_isolation: true`
- **AND** an active session in workspace `abc-123`
- **WHEN** the agent attempts to read `/workspace/../` or `cd ..`
- **THEN** the agent remains within the workspace mount
- **AND** cannot see other workspaces, sessions, or project metadata

### Requirement: Protected paths are read-only

The system MUST mount specified paths as read-only when configured.

#### Scenario: Protect AGENTS.md and .factory directory
- **GIVEN** a project with `workspace_isolation: true`
- **AND** `protected_paths: ["AGENTS.md", ".factory/"]`
- **AND** workspace contains both `AGENTS.md` and `.factory/`
- **WHEN** a session is started for that workspace
- **THEN** `/workspace/AGENTS.md` is mounted read-only
- **AND** `/workspace/.factory/` is mounted read-only
- **AND** other files in `/workspace/` remain writable

#### Scenario: Agent attempts to modify protected file
- **GIVEN** a project with `workspace_isolation: true`
- **AND** `protected_paths: ["AGENTS.md"]`
- **AND** an active session
- **WHEN** the agent attempts to write to `/workspace/AGENTS.md`
- **THEN** the operation fails with "Read-only file system" error
- **AND** the file remains unchanged

#### Scenario: Protected path does not exist
- **GIVEN** a project with `workspace_isolation: true`
- **AND** `protected_paths: ["AGENTS.md", "nonexistent.md"]`
- **AND** workspace contains `AGENTS.md` but not `nonexistent.md`
- **WHEN** a session is started
- **THEN** `/workspace/AGENTS.md` is mounted read-only
- **AND** no error occurs for the missing path
- **AND** the session starts successfully

#### Scenario: Protected paths only apply with isolation enabled
- **GIVEN** a project with `workspace_isolation: false`
- **AND** `protected_paths: ["AGENTS.md"]`
- **WHEN** a session is started
- **THEN** the `protected_paths` setting is ignored
- **AND** all files remain writable (current behavior preserved)
