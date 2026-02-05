# Proposal: Add Scheduled Tasks

## Summary

Add the ability to create scheduled/recurring tasks that execute prompts against projects and workspaces on a cron schedule. Schedules are managed via MCP tools and can target multiple project/workspace combinations.

## Motivation

Currently, all agent sessions are triggered manually via `session_spawn` or `session_message`. There's no way to:
- Run periodic maintenance tasks (e.g., daily code review, weekly dependency updates)
- Schedule recurring data processing or report generation
- Trigger agents based on time rather than external events

## Proposed Solution

### Schedule Model

A schedule consists of:
- **Cron expression** - Standard 5-field cron (`minute hour day month weekday`)
- **Prompt** - The message to send to the agent
- **Targets** - One or more project/workspace pairs
- **Overlap behavior** - What to do if previous run is still active: `skip`, `queue`, or `parallel`
- **Session behavior** - `resume` (default, resume existing or spawn new) or `new` (always fresh session)
- **Enabled flag** - Can be paused/resumed

### MCP Tools

| Tool | Description |
|------|-------------|
| `schedule_create` | Create a new scheduled task |
| `schedule_list` | List schedules (all if admin, filtered by project scope) |
| `schedule_get` | Get schedule details |
| `schedule_update` | Update schedule (cron, prompt, targets, enabled, overlap behavior) |
| `schedule_delete` | Delete a schedule |
| `schedule_trigger` | Manually trigger a scheduled task immediately |

### Storage

File-based storage in `data/schedules/`:
```
data/schedules/
├── index.json          # Schedule metadata index
└── <schedule_id>.json  # Individual schedule definitions
```

### Execution

- Background goroutine runs a scheduler loop
- On trigger: calls `session_message` internally for each target
- Respects overlap behavior setting
- Logs execution to server logs
- Emits structured log entries for observability

### Authorization

- Schedules inherit the scope of the creating token
- `admin` scope: can create schedules for any project
- `project:<uuid>` scope: can only create schedules targeting that project
- Listing shows only schedules the token has access to

## Out of Scope

- Web UI for schedule management (CLI/MCP only)
- Execution history persistence (logs only for now)
- Complex dependencies between scheduled tasks
- Timezone configuration (UTC only initially)

## Benefits

1. **Automation** - Periodic tasks without external cron/scheduler
2. **Multi-target** - One schedule can trigger multiple workspaces
3. **Flexible overlap** - Choose skip/queue/parallel per schedule
4. **Consistent auth** - Uses existing token scope model
