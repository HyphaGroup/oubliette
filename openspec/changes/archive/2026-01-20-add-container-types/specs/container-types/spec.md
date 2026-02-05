# Container Types

Container type system for specialized execution environments.

## ADDED Requirements

### Requirement: Container type definitions MUST exist in containers/ directory

The system SHALL provide pre-defined container types (base, dev, osint) with Dockerfiles in the `containers/` directory.

#### Scenario: Base container is minimal

- Given the base container type
- When inspecting its contents
- Then it includes only essential tools (bash, curl, git, jq)
- And Factory Droid CLI
- And oubliette-relay/client binaries

#### Scenario: Dev container extends base

- Given the dev container type
- When built
- Then it is based on oubliette-base:latest
- And includes Node.js, Python, Java runtimes
- And includes build tools and CLI utilities

#### Scenario: OSINT container extends base

- Given the osint container type
- When built
- Then it is based on oubliette-base:latest
- And includes OSINT tools (theHarvester, amass, subfinder, httpx, nuclei)

### Requirement: Projects MUST be able to specify container type at creation

The `project_create` MCP tool SHALL accept an optional `container_type` parameter.

#### Scenario: Create project with default container type

- Given no container_type specified
- When creating a project
- Then the project uses "dev" as container type
- And the container uses oubliette-dev:latest image

#### Scenario: Create project with specific container type

- Given container_type="osint"
- When creating a project
- Then the project uses "osint" as container type
- And the container uses oubliette-osint:latest image

#### Scenario: Invalid container type rejected

- Given container_type="invalid"
- When creating a project
- Then the request fails with validation error

### Requirement: Container images MUST be built on demand

When a project's container type image doesn't exist, the system SHALL build it automatically.

#### Scenario: Image built when missing

- Given oubliette-osint:latest does not exist
- And a project with container_type="osint"
- When starting the container
- Then oubliette-base:latest is built first (if missing)
- And oubliette-osint:latest is built
- And the container starts successfully

#### Scenario: Existing image is reused

- Given oubliette-dev:latest already exists
- And a project with container_type="dev"
- When starting the container
- Then no build occurs
- And the existing image is used

### Requirement: Container types MUST be included in project_options

The `project_options` tool SHALL include container types in its response.

#### Scenario: Get container types from project_options

- Given container types base, dev, and osint are available
- When calling `project_options`
- Then response includes `container_types` section
- And `container_types.available` lists all types with descriptions
- And `container_types.default` is "dev"

### Requirement: Manager script MUST be able to build container images

The `manager.sh` script SHALL provide a `rebuild-images` command.

#### Scenario: Rebuild all images

- Given running `manager.sh rebuild-images`
- When the command completes
- Then all container type images are built
- And built in dependency order (base first)

#### Scenario: Rebuild specific type

- Given running `manager.sh rebuild-images --type osint`
- When the command completes
- Then oubliette-base:latest is built (dependency)
- And oubliette-osint:latest is rebuilt

### Requirement: Image rebuild MCP tool MUST support container types

The `image_rebuild` MCP tool SHALL accept an optional `container_type` parameter.

#### Scenario: Rebuild project's container type

- Given a project with container_type="osint"
- When calling image_rebuild with project_id
- Then oubliette-osint:latest is rebuilt

#### Scenario: Rebuild specific type without project

- Given calling image_rebuild with container_type="base"
- When the command completes
- Then oubliette-base:latest is rebuilt
