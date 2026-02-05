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
	// Unified project tool with action parameter
	Register(r, ToolDef{
		Name:        "project",
		Description: "Manage projects. Actions: create, list, get, delete, options",
		Target:      TargetGlobal,
		Access:      AccessWrite,
	}, s.handleProject)

	// Standalone tools that don't fit the CRUD pattern
	Register(r, ToolDef{
		Name:        "project_changes",
		Description: "List OpenSpec changes for a project. Returns change names, task counts, and status from openspec list --json",
		Target:      TargetProject,
		Access:      AccessRead,
	}, s.handleProjectChanges)

	Register(r, ToolDef{
		Name:        "project_tasks",
		Description: "Get task details for an OpenSpec change. Returns task list with completion status from openspec instructions apply --json",
		Target:      TargetProject,
		Access:      AccessRead,
	}, s.handleProjectTasks)
}

func (s *Server) registerContainerTools(r *Registry) {
	// Unified container tool with action parameter
	Register(r, ToolDef{
		Name:        "container",
		Description: "Manage containers. Actions: start, stop, logs, exec",
		Target:      TargetProject,
		Access:      AccessWrite,
	}, s.handleContainer)

	// Container refresh - pull latest images and restart containers
	Register(r, ToolDef{
		Name:        "container_refresh",
		Description: "Refresh container images. Pulls the latest image version. Specify container_type to pull a specific type, or project_id to refresh a project's container (fails if active sessions exist).",
		Target:      TargetGlobal,
		Access:      AccessWrite,
	}, s.handleContainerRefresh)
}

func (s *Server) registerSessionTools(r *Registry) {
	// Unified session tool with action parameter
	Register(r, ToolDef{
		Name:        "session",
		Description: "Manage sessions. Actions: spawn, message, get, list, end, events, cleanup",
		Target:      TargetProject,
		Access:      AccessWrite,
	}, s.handleSession)

	// Standalone tool for caller relay
	Register(r, ToolDef{
		Name:        "caller_tool_response",
		Description: "Respond to a caller_tool_request event with the tool execution result",
		Target:      TargetProject,
		Access:      AccessWrite,
	}, s.handleCallerToolResponse)
}

func (s *Server) registerWorkspaceTools(r *Registry) {
	// Unified workspace tool with action parameter
	Register(r, ToolDef{
		Name:        "workspace",
		Description: "Manage workspaces. Actions: list, delete",
		Target:      TargetProject,
		Access:      AccessWrite,
	}, s.handleWorkspace)
}

func (s *Server) registerConfigTools(r *Registry) {
	Register(r, ToolDef{
		Name:        "config_limits",
		Description: "Get recursion limits and depth information for project or session",
		Target:      TargetProject,
		Access:      AccessRead,
	}, s.handleGetRecursionLimits)
}

func (s *Server) registerTokenTools(r *Registry) {
	// Unified token tool with action parameter
	Register(r, ToolDef{
		Name:        "token",
		Description: "Manage API tokens. Actions: create, list, revoke. Requires admin scope.",
		Target:      TargetGlobal,
		Access:      AccessAdmin,
	}, s.handleToken)
}

func (s *Server) registerScheduleTools(r *Registry) {
	// Unified schedule tool with action parameter
	Register(r, ToolDef{
		Name:        "schedule",
		Description: "Manage scheduled tasks. Actions: create, list, get, update, delete, trigger",
		Target:      TargetGlobal,
		Access:      AccessWrite,
	}, s.handleSchedule)
}
