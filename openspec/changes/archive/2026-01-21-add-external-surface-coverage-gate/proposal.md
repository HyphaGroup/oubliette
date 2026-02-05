# Change: Add External Surface Coverage Gate

## Why

We need a fast, repeatable pre-commit/approval gate that verifies all external interfaces (MCP tools, CLI binaries, manager commands) have test coverage.

The project exposes functionality to external consumers through three interfaces:
1. **MCP tools** (25 tools used by Ant and other MCP clients)
2. **CLI binaries** (`oubliette-server`, `oubliette-token`, `oubliette-client`, `oubliette-relay`)
3. **Manager script** (`scripts/manager.sh` with 12 subcommands)

Currently, only MCP tools have automated coverage tracking (92% covered, missing `project_options` and `caller_tool_response`). CLI binaries and manager commands have no coverage enforcement, meaning regressions in these interfaces go undetected until production use.

## What Changes

- Extend the coverage analyzer to track all three external surface areas
- Use `registry.GetAllTools()` for MCP tool discovery (from `add-unified-tool-registry`)
- Add missing tests to achieve 100% coverage across all interfaces
- Create an enforceable gate command (`go run . --coverage-report`) that fails if any external interface lacks test coverage
- Add environment variable overrides to `manager.sh` so tests can run against temporary directories

**Depends on:** `add-unified-tool-registry` (for `registry.GetAllTools()`)

## Scope

In scope:
- Coverage gate for: MCP tools (25), CLI binaries (4), Manager commands (12).
- Integration tests that exercise each interface at least once.

Out of scope:
- Go line/branch coverage percentages.
- Existing test infrastructure changes.
- Manager.sh behavior in production use.
- CI pipeline changes (this gate is for local pre-commit use).

## Impact

- **Affected specs**: `testing-coverage`
- **Affected code**:
  - `test/pkg/coverage/` - Extend analyzer
  - `test/pkg/testing/testcase.go` - Add `Covers` metadata field
  - `test/pkg/suites/` - New tests for missing coverage
  - `scripts/manager.sh` - Add env var overrides
- **Breaking changes**: None (additive only)
