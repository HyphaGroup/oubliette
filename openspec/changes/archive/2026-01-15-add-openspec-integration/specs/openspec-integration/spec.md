# Capability: OpenSpec Integration

OpenSpec CLI and templates integrated into Oubliette containers for spec-driven agent workflows.

## ADDED Requirements

### Requirement: OpenSpec CLI Availability

The container image SHALL include the OpenSpec CLI (`@fission-ai/openspec`) globally installed and accessible in PATH.

#### Scenario: CLI version check
- **GIVEN** a running Oubliette container
- **WHEN** `openspec --version` is executed
- **THEN** the command succeeds and outputs a version number

#### Scenario: CLI help accessible
- **GIVEN** a running Oubliette container
- **WHEN** `openspec --help` is executed
- **THEN** the command outputs usage information

### Requirement: Pre-baked OpenSpec Templates

The container image SHALL include pre-generated OpenSpec templates in the `/template` directory.

The templates SHALL include:
- `openspec/AGENTS.md` - Agent workflow instructions
- `openspec/project.md` - Project context template
- `openspec/specs/` - Empty specs directory
- `openspec/changes/` - Empty changes directory with `archive/` subdirectory

#### Scenario: Template directory exists
- **GIVEN** the built container image
- **WHEN** listing `/template/openspec/`
- **THEN** AGENTS.md, project.md, specs/, and changes/ are present

#### Scenario: Factory Droid commands generated
- **GIVEN** the built container image
- **WHEN** listing `/template/.factory/commands/`
- **THEN** openspec-proposal.md, openspec-apply.md, and openspec-archive.md are present

### Requirement: Project Creation Includes OpenSpec

The `project_create` operation SHALL copy the OpenSpec template directory to new projects.

#### Scenario: New project has openspec directory
- **GIVEN** the Oubliette server is running
- **WHEN** `project_create` is called with a project name
- **THEN** the new project contains an `openspec/` directory
- **AND** the directory contains AGENTS.md, project.md, specs/, and changes/

#### Scenario: OpenSpec commands available in project
- **GIVEN** a newly created project
- **WHEN** listing the project's `.factory/commands/` directory
- **THEN** openspec-proposal.md, openspec-apply.md, and openspec-archive.md are present

### Requirement: Agent Access to OpenSpec

Spawned agent sessions SHALL have access to the OpenSpec CLI and project templates.

#### Scenario: Agent can list changes
- **GIVEN** an active agent session in a project
- **WHEN** the agent runs `openspec list`
- **THEN** the command succeeds (showing no changes or existing changes)

#### Scenario: Agent can validate specs
- **GIVEN** an active agent session in a project with openspec/ directory
- **WHEN** the agent runs `openspec validate --strict`
- **THEN** the command completes without errors for a valid project

#### Scenario: Agent can create proposal
- **GIVEN** an active agent session in a project
- **WHEN** the agent uses `/openspec-proposal` or creates a change manually
- **THEN** a new change directory is created under `openspec/changes/`

### Requirement: Workspace OpenSpec Inheritance

Workspaces SHALL have access to the project's OpenSpec directory via the shared filesystem.

#### Scenario: Workspace sees project specs
- **GIVEN** a project with openspec/ containing specs
- **WHEN** an agent session runs in a workspace of that project
- **THEN** the agent can read and modify the project's openspec/ directory

#### Scenario: Multiple workspaces share specs
- **GIVEN** two workspaces in the same project
- **WHEN** workspace A creates a change proposal
- **THEN** workspace B can see and continue work on that proposal

### Requirement: Project Changes MCP Tool

The system SHALL provide a `project_changes` MCP tool that returns structured change information by wrapping `openspec list --json`.

The tool SHALL:
- Execute `openspec list --json` in the project directory
- Pass through the structured JSON output
- Add session correlation information for active sessions

#### Scenario: Get changes for project
- **GIVEN** a project with active OpenSpec changes
- **WHEN** `project_changes` is called with the project_id
- **THEN** a JSON response is returned with change list
- **AND** each change includes name, completedTasks, totalTasks, status

#### Scenario: Changes sorted by naming convention
- **GIVEN** a project with changes named `010-first`, `020-second`, `alpha`
- **WHEN** `project_changes` is called
- **THEN** changes are returned sorted: `010-first`, `020-second`, `alpha`

#### Scenario: Empty project returns empty changes array
- **GIVEN** a project with no active OpenSpec changes
- **WHEN** `project_changes` is called
- **THEN** an empty `changes` array is returned

### Requirement: Project Tasks MCP Tool

The system SHALL provide a `project_tasks` MCP tool that returns structured task information by wrapping `openspec instructions apply --json`.

The tool SHALL:
- Execute `openspec instructions apply --change <id> --json` in the project directory
- Pass through the structured JSON output including tasks, progress, and state
- Add session correlation information for active sessions

#### Scenario: Get tasks for specific change
- **GIVEN** a project with `openspec/changes/add-feature/tasks.md` containing tasks
- **WHEN** `project_tasks` is called with project_id and change_id
- **THEN** a JSON response is returned with task list from OpenSpec CLI
- **AND** progress statistics are included (total, complete, remaining)
- **AND** state field indicates ready/blocked/all_done

#### Scenario: Correlate sessions with tasks
- **GIVEN** an active session spawned with `TaskContext` containing `change_id`
- **WHEN** `project_tasks` is called
- **THEN** the `active_sessions` array includes the session

#### Scenario: Detect completion state
- **GIVEN** a change with all tasks marked complete
- **WHEN** `project_tasks` is called
- **THEN** the `state` field is `all_done`

### Requirement: Session Modes

The system SHALL support session modes that configure agent behavior for different workflow stages.

The modes SHALL be:
- `interactive` (default) - Standard conversational mode
- `plan` - Agent focused on creating OpenSpec proposals
- `build` - Agent works through tasks until completion

#### Scenario: Spawn session in plan mode
- **GIVEN** the Oubliette server is running
- **WHEN** `session_message` is called with `mode: "plan"` and `message: "Add user auth"`
- **THEN** the agent receives the message as `/openspec-proposal Add user auth`
- **AND** the agent follows the OpenSpec proposal workflow

#### Scenario: Spawn session in build mode
- **GIVEN** a project with an active OpenSpec change "add-feature"
- **WHEN** `session_message` is called with `mode: "build"` and `change_id: "add-feature"`
- **THEN** the agent receives the message as `/openspec-apply add-feature`
- **AND** the agent follows the OpenSpec apply workflow

#### Scenario: Build mode without change_id builds all changes
- **GIVEN** a project with multiple incomplete OpenSpec changes
- **WHEN** `session_message` is called with `mode: "build"` but no `change_id`
- **THEN** Oubliette picks the first incomplete change from `openspec list --json`
- **AND** creates state file with `build_all: true`
- **AND** agent receives `/openspec-apply <first-change>`

#### Scenario: Build all advances to next change on completion
- **GIVEN** a build mode session with `build_all: true`
- **WHEN** the current change is complete (`state: "all_done"`)
- **AND** there are more incomplete changes
- **THEN** the stop hook updates state file with next change
- **AND** re-prompts agent with `/openspec-apply <next-change>`

#### Scenario: Build all completes when no more changes
- **GIVEN** a build mode session with `build_all: true`
- **WHEN** the current change completes
- **AND** there are no more incomplete changes
- **THEN** the stop hook deletes the state file
- **AND** allows the session to exit

#### Scenario: Default mode is interactive
- **GIVEN** the Oubliette server is running
- **WHEN** `session_message` is called without a `mode` parameter
- **THEN** the session operates in interactive mode (current behavior)

### Requirement: Build Mode Stop Hook

The system SHALL use a Factory Droid Stop hook to prevent premature exit and re-prompt the agent when tasks remain.

The Stop hook SHALL:
- Check for build mode state file (`$FACTORY_PROJECT_DIR/.factory/build-mode.json`)
- Query task completion via `openspec instructions apply --json`
- Block exit and re-prompt when `state != "all_done"` and iterations remain
- Allow exit when all tasks complete or max iterations reached
- Track iteration count to prevent infinite loops

#### Scenario: Stop hook blocks exit when tasks remain
- **GIVEN** a build mode session with incomplete tasks
- **WHEN** the agent tries to stop
- **THEN** the stop hook returns `decision: "block"`
- **AND** provides a reason with remaining task count
- **AND** the agent receives the reason and continues working

#### Scenario: Stop hook allows exit when complete
- **GIVEN** a build mode session where all tasks are marked complete
- **WHEN** the agent tries to stop
- **AND** `openspec instructions apply --json` returns `state: "all_done"`
- **THEN** the stop hook allows the exit (exit code 0)
- **AND** the build mode state file is deleted

#### Scenario: Stop hook respects max iterations
- **GIVEN** a build mode session that has reached max_iterations
- **WHEN** the agent tries to stop
- **THEN** the stop hook allows the exit regardless of task state
- **AND** logs that max iterations was reached

#### Scenario: Stop hook ignores non-build sessions
- **GIVEN** an interactive or plan mode session (no build-mode.json)
- **WHEN** the agent stops
- **THEN** the stop hook allows normal exit (exit code 0)

### Requirement: Build Mode State File

The system SHALL create a state file when spawning a build mode session.

The state file SHALL:
- Be located at `$FACTORY_PROJECT_DIR/.factory/build-mode.json`
- Contain `change_id`, `build_all`, `phase`, `max_iterations`, `iteration`, and `started_at` fields
- Be created by Oubliette when `session_message` is called with `mode: "build"`
- Be updated by the stop hook (increment iteration, change phase)
- Be deleted on completion or max iterations

#### Scenario: State file created on build mode spawn
- **GIVEN** the Oubliette server is running
- **WHEN** `session_message` is called with `mode: "build"` and `change_id`
- **THEN** a build-mode.json file is created in the project's `.factory/` directory
- **AND** the file contains the change_id, phase: "build", and initial iteration of 0

#### Scenario: State file deleted on completion
- **GIVEN** a build mode session where verification passes
- **WHEN** the stop hook detects VERIFIED marker after archiving
- **THEN** the build-mode.json file is deleted

### Requirement: Verification Phase

The system SHALL transition to a verification phase after tasks complete, before archiving.

#### Scenario: Transition to verify phase after tasks complete
- **GIVEN** a build mode session in phase: "build"
- **WHEN** `openspec instructions apply --json` returns `state: "all_done"`
- **THEN** the stop hook updates phase to "verify"
- **AND** sends verification prompt asking agent to run builds/tests
- **AND** blocks exit

#### Scenario: Verification prompt sent
- **GIVEN** a build mode session transitioning to verify phase
- **WHEN** the stop hook sends the verification prompt
- **THEN** the prompt instructs agent to run build, tests, linters
- **AND** the prompt tells agent to output "VERIFIED" when complete

#### Scenario: VERIFIED marker detected
- **GIVEN** a build mode session in phase: "verify"
- **WHEN** the agent outputs "VERIFIED" in its response
- **THEN** the stop hook archives the change via `openspec archive`
- **AND** commits the changes via git
- **AND** advances to next change (if build_all) or exits

#### Scenario: Verification continues until VERIFIED
- **GIVEN** a build mode session in phase: "verify"
- **WHEN** the agent tries to stop without outputting "VERIFIED"
- **THEN** the stop hook blocks exit
- **AND** re-prompts agent to continue verification

### Requirement: Task Reminder

The system SHALL remind the agent when tasks.md appears stale.

#### Scenario: Stale tasks.md triggers reminder
- **GIVEN** a build mode session where tasks.md was not modified since build started
- **WHEN** `openspec instructions apply --json` returns `state: "all_done"`
- **THEN** the stop hook detects tasks.md mtime < started_at
- **AND** blocks exit with reminder to update tasks.md
- **AND** does not transition to verify phase until tasks.md is updated

### Requirement: Archive on Completion

The system SHALL archive completed changes after verification passes.

#### Scenario: Change archived after verification
- **GIVEN** a build mode session in phase: "verify"
- **WHEN** the agent outputs "VERIFIED"
- **THEN** the stop hook runs `openspec archive <change_id>`
- **AND** runs `git add -A && git commit -m "feat(<change_id>): implementation complete"`

### Requirement: Enhanced session_events

The system SHALL support including child session events in session_events responses.

#### Scenario: session_events with include_children
- **GIVEN** a parent session with spawned child sessions
- **WHEN** `session_events` is called with `include_children: true`
- **THEN** events from both parent and child sessions are returned
- **AND** events are sorted by timestamp

#### Scenario: session_events without include_children
- **GIVEN** a parent session with spawned child sessions
- **WHEN** `session_events` is called without `include_children` parameter
- **THEN** only events from the parent session are returned

### Requirement: Node.js Runtime

The container image SHALL include Node.js runtime via NVM, sufficient to run OpenSpec.

Note: Node.js is already installed in the base image via NVM. This requirement documents the existing dependency.

#### Scenario: Node.js available via NVM
- **GIVEN** a running Oubliette container as user gogol
- **WHEN** `node --version` is executed (with NVM sourced)
- **THEN** the command succeeds and outputs a version >= 20.19.0

#### Scenario: npm available via NVM
- **GIVEN** a running Oubliette container as user gogol
- **WHEN** `npm --version` is executed (with NVM sourced)
- **THEN** the command succeeds and outputs a version number
