# Testing Strategy

This project uses **spec-driven development** via OpenSpec. Tests verify acceptance criteria from proposals, not implementation details.

## Test Pyramid

```
                    ┌─────────────┐
                    │   Smoke     │  ← Post-deploy sanity (2-3 tests)
                    │   Tests     │     "Does the system work at all?"
                    └──────┬──────┘
                           │
              ┌────────────┴────────────┐
              │    Integration Tests    │  ← Primary focus (100% coverage)
              │    (test/pkg/suites/)   │     Maps to spec acceptance criteria
              └────────────┬────────────┘
                           │
    ┌──────────────────────┴──────────────────────┐
    │              Unit Tests (selective)          │  ← Only for pure logic
    │         (internal/*/_test.go)                │     with complex edge cases
    └──────────────────────────────────────────────┘
```

## When to Write Each Test Type

| Test Type | When to Write | Examples |
|-----------|---------------|----------|
| **Integration** | Every new MCP tool, every spec acceptance criterion | `test_project_create_with_autonomy_param` |
| **Unit** | Pure logic with many edge cases, complex algorithms | Config translation, cron parsing, validation |
| **Smoke** | Critical path verification | Create project → spawn session → get response |
| **Load/Chaos** | Pre-release, performance concerns | Concurrent sessions, failure recovery |

## What NOT to Unit Test

- MCP handlers (covered by integration tests)
- Manager methods (covered via handler integration tests)
- Runtime-specific code (docker, applecontainer) - hard to mock, integration tests verify
- Simple CRUD operations

## OpenSpec → Test Workflow

When implementing an OpenSpec proposal:

```
openspec/changes/<change-id>/
├── proposal.md      ← Acceptance criteria defined here
├── tasks.md         ← Implementation tasks
└── (implementation)
         │
         ▼
┌─────────────────────────────────────────────────────┐
│  For each acceptance criterion in proposal.md:      │
│                                                     │
│  1. Write integration test in test/pkg/suites/     │
│  2. Test should call MCP tool and verify behavior  │
│  3. Run: cd test/cmd && go run . --test            │
│  4. Verify 100% coverage: go run . --coverage-report│
└─────────────────────────────────────────────────────┘
```

**Example**: If proposal says "autonomy parameter must accept values: off, low, medium, high":

```go
// test/pkg/suites/project.go
{
    Name:        "test_project_create_autonomy_validation",
    Description: "Test autonomy parameter accepts valid values",
    Tags:        []string{"project", "config"},
    Execute: func(ctx *testpkg.TestContext) error {
        for _, autonomy := range []string{"off", "low", "medium", "high"} {
            result, _ := ctx.Client.InvokeTool("project_create", map[string]any{
                "name":     fmt.Sprintf("test-autonomy-%s", autonomy),
                "autonomy": autonomy,
            })
            ctx.Assertions.AssertFalse(result.IsError, "Should accept autonomy=%s", autonomy)
        }
        return nil
    },
}
```

## Running Tests

```bash
# Integration tests (primary)
cd test/cmd && go run . --test

# Coverage report (must be 100%)
cd test/cmd && go run . --coverage-report

# Unit tests
go test ./... -short

# Smoke test (manual, post-deploy)
./scripts/smoke-test.sh

# Load tests (pre-release)
go test -v -tags=load ./test/load/... -timeout 45m

# Chaos tests (isolated environment)
go test -v -tags=chaos ./test/chaos/... -timeout 30m
```

## Coverage Requirements

| Metric | Target | Enforcement |
|--------|--------|-------------|
| MCP Tool Coverage | 100% | CI gate via `--coverage-report` |
| CLI Binary Coverage | 100% | CI gate via `--coverage-report` |
| Unit Test Coverage | No target | Not enforced (selective by design) |

## Integration Test Organization

Tests organized by functionality in `test/pkg/suites/`:

| File | Purpose |
|------|---------|
| `basic.go` | Smoke tests |
| `project.go` | Project lifecycle and config |
| `workspace.go` | Workspace management |
| `container.go` | Container operations |
| `session.go` | Agent session lifecycle |
| `recursion.go` | Depth tracking |
| `messaging.go` | Interactive features |
| `auth.go` | Authentication and tokens |
| `schedule.go` | Scheduled tasks |
| `openspec.go` | OpenSpec integration |
| `comprehensive.go` | End-to-end workflows |
| `cli.go` | CLI binary tests |
| `manager.go` | Manager script tests |

## Writing Integration Tests

```go
{
    Name:        "test_feature_scenario",
    Description: "Test that feature does X correctly",
    Tags:        []string{"category", "subcategory"},
    Timeout:     60 * time.Second,
    Execute: func(ctx *testpkg.TestContext) error {
        // Use helper methods
        err := ctx.CreateProject(projName, "Test project")
        ctx.Assertions.AssertNoError(err, "Should create project")

        // Execute and verify
        result, err := ctx.Client.InvokeTool("tool_name", params)
        ctx.Assertions.AssertFalse(result.IsError, "Should succeed")

        return nil
    },
}
```

## Coverage Tracking

**100% external surface coverage** - Every external interface must have at least one integration test:

1. **MCP Tools** - Discovered via `client.ListTools()` from the registry
2. **CLI Binaries** - Discovered by scanning repo root for `oubliette-*` executables

Check coverage: `cd test/cmd && go run . --coverage-report`

The coverage report exits with code 1 if coverage is below 100%, making it suitable as a CI gate.

**Adding Coverage for New Items:**
- MCP tools: Tests that call `InvokeTool("tool_name", ...)` are automatically detected
- CLI binaries: Add `Covers: []string{"cli:binary_name"}` to TestCase

## Unit Tests (Selective)

Only write unit tests for:

1. **Pure functions with complex logic** - config translation, validation
2. **Edge cases hard to trigger via integration** - boundary conditions, error paths
3. **Algorithms** - cron parsing, event buffering, rate limiting

Example (config translation has many edge cases):

```go
// internal/agent/config/opencode_test.go
func TestTranslateAutonomyToOpenCode(t *testing.T) {
    tests := []struct {
        name     string
        autonomy string
        want     map[string]string
    }{
        {"off", "off", map[string]string{"*": "allow"}},
        {"high", "high", map[string]string{"*": "allow", "external_directory": "ask"}},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := TranslateAutonomyToOpenCode(tt.autonomy)
            // assertions
        })
    }
}
```
