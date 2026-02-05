# Design: OpenSpec Integration

## Context

Oubliette provides containerized execution environments for AI agents. Agents currently receive instructions via prompts but lack a structured way to propose, review, and track specifications before implementation. OpenSpec is a lightweight spec-driven development tool that works with AI coding assistants.

**Stakeholders**: Developers using Oubliette for autonomous agent work, agents needing structured workflows.

**Constraints**:
- Container image size should not increase significantly
- Project creation performance should not degrade noticeably
- Must work with existing `.factory/` configuration pattern
- Agents must be able to use OpenSpec without additional setup

## Goals / Non-Goals

**Goals**:
- Provide spec-driven workflow tooling in all agent containers
- Pre-bake OpenSpec templates for fast project creation
- Enable agents to use OpenSpec slash commands
- Maintain consistency with existing template/copy patterns
- Provide `project_changes` and `project_tasks` MCP tools for structured visibility
- Correlate active sessions with OpenSpec tasks
- Enable autonomous development via plan and build modes
- Support completion detection for build loops (Ralph-style)

**Non-Goals**:
- Full MCP wrappers for all OpenSpec commands (CLI JSON output sufficient)
- Automatic task assignment to sessions
- Custom OpenSpec extensions for Oubliette
- Complex priority/dependency graphs (use naming convention instead)

## Decisions

### D1: Installation Method

**Decision**: Install OpenSpec globally via npm using existing NVM-managed Node.js

**Rationale**:
- OpenSpec is distributed via npm (`@fission-ai/openspec`)
- Container already has Node.js LTS installed via NVM for user `gogol`
- Global install via NVM makes `openspec` CLI available in PATH
- No additional Node.js installation needed

**Implementation**:
```dockerfile
# Install OpenSpec CLI (after NVM setup, as gogol user)
RUN . "$NVM_DIR/nvm.sh" && npm install -g @fission-ai/openspec@latest
```

### D2: Template Generation Strategy

**Decision**: Commit pre-generated OpenSpec templates to `template/openspec/` in the repository

**Rationale**:
- Matches existing pattern (`template/.factory/` is committed, not generated)
- No Docker build dependency on `openspec init`
- Templates can be customized for Oubliette context
- Simpler, more explicit than runtime generation

**Alternative Considered**: Run `openspec init` during Docker build
- Rejected: Adds build complexity, templates become opaque
- Current `.factory/` templates are hand-crafted, not generated

**Implementation**:
- Run `openspec init --tools factory` locally once
- Commit resulting `template/openspec/` to repository
- Customize `project.md` with Oubliette-specific context

### D3: Project Creation Integration

**Decision**: Copy `template/openspec/` to project root during `project_create`

**Rationale**:
- Matches existing pattern for `.factory/` copying
- Simple, predictable behavior
- Project-level specs make sense (workspaces share project specs)

**Alternative Considered**: Per-workspace openspec directories
- Rejected: Adds complexity, specs are typically project-scoped
- Can revisit if use cases emerge

### D4: Workspace Inheritance

**Decision**: Workspaces inherit project's openspec/ via working directory (no copy)

**Rationale**:
- Specs are project-level concerns
- Workspaces already share project files
- Avoids duplication and sync issues
- Changes in one workspace visible to others (intentional)

### D5: Slash Command Configuration

**Decision**: Use Factory Droid's `.factory/commands/` for OpenSpec commands

**Rationale**:
- Factory Droid already supports slash commands via this directory
- OpenSpec's `--tools factory` generates appropriate command files
- Commands: `/openspec-proposal`, `/openspec-apply`, `/openspec-archive`

**Files generated**:
```
.factory/commands/
├── openspec-proposal.md
├── openspec-apply.md
└── openspec-archive.md
```

### D6: project_changes MCP Tool

**Decision**: Add `project_changes` MCP tool as thin wrapper around `openspec list --json`

**Rationale**:
- OpenSpec CLI already provides structured JSON output
- No custom parsing needed - pass through CLI output
- Add session correlation on top of CLI output
- Enables change ordering/prioritization visibility

**Parameters**:
- `project_id` (required) - Project to get changes for

**Implementation**:
```go
// Execute: openspec list --json --sort name
// Parse JSON output and add session correlation
```

**Response Structure** (mirrors OpenSpec output + session info):
```json
{
  "project_id": "...",
  "changes": [
    {
      "name": "010-add-feature-x",
      "completedTasks": 4,
      "totalTasks": 10,
      "lastModified": "2025-01-12T10:30:00.000Z",
      "status": "in-progress",
      "active_sessions": ["sess_abc123"]
    }
  ]
}
```

**Ordering Convention**:
- Changes with numeric prefixes (010-, 020-) sorted numerically
- This determines build order priority
- No prefix = sorted alphabetically after numbered changes

### D7: project_tasks MCP Tool

**Decision**: Add `project_tasks` MCP tool as thin wrapper around `openspec instructions apply --json`

**Rationale**:
- OpenSpec CLI provides rich task information via `openspec instructions apply --json`
- Returns full task list with IDs, descriptions, completion status
- Includes progress summary and state (ready/blocked/all_done)
- No custom markdown parsing needed

**Parameters**:
- `project_id` (required) - Project to get tasks for
- `change_id` (required) - Which change to get tasks for

**Implementation**:
```go
// Execute: openspec instructions apply --change <change_id> --json
// Parse JSON output and add session correlation
```

**Response Structure** (mirrors OpenSpec output + session info):
```json
{
  "project_id": "...",
  "changeName": "add-feature-x",
  "schemaName": "spec-driven",
  "progress": {
    "total": 28,
    "complete": 4,
    "remaining": 24
  },
  "tasks": [
    {
      "id": "1",
      "description": "1.1 Create database schema",
      "done": false,
      "session_id": "sess_abc123"
    }
  ],
  "state": "ready",
  "active_sessions": [
    {
      "session_id": "sess_abc123",
      "status": "running"
    }
  ]
}
```

**Session Correlation**:
- Check `Session.TaskContext` for `change_id` and `task_id` fields
- Query active sessions from `ActiveSessionManager`
- Map sessions to tasks they're working on

### D8: Session Modes (Planning, Build, Interactive)

**Decision**: Add `mode` parameter to `session_message` with three modes

**Modes**:
- `interactive` (default) - Current behavior, agent responds to prompts
- `plan` - Agent instructed to create OpenSpec proposals
- `build` - Agent works through tasks until completion detected

**Rationale**:
- Separates intent clearly for different workflow stages
- Build mode enables Ralph-style autonomous loops
- Planning mode focuses agent on spec creation before implementation

**Parameters added to session_message**:
- `mode` (optional) - One of: `interactive`, `plan`, `build`
- `change_id` (optional) - Which change to work on (build mode only)

**Plan Mode Behavior**:
- Initial message automatically invokes `/openspec-proposal` with the user's request
- Agent follows the proposal workflow (scaffold change, validate strictly)
- No automatic completion detection - user reviews and approves

**Build Mode Behavior**:
- If `change_id` provided: builds that specific change
- If `change_id` omitted: builds ALL outstanding changes in order
- Initial message invokes `/openspec-apply <change_id>`
- Stop hook prevents premature exit until all tasks complete
- On change completion, stop hook advances to next change (if building all)
- Session ends when all targeted changes are complete

**Slash Command Integration**:
```
Plan Mode:
  User calls: session_message(mode="plan", message="Add user authentication")
  Agent receives: "/openspec-proposal Add user authentication"

Build Mode (single change):
  User calls: session_message(mode="build", change_id="add-auth")
  Agent receives: "/openspec-apply add-auth"
  
Build Mode (all changes):
  User calls: session_message(mode="build")
  Oubliette picks first incomplete change from `openspec list --json`
  Agent receives: "/openspec-apply 010-first-change"
  On completion, stop hook advances to next change automatically
```

This leverages the existing OpenSpec slash commands and enables fully autonomous build loops.

**Implementation**:
```go
type SessionMode string

const (
    SessionModeInteractive SessionMode = "interactive"
    SessionModePlan    SessionMode = "plan"
    SessionModeBuild       SessionMode = "build"
)
```

### D9: Build Mode Stop Hook

**Decision**: Use Factory Droid's Stop hook to prevent premature exit and re-prompt the agent

**Rationale**:
- Agents often stop before completing all tasks (decide they're "blocked", miscount, etc.)
- Stop hook intercepts exit attempts and can block + re-prompt
- Matches proven Ralph-wiggum pattern
- Hook runs in container, close to the agent

**Hook Location**: `template/.factory/hooks/build-mode-stop.sh`

**Hook Configuration** (in `.factory/settings.json`):
```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "\"$FACTORY_PROJECT_DIR\"/.factory/hooks/build-mode-stop.sh"
          }
        ]
      }
    ]
  }
}
```

**Hook Logic**:
```bash
#!/bin/bash
# Build Mode Stop Hook - prevents premature exit, handles verification, and advances through changes

set -euo pipefail

# Read hook input from stdin (includes transcript_path for output analysis)
HOOK_INPUT=$(cat)

# Check if we're in build mode (look for state file)
BUILD_STATE_FILE="$FACTORY_PROJECT_DIR/.factory/build-mode.json"
if [[ ! -f "$BUILD_STATE_FILE" ]]; then
  exit 0  # Not in build mode, allow exit
fi

# Get state from file
CHANGE_ID=$(jq -r '.change_id' "$BUILD_STATE_FILE")
BUILD_ALL=$(jq -r '.build_all // false' "$BUILD_STATE_FILE")
PHASE=$(jq -r '.phase // "build"' "$BUILD_STATE_FILE")
MAX_ITERATIONS=$(jq -r '.max_iterations // 100' "$BUILD_STATE_FILE")
ITERATION=$(jq -r '.iteration // 0' "$BUILD_STATE_FILE")
TASKS_MD="$FACTORY_PROJECT_DIR/openspec/changes/$CHANGE_ID/tasks.md"

# Check iteration limit
if [[ $ITERATION -ge $MAX_ITERATIONS ]]; then
  echo "Build mode: max iterations ($MAX_ITERATIONS) reached" >&2
  rm "$BUILD_STATE_FILE"
  exit 0  # Allow exit
fi

# Safety check: if stop_hook_active is true, we're already in a re-prompt loop
# This shouldn't happen with our iteration tracking, but provides defense in depth
STOP_HOOK_ACTIVE=$(echo "$HOOK_INPUT" | jq -r '.stop_hook_active // false')
if [[ "$STOP_HOOK_ACTIVE" == "true" && $ITERATION -gt 50 ]]; then
  echo "Build mode: stop_hook_active detected after 50+ iterations, allowing exit" >&2
  rm "$BUILD_STATE_FILE"
  exit 0
fi

# Helper: update state file
update_state() {
  local updates="$1"
  jq "$updates" "$BUILD_STATE_FILE" > "${BUILD_STATE_FILE}.tmp"
  mv "${BUILD_STATE_FILE}.tmp" "$BUILD_STATE_FILE"
}

# Helper: get last assistant output from transcript
get_last_output() {
  local transcript_path
  transcript_path=$(echo "$HOOK_INPUT" | jq -r '.transcript_path // empty')
  if [[ -n "$transcript_path" && -f "$transcript_path" ]]; then
    grep '"role":"assistant"' "$transcript_path" | tail -1 | jq -r '
      .message.content | map(select(.type == "text")) | map(.text) | join("\n")
    ' 2>/dev/null || echo ""
  fi
}

# Helper: check if tasks.md is stale (not modified since build started)
tasks_stale() {
  local started_at
  started_at=$(jq -r '.started_at // empty' "$BUILD_STATE_FILE")
  if [[ -z "$started_at" || ! -f "$TASKS_MD" ]]; then
    return 1  # Can't determine, assume not stale
  fi
  local started_epoch tasks_mtime
  started_epoch=$(date -j -f "%Y-%m-%dT%H:%M:%S" "${started_at%.*}" +%s 2>/dev/null || echo 0)
  tasks_mtime=$(stat -f %m "$TASKS_MD" 2>/dev/null || echo 0)
  [[ $tasks_mtime -lt $started_epoch ]]
}

# Helper: advance to next change or exit
advance_or_exit() {
  if [[ "$BUILD_ALL" == "true" ]]; then
    local next_change
    next_change=$(openspec list --json 2>/dev/null | jq -r '
      .changes | map(select(.status == "in-progress")) | .[0].name // empty
    ')
    
    if [[ -n "$next_change" ]]; then
      update_state --arg next "$next_change" '.change_id = $next | .phase = "build" | .iteration = 0'
      jq -n --arg change "$next_change" '{
        "decision": "block",
        "reason": "/openspec-apply \($change)"
      }'
      exit 0
    fi
  fi
  
  # No more changes - allow exit
  rm "$BUILD_STATE_FILE"
  exit 0
}

# PHASE: BUILD - check task completion
if [[ "$PHASE" == "build" ]]; then
  TASK_STATE=$(openspec instructions apply --change "$CHANGE_ID" --json 2>/dev/null | jq -r '.state')
  
  if [[ "$TASK_STATE" == "all_done" ]]; then
    # Check if tasks.md is stale (agent may have lied about completion)
    if tasks_stale; then
      update_state ".iteration = $((ITERATION + 1))"
      jq -n '{
        "decision": "block",
        "reason": "Agent is idle but tasks.md may not reflect completed work. Please update tasks.md to mark completed tasks before continuing."
      }'
      exit 0
    fi
    
    # Tasks complete - transition to verify phase
    update_state '.phase = "verify" | .iteration = 0'
    
    VERIFY_PROMPT="All tasks for change \"$CHANGE_ID\" are complete. Before archiving:

1. Run the project's build/compile step (check Makefile, build.sh, package.json, Cargo.toml, go.mod, etc.)
2. Run the project's test suite
3. Run any linters/typecheckers configured for the project (check AGENTS.md for guidance)
4. Fix any errors you find

When all builds pass and tests are green, output exactly: VERIFIED"

    jq -n --arg prompt "$VERIFY_PROMPT" '{
      "decision": "block",
      "reason": $prompt
    }'
    exit 0
  fi
  
  # Tasks remaining - re-prompt to continue
  PROGRESS=$(openspec instructions apply --change "$CHANGE_ID" --json 2>/dev/null | jq -r '"\(.progress.complete)/\(.progress.total)"')
  update_state ".iteration = $((ITERATION + 1))"
  
  jq -n --arg progress "$PROGRESS" --arg change "$CHANGE_ID" '{
    "decision": "block",
    "reason": "Tasks remaining: \($progress). Continue working on change: \($change). Check tasks.md for incomplete items and keep implementing."
  }'
  exit 0
fi

# PHASE: VERIFY - check for VERIFIED marker
if [[ "$PHASE" == "verify" ]]; then
  LAST_OUTPUT=$(get_last_output)
  
  if echo "$LAST_OUTPUT" | grep -q "VERIFIED"; then
    # Verification passed - archive and advance
    openspec archive "$CHANGE_ID" 2>/dev/null || true
    git add -A 2>/dev/null || true
    git commit -m "feat($CHANGE_ID): implementation complete" 2>/dev/null || true
    
    advance_or_exit
  fi
  
  # Not verified yet - re-prompt
  update_state ".iteration = $((ITERATION + 1))"
  jq -n --arg change "$CHANGE_ID" '{
    "decision": "block",
    "reason": "Verification not complete for change: \($change). Please ensure build passes, tests are green, and output VERIFIED when done."
  }'
  exit 0
fi
```

**Build Mode State File** (`$FACTORY_PROJECT_DIR/.factory/build-mode.json`):
```json
{
  "change_id": "010-current-change",
  "build_all": true,
  "phase": "build",
  "max_iterations": 100,
  "iteration": 0,
  "started_at": "2025-01-13T10:00:00Z"
}
```

**State File Fields**:
- `change_id` - Current change being worked on
- `build_all` - If true, advance to next change on completion
- `phase` - Current phase: `build` (implementing tasks) or `verify` (running tests)
- `max_iterations` - Safety limit (default 100)
- `iteration` - Current iteration count (resets to 0 on phase change)
- `started_at` - When build mode started (used for stale task detection)

**State File Lifecycle**:
1. Created by Oubliette when `session_message` called with `mode: "build"`
   - If `change_id` provided: `build_all: false`
   - If `change_id` omitted: `build_all: true`, pick first from `openspec list --json`
2. Updated by stop hook:
   - Increment iteration on each stop attempt
   - Update `change_id` when advancing to next change
3. Deleted when:
   - All targeted changes complete
   - Max iterations reached
   - Session explicitly ended via `session_end`

### D10: Build Mode Completion Detection

**Decision**: Completion detected via Stop hook checking OpenSpec state

**Rationale**:
- Stop hook already intercepts exit attempts
- Natural place to check completion
- Only allows exit when `state == "all_done"` or max iterations

**Detection Flow**:
1. Agent works on tasks
2. Agent tries to stop (decides it's done, blocked, etc.)
3. Stop hook fires, checks `openspec instructions apply --json`
4. If `state != "all_done"` and iterations remaining:
   - Block exit with `decision: "block"`
   - Re-prompt with "Tasks remaining: X/Y. Continue working..."
5. Agent receives the re-prompt and continues
6. Repeat until `state == "all_done"` or max iterations

**Alternative Considered**: Server-side polling after each turn
- Rejected: More complex, requires protocol changes
- Stop hook is simpler and proven (Ralph-wiggum pattern)

### D11: Verification Phase

**Decision**: After tasks complete, send verification prompt before archiving

**Rationale**:
- Agent already knows the project (just worked on it)
- Agent can discover build/test commands from project files
- Agent fixes its own mistakes
- No per-project configuration needed

**Verification Prompt**:
```
All tasks for change "<change_id>" are complete. Before archiving:

1. Run the project's build/compile step (check Makefile, build.sh, package.json, Cargo.toml, go.mod, etc.)
2. Run the project's test suite
3. Run any linters/typecheckers configured for the project (check AGENTS.md for guidance)
4. Fix any errors you find

When all builds pass and tests are green, output exactly: VERIFIED
```

**State File Phase Tracking**:
```json
{
  "change_id": "add-feature",
  "build_all": true,
  "phase": "build",
  "iteration": 5
}
```

Phases: `build` → `verify` → (next change or exit)

**Stop Hook Flow with Verification**:
1. Tasks incomplete (`state != "all_done"`) → re-prompt to continue tasks
2. Tasks complete, phase is `build` → transition to `verify`, send verification prompt
3. Phase is `verify`, no `VERIFIED` marker → re-prompt to continue verification
4. Phase is `verify`, `VERIFIED` detected → archive change, advance to next (or exit)

### D12: Task Reminders

**Decision**: Stop hook sends reminders when tasks.md appears stale

**Rationale**:
- Agents often claim completion without updating tasks.md
- Cassian uses this pattern successfully
- Prevents premature exit based on false completion claims

**Stale Detection**:
- Compare `tasks.md` mtime with last tool activity timestamp
- If tasks.md not modified since agent started working, it's stale

**Reminder Message**:
```
Agent is idle but tasks.md may not reflect completed work.
Please update tasks.md to mark completed tasks before continuing.
```

**Implementation**:
- Stop hook checks tasks.md mtime before allowing transition to verify phase
- If stale, blocks exit with reminder message
- Agent updates tasks.md, tries to exit again
- Stop hook re-checks, allows transition if tasks.md was updated

### D13: Archive on Completion

**Decision**: Run `openspec archive` after verification passes

**Rationale**:
- OpenSpec convention: completed changes move to `changes/archive/`
- Keeps active changes list clean
- Provides history of completed work

**Archive Flow**:
```bash
# After VERIFIED detected
openspec archive "$CHANGE_ID"
git add -A
git commit -m "feat($CHANGE_ID): implementation complete"
```

**Stop Hook Update**:
```bash
# After verification passes
if [[ "$PHASE" == "verify" ]] && echo "$LAST_OUTPUT" | grep -q "VERIFIED"; then
  # Archive the change
  openspec archive "$CHANGE_ID"
  
  # Commit (optional - could leave for user)
  git add -A
  git commit -m "feat($CHANGE_ID): implementation complete" || true
  
  # Advance to next change or exit
  ...
fi
```

### D14: Enhanced session_events with Child Sessions

**Decision**: Add `include_children` parameter to `session_events`

**Rationale**:
- Oubliette already tracks `parent_session_id` on sessions
- Users may want to see all events from a session tree
- Enables monitoring of delegated work

**Parameters added to session_events**:
- `include_children` (optional, default false) - Include events from child sessions

**Implementation**:
```go
// If include_children, query ActiveSessionManager for sessions with parent_session_id == sessionID
// Merge and sort events by timestamp
```

**Note**: This is an enhancement to existing functionality, not a new tool

## Directory Structure After Integration

```
template/
├── .factory/
│   ├── mcp.json
│   ├── droids/
│   ├── skills/
│   └── commands/
│       ├── openspec-proposal.md    # NEW
│       ├── openspec-apply.md       # NEW
│       └── openspec-archive.md     # NEW
└── openspec/                        # NEW
    ├── AGENTS.md
    ├── project.md
    ├── specs/
    └── changes/
        └── archive/

projects/<id>/
├── .factory/                        # Copied from template
├── openspec/                        # Copied from template
└── workspaces/<uuid>/
    └── (working directory, sees project's openspec/)
```

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Image size increase | Low | OpenSpec ~50MB (Node.js already installed) |
| Build time increase | Low | npm install adds ~10-20s to build |
| Template staleness | Low | Update committed templates when upgrading |
| Spec conflicts in shared workspace | Medium | Document as intentional behavior |

## Migration Plan

1. **Phase 1**: Update Dockerfile with Node.js and OpenSpec
2. **Phase 2**: Add template generation to Docker build
3. **Phase 3**: Update project creation to copy openspec/
4. **Phase 4**: Verify agent access to CLI and commands
5. **Phase 5**: Documentation updates

**Rollback**: Remove openspec from Dockerfile, templates still work but CLI unavailable.

## Open Questions

None - straightforward integration following existing patterns.
