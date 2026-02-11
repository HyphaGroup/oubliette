# Change: Simplify Session Tool

## Why

The `session` tool has 30+ parameters across 7 actions. Many are dead (Droid-era mode system), several are redundant, and the tool description gives agents zero guidance on what params actually do or what defaults apply. An agent seeing this tool for the first time has to guess.

## Current State: Full Parameter Audit

### Action: `spawn`

| Parameter | What it does | Verdict |
|-----------|-------------|---------|
| `project_id` | Required. Identifies project. | **Keep** |
| `prompt` | Required. Initial task for the agent. Becomes the first message sent to OpenCode. | **Keep, rename to `message`** (see below) |
| `workspace_id` | Optional. Selects workspace. Defaults to project's default workspace. | **Keep** |
| `create_workspace` | Creates workspace if missing or if workspace_id is empty. | **Keep** |
| `new_session` | If true, skip session resume and force a new session. Default: false (tries to resume latest). | **Keep** |
| `model` | Override model for this session. Defaults to project model from oubliette.jsonc. | **Keep** |
| `autonomy_level` | Maps to OpenCode permission level. Defaults to config. Values: off/low/medium/high. | **Keep** |
| `reasoning_level` | Sent as OpenCode `variant` per-message. Values: off/low/medium/high. | **Keep** |
| `tools_allowed` | Whitelist of tools the agent can use. | **Keep** |
| `tools_disallowed` | Blacklist of tools. | **Keep** |
| `append_system_prompt` | Appended to system prompt. For child sessions, this is the primary way to inject depth/context. For prime sessions, it's rarely used. | **Rename to `system_prompt`** |
| `context` | Arbitrary key/value map. Stored on session metadata as `TaskContext`. Never read by any runtime logic. Only consumed by child session `.rlm-context/` file writes. | **Keep for child spawns, document clearly** |
| `external_id` | Caller-provided ID for workspace correlation (e.g., PR number, ticket ID). | **Keep** |
| `source` | Label for workspace origin (e.g., "github", "linear"). | **Keep** |
| `use_spec` | **DEAD.** Threaded through 6 types but never read by OpenCode runtime. | **Remove** |

### Action: `message`

| Parameter | What it does | Verdict |
|-----------|-------------|---------|
| `project_id` | Required. | **Keep** |
| `message` | Required. Text to send to the agent. | **Keep** |
| `workspace_id` | Optional. Resolves which active session to target. | **Keep** |
| `create_workspace` | Same as spawn. | **Keep** |
| `external_id` | Same as spawn. | **Keep** |
| `source` | Same as spawn. | **Keep** |
| `context` | Stored on TaskContext. | **Keep** |
| `model` | Override model for this message. | **Keep** |
| `autonomy_level` | Override for this message. | **Keep** |
| `reasoning_level` | Override for this message. | **Keep** |
| `tools_allowed` | Override for this message. | **Keep** |
| `tools_disallowed` | Override for this message. | **Keep** |
| `append_system_prompt` | Appended to system prompt if new session spawned. Ignored for existing sessions. | **Rename to `system_prompt`, document behavior** |
| `caller_id` | ID of the caller agent (for tool relay). | **Keep** |
| `caller_tools` | Tool definitions the caller exposes to the spawned agent. | **Keep** |
| `attachments` | File attachments (base64 or URL). | **Keep** |
| `mode` | **DEAD.** Prepends Droid slash-commands (`/openspec-proposal`, `/openspec-apply`). OpenCode doesn't understand these. | **Remove** |
| `change_id` | **DEAD.** Only used by mode=build. | **Remove** |
| `build_all` | **DEAD.** Only used by mode=build. | **Remove** |

### Action: `get`

| Parameter | What it does | Verdict |
|-----------|-------------|---------|
| `session_id` | Required. Returns session details, turns, cost. | **Keep** |

### Action: `list`

| Parameter | What it does | Verdict |
|-----------|-------------|---------|
| `project_id` | Required. | **Keep** |
| `status` | Filter by status (active/completed/failed). | **Keep** |

### Action: `end`

| Parameter | What it does | Verdict |
|-----------|-------------|---------|
| `session_id` | Required. Ends session. | **Keep** |

### Action: `events`

| Parameter | What it does | Verdict |
|-----------|-------------|---------|
| `session_id` | Required. | **Keep** |
| `since_index` | Return events after this index. For polling. | **Keep** |
| `max_events` | Limit event count. | **Keep** |
| `include_children` | Include events from child sessions. | **Keep** |

### Action: `cleanup`

| Parameter | What it does | Verdict |
|-----------|-------------|---------|
| `project_id` | Optional. If empty, cleans all projects. | **Keep** |
| `max_age_hours` | Delete sessions older than this. Default: 24. | **Keep** |

## What Changes

### 1. Remove Dead Parameters (5 params)

- **`mode`** — Droid slash-command system. Remove from `SendMessageParams`, `SessionParams`.
- **`change_id`** — Build mode only. Remove.
- **`build_all`** — Build mode only. Remove.
- **`use_spec`** — Never read by OpenCode. Remove from `SpawnParams`, `SessionParams`, `StartOptions`, `ExecuteRequest`.
- Remove `transformMessageForMode()`, `getFirstIncompleteChange()`, `createBuildModeStateFile()`, `BuildModeState`, `SessionMode` type and constants.

### 2. Rename for Clarity (2 params)

- **`prompt` → `message`** on spawn action. The spawn "prompt" is the same thing as a "message" -- it's the first text sent to the agent. Having both `prompt` (on spawn) and `message` (on message action) for the same concept is confusing. Unify on `message`. Keep `prompt` as an alias for backwards compat for one release cycle, then remove.
- **`append_system_prompt` → `system_prompt`**. The "append" prefix is an implementation detail. Callers just want to set additional system prompt text.

### 3. Merge `spawn` into `message` (Action Consolidation)

Currently, `message` already auto-spawns if no active session exists. The only thing `spawn` adds is:
- Session resume logic (`new_session` flag)
- The explicit "session created" response format

Proposal: Keep both actions but document that `message` is the **primary** action for most use cases. `spawn` is only needed when you want to explicitly control session creation without sending work.

### 4. Add Rich Tool Description

Replace the current one-liner description with a structured description that helps agents understand what the tool does:

```
Manage autonomous agent sessions. Sessions run OpenCode inside containers.

Actions:
  message  - Send a task to an agent. Auto-creates session if needed. (most common)
  spawn    - Create/resume a session without sending work yet.
  get      - Get session details, turns, and token costs.
  list     - List sessions for a project. Filter by status.
  events   - Poll streaming events (tool calls, completions, errors).
  end      - End a session.
  cleanup  - Delete old sessions. Default: older than 24h.

Key behaviors:
  - Sessions auto-resume: sending a message to a project reuses the active session.
  - Set new_session=true on spawn to force a fresh session.
  - model defaults to the project's configured model.
  - system_prompt is appended to the agent's system prompt (useful for task-specific context).
  - Events are pushed via SSE notifications. Use events action to poll if needed.
```

### 5. Clean Up Dead Internal Code

Remove all code supporting the dead parameters:
- `transformMessageForMode()` function
- `getFirstIncompleteChange()` function  
- `createBuildModeStateFile()` function
- `BuildModeState` struct
- `SessionMode` type and constants (`ModeInteractive`, `ModePlan`, `ModeBuild`)
- `readFinalResponseFromSession()` function (reads from `.factory/sessions/` -- dead Droid path)
- `FinalResponse` field on `SessionEventsResult`
- Build-mode auto-selection path in `handleSendMessage`
- `TaskContext` struct on `ActiveSession` (the `Mode`/`ChangeID`/`BuildAll` fields; keep `SetTaskContext` for the generic `context` map)

## Impact

- **Breaking**: `prompt` field renamed to `message` on spawn (keep alias temporarily)
- **Breaking**: `mode`, `change_id`, `build_all`, `use_spec` removed (dead params, no real callers)
- **Non-breaking**: `append_system_prompt` renamed to `system_prompt` (keep alias temporarily)
- Net reduction: ~200 lines of dead code
- Better agent UX: clear tool description, fewer confusing params
