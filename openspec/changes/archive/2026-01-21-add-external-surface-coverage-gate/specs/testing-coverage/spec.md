## ADDED Requirements

### Requirement: External Surface Discovery

The coverage analyzer SHALL automatically discover all externally callable interfaces without manual maintenance of lists.

#### Scenario: MCP tools discovered from registry

- **WHEN** coverage analyzer runs
- **THEN** it calls `registry.GetAllTools()` to enumerate registered tools
- **AND** all registered tools are included in the coverage report

#### Scenario: Manager commands discovered from usage

- **WHEN** coverage analyzer runs
- **THEN** it parses `scripts/manager.sh` usage output
- **AND** all commands listed under "Commands:" are included in the coverage report

#### Scenario: CLI binaries discovered from repo root

- **WHEN** coverage analyzer runs
- **THEN** it scans the repository root for `oubliette-*` executables
- **AND** all found binaries are included in the coverage report

### Requirement: Coverage Annotation

Tests SHALL declare which external interfaces they cover using a structured annotation format.

#### Scenario: Test declares MCP tool coverage

- **GIVEN** a test case with `Covers: []string{"mcp:project_options"}`
- **WHEN** coverage analyzer scans the test
- **THEN** `project_options` is marked as covered in MCP tools section

#### Scenario: Test declares manager command coverage

- **GIVEN** a test case with `Covers: []string{"manager:create"}`
- **WHEN** coverage analyzer scans the test
- **THEN** `create` is marked as covered in manager commands section

#### Scenario: Test declares CLI binary coverage

- **GIVEN** a test case with `Covers: []string{"cli:oubliette-token"}`
- **WHEN** coverage analyzer scans the test
- **THEN** `oubliette-token` is marked as covered in CLI binaries section

#### Scenario: Test declares multiple coverage items

- **GIVEN** a test case with `Covers: []string{"mcp:project_create", "mcp:project_delete"}`
- **WHEN** coverage analyzer scans the test
- **THEN** both `project_create` and `project_delete` are marked as covered

### Requirement: Coverage Report Categories

The coverage report SHALL display separate coverage statistics for each external interface category.

#### Scenario: Report shows all categories

- **WHEN** coverage report is generated
- **THEN** it displays separate sections for MCP tools, manager commands, and CLI binaries
- **AND** each section shows total count, covered count, and percentage

#### Scenario: Report lists uncovered items per category

- **WHEN** coverage report is generated
- **AND** some items lack test coverage
- **THEN** uncovered items are listed under their respective category heading

### Requirement: Coverage Gate

The coverage report command SHALL exit with non-zero status when any external interface lacks test coverage.

#### Scenario: Gate passes with full coverage

- **GIVEN** all MCP tools have at least one test
- **AND** all manager commands have at least one test
- **AND** all CLI binaries have at least one test
- **WHEN** `go run . --coverage-report` is executed
- **THEN** exit code is 0

#### Scenario: Gate fails with missing MCP coverage

- **GIVEN** one MCP tool has no tests
- **WHEN** `go run . --coverage-report` is executed
- **THEN** exit code is non-zero
- **AND** the missing tool is listed in output

#### Scenario: Gate fails with missing manager coverage

- **GIVEN** one manager command has no tests
- **WHEN** `go run . --coverage-report` is executed
- **THEN** exit code is non-zero
- **AND** the missing command is listed in output

#### Scenario: Gate fails with missing CLI coverage

- **GIVEN** one CLI binary has no tests
- **WHEN** `go run . --coverage-report` is executed
- **THEN** exit code is non-zero
- **AND** the missing binary is listed in output

### Requirement: Manager Test Isolation

Manager command tests SHALL run against isolated temporary directories to prevent interference with real instances.

#### Scenario: Tests use temporary instances directory

- **GIVEN** `OUBLIETTE_INSTANCES_DIR` is set to a temporary path
- **WHEN** a manager test runs `create test-instance`
- **THEN** the instance is created under the temporary path
- **AND** no files are created in the default `instances/` directory

#### Scenario: Tests use temporary releases directory

- **GIVEN** `OUBLIETTE_RELEASES_DIR` is set to a temporary path
- **WHEN** a manager test runs operations requiring release builds
- **THEN** releases are stored under the temporary path
- **AND** no files are created in the default `releases/` directory

#### Scenario: Cleanup removes temporary directories

- **GIVEN** a manager test has created temporary directories
- **WHEN** the test completes (success or failure)
- **THEN** temporary directories are removed
