## Context

Oubliette exposes functionality through three external interfaces:
1. **MCP tools** (25 tools used by Ant and other MCP clients)
2. **CLI binaries** (`oubliette-server`, `oubliette-token`, `oubliette-client`, `oubliette-relay`)
3. **Manager script** (`scripts/manager.sh` with 12 subcommands)

Relays and agents depend on these interfaces remaining functional. Currently only MCP tools have partial coverage tracking.

## Goals

- Provide a fast local gate that can be run before commit/approval.
- Enforce 100% test coverage for all external surface areas (every tool, command, and binary has at least one test).

## Non-Goals

- Enforce Go line/branch coverage percentages.
- Replace existing unit tests.
- Change manager.sh behavior in production.

## Definition: External surface (coverage scope)

Interfaces covered at 100%:
- **MCP tools**: All 25 tools registered with the MCP server
- **CLI binaries**: `oubliette-server`, `oubliette-token`, `oubliette-client`, `oubliette-relay`
- **Manager commands**: All subcommands in `scripts/manager.sh`

Notes:
- Coverage is "at least one test exercises this interface", not line coverage.
- Tests declare coverage via `Covers` metadata field with format: `mcp:<tool>`, `cli:<binary>`, `manager:<command>`.
- Manager tests use env var overrides to run against temporary directories.

## Gate command

Preferred local command:
- `go run . --coverage-report`

It SHALL:
1. Discover all external interfaces automatically (no hardcoded lists).
2. Scan tests for `Covers` annotations.
3. Report coverage per category (MCP, CLI, Manager).
4. Exit non-zero if any interface lacks test coverage.

## Risks / Trade-offs

- Adding `Covers` annotations requires discipline; new tests must declare what they cover.
- Manager tests require env var overrides, adding complexity to test setup.

Mitigation:
- Gate command fails loudly when coverage is missing.
- Temporary directory helpers abstract env var setup.
