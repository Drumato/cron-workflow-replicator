# CronWorkflow Replicator

this is a simple tool to replicate Kubernetes CronWorkflows with different values.

## Usage

### Using Docker

You can run the CLI using the pre-built Docker images without installing Go or building the binary locally.

#### Available Images

- `ghcr.io/drumato/cron-workflow-replicator:main` (amd64)
- `ghcr.io/drumato/cron-workflow-replicator:main-arm` (arm64)

#### Running with Docker

To run the CLI in a temporary Docker container:

```bash
# For amd64 systems
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/drumato/cron-workflow-replicator:main \
  /cron-workflow-replicator --config examples/v1alpha1/novalue/config.yaml

# For arm64 systems (Apple Silicon, etc.)
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/drumato/cron-workflow-replicator:main-arm \
  /cron-workflow-replicator --config examples/v1alpha1/novalue/config.yaml
```

#### Examples

Run different examples:

```bash
# Basic example with no values
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/drumato/cron-workflow-replicator:main \
  /cron-workflow-replicator --config examples/v1alpha1/novalue/config.yaml

# Example with values
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/drumato/cron-workflow-replicator:main \
  /cron-workflow-replicator --config examples/v1alpha1/withvalue/config.yaml

# Example with base manifest
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/drumato/cron-workflow-replicator:main \
  /cron-workflow-replicator --config examples/v1alpha1/basemanifest/config.yaml
```

The `--rm` flag automatically removes the container after execution, and `-v $(pwd):/workspace -w /workspace` mounts the current directory as the working directory inside the container.

#### What Each Example Does

- **novalue**: Generates basic CronWorkflow YAML files with minimal configuration (outputs to `./output/`)
- **withvalue**: Demonstrates how to inject custom values into the generated manifests (outputs to `./output/`)
- **basemanifest**: Uses a base manifest template and applies different configurations to create multiple variants (outputs to `examples/v1alpha1/basemanifest/output/`)

## Roadmap

- [x] replicates CronWorkflow manifests with different values
- [x] supports multiple input files
- [x] can read baseManifestPath that is specified in config file
  - if the baseManifestPath is specified, the replicator reads the base manifest and unmarshal into base CronWorkflow object.
  - then, it applies the values from the config file to the base CronWorkflow object.
