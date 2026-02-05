## 1. Enable Runtime Override in Session Manager

- [x] 1.1 Add `RuntimeOverride agent.Runtime` field to `session.StartOptions`
- [x] 1.2 Update `CreateBidirectionalSession` to use `opts.RuntimeOverride` if set, otherwise `m.agentRuntime`
- [x] 1.3 Update `ResumeBidirectionalSession` to use `opts.RuntimeOverride` if set
- [x] 1.4 Update `Create` (single-turn) to use `opts.RuntimeOverride` if set

## 2. Expose Runtime Factory in MCP Server

- [x] 2.1 Add `RuntimeFactory` field to `ServerConfig` that can create runtimes by name
- [x] 2.2 Implement `GetRuntimeForProject(proj *project.Project) agent.Runtime` method on Server
- [x] 2.3 Method returns server default if project has no override, otherwise creates appropriate runtime

## 3. Wire Up Session Handlers

- [x] 3.1 Update `spawnAndRegisterSession` to call `GetRuntimeForProject` and pass to StartOptions
- [x] 3.2 Update `handleSendMessage` resume path to use project's runtime (N/A - sends to existing active session which already has runtime)
- [x] 3.3 Update socket handler child spawn to use project's runtime

## 4. Testing

- [x] 4.1 Add unit test: project with `agent_runtime: "opencode"` uses OpenCode runtime
- [x] 4.2 Add unit test: project with no `agent_runtime` uses server default
- [x] 4.3 Add integration test: create project with `agent_runtime`, verify parameter is accepted and runtimes are listed in project_options
