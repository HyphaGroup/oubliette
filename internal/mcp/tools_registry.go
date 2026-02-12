package mcp

// registerAllTools registers all MCP tools with the registry
func (s *Server) registerAllTools(r *Registry) {
	s.registerProjectTools(r)
	s.registerContainerTools(r)
	s.registerSessionTools(r)
	s.registerWorkspaceTools(r)
	s.registerConfigTools(r)
	s.registerTokenTools(r)
	s.registerScheduleTools(r)
}

func (s *Server) registerProjectTools(r *Registry) {
	Register(r, ToolDef{
		Name: "project",
		Description: `Manage projects — isolated environments backed by a Git repo and container.

Actions:
  create  — Create a project from a Git repo URL. Specify container_type (default: "base") and model.
  list    — List all projects. No parameters required.
  get     — Get project details by project_id. Returns config, status, container info.
  delete  — Delete a project and its data. Requires project_id.
  options — Show available container types and models with defaults.

Key parameters (create):
  repo_url        — Git repository URL (required for create)
  container_type  — Container image type: "base" or "dev" (default: "base")
  model           — LLM model for sessions. Use "options" action to see available models.
  description     — Human-readable project description
  name            — Display name (defaults to repo name)`,
		Target: TargetGlobal,
		Access: AccessWrite,
	}, s.handleProject)

	Register(r, ToolDef{
		Name: "project_changes",
		Description: `List OpenSpec changes for a project.

Returns change names, task counts, and status. Equivalent to "openspec list --json".
Requires project_id. Use to discover available change specs before applying tasks.`,
		Target: TargetProject,
		Access: AccessRead,
	}, s.handleProjectChanges)

	Register(r, ToolDef{
		Name: "project_tasks",
		Description: `Get task details for an OpenSpec change.

Returns the task list with completion status for a specific change. Equivalent to "openspec instructions apply --json".
Requires project_id and change_id. Use after project_changes to drill into a specific change.`,
		Target: TargetProject,
		Access: AccessRead,
	}, s.handleProjectTasks)
}

func (s *Server) registerContainerTools(r *Registry) {
	Register(r, ToolDef{
		Name: "container",
		Description: `Manage project containers — the Docker environments where agents execute.

Actions:
  start  — Start a project's container. Requires project_id.
  stop   — Stop a running container. Requires project_id.
  logs   — Get container logs. Requires project_id. Use tail (int) and since (duration like "1h") to filter.
  exec   — Execute a command inside the container. Requires project_id and command (string array).

Containers auto-start when sessions spawn. Use "start" to pre-warm, "exec" to debug.`,
		Target: TargetProject,
		Access: AccessWrite,
	}, s.handleContainer)

	Register(r, ToolDef{
		Name: "container_refresh",
		Description: `Pull the latest container image and optionally restart a project's container.

Specify container_type (e.g. "base", "dev") to pull a specific image, or project_id to refresh
a project's container. Fails if active sessions exist — end them first.`,
		Target: TargetGlobal,
		Access: AccessWrite,
	}, s.handleContainerRefresh)
}

func (s *Server) registerSessionTools(r *Registry) {
	Register(r, ToolDef{
		Name: "session",
		Description: `Manage autonomous agent sessions. Sessions run OpenCode inside containers.

Actions:
  message  — Send a task to an agent. Auto-creates session if none active. Most common action.
  spawn    — Create or resume a session explicitly. Use new_session=true to force fresh session.
  get      — Get session details, turns, and token costs by session_id.
  list     — List sessions for a project. Filter by status (active/completed/failed).
  events   — Poll streaming events by session_id. Use since_index for pagination.
  end      — End a session by session_id.
  cleanup  — Delete old sessions. Optionally filter by project_id and max_age_hours (default: 24).

Key behaviors:
  - Sessions auto-resume: sending a message reuses the active session for that project/workspace.
  - Use new_session=true on spawn to force a fresh session.
  - model/autonomy_level/reasoning_level default to project config if not specified.
  - Events are also pushed via MCP notifications. Use events action to poll or catch up.`,
		Target: TargetProject,
		Access: AccessWrite,
	}, s.handleSession)

	Register(r, ToolDef{
		Name: "caller_tool_response",
		Description: `Respond to a caller_tool_request event with the tool execution result.

When a child session calls a tool exposed by its parent (via caller_tools), the parent receives
a caller_tool_request notification. Use this tool to send the result back.
Requires session_id, tool_call_id, and result (string). Set is_error=true if the tool call failed.`,
		Target: TargetProject,
		Access: AccessWrite,
	}, s.handleCallerToolResponse)
}

func (s *Server) registerWorkspaceTools(r *Registry) {
	Register(r, ToolDef{
		Name: "workspace",
		Description: `Manage workspaces — isolated working directories within a project's container.

Actions:
  list    — List workspaces for a project. Requires project_id.
  delete  — Delete a workspace. Requires project_id and workspace_id. Fails if sessions are active.

Each session runs in a workspace. Workspaces persist between sessions for continuity.
Use external_id and source on spawn/message to correlate workspaces with external systems (PRs, tickets).`,
		Target: TargetProject,
		Access: AccessWrite,
	}, s.handleWorkspace)
}

func (s *Server) registerConfigTools(r *Registry) {
	Register(r, ToolDef{
		Name: "config_limits",
		Description: `Get recursion limits and current depth for a project or session.

Returns max_depth, max_agents, max_cost, and current depth. Use to check how deep child sessions
can nest before hitting limits. Requires project_id. Optionally pass session_id for session-specific depth.`,
		Target: TargetProject,
		Access: AccessRead,
	}, s.handleGetRecursionLimits)
}

func (s *Server) registerTokenTools(r *Registry) {
	Register(r, ToolDef{
		Name: "token",
		Description: `Manage API tokens for MCP authentication. Requires admin scope.

Actions:
  create  — Create a new token. Specify scope: "admin", "admin:ro", "project:<id>", "project:<id>:ro".
  list    — List all tokens with metadata (scope, created date, last used).
  revoke  — Revoke a token by token_id.

Tokens authenticate MCP clients. Project-scoped tokens restrict access to one project.`,
		Target: TargetGlobal,
		Access: AccessAdmin,
	}, s.handleToken)
}

func (s *Server) registerScheduleTools(r *Registry) {
	Register(r, ToolDef{
		Name: "schedule",
		Description: `Manage scheduled tasks — cron-based recurring agent sessions.

Actions:
  create   — Create a schedule. Requires name, cron_expr, prompt (task text), and targets (project list).
  list     — List all schedules. Optionally filter by project_id.
  get      — Get schedule details by schedule_id.
  update   — Update a schedule. Pass only fields to change.
  delete   — Delete a schedule by schedule_id.
  trigger  — Run a schedule immediately, ignoring cron timing.
  history  — View execution history for a schedule. Optionally limit results.

Schedules spawn sessions on a cron cadence. Set session_behavior to "resume" to reuse the same
session across runs, or "new" to create a fresh session each time.`,
		Target: TargetGlobal,
		Access: AccessWrite,
	}, s.handleSchedule)
}
