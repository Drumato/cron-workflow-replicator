# CronWorkflow Replicator

A simple tool to replicate Kubernetes CronWorkflows with different values.

## Quick Start

```bash
# Basic usage
./cron-workflow-replicator --config examples/v1alpha1/novalue/config.yaml

# Using Docker
docker run --rm -v $(pwd):/workspace -w /workspace \
  ghcr.io/drumato/cron-workflow-replicator:main \
  /cron-workflow-replicator --config examples/v1alpha1/novalue/config.yaml
```

## Documentation

- **English**: [docs/en/README.md](docs/en/README.md)
- **日本語**: [docs/ja/README.md](docs/ja/README.md)

## Roadmap

- [x] replicates CronWorkflow manifests with different values
- [x] supports multiple input files
- [x] can read baseManifestPath that is specified in config file
- [x] resolves relative paths relative to config file location
