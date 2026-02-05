## MODIFIED Requirements

### Requirement: Per-Project Runtime Selection

Projects SHALL use their configured `agent_runtime` when spawning sessions, falling back to the server default only when no project-level override is specified.

#### Scenario: Project with agent_runtime uses that runtime

- **GIVEN** a project created with `"agent_runtime": "opencode"`
- **AND** the server default is `"droid"`
- **WHEN** a session is spawned for that project via `session_message`
- **THEN** the session uses the OpenCode runtime
- **AND** the Droid runtime is not invoked

#### Scenario: Project without agent_runtime uses server default

- **GIVEN** a project created without `agent_runtime` field (or empty value)
- **AND** the server default is `"droid"`
- **WHEN** a session is spawned for that project via `session_message`
- **THEN** the session uses the Droid runtime

#### Scenario: Child sessions inherit project runtime

- **GIVEN** a project with `"agent_runtime": "opencode"`
- **WHEN** a session spawns a child session via the socket relay
- **THEN** the child session also uses the OpenCode runtime
- **AND** runtime selection is consistent across the session tree

#### Scenario: Runtime resolved at spawn time

- **GIVEN** a project with `"agent_runtime": "opencode"`
- **AND** the OpenCode runtime is available
- **WHEN** a session is spawned
- **THEN** a fresh OpenCode runtime instance is created for the session
- **AND** the runtime is properly initialized before execution
