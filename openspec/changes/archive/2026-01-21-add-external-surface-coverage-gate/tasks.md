## 1. Coverage Analyzer Infrastructure

- [x] 1.1 Update MCP tool discovery to use `registry.GetAllTools()` (depends on `add-unified-tool-registry`) - already works via client.ListTools() which queries server's registry
- [x] 1.2 Add manager command discovery by parsing `scripts/manager.sh` usage output
- [x] 1.3 Add CLI binary discovery by scanning repo root for `oubliette-*`
- [x] 1.4 Extend `CoverageReport` to include all three categories (MCP, Manager, CLI)
- [x] 1.5 Update `PrintReport()` to display all categories with totals and per-category breakdowns

## 2. Test Coverage Metadata

- [x] 2.1 Add `Covers []string` field to `TestCase` struct
- [x] 2.2 Update analyzer to scan for `Covers` field in test definitions
- [x] 2.3 Format: `mcp:<tool>`, `manager:<command>`, `cli:<binary>`

## 3. Manager Environment Overrides

- [x] 3.1 Add `OUBLIETTE_INSTANCES_DIR` and `OUBLIETTE_RELEASES_DIR` env overrides to `scripts/manager.sh`
- [x] 3.2 Ensure all directory references use overridable variables (RELEASES_DIR and INSTANCES_DIR are already used throughout)

## 4. Add Missing Tests

- [x] 4.1 Add test for `project_options` and `caller_tool_response`
- [x] 4.2 Create `test/pkg/suites/manager.go` with temp dir helper
- [x] 4.3 Add manager tests: `create`, `start`, `stop`, `restart`, `status`, `logs`, `delete`, `update`, `rollback`, `prune-releases`, `init-config`, `rebuild-images`
- [x] 4.4 Add CLI tests for `oubliette-server`, `oubliette-token`, `oubliette-client`, `oubliette-relay`

## 5. Validation

- [x] 5.1 Run `--coverage-report` and verify 100% across all categories - VERIFIED (42/42 items at 100%)
- [x] 5.2 Run full integration test suite - 62/71 pass, 9 pre-existing failures unrelated to coverage gate
