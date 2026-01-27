# Usage Guide

This guide shows you how to use the CronWorkflow Replicator tool.

## Basic Usage

```bash
./cron-workflow-replicator --config path/to/config.yaml
```

## Using Docker

You can run the CLI using the pre-built Docker images without installing Go or building the binary locally.

### Available Images

- `ghcr.io/drumato/cron-workflow-replicator:main` (amd64)
- `ghcr.io/drumato/cron-workflow-replicator:main-arm` (arm64)

### Running with Docker

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

The `--rm` flag automatically removes the container after execution, and `-v $(pwd):/workspace -w /workspace` mounts the current directory as the working directory inside the container.

## Examples

### Basic Example (No Values)

```bash
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/drumato/cron-workflow-replicator:main \
  /cron-workflow-replicator --config examples/v1alpha1/novalue/config.yaml
```

**What it does**: Generates basic CronWorkflow YAML files with minimal configuration (outputs to `./output/`)

### Example with Values

```bash
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/drumato/cron-workflow-replicator:main \
  /cron-workflow-replicator --config examples/v1alpha1/withvalue/config.yaml
```

**What it does**: Demonstrates how to inject custom values into the generated manifests (outputs to `./output/`)

### Example with Base Manifest

```bash
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/drumato/cron-workflow-replicator:main \
  /cron-workflow-replicator --config examples/v1alpha1/basemanifest/config.yaml
```

**What it does**: Uses a base manifest template and applies different configurations to create multiple variants (outputs to `examples/v1alpha1/basemanifest/output/`)

### Example with Kustomize Integration

```bash
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/drumato/cron-workflow-replicator:main \
  /cron-workflow-replicator --config examples/v1alpha1/kustomize/config.yaml
```

**What it does**: Generates CronWorkflow manifests and automatically creates/updates a kustomization.yaml file to include all generated resources (outputs to `examples/v1alpha1/kustomize/output/`)

## Building from Source

If you prefer to build the binary locally:

```bash
make build
./bin/cron-workflow-replicator --config examples/v1alpha1/novalue/config.yaml
```

## Testing

To run the tests:

```bash
make test
```