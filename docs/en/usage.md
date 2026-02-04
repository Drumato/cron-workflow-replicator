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

## Configuration Examples with JSONPath

### Example 1: Production Backup Workflow

```yaml
units:
  - outputDirectory: "./output"
    values:
      - filename: "production-backup"
        paths:
          - path: "$.metadata.name"
            value: "production-daily-backup"
          - path: "$.metadata.namespace"
            value: "production"
          - path: "$.metadata.labels.app"
            value: "backup-service"
          - path: "$.metadata.labels.environment"
            value: "production"
          - path: "$.spec.schedule"
            value: "0 2 * * *"  # 2 AM daily
          - path: "$.spec.concurrencyPolicy"
            value: "Forbid"
          - path: "$.spec.successfulJobsHistoryLimit"
            value: "3"
          - path: "$.spec.failedJobsHistoryLimit"
            value: "1"
```

### Example 2: Multi-Environment Data Processing

```yaml
units:
  - outputDirectory: "./output"
    values:
      - filename: "data-processing-staging"
        paths:
          - path: "$.metadata.name"
            value: "data-processing-staging"
          - path: "$.metadata.namespace"
            value: "staging"
          - path: "$.spec.schedule"
            value: "0 */4 * * *"  # Every 4 hours
          - path: "$.spec.workflowSpec.arguments.parameters[0].value"
            value: "s3://staging-data-bucket/"
      - filename: "data-processing-production"
        paths:
          - path: "$.metadata.name"
            value: "data-processing-production"
          - path: "$.metadata.namespace"
            value: "production"
          - path: "$.spec.schedule"
            value: "0 1 * * *"   # 1 AM daily
          - path: "$.spec.workflowSpec.arguments.parameters[0].value"
            value: "s3://production-data-bucket/"
```

### Example 3: Complex Nested Configuration

```yaml
units:
  - outputDirectory: "./output"
    baseManifestPath: "./templates/complex-workflow.yaml"
    values:
      - filename: "ml-training-pipeline"
        paths:
          - path: "$.metadata.name"
            value: "weekly-ml-training"
          - path: "$.spec.schedule"
            value: "0 0 * * 0"  # Weekly on Sunday
          - path: "$.spec.workflowSpec.templates[0].container.env[0].value"
            value: "production"
          - path: "$.spec.workflowSpec.templates[0].container.resources.requests.memory"
            value: "8Gi"
          - path: "$.spec.workflowSpec.templates[0].container.resources.requests.cpu"
            value: "4"
          - path: "$.spec.workflowSpec.arguments.parameters[0].name"
            value: "model-version"
          - path: "$.spec.workflowSpec.arguments.parameters[0].value"
            value: "v2.1.0"
```

These examples demonstrate:
- **Environment-specific configurations**: Different namespaces, schedules, and parameters for staging vs production
- **Resource management**: Setting memory and CPU requests
- **Complex path targeting**: Accessing deeply nested fields like container environment variables and array elements
- **Parameter injection**: Setting workflow arguments and parameters dynamically

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