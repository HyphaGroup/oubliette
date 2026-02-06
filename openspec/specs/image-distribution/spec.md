# image-distribution Specification

## Purpose
TBD - created by archiving change add-container-image-distribution. Update Purpose after archive.
## Requirements
### Requirement: Container types are defined in configuration

Container type to image mappings SHALL be defined in `oubliette.jsonc` under the `containers` key.

```jsonc
{
  "containers": {
    "base": "ghcr.io/hyphagroup/oubliette-base:latest",
    "dev": "ghcr.io/hyphagroup/oubliette-dev:latest",
    "custom": "myregistry.com/my-image:v1"
  }
}
```

#### Scenario: Config defines available container types

- Given oubliette.jsonc contains containers section with "base", "dev", "custom"
- When `project_options` is called
- Then available container types include "base", "dev", "custom"

#### Scenario: Default config includes shipped containers

- Given `oubliette init` is run
- When oubliette.jsonc is created
- Then containers section includes "base" and "dev" with ghcr.io paths

#### Scenario: Unknown container type rejected

- Given containers config only has "base" and "dev"
- When `project_create` is called with `container_type: "unknown"`
- Then an error is returned indicating invalid container type

### Requirement: Runtime interface supports image pull

The container `Runtime` interface SHALL include a `Pull` method to download images from a registry.

```go
Pull(ctx context.Context, imageName string) error
```

#### Scenario: Docker runtime pulls image from registry

- Given Docker runtime is configured
- And image `ghcr.io/hyphagroup/oubliette-dev:latest` does not exist locally
- When `Pull(ctx, "ghcr.io/hyphagroup/oubliette-dev:latest")` is called
- Then the image is downloaded from ghcr.io
- And the image exists locally after completion

#### Scenario: Apple Container runtime pulls image from registry

- Given Apple Container runtime is configured
- And image `ghcr.io/hyphagroup/oubliette-dev:latest` does not exist locally
- When `Pull(ctx, "ghcr.io/hyphagroup/oubliette-dev:latest")` is called
- Then `container image pull ghcr.io/hyphagroup/oubliette-dev:latest` is executed
- And the image exists locally after completion

#### Scenario: Pull fails with clear error

- Given the registry is unreachable
- When `Pull` is called
- Then an error is returned with the image name and failure reason

### Requirement: ImageManager resolves container types to images

The `ImageManager` SHALL resolve container type names to image references using the configuration.

#### Scenario: ImageManager returns configured image

- Given containers config maps "dev" to "ghcr.io/hyphagroup/oubliette-dev:latest"
- When `ImageManager.GetImageName("dev")` is called
- Then it returns "ghcr.io/hyphagroup/oubliette-dev:latest"

#### Scenario: ImageManager returns custom image

- Given containers config maps "python-ml" to "myregistry.com/ml-agent:v2"
- When `ImageManager.GetImageName("python-ml")` is called
- Then it returns "myregistry.com/ml-agent:v2"

#### Scenario: ImageManager rejects unknown type

- Given containers config does not contain "invalid"
- When `ImageManager.GetImageName("invalid")` is called
- Then an error is returned

### Requirement: Server pulls missing images on demand

When the server needs a container image that doesn't exist locally, it SHALL automatically pull from the registry.

#### Scenario: Server pulls image when starting session

- Given the image for container type "dev" does not exist locally
- When a session is spawned with container type "dev"
- Then the server pulls the configured image
- And the session starts successfully after pull completes

#### Scenario: Server uses cached image

- Given the image for container type "dev" exists locally
- When a session is spawned with container type "dev"
- Then no pull is performed
- And the session starts using the local image

### Requirement: GitHub Actions publishes images on release

The release workflow SHALL build and push container images to ghcr.io when a version tag is pushed. Images SHALL be tagged with both `:latest` and the version tag.

#### Scenario: Release workflow pushes images with version tags

- Given a tag `v1.0.0` is pushed
- When the release workflow runs
- Then `ghcr.io/hyphagroup/oubliette-base:latest` is built and pushed
- And `ghcr.io/hyphagroup/oubliette-base:v1.0.0` is built and pushed
- And `ghcr.io/hyphagroup/oubliette-dev:latest` is built and pushed
- And `ghcr.io/hyphagroup/oubliette-dev:v1.0.0` is built and pushed
- And binaries are released as before

### Requirement: Pull displays progress

When pulling images, the `Pull` method SHALL stream progress output so users can see download status.

#### Scenario: Docker pull shows progress

- Given Docker runtime is pulling a large image
- When `Pull` is called
- Then download progress is displayed to stdout
- And user sees layer-by-layer progress

#### Scenario: Apple Container pull shows progress

- Given Apple Container runtime is pulling a large image
- When `Pull` is called
- Then `container image pull` output is displayed
- And user sees download progress

### Requirement: Init pre-pulls all configured images

The `oubliette init` command SHALL pre-pull all container images listed in the default config so the first session spawn is fast.

#### Scenario: Init pulls all configured images

- Given `oubliette init` is run
- And ghcr.io is reachable
- When initialization completes
- Then `ghcr.io/hyphagroup/oubliette-base:latest` has been pulled
- And `ghcr.io/hyphagroup/oubliette-dev:latest` has been pulled
- And user sees pull progress during init for each image

#### Scenario: Init continues if pull fails

- Given `oubliette init` is run
- And ghcr.io is unreachable
- When initialization completes
- Then a warning is displayed about failed pull
- And manual pull command is suggested
- And init still succeeds (config/token created)

### Requirement: Development mode uses local images

When `OUBLIETTE_DEV=1` environment variable is set, default container images SHALL use local names without the ghcr.io prefix, and Pull SHALL be skipped.

#### Scenario: Dev mode defaults to local image names

- Given `OUBLIETTE_DEV=1` is set
- And oubliette.jsonc does not specify containers
- When default containers are applied
- Then "dev" maps to `oubliette-dev:latest` (no ghcr.io prefix)

#### Scenario: Dev mode skips pull

- Given `OUBLIETTE_DEV=1` is set
- And image does not exist locally
- When `EnsureImageExists` is called
- Then no pull is attempted
- And error indicates local image not found (build with build.sh)

#### Scenario: Production mode defaults to registry image names

- Given `OUBLIETTE_DEV` is not set
- And oubliette.jsonc does not specify containers
- When default containers are applied
- Then "dev" maps to `ghcr.io/hyphagroup/oubliette-dev:latest`

### Requirement: Custom containers can be added to config

Users SHALL be able to add custom container images to the configuration.

#### Scenario: User adds custom container type

- Given user edits oubliette.jsonc to add `"python-ml": "myregistry.com/ml:v1"`
- When `project_create` is called with `container_type: "python-ml"`
- Then project is created using image "myregistry.com/ml:v1"

#### Scenario: Custom container requires pull

- Given containers config maps "custom" to "myregistry.com/custom:latest"
- And image does not exist locally
- When session is spawned with container type "custom"
- Then the image is pulled from myregistry.com
- And session starts successfully

### Requirement: container_refresh MCP tool replaces image_rebuild

The `image_rebuild` MCP tool SHALL be replaced with `container_refresh` which pulls the latest image and restarts the container.

```json
{
  "name": "container_refresh",
  "arguments": {
    "project_id": "proj_xxx"
  }
}
```

#### Scenario: Refresh pulls and restarts container

- Given project "proj_xxx" has a running container using "dev" type
- And no active sessions exist for the project
- When `container_refresh` is called with `project_id: "proj_xxx"`
- Then the image for "dev" is pulled from registry
- And the current container is stopped
- And the old container is removed
- And a new container is started with the fresh image
- And success message is returned

#### Scenario: Refresh works with custom image

- Given project uses custom container type "python-ml"
- And no active sessions exist for the project
- When `container_refresh` is called
- Then "python-ml" image is pulled from configured registry
- And container is restarted with fresh image

#### Scenario: Next message uses new container

- Given `container_refresh` completed successfully
- When `session_message` is called
- Then the message is processed in the new container
- And any new tools/updates in the image are available

#### Scenario: Refresh fails if active sessions exist

- Given project "proj_xxx" has active sessions
- When `container_refresh` is called with `project_id: "proj_xxx"`
- Then an error is returned indicating active sessions must be ended first
- And the container is NOT restarted

#### Scenario: Refresh fails if no running container

- Given project has no running container
- When `container_refresh` is called
- Then an error is returned indicating no container to refresh

### Requirement: oubliette container CLI subcommand

The `oubliette` binary SHALL support a `container` subcommand for managing containers from the command line.

```bash
oubliette container list                 # list running containers
oubliette container refresh              # refresh all running containers
oubliette container refresh <project_id> # refresh specific container
oubliette container stop <project_id>    # stop specific container
oubliette container stop --all           # stop all containers
```

#### Scenario: List running containers

- Given projects "proj_a" and "proj_b" have running containers
- When `oubliette container list` is run
- Then output shows both containers with project ID, image, and status

#### Scenario: Refresh all containers

- Given projects "proj_a" and "proj_b" have running containers
- When `oubliette container refresh` is run without arguments
- Then both containers are refreshed (pull + restart)
- And progress is shown for each

#### Scenario: Refresh specific container

- Given project "proj_a" has a running container
- When `oubliette container refresh proj_a` is run
- Then only proj_a's container is refreshed

#### Scenario: Stop specific container

- Given project "proj_a" has a running container
- When `oubliette container stop proj_a` is run
- Then the container is stopped and removed

#### Scenario: Stop all containers

- Given multiple projects have running containers
- When `oubliette container stop --all` is run
- Then all containers are stopped and removed

